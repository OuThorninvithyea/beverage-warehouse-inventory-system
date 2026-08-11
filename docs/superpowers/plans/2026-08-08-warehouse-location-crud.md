# Warehouse and Location CRUD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver authenticated, role-scoped Warehouse and Location CRUD APIs with cursor pagination, soft deactivation, PostgreSQL persistence, tests, and API documentation.

**Architecture:** Add one focused `internal/modules/warehouse` domain module that follows the repository's `Route → Handler → Service → Repository → PostgreSQL` pattern. The service owns validation, normalization, cursor handling, and warehouse scoping; handlers own HTTP translation; the PostgreSQL repository owns SQL.

**Tech Stack:** Go 1.26.1, Fiber v3.4.0, pgx/v5, PostgreSQL 16, Go standard-library testing, Docker Compose, golang-migrate.

## Global Constraints

- Preserve the `/api/v1` prefix and existing `{success,data}` / `{success:false,error}` envelopes.
- Keep UUIDs as JSON strings and timestamps as RFC 3339 UTC values.
- Admin manages every warehouse and location.
- Warehouse managers may mutate locations only in their assigned warehouse.
- Pickers and viewers may read only their assigned warehouse and locations.
- A non-admin without an assigned warehouse receives `403 FORBIDDEN`.
- `DELETE` is an idempotent soft deactivation; it never removes a row.
- List endpoints use `limit` 1–100, opaque `after` cursors, and stable `(created_at DESC, id DESC)` ordering.
- Do not add a generic CRUD framework or unrelated frontend/product/inventory work.
- Write and run a failing test before each production behavior.

## File map

- Create `backend/migrations/000003_warehouse_location_constraints.up.sql`: case-insensitive warehouse and location business-key constraints.
- Create `backend/migrations/000003_warehouse_location_constraints.down.sql`: restore the previous named constraints.
- Create `backend/internal/modules/warehouse/model.go`: resource, actor, input, filter, cursor, and page types.
- Create `backend/internal/modules/warehouse/pagination.go`: cursor encoding and decoding.
- Create `backend/internal/modules/warehouse/repository.go`: repository contract and pgx implementation.
- Create `backend/internal/modules/warehouse/service.go`: use cases, validation, scoping, and domain errors.
- Create `backend/internal/modules/warehouse/handler.go`: HTTP request/query parsing and error mapping.
- Create `backend/internal/modules/warehouse/route.go`: authenticated and role-gated routes.
- Create `backend/internal/modules/warehouse/pagination_test.go`: cursor behavior.
- Create `backend/internal/modules/warehouse/service_test.go`: business rules and access scope.
- Create `backend/internal/modules/warehouse/handler_test.go`: Fiber status and envelope behavior.
- Create `backend/internal/modules/warehouse/repository_integration_test.go`: opt-in PostgreSQL query and constraint verification.
- Modify `backend/internal/app/app.go`: compose and register the new module.
- Modify `docs/api-contract.md`: record endpoints, bodies, list envelopes, permissions, and error codes.
- Modify `docs/requirements-traceability.md`: advance FR-5 evidence to implemented API tests.

---

### Task 1: Domain models and opaque cursor

**Files:**
- Create: `backend/internal/modules/warehouse/model.go`
- Create: `backend/internal/modules/warehouse/pagination.go`
- Test: `backend/internal/modules/warehouse/pagination_test.go`

**Interfaces:**
- Produces: `Actor`, `Warehouse`, `Location`, `WarehouseInput`, `LocationInput`, `ListFilter`, `Page[T]`, `EncodeCursor(time.Time, string) string`, and `DecodeCursor(string) (Cursor, error)`.
- Consumes: Go standard library only.

- [ ] **Step 1: Write the failing cursor tests**

```go
func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{CreatedAt: time.Date(2026, 8, 8, 8, 0, 0, 123, time.UTC), ID: "7e5d55b1-6356-492b-8296-2b981867fcf2"}
	got, err := DecodeCursor(EncodeCursor(want.CreatedAt, want.ID))
	if err != nil { t.Fatalf("DecodeCursor() error = %v", err) }
	if got != want { t.Fatalf("cursor = %#v, want %#v", got, want) }
}

func TestDecodeCursorRejectsMalformedValue(t *testing.T) {
	if _, err := DecodeCursor("not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("DecodeCursor() error = %v, want ErrInvalidCursor", err)
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd backend && go test ./internal/modules/warehouse -run 'Test(CursorRoundTrip|DecodeCursorRejectsMalformedValue)'`

Expected: compilation fails because `Cursor`, `EncodeCursor`, `DecodeCursor`, and `ErrInvalidCursor` do not exist.

- [ ] **Step 3: Add the minimal models and cursor implementation**

Define the public shapes exactly as follows:

```go
type Actor struct { Role string; WarehouseID *string }
type Cursor struct { CreatedAt time.Time; ID string }
type WarehouseInput struct {
	Code string
	Name string
	Address *string
	IsActive *bool
}
type LocationInput struct {
	Code string
	Zone *string
	Aisle *string
	Rack *string
	Shelf *string
	Barcode *string
	IsPickable *bool
	IsActive *bool
}
type ListFilter struct {
	Limit int
	After *Cursor
	Search string
	IsActive *bool
	IsPickable *bool
}
type PageInfo struct {
	NextCursor *string `json:"next_cursor"`
	HasMore bool `json:"has_more"`
}
type Page[T any] struct {
	Items []T `json:"items"`
	Page PageInfo `json:"page"`
}
```

Encode `created_at` in RFC3339Nano plus the UUID separated by a newline, then use `base64.RawURLEncoding`. Reject empty IDs, invalid base64, missing separators, invalid timestamps, and extra fields with `ErrInvalidCursor`.

- [ ] **Step 4: Run the focused and complete module tests**

Run: `cd backend && go test ./internal/modules/warehouse`

Expected: PASS.

- [ ] **Step 5: Commit the cursor foundation**

```bash
git add backend/internal/modules/warehouse
git commit -m "feat: add warehouse domain pagination"
```

---

### Task 2: Warehouse service behavior

**Files:**
- Create: `backend/internal/modules/warehouse/service.go`
- Create: `backend/internal/modules/warehouse/repository.go` with the interface first; PostgreSQL methods remain for Task 4.
- Test: `backend/internal/modules/warehouse/service_test.go`

**Interfaces:**
- Consumes: Task 1's `Actor`, `Warehouse`, `WarehouseInput`, `ListFilter`, and `Page[Warehouse]`.
- Produces: `Service` methods `ListWarehouses`, `GetWarehouse`, `CreateWarehouse`, `UpdateWarehouse`, and `DeactivateWarehouse`; repository methods with the same persistence intent.

- [ ] **Step 1: Write failing tests for warehouse normalization and admin creation**

```go
func TestCreateWarehouseNormalizesBusinessFields(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	address := "  Sen Sok  "
	got, err := service.CreateWarehouse(context.Background(), Actor{Role: auth.RoleAdmin}, WarehouseInput{
		Code: " pp-01 ", Name: " Phnom Penh Main ", Address: &address,
	})
	if err != nil { t.Fatalf("CreateWarehouse() error = %v", err) }
	if repository.createdWarehouse.Code != "PP-01" { t.Fatalf("code = %q", repository.createdWarehouse.Code) }
	if got.Name != "Phnom Penh Main" { t.Fatalf("name = %q", got.Name) }
}
```

Add separate tests proving: non-admin create returns `ErrForbidden`; blank code/name returns `ErrValidation`; duplicate code is preserved as `ErrWarehouseCodeConflict`; create defaults `IsActive=true`; and an empty address becomes `nil`.

- [ ] **Step 2: Run the warehouse service tests and verify RED**

Run: `cd backend && go test ./internal/modules/warehouse -run 'Test(CreateWarehouse|Warehouse)'`

Expected: compilation fails because `NewService`, service methods, repository contract, and domain errors do not exist.

- [ ] **Step 3: Implement minimal warehouse service behavior**

Use these signatures:

```go
type Service interface {
	ListWarehouses(context.Context, Actor, ListFilter) (Page[Warehouse], error)
	GetWarehouse(context.Context, Actor, string) (Warehouse, error)
	CreateWarehouse(context.Context, Actor, WarehouseInput) (Warehouse, error)
	UpdateWarehouse(context.Context, Actor, string, WarehouseInput) (Warehouse, error)
	DeactivateWarehouse(context.Context, Actor, string) error
	ListLocations(context.Context, Actor, string, ListFilter) (Page[Location], error)
	GetLocation(context.Context, Actor, string, string) (Location, error)
	CreateLocation(context.Context, Actor, string, LocationInput) (Location, error)
	UpdateLocation(context.Context, Actor, string, string, LocationInput) (Location, error)
	DeactivateLocation(context.Context, Actor, string, string) error
}
```

Admin list scope passes `nil` to the repository; every other role must have a non-empty assigned warehouse and passes that ID as the repository scope. Validate UUID syntax with `github.com/google/uuid` before repository calls.

- [ ] **Step 4: Add and run tests for read/list/update/deactivate scope**

Cover admin all-warehouse listing, manager assigned-warehouse listing, cross-warehouse `ErrForbidden`, missing assignment `ErrForbidden`, not found propagation, limit defaults/caps, and idempotent deactivation.

Run: `cd backend && go test ./internal/modules/warehouse -run Warehouse`

Expected: PASS.

- [ ] **Step 5: Commit warehouse service behavior**

```bash
git add backend/internal/modules/warehouse
git commit -m "feat: add warehouse service rules"
```

---

### Task 3: Location service behavior

**Files:**
- Modify: `backend/internal/modules/warehouse/service.go`
- Modify: `backend/internal/modules/warehouse/repository.go`
- Modify: `backend/internal/modules/warehouse/service_test.go`

**Interfaces:**
- Consumes: Task 2's service and repository contracts.
- Produces: role-scoped nested Location CRUD behavior.

- [ ] **Step 1: Write a failing manager-scope test**

```go
func TestManagerCreatesLocationOnlyInAssignedWarehouse(t *testing.T) {
	assigned := "7e5d55b1-6356-492b-8296-2b981867fcf2"
	repository := &fakeRepository{}
	service := NewService(repository)
	_, err := service.CreateLocation(context.Background(), Actor{
		Role: auth.RoleWarehouseManager, WarehouseID: &assigned,
	}, "a0f31ce4-8c52-4441-9904-8088c47757fd", LocationInput{Code: "A-01"})
	if !errors.Is(err, ErrForbidden) { t.Fatalf("error = %v, want ErrForbidden", err) }
}
```

Add separate tests proving managers can mutate their assigned warehouse, pickers/viewers cannot mutate, all four roles can read within scope, code is uppercased, optional strings are trimmed to `nil`, and duplicate code/barcode conflicts remain distinguishable.

- [ ] **Step 2: Run location tests and verify RED**

Run: `cd backend && go test ./internal/modules/warehouse -run Location`

Expected: failing assertions because location access and normalization are not implemented.

- [ ] **Step 3: Implement the minimal location service methods**

Enforce warehouse scope before checking repository existence. Pass both `warehouseID` and `locationID` to every single-location repository method. Default `IsPickable` and `IsActive` to true on create; preserve the stored values on update when the corresponding input pointers are omitted.

- [ ] **Step 4: Run all service tests**

Run: `cd backend && go test ./internal/modules/warehouse -run 'Test(CreateLocation|UpdateLocation|GetLocation|ListLocations|DeactivateLocation|Manager)'`

Expected: PASS.

- [ ] **Step 5: Commit location service behavior**

```bash
git add backend/internal/modules/warehouse
git commit -m "feat: add scoped location service rules"
```

---

### Task 4: PostgreSQL migration and repository

**Files:**
- Create: `backend/migrations/000003_warehouse_location_constraints.up.sql`
- Create: `backend/migrations/000003_warehouse_location_constraints.down.sql`
- Complete: `backend/internal/modules/warehouse/repository.go`
- Test: `backend/internal/modules/warehouse/repository_integration_test.go`

**Interfaces:**
- Consumes: Task 2's complete `Repository` interface and Task 1's cursor/filter types.
- Produces: `NewPostgresRepository(*pgxpool.Pool) Repository`.

- [ ] **Step 1: Write the opt-in failing PostgreSQL integration test**

The test reads `BWIMS_TEST_DATABASE_URL`; when unset it calls `t.Skip`. When set, it connects with `pgxpool.New`, truncates `locations` and dependent test rows, and asserts:

```go
first, err := repository.CreateWarehouse(ctx, WarehouseInput{Code: "PP-01", Name: "Main", IsActive: boolPtr(true)})
if err != nil { t.Fatalf("CreateWarehouse() error = %v", err) }
_, err = repository.CreateWarehouse(ctx, WarehouseInput{Code: "pp-01", Name: "Duplicate", IsActive: boolPtr(true)})
if !errors.Is(err, ErrWarehouseCodeConflict) { t.Fatalf("error = %v, want code conflict", err) }
```

Add assertions for nested location ownership, duplicate case-insensitive location code, duplicate barcode, stable two-page cursor traversal, and idempotent deactivation.

- [ ] **Step 2: Run the repository test and verify the first RED state**

Run: `cd backend && go test ./internal/modules/warehouse -run TestPostgresRepository -count=1`

Expected: compilation fails because `NewPostgresRepository` and its methods do not exist.

- [ ] **Step 3: Implement the repository SQL**

Use `errors.Is(err, pgx.ErrNoRows)` for not found. Use `*pgconn.PgError` constraint names to return the three domain conflict errors. List queries fetch `limit+1`, use parameterized optional filters, and apply:

```sql
AND ($after_time::timestamptz IS NULL OR (created_at, id) < ($after_time, $after_id::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $limit_plus_one
```

All location-by-ID operations include `WHERE id=$1 AND warehouse_id=$2`.

- [ ] **Step 4: Apply only migrations 000001–000002 and verify the constraint-specific RED state**

Run PostgreSQL, recreate the test database, apply through migration 2, then run:

`cd backend && BWIMS_TEST_DATABASE_URL='postgres://bwims:bwims@localhost:5432/bwims?sslmode=disable' go test ./internal/modules/warehouse -run TestPostgresRepository -count=1`

Expected: the lower-case duplicate warehouse or location code is accepted, proving the hardening migration is missing.

- [ ] **Step 5: Add migration 000003**

The up migration must:

```sql
ALTER TABLE warehouses DROP CONSTRAINT warehouses_code_key;
CREATE UNIQUE INDEX warehouses_code_lower_unique_idx ON warehouses (LOWER(code));
ALTER TABLE locations DROP CONSTRAINT locations_warehouse_code_unique;
CREATE UNIQUE INDEX locations_warehouse_code_lower_unique_idx ON locations (warehouse_id, LOWER(code));
```

The down migration drops the functional indexes and restores the original named unique constraints.

- [ ] **Step 6: Apply migration 000003 and verify GREEN**

Run the same `TestPostgresRepository` command.

Expected: PASS.

- [ ] **Step 7: Run module tests and commit**

Run: `cd backend && go test ./internal/modules/warehouse`

```bash
git add backend/migrations backend/internal/modules/warehouse
git commit -m "feat: persist warehouses and locations"
```

---

### Task 5: HTTP handlers, routing, and RBAC

**Files:**
- Create: `backend/internal/modules/warehouse/handler.go`
- Create: `backend/internal/modules/warehouse/route.go`
- Create: `backend/internal/modules/warehouse/handler_test.go`

**Interfaces:**
- Consumes: Task 2/3 `Service`; existing `auth.Authenticate`, `auth.RequireRoles`, `auth.ClaimsFromContext`, and `httpx` envelopes.
- Produces: `NewHandler(Service) *Handler` and `RegisterRoutes(*fiber.App, *Handler, *auth.TokenManager)`.

- [ ] **Step 1: Write failing route and envelope tests**

Create a Fiber test app with the existing error handler and real development token manager. Register routes against a fake service. Assert:

```go
response := authenticatedRequest(t, app, manager, auth.User{ID: "admin", Role: auth.RoleAdmin}, "POST", "/api/v1/warehouses", `{"code":"PP-01","name":"Main"}`)
if response.StatusCode != fiber.StatusCreated { /* include response body */ }
```

Add focused tests for missing token `401`, viewer create `403`, malformed JSON `400`, blank fields `422`, code conflict `409`, missing warehouse `404`, manager cross-warehouse `403`, successful create `201`, reads/updates `200`, and deactivation `204`.

- [ ] **Step 2: Run handler tests and verify RED**

Run: `cd backend && go test ./internal/modules/warehouse -run Handler`

Expected: compilation fails because Handler and RegisterRoutes do not exist.

- [ ] **Step 3: Implement route registration**

Register exactly the paths from the design. Apply `Authenticate` to the group, `RequireRoles(admin)` to warehouse mutations, and `RequireRoles(admin, warehouse_manager)` to location mutations. Read routes rely on the service for assigned-warehouse scope.

- [ ] **Step 4: Implement binding and domain-error mapping**

Handlers derive `Actor` from claims. Parse `limit`, `after`, `search`, `is_active`, and `is_pickable` explicitly. Map errors with `errors.Is` to the exact status/code table from the design and wrap unexpected failures with `WAREHOUSE_OPERATION_FAILED` without exposing the storage error.

- [ ] **Step 5: Run handler and module tests**

Run: `cd backend && go test ./internal/modules/warehouse`

Expected: PASS.

- [ ] **Step 6: Commit HTTP behavior**

```bash
git add backend/internal/modules/warehouse
git commit -m "feat: expose warehouse and location APIs"
```

---

### Task 6: Application composition, documentation, and full verification

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `docs/api-contract.md`
- Modify: `docs/requirements-traceability.md`
- Test: all backend packages and Docker-backed repository checks.

**Interfaces:**
- Consumes: `warehouse.NewPostgresRepository`, `warehouse.NewService`, `warehouse.NewHandler`, and `warehouse.RegisterRoutes`.
- Produces: running API routes in the application composition root and truthful delivery evidence.

- [ ] **Step 1: Write a failing application route registration test**

Add `backend/internal/app/app_test.go` if it does not exist. Build the app with test config and a test PostgreSQL pool, issue an admin token through the login flow, and assert `GET /api/v1/warehouses` is no longer `404 ROUTE_NOT_FOUND`.

- [ ] **Step 2: Run the app test and verify RED**

Run: `cd backend && BWIMS_TEST_DATABASE_URL='postgres://bwims:bwims@localhost:5432/bwims?sslmode=disable' go test ./internal/app -run TestWarehouseRoutesRegistered -count=1`

Expected: FAIL with `404 ROUTE_NOT_FOUND`.

- [ ] **Step 3: Wire the module into `app.New`**

Construct the repository, service, and handler once, then register routes after authentication configuration and before the final not-found middleware.

- [ ] **Step 4: Document the delivered contract**

Add request/response examples, query parameters, list page shape, permission matrix, soft-delete rule, and exact error codes to `docs/api-contract.md`. Change FR-5 in `docs/requirements-traceability.md` from schema-only evidence to migration + service + handler + PostgreSQL repository tests.

- [ ] **Step 5: Format and run all automated verification**

Run:

```bash
make backend-format
cd backend && go test ./...
cd backend && go vet ./...
git diff --check
docker compose config --quiet
```

With PostgreSQL running, also run:

```bash
cd backend && BWIMS_TEST_DATABASE_URL='postgres://bwims:bwims@localhost:5432/bwims?sslmode=disable' go test ./internal/modules/warehouse ./internal/app -count=1
```

Expected: every command exits 0.

- [ ] **Step 6: Commit the complete backend slice**

```bash
git add backend/internal/app docs/api-contract.md docs/requirements-traceability.md
git commit -m "docs: record warehouse CRUD contract"
```

- [ ] **Step 7: Update delivery tracking only from evidence**

Add a Linear comment to PRO-13 with the migration and PostgreSQL test evidence. Move PRO-13 to Done only if those checks passed. Add a Linear comment to PRO-8 with endpoint and test evidence; move it to Done only when the full suite and live PostgreSQL checks pass. Do not close hardware-scanner Issue #6 or unrelated Week 8 issues.

---

## Final acceptance checklist

- [ ] Migration 000003 applies and rolls back cleanly.
- [ ] Case-insensitive warehouse and per-warehouse location codes are unique.
- [ ] Location barcodes remain unique when present.
- [ ] Every endpoint uses authentication and correct role gates.
- [ ] Non-admin users cannot escape their assigned warehouse scope.
- [ ] Cursor pagination is stable and bounded.
- [ ] Soft deactivation is idempotent and preserves rows.
- [ ] Domain conflicts and not-found errors use stable public codes.
- [ ] `go test ./...`, PostgreSQL-backed tests, `go vet ./...`, `git diff --check`, and `docker compose config --quiet` pass.
- [ ] API and traceability documentation match the delivered code.
