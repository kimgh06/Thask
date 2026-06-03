package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// Version is the backend build version. Overridden at build time via
//
//	-ldflags "-X github.com/thask/backend/internal/handler.Version=X.Y.Z"
var Version = "dev"

// HealthHandler reports liveness and structural readiness so the CLI (`thask
// doctor`) and external monitors can distinguish "server up, DB down" from
// "server up, migrations pending" from "fully healthy" without parsing logs.
type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

type HealthResponse struct {
	Status            string `json:"status"` // ok | degraded | down
	Version           string `json:"version"`
	DB                string `json:"db"` // ok | error
	DBError           string `json:"dbError,omitempty"`
	MigrationVersion  int    `json:"migrationVersion"`
	MigrationsApplied int    `json:"migrationsApplied"`
	Uptime            string `json:"uptime"`
}

var startedAt = time.Now()

func (h *HealthHandler) Get(c echo.Context) error {
	resp := HealthResponse{
		Status:  "ok",
		Version: Version,
		DB:      "ok",
		Uptime:  time.Since(startedAt).Round(time.Second).String(),
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		// Unauthenticated endpoint — never leak DSN / hostname / SSL paths.
		slog.Error("health: db ping failed", "error", err)
		resp.Status = "down"
		resp.DB = "error"
		resp.DBError = "db ping failed"
		return c.JSON(http.StatusServiceUnavailable, resp)
	}

	// schema_migrations is created by migration 001 so a missing/zero result
	// means migrations have not been run.
	var maxVersion int
	var count int
	if err := h.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(version),0), COUNT(*) FROM schema_migrations`,
	).Scan(&maxVersion, &count); err != nil {
		slog.Error("health: schema_migrations unreadable", "error", err)
		resp.Status = "degraded"
		resp.DB = "ok"
		resp.DBError = "schema_migrations unreadable"
		return c.JSON(http.StatusOK, resp)
	}
	resp.MigrationVersion = maxVersion
	resp.MigrationsApplied = count

	return c.JSON(http.StatusOK, resp)
}
