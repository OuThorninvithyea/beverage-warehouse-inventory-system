# Development demo data

`make seed-demo` loads a full beverage-warehouse dataset so every screen, filter
and edge case has something realistic behind it. The seeder refuses to run
unless `APP_ENV=development`, and it is safe to run repeatedly.

```bash
make seed-demo                         # migrations, admin and demo data
SEED_DEMO_DATA=false make seed-admin   # administrator only
```

## What you get

| Entity | Active | Inactive | Notes |
| --- | --- | --- | --- |
| Warehouses | 3 | 1 | Deep, mid-size and near-empty sites, plus a closed one |
| Locations | 15 | 2 | Receiving, shipping, quarantine, ambient and chilled |
| Users | 10 | 2 | All four roles, across all three live warehouses |
| Categories | 12 | 1 | Two-level hierarchy under Beverages and Packaging |
| Products | 41 | 2 | 43 total, so product lists span three pages |
| Lots | 42 | — | Expiry spread from already-expired to 480 days out |
| Balances | 73 | — | 23 products stocked in more than one location |
| Movements | 112 | — | 66 receives, 33 picks, 7 transfers, 6 adjustments |
| Cost layers | 68 | — | Several layers per product, so FIFO has depth |
| Audit records | 2 | — | `EXPIRED_LOT_PICK` entries from deliberate expired picks |

Lists default to 20 rows per page, so products, balances and movements all
paginate out of the box.

The fixtures live in `backend/cmd/seed/demo_data.go`, with the bulk stock
generated from them in `demo_generate.go`. Balances and FIFO cost layers are
**derived** by replaying the movement list through `ledger.go` rather than being
written by hand, so the seeded ledger is internally consistent: what the
inventory screens show is exactly what those movements would have produced
through the API. A Go test fails the build if any balance would go negative, if
a reservation exceeds its balance, or if stock lands in an inactive or
non-pickable location.

## Accounts

All demo accounts share one password, `DemoPass123!` by default; override it
with `SEED_DEMO_PASSWORD`. Development credentials only.

| Email | Role | Warehouse |
| --- | --- | --- |
| admin@bwims.local | admin | all |
| manager@bwims.local | warehouse_manager | PP-CENTRAL |
| manager.sr@bwims.local | warehouse_manager | SR-DEPOT |
| manager.bb@bwims.local | warehouse_manager | BB-HUB |
| picker@bwims.local | picker | PP-CENTRAL |
| picker2@bwims.local | picker | PP-CENTRAL |
| picker.sr@bwims.local | picker | SR-DEPOT |
| picker.bb@bwims.local | picker | BB-HUB |
| viewer@bwims.local | viewer | none |
| auditor@bwims.local | viewer | PP-CENTRAL |
| former.picker@bwims.local | picker | PP-CENTRAL — **deactivated** |
| former.manager@bwims.local | warehouse_manager | SR-DEPOT — **deactivated** |

The administrator password comes from `SEED_ADMIN_PASSWORD` (`ChangeMe123!` by
default), not `SEED_DEMO_PASSWORD`.

## Where the stock is

Picking is location-scoped: a pick only sees stock in the exact location you
name, so knowing this table saves a lot of `INSUFFICIENT_STOCK` confusion.

| Warehouse | Location | Distinct products |
| --- | --- | --- |
| PP-CENTRAL | A-01-01 | 17 |
| PP-CENTRAL | A-01-02 | 10 |
| PP-CENTRAL | A-02-01 | 11 |
| PP-CENTRAL | B-01-01 | 11 |
| PP-CENTRAL | COLD-01 | 6 |
| PP-CENTRAL | COLD-02 | 3 |
| SR-DEPOT | A-01-01 | 9 |
| SR-DEPOT | COLD-01 | 1 |
| BB-HUB | A-01-01 | 3 |

Receiving, shipping and quarantine locations are deliberately left empty and
non-pickable, and `PP-CENTRAL/OLD-01` and the whole `KP-CLOSED` warehouse are
inactive.

## Edge cases the fixtures cover on purpose

- **Inactive rows** of every kind: warehouse, location, user, category, product
- **Products with no barcode** (`PKG-PALLET-EU`, `MSC-UNSORTED-001`)
- **A product with no category** (`MSC-UNSORTED-001`)
- **Products that are not lot tracked**, so balances with a NULL `lot_id` exist
- **A lot with no expiration date** (`SPI-RUM-700`)
- **Reserved quantities** on three balances, so available differs from on hand
- **Multi-location stock** for 23 products, which is what makes transfers and
  location-scoped picks testable
- **Both barcode formats**: EAN-13 throughout, UPC-A on the two imported SKUs

## Expiry and FEFO scenarios

Lot expiry is relative to the seed run, so these stay meaningful whenever you
load the data:

| Lot | Expiry | Demonstrates |
| --- | --- | --- |
| L-ORNG-EXPIRED | 5 days ago | Expired stock on hand, and the FR-19 audit flag |
| L-YOG-EXPIRED | 12 days ago | A second expired lot, withdrawn for disposal |
| L-COCO-EXPIRED | 2 days ago | Expired stock in Siem Reap, not just the main DC |
| L-MILK-2607 | in 7 days | Expiry alerts at the urgent end |
| L-LATTE-2607 | in 12 days | Expiry alerts |
| L-ORNG-2606 | in 21 days | FEFO against the same product's expired lot |
| L-COLA-2512 / L-COLA-2601 | in 60 / 180 days | Two lots, one location: the FEFO demo |
| L-RUM-NOEXP | none | A lot that never expires |

## Rebuilding from scratch

Reference data is upserted on every run. The movement ledger is append-only, so
the seeder writes it only when no `SEED-` movement exists yet. To start over:

```bash
docker compose down -v && make seed-demo
```
