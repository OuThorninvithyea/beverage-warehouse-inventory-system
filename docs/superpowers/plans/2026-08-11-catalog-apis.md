# Category, Product, and Barcode API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build authenticated Category and Product CRUD APIs with admin/warehouse-manager mutation access, cursor pagination, soft deactivation, and exact EAN-13/UPC-A lookup.

**Architecture:** Add one `catalog` vertical slice using the existing `Route -> Handler -> Service -> Repository -> PostgreSQL` pattern. Keep normalization and authorization in the service, transactional hierarchy/reference rules in the pgx repository, and HTTP parsing/error mapping in the handler.

**Tech Stack:** Go, Fiber v3, pgx v5, PostgreSQL 16, UUIDs, Go `testing`, Docker Compose, Markdown API documentation.

## Global Constraints

- All routes use `/api/v1` and require a valid Bearer access token.
- `admin` and `warehouse_manager` may create, update, and deactivate global catalog resources.
- `picker` and `viewer` are read-only.
- Barcode lookup is read-only and never changes inventory.
- Barcodes are optional but, when present, must be valid 12-digit UPC-A or 13-digit EAN-13 values.
- Category/Product rows are soft-deactivated, never physically deleted.
- Lists use `limit` 1-100, opaque `after` cursors, and `(created_at DESC, id DESC)` ordering.
- Success/error bodies use the existing BWIMS envelopes; raw database errors are never exposed.
- Follow test-first red-green-refactor for every production behavior.
- Use direct pgx; do not introduce a generic CRUD framework, ORM, or reflection layer.

---

## File Structure

Create:

```text
backend/internal/modules/catalog/model.go
backend/internal/modules/catalog/optional.go
backend/internal/modules/catalog/optional_test.go
backend/internal/modules/catalog/barcode.go
backend/internal/modules/catalog/barcode_test.go
backend/internal/modules/catalog/pagination.go
backend/internal/modules/catalog/pagination_test.go
backend/internal/modules/catalog/service.go
backend/internal/modules/catalog/service_test.go
backend/internal/modules/catalog/repository.go
backend/internal/modules/catalog/repository_integration_test.go
backend/internal/modules/catalog/handler.go
backend/internal/modules/catalog/handler_test.go
backend/internal/modules/catalog/route.go
backend/migrations/000004_catalog_constraints.up.sql
backend/migrations/000004_catalog_constraints.down.sql
docs/week-8-catalog-validation.md
```

Modify:

```text
backend/internal/app/app.go
backend/internal/app/app_test.go
docs/api-contract.md
docs/requirements-traceability.md
```

---

### Task 1: Catalog Models, Optional JSON Values, Barcode Validation, and Pagination

**Files:**
- Create: `backend/internal/modules/catalog/model.go`
- Create: `backend/internal/modules/catalog/optional.go`
- Create: `backend/internal/modules/catalog/optional_test.go`
- Create: `backend/internal/modules/catalog/barcode.go`
- Create: `backend/internal/modules/catalog/barcode_test.go`
- Create: `backend/internal/modules/catalog/pagination.go`
- Create: `backend/internal/modules/catalog/pagination_test.go`

**Interfaces:**
- Consumes: `auth.RoleAdmin`, `auth.RoleWarehouseManager`, `auth.RolePicker`, `auth.RoleViewer`; UUID and RFC3339 conventions from existing modules.
- Produces: `Actor`, `Category`, `Product`, `OptionalString`, `CategoryInput`, `ProductInput`, `ListFilter`, `Page[T]`, `EncodeCursor`, `DecodeCursor`, and `ValidateBarcode` for all later tasks.

- [ ] **Step 1: Write failing OptionalString decoding tests**

Add table tests proving all three JSON states:

```go
func TestOptionalStringJSONStates(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantSet   bool
		wantValue *string
	}{
		{name: "omitted", body: `{}`, wantSet: false},
		{name: "null", body: `{"value":null}`, wantSet: true},
		{name: "string", body: `{"value":"abc"}`, wantSet: true, wantValue: stringPointer("abc")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got struct { Value OptionalString `json:"value"` }
			if err := json.Unmarshal([]byte(tt.body), &got); err != nil { t.Fatal(err) }
			if got.Value.Set != tt.wantSet { t.Fatalf("Set=%v want %v", got.Value.Set, tt.wantSet) }
			if !equalStringPointers(got.Value.Value, tt.wantValue) { t.Fatalf("Value=%v want %v", got.Value.Value, tt.wantValue) }
		})
	}
}
```

- [ ] **Step 2: Run the OptionalString test and verify RED**

Run from `backend/`:

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run TestOptionalStringJSONStates -count=1
```

Expected: compilation fails because `OptionalString` does not exist.

- [ ] **Step 3: Implement OptionalString and catalog model types**

Use this decoding contract in `optional.go`:

```go
type OptionalString struct {
	Set   bool
	Value *string
}

func (value *OptionalString) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = &decoded
	return nil
}
```

Define these exact public types in `model.go`:

```go
type Actor struct { Role string }

type Category struct {
	ID string `json:"id"`
	ParentID *string `json:"parent_id"`
	Name string `json:"name"`
	IsActive bool `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID string `json:"id"`
	CategoryID *string `json:"category_id"`
	SKU string `json:"sku"`
	Barcode *string `json:"barcode"`
	Name string `json:"name"`
	Unit string `json:"unit"`
	IsLotTracked bool `json:"is_lot_tracked"`
	IsActive bool `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CategoryInput struct {
	Name string
	ParentID OptionalString
	IsActive *bool
}

type ProductInput struct {
	CategoryID OptionalString
	SKU string
	Barcode OptionalString
	Name string
	Unit string
	IsLotTracked *bool
	IsActive *bool
}

type Cursor struct { CreatedAt time.Time; ID string }
type ListFilter struct {
	Limit int
	After *Cursor
	Search string
	IsActive *bool
	ParentID *string
	CategoryID *string
}
type PageInfo struct { NextCursor *string `json:"next_cursor"`; HasMore bool `json:"has_more"` }
type Page[T any] struct { Items []T `json:"items"`; Page PageInfo `json:"page"` }
```

- [ ] **Step 4: Re-run OptionalString tests and verify GREEN**

Run the Step 2 command. Expected: PASS.

- [ ] **Step 5: Write failing barcode checksum tests**

Test valid EAN-13 `4006381333931`, valid UPC-A `036000291452`, bad checksum variants, non-digits, empty text, and 11/14-digit values.

```go
func TestValidateBarcode(t *testing.T) {
	tests := map[string]bool{
		"4006381333931": true,
		"036000291452": true,
		"4006381333932": false,
		"036000291453": false,
		"ABC600029145": false,
		"": false,
		"12345678901": false,
		"12345678901234": false,
	}
	for value, want := range tests {
		if got := ValidateBarcode(value); got != want { t.Fatalf("ValidateBarcode(%q)=%v want %v", value, got, want) }
	}
}
```

- [ ] **Step 6: Run barcode tests and verify RED**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run TestValidateBarcode -count=1
```

Expected: compilation fails because `ValidateBarcode` does not exist.

- [ ] **Step 7: Implement the shared UPC-A/EAN-13 checksum**

Implement digit-only length validation and calculate the final check digit from right to left, beginning with weight 3 on the rightmost payload digit and alternating weights 3 and 1:

```go
func ValidateBarcode(value string) bool {
	if len(value) != 12 && len(value) != 13 { return false }
	for _, character := range value { if character < '0' || character > '9' { return false } }
	sum, weight := 0, 3
	for index := len(value)-2; index >= 0; index-- {
		sum += int(value[index]-'0') * weight
		if weight == 3 { weight = 1 } else { weight = 3 }
	}
	want := (10 - sum%10) % 10
	return int(value[len(value)-1]-'0') == want
}
```

- [ ] **Step 8: Re-run barcode tests and verify GREEN**

Run the Step 6 command. Expected: PASS.

- [ ] **Step 9: Write failing cursor and page tests**

Copy the Warehouse cursor behavior into catalog-specific tests: round-trip an RFC3339Nano timestamp and UUID, reject malformed/base64/non-UUID cursors, clamp/default limits, return `[]` for nil items, and expose a next cursor only when `limit + 1` records exist.

- [ ] **Step 10: Implement pagination helpers and verify GREEN**

Implement these exact signatures in `pagination.go`:

```go
func EncodeCursor(createdAt time.Time, id string) string
func DecodeCursor(value string) (Cursor, error)
func normalizeLimit(limit int) int
func categoryPage(items []Category, requestedLimit int) Page[Category]
func productPage(items []Product, requestedLimit int) Page[Product]
```

Use `ErrInvalidCursor` for decoding failures and base64 URL encoding of `RFC3339Nano + "|" + UUID`. Run:

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -count=1
```

Expected: PASS.

- [ ] **Step 11: Commit domain primitives**

```bash
git add backend/internal/modules/catalog/model.go backend/internal/modules/catalog/optional.go backend/internal/modules/catalog/optional_test.go backend/internal/modules/catalog/barcode.go backend/internal/modules/catalog/barcode_test.go backend/internal/modules/catalog/pagination.go backend/internal/modules/catalog/pagination_test.go
git commit -m "feat: add catalog domain primitives"
```

---

### Task 2: Category Service Rules

**Files:**
- Create: `backend/internal/modules/catalog/service.go`
- Create: `backend/internal/modules/catalog/service_test.go`

**Interfaces:**
- Consumes: Task 1 model, pagination, optional-value, and auth role types.
- Produces: catalog `Service` and `Repository` interfaces; Category authorization, normalization, pagination, CRUD, and deactivation behavior used by the handler and PostgreSQL repository.

- [ ] **Step 1: Define repository and service interfaces in a failing service test**

Use these exact interfaces:

```go
type Repository interface {
	ListCategories(context.Context, ListFilter) ([]Category, error)
	GetCategory(context.Context, string) (Category, error)
	CreateCategory(context.Context, CategoryInput) (Category, error)
	UpdateCategory(context.Context, string, CategoryInput) (Category, error)
	DeactivateCategory(context.Context, string) error
	ListProducts(context.Context, ListFilter) ([]Product, error)
	GetProduct(context.Context, string) (Product, error)
	GetProductByBarcode(context.Context, string) (Product, error)
	CreateProduct(context.Context, ProductInput) (Product, error)
	UpdateProduct(context.Context, string, ProductInput) (Product, error)
	DeactivateProduct(context.Context, string) error
}

type Service interface {
	ListCategories(context.Context, Actor, ListFilter) (Page[Category], error)
	GetCategory(context.Context, Actor, string) (Category, error)
	CreateCategory(context.Context, Actor, CategoryInput) (Category, error)
	UpdateCategory(context.Context, Actor, string, CategoryInput) (Category, error)
	DeactivateCategory(context.Context, Actor, string) error
	ListProducts(context.Context, Actor, ListFilter) (Page[Product], error)
	GetProduct(context.Context, Actor, string) (Product, error)
	GetProductByBarcode(context.Context, Actor, string) (Product, error)
	CreateProduct(context.Context, Actor, ProductInput) (Product, error)
	UpdateProduct(context.Context, Actor, string, ProductInput) (Product, error)
	DeactivateProduct(context.Context, Actor, string) error
}
```

Create a fake repository that records inputs and returns configured values. Write tests proving admin and manager Category mutations reach the repository while picker/viewer mutations return `ErrForbidden` without repository calls.

- [ ] **Step 2: Run Category authorization tests and verify RED**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run 'TestCategory.*Role' -count=1
```

Expected: compilation fails because `NewService`, service errors, and interfaces do not exist.

- [ ] **Step 3: Implement service errors, interfaces, and role checks**

Define exact sentinel errors:

```go
var (
	ErrForbidden = errors.New("catalog access is forbidden")
	ErrValidation = errors.New("catalog data is invalid")
	ErrInvalidID = errors.New("resource id is invalid")
	ErrInvalidCursor = errors.New("catalog cursor is invalid")
	ErrInvalidBarcode = errors.New("barcode is invalid")
	ErrCategoryNotFound = errors.New("category not found")
	ErrProductNotFound = errors.New("product not found")
	ErrCategoryNameConflict = errors.New("category name already exists")
	ErrCategoryInUse = errors.New("category is in use")
	ErrCategoryCycle = errors.New("category hierarchy cycle")
	ErrProductSKUConflict = errors.New("product sku already exists")
	ErrProductBarcodeConflict = errors.New("product barcode already exists")
)
```

`canMutateCatalog` returns true only for `auth.RoleAdmin` and `auth.RoleWarehouseManager`.

- [ ] **Step 4: Re-run Category authorization tests and verify GREEN**

Run the Step 2 command. Expected: PASS.

- [ ] **Step 5: Write failing Category normalization and pagination tests**

Prove:

- names are trimmed and blank names return `ErrValidation`;
- create defaults `is_active=true`;
- supplied parent IDs must be UUIDs before repository calls;
- update/deactivate resource IDs must be UUIDs;
- `search` is trimmed;
- list requests pass `requestedLimit + 1` to the repository;
- nil repository results serialize to empty `items`;
- repository errors including conflict/cycle/in-use sentinels are preserved.

- [ ] **Step 6: Implement minimal Category service behavior**

Implement `normalizeCategoryInput(input CategoryInput, defaults bool)` and Category service methods. On create, treat unset `ParentID` as explicit SQL null by setting `Set=true, Value=nil`; on update, preserve an unset parent. Use `uuid.Parse` for any non-null parent and resource IDs.

- [ ] **Step 7: Run all Category service tests and verify GREEN**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run 'Test(Category|ListCategories|GetCategory|CreateCategory|UpdateCategory|DeactivateCategory)' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Category service behavior**

```bash
git add backend/internal/modules/catalog/service.go backend/internal/modules/catalog/service_test.go
git commit -m "feat: add category service rules"
```

---

### Task 3: Product Service and Barcode Lookup Rules

**Files:**
- Modify: `backend/internal/modules/catalog/service.go`
- Modify: `backend/internal/modules/catalog/service_test.go`

**Interfaces:**
- Consumes: Task 1 `ValidateBarcode`, Task 2 service/repository interfaces and errors.
- Produces: normalized Product CRUD, global read access, exact active-barcode lookup, and Product pagination behavior.

- [ ] **Step 1: Write failing Product role and normalization tests**

Prove:

- admin and manager may create/update/deactivate;
- picker and viewer cannot mutate but all four roles can list/get/lookup;
- SKU becomes uppercase and is required;
- name is trimmed and required;
- unit becomes lowercase and is required;
- create defaults `is_lot_tracked=true` and `is_active=true`;
- Category UUIDs are validated;
- omitted Category/barcode values become null on create but preserve on update;
- explicit JSON-null domain values remain `Set=true, Value=nil`;
- invalid UPC/EAN values return `ErrInvalidBarcode` without repository access;
- valid barcode text is preserved as digits.

- [ ] **Step 2: Run Product service tests and verify RED**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run 'TestProduct.*(Role|Normalize|Barcode)' -count=1
```

Expected: tests fail because Product service methods are not implemented.

- [ ] **Step 3: Implement Product normalization and authorization**

Implement `normalizeProductInput(input ProductInput, defaults bool)` with:

```go
input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))
input.Name = strings.TrimSpace(input.Name)
input.Unit = strings.ToLower(strings.TrimSpace(input.Unit))
```

Reject any blank required field. Validate optional Category UUIDs. Trim barcode values, reject empty strings, and call `ValidateBarcode`. On create, convert unset Category/barcode to explicit null and default both booleans to true.

- [ ] **Step 4: Re-run Product role/normalization tests and verify GREEN**

Run the Step 2 command. Expected: PASS.

- [ ] **Step 5: Write failing Product list and lookup tests**

Prove search trimming, Category UUID validation, `limit + 1` repository calls, empty arrays, next-cursor behavior, invalid lookup checksum rejection, and exact propagation of `ErrProductNotFound` for valid but missing/inactive barcodes.

- [ ] **Step 6: Implement Product list/get/lookup methods**

`GetProductByBarcode` trims the path value, validates its checksum, and calls `repository.GetProductByBarcode`; it never invokes mutation methods. Product list normalizes filters and calls `productPage`.

- [ ] **Step 7: Run the complete catalog service suite**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit Product service behavior**

```bash
git add backend/internal/modules/catalog/service.go backend/internal/modules/catalog/service_test.go
git commit -m "feat: add product and barcode service rules"
```

---

### Task 4: Catalog Migration and PostgreSQL Repository

**Files:**
- Create: `backend/migrations/000004_catalog_constraints.up.sql`
- Create: `backend/migrations/000004_catalog_constraints.down.sql`
- Create: `backend/internal/modules/catalog/repository.go`
- Create: `backend/internal/modules/catalog/repository_integration_test.go`

**Interfaces:**
- Consumes: Task 2 `Repository`, domain inputs, filters, and sentinel errors; existing `pgxpool.Pool`.
- Produces: transactional PostgreSQL implementation used by app wiring and HTTP integration tests.

- [ ] **Step 1: Write the failing migration/repository integration test**

Follow the Warehouse integration-test environment contract: skip only when `BWIMS_TEST_DATABASE_URL` is empty, create unique names/SKUs with a timestamp suffix, and clean inserted rows in dependency order. Test:

- Category create/get/update/list/deactivate.
- Product create/get/update/list/deactivate.
- case-insensitive Category name conflict;
- case-insensitive Product SKU conflict;
- global barcode conflict;
- missing/inactive Category rejection;
- Category cycle rejection;
- Category-in-use rejection for active children and Products;
- explicit clearing of parent, Product Category, and Product barcode;
- exact active-only barcode lookup;
- stable search/filter pagination.

- [ ] **Step 2: Run repository integration tests and verify RED**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run TestPostgresCatalogLifecycle -count=1
```

Expected: compilation fails because `NewPostgresRepository` does not exist.

- [ ] **Step 3: Add migration 000004**

Up migration:

```sql
ALTER TABLE categories
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE products DROP CONSTRAINT products_sku_key;
CREATE UNIQUE INDEX products_sku_lower_unique_idx ON products (LOWER(sku));
CREATE INDEX categories_parent_active_idx ON categories (parent_id, is_active);
CREATE INDEX products_category_active_idx ON products (category_id, is_active);
CREATE INDEX products_active_created_idx ON products (is_active, created_at DESC, id DESC);
```

Down migration:

```sql
DROP INDEX IF EXISTS products_active_created_idx;
DROP INDEX IF EXISTS products_category_active_idx;
DROP INDEX IF EXISTS categories_parent_active_idx;
DROP INDEX IF EXISTS products_sku_lower_unique_idx;
ALTER TABLE products ADD CONSTRAINT products_sku_key UNIQUE (sku);
ALTER TABLE categories DROP COLUMN is_active;
```

- [ ] **Step 4: Implement Category repository operations**

Use `limit + 1` list queries with case-insensitive search and optional parent/active/cursor filters. Use `CASE WHEN $set THEN $value::uuid ELSE parent_id END` for parent updates.

Category create/update transactions must:

1. lock a supplied parent using `SELECT id FROM categories WHERE id=$1 AND is_active FOR UPDATE`;
2. return `ErrCategoryNotFound` when absent;
3. reject self-parent;
4. use a recursive descendants CTE to return `ErrCategoryCycle` when the proposed parent is a descendant;
5. insert/update and return the full Category.

Category deactivation must lock the target row, return `ErrCategoryNotFound` when absent, check active children and active Products, return `ErrCategoryInUse` when either exists, then set `is_active=false` and `updated_at=NOW()` in the same transaction.

- [ ] **Step 5: Implement Product repository operations**

Product create/update transactions lock a supplied active Category before writing. Use boolean set flags for optional Category and barcode fields:

```sql
category_id = CASE WHEN $category_set THEN $category_id::uuid ELSE category_id END,
barcode = CASE WHEN $barcode_set THEN $barcode ELSE barcode END
```

`GetProductByBarcode` includes `WHERE barcode=$1 AND is_active=TRUE`. List search covers SKU, name, and barcode. Deactivation is idempotent for an existing row.

- [ ] **Step 6: Map exact PostgreSQL constraints**

Map:

```text
categories_name_unique_idx -> ErrCategoryNameConflict
products_sku_lower_unique_idx -> ErrProductSKUConflict
products_barcode_unique_idx -> ErrProductBarcodeConflict
categories_parent_id_fkey -> ErrCategoryNotFound
products_category_id_fkey -> ErrCategoryNotFound
```

Wrap every other error with its operation name.

- [ ] **Step 7: Apply migrations and run PostgreSQL tests**

Create a clean database and apply migrations 000001-000004:

```bash
docker compose up -d postgres
docker compose exec -T postgres dropdb -U bwims --if-exists bwims_catalog_test
docker compose exec -T postgres createdb -U bwims bwims_catalog_test
docker compose run --rm migrate -path=/migrations -database='postgres://bwims:bwims@postgres:5432/bwims_catalog_test?sslmode=disable' up
```

Then run:

```bash
env BWIMS_TEST_DATABASE_URL='postgres://bwims:bwims@localhost:5432/bwims_catalog_test?sslmode=disable' GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run TestPostgresCatalogLifecycle -count=1
```

Expected: PASS.

- [ ] **Step 8: Verify migration rollback/reapply**

```bash
docker compose run --rm migrate -path=/migrations -database='postgres://bwims:bwims@postgres:5432/bwims_catalog_test?sslmode=disable' down 1
docker compose run --rm migrate -path=/migrations -database='postgres://bwims:bwims@postgres:5432/bwims_catalog_test?sslmode=disable' up 1
env BWIMS_TEST_DATABASE_URL='postgres://bwims:bwims@localhost:5432/bwims_catalog_test?sslmode=disable' GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run TestPostgresCatalogLifecycle -count=1
```

Expected: both migration commands succeed and the integration test passes.

- [ ] **Step 9: Commit persistence**

```bash
git add backend/migrations/000004_catalog_constraints.up.sql backend/migrations/000004_catalog_constraints.down.sql backend/internal/modules/catalog/repository.go backend/internal/modules/catalog/repository_integration_test.go
git commit -m "feat: persist catalog resources"
```

---

### Task 5: Fiber Handlers, Routes, Error Mapping, and App Wiring

**Files:**
- Create: `backend/internal/modules/catalog/handler.go`
- Create: `backend/internal/modules/catalog/handler_test.go`
- Create: `backend/internal/modules/catalog/route.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/app_test.go`

**Interfaces:**
- Consumes: Task 2/3 `Service`; Task 4 `NewPostgresRepository`; auth claims/middleware; `httpx.Success` and `httpx.NewError`.
- Produces: authenticated `/api/v1/categories`, `/api/v1/products`, and `/api/v1/products/by-barcode/:barcode` routes.

- [ ] **Step 1: Write failing handler/RBAC tests**

Use a fake Service and real Fiber/auth middleware. Test:

- missing/tampered tokens return 401;
- admin and manager POST/PUT/DELETE reach the service;
- picker/viewer POST/PUT/DELETE return 403;
- all four roles can GET lists, IDs, and barcode lookup;
- barcode route is matched before `/:product_id`;
- invalid JSON/query/UUID/cursor inputs produce the specified status/code;
- service sentinels map to every documented 403/404/409/422 response;
- unknown errors map to 500 `CATALOG_OPERATION_FAILED` without the internal message;
- DELETE returns 204;
- list results contain `items: []` rather than `null`.

- [ ] **Step 2: Run handler tests and verify RED**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog -run 'TestCatalogHTTP|TestCatalogRoutes' -count=1
```

Expected: compilation fails because Handler and routes do not exist.

- [ ] **Step 3: Implement request decoding and handlers**

Use request structs with `OptionalString` for `parent_id`, `category_id`, and `barcode`. Implement list parsing for `limit`, `after`, `search`, `is_active`, `parent_id`, and `category_id`. Parse filter UUIDs before calling the service and return `INVALID_REQUEST` for malformed values.

Return 201 for create, 200 for get/list/update, and 204 for deactivation.

- [ ] **Step 4: Implement exact HTTP error mapping**

Map all sentinels to the status/code table in the design. Use message text that names the failed resource without leaking repository details.

- [ ] **Step 5: Register routes with explicit order and RBAC**

```go
func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	mutators := []string{auth.RoleAdmin, auth.RoleWarehouseManager}

	categories := api.Group("/categories")
	categories.Get("", handler.ListCategories)
	categories.Post("", auth.RequireRoles(mutators...), handler.CreateCategory)
	categories.Get("/:category_id", handler.GetCategory)
	categories.Put("/:category_id", auth.RequireRoles(mutators...), handler.UpdateCategory)
	categories.Delete("/:category_id", auth.RequireRoles(mutators...), handler.DeactivateCategory)

	products := api.Group("/products")
	products.Get("", handler.ListProducts)
	products.Post("", auth.RequireRoles(mutators...), handler.CreateProduct)
	products.Get("/by-barcode/:barcode", handler.GetProductByBarcode)
	products.Get("/:product_id", handler.GetProduct)
	products.Put("/:product_id", auth.RequireRoles(mutators...), handler.UpdateProduct)
	products.Delete("/:product_id", auth.RequireRoles(mutators...), handler.DeactivateProduct)
}
```

- [ ] **Step 6: Wire the catalog module before fallback**

In `app.New`:

```go
catalogRepository := catalog.NewPostgresRepository(db)
catalogService := catalog.NewService(catalogRepository)
catalog.RegisterRoutes(server, catalog.NewHandler(catalogService), tokenManager)
```

Add an app test proving an authenticated catalog request does not return `ROUTE_NOT_FOUND`.

- [ ] **Step 7: Run HTTP and full backend tests**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./internal/modules/catalog ./internal/app -count=1
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit the API surface**

```bash
git add backend/internal/modules/catalog/handler.go backend/internal/modules/catalog/handler_test.go backend/internal/modules/catalog/route.go backend/internal/app/app.go backend/internal/app/app_test.go
git commit -m "feat: expose catalog APIs"
```

---

### Task 6: Documentation, Validation Evidence, and Final Verification

**Files:**
- Modify: `docs/api-contract.md`
- Modify: `docs/requirements-traceability.md`
- Create: `docs/week-8-catalog-validation.md`

**Interfaces:**
- Consumes: verified API routes and test outputs from Tasks 1-5.
- Produces: frontend/QA contract and evidence required to close PRO-9 after review and merge.

- [ ] **Step 1: Update the API contract**

Document exact Category/Product request and response examples, list filters, barcode lookup, RBAC, normalization, deactivation rules, and every error code from the design.

- [ ] **Step 2: Update requirements traceability**

Change FR-6 Product CRUD, FR-7 Categories, FR-8 barcode lookup, and FR-9 Product search from planned/schema-ready to implemented, referencing catalog service/handler/repository tests. Do not change Admin User Management or inventory movement requirements.

- [ ] **Step 3: Run backend verification**

```bash
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go test ./... -count=1
env GOCACHE=/tmp/bwims-go-cache GOMODCACHE=/tmp/bwims-go-mod go vet ./...
```

Expected: PASS with no vet output.

- [ ] **Step 4: Run frontend regression checks**

From `frontend/`:

```bash
npm test
npm run typecheck
npm run build
```

Expected: existing tests pass, typecheck succeeds, and production build succeeds.

- [ ] **Step 5: Run repository-wide checks**

From the repository root:

```bash
git diff --check
docker compose config --quiet
```

Expected: both commands exit 0 without output.

- [ ] **Step 6: Record truthful validation evidence**

In `docs/week-8-catalog-validation.md`, record the date, commit, database used, exact commands, pass/fail results, tested RBAC roles, valid test barcodes, migration apply/rollback/reapply result, and any remaining physical-scanner/frontend work. Do not claim a real hardware scan occurred.

- [ ] **Step 7: Commit documentation and evidence**

```bash
git add docs/api-contract.md docs/requirements-traceability.md docs/week-8-catalog-validation.md
git commit -m "docs: record catalog API contract"
```

- [ ] **Step 8: Final branch audit**

```bash
git status -sb
git log --oneline origin/main..HEAD
git diff --check origin/main...HEAD
```

Expected: clean working tree, only PRO-9 commits, and no whitespace errors.

After review and merge, update Linear PRO-9 to Done with the validation evidence and update Gantt task 2.2 only to partial completion because Admin User Management remains outstanding.
