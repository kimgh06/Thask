package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"

	"github.com/thask/backend/internal/handler"
	"github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

type Handlers struct {
	Auth    *handler.AuthHandler
	Team    *handler.TeamHandler
	Project *handler.ProjectHandler
	Node    *handler.NodeHandler
	Edge    *handler.EdgeHandler
	Impact  *handler.ImpactHandler
	Summary *handler.SummaryHandler
	APIKey  *handler.APIKeyHandler
	Event   *handler.EventHandler
}

func RegisterRoutes(e *echo.Echo, h Handlers, sessionRepo *repository.SessionRepo, apiKeyRepo *repository.APIKeyRepo, teamRepo *repository.TeamRepo, projectRepo *repository.ProjectRepo, pmRepo *repository.ProjectMemberRepo) {
	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth routes (public)
	auth := e.Group("/api/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)

	// Auth routes (protected)
	authed := e.Group("", middleware.Auth(sessionRepo, apiKeyRepo))

	authed.GET("/api/auth/me", h.Auth.Me)
	authed.POST("/api/auth/logout", h.Auth.Logout)
	authed.POST("/api/auth/api-keys", h.APIKey.Create)
	authed.GET("/api/auth/api-keys", h.APIKey.List)
	authed.DELETE("/api/auth/api-keys/:keyId", h.APIKey.Delete)

	// Teams (no team context needed)
	authed.GET("/api/teams", h.Team.List)
	authed.POST("/api/teams", h.Team.Create)

	// Team-scoped routes (TeamAccess resolves slug + role)
	teamGroup := authed.Group("/api/teams/:teamSlug", middleware.TeamAccess(teamRepo))

	// Read — all roles
	teamGroup.GET("", h.Team.GetBySlug)
	teamGroup.GET("/members", h.Team.ListMembers)
	teamGroup.GET("/projects", h.Team.ListProjects)

	// Any member can leave
	teamGroup.POST("/leave", h.Team.Leave)

	// Admin+ — team management
	teamAdmin := teamGroup.Group("", middleware.RequireRole(model.TeamRoleAdmin))
	teamAdmin.PATCH("", h.Team.Update)
	teamAdmin.POST("/members", h.Team.InviteMember)
	teamAdmin.PATCH("/members/:userId", h.Team.UpdateMemberRole)
	teamAdmin.DELETE("/members/:userId", h.Team.RemoveMember)

	// Member+ — project creation
	teamMember := teamGroup.Group("", middleware.RequireRole(model.TeamRoleMember))
	teamMember.POST("/projects", h.Team.CreateProject)

	// Owner only — destructive operations
	teamOwner := teamGroup.Group("", middleware.RequireRole(model.TeamRoleOwner))
	teamOwner.DELETE("", h.Team.Delete)
	teamOwner.POST("/transfer", h.Team.TransferOwnership)

	// Projects (ProjectAccess resolves project + role)
	projectGroup := authed.Group("/api/projects/:projectId", middleware.ProjectAccess(projectRepo, teamRepo, pmRepo))

	// SSE — real-time events (all roles)
	projectGroup.GET("/events", h.Event.Stream)

	// Sharing (admin+)
	projectSharing := projectGroup.Group("", middleware.RequireRole(model.TeamRoleAdmin))
	projectSharing.GET("/sharing", h.Project.GetSharing)
	projectSharing.PUT("/sharing", h.Project.UpdateSharing)
	projectSharing.POST("/sharing/members", h.Project.AddMember)
	projectSharing.PATCH("/sharing/members/:userId", h.Project.UpdateMember)
	projectSharing.DELETE("/sharing/members/:userId", h.Project.RemoveMember)

	// Read — all roles (including viewer)
	projectGroup.GET("", h.Project.Get)
	projectGroup.GET("/graph", h.Node.Graph)
	projectGroup.GET("/nodes", h.Node.List)
	projectGroup.GET("/nodes/:nodeId", h.Node.Get)
	projectGroup.GET("/edges", h.Edge.List)
	projectGroup.GET("/impact", h.Impact.Analyze)

	// Write — member+ only
	projectWrite := projectGroup.Group("", middleware.RequireRole(model.TeamRoleMember))
	projectWrite.PATCH("", h.Project.Update)
	projectWrite.DELETE("", h.Project.Delete)
	projectWrite.POST("/graph/import", h.Node.Import)
	projectWrite.POST("/graph/layout", h.Node.Layout)
	projectWrite.POST("/nodes", h.Node.Create)
	projectWrite.PATCH("/nodes/:nodeId", h.Node.Update)
	projectWrite.DELETE("/nodes/:nodeId", h.Node.Delete)
	projectWrite.PATCH("/nodes/positions", h.Node.BatchUpdatePositions)
	projectWrite.POST("/nodes/batch-delete", h.Node.BatchDelete)
	projectWrite.PATCH("/nodes/batch-status", h.Node.BatchUpdateStatus)
	projectWrite.POST("/edges", h.Edge.Create)
	projectWrite.PATCH("/edges/:edgeId", h.Edge.Update)
	projectWrite.DELETE("/edges/:edgeId", h.Edge.Delete)

	// Summary
	authed.GET("/api/projects/summary", h.Summary.Get)

	// Shared (public, no auth required) — stricter rate limit for unauthenticated access
	shared := e.Group("/api/shared/:shareToken",
		echoMw.RateLimiter(echoMw.NewRateLimiterMemoryStore(5)),
		middleware.SharedAccess(projectRepo),
	)
	shared.GET("", h.Project.SharedGet)
	shared.GET("/graph", h.Node.SharedGraph)
	shared.GET("/events", h.Event.Stream)

	// Shared write — only when linkSharing == "editor" (member role)
	sharedWrite := shared.Group("", middleware.RequireRole(model.TeamRoleMember))
	sharedWrite.POST("/nodes", h.Node.Create)
	sharedWrite.PATCH("/nodes/:nodeId", h.Node.Update)
	sharedWrite.DELETE("/nodes/:nodeId", h.Node.Delete)
	sharedWrite.PATCH("/nodes/positions", h.Node.BatchUpdatePositions)
	sharedWrite.POST("/edges", h.Edge.Create)
	sharedWrite.PATCH("/edges/:edgeId", h.Edge.Update)
	sharedWrite.DELETE("/edges/:edgeId", h.Edge.Delete)
}
