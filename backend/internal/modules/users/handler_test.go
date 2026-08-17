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
