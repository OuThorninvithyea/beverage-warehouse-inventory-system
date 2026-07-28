package auth

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

func TestAuthenticateAndRequireRoles(t *testing.T) {
	manager := testTokenManager(t)
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	app.Get(
		"/admin",
		Authenticate(manager),
		RequireRoles(RoleAdmin),
		func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) },
	)

	t.Run("missing token", func(t *testing.T) {
		response, err := app.Test(httptest.NewRequest("GET", "/admin", nil))
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusUnauthorized {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, body = %s", response.StatusCode, body)
		}
	})

	t.Run("viewer forbidden", func(t *testing.T) {
		pair, err := manager.Issue(User{ID: "viewer-id", Role: RoleViewer})
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}
		request := httptest.NewRequest("GET", "/admin", nil)
		request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status = %d, want 403", response.StatusCode)
		}
	})

	t.Run("admin allowed", func(t *testing.T) {
		pair, err := manager.Issue(User{ID: "admin-id", Role: RoleAdmin})
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}
		request := httptest.NewRequest("GET", "/admin", nil)
		request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("app.Test() error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != fiber.StatusNoContent {
			t.Fatalf("status = %d, want 204", response.StatusCode)
		}
	})
}
