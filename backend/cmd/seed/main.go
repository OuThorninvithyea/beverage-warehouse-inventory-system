package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
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
	if cfg.Environment != "development" {
		logger.Error("development seed refused", "environment", cfg.Environment)
		os.Exit(1)
	}

	email := strings.TrimSpace(os.Getenv("SEED_ADMIN_EMAIL"))
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	fullName := strings.TrimSpace(os.Getenv("SEED_ADMIN_NAME"))
	if email == "" || password == "" || fullName == "" {
		logger.Error("SEED_ADMIN_EMAIL, SEED_ADMIN_PASSWORD and SEED_ADMIN_NAME are required")
		os.Exit(1)
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		logger.Error("validate seed password", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		logger.Error("connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repository := auth.NewPostgresRepository(pool)
	if err := repository.EnsureDevelopmentAdmin(ctx, email, passwordHash, fullName); err != nil {
		logger.Error("seed administrator", "error", err)
		os.Exit(1)
	}

	logger.Info("development administrator ready", "email", email)

	if !demoDataEnabled() {
		logger.Info("demo data skipped", "reason", "SEED_DEMO_DATA is disabled")
		return
	}
	if err := seedDemoData(ctx, pool, logger, demoPassword()); err != nil {
		logger.Error("seed demo data", "error", err)
		os.Exit(1)
	}
}

// demoDataEnabled reports whether the catalog and inventory demo dataset should
// be loaded after the administrator. It is on by default because the seeder
// already refuses to run outside development.
func demoDataEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SEED_DEMO_DATA"))) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func demoPassword() string {
	if password := os.Getenv("SEED_DEMO_PASSWORD"); password != "" {
		return password
	}
	return "DemoPass123!"
}
