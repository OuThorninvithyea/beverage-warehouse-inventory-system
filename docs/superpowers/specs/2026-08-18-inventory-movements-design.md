# Inventory Balances and Stock Movements API Design

**Status:** Approved for implementation on 2026-08-18
**Gantt scope:** Weeks 9–10, Task 3.1/3.2 (Inventory, lots, receive, FEFO pick, transfer, adjustment, FIFO costing)
**Branch:** `codex/inventory-movements`

## Goal

Deliver the inventory core: current stock balances per location/lot, lot and
expiry tracking, and the four audited stock movements (receive, FEFO pick,
transfer, adjust) with FIFO cost-layer consumption. This is the largest
remaining backend slice and blocks every Week 11–12 reporting feature
(expiry alerts, valuation, movement summaries).

The implementation follows the existing BWIMS structure:

`Route -> Handler -> Service -> Repository -> PostgreSQL`

## Scope

This slice includes:

- Read-only inventory balance listing, scoped by warehouse for non-admins.
- Read-only lot listing per product, ordered FEFO.
- Receive: creates/reuses a lot, increments a location balance, opens a cost
  layer.
- FEFO pick: consumes stock from the earliest-expiring lot(s) at a location
  (or an explicit lot override), decrements balances, consumes cost layers
  FIFO by receipt date.
- Transfer: moves a single lot's quantity between two locations.
- Adjust: directional (increase/decrease) count correction against one
  balance row. Does not touch cost layers (see "FIFO Cost Layer Consumption").
- Movement history listing and single-movement read.
- A small additive migration for `inventory_balances.created_at`.
- Tests, API documentation, and traceability evidence, matching the rigor of
  the catalog and users modules.

This slice excludes:

- Expiry alerts, inventory valuation reports, movement summaries, product
  velocity, CSV export (Week 11–12 reporting module; consumes this module's
  data but is not part of it).
- Barcode label generation.
- Reservations / `reserved_quantity` write paths (the column exists in the
  schema for a future sales/allocation feature; this slice never sets it
  above zero).
- Frontend screens.
- Physical barcode hardware evidence (tracked separately, issue #6).

## Architecture

New module `backend/internal/modules/inventory/`:

```text
backend/internal/modules/inventory/
  model.go        # Balance, Lot, Movement, *Input types, ListFilter, Page[T]
  optional.go      # OptionalString tri-state (local copy, mirrors catalog/users)
  pagination.go   # cursor encode/decode, normalizeLimit
  repository.go   # pgx repository: balance locking, FIFO/FEFO consumption, lot resolution
  service.go      # RBAC, validation, warehouse-scoping, orchestration
  handler.go      # Fiber handlers, domain-error -> HTTP mapping
  route.go        # RegisterRoutes
```

All four movement writes share one transactional core in the repository:
lock the relevant `inventory_balances` row(s) `FOR UPDATE`, apply the
quantity change, insert the `stock_movements` row(s), and (for receive/pick)
touch `cost_layers`. Four thin service methods call into it with
type-specific validation — the same shared-helper pattern used in `catalog`
(`lockActiveWarehouse`) and `users` (`requireMoreThanOneActiveAdmin`) rather
than four independent copies of locking logic.

No event-sourcing / balance-recompute-from-log approach: the schema already
commits to a materialized `inventory_balances` table with
`reserved_quantity`, so `stock_movements` is the append-only audit trail and
`inventory_balances` is the derived, kept-in-sync current state.

The module is wired into `backend/internal/app/app.go` before the fallback
route, matching every prior module.

## Authorization

Non-admins are scoped to their assigned `warehouse_id`, resolved via
`locations.warehouse_id` — the same pattern as the warehouse/location module.
A non-admin token without an assigned `warehouse_id` receives HTTP 403 with
`FORBIDDEN` on every route in this module.

| Role | Read balances/lots/movements | Receive, Pick | Transfer, Adjust |
| --- | --- | --- | --- |
| `admin` | All warehouses | Allowed | Allowed |
| `warehouse_manager` | Assigned warehouse | Assigned warehouse | Assigned warehouse only |
| `picker` | Assigned warehouse | Assigned warehouse | Forbidden |
| `viewer` | Assigned warehouse | Forbidden | Forbidden |

Transfer additionally requires **both** `from_location_id` and
`to_location_id` to resolve to the actor's assigned warehouse for
non-admins — `warehouse_manager` cannot move stock into or out of a
warehouse they are not assigned to. Only `admin` may transfer across
warehouses.

## Data Model

### Balance (read-only via this API; only ever written by movement writes)

```json
{
  "id": "<uuid>",
  "location_id": "<uuid>",
  "product_id": "<uuid>",
  "lot_id": null,
  "quantity": "120.000",
  "reserved_quantity": "0.000",
  "available_quantity": "120.000",
  "created_at": "2026-08-18T08:00:00Z",
  "updated_at": "2026-08-18T08:00:00Z"
}
```

`available_quantity` is computed (`quantity - reserved_quantity`), not
stored. `GET /api/v1/inventory` accepts `limit`, `after`, `location_id`,
`product_id`, `warehouse_id`, `lot_id`; cursor-paginated on
`(created_at DESC, id DESC)`.

### Lot (read-only via this API; created only as a side effect of receive)

```json
{
  "id": "<uuid>",
  "product_id": "<uuid>",
  "lot_number": "LOT-2026-08-18-A",
  "expiration_date": "2026-11-01",
  "received_at": "2026-08-18T08:00:00Z",
  "available_quantity": "120.000"
}
```

`GET /api/v1/products/:product_id/lots` lists lots for a product, ordered
FEFO (`expiration_date ASC NULLS LAST, received_at ASC`), each annotated
with total available quantity across locations the caller can see. This is
what a frontend uses to let a user override FEFO auto-selection with an
explicit `lot_id`.

### Movement (immutable; created only via the four write endpoints)

```json
{
  "id": "<uuid>",
  "movement_type": "receive",
  "product_id": "<uuid>",
  "lot_id": null,
  "from_location_id": null,
  "to_location_id": "<uuid>",
  "quantity": "50.000",
  "unit_cost": "1.2500",
  "reference": "PO-1042",
  "notes": null,
  "performed_by": "<uuid>",
  "created_at": "2026-08-18T08:00:00Z"
}
```

`GET /api/v1/movements` accepts `limit`, `after`, `product_id`,
`location_id`, `movement_type`, `from`, `to` (date range on `created_at`);
cursor-paginated on `(created_at DESC, id DESC)`, using the existing
`stock_movements_product_created_idx`. `GET /api/v1/movements/:movement_id`
reads one.

## Movement Write Endpoints

### `POST /api/v1/movements/receive`

```json
{
  "location_id": "<uuid>",
  "product_id": "<uuid>",
  "quantity": "50.000",
  "unit_cost": "1.2500",
  "lot_number": "LOT-2026-08-18-A",
  "expiration_date": "2026-11-01",
  "reference": "PO-1042",
  "notes": null
}
```

`lot_number` is required when the product is lot-tracked and forbidden
otherwise (`VALIDATION_ERROR`). If a lot with that `(product_id,
lot_number)` already exists, it is reused (its `expiration_date` is not
overwritten); otherwise a new lot is created. `expiration_date` is optional
and only read when `lot_number` is present. Response:
`{ "movement": {...}, "balance": {...} }`.

### `POST /api/v1/movements/pick`

```json
{
  "location_id": "<uuid>",
  "product_id": "<uuid>",
  "quantity": "30.000",
  "lot_id": null,
  "reference": "SO-2201",
  "notes": null
}
```

`lot_id` is optional. Omitted: the repository auto-selects lots at that
location in FEFO order, splitting across lots as needed. Provided: picks
only from that exact lot. A pick that cannot be fully satisfied from the
location (across all eligible lots) fails atomically with
`INSUFFICIENT_STOCK` — no partial movement is written. Response, since a
pick can span multiple lots:

```json
{ "movements": [ { "...": "one stock_movements row per lot consumed" } ], "total_quantity": "30.000" }
```

### `POST /api/v1/movements/transfer`

```json
{
  "product_id": "<uuid>",
  "lot_id": null,
  "quantity": "10.000",
  "from_location_id": "<uuid>",
  "to_location_id": "<uuid>",
  "reference": null,
  "notes": null
}
```

`lot_id` is required when the product is lot-tracked, forbidden otherwise.
Single lot, no auto-splitting — moving a quantity spread across multiple
lots means multiple transfer calls. `from_location_id` and
`to_location_id` must differ (`SAME_LOCATION_TRANSFER`). Response:
`{ "movement": {...}, "source_balance": {...}, "destination_balance": {...} }`.

### `POST /api/v1/movements/adjust`

```json
{
  "location_id": "<uuid>",
  "product_id": "<uuid>",
  "lot_id": null,
  "direction": "increase",
  "quantity": "5.000",
  "notes": "cycle count correction"
}
```

`direction` is `"increase"` or `"decrease"`. `lot_id` is required when the
product is lot-tracked and must reference a lot that already exists for
that product (created by a prior receive) — adjust never creates a new lot,
since it has no `unit_cost`/`expiration_date` to seed one. The balance row
at that `(location_id, product_id, lot_id)` itself does not need to
pre-exist: an `increase` may create it (upsert, starting from zero),
exactly like receive does. A `decrease` requires the balance row to already
exist with sufficient `available_quantity`. `decrease` is validated against
`available_quantity` (`INSUFFICIENT_STOCK` if it would go negative — the
schema's `inventory_quantity_nonnegative` check is the last line of
defense, but the service returns a clean domain error first). Response:
`{ "movement": {...}, "balance": {...} }`.

## FEFO / FIFO Mechanics

Two independent consumption passes happen inside every pick transaction,
both required to succeed atomically:

1. **Physical stock (FEFO).** Candidate lot balances at the location are
   locked `FOR UPDATE` in `(expiration_date ASC NULLS LAST, received_at
   ASC)` order (or restricted to the caller's exact `lot_id`). The
   repository consumes across lots until the requested quantity is
   satisfied, writing one `stock_movements` row per lot touched.
2. **Cost layers (FIFO).** Consumed separately, in
   `(warehouse_id, product_id, received_at, id)` order among rows with
   `remaining_quantity > 0` — this matches the existing
   `cost_layers_fifo_idx` partial index and is **not** scoped by lot.

These two passes are intentionally decoupled: FEFO (physical, by expiry)
and FIFO (cost, by receipt date) can diverge in beverage distribution when
a later-received lot happens to expire sooner. The schema's cost-layer
index already reflects this — it has no `lot_id` column in its key — so
this design follows the schema rather than inventing lot-scoped costing
that the index doesn't support efficiently.

**Adjust never touches cost layers.** It has no `unit_cost` input, so
inventing a cost basis for a count correction would be a fabrication, not a
real accounting entry. This is a documented simplification: FIFO valuation
reports (a Week 11–12 feature, out of scope here) will treat adjustment
deltas as cost-neutral. If a mismatch between `inventory_balances.quantity`
and the sum of `cost_layers.remaining_quantity` becomes a real financial
concern, closing it is a valuation-module decision, not an inventory-module
one.

If cost-layer remaining quantity is ever insufficient to cover a pick that
the physical balance says should succeed, that indicates the two ledgers
have drifted out of sync from a bug elsewhere (not user error) — the
repository returns `INVENTORY_OPERATION_FAILED` (500) rather than a 4xx in
that case, and rolls back.

## Error Contract

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, filter, cursor, or limit |
| 403 | `FORBIDDEN` | Role, or non-admin without/outside assigned warehouse |
| 404 | `PRODUCT_NOT_FOUND` | Product does not exist or is inactive |
| 404 | `LOCATION_NOT_FOUND` | Location does not exist |
| 404 | `LOT_NOT_FOUND` | Referenced lot does not exist for the product |
| 409 | `INSUFFICIENT_STOCK` | Not enough available quantity to satisfy pick/transfer/decrease |
| 409 | `SAME_LOCATION_TRANSFER` | `from_location_id` equals `to_location_id` |
| 422 | `WAREHOUSE_MISMATCH` | Non-admin location(s) fall outside their assigned warehouse |
| 422 | `VALIDATION_ERROR` | Required business data missing/invalid, or lot fields mismatch `is_lot_tracked` |
| 500 | `INVENTORY_OPERATION_FAILED` | Unexpected failure, including cost/balance ledger drift |

## Migration

`000005_inventory_constraints`:

```sql
ALTER TABLE inventory_balances
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX inventory_balances_created_idx
    ON inventory_balances (created_at DESC, id DESC);
```

Down migration drops the index and column. No other schema changes —
`lots`, `stock_movements`, and `cost_layers` already support this slice as
designed in `000002_initial_schema`.

## Repository and Transaction Boundaries

- Every write (receive/pick/transfer/adjust) runs inside one `pgx`
  transaction; balance rows are always locked `FOR UPDATE` before quantity
  math.
- Pick and transfer additionally lock the source lot/balance rows in a
  stable order (by lot `id`) to avoid deadlocks when two concurrent
  operations touch overlapping lots.
- Pagination fetches `limit + 1` rows to determine `has_more`, matching
  every other module.
- Repository methods always receive `context.Context`.

## Testing Strategy

Red-green-refactor, matching the catalog and users modules.

### Unit tests
- FEFO lot ordering and splitting across multiple lots for a single pick.
- FIFO cost-layer consumption order and remaining-quantity tracking.
- RBAC per movement type and warehouse-scoping (including the "no assigned
  warehouse" 403 case).
- Lot-field validation against `is_lot_tracked` for receive/transfer/adjust.
- Insufficient-stock rejection for pick/transfer/decrease-adjust.
- Same-location transfer rejection.
- Cursor encode/decode and pagination edge cases.

### HTTP tests
- Authentication required on every route.
- Role boundaries return the expected status and error code.
- Malformed JSON/UUID/query input rejected.
- Multi-lot pick response shape.

### PostgreSQL integration test
One lifecycle test: receive (lot created, balance appears, cost layer
opens) -> FEFO pick spanning two lots (verify oldest-expiry consumed
first, cost layers consumed oldest-received first) -> transfer -> adjust
increase -> adjust decrease -> movement history query reflects all of the
above -> migration `000005` applies, rolls back, and reapplies cleanly.

### Full validation
`go test ./... -count=1`, `go vet ./...`, clean PostgreSQL migration run,
existing frontend tests/typecheck/build (no frontend changes in this
slice), `git diff --check`, `docker compose config --quiet`.

## Documentation and Completion Evidence

Update:

- `docs/api-contract.md` with the inventory and movement endpoints;
- `docs/requirements-traceability.md` for FR-10 through FR-19;
- validation evidence with exact commands and results, mirroring
  `docs/week-8-catalog-validation.md` and
  `docs/week-9-user-management-validation.md`.

The slice is complete only when tests pass and the branch is reviewed and
merged. Reporting features that consume this data (expiry alerts,
valuation, movement summaries) remain a separate Week 11–12 slice.
