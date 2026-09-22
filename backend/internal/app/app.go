package app

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/catalog"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/health"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/inventory"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/reports"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/users"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/warehouse"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

// New is the composition root. Dependencies are created once in main and wired
// into modules here, which keeps handlers and services easy to test.
func New(cfg config.Config, db *pgxpool.Pool, appLogger *slog.Logger) (*fiber.App, error) {
	server := fiber.New(fiber.Config{
		AppName:      "BWIMS API",
		ErrorHandler: httpx.ErrorHandler(appLogger),
	})

	server.Use(recover.New())
	server.Use(logger.New())

	healthRepository := health.NewPostgresRepository(db)
	healthService := health.NewService(healthRepository, cfg.Environment)
	health.RegisterRoutes(server, health.NewHandler(healthService))

	tokenManager, err := auth.NewTokenManager(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	authRepository := auth.NewPostgresRepository(db)
	authService := auth.NewService(authRepository, tokenManager)
	auth.RegisterRoutes(server, auth.NewHandler(authService), tokenManager)

	catalogRepository := catalog.NewPostgresRepository(db)
	catalogService := catalog.NewService(catalogRepository)
	catalog.RegisterRoutes(server, catalog.NewHandler(catalogService), tokenManager)

	warehouseRepository := warehouse.NewPostgresRepository(db)
	warehouseService := warehouse.NewService(warehouseRepository)
	warehouse.RegisterRoutes(server, warehouse.NewHandler(warehouseService), tokenManager)

	usersRepository := users.NewPostgresRepository(db)
	usersService := users.NewService(usersRepository)
	users.RegisterRoutes(server, users.NewHandler(usersService), tokenManager)

	inventoryRepository := inventory.NewPostgresRepository(db)
	inventoryService := inventory.NewService(inventoryRepository)
	inventory.RegisterRoutes(server, inventory.NewHandler(inventoryService), tokenManager)

	reportsRepository := reports.NewPostgresRepository(db)
	reportsService := reports.NewService(reportsRepository)
	reports.RegisterRoutes(server, reports.NewHandler(reportsService), tokenManager)

	server.Use(func(c fiber.Ctx) error {
		return httpx.NewError(fiber.StatusNotFound, "ROUTE_NOT_FOUND", "route not found", nil)
	})

	return server, nil
}
