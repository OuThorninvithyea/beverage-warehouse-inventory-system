# Week 10 inventory validation

**Validation date:** 2026-08-18
**Branch:** `codex/inventory-movements`
**API checkpoint:** `f477ecf` (`docs: mark FR-10 through FR-18 implemented, FR-19 partial`)

## Implemented scope

- Read-only inventory balance listing (`GET /api/v1/inventory`), warehouse-scoped
  for non-admins, cursor-paginated on `created_at`.
- Read-only lot listing per product (`GET /api/v1/inventory/products/:product_id/lots`),
  ordered FEFO, with availability summed per the caller's visible warehouse scope.
- Receive: creates or reuses a lot, opens one `cost_layers` row, upserts the
  destination balance, all in one transaction.
- FEFO pick: auto-selects lots by earliest expiration (or an explicit
  `lot_id` override), splitting a single pick across multiple lots when
  needed, and rejecting atomically with `INSUFFICIENT_STOCK` if the
  location cannot cover the full requested quantity.
- FIFO cost-layer consumption on pick, ordered by `received_at` — proven to
  diverge correctly from FEFO physical consumption in the integration test
  (a later-received, sooner-expiring lot is drained physically first, while
  the earlier-received cost layer is drained financially first).
- Transfer: single-lot, warehouse-scoped on both ends for non-admins,
  rejects same-location transfers.
- Adjust: directional (increase/decrease) count correction; intentionally
  does not touch `cost_layers` (no cost basis input — see the design doc).
- Movement history listing and single-movement read.
- One additive migration (`000005_inventory_constraints`): adds
  `inventory_balances.created_at` for cursor pagination consistency with
  every other module.

## Automated verification

Commands run from `backend/`:

```text
go build ./...
go vet ./...
go test ./... -count=1
```

Result on 2026-08-18:

- `go build ./...` — PASS.
- `go vet ./...` — PASS, no output.
- `go test ./... -count=1` — PASS, all packages:

```text
ok  	.../backend/internal/app	0.671s
ok  	.../backend/internal/modules/auth	1.689s
ok  	.../backend/internal/modules/catalog	3.598s
ok  	.../backend/internal/modules/health	1.732s
ok  	.../backend/internal/modules/inventory	3.227s
ok  	.../backend/internal/modules/users	4.160s
ok  	.../backend/internal/modules/warehouse	3.438s
ok  	.../backend/internal/platform/config	4.055s
```

`TestPostgresInventoryLifecycle` (the full FEFO/FIFO divergence case,
receive → multi-lot pick → transfer → adjust increase/decrease → movement
history → error cases) passed as part of the `inventory` package run above.

`docker compose config --quiet` — PASS, no output.
`git diff --check` — PASS, no output (no whitespace errors).

Frontend re-run (no frontend code changed in this slice, confirming no
regression), from `frontend/`:

```text
npm run test        # 1 test file, 4 tests passed
npm run typecheck    # vue-tsc --noEmit, no errors
npm run build        # succeeded; same pre-existing non-blocking warning
                      # that BarcodeTestView's chunk exceeds 500 kB,
                      # already noted in week-8-catalog-validation.md
```

## Database migration status

Applied against a local, dedicated Postgres 16 database (`bwims_inventory_test`,
via Homebrew `postgresql@18` running the standard `postgres` protocol on
`localhost:5432`), separate from any development database. Migrations
`000001` through `000005` applied cleanly from empty. `000005` was rolled
back one step and reapplied cleanly before development began (`migrate ...
down 1` then `up`). The normal `bwims` development database was not
touched by this validation.

One real issue was found and fixed during development, not swept under the
rug: the integration test fixture originally seeded its test actor with the
`admin` role. Because `stock_movements` is protected by an append-only
immutability trigger (`stock_movements_immutable`), that seeded user could
never actually be deleted once a movement referenced it via
`performed_by` — so repeated test runs accumulated leftover active admin
users in the shared test database, which broke the *unrelated*
`TestPostgresUsersLifecycle` test's "last active admin" invariant (a count
taken globally across the `users` table, not scoped to any one test's
fixture). Fixed by seeding the test actor as `picker` instead — a
non-admin role has no effect on that invariant — and by removing the
integration test's cleanup attempts for `stock_movements`, `products`,
`locations`, `users`, and `warehouses`, which were always silently failing
(errors were ignored) for the same append-only/foreign-key reason.
`cost_layers` and `inventory_balances` have no such restriction and are
still deleted in cleanup. This means the dedicated test database
accumulates uniquely-suffixed warehouses/locations/products/users/movements
across repeated runs — expected behavior for an audit-trail schema, not a
defect, and it does not affect correctness since every run uses a fresh
random suffix.

## Remaining work and boundaries

- No frontend screens for inventory — separate slice, not started.
- No expiry alerts, inventory valuation, movement summary, or product
  velocity reporting — Week 11–12 scope, depends on this module's data but
  is not part of it.
- FR-19 (prevent expired-lot picking) is only partially implemented: FEFO
  ordering means the earliest-expiring lot is always drawn down first, but
  there is no hard rejection of picking an *already-expired* lot. Blocking
  it outright, warning-and-confirming, or allowing it with an audit flag
  are all defensible choices with different tradeoffs for a beverage
  distributor — that is a product decision this slice deliberately did not
  make unilaterally.
- Adjustments do not create or touch `cost_layers` rows by design (no cost
  basis input on that endpoint) — a future valuation report will need to
  treat adjustment deltas as cost-neutral.
- Reservation writes (`reserved_quantity` above zero) are not exercised by
  any endpoint in this slice; the column exists in the schema for a future
  sales/allocation feature.
- Real hardware barcode-scanner evidence for issue #6 is unrelated to this
  slice and remains outstanding separately.
