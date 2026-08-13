# Category, Product, and Barcode API Design

**Status:** Approved for implementation on 2026-08-11
**Linear scope:** PRO-9
**Gantt scope:** Task 2.2, Product and Category portion
**Branch:** `codex/pro-9-catalog-apis`

## Goal

Deliver the global beverage catalog required by inventory operations: authorized
Category CRUD, authorized Product CRUD, searchable cursor-paginated lists, and
exact EAN-13/UPC-A barcode lookup for the default hardware-scanner flow.

The implementation follows the existing BWIMS structure:

`Route -> Handler -> Service -> Repository -> PostgreSQL`

## Scope

This slice includes:

- Category create, list, view, update, and soft deactivation.
- Product create, list, view, update, and soft deactivation.
- Optional parent categories with cycle prevention.
- Exact product lookup by EAN-13 or UPC-A barcode.
- SKU, name, barcode, category, and active-status search/filtering.
- Cursor pagination using stable `(created_at DESC, id DESC)` ordering.
- Authentication, role authorization, validation, conflict handling, tests,
  migration updates, API documentation, and traceability evidence.

This slice excludes:

- Inventory balances, lots, receive, pick, transfer, and adjustment operations.
- Sales, customers, orders, invoices, payments, or POS behavior.
- Barcode label generation and physical scanner evidence.
- Product pricing, supplier purchasing, and unit conversion.
- Product and Category frontend screens.
- Admin User Management APIs.

Those features remain separate recovery slices so this module is independently
reviewable and testable.

## Architecture

Create one focused catalog module:

```text
backend/internal/modules/catalog/
  model.go
  pagination.go
  barcode.go
  repository.go
  service.go
  handler.go
  route.go
```

Category and Product behavior stays together because both resources share
authorization, category-reference validation, list pagination, normalization,
and database error mapping. Inventory modules will consume Product IDs through
repository/service interfaces rather than reaching into catalog internals.

The module will be wired into `backend/internal/app/app.go` before the fallback
route, matching the Warehouse module.

## Authorization

Every route requires a valid Bearer access token.

| Role | Read/list/search/barcode lookup | Create/update/deactivate |
| --- | --- | --- |
| `admin` | Allowed | Allowed |
| `warehouse_manager` | Allowed | Allowed |
| `picker` | Allowed | Forbidden |
| `viewer` | Allowed | Forbidden |

Categories and Products are global catalog data, so warehouse managers are not
restricted by `warehouse_id` for this module. Picker and viewer mutation
attempts return HTTP 403 with `FORBIDDEN`.

## Data Model

### Category

```json
{
  "id": "<uuid>",
  "parent_id": null,
  "name": "Soft Drinks",
  "is_active": true,
  "created_at": "2026-08-11T08:00:00Z",
  "updated_at": "2026-08-11T08:00:00Z"
}
```

- `name` is required, trimmed, and unique without regard to letter case.
- `parent_id` is optional.
- A parent must exist and be active.
- A category cannot parent itself or create an ancestor/descendant cycle.
- `is_active` defaults to `true`.
- Deactivation is rejected with `CATEGORY_IN_USE` while an active child
  category or active Product references the category.

### Product

```json
{
  "id": "<uuid>",
  "category_id": "<uuid>",
  "sku": "COKE-330-CAN",
  "barcode": "9556001234567",
  "name": "Coca-Cola 330 ml Can",
  "unit": "case",
  "is_lot_tracked": true,
  "is_active": true,
  "created_at": "2026-08-11T08:00:00Z",
  "updated_at": "2026-08-11T08:00:00Z"
}
```

- `sku`, `name`, and `unit` are required.
- SKU is trimmed, normalized to uppercase, and unique without regard to case.
- Name is trimmed.
- Unit is trimmed and normalized to lowercase; the API accepts a nonblank unit
  string without introducing a unit-conversion subsystem.
- `category_id` is optional, but a supplied category must exist and be active.
- Barcode is optional. A supplied value must contain only digits, be a valid
  12-digit UPC-A or 13-digit EAN-13 value, and be globally unique.
- `is_lot_tracked` and `is_active` default to `true` on create.
- Omitted booleans preserve stored values on update.
- Deactivation sets `is_active=false`; Product rows are never physically
  deleted because lots, balances, movements, and cost layers reference them.

## Migration

Add a forward-only migration after `000003` that:

- adds `categories.is_active BOOLEAN NOT NULL DEFAULT TRUE`;
- replaces case-sensitive Product SKU uniqueness with a unique index on
  `LOWER(sku)`;
- preserves the existing global non-null Product barcode unique index;
- adds an index supporting Category parent/active queries;
- adds indexes supporting active Product/category list filters;
- provides a down migration that reverses only these catalog-specific changes.

The migration must apply from an empty database, apply over migrations
`000001`-`000003`, and roll back/reapply cleanly.

## API Surface

All routes use the `/api/v1` prefix and the existing success/error envelopes.

### Category routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/categories` | List accessible categories |
| `POST` | `/api/v1/categories` | Create; admin or manager |
| `GET` | `/api/v1/categories/:category_id` | Read one category |
| `PUT` | `/api/v1/categories/:category_id` | Update; admin or manager |
| `DELETE` | `/api/v1/categories/:category_id` | Soft-deactivate; admin or manager |

Category lists accept `limit`, `after`, `search`, `parent_id`, and `is_active`.
A `parent_id=null` filter is not introduced in this slice; omitting `parent_id`
means all hierarchy levels.

### Product routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/products` | List accessible Products |
| `POST` | `/api/v1/products` | Create; admin or manager |
| `GET` | `/api/v1/products/by-barcode/:barcode` | Exact active-Product lookup |
| `GET` | `/api/v1/products/:product_id` | Read one Product |
| `PUT` | `/api/v1/products/:product_id` | Update; admin or manager |
| `DELETE` | `/api/v1/products/:product_id` | Soft-deactivate; admin or manager |

The barcode route is registered before `/:product_id` so `by-barcode` cannot be
captured as a Product ID.

Product lists accept `limit`, `after`, `search`, `category_id`, and `is_active`.
`search` performs case-insensitive matching across SKU, name, and barcode.

List responses use:

```json
{
  "success": true,
  "data": {
    "items": [],
    "page": {
      "next_cursor": null,
      "has_more": false
    }
  }
}
```

`limit` is 1-100 with a default of 20. Cursors are opaque and encode the final
row's creation time and UUID. Empty lists serialize as `[]`.

## Barcode Lookup Behavior

The hardware scanner enters a barcode as keyboard input and submits it on
Enter. The frontend sends the captured digits to:

`GET /api/v1/products/by-barcode/:barcode`

The API validates the checksum before querying. A valid active Product returns
HTTP 200. Invalid input returns HTTP 422 with `INVALID_BARCODE`; a valid barcode
with no active Product returns HTTP 404 with `PRODUCT_NOT_FOUND`. An inactive
Product is deliberately unavailable through scanner lookup so it cannot be
selected for a later inventory operation.

Lookup is read-only. It never creates a movement or changes stock.

## Update and Deactivation Semantics

- `PUT` accepts a complete business payload while pointer booleans distinguish
  omitted values from explicit `false`.
- Product Category assignment is replaceable. Omitting `category_id` on update
  preserves the stored category; an explicit JSON `null` clears it.
- Category parent assignment follows the same preserve-versus-clear behavior.
- Product barcode assignment also supports preserve, replace, and explicit
  `null` clearing.
- Update request decoding uses a small optional-string input type with separate
  `Set` and `Value` state for `parent_id`, `category_id`, and `barcode`. This
  avoids Go's ordinary pointer decoding ambiguity, where omitted and explicit
  `null` would otherwise both become `nil`.
- `DELETE` is idempotent for an existing inactive resource and returns HTTP
  204.
- Reading an inactive resource by ID remains possible to authenticated users;
  lists include it only when `is_active=false` is requested.

## Error Contract

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, filter, cursor, or limit |
| 403 | `FORBIDDEN` | Role cannot mutate catalog data |
| 404 | `CATEGORY_NOT_FOUND` | Category or requested parent does not exist |
| 404 | `PRODUCT_NOT_FOUND` | Product or active barcode lookup does not exist |
| 409 | `CATEGORY_NAME_CONFLICT` | Category name already exists |
| 409 | `CATEGORY_IN_USE` | Active child or Product prevents deactivation |
| 409 | `PRODUCT_SKU_CONFLICT` | Product SKU already exists |
| 409 | `PRODUCT_BARCODE_CONFLICT` | Product barcode already exists |
| 422 | `INVALID_BARCODE` | Barcode format or checksum is invalid |
| 422 | `CATEGORY_CYCLE` | Parent change would create a hierarchy cycle |
| 422 | `VALIDATION_ERROR` | Required business data is missing or invalid |
| 500 | `CATALOG_OPERATION_FAILED` | Unexpected failure without storage details |

Database constraint errors are mapped with `errors.Is`-compatible sentinel
errors. Raw PostgreSQL messages are never exposed.

## Repository and Transaction Boundaries

The PostgreSQL repository uses `pgx` directly, matching the Warehouse module.

- Create/update statements return the stored normalized resource.
- Category parent validation and cycle checks occur through repository methods
  invoked by the service.
- Category deactivation checks active children and active Products in the same
  transaction as the status update to prevent a race.
- Product create/update locks the referenced Category row while validating that
  it is active. Category deactivation locks the same row before its in-use
  checks, preventing a Product from being attached during deactivation.
- Pagination fetches `limit + 1` rows to determine `has_more`.
- Repository methods always receive `context.Context`.

No generic CRUD framework or reflection layer is introduced.

## Testing Strategy

Implementation follows red-green-refactor. Each production behavior begins
with a failing test.

### Unit tests

- EAN-13 and UPC-A checksum acceptance and rejection.
- Category/Product normalization and required-field validation.
- Admin/manager mutation access and picker/viewer rejection.
- Parent existence, self-parent, and multi-level cycle rejection.
- SKU and barcode conflict mapping.
- Product Category preserve, replace, and clear behavior.
- Boolean default and update-preservation behavior.
- Cursor encode/decode and invalid cursor handling.
- Active-only barcode lookup.

### HTTP tests

- Authentication is required for every catalog route.
- Role boundaries return the expected status and error code.
- Route ordering preserves `/by-barcode/:barcode`.
- Invalid UUID, query, JSON, and barcode inputs are rejected.
- Success responses and empty lists match the API envelope.

### PostgreSQL integration tests

- Category and Product lifecycle behavior.
- Case-insensitive Category name and Product SKU conflicts.
- Global barcode conflicts.
- Transactional Category-in-use protection.
- Stable pagination and filters.
- Migration apply, rollback, and reapply.

### Full validation

- `go test ./...`
- `go vet ./...`
- clean PostgreSQL migration run
- catalog PostgreSQL integration tests
- existing frontend tests, typecheck, and production build
- `git diff --check`
- `docker compose config --quiet`

## Documentation and Completion Evidence

Update:

- `docs/api-contract.md` with Category, Product, and barcode endpoints;
- `docs/requirements-traceability.md` for FR-6 through FR-9;
- validation evidence with exact commands and results.

The slice is complete only when tests pass, the branch is reviewed and merged,
Linear PRO-9 reflects the evidence, and Gantt task 2.2 is updated as partially
complete. Task 2.2 cannot reach 100% until Admin User Management is also done.
