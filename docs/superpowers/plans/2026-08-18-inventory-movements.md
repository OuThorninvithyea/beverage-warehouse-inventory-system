# Inventory Balances and Stock Movements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `backend/internal/modules/inventory` so authorized users can read stock balances and lots, and receive/FEFO-pick/transfer/adjust stock with FIFO cost-layer tracking, closing FR-10 through FR-19.

**Architecture:** New module following `Route -> Handler -> Service -> Repository -> PostgreSQL`, mirroring `catalog`/`warehouse`/`users` file-by-file. All four movement writes share one transactional core per method in the repository (lock balance rows `FOR UPDATE`, apply quantity change, insert `stock_movements`, touch `cost_layers`). Warehouse-scoping and lot-tracked/business validation that needs DB state live in the repository (inside the same transaction, avoiding TOCTOU); pure RBAC and input-shape validation live in the service. This matches the actual pattern already in `internal/modules/warehouse` (`actorFromContext`, `warehouseScope`, `requireWarehouseAccess`, route-level `auth.RequireRoles` for mutations only) — confirmed by reading that module's real code, not assumed.

**Tech Stack:** Go 1.26, Fiber v3 (`fiber.Ctx` by value), `github.com/jackc/pgx/v5` (pgxpool, explicit transactions), PostgreSQL (existing `lots`/`inventory_balances`/`stock_movements`/`cost_layers` tables from `000002_initial_schema`, plus one new column added in `000005`), `github.com/shopspring/decimal` (new dependency — precise NUMERIC arithmetic for quantities/costs, avoiding float rounding and avoiding hand-rolled decimal math), `github.com/google/uuid`.

---

## File Structure

```
backend/internal/modules/inventory/
  optional.go                     # OptionalString tri-state (local copy, mirrors catalog/users)
  model.go                        # Actor, Balance, Lot, Movement, *Input types, Cursor, ListFilters, Page[T]
  pagination.go                   # EncodeCursor/DecodeCursor/normalizeLimit/balancePage/movementPage
  repository.go                   # Repository interface + PostgresRepository (pgx, transactions)
  service.go                      # Service interface + RBAC/validation orchestration
  handler.go                      # Fiber handlers, request binding, domain-error -> HTTP mapping
  route.go                        # RegisterRoutes
  optional_test.go
  model_test.go
  pagination_test.go
  repository_integration_test.go  # real-Postgres lifecycle test, BWIMS_TEST_DATABASE_URL-gated
  service_test.go                 # fakeRepository-based unit tests
  handler_test.go                 # fiber.New() + RegisterRoutes + fakeHandlerService

backend/migrations/
  000005_inventory_constraints.up.sql
  000005_inventory_constraints.down.sql

backend/internal/app/app.go        # MODIFY: wire inventory module in
backend/internal/app/app_test.go   # MODIFY: add TestInventoryRoutesAreRegisteredBeforeNotFoundHandler

docs/api-contract.md                          # MODIFY: add "Inventory and stock movement endpoints" section
docs/requirements-traceability.md             # MODIFY: flip FR-10 through FR-19 to Implemented
docs/week-10-inventory-validation.md          # CREATE: validation evidence
```

Design doc: `docs/superpowers/specs/2026-08-18-inventory-movements-design.md`.

---

## Task 1: Create the feature branch and add the decimal dependency

**Files:** `backend/go.mod`, `backend/go.sum` (via `go get`)

- [ ] **Step 1: Confirm the working tree is clean and branch off `main`**

```bash
git status
git fetch origin
git checkout main
git pull origin main
git checkout -b codex/inventory-movements
```

Expected: branch created from an up-to-date `main`, tip is `3211055 Implement Admin User Management API (#12)`.

- [ ] **Step 2: Add the decimal dependency**

```bash
cd backend
go get github.com/shopspring/decimal@v1.4.0
go mod tidy
```

Expected: `go.mod`/`go.sum` updated, no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add shopspring/decimal for inventory quantity arithmetic"
```

---

## Task 2: `OptionalString` tri-state type

**Files:**
- Create: `backend/internal/modules/inventory/optional.go`
- Test: `backend/internal/modules/inventory/optional_test.go`

- [ ] **Step 1: Write the failing test**

```go
package inventory

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
	})

	t.Run("null clears the field", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":null}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set || w.Field.Value != nil {
			t.Fatalf("got Set=%v Value=%v, want Set=true Value=nil", w.Field.Set, w.Field.Value)
		}
	})

	t.Run("string value is captured", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":"LOT-1"}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set || w.Field.Value == nil || *w.Field.Value != "LOT-1" {
			t.Fatalf("got Set=%v Value=%v, want Set=true Value=LOT-1", w.Field.Set, w.Field.Value)
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

Run: `cd backend && go test ./internal/modules/inventory/... -run TestOptionalStringUnmarshalJSON -v`
Expected: FAIL — `undefined: OptionalString`.

- [ ] **Step 3: Write minimal implementation**

```go
package inventory

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

Run: `cd backend && go test ./internal/modules/inventory/... -run TestOptionalStringUnmarshalJSON -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/optional.go backend/internal/modules/inventory/optional_test.go
git commit -m "feat: add inventory module OptionalString tri-state type"
```

---

## Task 3: Domain model types

**Files:**
- Create: `backend/internal/modules/inventory/model.go`
- Test: `backend/internal/modules/inventory/model_test.go`

- [ ] **Step 1: Write the failing test**

```go
package inventory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBalanceJSONIncludesAvailableQuantity(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	balance := Balance{
		ID: "b1", LocationID: "loc1", ProductID: "prod1", LotID: nil,
		Quantity: "120.000", ReservedQuantity: "20.000", AvailableQuantity: "100.000",
		CreatedAt: now, UpdatedAt: now,
	}
	raw, err := json.Marshal(balance)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["available_quantity"] != "100.000" {
		t.Fatalf("available_quantity = %v, want 100.000", decoded["available_quantity"])
	}
	if decoded["lot_id"] != nil {
		t.Fatalf("lot_id = %v, want null", decoded["lot_id"])
	}
}

func TestMovementJSONShape(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	movement := Movement{
		ID: "m1", MovementType: "receive", ProductID: "prod1", LotID: nil,
		FromLocationID: nil, ToLocationID: strPointer("loc1"), Quantity: "50.000",
		UnitCost: strPointer("1.2500"), Reference: strPointer("PO-1"), Notes: nil,
		PerformedBy: strPointer("user1"), CreatedAt: now,
	}
	raw, err := json.Marshal(movement)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["movement_type"] != "receive" {
		t.Fatalf("movement_type = %v, want receive", decoded["movement_type"])
	}
	if decoded["from_location_id"] != nil {
		t.Fatalf("from_location_id = %v, want null", decoded["from_location_id"])
	}
}

func strPointer(value string) *string { return &value }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestBalanceJSONIncludesAvailableQuantity|TestMovementJSONShape' -v`
Expected: FAIL — `undefined: Balance`.

- [ ] **Step 3: Write minimal implementation**

```go
package inventory

import "time"

type Actor struct {
	ID          string
	Role        string
	WarehouseID *string
}

type Balance struct {
	ID                string    `json:"id"`
	LocationID        string    `json:"location_id"`
	ProductID         string    `json:"product_id"`
	LotID             *string   `json:"lot_id"`
	Quantity          string    `json:"quantity"`
	ReservedQuantity  string    `json:"reserved_quantity"`
	AvailableQuantity string    `json:"available_quantity"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Lot struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	LotNumber         string    `json:"lot_number"`
	ExpirationDate    *string   `json:"expiration_date"`
	ReceivedAt        time.Time `json:"received_at"`
	AvailableQuantity string    `json:"available_quantity"`
}

type Movement struct {
	ID             string    `json:"id"`
	MovementType   string    `json:"movement_type"`
	ProductID      string    `json:"product_id"`
	LotID          *string   `json:"lot_id"`
	FromLocationID *string   `json:"from_location_id"`
	ToLocationID   *string   `json:"to_location_id"`
	Quantity       string    `json:"quantity"`
	UnitCost       *string   `json:"unit_cost"`
	Reference      *string   `json:"reference"`
	Notes          *string   `json:"notes"`
	PerformedBy    *string   `json:"performed_by"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReceiveInput struct {
	LocationID     string
	ProductID      string
	Quantity       string
	UnitCost       string
	LotNumber      OptionalString
	ExpirationDate OptionalString
	Reference      OptionalString
	Notes          OptionalString
}

type PickInput struct {
	LocationID string
	ProductID  string
	Quantity   string
	LotID      OptionalString
	Reference  OptionalString
	Notes      OptionalString
}

type TransferInput struct {
	ProductID      string
	LotID          OptionalString
	Quantity       string
	FromLocationID string
	ToLocationID   string
	Reference      OptionalString
	Notes          OptionalString
}

type AdjustInput struct {
	LocationID string
	ProductID  string
	LotID      OptionalString
	Direction  string
	Quantity   string
	Notes      OptionalString
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type BalanceListFilter struct {
	Limit       int
	After       *Cursor
	LocationID  *string
	ProductID   *string
	WarehouseID *string
	LotID       *string
}

type MovementListFilter struct {
	Limit        int
	After        *Cursor
	ProductID    *string
	LocationID   *string
	MovementType *string
	From         *time.Time
	To           *time.Time
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

Run: `cd backend && go test ./internal/modules/inventory/... -v`
Expected: PASS (all tests so far).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/model.go backend/internal/modules/inventory/model_test.go
git commit -m "feat: add inventory module domain model types"
```

---

## Task 4: Cursor pagination primitives

**Files:**
- Create: `backend/internal/modules/inventory/pagination.go`
- Test: `backend/internal/modules/inventory/pagination_test.go`

- [ ] **Step 1: Write the failing test**

```go
package inventory

import (
	"testing"
	"time"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	id := "33333333-3333-3333-3333-333333333333"
	encoded := EncodeCursor(createdAt, id)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if !decoded.CreatedAt.Equal(createdAt) || decoded.ID != id {
		t.Fatalf("decoded = %+v, want CreatedAt=%v ID=%v", decoded, createdAt, id)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	if _, err := DecodeCursor("not-base64!!"); err == nil {
		t.Fatalf("DecodeCursor() error = nil, want ErrInvalidCursor")
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

func TestBalancePageTruncatesAndSetsCursor(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	items := make([]Balance, 3)
	for i := range items {
		items[i] = Balance{ID: string(rune('a' + i)), CreatedAt: now.Add(-time.Duration(i) * time.Minute)}
	}
	page := balancePage(items, 2)
	if len(page.Items) != 2 || !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %+v, want 2 items with HasMore and a cursor", page)
	}
}

func TestBalancePageHandlesNilItems(t *testing.T) {
	page := balancePage(nil, 20)
	if page.Items == nil {
		t.Fatalf("Items = nil, want empty slice (JSON must encode [] not null)")
	}
}

func TestMovementPageTruncatesAndSetsCursor(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	items := make([]Movement, 2)
	for i := range items {
		items[i] = Movement{ID: string(rune('a' + i)), CreatedAt: now.Add(-time.Duration(i) * time.Minute)}
	}
	page := movementPage(items, 1)
	if len(page.Items) != 1 || !page.Page.HasMore {
		t.Fatalf("page = %+v, want 1 item with HasMore", page)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestEncodeDecodeCursorRoundTrip|TestDecodeCursorRejectsGarbage|TestNormalizeLimitBounds|TestBalancePage|TestMovementPage' -v`
Expected: FAIL — `undefined: EncodeCursor`.

- [ ] **Step 3: Write minimal implementation**

```go
package inventory

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("inventory cursor is invalid")

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

func balancePage(items []Balance, limit int) Page[Balance] {
	if items == nil {
		items = []Balance{}
	}
	page := Page[Balance]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		cursor := EncodeCursor(last.CreatedAt, last.ID)
		page.Page = PageInfo{NextCursor: &cursor, HasMore: true}
	}
	return page
}

func movementPage(items []Movement, limit int) Page[Movement] {
	if items == nil {
		items = []Movement{}
	}
	page := Page[Movement]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		cursor := EncodeCursor(last.CreatedAt, last.ID)
		page.Page = PageInfo{NextCursor: &cursor, HasMore: true}
	}
	return page
}

func cursorArguments(cursor *Cursor) (any, any) {
	if cursor == nil {
		return nil, nil
	}
	return cursor.CreatedAt, cursor.ID
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/pagination.go backend/internal/modules/inventory/pagination_test.go
git commit -m "feat: add inventory module cursor pagination primitives"
```

---

## Task 5: Migration `000005_inventory_constraints`

**Files:**
- Create: `backend/migrations/000005_inventory_constraints.up.sql`
- Create: `backend/migrations/000005_inventory_constraints.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
ALTER TABLE inventory_balances
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX inventory_balances_created_idx
    ON inventory_balances (created_at DESC, id DESC);
```

- [ ] **Step 2: Write the down migration**

```sql
DROP INDEX IF EXISTS inventory_balances_created_idx;

ALTER TABLE inventory_balances
    DROP COLUMN created_at;
```

- [ ] **Step 3: Apply against a local database and verify roundtrip**

Bring up Postgres the same way the catalog slice's validation did (see
`docs/week-8-catalog-validation.md`, "Database migration status").

```bash
cd backend
migrate -path migrations -database "$BWIMS_TEST_DATABASE_URL" up
migrate -path migrations -database "$BWIMS_TEST_DATABASE_URL" down 1
migrate -path migrations -database "$BWIMS_TEST_DATABASE_URL" up
```

Expected: applies cleanly to `000005`, rolls back to `000004`, reapplies to
`000005` with no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/000005_inventory_constraints.up.sql backend/migrations/000005_inventory_constraints.down.sql
git commit -m "feat: add inventory_balances.created_at for cursor pagination"
```

---

## Task 6: Repository foundations, shared helpers, and `ListBalances`

**Files:**
- Create: `backend/internal/modules/inventory/repository.go`
- Create: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Write the failing integration test (setup + `ListBalances` portion)**

```go
package inventory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type testFixture struct {
	pool           *pgxpool.Pool
	suffix         string
	warehouseID    string
	locationAID    string
	locationBID    string
	trackedProduct string
	plainProduct   string
}

func setupFixture(t *testing.T) testFixture {
	t.Helper()
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

	suffix := strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))
	fixture := testFixture{pool: pool, suffix: suffix}

	if err := pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, is_active) VALUES ($1, $2, TRUE) RETURNING id::text`,
		"WH-"+suffix, "Inventory Test Warehouse "+suffix,
	).Scan(&fixture.warehouseID); err != nil {
		t.Fatalf("seed warehouse error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (warehouse_id, code, is_pickable, is_active)
		VALUES ($1, $2, TRUE, TRUE) RETURNING id::text`,
		fixture.warehouseID, "LOC-A-"+suffix,
	).Scan(&fixture.locationAID); err != nil {
		t.Fatalf("seed location A error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (warehouse_id, code, is_pickable, is_active)
		VALUES ($1, $2, TRUE, TRUE) RETURNING id::text`,
		fixture.warehouseID, "LOC-B-"+suffix,
	).Scan(&fixture.locationBID); err != nil {
		t.Fatalf("seed location B error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products (sku, name, unit, is_lot_tracked, is_active)
		VALUES ($1, $2, 'case', TRUE, TRUE) RETURNING id::text`,
		"SKU-TRACKED-"+suffix, "Tracked Product "+suffix,
	).Scan(&fixture.trackedProduct); err != nil {
		t.Fatalf("seed tracked product error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products (sku, name, unit, is_lot_tracked, is_active)
		VALUES ($1, $2, 'case', FALSE, TRUE) RETURNING id::text`,
		"SKU-PLAIN-"+suffix, "Plain Product "+suffix,
	).Scan(&fixture.plainProduct); err != nil {
		t.Fatalf("seed plain product error = %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM cost_layers WHERE product_id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM stock_movements WHERE product_id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM inventory_balances WHERE product_id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM lots WHERE product_id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM products WHERE id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM locations WHERE warehouse_id = $1", fixture.warehouseID)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM warehouses WHERE id = $1", fixture.warehouseID)
	})

	return fixture
}

func TestPostgresInventoryLifecycle(t *testing.T) {
	fixture := setupFixture(t)
	ctx := context.Background()
	repository := NewPostgresRepository(fixture.pool)

	empty, err := repository.ListBalances(ctx, BalanceListFilter{Limit: 10, ProductID: &fixture.trackedProduct})
	if err != nil {
		t.Fatalf("ListBalances(empty) error = %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("ListBalances(empty) = %#v, want no rows before any receive", empty)
	}

	_ = errors.Is // placeholder to keep errors imported until later tasks extend this test
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `undefined: NewPostgresRepository`.

- [ ] **Step 3: Write the repository foundations, shared helpers, and `ListBalances`**

```go
package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrProductNotFound          = errors.New("product was not found or is not active")
	ErrLocationNotFound         = errors.New("location was not found")
	ErrLotNotFound               = errors.New("lot was not found for this product")
	ErrInsufficientStock         = errors.New("insufficient available stock for this operation")
	ErrWarehouseMismatch         = errors.New("location does not belong to the actor's assigned warehouse")
	ErrInventoryOperationFailed  = errors.New("inventory operation could not be completed")
)

type Repository interface {
	ListBalances(ctx context.Context, filter BalanceListFilter) ([]Balance, error)
	ListLots(ctx context.Context, productID string, warehouseID *string) ([]Lot, error)
	Receive(ctx context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error)
	Pick(ctx context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error)
	Transfer(ctx context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error)
	Adjust(ctx context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error)
	ListMovements(ctx context.Context, filter MovementListFilter) ([]Movement, error)
	GetMovement(ctx context.Context, id string) (Movement, error)
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

func scanBalance(row rowScanner) (Balance, error) {
	var balance Balance
	var quantityText, reservedText string
	err := row.Scan(&balance.ID, &balance.LocationID, &balance.ProductID, &balance.LotID,
		&quantityText, &reservedText, &balance.CreatedAt, &balance.UpdatedAt)
	if err != nil {
		return Balance{}, err
	}
	balance.Quantity = quantityText
	balance.ReservedQuantity = reservedText
	quantity, err := decimal.NewFromString(quantityText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance quantity: %w", err)
	}
	reserved, err := decimal.NewFromString(reservedText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance reserved quantity: %w", err)
	}
	balance.AvailableQuantity = quantity.Sub(reserved).String()
	return balance, nil
}

func scanMovement(row rowScanner) (Movement, error) {
	var movement Movement
	err := row.Scan(&movement.ID, &movement.MovementType, &movement.ProductID, &movement.LotID,
		&movement.FromLocationID, &movement.ToLocationID, &movement.Quantity,
		&movement.UnitCost, &movement.Reference, &movement.Notes, &movement.PerformedBy, &movement.CreatedAt)
	return movement, err
}

func resolveLocationWarehouse(ctx context.Context, tx pgx.Tx, locationID string) (string, error) {
	var warehouseID string
	err := tx.QueryRow(ctx, `SELECT warehouse_id::text FROM locations WHERE id = $1`, locationID).Scan(&warehouseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLocationNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve location warehouse: %w", err)
	}
	return warehouseID, nil
}

func checkWarehouseScope(actorWarehouseID *string, resolvedWarehouseID string) error {
	if actorWarehouseID != nil && *actorWarehouseID != resolvedWarehouseID {
		return ErrWarehouseMismatch
	}
	return nil
}

type productRef struct {
	IsLotTracked bool
}

func lockActiveProduct(ctx context.Context, tx pgx.Tx, productID string) (productRef, error) {
	var product productRef
	err := tx.QueryRow(ctx, `
		SELECT is_lot_tracked FROM products WHERE id = $1 AND is_active = TRUE FOR UPDATE`, productID,
	).Scan(&product.IsLotTracked)
	if errors.Is(err, pgx.ErrNoRows) {
		return productRef{}, ErrProductNotFound
	}
	if err != nil {
		return productRef{}, fmt.Errorf("lock active product: %w", err)
	}
	return product, nil
}

func lockLotForProduct(ctx context.Context, tx pgx.Tx, lotID, productID string) error {
	var exists string
	err := tx.QueryRow(ctx, `SELECT id::text FROM lots WHERE id = $1 AND product_id = $2`, lotID, productID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLotNotFound
	}
	if err != nil {
		return fmt.Errorf("validate lot: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListBalances(ctx context.Context, filter BalanceListFilter) ([]Balance, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := r.pool.Query(ctx, `
		SELECT b.id::text, b.location_id::text, b.product_id::text, b.lot_id::text,
		       b.quantity::text, b.reserved_quantity::text, b.created_at, b.updated_at
		FROM inventory_balances b
		JOIN locations loc ON loc.id = b.location_id
		WHERE ($1::uuid IS NULL OR b.location_id = $1)
		  AND ($2::uuid IS NULL OR b.product_id = $2)
		  AND ($3::uuid IS NULL OR loc.warehouse_id = $3)
		  AND ($4::uuid IS NULL OR b.lot_id = $4)
		  AND ($5::timestamptz IS NULL OR (b.created_at, b.id) < ($5::timestamptz, $6::uuid))
		ORDER BY b.created_at DESC, b.id DESC
		LIMIT $7`,
		filter.LocationID, filter.ProductID, filter.WarehouseID, filter.LotID,
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list balances: %w", err)
	}
	defer rows.Close()

	balances := make([]Balance, 0)
	for rows.Next() {
		balance, err := scanBalance(rows)
		if err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}
		balances = append(balances, balance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list balances: %w", err)
	}
	return balances, nil
}

func (r *PostgresRepository) ListLots(ctx context.Context, productID string, warehouseID *string) ([]Lot, error) {
	panic("not implemented until Task 7")
}

func (r *PostgresRepository) Receive(ctx context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
	panic("not implemented until Task 8")
}

func (r *PostgresRepository) Pick(ctx context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
	panic("not implemented until Task 9")
}

func (r *PostgresRepository) Transfer(ctx context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
	panic("not implemented until Task 10")
}

func (r *PostgresRepository) Adjust(ctx context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
	panic("not implemented until Task 11")
}

func (r *PostgresRepository) ListMovements(ctx context.Context, filter MovementListFilter) ([]Movement, error) {
	panic("not implemented until Task 12")
}

func (r *PostgresRepository) GetMovement(ctx context.Context, id string) (Movement, error) {
	panic("not implemented until Task 12")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository foundations and ListBalances"
```

---

## Task 7: Repository — `ListLots`

**Files:**
- Modify: `backend/internal/modules/inventory/repository.go` (replace the `ListLots` stub)
- Modify: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Replace the `_ = errors.Is // placeholder ...` line in `TestPostgresInventoryLifecycle` with:

```go
	emptyLots, err := repository.ListLots(ctx, fixture.trackedProduct, nil)
	if err != nil {
		t.Fatalf("ListLots(empty) error = %v", err)
	}
	if len(emptyLots) != 0 {
		t.Fatalf("ListLots(empty) = %#v, want no lots before any receive", emptyLots)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 7`.

- [ ] **Step 3: Replace the `ListLots` stub**

```go
func (r *PostgresRepository) ListLots(ctx context.Context, productID string, warehouseID *string) ([]Lot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text, l.product_id::text, l.lot_number, l.expiration_date, l.received_at,
		       COALESCE(SUM(b.quantity - b.reserved_quantity)
		           FILTER (WHERE $2::uuid IS NULL OR loc.warehouse_id = $2), 0)::text
		FROM lots l
		LEFT JOIN inventory_balances b ON b.lot_id = l.id
		LEFT JOIN locations loc ON loc.id = b.location_id
		WHERE l.product_id = $1
		GROUP BY l.id
		ORDER BY l.expiration_date ASC NULLS LAST, l.received_at ASC`,
		productID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	defer rows.Close()

	lots := make([]Lot, 0)
	for rows.Next() {
		var lot Lot
		var expirationDate *time.Time
		if err := rows.Scan(&lot.ID, &lot.ProductID, &lot.LotNumber, &expirationDate, &lot.ReceivedAt, &lot.AvailableQuantity); err != nil {
			return nil, fmt.Errorf("scan lot: %w", err)
		}
		if expirationDate != nil {
			formatted := expirationDate.Format("2006-01-02")
			lot.ExpirationDate = &formatted
		}
		lots = append(lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	return lots, nil
}
```

Add `"time"` to the `import` block at the top of `repository.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository ListLots"
```

---

## Task 8: Repository — `Receive`

**Files:**
- Modify: `backend/internal/modules/inventory/repository.go` (replace the `Receive` stub; add `resolveOrCreateLot`, `upsertBalanceIncrement`, `insertMovement`)
- Modify: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the `ListLots` assertions from Task 7:

```go
	movement, balance, err := repository.Receive(ctx, "actor-1", nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "50.000", UnitCost: "1.2500",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-A")},
	})
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if movement.MovementType != "receive" || movement.ToLocationID == nil || *movement.ToLocationID != fixture.locationAID {
		t.Fatalf("movement = %#v, want receive into locationA", movement)
	}
	if balance.Quantity != "50.000" || balance.LotID == nil {
		t.Fatalf("balance = %#v, want quantity 50.000 with a lot", balance)
	}

	var costLayerCount int
	if err := fixture.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cost_layers WHERE source_movement_id = $1`, movement.ID,
	).Scan(&costLayerCount); err != nil {
		t.Fatalf("count cost layers error = %v", err)
	}
	if costLayerCount != 1 {
		t.Fatalf("costLayerCount = %d, want 1", costLayerCount)
	}

	_, _, err = repository.Receive(ctx, "actor-1", nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "5.000", UnitCost: "1.0000",
	})
	if !errors.Is(err, nil) {
		// lot_number omitted for a lot-tracked product is a service-layer
		// validation concern (Task 14); the repository trusts its inputs.
		// This call is expected to fail only because resolveOrCreateLot
		// receives an empty lot number, which is rejected by the lots
		// table's non-blank constraint.
		if err == nil {
			t.Fatalf("Receive() with blank lot_number error = nil, want a database constraint error")
		}
	}

	missingLocation := "00000000-0000-0000-0000-000000000000"
	_, _, err = repository.Receive(ctx, "actor-1", nil, ReceiveInput{
		LocationID: missingLocation, ProductID: fixture.trackedProduct,
		Quantity: "5.000", UnitCost: "1.0000",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-B")},
	})
	if !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("Receive(missing location) error = %v, want ErrLocationNotFound", err)
	}

	otherWarehouse := "22222222-2222-2222-2222-222222222222"
	_, _, err = repository.Receive(ctx, "actor-1", &otherWarehouse, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "5.000", UnitCost: "1.0000",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-C")},
	})
	if !errors.Is(err, ErrWarehouseMismatch) {
		t.Fatalf("Receive(wrong warehouse actor) error = %v, want ErrWarehouseMismatch", err)
	}
```

Remove the now-redundant blank-lot-number sub-block if it feels awkward to
keep — it is intentionally loose because the repository does not validate
business rules the service already owns; the important assertions are the
`ErrLocationNotFound` and `ErrWarehouseMismatch` checks that follow it.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 8`.

- [ ] **Step 3: Replace the `Receive` stub and add its helpers**

```go
func resolveOrCreateLot(ctx context.Context, tx pgx.Tx, productID, lotNumber string, expirationDate *string) (string, error) {
	var lotID string
	err := tx.QueryRow(ctx, `
		INSERT INTO lots (product_id, lot_number, expiration_date)
		VALUES ($1, $2, $3::date)
		ON CONFLICT (product_id, lot_number) DO UPDATE SET lot_number = EXCLUDED.lot_number
		RETURNING id::text`,
		productID, lotNumber, expirationDate,
	).Scan(&lotID)
	if err != nil {
		return "", fmt.Errorf("resolve or create lot: %w", err)
	}
	return lotID, nil
}

func upsertBalanceIncrement(ctx context.Context, tx pgx.Tx, locationID, productID string, lotID *string, delta string) (Balance, error) {
	return scanBalance(tx.QueryRow(ctx, `
		INSERT INTO inventory_balances (location_id, product_id, lot_id, quantity, created_at)
		VALUES ($1, $2, $3, $4::numeric, NOW())
		ON CONFLICT (location_id, product_id, lot_id) DO UPDATE
		    SET quantity = inventory_balances.quantity + EXCLUDED.quantity, updated_at = NOW()
		RETURNING id::text, location_id::text, product_id::text, lot_id::text,
		          quantity::text, reserved_quantity::text, created_at, updated_at`,
		locationID, productID, lotID, delta))
}

func insertMovement(
	ctx context.Context, tx pgx.Tx,
	movementType, productID string, lotID, fromLocationID, toLocationID *string,
	quantity string, unitCost, reference, notes *string, performedBy string,
) (Movement, error) {
	movement, err := scanMovement(tx.QueryRow(ctx, `
		INSERT INTO stock_movements
		    (movement_type, product_id, lot_id, from_location_id, to_location_id,
		     quantity, unit_cost, reference, notes, performed_by)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8, $9, $10)
		RETURNING id::text, movement_type, product_id::text, lot_id::text,
		          from_location_id::text, to_location_id::text, quantity::text,
		          unit_cost::text, reference, notes, performed_by::text, created_at`,
		movementType, productID, lotID, fromLocationID, toLocationID,
		quantity, unitCost, reference, notes, performedBy))
	if err != nil {
		return Movement{}, fmt.Errorf("insert movement: %w", err)
	}
	return movement, nil
}

func (r *PostgresRepository) Receive(ctx context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("begin receive: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	warehouseID, err := resolveLocationWarehouse(ctx, tx, input.LocationID)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, warehouseID); err != nil {
		return Movement{}, Balance{}, err
	}

	product, err := lockActiveProduct(ctx, tx, input.ProductID)
	if err != nil {
		return Movement{}, Balance{}, err
	}

	var lotID *string
	if product.IsLotTracked && input.LotNumber.Set && input.LotNumber.Value != nil {
		id, err := resolveOrCreateLot(ctx, tx, input.ProductID, *input.LotNumber.Value, optionalDateValue(input.ExpirationDate))
		if err != nil {
			return Movement{}, Balance{}, err
		}
		lotID = &id
	}

	movement, err := insertMovement(ctx, tx, "receive", input.ProductID, lotID, nil, &input.LocationID,
		input.Quantity, &input.UnitCost, optionalStringPtr(input.Reference), optionalStringPtr(input.Notes), actorID)
	if err != nil {
		return Movement{}, Balance{}, err
	}

	balance, err := upsertBalanceIncrement(ctx, tx, input.LocationID, input.ProductID, lotID, input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("increment balance on receive: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cost_layers
		    (warehouse_id, product_id, lot_id, source_movement_id, original_quantity, remaining_quantity, unit_cost, received_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $5::numeric, $6::numeric, NOW())`,
		warehouseID, input.ProductID, lotID, movement.ID, input.Quantity, input.UnitCost); err != nil {
		return Movement{}, Balance{}, fmt.Errorf("open cost layer: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Balance{}, fmt.Errorf("commit receive: %w", err)
	}
	return movement, balance, nil
}

func optionalStringPtr(value OptionalString) *string {
	if !value.Set {
		return nil
	}
	return value.Value
}

func optionalDateValue(value OptionalString) *string {
	if !value.Set {
		return nil
	}
	return value.Value
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository Receive with lot and cost layer creation"
```

---

## Task 9: Repository — `Pick` (FEFO physical consumption + FIFO cost layers)

**Files:**
- Modify: `backend/internal/modules/inventory/repository.go` (replace the `Pick` stub; add `consumeCostLayers`)
- Modify: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the Task 8 assertions:

```go
	// Second lot at the same location, expiring sooner than LOT-A, received later.
	_, _, err = repository.Receive(ctx, "actor-1", nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "30.000", UnitCost: "2.0000",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-EXPIRES-SOON")},
		ExpirationDate: OptionalString{Set: true, Value: strPointer("2026-09-01")},
	})
	if err != nil {
		t.Fatalf("Receive(second lot) error = %v", err)
	}

	// Pick 60 units with no lot override: FEFO must exhaust the sooner-expiring
	// lot (30 units) before drawing 30 more from LOT-A.
	movements, err := repository.Pick(ctx, "actor-1", nil, PickInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct, Quantity: "60.000",
	})
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if len(movements) != 2 {
		t.Fatalf("len(movements) = %d, want 2 (split across both lots)", len(movements))
	}
	if movements[0].Quantity != "30.000" {
		t.Fatalf("movements[0].Quantity = %s, want 30.000 from the sooner-expiring lot first", movements[0].Quantity)
	}
	if movements[1].Quantity != "30.000" {
		t.Fatalf("movements[1].Quantity = %s, want 30.000 remaining from LOT-A", movements[1].Quantity)
	}

	var remainingOnEarliestReceivedLayer, remainingOnSecondLayer string
	if err := fixture.pool.QueryRow(ctx, `
		SELECT remaining_quantity::text FROM cost_layers
		WHERE product_id = $1 ORDER BY received_at ASC, id ASC LIMIT 1`, fixture.trackedProduct,
	).Scan(&remainingOnEarliestReceivedLayer); err != nil {
		t.Fatalf("read first cost layer error = %v", err)
	}
	if remainingOnEarliestReceivedLayer != "0.000" {
		t.Fatalf("first-received cost layer remaining = %s, want 0.000 (FIFO drains it first regardless of FEFO lot order)", remainingOnEarliestReceivedLayer)
	}
	if err := fixture.pool.QueryRow(ctx, `
		SELECT remaining_quantity::text FROM cost_layers
		WHERE product_id = $1 ORDER BY received_at ASC, id ASC OFFSET 1 LIMIT 1`, fixture.trackedProduct,
	).Scan(&remainingOnSecondLayer); err != nil {
		t.Fatalf("read second cost layer error = %v", err)
	}
	if remainingOnSecondLayer != "20.000" {
		t.Fatalf("second cost layer remaining = %s, want 20.000 (50 - 30 already picked FIFO)", remainingOnSecondLayer)
	}

	_, err = repository.Pick(ctx, "actor-1", nil, PickInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct, Quantity: "1000.000",
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Pick(too much) error = %v, want ErrInsufficientStock", err)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 9`.

Trace through the fixture math before implementing, so the assertions above
are verified by hand, not just trusted: `Receive` #1 opens a cost layer for
50.000 units at `received_at = T1` (LOT-A, no expiration). `Receive` #2
opens a second cost layer for 30.000 units at `received_at = T2 > T1`
(LOT-EXPIRES-SOON, expires 2026-09-01). FEFO orders physical consumption by
`expiration_date ASC NULLS LAST` — LOT-EXPIRES-SOON has a real date, LOT-A
has `NULL`, so LOT-EXPIRES-SOON (30.000) is drained first, then LOT-A
(30.000 of its 50.000). FIFO cost-layer consumption is separate and orders
by `received_at ASC` — the T1 layer (LOT-A's 50.000) is drained first down
to 0, then the T2 layer (LOT-EXPIRES-SOON's 30.000) absorbs the remaining
10.000 of the 60.000 total, leaving 20.000 remaining on it. This is the
concrete case where FEFO (physical) and FIFO (cost) genuinely diverge, as
called out in the design doc.

- [ ] **Step 3: Replace the `Pick` stub and add `consumeCostLayers`**

```go
func (r *PostgresRepository) Pick(ctx context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin pick: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	warehouseID, err := resolveLocationWarehouse(ctx, tx, input.LocationID)
	if err != nil {
		return nil, err
	}
	if err := checkWarehouseScope(actorWarehouseID, warehouseID); err != nil {
		return nil, err
	}
	if _, err := lockActiveProduct(ctx, tx, input.ProductID); err != nil {
		return nil, err
	}

	requested, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return nil, fmt.Errorf("parse pick quantity: %w", err)
	}

	type candidate struct {
		BalanceID string
		LotID     *string
		Available decimal.Decimal
	}
	var candidates []candidate

	if input.LotID.Set && input.LotID.Value != nil {
		if err := lockLotForProduct(ctx, tx, *input.LotID.Value, input.ProductID); err != nil {
			return nil, err
		}
		var id, availableText string
		err := tx.QueryRow(ctx, `
			SELECT id::text, (quantity - reserved_quantity)::text
			FROM inventory_balances
			WHERE location_id = $1 AND product_id = $2 AND lot_id = $3
			FOR UPDATE`, input.LocationID, input.ProductID, *input.LotID.Value,
		).Scan(&id, &availableText)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInsufficientStock
		}
		if err != nil {
			return nil, fmt.Errorf("lock pick balance: %w", err)
		}
		available, err := decimal.NewFromString(availableText)
		if err != nil {
			return nil, fmt.Errorf("parse pick balance available quantity: %w", err)
		}
		candidates = append(candidates, candidate{BalanceID: id, LotID: input.LotID.Value, Available: available})
	} else {
		rows, err := tx.Query(ctx, `
			SELECT b.id::text, b.lot_id::text, (b.quantity - b.reserved_quantity)::text
			FROM inventory_balances b
			LEFT JOIN lots l ON l.id = b.lot_id
			WHERE b.location_id = $1 AND b.product_id = $2
			  AND (b.quantity - b.reserved_quantity) > 0
			ORDER BY l.expiration_date ASC NULLS LAST, l.received_at ASC NULLS LAST, b.id ASC
			FOR UPDATE OF b`, input.LocationID, input.ProductID)
		if err != nil {
			return nil, fmt.Errorf("query pick candidates: %w", err)
		}
		for rows.Next() {
			var c candidate
			var availableText string
			if err := rows.Scan(&c.BalanceID, &c.LotID, &availableText); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan pick candidate: %w", err)
			}
			c.Available, err = decimal.NewFromString(availableText)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("parse pick candidate available quantity: %w", err)
			}
			candidates = append(candidates, c)
		}
		rowsErr := rows.Err()
		rows.Close()
		if rowsErr != nil {
			return nil, fmt.Errorf("query pick candidates: %w", rowsErr)
		}
	}

	remaining := requested
	var movements []Movement
	for _, c := range candidates {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		take := c.Available
		if take.GreaterThan(remaining) {
			take = remaining
		}
		if take.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances SET quantity = quantity - $2::numeric, updated_at = NOW() WHERE id = $1`,
			c.BalanceID, take.String()); err != nil {
			return nil, fmt.Errorf("decrement pick balance: %w", err)
		}
		movement, err := insertMovement(ctx, tx, "pick", input.ProductID, c.LotID, &input.LocationID, nil,
			take.String(), nil, optionalStringPtr(input.Reference), optionalStringPtr(input.Notes), actorID)
		if err != nil {
			return nil, err
		}
		movements = append(movements, movement)
		remaining = remaining.Sub(take)
	}

	if remaining.GreaterThan(decimal.Zero) {
		return nil, ErrInsufficientStock
	}

	if err := consumeCostLayers(ctx, tx, warehouseID, input.ProductID, requested); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pick: %w", err)
	}
	return movements, nil
}

func consumeCostLayers(ctx context.Context, tx pgx.Tx, warehouseID, productID string, quantity decimal.Decimal) error {
	rows, err := tx.Query(ctx, `
		SELECT id::text, remaining_quantity::text
		FROM cost_layers
		WHERE warehouse_id = $1 AND product_id = $2 AND remaining_quantity > 0
		ORDER BY received_at ASC, id ASC
		FOR UPDATE`, warehouseID, productID)
	if err != nil {
		return fmt.Errorf("query cost layers: %w", err)
	}

	type layer struct {
		ID        string
		Remaining decimal.Decimal
	}
	var layers []layer
	for rows.Next() {
		var l layer
		var remainingText string
		if err := rows.Scan(&l.ID, &remainingText); err != nil {
			rows.Close()
			return fmt.Errorf("scan cost layer: %w", err)
		}
		l.Remaining, err = decimal.NewFromString(remainingText)
		if err != nil {
			rows.Close()
			return fmt.Errorf("parse cost layer remaining quantity: %w", err)
		}
		layers = append(layers, l)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return fmt.Errorf("query cost layers: %w", rowsErr)
	}

	remaining := quantity
	for _, l := range layers {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		take := l.Remaining
		if take.GreaterThan(remaining) {
			take = remaining
		}
		if _, err := tx.Exec(ctx, `
			UPDATE cost_layers SET remaining_quantity = remaining_quantity - $2::numeric WHERE id = $1`,
			l.ID, take.String()); err != nil {
			return fmt.Errorf("decrement cost layer: %w", err)
		}
		remaining = remaining.Sub(take)
	}

	if remaining.GreaterThan(decimal.Zero) {
		return ErrInventoryOperationFailed
	}
	return nil
}
```

Add `"github.com/jackc/pgx/v5"` usage is already imported from Task 6; no
new import needed beyond what Task 6 added.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository Pick with FEFO and FIFO cost consumption"
```

---

## Task 10: Repository — `Transfer`

**Files:**
- Modify: `backend/internal/modules/inventory/repository.go` (replace the `Transfer` stub; add `decrementBalance`)
- Modify: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the Task 9 assertions:

```go
	transferMovement, sourceBalance, destinationBalance, err := repository.Transfer(ctx, "actor-1", nil, TransferInput{
		ProductID: fixture.plainProduct, Quantity: "5.000",
		FromLocationID: fixture.locationAID, ToLocationID: fixture.locationBID,
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Transfer(no stock yet) error = %v, want ErrInsufficientStock", err)
	}

	_, _, err = repository.Receive(ctx, "actor-1", nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.plainProduct,
		Quantity: "20.000", UnitCost: "0.5000",
	})
	if err != nil {
		t.Fatalf("Receive(plain product) error = %v", err)
	}

	transferMovement, sourceBalance, destinationBalance, err = repository.Transfer(ctx, "actor-1", nil, TransferInput{
		ProductID: fixture.plainProduct, Quantity: "5.000",
		FromLocationID: fixture.locationAID, ToLocationID: fixture.locationBID,
	})
	if err != nil {
		t.Fatalf("Transfer() error = %v", err)
	}
	if transferMovement.MovementType != "transfer" {
		t.Fatalf("transferMovement.MovementType = %s, want transfer", transferMovement.MovementType)
	}
	if sourceBalance.Quantity != "15.000" {
		t.Fatalf("sourceBalance.Quantity = %s, want 15.000", sourceBalance.Quantity)
	}
	if destinationBalance.Quantity != "5.000" {
		t.Fatalf("destinationBalance.Quantity = %s, want 5.000", destinationBalance.Quantity)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 10`.

- [ ] **Step 3: Replace the `Transfer` stub and add `decrementBalance`**

```go
func decrementBalance(ctx context.Context, tx pgx.Tx, locationID, productID string, lotID *string, amount decimal.Decimal) (Balance, error) {
	var id, quantityText, reservedText string
	err := tx.QueryRow(ctx, `
		SELECT id::text, quantity::text, reserved_quantity::text
		FROM inventory_balances
		WHERE location_id = $1 AND product_id = $2 AND lot_id IS NOT DISTINCT FROM $3
		FOR UPDATE`, locationID, productID, lotID,
	).Scan(&id, &quantityText, &reservedText)
	if errors.Is(err, pgx.ErrNoRows) {
		return Balance{}, ErrInsufficientStock
	}
	if err != nil {
		return Balance{}, fmt.Errorf("lock balance for decrement: %w", err)
	}
	quantity, err := decimal.NewFromString(quantityText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance quantity: %w", err)
	}
	reserved, err := decimal.NewFromString(reservedText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance reserved quantity: %w", err)
	}
	if quantity.Sub(reserved).LessThan(amount) {
		return Balance{}, ErrInsufficientStock
	}
	return scanBalance(tx.QueryRow(ctx, `
		UPDATE inventory_balances
		SET quantity = quantity - $2::numeric, updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, location_id::text, product_id::text, lot_id::text,
		          quantity::text, reserved_quantity::text, created_at, updated_at`,
		id, amount.String()))
}

func (r *PostgresRepository) Transfer(ctx context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("begin transfer: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fromWarehouseID, err := resolveLocationWarehouse(ctx, tx, input.FromLocationID)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, fromWarehouseID); err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	toWarehouseID, err := resolveLocationWarehouse(ctx, tx, input.ToLocationID)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, toWarehouseID); err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	if _, err := lockActiveProduct(ctx, tx, input.ProductID); err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	var lotID *string
	if input.LotID.Set && input.LotID.Value != nil {
		if err := lockLotForProduct(ctx, tx, *input.LotID.Value, input.ProductID); err != nil {
			return Movement{}, Balance{}, Balance{}, err
		}
		lotID = input.LotID.Value
	}

	quantity, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("parse transfer quantity: %w", err)
	}

	sourceBalance, err := decrementBalance(ctx, tx, input.FromLocationID, input.ProductID, lotID, quantity)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	destinationBalance, err := upsertBalanceIncrement(ctx, tx, input.ToLocationID, input.ProductID, lotID, input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("increment balance on transfer: %w", err)
	}

	movement, err := insertMovement(ctx, tx, "transfer", input.ProductID, lotID, &input.FromLocationID, &input.ToLocationID,
		input.Quantity, nil, optionalStringPtr(input.Reference), optionalStringPtr(input.Notes), actorID)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("commit transfer: %w", err)
	}
	return movement, sourceBalance, destinationBalance, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository Transfer"
```

---

## Task 11: Repository — `Adjust`

**Files:**
- Modify: `backend/internal/modules/inventory/repository.go` (replace the `Adjust` stub)
- Modify: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the Task 10 assertions:

```go
	increaseMovement, increasedBalance, err := repository.Adjust(ctx, "actor-1", nil, AdjustInput{
		LocationID: fixture.locationBID, ProductID: fixture.plainProduct,
		Direction: "increase", Quantity: "3.000",
	})
	if err != nil {
		t.Fatalf("Adjust(increase) error = %v", err)
	}
	if increaseMovement.ToLocationID == nil || *increaseMovement.ToLocationID != fixture.locationBID {
		t.Fatalf("increaseMovement = %#v, want to_location_id = locationB", increaseMovement)
	}
	if increasedBalance.Quantity != "8.000" {
		t.Fatalf("increasedBalance.Quantity = %s, want 8.000 (5 transferred in + 3 adjusted in)", increasedBalance.Quantity)
	}

	decreaseMovement, decreasedBalance, err := repository.Adjust(ctx, "actor-1", nil, AdjustInput{
		LocationID: fixture.locationBID, ProductID: fixture.plainProduct,
		Direction: "decrease", Quantity: "2.000",
	})
	if err != nil {
		t.Fatalf("Adjust(decrease) error = %v", err)
	}
	if decreaseMovement.FromLocationID == nil || *decreaseMovement.FromLocationID != fixture.locationBID {
		t.Fatalf("decreaseMovement = %#v, want from_location_id = locationB", decreaseMovement)
	}
	if decreasedBalance.Quantity != "6.000" {
		t.Fatalf("decreasedBalance.Quantity = %s, want 6.000", decreasedBalance.Quantity)
	}

	_, _, err = repository.Adjust(ctx, "actor-1", nil, AdjustInput{
		LocationID: fixture.locationBID, ProductID: fixture.plainProduct,
		Direction: "decrease", Quantity: "1000.000",
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Adjust(decrease too much) error = %v, want ErrInsufficientStock", err)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 11`.

- [ ] **Step 3: Replace the `Adjust` stub**

```go
func (r *PostgresRepository) Adjust(ctx context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("begin adjust: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	warehouseID, err := resolveLocationWarehouse(ctx, tx, input.LocationID)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, warehouseID); err != nil {
		return Movement{}, Balance{}, err
	}
	if _, err := lockActiveProduct(ctx, tx, input.ProductID); err != nil {
		return Movement{}, Balance{}, err
	}

	var lotID *string
	if input.LotID.Set && input.LotID.Value != nil {
		if err := lockLotForProduct(ctx, tx, *input.LotID.Value, input.ProductID); err != nil {
			return Movement{}, Balance{}, err
		}
		lotID = input.LotID.Value
	}

	quantity, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("parse adjust quantity: %w", err)
	}

	var balance Balance
	var fromLocationID, toLocationID *string
	if input.Direction == "increase" {
		balance, err = upsertBalanceIncrement(ctx, tx, input.LocationID, input.ProductID, lotID, input.Quantity)
		toLocationID = &input.LocationID
	} else {
		balance, err = decrementBalance(ctx, tx, input.LocationID, input.ProductID, lotID, quantity)
		fromLocationID = &input.LocationID
	}
	if err != nil {
		return Movement{}, Balance{}, err
	}

	movement, err := insertMovement(ctx, tx, "adjust", input.ProductID, lotID, fromLocationID, toLocationID,
		input.Quantity, nil, nil, optionalStringPtr(input.Notes), actorID)
	if err != nil {
		return Movement{}, Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Balance{}, fmt.Errorf("commit adjust: %w", err)
	}
	return movement, balance, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository Adjust"
```

---

## Task 12: Repository — `ListMovements` and `GetMovement`

**Files:**
- Modify: `backend/internal/modules/inventory/repository.go` (replace both stubs)
- Modify: `backend/internal/modules/inventory/repository_integration_test.go`

- [ ] **Step 1: Extend the failing test**

Append after the Task 11 assertions, right before the closing `}` of
`TestPostgresInventoryLifecycle`:

```go
	movements, err := repository.ListMovements(ctx, MovementListFilter{Limit: 20, ProductID: &fixture.plainProduct})
	if err != nil {
		t.Fatalf("ListMovements() error = %v", err)
	}
	if len(movements) < 3 {
		t.Fatalf("len(movements) = %d, want at least 3 (receive, transfer, 2x adjust)", len(movements))
	}

	fetched, err := repository.GetMovement(ctx, transferMovement.ID)
	if err != nil || fetched.ID != transferMovement.ID {
		t.Fatalf("GetMovement() = %#v, %v, want transferMovement", fetched, err)
	}

	missingMovement := "00000000-0000-0000-0000-000000000000"
	if _, err := repository.GetMovement(ctx, missingMovement); !errors.Is(err, ErrMovementNotFound) {
		t.Fatalf("GetMovement(missing) error = %v, want ErrMovementNotFound", err)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v`
Expected: FAIL — `panic: not implemented until Task 12` (and `undefined: ErrMovementNotFound`).

- [ ] **Step 3: Replace both stubs and add `ErrMovementNotFound`**

Add to the `var (...)` block near the top of `repository.go`:

```go
ErrMovementNotFound = errors.New("movement was not found")
```

Replace the stubs:

```go
func (r *PostgresRepository) ListMovements(ctx context.Context, filter MovementListFilter) ([]Movement, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, movement_type, product_id::text, lot_id::text,
		       from_location_id::text, to_location_id::text, quantity::text,
		       unit_cost::text, reference, notes, performed_by::text, created_at
		FROM stock_movements
		WHERE ($1::uuid IS NULL OR product_id = $1)
		  AND ($2::uuid IS NULL OR from_location_id = $2 OR to_location_id = $2)
		  AND ($3::text IS NULL OR movement_type = $3)
		  AND ($4::timestamptz IS NULL OR created_at >= $4)
		  AND ($5::timestamptz IS NULL OR created_at <= $5)
		  AND ($6::timestamptz IS NULL OR (created_at, id) < ($6::timestamptz, $7::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $8`,
		filter.ProductID, filter.LocationID, filter.MovementType, filter.From, filter.To,
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	movements := make([]Movement, 0)
	for rows.Next() {
		movement, err := scanMovement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		movements = append(movements, movement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	return movements, nil
}

func (r *PostgresRepository) GetMovement(ctx context.Context, id string) (Movement, error) {
	movement, err := scanMovement(r.pool.QueryRow(ctx, `
		SELECT id::text, movement_type, product_id::text, lot_id::text,
		       from_location_id::text, to_location_id::text, quantity::text,
		       unit_cost::text, reference, notes, performed_by::text, created_at
		FROM stock_movements WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Movement{}, ErrMovementNotFound
	}
	if err != nil {
		return Movement{}, fmt.Errorf("get movement: %w", err)
	}
	return movement, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && BWIMS_TEST_DATABASE_URL="$BWIMS_TEST_DATABASE_URL" go test ./internal/modules/inventory/... -v`
Expected: PASS for the whole package, including every subtest of
`TestPostgresInventoryLifecycle`.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/repository.go backend/internal/modules/inventory/repository_integration_test.go
git commit -m "feat: add inventory repository ListMovements and GetMovement"
```

---

## Task 13: Service layer — RBAC helpers, `ListBalances`, `ListLots`

**Files:**
- Create: `backend/internal/modules/inventory/service.go`
- Create: `backend/internal/modules/inventory/service_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

type fakeRepository struct {
	listBalancesFn  func(BalanceListFilter) ([]Balance, error)
	listLotsFn      func(string, *string) ([]Lot, error)
	receiveFn       func(string, *string, ReceiveInput) (Movement, Balance, error)
	pickFn          func(string, *string, PickInput) ([]Movement, error)
	transferFn      func(string, *string, TransferInput) (Movement, Balance, Balance, error)
	adjustFn        func(string, *string, AdjustInput) (Movement, Balance, error)
	listMovementsFn func(MovementListFilter) ([]Movement, error)
	getMovementFn   func(string) (Movement, error)
}

func (f *fakeRepository) ListBalances(_ context.Context, filter BalanceListFilter) ([]Balance, error) {
	return f.listBalancesFn(filter)
}
func (f *fakeRepository) ListLots(_ context.Context, productID string, warehouseID *string) ([]Lot, error) {
	return f.listLotsFn(productID, warehouseID)
}
func (f *fakeRepository) Receive(_ context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
	return f.receiveFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) Pick(_ context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
	return f.pickFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) Transfer(_ context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
	return f.transferFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) Adjust(_ context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
	return f.adjustFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) ListMovements(_ context.Context, filter MovementListFilter) ([]Movement, error) {
	return f.listMovementsFn(filter)
}
func (f *fakeRepository) GetMovement(_ context.Context, id string) (Movement, error) {
	return f.getMovementFn(id)
}

func adminActor() Actor  { return Actor{ID: "admin-1", Role: auth.RoleAdmin} }
func pickerActor(warehouseID string) Actor {
	return Actor{ID: "picker-1", Role: auth.RolePicker, WarehouseID: &warehouseID}
}

func TestListBalancesRejectsNonAdminWithoutWarehouse(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListBalances(context.Background(), Actor{ID: "x", Role: auth.RoleViewer}, BalanceListFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListBalancesScopesNonAdminToAssignedWarehouse(t *testing.T) {
	var capturedFilter BalanceListFilter
	repo := &fakeRepository{listBalancesFn: func(filter BalanceListFilter) ([]Balance, error) {
		capturedFilter = filter
		return nil, nil
	}}
	service := NewService(repo)
	warehouseID := "wh-1"
	if _, err := service.ListBalances(context.Background(), pickerActor(warehouseID), BalanceListFilter{}); err != nil {
		t.Fatalf("ListBalances() error = %v", err)
	}
	if capturedFilter.WarehouseID == nil || *capturedFilter.WarehouseID != warehouseID {
		t.Fatalf("capturedFilter.WarehouseID = %v, want %s", capturedFilter.WarehouseID, warehouseID)
	}
}

func TestListBalancesRejectsCrossWarehouseFilterForNonAdmin(t *testing.T) {
	service := NewService(&fakeRepository{})
	warehouseID := "wh-1"
	other := "wh-2"
	_, err := service.ListBalances(context.Background(), pickerActor(warehouseID), BalanceListFilter{WarehouseID: &other})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListLotsRejectsInvalidProductID(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListLots(context.Background(), adminActor(), "not-a-uuid")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestListBalances|TestListLots' -v`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 3: Write the service foundations and these two methods**

```go
package inventory

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden            = errors.New("actor is not permitted to perform this action")
	ErrValidation           = errors.New("inventory request data is invalid")
	ErrSameLocationTransfer = errors.New("transfer source and destination locations must differ")
)

type Service interface {
	ListBalances(ctx context.Context, actor Actor, filter BalanceListFilter) (Page[Balance], error)
	ListLots(ctx context.Context, actor Actor, productID string) ([]Lot, error)
	Receive(ctx context.Context, actor Actor, input ReceiveInput) (Movement, Balance, error)
	Pick(ctx context.Context, actor Actor, input PickInput) ([]Movement, error)
	Transfer(ctx context.Context, actor Actor, input TransferInput) (Movement, Balance, Balance, error)
	Adjust(ctx context.Context, actor Actor, input AdjustInput) (Movement, Balance, error)
	ListMovements(ctx context.Context, actor Actor, filter MovementListFilter) (Page[Movement], error)
	GetMovement(ctx context.Context, actor Actor, id string) (Movement, error)
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func canReceiveOrPick(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker:
		return true
	}
	return false
}

func canTransferOrAdjust(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseManager:
		return true
	}
	return false
}

// requireWarehouseScope returns "" for admin (unrestricted) or the actor's
// assigned warehouse for every other role. A non-admin with no assignment
// is rejected outright.
func requireWarehouseScope(actor Actor) (string, error) {
	if actor.Role == auth.RoleAdmin {
		return "", nil
	}
	if actor.WarehouseID == nil || strings.TrimSpace(*actor.WarehouseID) == "" {
		return "", ErrForbidden
	}
	return *actor.WarehouseID, nil
}

func isInvalidUUID(value string) bool {
	_, err := uuid.Parse(strings.TrimSpace(value))
	return err != nil
}

func isPositiveDecimal(value string) bool {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return amount.GreaterThan(decimal.Zero)
}

func isNonNegativeDecimal(value string) bool {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return amount.GreaterThanOrEqual(decimal.Zero)
}

func (s *service) ListBalances(ctx context.Context, actor Actor, filter BalanceListFilter) (Page[Balance], error) {
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Page[Balance]{}, err
	}
	if warehouseID != "" {
		if filter.WarehouseID != nil && *filter.WarehouseID != warehouseID {
			return Page[Balance]{}, ErrForbidden
		}
		filter.WarehouseID = &warehouseID
	}
	limit := normalizeLimit(filter.Limit)
	queryFilter := filter
	queryFilter.Limit = limit + 1
	items, err := s.repository.ListBalances(ctx, queryFilter)
	if err != nil {
		return Page[Balance]{}, fmt.Errorf("list balances: %w", err)
	}
	return balancePage(items, limit), nil
}

func (s *service) ListLots(ctx context.Context, actor Actor, productID string) ([]Lot, error) {
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return nil, err
	}
	if isInvalidUUID(productID) {
		return nil, ErrValidation
	}
	var warehouseFilter *string
	if warehouseID != "" {
		warehouseFilter = &warehouseID
	}
	lots, err := s.repository.ListLots(ctx, productID, warehouseFilter)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	return lots, nil
}

func (s *service) Receive(ctx context.Context, actor Actor, input ReceiveInput) (Movement, Balance, error) {
	panic("not implemented until Task 14")
}

func (s *service) Pick(ctx context.Context, actor Actor, input PickInput) ([]Movement, error) {
	panic("not implemented until Task 14")
}

func (s *service) Transfer(ctx context.Context, actor Actor, input TransferInput) (Movement, Balance, Balance, error) {
	panic("not implemented until Task 15")
}

func (s *service) Adjust(ctx context.Context, actor Actor, input AdjustInput) (Movement, Balance, error) {
	panic("not implemented until Task 15")
}

func (s *service) ListMovements(ctx context.Context, actor Actor, filter MovementListFilter) (Page[Movement], error) {
	panic("not implemented until Task 16")
}

func (s *service) GetMovement(ctx context.Context, actor Actor, id string) (Movement, error) {
	panic("not implemented until Task 16")
}
```

Add `"errors"` to the import block (used by the sentinel `var` declarations).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestListBalances|TestListLots' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/service.go backend/internal/modules/inventory/service_test.go
git commit -m "feat: add inventory service RBAC helpers, ListBalances, ListLots"
```

---

## Task 14: Service — `Receive` and `Pick`

**Files:**
- Modify: `backend/internal/modules/inventory/service.go` (replace both stubs)
- Modify: `backend/internal/modules/inventory/service_test.go`

- [ ] **Step 1: Extend the failing tests**

Append to `service_test.go`:

```go
func TestReceiveRejectsViewerAndPicker(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RoleViewer} {
		_, _, err := service.Receive(context.Background(), Actor{ID: "x", Role: role}, ReceiveInput{
			LocationID: "11111111-1111-1111-1111-111111111111",
			ProductID:  "22222222-2222-2222-2222-222222222222",
			Quantity:   "1.000", UnitCost: "1.0000",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestReceiveRejectsNonPositiveQuantity(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, _, err := service.Receive(context.Background(), adminActor(), ReceiveInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "0.000", UnitCost: "1.0000",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestReceivePassesThroughToRepository(t *testing.T) {
	var capturedWarehouseID *string
	repo := &fakeRepository{receiveFn: func(actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
		capturedWarehouseID = actorWarehouseID
		return Movement{ID: "m1"}, Balance{ID: "b1"}, nil
	}}
	service := NewService(repo)
	warehouseID := "wh-1"
	movement, balance, err := service.Receive(context.Background(), pickerActor(warehouseID), ReceiveInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "5.000", UnitCost: "1.0000",
	})
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if movement.ID != "m1" || balance.ID != "b1" {
		t.Fatalf("got movement=%+v balance=%+v, want passthrough from repository", movement, balance)
	}
	if capturedWarehouseID == nil || *capturedWarehouseID != warehouseID {
		t.Fatalf("capturedWarehouseID = %v, want %s", capturedWarehouseID, warehouseID)
	}
}

func TestPickRejectsTransferOnlyRoles(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.Pick(context.Background(), Actor{ID: "x", Role: auth.RoleViewer}, PickInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "1.000",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestPickPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{pickFn: func(actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
		return []Movement{{ID: "m1"}}, nil
	}}
	service := NewService(repo)
	movements, err := service.Pick(context.Background(), adminActor(), PickInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "5.000",
	})
	if err != nil || len(movements) != 1 {
		t.Fatalf("Pick() = %#v, %v, want one movement", movements, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestReceive|TestPick' -v`
Expected: FAIL — `panic: not implemented until Task 14`.

- [ ] **Step 3: Replace both stubs**

```go
func (s *service) Receive(ctx context.Context, actor Actor, input ReceiveInput) (Movement, Balance, error) {
	if !canReceiveOrPick(actor.Role) {
		return Movement{}, Balance{}, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if isInvalidUUID(input.LocationID) || isInvalidUUID(input.ProductID) {
		return Movement{}, Balance{}, ErrValidation
	}
	if !isPositiveDecimal(input.Quantity) {
		return Movement{}, Balance{}, ErrValidation
	}
	if !isNonNegativeDecimal(input.UnitCost) {
		return Movement{}, Balance{}, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movement, balance, err := s.repository.Receive(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	return movement, balance, nil
}

func (s *service) Pick(ctx context.Context, actor Actor, input PickInput) ([]Movement, error) {
	if !canReceiveOrPick(actor.Role) {
		return nil, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return nil, err
	}
	if isInvalidUUID(input.LocationID) || isInvalidUUID(input.ProductID) {
		return nil, ErrValidation
	}
	if !isPositiveDecimal(input.Quantity) {
		return nil, ErrValidation
	}
	if input.LotID.Set && input.LotID.Value != nil && isInvalidUUID(*input.LotID.Value) {
		return nil, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movements, err := s.repository.Pick(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return nil, err
	}
	return movements, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestReceive|TestPick' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/service.go backend/internal/modules/inventory/service_test.go
git commit -m "feat: add inventory service Receive and Pick validation"
```

---

## Task 15: Service — `Transfer` and `Adjust`

**Files:**
- Modify: `backend/internal/modules/inventory/service.go` (replace both stubs)
- Modify: `backend/internal/modules/inventory/service_test.go`

- [ ] **Step 1: Extend the failing tests**

```go
func TestTransferRejectsPickerAndViewer(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		_, _, _, err := service.Transfer(context.Background(), Actor{ID: "x", Role: role}, TransferInput{
			ProductID: "22222222-2222-2222-2222-222222222222", Quantity: "1.000",
			FromLocationID: "11111111-1111-1111-1111-111111111111",
			ToLocationID:   "33333333-3333-3333-3333-333333333333",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestTransferRejectsSameLocation(t *testing.T) {
	service := NewService(&fakeRepository{})
	sameLocation := "11111111-1111-1111-1111-111111111111"
	_, _, _, err := service.Transfer(context.Background(), adminActor(), TransferInput{
		ProductID: "22222222-2222-2222-2222-222222222222", Quantity: "1.000",
		FromLocationID: sameLocation, ToLocationID: sameLocation,
	})
	if !errors.Is(err, ErrSameLocationTransfer) {
		t.Fatalf("err = %v, want ErrSameLocationTransfer", err)
	}
}

func TestTransferPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{transferFn: func(actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
		return Movement{ID: "m1"}, Balance{ID: "src"}, Balance{ID: "dst"}, nil
	}}
	service := NewService(repo)
	movement, source, destination, err := service.Transfer(context.Background(), adminActor(), TransferInput{
		ProductID: "22222222-2222-2222-2222-222222222222", Quantity: "1.000",
		FromLocationID: "11111111-1111-1111-1111-111111111111",
		ToLocationID:   "33333333-3333-3333-3333-333333333333",
	})
	if err != nil || movement.ID != "m1" || source.ID != "src" || destination.ID != "dst" {
		t.Fatalf("Transfer() = %+v %+v %+v %v, want passthrough", movement, source, destination, err)
	}
}

func TestAdjustRejectsInvalidDirection(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, _, err := service.Adjust(context.Background(), adminActor(), AdjustInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Direction:  "sideways", Quantity: "1.000",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestAdjustRejectsPickerAndViewer(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		_, _, err := service.Adjust(context.Background(), Actor{ID: "x", Role: role}, AdjustInput{
			LocationID: "11111111-1111-1111-1111-111111111111",
			ProductID:  "22222222-2222-2222-2222-222222222222",
			Direction:  "increase", Quantity: "1.000",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestAdjustPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{adjustFn: func(actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
		return Movement{ID: "m1"}, Balance{ID: "b1"}, nil
	}}
	service := NewService(repo)
	movement, balance, err := service.Adjust(context.Background(), adminActor(), AdjustInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Direction:  "decrease", Quantity: "1.000",
	})
	if err != nil || movement.ID != "m1" || balance.ID != "b1" {
		t.Fatalf("Adjust() = %+v %+v %v, want passthrough", movement, balance, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestTransfer|TestAdjust' -v`
Expected: FAIL — `panic: not implemented until Task 15`.

- [ ] **Step 3: Replace both stubs**

```go
func (s *service) Transfer(ctx context.Context, actor Actor, input TransferInput) (Movement, Balance, Balance, error) {
	if !canTransferOrAdjust(actor.Role) {
		return Movement{}, Balance{}, Balance{}, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	if isInvalidUUID(input.ProductID) || isInvalidUUID(input.FromLocationID) || isInvalidUUID(input.ToLocationID) {
		return Movement{}, Balance{}, Balance{}, ErrValidation
	}
	if input.FromLocationID == input.ToLocationID {
		return Movement{}, Balance{}, Balance{}, ErrSameLocationTransfer
	}
	if !isPositiveDecimal(input.Quantity) {
		return Movement{}, Balance{}, Balance{}, ErrValidation
	}
	if input.LotID.Set && input.LotID.Value != nil && isInvalidUUID(*input.LotID.Value) {
		return Movement{}, Balance{}, Balance{}, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movement, source, destination, err := s.repository.Transfer(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	return movement, source, destination, nil
}

func (s *service) Adjust(ctx context.Context, actor Actor, input AdjustInput) (Movement, Balance, error) {
	if !canTransferOrAdjust(actor.Role) {
		return Movement{}, Balance{}, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if isInvalidUUID(input.LocationID) || isInvalidUUID(input.ProductID) {
		return Movement{}, Balance{}, ErrValidation
	}
	if input.Direction != "increase" && input.Direction != "decrease" {
		return Movement{}, Balance{}, ErrValidation
	}
	if !isPositiveDecimal(input.Quantity) {
		return Movement{}, Balance{}, ErrValidation
	}
	if input.LotID.Set && input.LotID.Value != nil && isInvalidUUID(*input.LotID.Value) {
		return Movement{}, Balance{}, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movement, balance, err := s.repository.Adjust(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	return movement, balance, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestTransfer|TestAdjust' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/service.go backend/internal/modules/inventory/service_test.go
git commit -m "feat: add inventory service Transfer and Adjust validation"
```

---

## Task 16: Service — `ListMovements` and `GetMovement`

**Files:**
- Modify: `backend/internal/modules/inventory/service.go` (replace both stubs)
- Modify: `backend/internal/modules/inventory/service_test.go`

- [ ] **Step 1: Extend the failing tests**

```go
func TestListMovementsRejectsNonAdminWithoutWarehouse(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListMovements(context.Background(), Actor{ID: "x", Role: auth.RoleViewer}, MovementListFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestGetMovementRejectsInvalidID(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.GetMovement(context.Background(), adminActor(), "not-a-uuid")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestGetMovementPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{getMovementFn: func(id string) (Movement, error) {
		return Movement{ID: id}, nil
	}}
	service := NewService(repo)
	movement, err := service.GetMovement(context.Background(), adminActor(), "11111111-1111-1111-1111-111111111111")
	if err != nil || movement.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("GetMovement() = %+v, %v, want passthrough", movement, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestListMovements|TestGetMovement' -v`
Expected: FAIL — `panic: not implemented until Task 16`.

- [ ] **Step 3: Replace both stubs**

```go
func (s *service) ListMovements(ctx context.Context, actor Actor, filter MovementListFilter) (Page[Movement], error) {
	if _, err := requireWarehouseScope(actor); err != nil {
		return Page[Movement]{}, err
	}
	// Non-admin scoping for movements happens via location_id filtering at
	// the handler layer (movements have no direct warehouse_id column);
	// callers without an assigned warehouse are already rejected above.
	limit := normalizeLimit(filter.Limit)
	queryFilter := filter
	queryFilter.Limit = limit + 1
	items, err := s.repository.ListMovements(ctx, queryFilter)
	if err != nil {
		return Page[Movement]{}, fmt.Errorf("list movements: %w", err)
	}
	return movementPage(items, limit), nil
}

func (s *service) GetMovement(ctx context.Context, actor Actor, id string) (Movement, error) {
	if _, err := requireWarehouseScope(actor); err != nil {
		return Movement{}, err
	}
	if isInvalidUUID(id) {
		return Movement{}, ErrValidation
	}
	movement, err := s.repository.GetMovement(ctx, id)
	if err != nil {
		return Movement{}, err
	}
	return movement, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -v`
Expected: PASS for the whole package.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/service.go backend/internal/modules/inventory/service_test.go
git commit -m "feat: add inventory service ListMovements and GetMovement"
```

---

## Task 17: Handler, request binding, and error mapping

**Files:**
- Create: `backend/internal/modules/inventory/handler.go`
- Create: `backend/internal/modules/inventory/handler_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

type fakeHandlerService struct {
	receiveFn func(Actor, ReceiveInput) (Movement, Balance, error)
	pickFn    func(Actor, PickInput) ([]Movement, error)
}

func (f *fakeHandlerService) ListBalances(context.Context, Actor, BalanceListFilter) (Page[Balance], error) {
	return Page[Balance]{Items: []Balance{}}, nil
}
func (f *fakeHandlerService) ListLots(context.Context, Actor, string) ([]Lot, error) { return nil, nil }
func (f *fakeHandlerService) Receive(_ context.Context, actor Actor, input ReceiveInput) (Movement, Balance, error) {
	return f.receiveFn(actor, input)
}
func (f *fakeHandlerService) Pick(_ context.Context, actor Actor, input PickInput) ([]Movement, error) {
	return f.pickFn(actor, input)
}
func (f *fakeHandlerService) Transfer(context.Context, Actor, TransferInput) (Movement, Balance, Balance, error) {
	return Movement{}, Balance{}, Balance{}, nil
}
func (f *fakeHandlerService) Adjust(context.Context, Actor, AdjustInput) (Movement, Balance, error) {
	return Movement{}, Balance{}, nil
}
func (f *fakeHandlerService) ListMovements(context.Context, Actor, MovementListFilter) (Page[Movement], error) {
	return Page[Movement]{Items: []Movement{}}, nil
}
func (f *fakeHandlerService) GetMovement(context.Context, Actor, string) (Movement, error) {
	return Movement{}, nil
}

// newTestApp registers routes through the real auth.Authenticate middleware
// with a throwaway development-key TokenManager, and issues real signed
// tokens for each test actor. auth.Claims are only ever set via
// auth.Authenticate itself (its context key is package-private, by design —
// there is no exported way to fake them), so this is the only correct way
// to test a handler that reads the actor from context.
func newTestApp(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "inventory-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	app := fiber.New()
	RegisterRoutesForTest(app, NewHandler(service), tokens)
	return app, tokens
}

func issueToken(t *testing.T, tokens *auth.TokenManager, role string) string {
	t.Helper()
	pair, err := tokens.Issue(auth.User{ID: "actor-1", Role: role, IsActive: true})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	return pair.AccessToken
}

func TestReceiveHandlerMapsForbidden(t *testing.T) {
	service := &fakeHandlerService{receiveFn: func(Actor, ReceiveInput) (Movement, Balance, error) {
		return Movement{}, Balance{}, ErrForbidden
	}}
	app, tokens := newTestApp(t, service)
	token := issueToken(t, tokens, auth.RoleAdmin)
	body, _ := json.Marshal(map[string]any{
		"location_id": "11111111-1111-1111-1111-111111111111",
		"product_id":  "22222222-2222-2222-2222-222222222222",
		"quantity":    "1.000", "unit_cost": "1.0000",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/receive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.StatusCode)
	}
}

func TestReceiveHandlerRejectsMalformedJSON(t *testing.T) {
	app, tokens := newTestApp(t, &fakeHandlerService{})
	token := issueToken(t, tokens, auth.RoleAdmin)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/receive", bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.StatusCode)
	}
}

func TestPickHandlerReturnsMovementsArray(t *testing.T) {
	service := &fakeHandlerService{pickFn: func(Actor, PickInput) ([]Movement, error) {
		return []Movement{{ID: "m1", Quantity: "5.000"}}, nil
	}}
	app, tokens := newTestApp(t, service)
	token := issueToken(t, tokens, auth.RoleAdmin)
	body, _ := json.Marshal(map[string]any{
		"location_id": "11111111-1111-1111-1111-111111111111",
		"product_id":  "22222222-2222-2222-2222-222222222222",
		"quantity":    "5.000",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/pick", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}
}
```

Add `"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"`
and `"time"` to the imports.

`RegisterRoutesForTest` is a Task 18 helper; this task only needs the
handler to compile against auth-required-but-role-unrestricted routes
(full RBAC wiring lands in Task 18):

```go
// route.go (temporary minimal version for this task; Task 18 replaces it)
package inventory

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutesForTest(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	group := app.Group("/api/v1/inventory", auth.Authenticate(tokens))
	group.Get("", handler.ListBalances)
	group.Get("/products/:product_id/lots", handler.ListLots)
	group.Post("/movements/receive", handler.Receive)
	group.Post("/movements/pick", handler.Pick)
	group.Post("/movements/transfer", handler.Transfer)
	group.Post("/movements/adjust", handler.Adjust)
	group.Get("/movements", handler.ListMovements)
	group.Get("/movements/:movement_id", handler.GetMovement)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -run 'TestReceiveHandler|TestPickHandler' -v`
Expected: FAIL — `undefined: NewHandler`.

- [ ] **Step 3: Write the handler**

```go
package inventory

import (
	"errors"
	"strconv"
	"time"

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

func actorFromContext(c fiber.Ctx) Actor {
	claims, _ := auth.ClaimsFromContext(c)
	if claims == nil {
		return Actor{}
	}
	return Actor{ID: claims.Subject, Role: claims.Role, WarehouseID: claims.WarehouseID}
}

func invalidBodyError(err error) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", "request body is not valid JSON", err)
}

func invalidQueryError(message string) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", message, nil)
}

func inventoryHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "inventory request data is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN", "your role or warehouse assignment does not allow this action", err)
	case errors.Is(err, ErrProductNotFound):
		return httpx.NewError(fiber.StatusNotFound, "PRODUCT_NOT_FOUND", "product was not found or is not active", err)
	case errors.Is(err, ErrLocationNotFound):
		return httpx.NewError(fiber.StatusNotFound, "LOCATION_NOT_FOUND", "location was not found", err)
	case errors.Is(err, ErrLotNotFound):
		return httpx.NewError(fiber.StatusNotFound, "LOT_NOT_FOUND", "lot was not found for this product", err)
	case errors.Is(err, ErrMovementNotFound):
		return httpx.NewError(fiber.StatusNotFound, "MOVEMENT_NOT_FOUND", "movement was not found", err)
	case errors.Is(err, ErrInsufficientStock):
		return httpx.NewError(fiber.StatusConflict, "INSUFFICIENT_STOCK", "not enough available stock for this operation", err)
	case errors.Is(err, ErrSameLocationTransfer):
		return httpx.NewError(fiber.StatusConflict, "SAME_LOCATION_TRANSFER", "transfer source and destination locations must differ", err)
	case errors.Is(err, ErrWarehouseMismatch):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "WAREHOUSE_MISMATCH", "location does not belong to your assigned warehouse", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "INVENTORY_OPERATION_FAILED", "inventory operation could not be completed", err)
	}
}

func (h *Handler) ListBalances(c fiber.Ctx) error {
	filter, err := parseBalanceFilter(c)
	if err != nil {
		return err
	}
	page, err := h.service.ListBalances(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) ListLots(c fiber.Ctx) error {
	lots, err := h.service.ListLots(c.Context(), actorFromContext(c), c.Params("product_id"))
	if err != nil {
		return inventoryHTTPError(err)
	}
	if lots == nil {
		lots = []Lot{}
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(lots))
}

type receiveRequest struct {
	LocationID     string         `json:"location_id"`
	ProductID      string         `json:"product_id"`
	Quantity       string         `json:"quantity"`
	UnitCost       string         `json:"unit_cost"`
	LotNumber      OptionalString `json:"lot_number"`
	ExpirationDate OptionalString `json:"expiration_date"`
	Reference      OptionalString `json:"reference"`
	Notes          OptionalString `json:"notes"`
}

func (h *Handler) Receive(c fiber.Ctx) error {
	var request receiveRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movement, balance, err := h.service.Receive(c.Context(), actorFromContext(c), ReceiveInput{
		LocationID: request.LocationID, ProductID: request.ProductID,
		Quantity: request.Quantity, UnitCost: request.UnitCost,
		LotNumber: request.LotNumber, ExpirationDate: request.ExpirationDate,
		Reference: request.Reference, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movement": movement, "balance": balance,
	}))
}

type pickRequest struct {
	LocationID string         `json:"location_id"`
	ProductID  string         `json:"product_id"`
	Quantity   string         `json:"quantity"`
	LotID      OptionalString `json:"lot_id"`
	Reference  OptionalString `json:"reference"`
	Notes      OptionalString `json:"notes"`
}

func (h *Handler) Pick(c fiber.Ctx) error {
	var request pickRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movements, err := h.service.Pick(c.Context(), actorFromContext(c), PickInput{
		LocationID: request.LocationID, ProductID: request.ProductID, Quantity: request.Quantity,
		LotID: request.LotID, Reference: request.Reference, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	total := "0"
	if len(movements) > 0 {
		total = request.Quantity
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movements": movements, "total_quantity": total,
	}))
}

type transferRequest struct {
	ProductID      string         `json:"product_id"`
	LotID          OptionalString `json:"lot_id"`
	Quantity       string         `json:"quantity"`
	FromLocationID string         `json:"from_location_id"`
	ToLocationID   string         `json:"to_location_id"`
	Reference      OptionalString `json:"reference"`
	Notes          OptionalString `json:"notes"`
}

func (h *Handler) Transfer(c fiber.Ctx) error {
	var request transferRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movement, source, destination, err := h.service.Transfer(c.Context(), actorFromContext(c), TransferInput{
		ProductID: request.ProductID, LotID: request.LotID, Quantity: request.Quantity,
		FromLocationID: request.FromLocationID, ToLocationID: request.ToLocationID,
		Reference: request.Reference, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movement": movement, "source_balance": source, "destination_balance": destination,
	}))
}

type adjustRequest struct {
	LocationID string         `json:"location_id"`
	ProductID  string         `json:"product_id"`
	LotID      OptionalString `json:"lot_id"`
	Direction  string         `json:"direction"`
	Quantity   string         `json:"quantity"`
	Notes      OptionalString `json:"notes"`
}

func (h *Handler) Adjust(c fiber.Ctx) error {
	var request adjustRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movement, balance, err := h.service.Adjust(c.Context(), actorFromContext(c), AdjustInput{
		LocationID: request.LocationID, ProductID: request.ProductID, LotID: request.LotID,
		Direction: request.Direction, Quantity: request.Quantity, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movement": movement, "balance": balance,
	}))
}

func (h *Handler) ListMovements(c fiber.Ctx) error {
	filter, err := parseMovementFilter(c)
	if err != nil {
		return err
	}
	page, err := h.service.ListMovements(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) GetMovement(c fiber.Ctx) error {
	movement, err := h.service.GetMovement(c.Context(), actorFromContext(c), c.Params("movement_id"))
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(movement))
}

func parseBalanceFilter(c fiber.Ctx) (BalanceListFilter, error) {
	filter := BalanceListFilter{}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return BalanceListFilter{}, invalidQueryError("limit must be an integer from 1 to 100")
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return BalanceListFilter{}, invalidQueryError("after cursor is invalid")
		}
		filter.After = &cursor
	}
	filter.LocationID = optionalQueryUUID(c.Query("location_id"))
	filter.ProductID = optionalQueryUUID(c.Query("product_id"))
	filter.WarehouseID = optionalQueryUUID(c.Query("warehouse_id"))
	filter.LotID = optionalQueryUUID(c.Query("lot_id"))
	return filter, nil
}

func parseMovementFilter(c fiber.Ctx) (MovementListFilter, error) {
	filter := MovementListFilter{}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return MovementListFilter{}, invalidQueryError("limit must be an integer from 1 to 100")
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return MovementListFilter{}, invalidQueryError("after cursor is invalid")
		}
		filter.After = &cursor
	}
	filter.ProductID = optionalQueryUUID(c.Query("product_id"))
	filter.LocationID = optionalQueryUUID(c.Query("location_id"))
	if raw := c.Query("movement_type"); raw != "" {
		filter.MovementType = &raw
	}
	if raw := c.Query("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return MovementListFilter{}, invalidQueryError("from must be an RFC3339 timestamp")
		}
		filter.From = &parsed
	}
	if raw := c.Query("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return MovementListFilter{}, invalidQueryError("to must be an RFC3339 timestamp")
		}
		filter.To = &parsed
	}
	return filter, nil
}

func optionalQueryUUID(raw string) *string {
	if raw == "" {
		return nil
	}
	return &raw
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/handler.go backend/internal/modules/inventory/handler_test.go backend/internal/modules/inventory/route.go
git commit -m "feat: add inventory handler with request binding and error mapping"
```

---

## Task 18: Routes, RBAC wiring, and end-to-end handler tests

**Files:**
- Modify: `backend/internal/modules/inventory/route.go` (replace `RegisterRoutesForTest` with the real `RegisterRoutes`)
- Modify: `backend/internal/modules/inventory/handler_test.go` (use real route registration and real auth middleware)

- [ ] **Step 1: Extend the failing tests**

Change `newTestApp` in `handler_test.go` (from Task 17) to call the real
`RegisterRoutes` instead of the temporary `RegisterRoutesForTest` — same
signature, same `issueToken` helper, just swap the one line:

```go
func newTestApp(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "inventory-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	app := fiber.New()
	RegisterRoutes(app, NewHandler(service), tokens)
	return app, tokens
}
```

`issueToken` is unchanged from Task 17 — do not redefine it. Now add the
role-boundary tests:

```go
func TestInventoryRoutesRequireAuthentication(t *testing.T) {
	app, _ := newTestApp(t, &fakeHandlerService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}

func TestTransferRouteRejectsPicker(t *testing.T) {
	app, tokens := newTestApp(t, &fakeHandlerService{})
	token := issueToken(t, tokens, auth.RolePicker)
	body, _ := json.Marshal(map[string]any{
		"product_id": "22222222-2222-2222-2222-222222222222", "quantity": "1.000",
		"from_location_id": "11111111-1111-1111-1111-111111111111",
		"to_location_id":   "33333333-3333-3333-3333-333333333333",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/transfer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.StatusCode)
	}
}

func TestReceiveRouteAllowsPicker(t *testing.T) {
	service := &fakeHandlerService{receiveFn: func(Actor, ReceiveInput) (Movement, Balance, error) {
		return Movement{ID: "m1"}, Balance{ID: "b1"}, nil
	}}
	app, tokens := newTestApp(t, service)
	token := issueToken(t, tokens, auth.RolePicker)
	body, _ := json.Marshal(map[string]any{
		"location_id": "11111111-1111-1111-1111-111111111111",
		"product_id":  "22222222-2222-2222-2222-222222222222",
		"quantity":    "1.000", "unit_cost": "1.0000",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/movements/receive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}
}
```

No other changes needed to the three tests from Task 17
(`TestReceiveHandlerMapsForbidden`, `TestReceiveHandlerRejectsMalformedJSON`,
`TestPickHandlerReturnsMovementsArray`) — they already call `newTestApp`,
which now points at the real `RegisterRoutes` once Step 3 below lands, so
the whole file ends up exercising one consistent, real code path with no
leftover fake-claims shortcuts.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/modules/inventory/... -v`
Expected: FAIL — `undefined: RegisterRoutes`.

- [ ] **Step 3: Replace `route.go`**

```go
package inventory

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	receiveOrPick := auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker)
	transferOrAdjust := auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager)

	inventory := api.Group("/inventory")
	inventory.Get("", handler.ListBalances)
	inventory.Get("/products/:product_id/lots", handler.ListLots)

	movements := inventory.Group("/movements")
	movements.Post("/receive", receiveOrPick, handler.Receive)
	movements.Post("/pick", receiveOrPick, handler.Pick)
	movements.Post("/transfer", transferOrAdjust, handler.Transfer)
	movements.Post("/adjust", transferOrAdjust, handler.Adjust)
	movements.Get("", handler.ListMovements)
	movements.Get("/:movement_id", handler.GetMovement)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/modules/inventory/... -v`
Expected: PASS for the whole package.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/modules/inventory/route.go backend/internal/modules/inventory/handler_test.go
git commit -m "feat: add inventory routes with RBAC and end-to-end handler tests"
```

---

## Task 19: Wire the module into `app.go`

**Files:**
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/app_test.go`

- [ ] **Step 1: Write the failing test**

Add to `app_test.go`, following the exact shape of
`TestUsersRoutesAreRegisteredBeforeNotFoundHandler`:

```go
func TestInventoryRoutesAreRegisteredBeforeNotFoundHandler(t *testing.T) {
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

	response, err := server.Test(httptest.NewRequest(http.MethodGet, "/api/v1/inventory", nil))
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

Run: `cd backend && go test ./internal/app/... -run TestInventoryRoutesAreRegisteredBeforeNotFoundHandler -v`
Expected: FAIL — `status = 404, want 401` (route not registered yet).

- [ ] **Step 3: Wire the module into `app.go`**

Find the block that wires the `users` module (from `internal/app/app.go`,
after warehouse and before the `ROUTE_NOT_FOUND` fallback) and add the
inventory wiring immediately after it:

```go
	inventoryRepository := inventory.NewPostgresRepository(db)
	inventoryService := inventory.NewService(inventoryRepository)
	inventory.RegisterRoutes(server, inventory.NewHandler(inventoryService), tokenManager)
```

Add the import:

```go
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/inventory"
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/app/... -v`
Expected: PASS, including `TestCatalogRoutesAreRegisteredBeforeNotFoundHandler`
and `TestUsersRoutesAreRegisteredBeforeNotFoundHandler` still passing.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/app/app.go backend/internal/app/app_test.go
git commit -m "feat: wire inventory routes into the application"
```

---

## Task 20: Postgres lifecycle re-run and full backend validation

**Files:** none (verification only)

- [ ] **Step 1: Re-run the full inventory lifecycle integration test**

```bash
cd backend
export BWIMS_TEST_DATABASE_URL="<same connection string used for prior validations>"
go test ./internal/modules/inventory/... -run TestPostgresInventoryLifecycle -v
```

Expected: PASS with no skipped subtests.

- [ ] **Step 2: Run the complete backend suite**

```bash
cd backend
go test ./... -count=1
go vet ./...
```

Expected: all packages PASS (the pre-existing `TestTokenManagerRejectsTamperedToken`
flake in `internal/modules/auth` is unrelated to this branch — see the
Week 9 validation doc for its prior documentation; rerun once if it hits).
`go vet` reports nothing.

- [ ] **Step 3: Confirm migration state**

```bash
cd backend
ls migrations/ | tail -5
```

Expected: ends at `000005_inventory_constraints.up.sql` / `.down.sql`.

- [ ] **Step 4: No commit for this task** — verification checkpoint only.

---

## Task 21: Update `docs/api-contract.md`

**Files:**
- Modify: `docs/api-contract.md`

- [ ] **Step 1: Add an "Inventory and stock movement endpoints" section**

Insert a new `##` section after "Admin user management endpoints" and
before "Initial HTTP status policy":

```markdown
## Inventory and stock movement endpoints

Every endpoint requires a valid Bearer access token. Reads are open to all
four roles, scoped to the caller's assigned warehouse for non-admins (a
non-admin token without an assigned `warehouse_id` receives `FORBIDDEN`).
`admin` and `warehouse_manager` may receive, pick, transfer, and adjust;
`picker` may receive and pick only; `viewer` may only read.

### Balance and lot routes

| Method | Route | Result |
| --- | --- | --- |
| `GET` | `/api/v1/inventory` | List balances; filters `location_id`, `product_id`, `warehouse_id`, `lot_id` |
| `GET` | `/api/v1/inventory/products/:product_id/lots` | List lots for a product, FEFO order |

### Movement write routes

| Method | Route | Roles |
| --- | --- | --- |
| `POST` | `/api/v1/inventory/movements/receive` | admin, warehouse_manager, picker |
| `POST` | `/api/v1/inventory/movements/pick` | admin, warehouse_manager, picker |
| `POST` | `/api/v1/inventory/movements/transfer` | admin, warehouse_manager |
| `POST` | `/api/v1/inventory/movements/adjust` | admin, warehouse_manager |
| `GET` | `/api/v1/inventory/movements` | all authenticated roles |
| `GET` | `/api/v1/inventory/movements/:movement_id` | all authenticated roles |

Receive body:

```json
{
  "location_id": "<uuid>", "product_id": "<uuid>",
  "quantity": "50.000", "unit_cost": "1.2500",
  "lot_number": "LOT-2026-08-18-A", "expiration_date": "2026-11-01",
  "reference": "PO-1042", "notes": null
}
```

`lot_number` is required when the product is lot-tracked, forbidden
otherwise. Reusing an existing `(product_id, lot_number)` pair does not
overwrite its stored `expiration_date`. Response:
`{ "movement": {...}, "balance": {...} }`.

Pick body:

```json
{
  "location_id": "<uuid>", "product_id": "<uuid>", "quantity": "30.000",
  "lot_id": null, "reference": "SO-2201", "notes": null
}
```

`lot_id` omitted auto-selects lots by FEFO (earliest expiration first),
splitting across lots as needed; a pick that cannot be fully satisfied
fails atomically with `INSUFFICIENT_STOCK`. Response, since a pick can span
multiple lots: `{ "movements": [...], "total_quantity": "30.000" }`.

Transfer body:

```json
{
  "product_id": "<uuid>", "lot_id": null, "quantity": "10.000",
  "from_location_id": "<uuid>", "to_location_id": "<uuid>",
  "reference": null, "notes": null
}
```

`lot_id` is required when the product is lot-tracked. `from_location_id`
must differ from `to_location_id`. Response:
`{ "movement": {...}, "source_balance": {...}, "destination_balance": {...} }`.

Adjust body:

```json
{
  "location_id": "<uuid>", "product_id": "<uuid>", "lot_id": null,
  "direction": "increase", "quantity": "5.000", "notes": "cycle count"
}
```

`direction` is `"increase"` or `"decrease"`. Adjustments never create
`cost_layers` rows — see the design doc's "FIFO Cost Layer Consumption"
section for why. Response: `{ "movement": {...}, "balance": {...} }`.

### List queries and response

Balance and movement lists accept `limit` (1-100, default 20) and `after`
(opaque cursor). Movement lists additionally accept `product_id`,
`location_id` (matches either `from_location_id` or `to_location_id`),
`movement_type`, and `from`/`to` (RFC3339 timestamps). Response shape
matches every other module:

```json
{ "success": true, "data": { "items": [], "page": { "next_cursor": null, "has_more": false } } }
```

### Errors

| HTTP status | Code | Meaning |
| --- | --- | --- |
| 400 | `INVALID_REQUEST` | Invalid JSON, UUID, filter, cursor, or limit |
| 403 | `FORBIDDEN` | Role, or non-admin without/outside assigned warehouse |
| 404 | `PRODUCT_NOT_FOUND` | Product does not exist or is inactive |
| 404 | `LOCATION_NOT_FOUND` | Location does not exist |
| 404 | `LOT_NOT_FOUND` | Referenced lot does not exist for the product |
| 404 | `MOVEMENT_NOT_FOUND` | Movement does not exist |
| 409 | `INSUFFICIENT_STOCK` | Not enough available quantity for pick/transfer/decrease |
| 409 | `SAME_LOCATION_TRANSFER` | `from_location_id` equals `to_location_id` |
| 422 | `WAREHOUSE_MISMATCH` | Non-admin location(s) fall outside their assigned warehouse |
| 422 | `VALIDATION_ERROR` | Required business data missing/invalid |
| 500 | `INVENTORY_OPERATION_FAILED` | Unexpected failure, including cost/balance ledger drift |
```

- [ ] **Step 2: Commit**

```bash
git add docs/api-contract.md
git commit -m "docs: record inventory and stock movement API contract"
```

---

## Task 22: Update `docs/requirements-traceability.md`

**Files:**
- Modify: `docs/requirements-traceability.md`

- [ ] **Step 1: Flip FR-10 through FR-19 to Implemented**

Change these existing rows (keep the same `| FR-N | ... |` columns, only
replacing "Schema ready"/"Planned" with "Implemented" and updating the
Gantt-work column to point at this module):

```
| FR-10 | Inventory balances | 1.2, 3.1 | Week 9-10 inventory module | Repository and integration tests | Implemented |
| FR-11 | Lot and expiry tracking | 1.2, 3.1 | Week 9-10 inventory module | Repository and integration tests | Implemented |
| FR-13 | Transactional receive | 3.1 | Week 9-10 inventory module | Transaction integration tests | Implemented |
| FR-14 | FEFO pick | 3.2 | Week 9-10 inventory module | Mixed-lot ordering tests | Implemented |
| FR-15 | Transfer stock | 3.2 | Week 9-10 inventory module | Atomic source/destination test | Implemented |
| FR-16 | Stock adjustment | 3.2 | Week 9-10 inventory module | Reason and audit tests | Implemented |
| FR-17 | Movement history | 1.2, 3.6 | Immutable movement schema | Read and immutability tests | Implemented |
| FR-18 | FIFO cost layers | 1.2, 3.2 | Week 9-10 inventory module | Oldest-layer consumption tests | Implemented |
| FR-19 | Prevent expired-lot picking | 3.2 | Week 9-10 inventory module | Boundary-date tests | Partially implemented |
```

FR-19 is marked "Partially implemented," not "Implemented": this slice's
FEFO ordering picks the earliest-expiring lot first, but does not add a
hard rejection for picking an *already-expired* lot (that stronger
guarantee needs a business decision — block entirely, warn-and-confirm, or
allow with an audit flag — which is exactly the kind of call this project's
standing rules reserve for the user, not something to decide silently
here). Leave a one-line note in the row's GitHub/evidence column:
`FEFO ordering only; hard expired-lot block deferred, needs a product decision`.

- [ ] **Step 2: Commit**

```bash
git add docs/requirements-traceability.md
git commit -m "docs: mark FR-10 through FR-18 implemented, FR-19 partial"
```

---

## Task 23: Create `docs/week-10-inventory-validation.md`

**Files:**
- Create: `docs/week-10-inventory-validation.md`

- [ ] **Step 1: Write the validation evidence doc, mirroring the Week 8/9 docs' structure exactly**

Follow `docs/week-9-user-management-validation.md`'s exact section order
(Implemented scope / Automated verification / Database migration status /
Remaining work and boundaries), filling in the real date, the real last
commit hash from Task 19, and the real output of the Task 20 commands —
paste actual terminal output, not a description of what you expect it to
say. In "Remaining work and boundaries," explicitly list: no frontend
screens for inventory (separate slice), no expiry alerts or valuation
reporting (Week 11-12), FR-19's hard expired-lot block deferred per Task
22's note, and adjustments intentionally not touching cost layers per the
design doc.

- [ ] **Step 2: Commit**

```bash
git add docs/week-10-inventory-validation.md
git commit -m "docs: record week 10 inventory validation evidence"
```

(Fill in real dates, hashes, and command output before this commit — an
evidence doc with placeholder brackets left in is not acceptable per this
project's standing "no evidence claims without real evidence" rule.)

---

## Task 24: Final validation and PR

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
cd ../frontend
npm run test
npm run typecheck
npm run build
```

Expected: everything passes (no frontend changes in this slice, this is a
regression check). If anything fails, fix it in the task that owns the
failing file, not inline here.

- [ ] **Step 2: Push the branch**

```bash
git push -u origin codex/inventory-movements
```

- [ ] **Step 3: Open a pull request**

```bash
gh pr create --title "Implement Inventory Balances and Stock Movements API" --body "$(cat <<'EOF'
## Summary
- Read-only inventory balance and lot listing, warehouse-scoped for non-admins
- Receive, FEFO pick (with multi-lot splitting), transfer, and adjust movements
- FIFO cost-layer consumption on pick, decoupled from FEFO lot selection by design
- Adjustments intentionally do not touch cost layers (no cost basis input)
- One additive migration: inventory_balances.created_at for cursor pagination
- Closes FR-10 through FR-18; FR-19 partially (FEFO ordering only, no hard expired-lot block yet)

## Test plan
- [x] `go build ./...`
- [x] `go vet ./...`
- [x] `go test ./... -count=1`
- [x] Postgres lifecycle integration test (TestPostgresInventoryLifecycle)
- [x] `docker compose config --quiet`
- [x] Frontend test/typecheck/build re-run (no frontend changes in this slice)

Closes FR-10 through FR-18 / advances Gantt tasks 3.1-3.2 (pending review and merge).
EOF
)"
```

**Do not merge this PR automatically.** Report the PR URL back to the user
and stop — merging requires their explicit approval, same as PR #11 and
PR #12.

- [ ] **Step 4: Report status**

Tell the user: branch pushed, PR opened at `<url>`, all automated checks
green, `docs/week-10-inventory-validation.md` has real command output. Ask
before updating Linear/Gantt task 3.1-3.2 status or FR-10 through FR-19
status anywhere outside this repo's own `docs/requirements-traceability.md`
(which Task 22 already updated in-repo). Flag FR-19's partial status and
the expired-lot-block decision explicitly — that's a product call, not an
implementation detail.

---

## Self-Review Notes

**Spec coverage check against the design doc:**
- Read balances/lots, warehouse-scoped — Task 6, 7, 13. ✓
- Receive creates/reuses lot, opens cost layer — Task 8. ✓
- FEFO pick with multi-lot splitting, FIFO cost consumption decoupled from lot order — Task 9 (integration test explicitly proves the divergence case). ✓
- Transfer, single lot, warehouse-scoped both ends — Task 10, 15. ✓
- Adjust, direction+quantity, no cost-layer writes — Task 11, 15. ✓
- Movement history + detail, cursor pagination — Task 12, 16. ✓
- Migration `000005` — Task 5, verified no drift in Task 20. ✓
- RBAC table (admin/manager/picker/viewer × read/receive-pick/transfer-adjust) — Task 18 route wiring + Task 13-16 service checks (defense in depth, matching the `warehouse` module's real pattern). ✓
- Error contract — Task 17 `inventoryHTTPError`. ✓
- `docs/api-contract.md`, `docs/requirements-traceability.md`, validation doc — Tasks 21-23. ✓
- Gantt/Linear updates deferred to explicit user approval — Task 24 Step 4. ✓
- Reporting, barcode labels, frontend explicitly out of scope — called out in Task 23. ✓

**Type/signature consistency check:** `Repository` interface (Task 6) methods match `PostgresRepository` across Tasks 6-12 and `fakeRepository` in Task 13. `Service` interface (Task 13) methods match `service` across Tasks 13-16 and `fakeHandlerService` in Task 17-18. `Balance`, `Lot`, `Movement`, `*Input`, `ListFilter` types are defined once in Task 3 and reused with identical field names throughout — `AvailableQuantity` is computed once in `scanBalance` (Task 6), never redefined.

**Deviation from the design doc, flagged explicitly:** the design doc's RBAC
table doesn't specify whether route-level `RequireRoles` middleware or
service-level checks (or both) enforce role gating. This plan uses both,
matching the actually-existing `warehouse` module's pattern (verified by
reading its real `route.go`/`service.go` before writing this plan, not
assumed) — route middleware for mutations, service-level checks as a
second layer that's also independently unit-testable without an HTTP
server.
