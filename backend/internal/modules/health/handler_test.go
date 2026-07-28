package health

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

func TestLivenessHandler(t *testing.T) {
	server := testServer(fakeRepository{})
	response, err := server.Test(httptest.NewRequest("GET", "/health", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}

	var body httpx.Envelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success {
		t.Fatal("success = false, want true")
	}
}

func TestReadinessHandlerReturnsServiceUnavailable(t *testing.T) {
	server := testServer(fakeRepository{err: errors.New("database down")})
	response, err := server.Test(httptest.NewRequest("GET", "/ready", nil))
	if err != nil {
		t.Fatalf("server.Test() error = %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusServiceUnavailable {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, fiber.StatusServiceUnavailable, body)
	}
}

func testServer(repository Repository) *fiber.App {
	server := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	RegisterRoutes(server, NewHandler(NewService(repository, "test")))
	return server
}

var _ Repository = fakeRepository{}
