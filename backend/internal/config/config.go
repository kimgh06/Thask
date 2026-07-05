package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL           string
	SessionSecret         string
	Port                  string
	FrontendURL           string
	CaptureURL            string
	CaptureInternalSecret string
	CaptureTimeoutSeconds int
	V1AllowedOrigins      []string
	MaxRequestBodyBytes   int64

	// v0.6.0 B-2 attachments (filesystem backend). AttachmentDir is where
	// bytes land inside the container; the docker-compose file mounts a
	// named volume there so uploads survive restarts. Empty AttachmentDir
	// disables the feature at the handler layer.
	AttachmentDir      string
	AttachmentMaxBytes int64
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
	captureTimeout := 30
	if v := os.Getenv("CAPTURE_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			captureTimeout = n
		}
	}

	attMax := int64(10 * 1024 * 1024) // 10 MB default
	if v := os.Getenv("THASK_ATTACHMENT_MAX_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			attMax = n
		}
	}

	return &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgresql://thask:thask_dev_password@localhost:7242/thask"),
		SessionSecret:         getEnv("SESSION_SECRET", "change-me-to-a-random-64-char-string"),
		Port:                  getEnv("PORT", "7244"),
		FrontendURL:           getEnv("FRONTEND_URL", "http://localhost:7243"),
		CaptureURL:            getEnv("CAPTURE_URL", "http://localhost:7241"),
		CaptureInternalSecret: getEnv("CAPTURE_INTERNAL_SECRET", ""),
		CaptureTimeoutSeconds: captureTimeout,
		V1AllowedOrigins:      allowedOrigins,
		MaxRequestBodyBytes:   maxBody,
		AttachmentDir:         getEnv("THASK_ATTACHMENT_DIR", ""),
		AttachmentMaxBytes:    attMax,
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
