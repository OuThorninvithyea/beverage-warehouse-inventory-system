# BWIMS backend

The Week 6 backend foundation uses Go, Fiber v3 and PostgreSQL through `pgxpool`.
It keeps Kaifin's vertical-module pattern while moving shared infrastructure to
`internal/platform` and wiring dependencies explicitly in `internal/app`.

## Architecture

```text
Route -> Handler -> Service -> Repository -> PostgreSQL
```

- `cmd/api`: process entrypoint and graceful shutdown
- `internal/app`: application composition root
- `internal/platform`: configuration, database and shared HTTP behavior
- `internal/modules`: complete vertical feature slices
- `migrations`: versioned PostgreSQL migrations
- `queries`: future sqlc queries generated from the approved schema

## Local setup

```bash
cp backend/.env.example backend/.env
docker compose up -d postgres
docker compose run --rm migrate
docker compose up --build api
```

The schema and relationships are documented in
[`docs/database-erd.md`](../docs/database-erd.md). Migration `000002` creates
the initial application tables, indexes, constraints, seeded roles, and
append-only protections.

Run the API directly from the backend directory when PostgreSQL is already
available:

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

## Verification

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
cd backend && go test ./... && go vet ./...
```

`/health` is a liveness check and does not query PostgreSQL. `/ready` checks the
database and returns HTTP 503 when the application is not ready to serve.

## API envelopes

Success:

```json
{"success":true,"data":{"status":"ok","environment":"development"}}
```

Error:

```json
{"success":false,"error":{"code":"SERVICE_NOT_READY","message":"database is unavailable"}}
```

Internal errors are logged by the centralized Fiber error handler and are not
exposed to API clients.
