# Week 9 admin user management validation

**Validation date:** 2026-08-17
**Branch:** `codex/pro-16-admin-user-management`
**API checkpoint:** `83177d1` (`feat: wire admin user management routes into the application`, rebased onto `origin/main` at `b43ce10` to include the merged catalog module)

## Implemented scope

- Admin-only CRUD for user accounts: create, list, read, update, soft
  deactivate.
- Admin-triggered password reset.
- Search/filter by name, email, role, warehouse, and active status with
  cursor pagination, matching the Catalog and Warehouse response shape.
- Deactivation and password reset both revoke every active refresh token for
  the target user in the same database transaction as the write.
- Last-admin lockout protection enforced inside the same transaction as the
  write that would reduce the active-admin count, closing the
  check-then-act race between concurrent requests.
- Self-deactivation and self-demotion are rejected unconditionally; ordinary
  self field-updates and self password resets remain allowed.
- No new database migration — reuses `users`, `roles`, `refresh_tokens`, and
  `warehouses` tables and indexes from `000002_initial_schema`.

## Automated verification

Commands run from `backend/`:

```text
go build ./...
go vet ./...
go test ./... -count=1
BWIMS_TEST_DATABASE_URL=<local connection string> go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v
git diff --check
docker compose config --quiet
```

Result on 2026-08-17:

- `go build ./...` — PASS.
- `go vet ./...` — PASS, no output.
- `go test ./... -count=1` — all packages passed except one pre-existing,
  unrelated flaky test: `TestTokenManagerRejectsTamperedToken` in
  `internal/modules/auth`. This test tampers the last base64url character of
  an RS256 JWT and expects signature verification to fail; because the final
  base64 character of the RSA signature only encodes a few significant bits,
  the substitution occasionally decodes to the same byte value and the
  tampered token still verifies, producing a non-deterministic failure. This
  is unrelated to this slice: `git diff origin/main -- backend/internal/modules/auth`
  is empty (no files in that package were touched by this branch), and
  rerunning the test in isolation (`go test ./internal/modules/auth/... -run
  TestTokenManagerRejectsTamperedToken -count=5 -v`) shows a mix of PASS and
  FAIL across runs. `internal/modules/app`, `internal/modules/catalog`,
  `internal/modules/health`, `internal/modules/users`, `internal/modules/warehouse`,
  and `internal/platform/config` all passed on every run, including the new
  `TestUsersRoutesAreRegisteredBeforeNotFoundHandler` and
  `TestCatalogRoutesAreRegisteredBeforeNotFoundHandler` (both now present in
  `internal/app/app_test.go` after the rebase merge).
- `TestPostgresUsersLifecycle` — PASS.
- `git diff --check` — passed, no output (no trailing whitespace).
- `docker compose config --quiet` — passed, no output.

Frontend re-run (no frontend code changed in this slice, but confirming no
regression):

```text
npm install
npm test
npm run typecheck
npm run build
```

Result on 2026-08-17:
- `npm test` — 1 test file, 4 tests passed.
- `npm run typecheck` — passed, no output.
- `npm run build` — passed. Vite reported the same pre-existing, non-blocking
  warning that the `BarcodeTestView` chunk is larger than 500 kB after
  minification (unrelated to this slice; also present in the Week 8 catalog
  validation).

## Database migration status

Local PostgreSQL 18 (Homebrew `postgresql@18`, listening on `localhost:5432`)
was used with a fresh, dedicated `bwims_users_test` database, migrated with
the `migrate` CLI (`migrate -path migrations -database <connection string>
up`) through `000004_catalog_constraints` — the same four migrations present
on `main` after this branch was rebased. This slice adds no new migration
file; `ls migrations/ | tail -5` still ends at `000004_catalog_constraints.up.sql`
/ `.down.sql`. `TestPostgresUsersLifecycle`'s `t.Cleanup` deletes its seeded
refresh-token rows, user rows (matched by a per-run email domain suffix), and
warehouse row after each run, so no test data was left behind in the shared
test database. The normal `bwims` database was not touched by this
validation.

## Note on branch history

`codex/pro-16-admin-user-management` was originally created from `main` at
`8f29cbd` (before catalog PR #11 merged), per the implementation plan's Task
1. By the time Tasks 1–14 were committed, `main` had advanced to `b43ce10`
(catalog module merged). Before finishing Tasks 15–19, this branch was
rebased onto `origin/main` so it carries the catalog module and its
`docs/api-contract.md` section; the only conflict was in
`backend/internal/app/app_test.go`, where both branches added a
route-registration test with the same surrounding code — resolved by keeping
both `TestCatalogRoutesAreRegisteredBeforeNotFoundHandler` and
`TestUsersRoutesAreRegisteredBeforeNotFoundHandler`. `backend/internal/app/app.go`
merged automatically with both modules wired in.

## Remaining work and boundaries

- No frontend screen for Admin User Management — that is PRO-11, tracked
  separately, and is explicitly out of scope for this backend slice.
- No self-service registration, self-service password reset, or "forgot
  password" email flow — proposal Section 2 assigns user management to
  Administrators only.
- No per-warehouse scoping of visibility — any admin can see and manage
  every user across every warehouse.
- Real hardware barcode-scanner evidence for issue #6 is unrelated to this
  slice and remains outstanding separately (tracked under PRO-15).
- `TestTokenManagerRejectsTamperedToken` flakiness (see above) is a
  pre-existing issue in the `auth` module, outside this slice's scope; it is
  flagged here as evidence, not fixed as part of this PR.
