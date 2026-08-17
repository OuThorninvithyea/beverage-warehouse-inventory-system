# Week 8 catalog validation

**Validation date:** 2026-08-13
**Branch:** `codex/pro-9-catalog-apis`
**API checkpoint:** `7d82b83` (`feat: expose catalog APIs`)

## Implemented scope

- Category create, list, read, update, and soft deactivation.
- Product create, list, read, update, and soft deactivation.
- Global authenticated catalog reads for all four roles.
- Catalog mutations for `admin` and `warehouse_manager`; mutation rejection for
  `picker` and `viewer`.
- Case-insensitive category-name and Product-SKU uniqueness.
- Category parent validation, cycle prevention, and in-use protection.
- Product category validation and tri-state Category/barcode updates.
- Cursor pagination and search/filter queries.
- Exact active-Product lookup for valid UPC-A and EAN-13 barcodes.
- PostgreSQL migration `000004_catalog_constraints` and repository lifecycle
  test coverage.

## Automated verification

Run from `backend/` with the existing local Go module cache:

```text
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/Users/outhorninvuth/go/pkg/mod go test ./internal/modules/catalog ./internal/app -count=1
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/Users/outhorninvuth/go/pkg/mod go test ./... -count=1
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/Users/outhorninvuth/go/pkg/mod go vet ./...
git diff --check
docker compose config --quiet
```

Result on 2026-08-13:

- Catalog and app tests: passed.
- Full backend test suite: passed.
- Go vet: passed with no output.
- Whitespace validation: passed with no output.
- Docker Compose configuration validation: passed with no output.
- PostgreSQL catalog lifecycle test: passed against the dedicated
  `bwims_catalog_test` database.

Run from `frontend/`:

```text
npm test
npm run typecheck
npm run build
```

- Frontend tests: 4 passed.
- Vue/TypeScript typecheck: passed.
- Production build: passed. Vite reported a non-blocking warning that the
  existing `BarcodeTestView` chunk is larger than 500 kB after minification.

The automated cases cover authentication, tampered tokens, all four RBAC roles,
create/update/deactivate status codes, tri-state JSON values, filters, cursor
validation, barcode route ordering, every documented domain error, safe 500
responses, and `items: []` serialization.

Test barcode values:

- EAN-13 valid: `4006381333931`.
- UPC-A valid: `036000291452`.
- Invalid checksum examples: `4006381333932`, `036000291453`.

## Database migration status

Docker Desktop PostgreSQL 16 was used with a fresh, separate
`bwims_catalog_test` database. Migrations `000001` through `000004` applied from
empty successfully. With `BWIMS_TEST_DATABASE_URL` pointing to that database,
`TestPostgresCatalogLifecycle` passed. Migration `000004_catalog_constraints`
then rolled down once and reapplied once successfully. The normal `bwims`
database was not migrated or used by this validation.

## Remaining work and boundaries

- Capture physical evidence from one real EAN/UPC barcode using the default
  USB/Bluetooth hardware scanner. No real scanner test is claimed here.
- Product and Category frontend screens are a separate slice.
- Admin User Management remains outstanding, so Gantt task 2.2 is only
  partially complete even though FR-6 through FR-9 are implemented.
- Inventory receipts, lots, balances, FEFO picking, FIFO costing, transfers,
  adjustments, and sales are outside this catalog slice. The system still does
  not provide a sales module.
