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
