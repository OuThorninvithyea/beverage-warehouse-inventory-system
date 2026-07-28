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
