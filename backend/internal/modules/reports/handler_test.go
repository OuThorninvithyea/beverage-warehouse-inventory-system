package reports

import (
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
	captured Filter
}

func (f *fakeHandlerService) Dashboard(_ context.Context, _ Actor, filter Filter) (Dashboard, error) {
	f.captured = filter
	return Dashboard{MovementsByType: map[string]int{"pick": 3}, MovementWindow: filter.Days}, nil
}
func (f *fakeHandlerService) Valuation(_ context.Context, _ Actor, filter Filter) (Valuation, error) {
	f.captured = filter
	return Valuation{TotalValue: "1234.5600", Rows: []ValuationRow{}}, nil
}
func (f *fakeHandlerService) MovementSummary(_ context.Context, _ Actor, filter Filter) (MovementSummary, error) {
	f.captured = filter
	return MovementSummary{WindowDays: filter.Days, Totals: map[string]int{}, Rows: []MovementSummaryRow{}}, nil
}
func (f *fakeHandlerService) Velocity(_ context.Context, _ Actor, filter Filter) (Velocity, error) {
	f.captured = filter
	return Velocity{WindowDays: filter.Days, Rows: []VelocityRow{}}, nil
}

func newTestApp(t *testing.T, service Service) (*fiber.App, *auth.TokenManager) {
	t.Helper()
	tokens, err := auth.NewTokenManager(config.AuthConfig{
		Issuer: "reports-test", AccessTokenTTL: time.Minute,
		RefreshTokenTTL: time.Hour, AllowDevelopmentKeys: true,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
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

func request(t *testing.T, app *fiber.App, target, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(%s) error = %v", target, err)
	}
	return response
}

var reportRoutes = []string{
	"/api/v1/reports/dashboard",
	"/api/v1/reports/valuation",
	"/api/v1/reports/movement-summary",
	"/api/v1/reports/velocity",
}

func TestReportRoutesRequireAuthentication(t *testing.T) {
	app, _ := newTestApp(t, &fakeHandlerService{})
	for _, route := range reportRoutes {
		if status := request(t, app, route, "").StatusCode; status != fiber.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", route, status)
		}
	}
}

func TestReportRoutesRejectPickerAndViewerAtTheRoute(t *testing.T) {
	app, tokens := newTestApp(t, &fakeHandlerService{})
	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		token := issueToken(t, tokens, role)
		for _, route := range reportRoutes {
			if status := request(t, app, route, token).StatusCode; status != fiber.StatusForbidden {
				t.Errorf("%s as %s: status = %d, want 403", route, role, status)
			}
		}
	}
}

func TestReportRoutesAllowAdminAndManager(t *testing.T) {
	app, tokens := newTestApp(t, &fakeHandlerService{})
	for _, role := range []string{auth.RoleAdmin, auth.RoleWarehouseManager} {
		token := issueToken(t, tokens, role)
		for _, route := range reportRoutes {
			if status := request(t, app, route, token).StatusCode; status != fiber.StatusOK {
				t.Errorf("%s as %s: status = %d, want 200", route, role, status)
			}
		}
	}
}

func TestDashboardHandlerForwardsTheWindow(t *testing.T) {
	service := &fakeHandlerService{}
	app, tokens := newTestApp(t, service)
	token := issueToken(t, tokens, auth.RoleAdmin)

	response := request(t, app, "/api/v1/reports/dashboard?days=7", token)
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if service.captured.Days != 7 {
		t.Fatalf("captured.Days = %d, want 7", service.captured.Days)
	}

	var payload struct {
		Success bool      `json:"success"`
		Data    Dashboard `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if !payload.Success || payload.Data.MovementsByType["pick"] != 3 {
		t.Fatalf("payload = %+v, want the service result passed through", payload)
	}
}

func TestReportHandlersRejectInvalidQueries(t *testing.T) {
	app, tokens := newTestApp(t, &fakeHandlerService{})
	token := issueToken(t, tokens, auth.RoleAdmin)

	for _, query := range []string{"days=0", "days=400", "days=soon", "limit=0", "limit=501", "limit=many"} {
		if status := request(t, app, "/api/v1/reports/velocity?"+query, token).StatusCode; status != fiber.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", query, status)
		}
	}
}
