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
	"github.com/thask/backend/internal/repository"
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

	// Repositories
	userRepo := repository.NewUserRepo(pool)
	sessionRepo := repository.NewSessionRepo(pool)
	teamRepo := repository.NewTeamRepo(pool)
	projectRepo := repository.NewProjectRepo(pool)
	nodeRepo := repository.NewNodeRepo(pool)
	edgeRepo := repository.NewEdgeRepo(pool)
	historyRepo := repository.NewHistoryRepo(pool)

	// Handlers
	authHandler := handler.NewAuthHandler(userRepo, sessionRepo)
	teamHandler := handler.NewTeamHandler(teamRepo, projectRepo, userRepo)
	projectHandler := handler.NewProjectHandler(projectRepo, teamRepo)
	nodeHandler := handler.NewNodeHandler(nodeRepo, edgeRepo, historyRepo)
	edgeHandler := handler.NewEdgeHandler(edgeRepo)
	impactHandler := handler.NewImpactHandler(nodeRepo, edgeRepo)
	summaryHandler := handler.NewSummaryHandler(teamRepo, projectRepo)

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
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
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
	authed := e.Group("", middleware.Auth(sessionRepo))

	authed.GET("/api/auth/me", authHandler.Me)
	authed.POST("/api/auth/logout", authHandler.Logout)

	// Teams
	authed.GET("/api/teams", teamHandler.List)
	authed.POST("/api/teams", teamHandler.Create)
	authed.GET("/api/teams/:teamSlug", teamHandler.GetBySlug)
	authed.DELETE("/api/teams/:teamSlug", teamHandler.Delete)
	authed.GET("/api/teams/:teamSlug/members", teamHandler.ListMembers)
	authed.POST("/api/teams/:teamSlug/members", teamHandler.InviteMember)
	authed.GET("/api/teams/:teamSlug/projects", teamHandler.ListProjects)
	authed.POST("/api/teams/:teamSlug/projects", teamHandler.CreateProject)

	// Projects
	projectGroup := authed.Group("/api/projects/:projectId", middleware.ProjectAccess(projectRepo))
	projectGroup.GET("", projectHandler.Get)
	projectGroup.PATCH("", projectHandler.Update)
	projectGroup.DELETE("", projectHandler.Delete)

	// Nodes
	projectGroup.GET("/nodes", nodeHandler.List)
	projectGroup.POST("/nodes", nodeHandler.Create)
	projectGroup.GET("/nodes/:nodeId", nodeHandler.Get)
	projectGroup.PATCH("/nodes/:nodeId", nodeHandler.Update)
	projectGroup.DELETE("/nodes/:nodeId", nodeHandler.Delete)
	projectGroup.PATCH("/nodes/positions", nodeHandler.BatchUpdatePositions)
	projectGroup.POST("/nodes/batch-delete", nodeHandler.BatchDelete)
	projectGroup.PATCH("/nodes/batch-status", nodeHandler.BatchUpdateStatus)

	// Edges
	projectGroup.GET("/edges", edgeHandler.List)
	projectGroup.POST("/edges", edgeHandler.Create)
	projectGroup.PATCH("/edges/:edgeId", edgeHandler.Update)
	projectGroup.DELETE("/edges/:edgeId", edgeHandler.Delete)

	// Impact
	projectGroup.GET("/impact", impactHandler.Analyze)

	// Summary
	authed.GET("/api/projects/summary", summaryHandler.Get)

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
