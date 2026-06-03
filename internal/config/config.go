package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
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

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, reading from environment variables")
	}

	jwtAccessExpiry, err := parseDuration(getEnv("JWT_ACCESS_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_EXPIRY: %w", err)
	}

	jwtRefreshExpiry, err := parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}

	allowedOrigins := splitComma(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"))

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://friend_diaries:friend_diaries_pass@localhost:5432/friend_diaries?sslmode=disable"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "change-this-secret-in-production"),
			AccessExpiry:  jwtAccessExpiry,
			RefreshExpiry: jwtRefreshExpiry,
		},
		CORS: CORSConfig{
			AllowedOrigins: allowedOrigins,
		},
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
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
