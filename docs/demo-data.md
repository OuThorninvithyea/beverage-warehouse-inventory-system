# Development demo data

`make seed-demo` loads a small but complete beverage-warehouse dataset so the
dashboards, lists, inventory screens and manual QA have realistic data to work
with. The seeder refuses to run unless `APP_ENV=development`, and it is safe to
run repeatedly.

```bash
make seed-demo                  # migrations, admin and demo data
SEED_DEMO_DATA=false make seed-admin   # administrator only
```

The fixtures live in `backend/cmd/seed/demo_data.go`. Balances and FIFO cost
layers are **derived** from the movement list in `backend/cmd/seed/ledger.go`
rather than written by hand, so the seeded ledger is internally consistent: what
you see in the inventory screens is exactly what those movements would have
produced through the API.

## Accounts

All demo accounts share one password, `DemoPass123!` by default; override it
with `SEED_DEMO_PASSWORD`. These are development credentials only.

| Email | Role | Warehouse |
| --- | --- | --- |
| admin@bwims.local | admin | all |
| manager@bwims.local | warehouse_manager | PP-CENTRAL |
| picker@bwims.local | picker | PP-CENTRAL |
| picker.sr@bwims.local | picker | SR-DEPOT |
| viewer@bwims.local | viewer | none |

The administrator password still comes from `SEED_ADMIN_PASSWORD`
(`ChangeMe123!` by default).

## Warehouses and locations

| Warehouse | Location | Zone | Pickable |
| --- | --- | --- | --- |
| PP-CENTRAL (Phnom Penh Central DC) | RECV-DOCK | RECEIVING | no |
| PP-CENTRAL | QUAR-01 | QUARANTINE | no |
| PP-CENTRAL | A-01-01 | AMBIENT | yes |
| PP-CENTRAL | A-01-02 | AMBIENT | yes |
| PP-CENTRAL | COLD-01 | CHILLED | yes |
| SR-DEPOT (Siem Reap Depot) | RECV-DOCK | RECEIVING | no |
| SR-DEPOT | A-01-01 | AMBIENT | yes |
| SR-DEPOT | A-01-02 | AMBIENT | yes |

## Catalog

15 products across seven categories: carbonated soft drinks, water, juice,
energy drinks, beer, dairy and coffee, and packaging. Fourteen are lot tracked;
`PKG-CRATE-24` is not, so the untracked path (a balance with a NULL `lot_id`)
is covered too. Every product carries a valid barcode — see
[barcode-test-data.md](barcode-test-data.md) for the full list.

## Lots and expiry

Lot expiry dates are relative to the seed run, so the expiry scenarios stay
meaningful whenever the data is loaded:

| Lot | Expiry | What it demonstrates |
| --- | --- | --- |
| L-ORNG-EXPIRED | 5 days ago | Expired stock still on hand, and the FR-19 audit flag |
| L-MILK-2607 | in 7 days | Near-expiry alerting |
| L-LATTE-2607 | in 12 days | Near-expiry alerting |
| L-ORNG-2606 | in 21 days | FEFO ordering against the expired lot |
| L-COLA-2512 / L-COLA-2601 | in 60 / 180 days | Two lots of one product for FEFO picking |
| others | 100–540 days | Healthy stock |

## Movement history

27 movements spread over the last 120 days, covering every movement type:

- **receive** into both warehouses, at different unit costs so FIFO valuation
  has more than one layer per product;
- **pick**, including a deliberate pick from the expired orange juice lot, which
  writes the `EXPIRED_LOT_PICK` row in `audit_records` (FR-19);
- **transfer**, both inside one warehouse (dock to rack putaway) and across
  warehouses, which moves the cost layer as well as the stock;
- **adjust**, one decrease for damage and one increase from a cycle count.

Two balances carry a reserved quantity so the reserved and available columns are
not all zero.

All seeded movements use a `SEED-` reference prefix. Because `stock_movements`
and `audit_records` are append-only, the seeder writes that ledger only when no
`SEED-` movement exists yet; reference data (warehouses, locations, users,
categories, products, lots) is upserted on every run. To rebuild the ledger from
scratch, drop the database volume and re-run the migrations:

```bash
docker compose down -v && make seed-demo
```
