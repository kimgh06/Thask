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
	Auth          *handler.AuthHandler
	Team          *handler.TeamHandler
	Project       *handler.ProjectHandler
	Node          *handler.NodeHandler
	Edge          *handler.EdgeHandler
	Impact        *handler.ImpactHandler
	GraphAnalysis *handler.GraphAnalysisHandler
	Summary       *handler.SummaryHandler
	APIKey        *handler.APIKeyHandler
	Event         *handler.EventHandler
	Activity      *handler.ActivityHandler
	Suggestion    *handler.SuggestionHandler
	KnowledgeOS   *handler.KnowledgeOSHandler
	Health        *handler.HealthHandler
}

func RegisterRoutes(e *echo.Echo, h Handlers, sessionRepo *repository.SessionRepo, apiKeyRepo *repository.APIKeyRepo, teamRepo *repository.TeamRepo, projectRepo *repository.ProjectRepo, pmRepo *repository.ProjectMemberRepo) {
	// Health check — DB ping + migration version + build info so `thask
	// doctor` and external monitors can distinguish liveness from readiness.
	if h.Health != nil {
		e.GET("/health", h.Health.Get)
		e.GET("/api/health", h.Health.Get)
	} else {
		e.GET("/health", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
	}

	// Auth routes (public)
	auth := e.Group("/api/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)

	// Auth routes (protected)
	authed := e.Group("", middleware.Auth(sessionRepo, apiKeyRepo))

	// A project-scoped API key (api_keys.project_id != NULL) must not reach
	// account / team routes — otherwise it could mint an unscoped key or
	// modify unrelated teams (self-escalation). Cookie-auth calls have no
	// scope set and pass through freely.
	accountRoutes := authed.Group("", middleware.RejectScopedKey())

	accountRoutes.GET("/api/auth/me", h.Auth.Me)
	accountRoutes.POST("/api/auth/logout", h.Auth.Logout)
	accountRoutes.POST("/api/auth/api-keys", h.APIKey.Create)
	accountRoutes.GET("/api/auth/api-keys", h.APIKey.List)
	accountRoutes.DELETE("/api/auth/api-keys/:keyId", h.APIKey.Delete)

	// Teams (no team context needed)
	accountRoutes.GET("/api/teams", h.Team.List)
	accountRoutes.POST("/api/teams", h.Team.Create)

	// Team-scoped routes (TeamAccess resolves slug + role). Also gated on
	// RejectScopedKey — a project-scoped key has no business managing teams
	// or minting new team members.
	teamGroup := accountRoutes.Group("/api/teams/:teamSlug", middleware.TeamAccess(teamRepo))

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
	projectGroup.GET("/graph/analyze", h.GraphAnalysis.Analyze)
	projectGroup.GET("/activity", h.Activity.List)

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

	// Bulk endpoints (v0.5.10). 207 Multi-Status when any items skip; atomic
	// on permission/cycle/db failure. See docs/API.md > Bulk operations.
	projectWrite.PATCH("/nodes/batch-update", h.Node.BatchUpdate)
	projectWrite.POST("/edges/batch-create", h.Edge.BatchCreate)
	projectWrite.POST("/edges/batch-delete", h.Edge.BatchDelete)

	// Provenance & suggestion queue (migration 009). Agents propose to the
	// queue; humans approve / verify. permissions JSONB gates each path.
	projectWrite.POST("/nodes/:nodeId/suggestions", h.Suggestion.Suggest)
	projectWrite.POST("/nodes/:nodeId/verify", h.Suggestion.Verify)
	projectGroup.GET("/suggestions", h.Suggestion.List)
	projectWrite.PATCH("/suggestions/:suggestionId", h.Suggestion.Decide)

	// v0.6.0 Knowledge OS side tables: comments, attachments, project tags.
	if h.KnowledgeOS != nil {
		projectGroup.GET("/nodes/:nodeId/comments", h.KnowledgeOS.ListComments)
		projectWrite.POST("/nodes/:nodeId/comments", h.KnowledgeOS.CreateComment)
		projectWrite.PATCH("/comments/:commentId", h.KnowledgeOS.UpdateComment)
		projectWrite.POST("/comments/:commentId/resolve", h.KnowledgeOS.ResolveComment)
		projectWrite.DELETE("/comments/:commentId", h.KnowledgeOS.DeleteComment)

		projectGroup.GET("/nodes/:nodeId/attachments", h.KnowledgeOS.ListAttachments)
		projectWrite.POST("/nodes/:nodeId/attachments", h.KnowledgeOS.UploadAttachment)
		projectGroup.GET("/attachments/:attachmentId", h.KnowledgeOS.DownloadAttachment)
		projectWrite.DELETE("/attachments/:attachmentId", h.KnowledgeOS.DeleteAttachment)

		projectGroup.GET("/tags", h.KnowledgeOS.ListTags)
		projectWrite.PUT("/tags/:tag", h.KnowledgeOS.UpsertTag)
		projectWrite.DELETE("/tags/:tag", h.KnowledgeOS.DeleteTag)
	}

	// Summary — cross-project cardinality is account-level state, not per-project.
	// A project-scoped API key has no business reading it (info leak of the
	// account's project list). accountRoutes runs RejectScopedKey().
	accountRoutes.GET("/api/projects/summary", h.Summary.Get)

	// Shared (public, no auth required) — stricter rate limit for unauthenticated access
	shared := e.Group("/api/shared/:shareToken",
		echoMw.RateLimiter(echoMw.NewRateLimiterMemoryStore(5)),
		middleware.SharedAccess(projectRepo),
	)
	shared.GET("", h.Project.SharedGet)
	shared.GET("/graph", h.Node.SharedGraph)
	shared.GET("/og-image", h.Node.OGImage)
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
