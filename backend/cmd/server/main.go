package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"

	"github.com/thask/backend/internal/config"
	"github.com/thask/backend/internal/handler"
	"github.com/thask/backend/internal/migrate"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

func main() {
	cfg := config.Load()

	// Database
	ctx := context.Background()
	pool, err := repository.NewPool(ctx, cfg.DSN())
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Migrations
	if err := migrate.Run(ctx, pool, "migrations"); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	// Repositories
	userRepo := repository.NewUserRepo(pool)
	sessionRepo := repository.NewSessionRepo(pool)
	teamRepo := repository.NewTeamRepo(pool)
	projectRepo := repository.NewProjectRepo(pool)
	nodeRepo := repository.NewNodeRepo(pool)
	edgeRepo := repository.NewEdgeRepo(pool)
	historyRepo := repository.NewHistoryRepo(pool)
	apiKeyRepo := repository.NewAPIKeyRepo(pool)
	pmRepo := repository.NewProjectMemberRepo(pool)
	idempotencyRepo := repository.NewIdempotencyRepo(pool)

	// Event Hub
	hub := service.NewHub()

	// Periodic cleanup of expired idempotency keys (every hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		// Run once at startup
		if n, err := idempotencyRepo.Cleanup(ctx); err != nil {
			slog.Warn("idempotency cleanup failed", "error", err)
		} else if n > 0 {
			slog.Info("cleaned up expired idempotency keys", "count", n)
		}
		for range ticker.C {
			cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if n, err := idempotencyRepo.Cleanup(cleanCtx); err != nil {
				slog.Warn("idempotency cleanup failed", "error", err)
			} else if n > 0 {
				slog.Info("cleaned up expired idempotency keys", "count", n)
			}
			cancel()
		}
	}()

	// Handlers
	h := Handlers{
		Auth:          handler.NewAuthHandler(userRepo, sessionRepo),
		Team:          handler.NewTeamHandler(teamRepo, projectRepo, userRepo),
		Project:       handler.NewProjectHandler(projectRepo, teamRepo, pmRepo, userRepo),
		Node:          handler.NewNodeHandler(nodeRepo, edgeRepo, historyRepo, hub),
		Edge:          handler.NewEdgeHandler(edgeRepo, hub),
		Impact:        handler.NewImpactHandler(nodeRepo, edgeRepo),
		GraphAnalysis: handler.NewGraphAnalysisHandler(edgeRepo),
		Summary:       handler.NewSummaryHandler(teamRepo, projectRepo),
		APIKey:        handler.NewAPIKeyHandler(apiKeyRepo),
		Event:         handler.NewEventHandler(hub),
		Activity:      handler.NewActivityHandler(historyRepo),
	}

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.Validator = handler.NewValidator()

	// Global middleware
	e.Use(echoMw.LoggerWithConfig(echoMw.LoggerConfig{
		Format: "\033[36m${time_rfc3339}\033[0m ${method} ${uri} \033[32m${status}\033[0m ${latency_human}\n",
		Output: os.Stdout,
	}))
	e.Use(echoMw.Recover())
	// Internal CORS (for /api/ routes — frontend only)
	internalCORS := echoMw.CORSWithConfig(echoMw.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	})
	e.Use(internalCORS)
	e.Use(echoMw.RateLimiter(echoMw.NewRateLimiterMemoryStore(20)))

	// Routes
	RegisterRoutes(e, h, sessionRepo, apiKeyRepo, teamRepo, projectRepo, pmRepo)
	RegisterV1Routes(e, h, cfg, sessionRepo, apiKeyRepo, teamRepo, projectRepo, pmRepo, idempotencyRepo)

	// Graceful shutdown
	go func() {
		if err := e.Start(cfg.Addr()); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
