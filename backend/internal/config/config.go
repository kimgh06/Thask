package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL         string
	SessionSecret       string
	Port                string
	FrontendURL         string
	V1AllowedOrigins    []string
	MaxRequestBodyBytes int64
}

func Load() *Config {
	// V1_ALLOWED_ORIGINS: comma-separated list of allowed origins for /api/v1/ CORS.
	// Default "*" allows all origins (suitable for development).
	// In production, set to specific domains (e.g., "https://app.example.com,https://other.example.com").
	origins := getEnv("V1_ALLOWED_ORIGINS", "*")
	if origins == "*" {
		slog.Warn("V1_ALLOWED_ORIGINS is wildcard '*' — restrict in production via V1_ALLOWED_ORIGINS env var")
	}
	var allowedOrigins []string
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowedOrigins = append(allowedOrigins, o)
		}
	}

	maxBody := int64(1048576) // 1MB default
	if v := os.Getenv("MAX_REQUEST_BODY_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			maxBody = n
		}
	}

	return &Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgresql://thask:thask_dev_password@localhost:7242/thask"),
		SessionSecret:       getEnv("SESSION_SECRET", "change-me-to-a-random-64-char-string"),
		Port:                getEnv("PORT", "7244"),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:7243"),
		V1AllowedOrigins:    allowedOrigins,
		MaxRequestBodyBytes: maxBody,
	}
}

func (c *Config) DSN() string {
	return c.DatabaseURL
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
