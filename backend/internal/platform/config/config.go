package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Auth        AuthConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	ShutdownTimeout time.Duration
}

func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type DatabaseConfig struct {
	URL            string
	MaxConnections int32
	MinConnections int32
	ConnectTimeout time.Duration
}

type AuthConfig struct {
	Issuer               string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	PrivateKeyBase64     string
	PublicKeyBase64      string
	AllowDevelopmentKeys bool
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}
	return LoadFromEnvironment()
}

func LoadFromEnvironment() (Config, error) {
	port, err := intValue("API_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	if port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("API_PORT must be between 1 and 65535")
	}

	maxConnections, err := int32Value("DB_MAX_CONNECTIONS", 10)
	if err != nil {
		return Config{}, err
	}
	minConnections, err := int32Value("DB_MIN_CONNECTIONS", 2)
	if err != nil {
		return Config{}, err
	}
	if minConnections < 0 || maxConnections < 1 || minConnections > maxConnections {
		return Config{}, fmt.Errorf("database connection limits are invalid")
	}

	connectTimeout, err := durationValue("DB_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := durationValue("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	accessTokenTTL, err := durationValue("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	refreshTokenTTL, err := durationValue("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return Config{
		Environment: stringValue("APP_ENV", "development"),
		Server: ServerConfig{
			Host:            stringValue("API_HOST", "0.0.0.0"),
			Port:            port,
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			URL:            databaseURL,
			MaxConnections: maxConnections,
			MinConnections: minConnections,
			ConnectTimeout: connectTimeout,
		},
		Auth: AuthConfig{
			Issuer:               stringValue("JWT_ISSUER", "bwims-api"),
			AccessTokenTTL:       accessTokenTTL,
			RefreshTokenTTL:      refreshTokenTTL,
			PrivateKeyBase64:     strings.TrimSpace(os.Getenv("JWT_PRIVATE_KEY_BASE64")),
			PublicKeyBase64:      strings.TrimSpace(os.Getenv("JWT_PUBLIC_KEY_BASE64")),
			AllowDevelopmentKeys: stringValue("APP_ENV", "development") == "development",
		},
	}, nil
}

func stringValue(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intValue(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func int32Value(key string, fallback int32) (int32, error) {
	value, err := intValue(key, int(fallback))
	if err != nil {
		return 0, err
	}
	return int32(value), nil
}

func durationValue(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 5s: %w", key, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return parsed, nil
}
