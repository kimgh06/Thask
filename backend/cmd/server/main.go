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
	"github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
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

	// Run migrations
	migrationSQL, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		slog.Warn("migration file not found, skipping", "error", err)
	} else {
		if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
			slog.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied")
	}

	// Run migration 002 (idempotent — only if schema_migrations tracking says not applied)
	var m002Applied bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 2)`).Scan(&m002Applied)
	if !m002Applied {
		migration002, err := os.ReadFile("migrations/002_api_keys.sql")
		if err != nil {
			slog.Warn("migration 002 file not found, skipping", "error", err)
		} else {
			if _, err := pool.Exec(ctx, string(migration002)); err != nil {
				slog.Error("failed to run migration 002", "error", err)
				os.Exit(1)
			}
			pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES (2) ON CONFLICT DO NOTHING`)
			slog.Info("migration 002 applied")
		}
	}

	// Run migration 003
	var m003Applied bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 3)`).Scan(&m003Applied)
	if !m003Applied {
		migration003, err := os.ReadFile("migrations/003_project_access.sql")
		if err != nil {
			slog.Warn("migration 003 file not found, skipping", "error", err)
		} else {
			if _, err := pool.Exec(ctx, string(migration003)); err != nil {
				slog.Error("failed to run migration 003", "error", err)
				os.Exit(1)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES (3) ON CONFLICT DO NOTHING`); err != nil {
				slog.Warn("failed to record migration 003", "error", err)
			}
			slog.Info("migration 003 applied")
		}
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

	// Event Hub
	hub := service.NewHub()

	// Handlers
	authHandler := handler.NewAuthHandler(userRepo, sessionRepo)
	teamHandler := handler.NewTeamHandler(teamRepo, projectRepo, userRepo)
	projectHandler := handler.NewProjectHandler(projectRepo, teamRepo, pmRepo, userRepo)
	nodeHandler := handler.NewNodeHandler(nodeRepo, edgeRepo, historyRepo, hub)
	edgeHandler := handler.NewEdgeHandler(edgeRepo, hub)
	impactHandler := handler.NewImpactHandler(nodeRepo, edgeRepo)
	summaryHandler := handler.NewSummaryHandler(teamRepo, projectRepo)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyRepo)
	eventHandler := handler.NewEventHandler(hub)

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
	e.Use(echoMw.CORSWithConfig(echoMw.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))
	e.Use(echoMw.RateLimiter(echoMw.NewRateLimiterMemoryStore(20)))

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth routes (public)
	auth := e.Group("/api/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	// Auth routes (protected)
	authed := e.Group("", middleware.Auth(sessionRepo, apiKeyRepo))

	authed.GET("/api/auth/me", authHandler.Me)
	authed.POST("/api/auth/logout", authHandler.Logout)
	authed.POST("/api/auth/api-keys", apiKeyHandler.Create)
	authed.GET("/api/auth/api-keys", apiKeyHandler.List)
	authed.DELETE("/api/auth/api-keys/:keyId", apiKeyHandler.Delete)

	// Teams (no team context needed)
	authed.GET("/api/teams", teamHandler.List)
	authed.POST("/api/teams", teamHandler.Create)

	// Team-scoped routes (TeamAccess resolves slug + role)
	teamGroup := authed.Group("/api/teams/:teamSlug", middleware.TeamAccess(teamRepo))

	// Read — all roles
	teamGroup.GET("", teamHandler.GetBySlug)
	teamGroup.GET("/members", teamHandler.ListMembers)
	teamGroup.GET("/projects", teamHandler.ListProjects)

	// Any member can leave
	teamGroup.POST("/leave", teamHandler.Leave)

	// Admin+ — team management
	teamAdmin := teamGroup.Group("", middleware.RequireRole(model.TeamRoleAdmin))
	teamAdmin.PATCH("", teamHandler.Update)
	teamAdmin.POST("/members", teamHandler.InviteMember)
	teamAdmin.PATCH("/members/:userId", teamHandler.UpdateMemberRole)
	teamAdmin.DELETE("/members/:userId", teamHandler.RemoveMember)

	// Member+ — project creation
	teamMember := teamGroup.Group("", middleware.RequireRole(model.TeamRoleMember))
	teamMember.POST("/projects", teamHandler.CreateProject)

	// Owner only — destructive operations
	teamOwner := teamGroup.Group("", middleware.RequireRole(model.TeamRoleOwner))
	teamOwner.DELETE("", teamHandler.Delete)
	teamOwner.POST("/transfer", teamHandler.TransferOwnership)

	// Projects (ProjectAccess resolves project + role)
	projectGroup := authed.Group("/api/projects/:projectId", middleware.ProjectAccess(projectRepo, teamRepo, pmRepo))

	// SSE — real-time events (all roles)
	projectGroup.GET("/events", eventHandler.Stream)

	// Sharing (admin+)
	projectSharing := projectGroup.Group("", middleware.RequireRole(model.TeamRoleAdmin))
	projectSharing.GET("/sharing", projectHandler.GetSharing)
	projectSharing.PUT("/sharing", projectHandler.UpdateSharing)
	projectSharing.POST("/sharing/members", projectHandler.AddMember)
	projectSharing.PATCH("/sharing/members/:userId", projectHandler.UpdateMember)
	projectSharing.DELETE("/sharing/members/:userId", projectHandler.RemoveMember)

	// Read — all roles (including viewer)
	projectGroup.GET("", projectHandler.Get)
	projectGroup.GET("/graph", nodeHandler.Graph)
	projectGroup.GET("/nodes", nodeHandler.List)
	projectGroup.GET("/nodes/:nodeId", nodeHandler.Get)
	projectGroup.GET("/edges", edgeHandler.List)
	projectGroup.GET("/impact", impactHandler.Analyze)

	// Write — member+ only
	projectWrite := projectGroup.Group("", middleware.RequireRole(model.TeamRoleMember))
	projectWrite.PATCH("", projectHandler.Update)
	projectWrite.DELETE("", projectHandler.Delete)
	projectWrite.POST("/graph/import", nodeHandler.Import)
	projectWrite.POST("/graph/layout", nodeHandler.Layout)
	projectWrite.POST("/nodes", nodeHandler.Create)
	projectWrite.PATCH("/nodes/:nodeId", nodeHandler.Update)
	projectWrite.DELETE("/nodes/:nodeId", nodeHandler.Delete)
	projectWrite.PATCH("/nodes/positions", nodeHandler.BatchUpdatePositions)
	projectWrite.POST("/nodes/batch-delete", nodeHandler.BatchDelete)
	projectWrite.PATCH("/nodes/batch-status", nodeHandler.BatchUpdateStatus)
	projectWrite.POST("/edges", edgeHandler.Create)
	projectWrite.PATCH("/edges/:edgeId", edgeHandler.Update)
	projectWrite.DELETE("/edges/:edgeId", edgeHandler.Delete)

	// Summary
	authed.GET("/api/projects/summary", summaryHandler.Get)

	// Shared (public, no auth required) — stricter rate limit for unauthenticated access
	shared := e.Group("/api/shared/:shareToken",
		echoMw.RateLimiter(echoMw.NewRateLimiterMemoryStore(5)),
		middleware.SharedAccess(projectRepo),
	)
	shared.GET("", projectHandler.SharedGet)
	shared.GET("/graph", nodeHandler.SharedGraph)

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
