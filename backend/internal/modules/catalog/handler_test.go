package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	err               error
	category          Category
	categories        Page[Category]
	product           Product
	products          Page[Product]
	actor             Actor
	id                string
	barcode           string
	categoryInput     CategoryInput
	productInput      ProductInput
	filter            ListFilter
	deactivateInvoked bool
}

func (s *fakeHandlerService) ListCategories(_ context.Context, actor Actor, filter ListFilter) (Page[Category], error) {
	s.actor, s.filter = actor, filter
	return s.categories, s.err
}

func (s *fakeHandlerService) GetCategory(_ context.Context, actor Actor, id string) (Category, error) {
	s.actor, s.id = actor, id
	return s.category, s.err
}

func (s *fakeHandlerService) CreateCategory(_ context.Context, actor Actor, input CategoryInput) (Category, error) {
	s.actor, s.categoryInput = actor, input
	return s.category, s.err
}

func (s *fakeHandlerService) UpdateCategory(_ context.Context, actor Actor, id string, input CategoryInput) (Category, error) {
	s.actor, s.id, s.categoryInput = actor, id, input
	return s.category, s.err
}

func (s *fakeHandlerService) DeactivateCategory(_ context.Context, actor Actor, id string) error {
	s.actor, s.id, s.deactivateInvoked = actor, id, true
	return s.err
}

func (s *fakeHandlerService) ListProducts(_ context.Context, actor Actor, filter ListFilter) (Page[Product], error) {
	s.actor, s.filter = actor, filter
	return s.products, s.err
}

func (s *fakeHandlerService) GetProduct(_ context.Context, actor Actor, id string) (Product, error) {
	s.actor, s.id = actor, id
	return s.product, s.err
}

func (s *fakeHandlerService) GetProductByBarcode(_ context.Context, actor Actor, barcode string) (Product, error) {
	s.actor, s.barcode = actor, barcode
	return s.product, s.err
}

func (s *fakeHandlerService) CreateProduct(_ context.Context, actor Actor, input ProductInput) (Product, error) {
	s.actor, s.productInput = actor, input
	return s.product, s.err
}

func (s *fakeHandlerService) UpdateProduct(_ context.Context, actor Actor, id string, input ProductInput) (Product, error) {
	s.actor, s.id, s.productInput = actor, id, input
	return s.product, s.err
}

func (s *fakeHandlerService) DeactivateProduct(_ context.Context, actor Actor, id string) error {
	s.actor, s.id, s.deactivateInvoked = actor, id, true
	return s.err
}

func TestCatalogRoutesRequireAuthentication(t *testing.T) {
	app, _ := catalogTestServer(t, &fakeHandlerService{})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/products", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}

func TestCatalogRoutesRejectTamperedAccessToken(t *testing.T) {
	app, tokens := catalogTestServer(t, &fakeHandlerService{})
	pair, err := tokens.Issue(auth.User{ID: "viewer-id", Role: auth.RoleViewer})
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+pair.AccessToken+"tampered")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}

func TestCatalogMutationRoles(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{name: "admin", role: auth.RoleAdmin, wantStatus: fiber.StatusCreated},
		{name: "warehouse manager", role: auth.RoleWarehouseManager, wantStatus: fiber.StatusCreated},
		{name: "picker", role: auth.RolePicker, wantStatus: fiber.StatusForbidden},
		{name: "viewer", role: auth.RoleViewer, wantStatus: fiber.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeHandlerService{product: Product{ID: testProductID, SKU: "COKE-330", Name: "Cola", Unit: "case"}}
			app, tokens := catalogTestServer(t, service)
			response := authenticatedCatalogRequest(t, app, tokens, auth.User{ID: "user-id", Role: tt.role},
				http.MethodPost, "/api/v1/products", `{"sku":"COKE-330","name":"Cola","unit":"case"}`)
			defer response.Body.Close()
			if response.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(response.Body)
				t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, tt.wantStatus, body)
			}
		})
	}
}

func TestCatalogAdminAndManagerCanUpdateAndDeactivate(t *testing.T) {
	for _, role := range []string{auth.RoleAdmin, auth.RoleWarehouseManager} {
		t.Run(role, func(t *testing.T) {
			service := &fakeHandlerService{category: Category{ID: testCategoryID, Name: "Soft Drinks"}}
			app, tokens := catalogTestServer(t, service)
			user := auth.User{ID: "mutator-id", Role: role}

			update := authenticatedCatalogRequest(t, app, tokens, user, http.MethodPut,
				"/api/v1/categories/"+testCategoryID, `{"name":"Soft Drinks"}`)
			defer update.Body.Close()
			if update.StatusCode != fiber.StatusOK || service.id != testCategoryID {
				t.Fatalf("update status=%d id=%q, want 200 and %s", update.StatusCode, service.id, testCategoryID)
			}

			remove := authenticatedCatalogRequest(t, app, tokens, user, http.MethodDelete,
				"/api/v1/categories/"+testCategoryID, "")
			defer remove.Body.Close()
			if remove.StatusCode != fiber.StatusNoContent || !service.deactivateInvoked {
				t.Fatalf("delete status=%d invoked=%v, want 204 true", remove.StatusCode, service.deactivateInvoked)
			}
		})
	}
}

func TestCatalogAllRolesCanRead(t *testing.T) {
	for _, role := range []string{auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker, auth.RoleViewer} {
		t.Run(role, func(t *testing.T) {
			service := &fakeHandlerService{
				categories: Page[Category]{Items: []Category{}},
				category:   Category{ID: testCategoryID, Name: "Soft Drinks"},
				product:    Product{ID: testProductID, SKU: "COKE-330", Name: "Cola", Unit: "case"},
			}
			app, tokens := catalogTestServer(t, service)
			user := auth.User{ID: "reader-id", Role: role}
			paths := []string{
				"/api/v1/categories",
				"/api/v1/categories/" + testCategoryID,
				"/api/v1/products/by-barcode/4006381333931",
			}
			for _, path := range paths {
				response := authenticatedCatalogRequest(t, app, tokens, user, http.MethodGet, path, "")
				response.Body.Close()
				if response.StatusCode != fiber.StatusOK {
					t.Fatalf("GET %s status = %d, want 200", path, response.StatusCode)
				}
			}
		})
	}
}

func TestCatalogEmptyListUsesJSONArray(t *testing.T) {
	service := &fakeHandlerService{categories: Page[Category]{Items: []Category{}}}
	app, tokens := catalogTestServer(t, service)
	response := authenticatedCatalogRequest(t, app, tokens, auth.User{ID: "viewer-id", Role: auth.RoleViewer},
		http.MethodGet, "/api/v1/categories", "")
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Contains(body, []byte(`"items":[]`)) {
		t.Fatalf("body = %s, want items array", body)
	}
}

func TestCatalogBarcodeRoutePrecedesProductIDRoute(t *testing.T) {
	service := &fakeHandlerService{product: Product{ID: testProductID, SKU: "COKE-330", Name: "Cola", Unit: "case"}}
	app, tokens := catalogTestServer(t, service)
	response := authenticatedCatalogRequest(t, app, tokens, auth.User{ID: "picker-id", Role: auth.RolePicker},
		http.MethodGet, "/api/v1/products/by-barcode/4006381333931", "")
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 200; body=%s", response.StatusCode, body)
	}
	if service.barcode != "4006381333931" || service.id != "" {
		t.Fatalf("barcode=%q id=%q, want barcode handler", service.barcode, service.id)
	}
}

func TestCatalogHandlersBindTriStateFieldsAndFilters(t *testing.T) {
	service := &fakeHandlerService{product: Product{ID: testProductID}}
	app, tokens := catalogTestServer(t, service)
	manager := auth.User{ID: "manager-id", Role: auth.RoleWarehouseManager}

	update := authenticatedCatalogRequest(t, app, tokens, manager, http.MethodPut, "/api/v1/products/"+testProductID,
		`{"category_id":null,"sku":"COKE-330","barcode":null,"name":"Cola","unit":"case","is_lot_tracked":false}`)
	defer update.Body.Close()
	if update.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(update.Body)
		t.Fatalf("update status = %d, want 200; body=%s", update.StatusCode, body)
	}
	if !service.productInput.CategoryID.Set || service.productInput.CategoryID.Value != nil ||
		!service.productInput.Barcode.Set || service.productInput.Barcode.Value != nil ||
		service.productInput.IsLotTracked == nil || *service.productInput.IsLotTracked {
		t.Fatalf("product input = %#v", service.productInput)
	}

	list := authenticatedCatalogRequest(t, app, tokens, manager, http.MethodGet,
		"/api/v1/products?limit=10&search=cola&category_id="+testCategoryID+"&is_active=false", "")
	defer list.Body.Close()
	if list.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(list.Body)
		t.Fatalf("list status = %d, want 200; body=%s", list.StatusCode, body)
	}
	if service.filter.Limit != 10 || service.filter.Search != "cola" ||
		service.filter.CategoryID == nil || *service.filter.CategoryID != testCategoryID ||
		service.filter.IsActive == nil || *service.filter.IsActive {
		t.Fatalf("filter = %#v", service.filter)
	}
}

func TestCatalogHandlersRejectMalformedInput(t *testing.T) {
	app, tokens := catalogTestServer(t, &fakeHandlerService{})
	admin := auth.User{ID: "admin-id", Role: auth.RoleAdmin}

	for _, tt := range []struct {
		name string
		path string
		body string
	}{
		{name: "malformed json", path: "/api/v1/categories", body: "{"},
		{name: "bad limit", path: "/api/v1/categories?limit=none"},
		{name: "bad active", path: "/api/v1/categories?is_active=maybe"},
		{name: "bad cursor", path: "/api/v1/categories?after=bad"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			method := http.MethodGet
			if tt.body != "" {
				method = http.MethodPost
			}
			response := authenticatedCatalogRequest(t, app, tokens, admin, method, tt.path, tt.body)
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.StatusCode)
			}
		})
	}
}

func TestCatalogHandlersMapDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", err: ErrValidation, wantStatus: 422, wantCode: "VALIDATION_ERROR"},
		{name: "invalid barcode", err: ErrInvalidBarcode, wantStatus: 422, wantCode: "INVALID_BARCODE"},
		{name: "cycle", err: ErrCategoryCycle, wantStatus: 422, wantCode: "CATEGORY_CYCLE"},
		{name: "invalid id", err: ErrInvalidID, wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "category missing", err: ErrCategoryNotFound, wantStatus: 404, wantCode: "CATEGORY_NOT_FOUND"},
		{name: "product missing", err: ErrProductNotFound, wantStatus: 404, wantCode: "PRODUCT_NOT_FOUND"},
		{name: "category name", err: ErrCategoryNameConflict, wantStatus: 409, wantCode: "CATEGORY_NAME_CONFLICT"},
		{name: "category in use", err: ErrCategoryInUse, wantStatus: 409, wantCode: "CATEGORY_IN_USE"},
		{name: "product sku", err: ErrProductSKUConflict, wantStatus: 409, wantCode: "PRODUCT_SKU_CONFLICT"},
		{name: "product barcode", err: ErrProductBarcodeConflict, wantStatus: 409, wantCode: "PRODUCT_BARCODE_CONFLICT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeHandlerService{err: tt.err}
			app, tokens := catalogTestServer(t, service)
			response := authenticatedCatalogRequest(t, app, tokens, auth.User{ID: "admin-id", Role: auth.RoleAdmin},
				http.MethodPost, "/api/v1/categories", `{"name":"Soft Drinks"}`)
			defer response.Body.Close()
			if response.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(response.Body)
				t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, tt.wantStatus, body)
			}
			var envelope httpx.Envelope
			if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if envelope.Error == nil || envelope.Error.Code != tt.wantCode {
				t.Fatalf("error = %#v, want code %s", envelope.Error, tt.wantCode)
			}
		})
	}
}

func TestCatalogHandlersHideUnexpectedErrors(t *testing.T) {
	service := &fakeHandlerService{err: errors.New("database password leaked")}
	app, tokens := catalogTestServer(t, service)
	response := authenticatedCatalogRequest(t, app, tokens, auth.User{ID: "admin-id", Role: auth.RoleAdmin},
		http.MethodPost, "/api/v1/categories", `{"name":"Soft Drinks"}`)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Contains(body, []byte(`"code":"CATALOG_OPERATION_FAILED"`)) ||
		bytes.Contains(body, []byte("database password leaked")) {
		t.Fatalf("body = %s, want safe catalog error", body)
	}
}

func catalogTestServer(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "bwims-catalog-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("auth.NewTokenManager() error = %v", err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	RegisterRoutes(app, NewHandler(service), tokens)
	return app, tokens
}

func authenticatedCatalogRequest(
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
