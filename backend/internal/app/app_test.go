package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
)

func TestWarehouseRoutesAreRegisteredBeforeNotFoundHandler(t *testing.T) {
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

	response, err := server.Test(httptest.NewRequest(http.MethodGet, "/api/v1/warehouses", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusUnauthorized {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 401 from registered authenticated route; body=%s", response.StatusCode, body)
	}
}

func TestCatalogRoutesAreRegisteredBeforeNotFoundHandler(t *testing.T) {
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

	response, err := server.Test(httptest.NewRequest(http.MethodGet, "/api/v1/products", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusUnauthorized {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 401 from registered authenticated route; body=%s", response.StatusCode, body)
	}
}

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
