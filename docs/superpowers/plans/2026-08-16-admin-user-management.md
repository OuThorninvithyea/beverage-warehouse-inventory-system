# Admin User Management API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an admin-only Go/Fiber module (`backend/internal/modules/users`) so an administrator can create, list, read, update, deactivate, and reset the password of any BWIMS user account, closing FR-3 and the last part of Gantt task 2.2.

**Architecture:** New module `backend/internal/modules/users` following the existing `Route -> Handler -> Service -> Repository -> PostgreSQL` layering, mirroring `backend/internal/modules/catalog` file-by-file (own local `Cursor`/`Page[T]`/`OptionalString`, no cross-import from `catalog`). Reuses `backend/internal/modules/auth` for role constants, `HashPassword`, `Authenticate`/`RequireRoles` middleware, and JWT claims. All six routes require the `admin` role, including reads. `DeactivateUser` and `ResetPassword` revoke the target's live `refresh_tokens` rows in the same transaction as the write. Last-admin lockout and self-deactivation/self-demotion are guarded — the last-admin check happens inside the same DB transaction as the write to close the race between concurrent requests; the self-target check happens in the service layer since only the service knows the caller's identity from JWT claims.

**Tech Stack:** Go 1.26, Fiber v3 (`fiber.Ctx` by value), `github.com/jackc/pgx/v5` (pgxpool, explicit transactions), PostgreSQL (existing `users`/`roles`/`refresh_tokens`/`warehouses` tables from `000002_initial_schema`, no new migration), `golang.org/x/crypto/bcrypt` via `auth.HashPassword`, `github.com/google/uuid` for ID validation.

No new Linear/Gantt/branch actions are taken automatically by this plan — those are called out as manual follow-ups in Task 18, to be done only after the user confirms.

---

## File Structure

```
backend/internal/modules/users/
  model.go                        # User, *Input types, Cursor, ListFilter, Page[T], Actor
  optional.go                     # OptionalString (local copy, mirrors catalog's tri-state pattern)
  pagination.go                   # EncodeCursor/DecodeCursor/normalizeLimit/userPage
  repository.go                   # Repository interface + PostgresRepository (pgx, transactions)
  service.go                      # Service interface + RBAC/validation/lockout orchestration
  handler.go                      # Fiber handlers, request binding, domain-error -> HTTP mapping
  route.go                        # RegisterRoutes (admin-only group)
  optional_test.go
  pagination_test.go
  model_test.go
  service_test.go                 # fakeRepository-based unit tests
  handler_test.go                 # fiber.New() + RegisterRoutes + fakeHandlerService
  repository_integration_test.go  # real-Postgres lifecycle test, BWIMS_TEST_DATABASE_URL-gated

backend/internal/app/app.go        # MODIFY: wire users module in, before the ROUTE_NOT_FOUND fallback
backend/internal/app/app_test.go   # MODIFY: add TestUsersRoutesAreRegisteredBeforeNotFoundHandler

docs/api-contract.md                          # MODIFY: add "Admin user management endpoints" section
docs/requirements-traceability.md             # MODIFY: flip FR-3 from Planned to Implemented
docs/week-9-user-management-validation.md     # CREATE: validation evidence, mirrors week-8 doc
```

---

## Task 1: Create the feature branch

**Files:** none (git only)

- [ ] **Step 1: Confirm the working tree is clean and branch off `main`**

```bash
git status
git fetch origin
git checkout main
git pull origin main
git checkout -b codex/pro-16-admin-user-management
```

Expected: branch created from an up-to-date `main` (not from `codex/pro-9-catalog-apis`, which is PR #11 and still under separate review).

- [ ] **Step 2: Confirm branch point**

```bash
git log --oneline -3
git merge-base --is-ancestor origin/main HEAD && echo "based on main"
```

Expected: the branch tip is `8f29cbd Codex/pro 8 warehouse location crud (#10)` (current `origin/main` HEAD) with no catalog-module commits.

---

## Task 2: `OptionalString` tri-state type

**Files:**
- Create: `backend/internal/modules/users/optional.go`
- Test: `backend/internal/modules/users/optional_test.go`

- [ ] **Step 1: Write the failing test**

```go
package users

import (
	"encoding/json"
	"testing"
)

func TestOptionalStringUnmarshalJSON(t *testing.T) {
	type wrapper struct {
		Field OptionalString `json:"field"`
	}

	t.Run("omitted field leaves Set false", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if w.Field.Set {
			t.Fatalf("Set = true, want false for omitted field")
		}
		if w.Field.Value != nil {
			t.Fatalf("Value = %v, want nil for omitted field", w.Field.Value)
		}
	})

	t.Run("null clears the field", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":null}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set {
			t.Fatalf("Set = false, want true for null field")
		}
		if w.Field.Value != nil {
			t.Fatalf("Value = %v, want nil for null field", w.Field.Value)
		}
	})

	t.Run("string value is captured", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":"abc-123"}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set {
			t.Fatalf("Set = false, want true for string field")
		}
		if w.Field.Value == nil || *w.Field.Value != "abc-123" {
			t.Fatalf("Value = %v, want \"abc-123\"", w.Field.Value)
		}
	})

	t.Run("non-string value is rejected", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":42}`), &w); err == nil {
			t.Fatalf("Unmarshal() error = nil, want error for non-string field")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/users/... -run TestOptionalStringUnmarshalJSON -v`
Expected: FAIL — `undefined: OptionalString` (package does not compile yet).

- [ ] **Step 3: Write minimal implementation**

```go
package users

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type OptionalString struct {
	Set   bool
	Value *string
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("optional string must be a string or null: %w", err)
	}
	o.Value = &value
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/users/... -run TestOptionalStringUnmarshalJSON -v`
Expected: PASS (all 4 subtests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/optional.go backend/internal/modules/users/optional_test.go
git commit -m "feat: add users module OptionalString tri-state type"
```

---

## Task 3: Domain model types

**Files:**
- Create: `backend/internal/modules/users/model.go`
- Test: `backend/internal/modules/users/model_test.go`

- [ ] **Step 1: Write the failing test**

```go
package users

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSONNeverIncludesPassword(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	user := User{
		ID:          "11111111-1111-1111-1111-111111111111",
		Email:       "manager@bwims.test",
		FullName:    "Sinat Chantha",
		Role:        "warehouse_manager",
		WarehouseID: nil,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if _, exists := decoded["password_hash"]; exists {
		t.Fatalf("encoded user must never include password_hash")
	}
	if _, exists := decoded["password"]; exists {
		t.Fatalf("encoded user must never include password")
	}
	if decoded["warehouse_id"] != nil {
		t.Fatalf("warehouse_id = %v, want null", decoded["warehouse_id"])
	}
	if decoded["role"] != "warehouse_manager" {
		t.Fatalf("role = %v, want warehouse_manager", decoded["role"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/users/... -run TestUserJSONNeverIncludesPassword -v`
Expected: FAIL — `undefined: User`.

- [ ] **Step 3: Write minimal implementation**

```go
package users

import "time"

type Actor struct {
	ID   string
	Role string
}

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	Role        string    `json:"role"`
	WarehouseID *string   `json:"warehouse_id"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserCreateInput struct {
	Email       string
	FullName    string
	Role        string
	WarehouseID OptionalString
	Password    string
}

type UserUpdateInput struct {
	FullName    string
	Role        string
	WarehouseID OptionalString
	IsActive    *bool
}

type PasswordResetInput struct {
	Password string
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type ListFilter struct {
	Limit       int
	After       *Cursor
	Search      string
	Role        string
	WarehouseID *string
	IsActive    *bool
}

type PageInfo struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type Page[T any] struct {
	Items []T      `json:"items"`
	Page  PageInfo `json:"page"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/users/... -run TestUserJSONNeverIncludesPassword -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/model.go backend/internal/modules/users/model_test.go
git commit -m "feat: add users module domain model types"
```

---

## Task 4: Cursor pagination primitives

**Files:**
- Create: `backend/internal/modules/users/pagination.go`
- Test: `backend/internal/modules/users/pagination_test.go`

- [ ] **Step 1: Write the failing test**

```go
package users

import (
	"testing"
	"time"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	id := "22222222-2222-2222-2222-222222222222"

	encoded := EncodeCursor(createdAt, id)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if !decoded.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", decoded.CreatedAt, createdAt)
	}
	if decoded.ID != id {
		t.Fatalf("ID = %v, want %v", decoded.ID, id)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	if _, err := DecodeCursor("not-base64!!"); err == nil {
		t.Fatalf("DecodeCursor() error = nil, want ErrInvalidCursor")
	}
	if _, err := DecodeCursor(""); err == nil {
		t.Fatalf("DecodeCursor(\"\") error = nil, want ErrInvalidCursor")
	}
}

func TestNormalizeLimitBounds(t *testing.T) {
	cases := map[int]int{0: 20, -5: 20, 1: 1, 100: 100, 250: 100}
	for input, want := range cases {
		if got := normalizeLimit(input); got != want {
			t.Fatalf("normalizeLimit(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestUserPageTruncatesAndSetsCursor(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	items := make([]User, 3)
	for i := range items {
		items[i] = User{ID: string(rune('a' + i)), CreatedAt: now.Add(-time.Duration(i) * time.Minute)}
	}

	page := userPage(items, 2)

	if len(page.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(page.Items))
	}
	if !page.Page.HasMore {
		t.Fatalf("HasMore = false, want true")
	}
	if page.Page.NextCursor == nil {
		t.Fatalf("NextCursor = nil, want a cursor")
	}
}

func TestUserPageWithoutOverflowHasNoCursor(t *testing.T) {
	items := []User{{ID: "a"}}
	page := userPage(items, 2)

	if page.Page.HasMore {
		t.Fatalf("HasMore = true, want false")
	}
	if page.Page.NextCursor != nil {
		t.Fatalf("NextCursor = %v, want nil", page.Page.NextCursor)
	}
}

func TestUserPageHandlesNilItems(t *testing.T) {
	page := userPage(nil, 20)
	if page.Items == nil {
		t.Fatalf("Items = nil, want empty slice (JSON must encode [] not null)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/users/... -run 'TestEncodeDecodeCursorRoundTrip|TestDecodeCursorRejectsGarbage|TestNormalizeLimitBounds|TestUserPage' -v`
Expected: FAIL — `undefined: EncodeCursor` etc.

- [ ] **Step 3: Write minimal implementation**

```go
package users

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("users cursor is invalid")

func EncodeCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Cursor{}, ErrInvalidCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{CreatedAt: createdAt.UTC(), ID: parts[1]}, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func userPage(items []User, limit int) Page[User] {
	if items == nil {
		items = []User{}
	}
	page := Page[User]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		cursor := EncodeCursor(last.CreatedAt, last.ID)
		page.Page = PageInfo{NextCursor: &cursor, HasMore: true}
	}
	return page
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/users/... -v`
Expected: PASS (all tests in the package so far).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/pagination.go backend/internal/modules/users/pagination_test.go
git commit -m "feat: add users module cursor pagination primitives"
```

---

## Task 5: Repository foundations + `CreateUser`

**Files:**
- Create: `backend/internal/modules/users/repository.go`
- Create: `backend/internal/modules/users/repository_integration_test.go`

- [ ] **Step 1: Write the failing integration test (create-user portion)**

```go
package users

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func TestPostgresUsersLifecycle(t *testing.T) {
	databaseURL := os.Getenv("BWIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("BWIMS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("database ping error = %v", err)
	}

	repository := NewPostgresRepository(pool)
	suffix := strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))
	emailDomain := "users-test-" + strings.ToLower(suffix) + ".bwims.test"

	var warehouseID string
	err = pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, is_active) VALUES ($1, $2, TRUE) RETURNING id::text`,
		"WH-"+suffix, "Users Test Warehouse "+suffix).Scan(&warehouseID)
	if err != nil {
		t.Fatalf("seed warehouse error = %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE email LIKE $1)", "%"+emailDomain)
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE email LIKE $1", "%"+emailDomain)
		_, _ = pool.Exec(context.Background(), "DELETE FROM warehouses WHERE id = $1", warehouseID)
	})

	passwordHash := "$2a$10$abcdefghijklmnopqrstuvABCDEFGHIJKLMNOPQRSTUVWXYZ012345"

	admin1, err := repository.CreateUser(ctx, UserCreateInput{
		Email: "admin1@" + emailDomain, FullName: "Admin One", Role: auth.RoleAdmin,
	}, passwordHash)
	if err != nil {
		t.Fatalf("CreateUser(admin1) error = %v", err)
	}
	if admin1.ID == "" || admin1.Role != auth.RoleAdmin || admin1.WarehouseID != nil {
		t.Fatalf("admin1 = %#v, want a persisted admin with no warehouse", admin1)
	}

	warehouseIDCopy := warehouseID
	picker, err := repository.CreateUser(ctx, UserCreateInput{
		Email: "picker1@" + emailDomain, FullName: "Picker One", Role: auth.RolePicker,
		WarehouseID: OptionalString{Set: true, Value: &warehouseIDCopy},
	}, passwordHash)
	if err != nil {
		t.Fatalf("CreateUser(picker) error = %v", err)
	}
	if picker.WarehouseID == nil || *picker.WarehouseID != warehouseID {
		t.Fatalf("picker.WarehouseID = %v, want %s", picker.WarehouseID, warehouseID)
	}

	_, err = repository.CreateUser(ctx, UserCreateInput{
		Email: strings.ToUpper("admin1@" + emailDomain), FullName: "Duplicate", Role: auth.RolePicker,
	}, passwordHash)
	if !errors.Is(err, ErrEmailConflict) {
		t.Fatalf("duplicate email (case-insensitive) error = %v, want ErrEmailConflict", err)
	}

	missingWarehouse := "00000000-0000-0000-0000-000000000000"
	_, err = repository.CreateUser(ctx, UserCreateInput{
		Email: "nowhere@" + emailDomain, FullName: "Nowhere", Role: auth.RolePicker,
		WarehouseID: OptionalString{Set: true, Value: &missingWarehouse},
	}, passwordHash)
	if !errors.Is(err, ErrWarehouseNotFound) {
		t.Fatalf("missing warehouse error = %v, want ErrWarehouseNotFound", err)
	}

	_, err = repository.CreateUser(ctx, UserCreateInput{
		Email: "badrole@" + emailDomain, FullName: "Bad Role", Role: "supervisor",
	}, passwordHash)
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("invalid role error = %v, want ErrInvalidRole", err)
	}

	_ = admin1
	_ = picker
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`

(If `BWIMS_TEST_DATABASE_URL` is not set locally, first bring up Postgres the same way the catalog module's validation doc did — check `docs/week-8-catalog-validation.md` "Database migration status" section for the exact `docker compose`/connection-string steps used previously — then export the variable before running.)

Expected: FAIL — compile error, `undefined: NewPostgresRepository` / `ErrEmailConflict` / `ErrWarehouseNotFound` / `ErrInvalidRole`.

- [ ] **Step 3: Write minimal implementation**

```go
package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound      = errors.New("user was not found")
	ErrWarehouseNotFound = errors.New("warehouse was not found or is not active")
	ErrEmailConflict     = errors.New("email is already in use")
	ErrInvalidRole       = errors.New("role is not a known role code")
)

type Repository interface {
	CreateUser(ctx context.Context, input UserCreateInput, passwordHash string) (User, error)
	ListUsers(ctx context.Context, filter ListFilter) ([]User, error)
	GetUser(ctx context.Context, id string) (User, error)
	UpdateUser(ctx context.Context, id string, input UserUpdateInput) (User, error)
	DeactivateUser(ctx context.Context, id string) error
	ResetPassword(ctx context.Context, id string, passwordHash string) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.Email, &user.FullName, &user.Role,
		&user.WarehouseID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalStringValue(value OptionalString) any {
	if !value.Set || value.Value == nil {
		return nil
	}
	return *value.Value
}

func cursorArguments(cursor *Cursor) (any, any) {
	if cursor == nil {
		return nil, nil
	}
	return cursor.CreatedAt, cursor.ID
}

func resolveRoleID(ctx context.Context, tx pgx.Tx, code string) (string, error) {
	var roleID string
	err := tx.QueryRow(ctx, `SELECT id::text FROM roles WHERE code = $1`, code).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidRole
	}
	if err != nil {
		return "", fmt.Errorf("resolve role id: %w", err)
	}
	return roleID, nil
}

func lockActiveWarehouse(ctx context.Context, tx pgx.Tx, id string) error {
	var lockedID string
	err := tx.QueryRow(ctx,
		`SELECT id::text FROM warehouses WHERE id = $1 AND is_active = TRUE FOR UPDATE`, id,
	).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWarehouseNotFound
	}
	if err != nil {
		return fmt.Errorf("lock active warehouse: %w", err)
	}
	return nil
}

func mapUsersWriteError(operation string, err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		if pgError.ConstraintName == "users_email_unique_idx" {
			return ErrEmailConflict
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func (repository *PostgresRepository) CreateUser(
	ctx context.Context,
	input UserCreateInput,
	passwordHash string,
) (User, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin create user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	roleID, err := resolveRoleID(ctx, tx, input.Role)
	if err != nil {
		return User{}, err
	}

	if input.WarehouseID.Set && input.WarehouseID.Value != nil {
		if err := lockActiveWarehouse(ctx, tx, *input.WarehouseID.Value); err != nil {
			return User{}, err
		}
	}

	user, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (role_id, warehouse_id, email, password_hash, full_name, is_active)
		VALUES ($1, $2, LOWER($3), $4, $5, TRUE)
		RETURNING id::text, email, full_name, $6::text, warehouse_id::text, is_active, created_at, updated_at`,
		roleID, optionalStringValue(input.WarehouseID), input.Email, passwordHash, input.FullName, input.Role))
	if err != nil {
		return User{}, mapUsersWriteError("create user", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit create user: %w", err)
	}
	return user, nil
}
```

The rest of the `Repository` interface methods (`ListUsers`, `GetUser`, `UpdateUser`, `DeactivateUser`, `ResetPassword`) are added in Tasks 6-9; the package will not compile until then, so for this step only run the single test above with `-run TestPostgresUsersLifecycle`, and expect it to fail to compile if you try to build the whole package — that's expected until Task 9 lands. To keep the package compiling at every commit, add stub methods now that panic, and replace them task-by-task:

```go
func (repository *PostgresRepository) ListUsers(ctx context.Context, filter ListFilter) ([]User, error) {
	panic("not implemented until Task 6")
}

func (repository *PostgresRepository) GetUser(ctx context.Context, id string) (User, error) {
	panic("not implemented until Task 7")
}

func (repository *PostgresRepository) UpdateUser(ctx context.Context, id string, input UserUpdateInput) (User, error) {
	panic("not implemented until Task 7")
}

func (repository *PostgresRepository) DeactivateUser(ctx context.Context, id string) error {
	panic("not implemented until Task 8")
}

func (repository *PostgresRepository) ResetPassword(ctx context.Context, id string, passwordHash string) error {
	panic("not implemented until Task 9")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/repository.go backend/internal/modules/users/repository_integration_test.go
git commit -m "feat: add users repository CreateUser with role and warehouse validation"
```

---

## Task 6: Repository — `ListUsers`

**Files:**
- Modify: `backend/internal/modules/users/repository.go` (replace the `ListUsers` stub)
- Modify: `backend/internal/modules/users/repository_integration_test.go` (extend `TestPostgresUsersLifecycle`)

- [ ] **Step 1: Extend the failing test**

Append to `TestPostgresUsersLifecycle`, right after the four `CreateUser` error-case checks and before the final `_ = admin1 / _ = picker` lines (remove those two placeholder lines now):

```go
	listed, err := repository.ListUsers(ctx, ListFilter{Limit: 10, Search: strings.ToLower(suffix)})
	if err != nil || len(listed) != 2 {
		t.Fatalf("ListUsers() = %#v, %v, want 2 users (admin1, picker)", listed, err)
	}

	adminOnly, err := repository.ListUsers(ctx, ListFilter{Limit: 10, Role: auth.RoleAdmin, Search: strings.ToLower(suffix)})
	if err != nil || len(adminOnly) != 1 || adminOnly[0].ID != admin1.ID {
		t.Fatalf("ListUsers(role=admin) = %#v, %v, want [admin1]", adminOnly, err)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 6`.

- [ ] **Step 3: Replace the `ListUsers` stub with the real implementation**

```go
func (repository *PostgresRepository) ListUsers(ctx context.Context, filter ListFilter) ([]User, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := repository.pool.Query(ctx, `
		SELECT u.id::text, u.email, u.full_name, r.code, u.warehouse_id::text, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE ($1 = '' OR u.email ILIKE '%' || $1 || '%' OR u.full_name ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR r.code = $2)
		  AND ($3::uuid IS NULL OR u.warehouse_id = $3::uuid)
		  AND ($4::boolean IS NULL OR u.is_active = $4)
		  AND (
			$5::timestamptz IS NULL
			OR (u.created_at, u.id) < ($5::timestamptz, $6::uuid)
		  )
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $7`,
		filter.Search, filter.Role, nullableString(filter.WarehouseID), nullableBool(filter.IsActive),
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/repository.go backend/internal/modules/users/repository_integration_test.go
git commit -m "feat: add users repository ListUsers with search, role, and warehouse filters"
```

---

## Task 7: Repository — `GetUser` and `UpdateUser`

**Files:**
- Modify: `backend/internal/modules/users/repository.go` (replace `GetUser` and `UpdateUser` stubs; add `requireMoreThanOneActiveAdmin` helper)
- Modify: `backend/internal/modules/users/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the `ListUsers` assertions from Task 6:

```go
	fetched, err := repository.GetUser(ctx, admin1.ID)
	if err != nil || fetched.Email != admin1.Email {
		t.Fatalf("GetUser() = %#v, %v, want admin1", fetched, err)
	}

	if _, err := repository.GetUser(ctx, missingWarehouse); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("GetUser(missing) error = %v, want ErrUserNotFound", err)
	}

	updated, err := repository.UpdateUser(ctx, picker.ID, UserUpdateInput{
		FullName: "Picker One Updated", Role: auth.RoleWarehouseManager, WarehouseID: OptionalString{Set: true},
	})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if updated.WarehouseID != nil || updated.Role != auth.RoleWarehouseManager || updated.FullName != "Picker One Updated" {
		t.Fatalf("updated = %#v, want cleared warehouse, warehouse_manager role, new name", updated)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 7`.

- [ ] **Step 3: Replace the `GetUser` and `UpdateUser` stubs, add the shared lockout helper**

```go
func (repository *PostgresRepository) GetUser(ctx context.Context, id string) (User, error) {
	user, err := scanUser(repository.pool.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.full_name, r.code, u.warehouse_id::text, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// requireMoreThanOneActiveAdmin locks every currently-active admin row and fails
// with ErrLastAdminProtected if fewer than two exist. It must be called inside the
// same transaction as the write that would reduce the admin count, so a concurrent
// deactivation of a different admin cannot race past this check.
func requireMoreThanOneActiveAdmin(ctx context.Context, tx pgx.Tx) error {
	rows, err := tx.Query(ctx, `
		SELECT u.id::text
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.code = 'admin' AND u.is_active = TRUE
		FOR UPDATE OF u`)
	if err != nil {
		return fmt.Errorf("lock active admins: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("count active admins: %w", err)
	}
	if count <= 1 {
		return ErrLastAdminProtected
	}
	return nil
}

func (repository *PostgresRepository) UpdateUser(
	ctx context.Context,
	id string,
	input UserUpdateInput,
) (User, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin update user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentRoleCode string
	var currentIsActive bool
	err = tx.QueryRow(ctx, `
		SELECT r.code, u.is_active
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1
		FOR UPDATE OF u`, id,
	).Scan(&currentRoleCode, &currentIsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("lock user for update: %w", err)
	}

	roleID, err := resolveRoleID(ctx, tx, input.Role)
	if err != nil {
		return User{}, err
	}

	demotingActiveAdmin := currentRoleCode == "admin" && currentIsActive && input.Role != "admin"
	deactivatingActiveAdmin := currentRoleCode == "admin" && currentIsActive &&
		input.IsActive != nil && !*input.IsActive
	if demotingActiveAdmin || deactivatingActiveAdmin {
		if err := requireMoreThanOneActiveAdmin(ctx, tx); err != nil {
			return User{}, err
		}
	}

	if input.WarehouseID.Set && input.WarehouseID.Value != nil {
		if err := lockActiveWarehouse(ctx, tx, *input.WarehouseID.Value); err != nil {
			return User{}, err
		}
	}

	user, err := scanUser(tx.QueryRow(ctx, `
		UPDATE users
		SET full_name = $2,
		    role_id = $3,
		    warehouse_id = CASE WHEN $4 THEN $5::uuid ELSE warehouse_id END,
		    is_active = COALESCE($6, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, email, full_name, $7::text, warehouse_id::text, is_active, created_at, updated_at`,
		id, input.FullName, roleID,
		input.WarehouseID.Set, optionalStringValue(input.WarehouseID),
		nullableBool(input.IsActive), input.Role))
	if err != nil {
		return User{}, mapUsersWriteError("update user", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit update user: %w", err)
	}
	return user, nil
}
```

Add the new sentinel error next to the others near the top of `repository.go`:

```go
var ErrLastAdminProtected = errors.New("this action would leave zero active administrators")
```

(Move this into the existing `var ( ... )` block from Task 5 rather than a second block.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/repository.go backend/internal/modules/users/repository_integration_test.go
git commit -m "feat: add users repository GetUser and UpdateUser with in-transaction admin lockout guard"
```

---

## Task 8: Repository — `DeactivateUser` (+ refresh-token revocation)

**Files:**
- Modify: `backend/internal/modules/users/repository.go` (replace `DeactivateUser` stub)
- Modify: `backend/internal/modules/users/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the `UpdateUser` assertions from Task 7:

```go
	_, err = pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, NOW() + interval '1 hour')`, admin1.ID, "test-hash-admin1-"+suffix)
	if err != nil {
		t.Fatalf("seed refresh token for admin1 error = %v", err)
	}

	admin2, err := repository.CreateUser(ctx, UserCreateInput{
		Email: "admin2@" + emailDomain, FullName: "Admin Two", Role: auth.RoleAdmin,
	}, passwordHash)
	if err != nil {
		t.Fatalf("CreateUser(admin2) error = %v", err)
	}

	if err := repository.DeactivateUser(ctx, admin1.ID); err != nil {
		t.Fatalf("DeactivateUser(admin1) error = %v, want success (admin2 keeps one admin active)", err)
	}

	var admin1TokenRevoked *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM refresh_tokens WHERE user_id = $1`, admin1.ID).Scan(&admin1TokenRevoked); err != nil {
		t.Fatalf("read admin1 refresh token error = %v", err)
	}
	if admin1TokenRevoked == nil {
		t.Fatalf("admin1 refresh token was not revoked on deactivation")
	}

	if err := repository.DeactivateUser(ctx, admin2.ID); !errors.Is(err, ErrLastAdminProtected) {
		t.Fatalf("deactivate sole remaining admin error = %v, want ErrLastAdminProtected", err)
	}

	if err := repository.DeactivateUser(ctx, missingWarehouse); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("deactivate missing user error = %v, want ErrUserNotFound", err)
	}
```

At this point in the lifecycle: `admin1` was deactivated while `admin2` was still active (two active admins, so it succeeds and its refresh token is revoked); then deactivating `admin2` — now the sole remaining active admin — is rejected with `ErrLastAdminProtected`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 8`.

- [ ] **Step 3: Replace the `DeactivateUser` stub**

```go
func (repository *PostgresRepository) DeactivateUser(ctx context.Context, id string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deactivate user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var roleCode string
	var isActive bool
	err = tx.QueryRow(ctx, `
		SELECT r.code, u.is_active
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1
		FOR UPDATE OF u`, id,
	).Scan(&roleCode, &isActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("lock user for deactivation: %w", err)
	}

	if roleCode == "admin" && isActive {
		if err := requireMoreThanOneActiveAdmin(ctx, tx); err != nil {
			return err
		}
	}

	tag, err := tx.Exec(ctx, `UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return fmt.Errorf("revoke refresh tokens on deactivate: %w", err)
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/repository.go backend/internal/modules/users/repository_integration_test.go
git commit -m "feat: add users repository DeactivateUser with lockout guard and refresh-token revocation"
```

---

## Task 9: Repository — `ResetPassword`

**Files:**
- Modify: `backend/internal/modules/users/repository.go` (replace `ResetPassword` stub)
- Modify: `backend/internal/modules/users/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the Task 8 assertions, right before the closing `}` of `TestPostgresUsersLifecycle`:

```go
	_, err = pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, NOW() + interval '1 hour')`, picker.ID, "test-hash-picker-"+suffix)
	if err != nil {
		t.Fatalf("seed refresh token for picker error = %v", err)
	}

	newHash := "$2a$10$newhashnewhashnewhashnewhashnewhashnewhashnewhashnewh"
	if err := repository.ResetPassword(ctx, picker.ID, newHash); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}

	var storedHash string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, picker.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read password hash error = %v", err)
	}
	if storedHash != newHash {
		t.Fatalf("storedHash = %q, want %q", storedHash, newHash)
	}

	var pickerTokenRevoked *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM refresh_tokens WHERE user_id = $1`, picker.ID).Scan(&pickerTokenRevoked); err != nil {
		t.Fatalf("read picker refresh token error = %v", err)
	}
	if pickerTokenRevoked == nil {
		t.Fatalf("picker refresh token was not revoked on password reset")
	}

	if err := repository.ResetPassword(ctx, missingWarehouse, newHash); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("ResetPassword(missing) error = %v, want ErrUserNotFound", err)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 9`.

- [ ] **Step 3: Replace the `ResetPassword` stub**

```go
func (repository *PostgresRepository) ResetPassword(ctx context.Context, id string, passwordHash string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reset password: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, id, passwordHash)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`, id); err != nil {
		return fmt.Errorf("revoke refresh tokens on password reset: %w", err)
	}

	return tx.Commit(ctx)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/users/... -v`
Expected: PASS for the whole `users` package, including every subtest of `TestPostgresUsersLifecycle`.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/repository.go backend/internal/modules/users/repository_integration_test.go
git commit -m "feat: add users repository ResetPassword with refresh-token revocation"
```

---

## Task 10: Service layer — RBAC, validation, `CreateUser`/`ListUsers`/`GetUser`

**Files:**
- Create: `backend/internal/modules/users/service.go`
- Create: `backend/internal/modules/users/service_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package users

import (
	"context"
	"errors"
	"testing"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

type fakeRepository struct {
	createFn     func(UserCreateInput, string) (User, error)
	listFn       func(ListFilter) ([]User, error)
	getFn        func(string) (User, error)
	updateFn     func(string, UserUpdateInput) (User, error)
	deactivateFn func(string) error
	resetFn      func(string, string) error
}

func (f *fakeRepository) CreateUser(_ context.Context, input UserCreateInput, hash string) (User, error) {
	return f.createFn(input, hash)
}
func (f *fakeRepository) ListUsers(_ context.Context, filter ListFilter) ([]User, error) {
	return f.listFn(filter)
}
func (f *fakeRepository) GetUser(_ context.Context, id string) (User, error) {
	return f.getFn(id)
}
func (f *fakeRepository) UpdateUser(_ context.Context, id string, input UserUpdateInput) (User, error) {
	return f.updateFn(id, input)
}
func (f *fakeRepository) DeactivateUser(_ context.Context, id string) error {
	return f.deactivateFn(id)
}
func (f *fakeRepository) ResetPassword(_ context.Context, id string, hash string) error {
	return f.resetFn(id, hash)
}

func adminActor() Actor { return Actor{ID: "admin-1", Role: auth.RoleAdmin} }

func TestCreateUserRejectsNonAdmin(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RoleWarehouseManager, auth.RolePicker, auth.RoleViewer} {
		_, err := service.CreateUser(context.Background(), Actor{ID: "x", Role: role}, UserCreateInput{
			Email: "a@bwims.test", FullName: "A", Role: auth.RolePicker, Password: "at-least-12-chars",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestCreateUserValidatesRequiredFields(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "", FullName: "A", Role: auth.RolePicker, Password: "at-least-12-chars",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestCreateUserRejectsUnknownRole(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "a@bwims.test", FullName: "A", Role: "supervisor", Password: "at-least-12-chars",
	})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("err = %v, want ErrInvalidRole", err)
	}
}

func TestCreateUserRejectsShortPassword(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "a@bwims.test", FullName: "A", Role: auth.RolePicker, Password: "short",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestCreateUserPassesHashedPasswordToRepository(t *testing.T) {
	var capturedHash string
	repo := &fakeRepository{
		createFn: func(input UserCreateInput, hash string) (User, error) {
			capturedHash = hash
			return User{ID: "new-1", Email: input.Email, Role: input.Role}, nil
		},
	}
	service := NewService(repo)
	user, err := service.CreateUser(context.Background(), adminActor(), UserCreateInput{
		Email: "A@BWIMS.test", FullName: "A", Role: auth.RolePicker, Password: "at-least-12-chars",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if capturedHash == "" || capturedHash == "at-least-12-chars" {
		t.Fatalf("capturedHash = %q, want a bcrypt hash, not the raw password", capturedHash)
	}
	if user.Email != "a@bwims.test" {
		t.Fatalf("Email = %q, want lower-cased email", user.Email)
	}
}

func TestGetUserRejectsInvalidID(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.GetUser(context.Background(), adminActor(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestListUsersRejectsUnknownRoleFilter(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListUsers(context.Background(), adminActor(), ListFilter{Role: "supervisor"})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("err = %v, want ErrInvalidRole", err)
	}
}

func TestListUsersAppliesDefaultAndOverflowLimit(t *testing.T) {
	var capturedLimit int
	repo := &fakeRepository{
		listFn: func(filter ListFilter) ([]User, error) {
			capturedLimit = filter.Limit
			return []User{}, nil
		},
	}
	service := NewService(repo)
	if _, err := service.ListUsers(context.Background(), adminActor(), ListFilter{}); err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if capturedLimit != 21 {
		t.Fatalf("capturedLimit = %d, want 21 (default 20 + 1 overflow probe)", capturedLimit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/users/... -run 'TestCreateUser|TestGetUser|TestListUsers' -v`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 3: Write minimal implementation**

```go
package users

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden                 = errors.New("user management access is forbidden")
	ErrValidation                = errors.New("user data is invalid")
	ErrInvalidID                 = errors.New("resource id is invalid")
	ErrSelfDeactivationForbidden = errors.New("an administrator cannot deactivate or demote themselves")
)

type Service interface {
	CreateUser(ctx context.Context, actor Actor, input UserCreateInput) (User, error)
	ListUsers(ctx context.Context, actor Actor, filter ListFilter) (Page[User], error)
	GetUser(ctx context.Context, actor Actor, id string) (User, error)
	UpdateUser(ctx context.Context, actor Actor, id string, input UserUpdateInput) (User, error)
	DeactivateUser(ctx context.Context, actor Actor, id string) error
	ResetPassword(ctx context.Context, actor Actor, id string, input PasswordResetInput) error
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

var knownRoles = map[string]struct{}{
	auth.RoleAdmin:            {},
	auth.RoleWarehouseManager: {},
	auth.RolePicker:           {},
	auth.RoleViewer:           {},
}

func isKnownRole(role string) bool {
	_, ok := knownRoles[role]
	return ok
}

func canManageUsers(role string) bool {
	return role == auth.RoleAdmin
}

func (service *service) CreateUser(ctx context.Context, actor Actor, input UserCreateInput) (User, error) {
	if !canManageUsers(actor.Role) {
		return User{}, ErrForbidden
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.FullName = strings.TrimSpace(input.FullName)
	if input.Email == "" || input.FullName == "" {
		return User{}, ErrValidation
	}
	if !isKnownRole(input.Role) {
		return User{}, ErrInvalidRole
	}
	if input.WarehouseID.Set && input.WarehouseID.Value != nil && !validUUID(*input.WarehouseID.Value) {
		return User{}, ErrInvalidID
	}
	if len(input.Password) < 12 {
		return User{}, ErrValidation
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return User{}, ErrValidation
	}

	return service.repository.CreateUser(ctx, input, passwordHash)
}

func (service *service) ListUsers(ctx context.Context, actor Actor, filter ListFilter) (Page[User], error) {
	if !canManageUsers(actor.Role) {
		return Page[User]{}, ErrForbidden
	}
	if filter.Role != "" && !isKnownRole(filter.Role) {
		return Page[User]{}, ErrInvalidRole
	}
	if filter.WarehouseID != nil {
		warehouseID := strings.TrimSpace(*filter.WarehouseID)
		if !validUUID(warehouseID) {
			return Page[User]{}, ErrInvalidID
		}
		filter.WarehouseID = &warehouseID
	}
	requestedLimit := normalizeLimit(filter.Limit)
	filter.Limit = requestedLimit + 1
	filter.Search = strings.TrimSpace(filter.Search)

	items, err := service.repository.ListUsers(ctx, filter)
	if err != nil {
		return Page[User]{}, err
	}
	return userPage(items, requestedLimit), nil
}

func (service *service) GetUser(ctx context.Context, actor Actor, id string) (User, error) {
	if !canManageUsers(actor.Role) {
		return User{}, ErrForbidden
	}
	if !validUUID(id) {
		return User{}, ErrInvalidID
	}
	return service.repository.GetUser(ctx, id)
}

func (service *service) UpdateUser(ctx context.Context, actor Actor, id string, input UserUpdateInput) (User, error) {
	panic("not implemented until Task 11")
}

func (service *service) DeactivateUser(ctx context.Context, actor Actor, id string) error {
	panic("not implemented until Task 11")
}

func (service *service) ResetPassword(ctx context.Context, actor Actor, id string, input PasswordResetInput) error {
	panic("not implemented until Task 11")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/users/... -run 'TestCreateUser|TestGetUser|TestListUsers' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/service.go backend/internal/modules/users/service_test.go
git commit -m "feat: add users service RBAC and validation for create, list, get"
```

---

## Task 11: Service layer — `UpdateUser`/`DeactivateUser`/`ResetPassword` with self-protection

**Files:**
- Modify: `backend/internal/modules/users/service.go` (replace the three stubs)
- Modify: `backend/internal/modules/users/service_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `service_test.go`:

```go
func TestUpdateUserRejectsSelfDeactivation(t *testing.T) {
	service := NewService(&fakeRepository{})
	actor := Actor{ID: "admin-1", Role: auth.RoleAdmin}
	inactive := false
	_, err := service.UpdateUser(context.Background(), actor, actor.ID, UserUpdateInput{
		FullName: "Admin", Role: auth.RoleAdmin, IsActive: &inactive,
	})
	if !errors.Is(err, ErrSelfDeactivationForbidden) {
		t.Fatalf("err = %v, want ErrSelfDeactivationForbidden", err)
	}
}

func TestUpdateUserRejectsSelfDemotion(t *testing.T) {
	service := NewService(&fakeRepository{})
	actor := Actor{ID: "admin-1", Role: auth.RoleAdmin}
	active := true
	_, err := service.UpdateUser(context.Background(), actor, actor.ID, UserUpdateInput{
		FullName: "Admin", Role: auth.RoleWarehouseManager, IsActive: &active,
	})
	if !errors.Is(err, ErrSelfDeactivationForbidden) {
		t.Fatalf("err = %v, want ErrSelfDeactivationForbidden", err)
	}
}

func TestUpdateUserAllowsSelfUpdateThatDoesNotDemoteOrDeactivate(t *testing.T) {
	repo := &fakeRepository{
		updateFn: func(id string, input UserUpdateInput) (User, error) {
			return User{ID: id, FullName: input.FullName, Role: input.Role}, nil
		},
	}
	service := NewService(repo)
	actor := Actor{ID: "admin-1", Role: auth.RoleAdmin}
	active := true
	_, err := service.UpdateUser(context.Background(), actor, actor.ID, UserUpdateInput{
		FullName: "New Name", Role: auth.RoleAdmin, IsActive: &active,
	})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v, want nil for non-demoting self update", err)
	}
}

func TestUpdateUserRejectsUnknownRole(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.UpdateUser(context.Background(), adminActor(), "22222222-2222-2222-2222-222222222222", UserUpdateInput{
		FullName: "A", Role: "supervisor",
	})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("err = %v, want ErrInvalidRole", err)
	}
}

func TestUpdateUserPropagatesLastAdminProtection(t *testing.T) {
	repo := &fakeRepository{
		updateFn: func(string, UserUpdateInput) (User, error) { return User{}, ErrLastAdminProtected },
	}
	service := NewService(repo)
	_, err := service.UpdateUser(context.Background(), adminActor(), "22222222-2222-2222-2222-222222222222", UserUpdateInput{
		FullName: "A", Role: auth.RolePicker,
	})
	if !errors.Is(err, ErrLastAdminProtected) {
		t.Fatalf("err = %v, want ErrLastAdminProtected", err)
	}
}

func TestDeactivateUserRejectsSelfDeactivation(t *testing.T) {
	service := NewService(&fakeRepository{})
	actor := Actor{ID: "admin-1", Role: auth.RoleAdmin}
	err := service.DeactivateUser(context.Background(), actor, actor.ID)
	if !errors.Is(err, ErrSelfDeactivationForbidden) {
		t.Fatalf("err = %v, want ErrSelfDeactivationForbidden", err)
	}
}

func TestDeactivateUserPropagatesLastAdminProtection(t *testing.T) {
	repo := &fakeRepository{
		deactivateFn: func(string) error { return ErrLastAdminProtected },
	}
	service := NewService(repo)
	err := service.DeactivateUser(context.Background(), adminActor(), "22222222-2222-2222-2222-222222222222")
	if !errors.Is(err, ErrLastAdminProtected) {
		t.Fatalf("err = %v, want ErrLastAdminProtected", err)
	}
}

func TestDeactivateUserRejectsNonAdmin(t *testing.T) {
	service := NewService(&fakeRepository{})
	err := service.DeactivateUser(context.Background(), Actor{ID: "x", Role: auth.RolePicker}, "22222222-2222-2222-2222-222222222222")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestResetPasswordRejectsShortPassword(t *testing.T) {
	service := NewService(&fakeRepository{})
	err := service.ResetPassword(context.Background(), adminActor(), "22222222-2222-2222-2222-222222222222", PasswordResetInput{Password: "short"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestResetPasswordHashesBeforeCallingRepository(t *testing.T) {
	var capturedHash string
	repo := &fakeRepository{
		resetFn: func(id string, hash string) error {
			capturedHash = hash
			return nil
		},
	}
	service := NewService(repo)
	err := service.ResetPassword(context.Background(), adminActor(), "22222222-2222-2222-2222-222222222222", PasswordResetInput{Password: "at-least-12-chars"})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if capturedHash == "" || capturedHash == "at-least-12-chars" {
		t.Fatalf("capturedHash = %q, want a bcrypt hash", capturedHash)
	}
}

func TestResetPasswordAllowsAdminToResetTheirOwnPassword(t *testing.T) {
	repo := &fakeRepository{resetFn: func(string, string) error { return nil }}
	service := NewService(repo)
	actor := Actor{ID: "admin-1", Role: auth.RoleAdmin}
	err := service.ResetPassword(context.Background(), actor, actor.ID, PasswordResetInput{Password: "at-least-12-chars"})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v, want nil (self password reset stays allowed)", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/users/... -run 'TestUpdateUser|TestDeactivateUser|TestResetPassword' -v`
Expected: FAIL — `panic: not implemented until Task 11`.

- [ ] **Step 3: Replace the three stubs**

```go
func (service *service) UpdateUser(ctx context.Context, actor Actor, id string, input UserUpdateInput) (User, error) {
	if !canManageUsers(actor.Role) {
		return User{}, ErrForbidden
	}
	if !validUUID(id) {
		return User{}, ErrInvalidID
	}
	input.FullName = strings.TrimSpace(input.FullName)
	if input.FullName == "" {
		return User{}, ErrValidation
	}
	if !isKnownRole(input.Role) {
		return User{}, ErrInvalidRole
	}
	if input.WarehouseID.Set && input.WarehouseID.Value != nil && !validUUID(*input.WarehouseID.Value) {
		return User{}, ErrInvalidID
	}

	selfTargeted := actor.ID == id
	demoting := input.Role != auth.RoleAdmin
	deactivating := input.IsActive != nil && !*input.IsActive
	if selfTargeted && (demoting || deactivating) {
		return User{}, ErrSelfDeactivationForbidden
	}

	return service.repository.UpdateUser(ctx, id, input)
}

func (service *service) DeactivateUser(ctx context.Context, actor Actor, id string) error {
	if !canManageUsers(actor.Role) {
		return ErrForbidden
	}
	if !validUUID(id) {
		return ErrInvalidID
	}
	if actor.ID == id {
		return ErrSelfDeactivationForbidden
	}
	return service.repository.DeactivateUser(ctx, id)
}

func (service *service) ResetPassword(ctx context.Context, actor Actor, id string, input PasswordResetInput) error {
	if !canManageUsers(actor.Role) {
		return ErrForbidden
	}
	if !validUUID(id) {
		return ErrInvalidID
	}
	if len(input.Password) < 12 {
		return ErrValidation
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return ErrValidation
	}
	return service.repository.ResetPassword(ctx, id, passwordHash)
}
```

Note: `ErrLastAdminProtected` is defined in `repository.go` (Task 7); `service.go` re-uses it directly (same package, no import needed) so `errors.Is` checks against it work whether the error originates from the repository or is passed straight through.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/users/... -v`
Expected: PASS for the entire package (unit tests; the Postgres integration test still requires `BWIMS_TEST_DATABASE_URL` or will skip).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/service.go backend/internal/modules/users/service_test.go
git commit -m "feat: add users service update, deactivate, and password reset with self-protection guards"
```

---

## Task 12: Handler, request binding, and error mapping

**Files:**
- Create: `backend/internal/modules/users/handler.go`

- [ ] **Step 1-4 combined (no isolated handler unit tests — handler behavior is covered end-to-end in Task 13's `handler_test.go`, matching the catalog module's convention where `handler.go` has no dedicated `handler_test.go`-free step)**

Write `handler.go` directly; Task 13 supplies the tests that exercise it. This is consistent with how `catalog/handler.go` and `catalog/handler_test.go` are structured as a pair (the test file drives real HTTP requests through the real routes, not isolated function calls).

```go
package users

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func usersActor(c fiber.Ctx) Actor {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok || claims == nil {
		return Actor{}
	}
	return Actor{ID: claims.Subject, Role: claims.Role}
}

func invalidUsersRequest(message string, err error) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", message, err)
}

func usersHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidCursor):
		return invalidUsersRequest("a resource identifier, filter, or cursor is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN", "your role does not allow this action", err)
	case errors.Is(err, ErrUserNotFound):
		return httpx.NewError(fiber.StatusNotFound, "USER_NOT_FOUND", "user was not found", err)
	case errors.Is(err, ErrWarehouseNotFound):
		return httpx.NewError(fiber.StatusNotFound, "WAREHOUSE_NOT_FOUND", "warehouse was not found or is not active", err)
	case errors.Is(err, ErrEmailConflict):
		return httpx.NewError(fiber.StatusConflict, "EMAIL_CONFLICT", "email is already in use", err)
	case errors.Is(err, ErrLastAdminProtected):
		return httpx.NewError(fiber.StatusConflict, "LAST_ADMIN_PROTECTED", "this action would leave zero active administrators", err)
	case errors.Is(err, ErrSelfDeactivationForbidden):
		return httpx.NewError(fiber.StatusConflict, "SELF_DEACTIVATION_FORBIDDEN", "an administrator cannot deactivate or demote themselves", err)
	case errors.Is(err, ErrInvalidRole):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "INVALID_ROLE", "role is not a known role code", err)
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "user data is invalid", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "USER_OPERATION_FAILED", "user operation could not be completed", err)
	}
}

type createUserRequest struct {
	Email       string         `json:"email"`
	FullName    string         `json:"full_name"`
	Role        string         `json:"role"`
	WarehouseID OptionalString `json:"warehouse_id"`
	Password    string         `json:"password"`
}

type updateUserRequest struct {
	FullName    string         `json:"full_name"`
	Role        string         `json:"role"`
	WarehouseID OptionalString `json:"warehouse_id"`
	IsActive    *bool          `json:"is_active"`
}

type passwordResetRequest struct {
	Password string `json:"password"`
}

func bindCreateUserRequest(c fiber.Ctx) (UserCreateInput, error) {
	var body createUserRequest
	if err := c.Bind().Body(&body); err != nil {
		return UserCreateInput{}, invalidUsersRequest("request body could not be parsed", err)
	}
	return UserCreateInput{
		Email: body.Email, FullName: body.FullName, Role: body.Role,
		WarehouseID: body.WarehouseID, Password: body.Password,
	}, nil
}

func bindUpdateUserRequest(c fiber.Ctx) (UserUpdateInput, error) {
	var body updateUserRequest
	if err := c.Bind().Body(&body); err != nil {
		return UserUpdateInput{}, invalidUsersRequest("request body could not be parsed", err)
	}
	return UserUpdateInput{
		FullName: body.FullName, Role: body.Role, WarehouseID: body.WarehouseID, IsActive: body.IsActive,
	}, nil
}

func bindPasswordResetRequest(c fiber.Ctx) (PasswordResetInput, error) {
	var body passwordResetRequest
	if err := c.Bind().Body(&body); err != nil {
		return PasswordResetInput{}, invalidUsersRequest("request body could not be parsed", err)
	}
	return PasswordResetInput{Password: body.Password}, nil
}

func usersOptionalBool(raw string) (*bool, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseUsersListFilter(c fiber.Ctx) (ListFilter, error) {
	filter := ListFilter{Search: c.Query("search"), Role: c.Query("role")}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return ListFilter{}, invalidUsersRequest("limit must be an integer from 1 to 100", nil)
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return ListFilter{}, invalidUsersRequest("after cursor is invalid", err)
		}
		filter.After = &cursor
	}
	if raw := c.Query("warehouse_id"); raw != "" {
		filter.WarehouseID = &raw
	}
	active, err := usersOptionalBool(c.Query("is_active"))
	if err != nil {
		return ListFilter{}, invalidUsersRequest("is_active must be true or false", err)
	}
	filter.IsActive = active
	return filter, nil
}

func (h *Handler) ListUsers(c fiber.Ctx) error {
	filter, err := parseUsersListFilter(c)
	if err != nil {
		return err
	}
	page, err := h.service.ListUsers(c.Context(), usersActor(c), filter)
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) CreateUser(c fiber.Ctx) error {
	input, err := bindCreateUserRequest(c)
	if err != nil {
		return err
	}
	user, err := h.service.CreateUser(c.Context(), usersActor(c), input)
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(user))
}

func (h *Handler) GetUser(c fiber.Ctx) error {
	user, err := h.service.GetUser(c.Context(), usersActor(c), c.Params("user_id"))
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(user))
}

func (h *Handler) UpdateUser(c fiber.Ctx) error {
	input, err := bindUpdateUserRequest(c)
	if err != nil {
		return err
	}
	user, err := h.service.UpdateUser(c.Context(), usersActor(c), c.Params("user_id"), input)
	if err != nil {
		return usersHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(user))
}

func (h *Handler) DeactivateUser(c fiber.Ctx) error {
	if err := h.service.DeactivateUser(c.Context(), usersActor(c), c.Params("user_id")); err != nil {
		return usersHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ResetPassword(c fiber.Ctx) error {
	input, err := bindPasswordResetRequest(c)
	if err != nil {
		return err
	}
	if err := h.service.ResetPassword(c.Context(), usersActor(c), c.Params("user_id"), input); err != nil {
		return usersHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/handler.go
git commit -m "feat: add users handler with request binding and domain-error to HTTP mapping"
```

(This task has no separate red/green cycle of its own — `go build ./...` must still succeed since `route.go` does not exist yet, so the package will not compile until Task 13. Verify with `cd backend && go vet ./internal/modules/users/... 2>&1 | grep -v 'undefined: RegisterRoutes'` if you want an early sanity check, but do not expect a clean `go build` until Task 13's `route.go` lands.)

---

## Task 13: Routes, admin-only wiring, and end-to-end handler tests

**Files:**
- Create: `backend/internal/modules/users/route.go`
- Create: `backend/internal/modules/users/handler_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package users

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type fakeHandlerService struct {
	createFn     func(Actor, UserCreateInput) (User, error)
	listFn       func(Actor, ListFilter) (Page[User], error)
	getFn        func(Actor, string) (User, error)
	updateFn     func(Actor, string, UserUpdateInput) (User, error)
	deactivateFn func(Actor, string) error
	resetFn      func(Actor, string, PasswordResetInput) error
}

func (f *fakeHandlerService) CreateUser(_ context.Context, actor Actor, input UserCreateInput) (User, error) {
	return f.createFn(actor, input)
}
func (f *fakeHandlerService) ListUsers(_ context.Context, actor Actor, filter ListFilter) (Page[User], error) {
	return f.listFn(actor, filter)
}
func (f *fakeHandlerService) GetUser(_ context.Context, actor Actor, id string) (User, error) {
	return f.getFn(actor, id)
}
func (f *fakeHandlerService) UpdateUser(_ context.Context, actor Actor, id string, input UserUpdateInput) (User, error) {
	return f.updateFn(actor, id, input)
}
func (f *fakeHandlerService) DeactivateUser(_ context.Context, actor Actor, id string) error {
	return f.deactivateFn(actor, id)
}
func (f *fakeHandlerService) ResetPassword(_ context.Context, actor Actor, id string, input PasswordResetInput) error {
	return f.resetFn(actor, id, input)
}

func usersTestServer(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "bwims-users-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("auth.NewTokenManager() error = %v", err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	RegisterRoutes(app, NewHandler(service), tokens)
	return app, tokens
}

func authenticatedUsersRequest(
	t *testing.T,
	app *fiber.App,
	tokens *auth.TokenManager,
	user auth.User,
	method string,
	path string,
	body string,
) *http.Response {
	t.Helper()
	pair, err := tokens.Issue(user)
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+pair.AccessToken)
	if body != "" {
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	return response
}

func trivialUsersService() *fakeHandlerService {
	return &fakeHandlerService{
		createFn:     func(Actor, UserCreateInput) (User, error) { return User{ID: "u1"}, nil },
		listFn:       func(Actor, ListFilter) (Page[User], error) { return Page[User]{Items: []User{}}, nil },
		getFn:        func(Actor, string) (User, error) { return User{ID: "u1"}, nil },
		updateFn:     func(Actor, string, UserUpdateInput) (User, error) { return User{ID: "u1"}, nil },
		deactivateFn: func(Actor, string) error { return nil },
		resetFn:      func(Actor, string, PasswordResetInput) error { return nil },
	}
}

func TestUsersRoutesRequireAuthentication(t *testing.T) {
	app, _ := usersTestServer(t, trivialUsersService())

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}

func TestUsersRoutesEnforceAdminOnlyAccess(t *testing.T) {
	app, tokens := usersTestServer(t, trivialUsersService())

	requests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list", http.MethodGet, "/api/v1/users", ""},
		{"create", http.MethodPost, "/api/v1/users", `{"email":"a@bwims.test","full_name":"A","role":"picker","password":"at-least-12-chars"}`},
		{"get", http.MethodGet, "/api/v1/users/11111111-1111-1111-1111-111111111111", ""},
		{"update", http.MethodPut, "/api/v1/users/11111111-1111-1111-1111-111111111111", `{"full_name":"A","role":"picker","is_active":true}`},
		{"deactivate", http.MethodDelete, "/api/v1/users/11111111-1111-1111-1111-111111111111", ""},
		{"password-reset", http.MethodPost, "/api/v1/users/11111111-1111-1111-1111-111111111111/password-reset", `{"password":"at-least-12-chars"}`},
	}
	roles := []string{auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker, auth.RoleViewer}

	for _, req := range requests {
		for _, role := range roles {
			t.Run(req.name+"_"+role, func(t *testing.T) {
				response := authenticatedUsersRequest(t, app, tokens,
					auth.User{ID: "actor-1", Email: "actor@bwims.test", FullName: "Actor", Role: role},
					req.method, req.path, req.body)
				defer response.Body.Close()

				if role == auth.RoleAdmin {
					if response.StatusCode == fiber.StatusForbidden {
						t.Fatalf("admin got 403 for %s %s, want non-403", req.method, req.path)
					}
				} else if response.StatusCode != fiber.StatusForbidden {
					t.Fatalf("role %s got %d for %s %s, want 403", role, response.StatusCode, req.method, req.path)
				}
			})
		}
	}
}

func TestCreateUserHandlerMapsDomainErrors(t *testing.T) {
	service := &fakeHandlerService{
		createFn: func(Actor, UserCreateInput) (User, error) { return User{}, ErrEmailConflict },
	}
	app, tokens := usersTestServer(t, service)

	response := authenticatedUsersRequest(t, app, tokens, auth.User{ID: "admin-1", Role: auth.RoleAdmin},
		http.MethodPost, "/api/v1/users",
		`{"email":"dup@bwims.test","full_name":"Dup","role":"picker","password":"at-least-12-chars"}`)
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !bytes.Contains(body, []byte(`"code":"EMAIL_CONFLICT"`)) {
		t.Fatalf("body = %s, want EMAIL_CONFLICT", body)
	}
}

func TestDeactivateUserHandlerMapsSelfDeactivationError(t *testing.T) {
	service := &fakeHandlerService{
		deactivateFn: func(Actor, string) error { return ErrSelfDeactivationForbidden },
	}
	app, tokens := usersTestServer(t, service)

	response := authenticatedUsersRequest(t, app, tokens, auth.User{ID: "admin-1", Role: auth.RoleAdmin},
		http.MethodDelete, "/api/v1/users/11111111-1111-1111-1111-111111111111", "")
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !bytes.Contains(body, []byte(`"code":"SELF_DEACTIVATION_FORBIDDEN"`)) {
		t.Fatalf("body = %s, want SELF_DEACTIVATION_FORBIDDEN", body)
	}
}

func TestListUsersHandlerReturnsEnvelope(t *testing.T) {
	app, tokens := usersTestServer(t, trivialUsersService())

	response := authenticatedUsersRequest(t, app, tokens, auth.User{ID: "admin-1", Role: auth.RoleAdmin},
		http.MethodGet, "/api/v1/users", "")
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !bytes.Contains(body, []byte(`"success":true`)) || !bytes.Contains(body, []byte(`"items":[]`)) {
		t.Fatalf("body = %s, want success envelope with empty items", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/users/... -v`
Expected: FAIL — `undefined: RegisterRoutes` (package still does not compile).

- [ ] **Step 3: Write minimal implementation**

```go
package users

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	adminOnly := auth.RequireRoles(auth.RoleAdmin)

	usersGroup := api.Group("/users", adminOnly)
	usersGroup.Get("", handler.ListUsers)
	usersGroup.Post("", handler.CreateUser)
	usersGroup.Get("/:user_id", handler.GetUser)
	usersGroup.Put("/:user_id", handler.UpdateUser)
	usersGroup.Delete("/:user_id", handler.DeactivateUser)
	usersGroup.Post("/:user_id/password-reset", handler.ResetPassword)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/users/... -v`
Expected: PASS for every test in the package (unit + handler tests; the Postgres lifecycle test still skips without `BWIMS_TEST_DATABASE_URL`).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/users/route.go backend/internal/modules/users/handler_test.go
git commit -m "feat: add users routes with admin-only access and end-to-end handler tests"
```

---

## Task 14: Wire the module into `app.go`

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/app_test.go`

- [ ] **Step 1: Write the failing test**

Add to `backend/internal/app/app_test.go`, after `TestCatalogRoutesAreRegisteredBeforeNotFoundHandler`:

```go
func TestUsersRoutesAreRegisteredBeforeNotFoundHandler(t *testing.T) {
	server, err := New(config.Config{
		Environment: "test",
		Auth: config.AuthConfig{
			Issuer: "bwims-app-test", AccessTokenTTL: time.Minute,
			RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
		},
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	response, err := server.Test(httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusUnauthorized {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 401 from registered authenticated route; body=%s", response.StatusCode, body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/app/... -run TestUsersRoutesAreRegisteredBeforeNotFoundHandler -v`
Expected: FAIL — status 404 (`ROUTE_NOT_FOUND`), because `users.RegisterRoutes` is not wired into `app.New` yet.

- [ ] **Step 3: Wire the module into `app.go`**

In `backend/internal/app/app.go`, add the import:

```go
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/users"
```

(alphabetically after `"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/warehouse"` if the import block is sorted by full path — `users` sorts after `warehouse`... actually `u` < `w` alphabetically, so place the `users` import line immediately **before** the `warehouse` import line, keeping the block `goimports`-sorted.)

Then insert the registration block after the existing `warehouse.RegisterRoutes(...)` line and **before** the trailing `server.Use(func(c fiber.Ctx) error { ... ROUTE_NOT_FOUND ... })` fallback:

```go
	warehouseRepository := warehouse.NewPostgresRepository(db)
	warehouseService := warehouse.NewService(warehouseRepository)
	warehouse.RegisterRoutes(server, warehouse.NewHandler(warehouseService), tokenManager)

	usersRepository := users.NewPostgresRepository(db)
	usersService := users.NewService(usersRepository)
	users.RegisterRoutes(server, users.NewHandler(usersService), tokenManager)

	server.Use(func(c fiber.Ctx) error {
		return httpx.NewError(fiber.StatusNotFound, "ROUTE_NOT_FOUND", "route not found", nil)
	})
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/app/... -v`
Expected: PASS, including the pre-existing `TestWarehouseRoutesAreRegisteredBeforeNotFoundHandler` and `TestCatalogRoutesAreRegisteredBeforeNotFoundHandler`.

- [ ] **Step 5: Run the full backend test suite and `go vet` to confirm nothing else broke, then commit**

```bash
cd backend
go build ./...
go vet ./...
go test ./... -count=1
go mod tidy
git add internal/app/app.go internal/app/app_test.go go.mod go.sum
git commit -m "feat: wire admin user management routes into the application"
```

(`go mod tidy` promotes `github.com/google/uuid` from an indirect to a direct dependency, since `users/pagination.go` and `users/service.go` now import it directly — the same way `catalog` already does. Review the `go.mod`/`go.sum` diff before committing; it should only move that one line from the indirect block to the direct `require` block, with no version changes.)

---

## Task 15: Postgres lifecycle re-run and full backend validation

**Files:** none (verification only)

- [ ] **Step 1: Run the full users lifecycle integration test against a real database**

Bring up Postgres the same way the catalog slice's validation did (see `docs/week-8-catalog-validation.md`, "Database migration status" section, for the exact connection string / `docker compose` invocation used previously), then:

```bash
cd backend
export BWIMS_TEST_DATABASE_URL="<same connection string used for the catalog validation>"
go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v
```

Expected: PASS, with no skipped subtests.

- [ ] **Step 2: Run the complete backend suite once more with the database available**

```bash
cd backend
go test ./... -count=1 -v 2>&1 | tee /tmp/users-backend-test-output.txt
go vet ./...
```

Expected: all packages PASS, `go vet` reports nothing.

- [ ] **Step 3: Confirm no migration drift**

```bash
cd backend
ls migrations/ | tail -5
```

Expected: still ends at `000004_catalog_constraints.up.sql` / `.down.sql` — this plan intentionally adds **no** new migration (see the design doc's "Migration" section: all needed tables/indexes already exist from `000002_initial_schema`). If Task 6-9 testing surfaced a genuine gap (for example, wanting an index for full-name search at scale), stop and flag it to the user rather than silently adding `000005_*` — that decision needs a human call per the design doc.

- [ ] **Step 4: No commit for this task** — it is a verification checkpoint. If anything fails, fix it in the relevant earlier task's files and re-commit there (do not silently patch around it here).

---

## Task 16: Update `docs/api-contract.md`

**Files:**
- Modify: `docs/api-contract.md`

- [ ] **Step 1: Add a new "Admin user management endpoints" section**

Insert a new `##` section after the existing "Category, product, and barcode endpoints" section and before "Initial HTTP status policy", following the exact template already used for Category/Product (one-line scope statement, `### Access rules` table, `### User routes` table + example body + normalization prose, `### List queries and response`, `### Deactivation, password reset, and errors`):

```markdown
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
```

- [ ] **Step 2: Commit**

```bash
git add docs/api-contract.md
git commit -m "docs: record admin user management API contract"
```

---

## Task 17: Update `docs/requirements-traceability.md`

**Files:**
- Modify: `docs/requirements-traceability.md`

- [ ] **Step 1: Flip the FR-3 row from Planned to Implemented**

Change the existing row:

```
| FR-3 | Admin user management | 2.2 | Week 8 domain APIs | API integration tests | Planned |
```

to:

```
| FR-3 | Admin user management | 2.2 | Week 9 users module | Service, handler and PostgreSQL repository tests | Implemented |
```

This matches the exact phrasing style already used for FR-4/FR-6 in the same table (verification-method wording).

- [ ] **Step 2: Commit**

```bash
git add docs/requirements-traceability.md
git commit -m "docs: mark FR-3 admin user management as implemented"
```

Note: do not touch the Gantt spreadsheet or Linear from this repo — that update happens after the PR is reviewed and merged (Task 19), and only with the user's explicit confirmation per the project's standing rules.

---

## Task 18: Create `docs/week-9-user-management-validation.md`

**Files:**
- Create: `docs/week-9-user-management-validation.md`

- [ ] **Step 1: Write the validation evidence doc, mirroring `docs/week-8-catalog-validation.md`'s structure exactly**

```markdown
# Week 9 admin user management validation

**Validation date:** <fill in the actual date this was run>
**Branch:** `codex/pro-16-admin-user-management`
**API checkpoint:** `<fill in the commit hash of the last commit from Task 14>`

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
BWIMS_TEST_DATABASE_URL=<connection string> go test ./internal/modules/users/... -run TestPostgresUsersLifecycle -v
```

Result on <fill in date>:
- `go build ./...` — <PASS/FAIL, paste exact output if FAIL>
- `go vet ./...` — <PASS/FAIL>
- `go test ./... -count=1` — <PASS/FAIL, paste package summary line>
- `TestPostgresUsersLifecycle` — <PASS/FAIL>

Frontend re-run (no frontend code changed in this slice, but confirming no
regression):

```text
npm run test
npm run typecheck
npm run build
```

Result on <fill in date>: <PASS/FAIL for each>

## Database migration status

<Fill in: which Postgres instance was used (same as the catalog validation's
setup), confirmation that no new migration file was added, and cleanup notes
from the integration test's `t.Cleanup` (test users, warehouse, and refresh
token rows are deleted by suffix/email-domain match after the run).>

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
```

- [ ] **Step 2: Commit**

```bash
git add docs/week-9-user-management-validation.md
git commit -m "docs: record week 9 admin user management validation evidence"
```

(Fill in the actual dates, commit hash, and command output for real before this commit — an evidence doc with placeholder brackets left in is not acceptable per the project's "do not claim things happened without real evidence" rule. This is the one place in the plan where the exact values cannot be known until the prior tasks have actually run.)

---

## Task 19: Final validation and PR

**Files:** none (verification and a GitHub PR only — no code changes)

- [ ] **Step 1: Run the complete validation suite one more time from a clean state**

```bash
cd backend
go build ./...
go vet ./...
go test ./... -count=1
git diff --check
```

```bash
docker compose config --quiet
```

```bash
cd ../frontend   # or wherever the frontend package.json lives, per the catalog slice's own commands
npm run test
npm run typecheck
npm run build
```

Expected: everything passes. If anything fails, go back to the task that owns the failing file and fix it there — do not patch it inline in this task.

- [ ] **Step 2: Push the branch**

```bash
git push -u origin codex/pro-16-admin-user-management
```

- [ ] **Step 3: Open a pull request (separate from PR #11)**

```bash
gh pr create --title "Implement Admin User Management API" --body "$(cat <<'EOF'
## Summary
- Admin-only CRUD for user accounts (create, list, read, update, deactivate)
- Admin-triggered password reset
- Last-admin lockout and self-deactivation/self-demotion protection
- Deactivation and password reset revoke the target's active refresh tokens
- No new migration — reuses existing users/roles/refresh_tokens/warehouses schema

## Test plan
- [x] `go build ./...`
- [x] `go vet ./...`
- [x] `go test ./... -count=1`
- [x] Postgres lifecycle integration test (TestPostgresUsersLifecycle)
- [x] `docker compose config --quiet`
- [x] Frontend test/typecheck/build re-run (no frontend changes in this slice)

Closes FR-3 / completes Gantt task 2.2 (pending review and merge).
EOF
)"
```

**Do not merge this PR, mark it ready-for-review-then-auto-merge, or close any
Linear/GitHub issue as part of this task** — per the project's standing rule,
merges and issue closures require the user's explicit approval. Report the PR
URL back and stop.

- [ ] **Step 4: Report status, do not touch Linear/Gantt automatically**

Tell the user: branch pushed, PR opened at `<url>`, all automated checks
green, `docs/week-9-user-management-validation.md` has real command output
(not placeholders). Ask the user before updating Linear PRO-16 status, the
Gantt task 2.2 percentage, or FR-3 status anywhere outside this repo's own
`docs/requirements-traceability.md` (which Task 17 already updated in-repo).

---

## Self-Review Notes

**Spec coverage check against the original design doc:**
- Create/list/get/update/deactivate/password-reset routes — Tasks 5-13. ✓
- Admin-only RBAC including reads — Task 13 (`adminOnly` applied at group level). ✓
- Search/filter by name, email, role, warehouse, active status + cursor pagination — Task 6 (repository), Task 10 (service). ✓
- Deactivation and password-reset revoke refresh tokens — Task 8, Task 9. ✓
- Last-admin lockout inside the same transaction as the write — Task 7 (`UpdateUser`), Task 8 (`DeactivateUser`), via shared `requireMoreThanOneActiveAdmin`. ✓
- Self-deactivation/self-demotion rejected unconditionally — Task 11 (service layer, has actor identity). ✓
- No new migration — Task 15 Step 3 explicitly checks for and flags drift. ✓
- `docs/api-contract.md`, `docs/requirements-traceability.md`, week-9 validation doc — Tasks 16-18. ✓
- Gantt/Linear updates deferred to explicit user approval — Task 19 Step 4, matches the project's standing rule 5/6. ✓
- Frontend (PRO-11) explicitly out of scope — called out in Task 18's "Remaining work" section. ✓

**Type/signature consistency check:** `Repository` interface (Task 5) methods match `PostgresRepository` and `fakeRepository` implementations exactly across Tasks 6-11. `Service` interface (Task 10) methods match `service` and `fakeHandlerService` implementations exactly across Tasks 10-13. `ListFilter`, `UserCreateInput`, `UserUpdateInput`, `PasswordResetInput`, `Actor`, `User`, `Page[User]` are defined once in Task 3 and used with the same field names throughout — no renamed fields between tasks.
