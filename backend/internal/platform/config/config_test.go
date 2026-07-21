package config

import (
	"testing"
	"time"
)

func TestLoadFromEnvironmentUsesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("APP_ENV", "")
	t.Setenv("API_HOST", "")
	t.Setenv("API_PORT", "")
	t.Setenv("DB_MAX_CONNECTIONS", "")
	t.Setenv("DB_MIN_CONNECTIONS", "")
	t.Setenv("DB_CONNECT_TIMEOUT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")

	cfg, err := LoadFromEnvironment()
	if err != nil {
		t.Fatalf("LoadFromEnvironment() error = %v", err)
	}
	if cfg.Environment != "development" || cfg.Server.Address() != "0.0.0.0:8080" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.Database.ConnectTimeout != 5*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 5s", cfg.Database.ConnectTimeout)
	}
}

func TestLoadFromEnvironmentRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := LoadFromEnvironment(); err == nil {
		t.Fatal("LoadFromEnvironment() error = nil, want DATABASE_URL error")
	}
}

func TestLoadFromEnvironmentRejectsInvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("API_PORT", "70000")

	if _, err := LoadFromEnvironment(); err == nil {
		t.Fatal("LoadFromEnvironment() error = nil, want port validation error")
	}
}
