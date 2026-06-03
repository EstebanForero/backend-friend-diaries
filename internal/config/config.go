package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

const (
	defaultServerPort       = "8080"
	defaultAppEnv           = "development"
	defaultDatabaseURL      = "postgres://friend_diaries:friend_diaries_pass@localhost:5433/friend_diaries?sslmode=disable"
	defaultJWTSecret        = "change-this-secret-in-production"
	defaultJWTAccessExpiry  = 72 * time.Hour        // 3 days
	defaultJWTRefreshExpiry = 30 * 24 * time.Hour   // 30 days
	defaultCORSOrigins      = "http://localhost:5173,http://tauri.localhost,https://tauri.localhost"
)

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, reading from environment variables")
	}

	jwtSecret := getEnv("JWT_SECRET", defaultJWTSecret)
	if jwtSecret == defaultJWTSecret {
		slog.Warn("using default JWT_SECRET; set JWT_SECRET before running in production")
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnvPort("SERVER_PORT", defaultServerPort),
			Env:  getEnv("APP_ENV", defaultAppEnv),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", defaultDatabaseURL),
		},
		JWT: JWTConfig{
			Secret:        jwtSecret,
			AccessExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", defaultJWTAccessExpiry),
			RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", defaultJWTRefreshExpiry),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitComma(getEnv("CORS_ALLOWED_ORIGINS", defaultCORSOrigins)),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

func getEnv(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		slog.Warn("invalid duration env var; using default", "key", key, "value", value, "default", defaultValue.String())
		return defaultValue
	}

	return duration
}

func getEnvPort(key string, defaultValue string) string {
	value := strings.TrimPrefix(getEnv(key, defaultValue), ":")
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		slog.Warn("invalid port env var; using default", "key", key, "value", value, "default", defaultValue)
		return defaultValue
	}
	return value
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := strings.TrimSpace(s[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	if len(result) == 0 {
		return splitComma(defaultCORSOrigins)
	}
	return result
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
