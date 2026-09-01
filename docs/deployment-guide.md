# BWIMS deployment guide

This describes how to run BWIMS locally and what a production deployment
needs. It documents what is actually in this repository — `docker-compose.yml`,
`Makefile`, and the `.env.example` files — not an aspirational setup.

## Architecture

Two independently deployable services plus a database:

- **Backend API** (`backend/`) — Go + Fiber v3, serves `/api/v1/*` and the
  unversioned `/health`/`/ready` probes.
- **Frontend SPA** (`frontend/`) — Vue 3 + Vite, built to static files and
  served separately (nginx in the Docker image; a CDN in production).
- **PostgreSQL 16** — single database, `bwims`.

There is no message queue, cache, or third service. Auth is stateless JWT
(RS256); refresh tokens are stored hashed in Postgres, not in Redis or any
external session store.

## Local development (Docker Compose)

This is the fastest path and matches CI.

```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
docker compose up --build
```

This starts, in dependency order: `postgres` (with a healthcheck gate),
`migrate` (runs all `backend/migrations/*.up.sql` files once and exits),
`api` (port 8080), and `frontend` (port 5173, served via nginx inside the
container).

Create the development administrator (required before logging in — no
account exists until this runs):

```bash
make seed-admin
```

Default credentials: `admin@bwims.local` / `ChangeMe123!`. Override via
`SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `SEED_ADMIN_NAME` environment
variables before running the command if shared default credentials are
inappropriate for your environment. The seeder refuses to run unless
`APP_ENV=development` — it cannot be pointed at a production database by
accident.

Verify:

```bash
curl http://localhost:8080/health   # liveness only, does not touch the DB
curl http://localhost:8080/ready    # checks DB connectivity, 503 if not ready
```

Tear down:

```bash
docker compose down          # stop containers, keep the postgres_data volume
docker compose down -v       # also delete all data
```

## Local development (without Docker)

Useful when Postgres is already running locally (e.g. via Homebrew) and you
want faster iteration on the Go binary without a container rebuild each time.

```bash
# Backend
cd backend
cp .env.example .env
# edit .env: point DATABASE_URL at your local Postgres instance
go run ./cmd/api

# Apply migrations directly (requires the golang-migrate CLI: `brew install golang-migrate`)
migrate -path migrations -database "$DATABASE_URL" up

# Seed the admin (requires APP_ENV=development in your shell or .env)
go run ./cmd/seed
```

```bash
# Frontend, in a separate terminal
cd frontend
npm install
npm run dev
```

The Vite dev server proxies `/api`, `/health`, and `/ready` to
`http://localhost:8080` (see `frontend/vite.config.ts`), so the frontend
and backend can run on different ports without CORS configuration during
development.

## Environment variables

### Backend (`backend/.env`)

| Variable | Default (dev) | Notes |
| --- | --- | --- |
| `APP_ENV` | `development` | Must be `development` for the seed command to run at all |
| `API_HOST` | `0.0.0.0` | |
| `API_PORT` | `8080` | |
| `DATABASE_URL` | `postgres://bwims:bwims@127.0.0.1:5432/bwims?sslmode=disable` | Standard `postgres://` DSN |
| `DB_MAX_CONNECTIONS` | `10` | pgxpool max |
| `DB_MIN_CONNECTIONS` | `2` | pgxpool min |
| `DB_CONNECT_TIMEOUT` | `5s` | |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful-shutdown grace period |
| `JWT_ISSUER` | `bwims-api` | |
| `JWT_ACCESS_TTL` | `15m` | |
| `JWT_REFRESH_TTL` | `168h` (7 days) | |
| `JWT_PRIVATE_KEY_BASE64` | *(empty)* | **Required in production.** Base64-encoded PKCS#8 RSA private key. If empty, an ephemeral RS256 key pair is generated at process start — every restart invalidates every existing token, which is fine for local dev and unacceptable in production. |
| `JWT_PUBLIC_KEY_BASE64` | *(empty)* | Base64-encoded PKIX RSA public key, paired with the private key above |

### Seed-only (`backend`, via `docker-compose.yml`'s `seed` service)

| Variable | Default | Notes |
| --- | --- | --- |
| `SEED_ADMIN_EMAIL` | `admin@bwims.local` | |
| `SEED_ADMIN_PASSWORD` | `ChangeMe123!` | Never use this default outside local development |
| `SEED_ADMIN_NAME` | `Development Administrator` | |

### Frontend (`frontend/.env`)

| Variable | Default | Notes |
| --- | --- | --- |
| `VITE_API_BASE_URL` | `/api/v1` | Baked in at build time (Vite env vars are compile-time, not runtime) |

## Database migrations

Migrations live in `backend/migrations/` as paired `NNNNNN_name.up.sql` /
`.down.sql` files, applied with `golang-migrate`. As of this document,
migrations run from `000001_foundation` through `000005_inventory_constraints`.

```bash
make migrate-up      # apply all pending migrations (via Docker)
make migrate-down     # roll back exactly one migration
```

Every migration in this repository has been verified to apply from empty,
roll back one step, and reapply cleanly (see the `docs/week-*-validation.md`
files for the exact commands and dates this was checked).

## Production deployment

The proposal's target architecture (`plan.md`, Key Design Decisions table)
is **separate deployment**: the Go API and the Vue SPA scale and deploy
independently, communicating over standard CORS-enabled HTTPS. This has
**not been executed** — no production environment exists yet. What's needed
when it does:

1. **Generate real RS256 keys** (not the ephemeral dev-generated pair) and
   set `JWT_PRIVATE_KEY_BASE64` / `JWT_PUBLIC_KEY_BASE64`. Losing or
   rotating these invalidates every issued token — plan a rotation strategy,
   not just initial generation.
2. **Managed Postgres** with automated backups (WAL archiving / PITR) —
   the risk register (proposal Section 11, risk "Data loss") calls this out
   explicitly.
3. **HTTPS termination** in front of both the API and the SPA — required
   for the camera-based barcode scanning fallback (`getUserMedia` refuses
   to run over plain HTTP except on `localhost`), and for JWT security in
   general.
4. **Run `migrate up` against the production database** before starting the
   API — the API does not auto-migrate itself in this architecture; the
   `migrate` step is a distinct, explicit action.
5. **Do not run the seed command against production** with default
   credentials. If an initial admin is needed, set unique
   `SEED_ADMIN_EMAIL`/`SEED_ADMIN_PASSWORD` values first.
6. **CI** already runs backend tests+vet, frontend tests+build, and Docker
   Compose config validation on every PR (see `.github/workflows/`) — no
   additional CI setup is needed before deploying, only the deployment
   target itself (hosting, DNS, TLS) which has not been decided or built.

## What this guide does not cover

Reporting/dashboard endpoints, barcode label generation, and the frontend
UI screens do not exist yet, so there is nothing to deploy for them beyond
what's described above. This guide will need a revision once those land.
