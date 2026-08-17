# BWIMS API contract

This contract is the Week 6 agreement between backend, frontend and QA. Feature
routes added in later weeks must use the `/api/v1` prefix. Operational probes
remain unversioned.

## Content type

- Requests with a body use `Content-Type: application/json`.
- Responses use `Content-Type: application/json`.
- Timestamps will use RFC 3339 in UTC.
- Resource identifiers will be represented as strings in JSON.

## Success envelope

```json
{
  "success": true,
  "data": {}
}
```

## Error envelope

```json
{
  "success": false,
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Human-readable message"
  }
}
```

The HTTP status communicates the protocol result. `error.code` is stable and is
safe for frontend logic; `error.message` is display-oriented. Internal database
or server errors must never be returned to clients.

## Foundation endpoints

### `GET /health`

Process liveness. It returns HTTP 200 while the API process can serve requests.
It intentionally does not query PostgreSQL.

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "environment": "development"
  }
}
```

### `GET /ready`

Service readiness. It pings PostgreSQL and returns HTTP 200 when dependencies
are available or HTTP 503 with `SERVICE_NOT_READY` otherwise.

```json
{
  "success": true,
  "data": {
    "status": "ready",
    "environment": "development",
    "database": "up"
  }
}
```

## Week 7 authentication endpoints

All authentication responses use the standard envelope. Access tokens are
RS256 JWTs sent as `Authorization: Bearer <access_token>`. Refresh tokens are
opaque values, stored hashed by the API, rotated on every refresh, and revoked
on logout.

### `POST /api/v1/auth/login`

```json
{
  "email": "admin@bwims.local",
  "password": "ChangeMe123!"
}
```

Successful response:

```json
{
  "success": true,
  "data": {
    "access_token": "<RS256 JWT>",
    "refresh_token": "<opaque token>",
    "token_type": "Bearer",
    "access_token_expires_at": "2026-07-28T07:00:00Z",
    "refresh_token_expires_at": "2026-08-04T06:45:00Z",
    "user": {
      "id": "<uuid>",
      "email": "admin@bwims.local",
      "full_name": "Development Administrator",
      "role": "admin"
    }
  }
}
```

Wrong credentials return HTTP 401 with `INVALID_CREDENTIALS`. The message does
not reveal whether the email exists.

### `POST /api/v1/auth/refresh`

```json
{
  "refresh_token": "<current opaque token>"
}
```

The response has the same token-pair shape as login. A successful call revokes
the presented refresh token and stores the replacement atomically. Reusing the
old value returns HTTP 401 with `INVALID_REFRESH_TOKEN`.

### `POST /api/v1/auth/logout`

```json
{
  "refresh_token": "<current opaque token>"
}
```

Returns HTTP 204 and revokes the refresh token.

### `GET /api/v1/auth/me`

Requires a valid access token and returns the current public user claims.

### `GET /api/v1/admin/ping`

Requires a valid access token and the `admin` role. It is the Week 7 RBAC
acceptance probe; non-admin roles receive HTTP 403 with `FORBIDDEN`.

## Warehouse and location endpoints

Every endpoint in this section requires a valid Bearer access token.

### Access rules

| Role | Warehouses | Locations |
| --- | --- | --- |
| `admin` | Read and manage every warehouse | Read and manage every location |
| `warehouse_manager` | Read the assigned warehouse | Read and manage locations in the assigned warehouse |
| `picker` | Read the assigned warehouse | Read locations in the assigned warehouse |
| `viewer` | Read the assigned warehouse | Read locations in the assigned warehouse |

Non-admin users cannot access another warehouse. A non-admin token without an
assigned `warehouse_id` receives HTTP 403 with `FORBIDDEN`.

### Warehouse routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/warehouses` | List accessible warehouses |
| `POST` | `/api/v1/warehouses` | Create a warehouse; admin only |
| `GET` | `/api/v1/warehouses/:warehouse_id` | Read an accessible warehouse |
| `PUT` | `/api/v1/warehouses/:warehouse_id` | Update a warehouse; admin only |
| `DELETE` | `/api/v1/warehouses/:warehouse_id` | Soft-deactivate a warehouse; admin only |

Create or update body:

```json
{
  "code": "PP-01",
  "name": "Phnom Penh Main Warehouse",
  "address": "Sen Sok, Phnom Penh",
  "is_active": true
}
```

`code` and `name` are required. Codes are trimmed, normalized to uppercase and
unique without regard to letter case. `is_active` defaults to `true` on create
and preserves its stored value when omitted from an update.

### Location routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/warehouses/:warehouse_id/locations` | List accessible locations |
| `POST` | `/api/v1/warehouses/:warehouse_id/locations` | Create a location; admin or assigned manager |
| `GET` | `/api/v1/warehouses/:warehouse_id/locations/:location_id` | Read an accessible location |
| `PUT` | `/api/v1/warehouses/:warehouse_id/locations/:location_id` | Update a location; admin or assigned manager |
| `DELETE` | `/api/v1/warehouses/:warehouse_id/locations/:location_id` | Soft-deactivate a location; admin or assigned manager |

Create or update body:

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

Location codes are case-insensitively unique inside their warehouse. A
non-null location barcode is globally unique. `is_pickable` and `is_active`
default to `true` on create and preserve their stored values when omitted from
an update.

### List queries and response

Warehouse and location lists accept:

- `limit`: 1–100; default 20.
- `after`: opaque cursor returned by the previous page.
- `search`: case-insensitive business-field search.
- `is_active`: optional `true` or `false`.
- `is_pickable`: optional location-only `true` or `false`.

Lists use stable `(created_at DESC, id DESC)` ordering:

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

Clients must treat `next_cursor` as opaque.

### Deactivation and errors

`DELETE` sets `is_active=false`, updates `updated_at` and returns HTTP 204. It
is idempotent for an existing inactive resource and never physically deletes
the row.

| HTTP status | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, filter, limit or cursor |
| 403 | `FORBIDDEN` | Role or assigned warehouse does not permit access |
| 404 | `WAREHOUSE_NOT_FOUND` | Warehouse does not exist |
| 404 | `LOCATION_NOT_FOUND` | Location does not exist under the warehouse |
| 409 | `WAREHOUSE_CODE_CONFLICT` | Warehouse code already exists |
| 409 | `LOCATION_CODE_CONFLICT` | Location code already exists in the warehouse |
| 409 | `LOCATION_BARCODE_CONFLICT` | Location barcode already exists |
| 422 | `VALIDATION_ERROR` | Required business data is missing or invalid |
| 500 | `WAREHOUSE_OPERATION_FAILED` | Unexpected operation failure |

## Category, product, and barcode endpoints

Every endpoint in this section requires a valid Bearer access token. Categories
and Products are global catalog data rather than warehouse-scoped data.

### Access rules

| Role | Read, list, search, barcode lookup | Create, update, deactivate |
| --- | --- | --- |
| `admin` | Allowed | Allowed |
| `warehouse_manager` | Allowed | Allowed |
| `picker` | Allowed | Forbidden |
| `viewer` | Allowed | Forbidden |

### Category routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/categories` | List and search categories |
| `POST` | `/api/v1/categories` | Create; admin or warehouse manager |
| `GET` | `/api/v1/categories/:category_id` | Read one category |
| `PUT` | `/api/v1/categories/:category_id` | Update; admin or warehouse manager |
| `DELETE` | `/api/v1/categories/:category_id` | Soft-deactivate; admin or warehouse manager |

Create or update body:

```json
{
  "name": "Soft Drinks",
  "parent_id": null,
  "is_active": true
}
```

`name` is required, trimmed, and unique without regard to letter case. A
supplied `parent_id` must identify an active category and cannot create a
hierarchy cycle. On update, omitting `parent_id` preserves the current parent;
an explicit `null` clears it. Deactivation is rejected while an active child
category or Product references the category.

Category lists accept `limit`, `after`, `search`, `parent_id`, and `is_active`.

### Product routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/products` | List and search Products |
| `POST` | `/api/v1/products` | Create; admin or warehouse manager |
| `GET` | `/api/v1/products/by-barcode/:barcode` | Exact active-Product barcode lookup |
| `GET` | `/api/v1/products/:product_id` | Read one Product |
| `PUT` | `/api/v1/products/:product_id` | Update; admin or warehouse manager |
| `DELETE` | `/api/v1/products/:product_id` | Soft-deactivate; admin or warehouse manager |

Create or update body:

```json
{
  "category_id": "<uuid>",
  "sku": "COKE-330-CAN",
  "barcode": "4006381333931",
  "name": "Coca-Cola 330 ml Can",
  "unit": "case",
  "is_lot_tracked": true,
  "is_active": true
}
```

`sku`, `name`, and `unit` are required. SKU is trimmed and normalized to
uppercase; unit is trimmed and normalized to lowercase. SKU uniqueness is
case-insensitive. `category_id` and `barcode` are optional. A supplied Category
must be active. A supplied barcode must be a checksum-valid 12-digit UPC-A or
13-digit EAN-13 value and must be globally unique. On create, `is_lot_tracked`
and `is_active` default to `true`. On update, omitted booleans preserve their
stored values. Omitting `category_id` or `barcode` on update preserves it;
explicit `null` clears it.

Product lists accept `limit`, `after`, `search`, `category_id`, and `is_active`.
`search` performs case-insensitive matching across SKU, name, and barcode. All
catalog lists use the same cursor response shape documented above, with a
default limit of 20 and a maximum of 100.

### Hardware barcode lookup

The default USB/Bluetooth hardware scanner enters barcode digits like a
keyboard and submits on Enter. The client sends the captured value to:

`GET /api/v1/products/by-barcode/:barcode`

The lookup validates the UPC-A/EAN-13 checksum before querying and returns only
an active Product. It is read-only: it never creates a stock movement or
changes inventory. Phone-camera scanning remains an optional client input
method using the same endpoint.

### Deactivation and errors

`DELETE` sets `is_active=false` and returns HTTP 204. It is idempotent for an
existing inactive resource. Inactive resources remain readable by ID, but an
inactive Product is unavailable through barcode lookup.

| HTTP status | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, filter, limit, or cursor |
| 403 | `FORBIDDEN` | Role cannot mutate catalog data |
| 404 | `CATEGORY_NOT_FOUND` | Category or requested parent does not exist |
| 404 | `PRODUCT_NOT_FOUND` | Product or active barcode lookup does not exist |
| 409 | `CATEGORY_NAME_CONFLICT` | Category name already exists |
| 409 | `CATEGORY_IN_USE` | Active child or Product prevents deactivation |
| 409 | `PRODUCT_SKU_CONFLICT` | Product SKU already exists |
| 409 | `PRODUCT_BARCODE_CONFLICT` | Product barcode already exists |
| 422 | `INVALID_BARCODE` | Barcode format or checksum is invalid |
| 422 | `CATEGORY_CYCLE` | Parent change would create a hierarchy cycle |
| 422 | `VALIDATION_ERROR` | Required catalog data is missing or invalid |
| 500 | `CATALOG_OPERATION_FAILED` | Unexpected operation failure |

## Admin user management endpoints

Every endpoint in this section requires a valid Bearer access token, and every
endpoint — including reads — is restricted to the `admin` role. This is
stricter than Catalog and Warehouse, where reads are open to all
authenticated roles.

### Access rules

| Role | List/read users | Create/update/deactivate/reset password |
| --- | --- | --- |
| `admin` | Allowed | Allowed |
| `warehouse_manager` | Forbidden | Forbidden |
| `picker` | Forbidden | Forbidden |
| `viewer` | Forbidden | Forbidden |

### User routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/users` | List and search users; admin only |
| `POST` | `/api/v1/users` | Create a user; admin only |
| `GET` | `/api/v1/users/:user_id` | Read one user; admin only |
| `PUT` | `/api/v1/users/:user_id` | Update a user; admin only |
| `DELETE` | `/api/v1/users/:user_id` | Soft-deactivate a user; admin only |
| `POST` | `/api/v1/users/:user_id/password-reset` | Admin sets a new password; admin only |

Create request body:

```json
{
  "email": "picker2@bwims.test",
  "full_name": "New Picker",
  "role": "picker",
  "warehouse_id": null,
  "password": "at-least-12-characters"
}
```

`email` is required, trimmed, and compared case-insensitively against existing
accounts. `full_name` is required and trimmed. `role` must be one of `admin`,
`warehouse_manager`, `picker`, or `viewer`. `warehouse_id`, if supplied, must
reference an existing, active warehouse. `password` is write-only, requires at
least 12 characters, and is never returned in any response.

Update request body (PUT is a full business-field replace; `warehouse_id` uses
tri-state semantics — omit to keep the current value, send `null` to clear
it, or send a UUID to replace it):

```json
{
  "full_name": "New Picker Name",
  "role": "picker",
  "warehouse_id": null,
  "is_active": true
}
```

Email is not updatable through this endpoint in this slice.

Password reset request body:

```json
{ "password": "at-least-12-characters" }
```

A successful `DELETE` or password reset returns `204 No Content` and
immediately revokes every active refresh token for that user, forcing
re-authentication.

### List queries and response

Accepts `limit` (1-100, default 20), `after` (opaque cursor), `search`
(case-insensitive match across email and full name), `role`, `warehouse_id`,
and `is_active`. Response shape matches Catalog and Warehouse:

```json
{
  "success": true,
  "data": {
    "items": [],
    "page": { "next_cursor": null, "has_more": false }
  }
}
```

### Deactivation, lockout protection, and errors

Deactivating the sole active administrator, or updating the sole active
administrator's role away from `admin`, is rejected with
`LAST_ADMIN_PROTECTED`. An administrator can never deactivate or demote
themselves — `SELF_DEACTIVATION_FORBIDDEN` — another administrator must do
it; ordinary field updates and self password resets remain allowed.

| HTTP status | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, filter, limit, or cursor |
| 403 | `FORBIDDEN` | Caller is not `admin` |
| 404 | `USER_NOT_FOUND` | User does not exist |
| 404 | `WAREHOUSE_NOT_FOUND` | Supplied `warehouse_id` does not reference an active warehouse |
| 409 | `EMAIL_CONFLICT` | Email already exists (case-insensitive) |
| 409 | `LAST_ADMIN_PROTECTED` | Change would leave zero active administrators |
| 409 | `SELF_DEACTIVATION_FORBIDDEN` | Admin cannot deactivate or demote themselves |
| 422 | `INVALID_ROLE` | Role is not one of the four known role codes |
| 422 | `VALIDATION_ERROR` | Required business data missing or invalid (e.g. password under 12 characters) |
| 500 | `USER_OPERATION_FAILED` | Unexpected failure without storage details |

## Initial HTTP status policy

| HTTP status | Usage |
| --- | --- |
| 200 | Successful read or update |
| 201 | Resource created |
| 204 | Successful request with no response body |
| 400 | Invalid request syntax or validation |
| 401 | Missing or invalid authentication |
| 403 | Authenticated but not permitted |
| 404 | Resource or route not found |
| 409 | Resource state conflict |
| 422 | Semantically invalid operation |
| 500 | Unexpected server failure |
| 503 | Required dependency unavailable |
