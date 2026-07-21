package app

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/health"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

// New is the composition root. Dependencies are created once in main and wired
// into modules here, which keeps handlers and services easy to test.
func New(cfg config.Config, db *pgxpool.Pool, appLogger *slog.Logger) *fiber.App {
	server := fiber.New(fiber.Config{
		AppName:      "BWIMS API",
		ErrorHandler: httpx.ErrorHandler(appLogger),
	})

	server.Use(recover.New())
	server.Use(logger.New())

	healthRepository := health.NewPostgresRepository(db)
	healthService := health.NewService(healthRepository, cfg.Environment)
	health.RegisterRoutes(server, health.NewHandler(healthService))

	server.Use(func(c fiber.Ctx) error {
		return httpx.NewError(fiber.StatusNotFound, "ROUTE_NOT_FOUND", "route not found", nil)
	})

	return server
}
