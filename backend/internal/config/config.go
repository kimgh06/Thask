package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL   string
	SessionSecret string
	Port          string
	FrontendURL   string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgresql://thask:thask_dev_password@localhost:7242/thask"),
		SessionSecret: getEnv("SESSION_SECRET", "change-me-to-a-random-64-char-string"),
		Port:          getEnv("PORT", "7244"),
		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:7243"),
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
