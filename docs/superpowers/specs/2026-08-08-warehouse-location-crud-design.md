# Warehouse and Location CRUD Design

**Date:** 2026-08-08  
**Linear scope:** PRO-8, supported by PRO-13  
**Milestone:** Weeks 7–8 — Authentication and Core CRUD recovery

## Objective

Implement the first authenticated core-domain slice for BWIMS: warehouse and
location management. The feature must preserve the existing `/api/v1` contract,
role model, response envelopes, and `Route → Handler → Service → Repository →
PostgreSQL` architecture.

This design deliberately excludes product, category, inventory-balance, stock
movement, frontend CRUD, and physical barcode-scanner work. Those remain
separate Linear issues and build on this API.

## Selected approach

Create one `warehouse` domain module with separate files for warehouse and
location HTTP behavior. Warehouses and locations share access-scoping rules,
and a location cannot exist outside a warehouse, so keeping them in one module
avoids duplicating authorization and ownership logic.

The module will contain:

- `model.go`: API and service domain types.
- `repository.go`: repository interface and PostgreSQL implementation.
- `service.go`: validation, normalization, access scoping, and domain errors.
- `handler.go`: request parsing and HTTP error mapping.
- `route.go`: authenticated route registration and role gates.
- focused `_test.go` files for service and handler behavior.

A generic CRUD framework will not be introduced. Shared abstractions will be
added only when another feature demonstrates genuine repetition.

## Permissions and data scope

The access token claims remain the source of the caller's role and assigned
warehouse.

| Role | Warehouse access | Location access |
| --- | --- | --- |
| `admin` | List, view, create, update, deactivate all warehouses | List, view, create, update, deactivate all locations |
| `warehouse_manager` | View only the assigned warehouse | List, view, create, update, deactivate locations in the assigned warehouse |
| `picker` | View only the assigned warehouse | List and view locations in the assigned warehouse |
| `viewer` | View only the assigned warehouse | List and view locations in the assigned warehouse |

A non-admin token without `warehouse_id` cannot access warehouse or location
resources. Requests for another warehouse return `403 FORBIDDEN`; they do not
leak whether the resource exists.

## HTTP endpoints

All endpoints require a valid Bearer access token.

### Warehouses

| Method | Route | Allowed roles | Result |
| --- | --- | --- | --- |
| `GET` | `/api/v1/warehouses` | all authenticated roles | Admin lists all; other roles receive only their assigned warehouse |
| `POST` | `/api/v1/warehouses` | admin | Create a warehouse |
| `GET` | `/api/v1/warehouses/:warehouse_id` | all authenticated roles | Get an accessible warehouse |
| `PUT` | `/api/v1/warehouses/:warehouse_id` | admin | Replace editable warehouse fields |
| `DELETE` | `/api/v1/warehouses/:warehouse_id` | admin | Soft-deactivate the warehouse |

### Locations

| Method | Route | Allowed roles | Result |
| --- | --- | --- | --- |
| `GET` | `/api/v1/warehouses/:warehouse_id/locations` | all authenticated roles | List accessible locations in the warehouse |
| `POST` | `/api/v1/warehouses/:warehouse_id/locations` | admin, warehouse manager | Create a location |
| `GET` | `/api/v1/warehouses/:warehouse_id/locations/:location_id` | all authenticated roles | Get an accessible location |
| `PUT` | `/api/v1/warehouses/:warehouse_id/locations/:location_id` | admin, warehouse manager | Replace editable location fields |
| `DELETE` | `/api/v1/warehouses/:warehouse_id/locations/:location_id` | admin, warehouse manager | Soft-deactivate the location |

`DELETE` never physically removes a warehouse or location. It sets
`is_active=false` and updates `updated_at`, preserving references and auditability.

## Request and response models

### Warehouse write request

```json
{
  "code": "PP-01",
  "name": "Phnom Penh Main Warehouse",
  "address": "Sen Sok, Phnom Penh",
  "is_active": true
}
```

`code` and `name` are required. Codes are trimmed and normalized to uppercase.
An empty address is stored as `NULL`. `is_active` defaults to `true` on create.

### Location write request

```json
{
  "code": "A-01-R02-S03",
  "zone": "Ambient",
  "aisle": "A-01",
  "rack": "R02",
  "shelf": "S03",
  "barcode": "LOC000001",
  "is_pickable": true,
  "is_active": true
}
```

`code` is required and normalized to uppercase. Optional text values are
trimmed; empty optional values are stored as `NULL`. Boolean fields default to
`true` on create.

Resource responses expose string UUIDs and RFC 3339 UTC timestamps through the
existing success envelope:

```json
{
  "success": true,
  "data": {
    "id": "7e5d55b1-6356-492b-8296-2b981867fcf2",
    "code": "PP-01",
    "name": "Phnom Penh Main Warehouse",
    "address": "Sen Sok, Phnom Penh",
    "is_active": true,
    "created_at": "2026-08-08T08:00:00Z",
    "updated_at": "2026-08-08T08:00:00Z"
  }
}
```

## Listing, filtering, and pagination

Warehouse and location lists support:

- `limit`: default 20, minimum 1, maximum 100.
- `after`: an opaque cursor produced by the previous response.
- `search`: case-insensitive matching against business identifiers and names.
- `is_active`: optional `true` or `false` filter.

Locations additionally support `is_pickable`.

Results use stable `(created_at DESC, id DESC)` ordering. The cursor encodes the
last row's timestamp and ID; clients must treat it as opaque.

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

The repository fetches `limit + 1` rows to determine `has_more`. No total count
is required for this slice.

## Database changes

The existing tables already contain the required columns and relationships. A
forward-only `000003` migration will harden business-key uniqueness:

- replace the case-sensitive warehouse-code uniqueness rule with a unique
  index on `LOWER(code)`;
- replace the case-sensitive per-warehouse location-code rule with a unique
  index on `(warehouse_id, LOWER(code))`;
- preserve the existing partial unique location-barcode index;
- add no new tables.

Application write queries explicitly set `updated_at=NOW()` on updates and
deactivation. A global timestamp trigger is unnecessary for this slice.

## Service and repository behavior

Handlers only bind HTTP input, parse path/query values, retrieve token claims,
call the service, and map domain errors.

The service owns:

- role and assigned-warehouse scoping;
- required-field validation;
- trimming and code normalization;
- pagination validation and cursor decoding;
- create, update, read, list, and deactivate use cases;
- domain error classification.

The repository owns SQL only. Every location query includes both
`location_id` and `warehouse_id` where applicable so a location cannot be
accessed through the wrong parent route.

## Error contract

Errors use the existing failure envelope. Internal PostgreSQL errors are never
returned to clients.

| Status | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, Boolean filter, limit, or cursor syntax |
| 401 | `UNAUTHENTICATED` | Missing or invalid access token |
| 403 | `FORBIDDEN` | Role or assigned-warehouse scope does not permit the action |
| 404 | `WAREHOUSE_NOT_FOUND` | Accessible warehouse does not exist |
| 404 | `LOCATION_NOT_FOUND` | Accessible location does not exist under the warehouse |
| 409 | `WAREHOUSE_CODE_CONFLICT` | Normalized warehouse code already exists |
| 409 | `LOCATION_CODE_CONFLICT` | Normalized location code already exists in the warehouse |
| 409 | `LOCATION_BARCODE_CONFLICT` | Location barcode already exists |
| 422 | `VALIDATION_ERROR` | Required or semantic business validation failed |
| 500 | `WAREHOUSE_OPERATION_FAILED` | Unexpected storage or service failure |

Deactivating an already inactive resource is idempotent and returns `204`.

## Testing strategy

Implementation follows red-green-refactor.

1. Service tests define normalization, validation, access scoping, cursor
   behavior, conflict propagation, and not-found behavior.
2. Handler tests define request binding, status codes, response envelopes, and
   role restrictions through real Fiber routes.
3. Repository queries are exercised against the Docker PostgreSQL service to
   verify uniqueness, nested location ownership, pagination, and soft delete.
4. The complete backend suite runs with `go test ./...` and `go vet ./...`.
5. Existing authentication acceptance behavior must remain unchanged.

## Delivery boundary

This design is complete when the migrations and authenticated Warehouse and
Location APIs pass their automated and PostgreSQL-backed checks, the API
contract is documented, and Linear PRO-8/PRO-13 contain truthful evidence.

Frontend screens, Product/Category APIs, User Management, inventory movements,
and physical-scanner evidence are intentionally not part of this branch.
