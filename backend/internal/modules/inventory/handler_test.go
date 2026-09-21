package inventory

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

type fakeHandlerService struct {
	receiveFn          func(Actor, ReceiveInput) (Movement, Balance, error)
	pickFn             func(Actor, PickInput) ([]Movement, error)
	listExpiryAlertsFn func(Actor, ExpiryAlertFilter) ([]ExpiryAlert, error)
}

func (f *fakeHandlerService) ListBalances(context.Context, Actor, BalanceListFilter) (Page[Balance], error) {
	return Page[Balance]{Items: []Balance{}}, nil
}
func (f *fakeHandlerService) ListLots(context.Context, Actor, string) ([]Lot, error) { return nil, nil }
func (f *fakeHandlerService) ListExpiryAlerts(_ context.Context, actor Actor, filter ExpiryAlertFilter) ([]ExpiryAlert, error) {
	if f.listExpiryAlertsFn == nil {
		return nil, nil
	}
	return f.listExpiryAlertsFn(actor, filter)
}
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
// auth.Authenticate itself (its context key is package-private, by design),
// so this is the only correct way to test a handler that reads the actor
// from context.
func newTestApp(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "inventory-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	RegisterRoutes(app, NewHandler(service), tokens)
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

func TestExpiryAlertsHandlerReturnsArrayAndForwardsTheWindow(t *testing.T) {
	var captured ExpiryAlertFilter
	service := &fakeHandlerService{listExpiryAlertsFn: func(_ Actor, filter ExpiryAlertFilter) ([]ExpiryAlert, error) {
		captured = filter
		return []ExpiryAlert{{
			SKU: "JUI-ORNG-1000", LotNumber: "L-ORNG-EXPIRED",
			ExpirationDate: "2026-09-17", DaysRemaining: -5, Status: ExpiryStatusExpired,
			Quantity: "96.000",
		}}, nil
	}}
	app, tokens := newTestApp(t, service)
	token := issueToken(t, tokens, auth.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/alerts?within_days=7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if captured.WithinDays == nil || *captured.WithinDays != 7 {
		t.Fatalf("captured.WithinDays = %v, want 7", captured.WithinDays)
	}

	var payload struct {
		Success bool          `json:"success"`
		Data    []ExpiryAlert `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response error = %v", err)
	}
	if !payload.Success || len(payload.Data) != 1 || payload.Data[0].Status != ExpiryStatusExpired {
		t.Fatalf("payload = %+v, want one expired alert", payload)
	}
}

// The literal /alerts route must win over /:product_id-style segments, the
// same ordering trap the catalog module hit with /by-barcode.
func TestExpiryAlertsRouteIsNotShadowed(t *testing.T) {
	called := false
	service := &fakeHandlerService{listExpiryAlertsFn: func(Actor, ExpiryAlertFilter) ([]ExpiryAlert, error) {
		called = true
		return nil, nil
	}}
	app, tokens := newTestApp(t, service)
	token := issueToken(t, tokens, auth.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/alerts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if _, err := app.Test(req); err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if !called {
		t.Fatal("GET /inventory/alerts did not reach the expiry alert handler")
	}
}

func TestExpiryAlertsHandlerRejectsAnInvalidWindow(t *testing.T) {
	app, tokens := newTestApp(t, &fakeHandlerService{})
	token := issueToken(t, tokens, auth.RoleAdmin)

	for _, query := range []string{"within_days=-1", "within_days=400", "within_days=soon", "limit=0", "limit=501"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/alerts?"+query, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test(%s) error = %v", query, err)
		}
		if response.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", query, response.StatusCode)
		}
	}
}

func TestExpiryAlertsHandlerRequiresAuthentication(t *testing.T) {
	app, _ := newTestApp(t, &fakeHandlerService{})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/inventory/alerts", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
}
