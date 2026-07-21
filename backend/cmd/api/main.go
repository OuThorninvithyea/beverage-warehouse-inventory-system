package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/app"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/config"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}

	connectCtx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer cancel()

	db, err := database.NewPostgresPool(connectCtx, cfg.Database)
	if err != nil {
		logger.Error("connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	server := app.New(cfg, db, logger)
	go waitForShutdown(server, cfg.Server.ShutdownTimeout, logger)

	logger.Info("starting API", "environment", cfg.Environment, "address", cfg.Server.Address())
	if err := server.Listen(cfg.Server.Address()); err != nil && !errors.Is(err, fiber.ErrServiceUnavailable) {
		logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func waitForShutdown(server *fiber.App, timeout time.Duration, logger *slog.Logger) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	<-signals
	logger.Info("shutting down API")
	if err := server.ShutdownWithTimeout(timeout); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
