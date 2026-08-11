package warehouse

import (
	"bytes"
	"context"
	"encoding/json"
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

type fakeService struct {
	err               error
	warehouse         Warehouse
	warehouses        Page[Warehouse]
	location          Location
	locations         Page[Location]
	actor             Actor
	warehouseID       string
	locationID        string
	warehouseInput    WarehouseInput
	locationInput     LocationInput
	filter            ListFilter
	deactivateInvoked bool
}

func (s *fakeService) ListWarehouses(_ context.Context, actor Actor, filter ListFilter) (Page[Warehouse], error) {
	s.actor = actor
	s.filter = filter
	return s.warehouses, s.err
}

func (s *fakeService) GetWarehouse(_ context.Context, actor Actor, id string) (Warehouse, error) {
	s.actor = actor
	s.warehouseID = id
	return s.warehouse, s.err
}

func (s *fakeService) CreateWarehouse(_ context.Context, actor Actor, input WarehouseInput) (Warehouse, error) {
	s.actor = actor
	s.warehouseInput = input
	return s.warehouse, s.err
}

func (s *fakeService) UpdateWarehouse(_ context.Context, actor Actor, id string, input WarehouseInput) (Warehouse, error) {
	s.actor = actor
	s.warehouseID = id
	s.warehouseInput = input
	return s.warehouse, s.err
}

func (s *fakeService) DeactivateWarehouse(_ context.Context, actor Actor, id string) error {
	s.actor = actor
	s.warehouseID = id
	s.deactivateInvoked = true
	return s.err
}

func (s *fakeService) ListLocations(_ context.Context, actor Actor, warehouseID string, filter ListFilter) (Page[Location], error) {
	s.actor = actor
	s.warehouseID = warehouseID
	s.filter = filter
	return s.locations, s.err
}

func (s *fakeService) GetLocation(_ context.Context, actor Actor, warehouseID, locationID string) (Location, error) {
	s.actor = actor
	s.warehouseID = warehouseID
	s.locationID = locationID
	return s.location, s.err
}

func (s *fakeService) CreateLocation(_ context.Context, actor Actor, warehouseID string, input LocationInput) (Location, error) {
	s.actor = actor
	s.warehouseID = warehouseID
	s.locationInput = input
	return s.location, s.err
}

func (s *fakeService) UpdateLocation(_ context.Context, actor Actor, warehouseID, locationID string, input LocationInput) (Location, error) {
	s.actor = actor
	s.warehouseID = warehouseID
	s.locationID = locationID
	s.locationInput = input
	return s.location, s.err
}

func (s *fakeService) DeactivateLocation(_ context.Context, actor Actor, warehouseID, locationID string) error {
	s.actor = actor
	s.warehouseID = warehouseID
	s.locationID = locationID
	s.deactivateInvoked = true
	return s.err
}

func TestHandlerRequiresAuthentication(t *testing.T) {
	app, _ := warehouseTestServer(t, &fakeService{})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/warehouses", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnauthorized {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 401; body=%s", response.StatusCode, body)
	}
}

func TestHandlerRoleGatesWarehouseAndLocationMutations(t *testing.T) {
	app, tokens := warehouseTestServer(t, &fakeService{})
	assigned := testWarehouseID

	viewerResponse := authenticatedWarehouseRequest(t, app, tokens, auth.User{
		ID: "viewer-id", Role: auth.RoleViewer, WarehouseID: &assigned,
	}, http.MethodPost, "/api/v1/warehouses", `{"code":"PP-01","name":"Main"}`)
	defer viewerResponse.Body.Close()
	if viewerResponse.StatusCode != fiber.StatusForbidden {
		t.Fatalf("viewer warehouse create status = %d, want 403", viewerResponse.StatusCode)
	}

	pickerResponse := authenticatedWarehouseRequest(t, app, tokens, auth.User{
		ID: "picker-id", Role: auth.RolePicker, WarehouseID: &assigned,
	}, http.MethodPost, "/api/v1/warehouses/"+assigned+"/locations", `{"code":"A-01"}`)
	defer pickerResponse.Body.Close()
	if pickerResponse.StatusCode != fiber.StatusForbidden {
		t.Fatalf("picker location create status = %d, want 403", pickerResponse.StatusCode)
	}
}

func TestHandlerCreatesWarehouseWithAdminClaims(t *testing.T) {
	service := &fakeService{warehouse: Warehouse{ID: testWarehouseID, Code: "PP-01", Name: "Main", IsActive: true}}
	app, tokens := warehouseTestServer(t, service)

	response := authenticatedWarehouseRequest(t, app, tokens, auth.User{
		ID: "admin-id", Role: auth.RoleAdmin,
	}, http.MethodPost, "/api/v1/warehouses", `{"code":"PP-01","name":"Main"}`)
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 201; body=%s", response.StatusCode, body)
	}
	if service.actor.Role != auth.RoleAdmin || service.warehouseInput.Code != "PP-01" {
		t.Fatalf("service call actor=%#v input=%#v", service.actor, service.warehouseInput)
	}
}

func TestHandlerRejectsMalformedJSONAndQueryValues(t *testing.T) {
	app, tokens := warehouseTestServer(t, &fakeService{})
	admin := auth.User{ID: "admin-id", Role: auth.RoleAdmin}

	badJSON := authenticatedWarehouseRequest(t, app, tokens, admin, http.MethodPost, "/api/v1/warehouses", `{`)
	defer badJSON.Body.Close()
	if badJSON.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("malformed JSON status = %d, want 400", badJSON.StatusCode)
	}

	badLimit := authenticatedWarehouseRequest(t, app, tokens, admin, http.MethodGet, "/api/v1/warehouses?limit=zero", "")
	defer badLimit.Body.Close()
	if badLimit.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("invalid limit status = %d, want 400", badLimit.StatusCode)
	}

	badActive := authenticatedWarehouseRequest(t, app, tokens, admin, http.MethodGet, "/api/v1/warehouses?is_active=maybe", "")
	defer badActive.Body.Close()
	if badActive.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("invalid is_active status = %d, want 400", badActive.StatusCode)
	}
}

func TestHandlerMapsDomainErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", err: ErrValidation, wantStatus: 422, wantCode: "VALIDATION_ERROR"},
		{name: "forbidden", err: ErrForbidden, wantStatus: 403, wantCode: "FORBIDDEN"},
		{name: "invalid id", err: ErrInvalidID, wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "warehouse missing", err: ErrWarehouseNotFound, wantStatus: 404, wantCode: "WAREHOUSE_NOT_FOUND"},
		{name: "warehouse code", err: ErrWarehouseCodeConflict, wantStatus: 409, wantCode: "WAREHOUSE_CODE_CONFLICT"},
		{name: "location code", err: ErrLocationCodeConflict, wantStatus: 409, wantCode: "LOCATION_CODE_CONFLICT"},
		{name: "location barcode", err: ErrLocationBarcodeConflict, wantStatus: 409, wantCode: "LOCATION_BARCODE_CONFLICT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeService{err: tt.err}
			app, tokens := warehouseTestServer(t, service)
			response := authenticatedWarehouseRequest(t, app, tokens, auth.User{
				ID: "admin-id", Role: auth.RoleAdmin,
			}, http.MethodPost, "/api/v1/warehouses", `{"code":"PP-01","name":"Main"}`)
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

func TestHandlerListsLocationsAndDeactivatesWarehouse(t *testing.T) {
	assigned := testWarehouseID
	service := &fakeService{locations: Page[Location]{
		Items: []Location{{ID: testLocationID, WarehouseID: assigned, Code: "A-01"}},
		Page:  PageInfo{HasMore: false},
	}}
	app, tokens := warehouseTestServer(t, service)
	manager := auth.User{ID: "manager-id", Role: auth.RoleWarehouseManager, WarehouseID: &assigned}

	listResponse := authenticatedWarehouseRequest(t, app, tokens, manager, http.MethodGet,
		"/api/v1/warehouses/"+assigned+"/locations?limit=10&is_pickable=true&search=rack", "")
	defer listResponse.Body.Close()
	if listResponse.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(listResponse.Body)
		t.Fatalf("list status = %d, want 200; body=%s", listResponse.StatusCode, body)
	}
	if service.filter.Limit != 10 || service.filter.IsPickable == nil || !*service.filter.IsPickable {
		t.Fatalf("filter = %#v", service.filter)
	}

	deleteResponse := authenticatedWarehouseRequest(t, app, tokens, auth.User{
		ID: "admin-id", Role: auth.RoleAdmin,
	}, http.MethodDelete, "/api/v1/warehouses/"+assigned, "")
	defer deleteResponse.Body.Close()
	if deleteResponse.StatusCode != fiber.StatusNoContent || !service.deactivateInvoked {
		t.Fatalf("delete status=%d invoked=%v, want 204 true", deleteResponse.StatusCode, service.deactivateInvoked)
	}
}

func warehouseTestServer(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "bwims-warehouse-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("auth.NewTokenManager() error = %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	RegisterRoutes(app, NewHandler(service), tokens)
	return app, tokens
}

func authenticatedWarehouseRequest(
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
