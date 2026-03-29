package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"

	"github.com/thask/backend/internal/config"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

// DIVERGENCE TRACKER: 0/3 endpoints with handler-level v1 branching.
// If this exceeds 3, refactor to service-layer boundary per ADR exit criterion.

func RegisterV1Routes(
	e *echo.Echo,
	h Handlers,
	cfg *config.Config,
	sessionRepo *repository.SessionRepo,
	apiKeyRepo *repository.APIKeyRepo,
	teamRepo *repository.TeamRepo,
	projectRepo *repository.ProjectRepo,
	pmRepo *repository.ProjectMemberRepo,
	idempotencyRepo *repository.IdempotencyRepo,
) {
	// External CORS — permissive for external API consumers
	externalCORS := echoMw.CORSWithConfig(echoMw.CORSConfig{
		AllowOrigins:     cfg.V1AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization, "Idempotency-Key"},
		AllowCredentials: false,
		MaxAge:           86400,
		ExposeHeaders:    []string{"X-API-Version", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-Idempotency-Replayed"},
	})

	// v1 group with external middleware stack
	v1 := e.Group("/api/v1",
		externalCORS,
		echoMw.BodyLimit(fmt.Sprintf("%dB", cfg.MaxRequestBodyBytes)),
	)

	// Version discovery (no auth required)
	v1.GET("", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"version": "v1",
			"status":  "stable",
		})
	})

	// Docs (no auth required)
	v1.GET("/openapi.yaml", serveOpenAPISpec)
	v1.GET("/docs", serveDocsUI)

	// Authenticated v1 routes with full middleware chain:
	// V1ResponseWrapper (wraps errors) → Auth → Idempotency
	v1Authed := v1.Group("",
		middleware.V1ResponseWrapper(),
		middleware.Auth(sessionRepo, apiKeyRepo),
		middleware.Idempotency(idempotencyRepo),
	)

	// Teams (list only)
	v1Authed.GET("/teams", h.Team.List)

	// Team-scoped (read)
	v1Team := v1Authed.Group("/teams/:teamSlug", middleware.TeamAccess(teamRepo))
	v1Team.GET("", h.Team.GetBySlug)
	v1Team.GET("/members", h.Team.ListMembers)
	v1Team.GET("/projects", h.Team.ListProjects)

	// Project-scoped (ProjectAccess intersects API key project_ids)
	v1Project := v1Authed.Group("/projects/:projectId", middleware.ProjectAccess(projectRepo, teamRepo, pmRepo))

	// Read — all roles (including viewer)
	v1Project.GET("", h.Project.Get)
	v1Project.GET("/graph", h.Node.Graph)       // NOT paginated
	v1Project.GET("/nodes", h.Node.List)         // paginated via v1 context check in handler
	v1Project.GET("/nodes/:nodeId", h.Node.Get)
	v1Project.GET("/edges", h.Edge.List)         // paginated via v1 context check in handler
	v1Project.GET("/impact", h.Impact.Analyze)
	v1Project.GET("/graph/analyze", h.GraphAnalysis.Analyze)
	v1Project.GET("/activity", h.Activity.List)

	// Write — member+ only
	v1ProjectWrite := v1Project.Group("", middleware.RequireRole(model.TeamRoleMember))
	v1ProjectWrite.PATCH("", h.Project.Update)
	v1ProjectWrite.POST("/nodes", h.Node.Create)
	v1ProjectWrite.PATCH("/nodes/:nodeId", h.Node.Update)
	v1ProjectWrite.DELETE("/nodes/:nodeId", h.Node.Delete)
	v1ProjectWrite.POST("/edges", h.Edge.Create)
	v1ProjectWrite.PATCH("/edges/:edgeId", h.Edge.Update)
	v1ProjectWrite.DELETE("/edges/:edgeId", h.Edge.Delete)
	v1ProjectWrite.POST("/graph/import", h.Node.Import)
	v1ProjectWrite.POST("/graph/layout", h.Node.Layout)

	// Sharing (admin+ only)
	v1ProjectAdmin := v1Project.Group("", middleware.RequireRole(model.TeamRoleAdmin))
	v1ProjectAdmin.GET("/sharing", h.Project.GetSharing)
	v1ProjectAdmin.PUT("/sharing", h.Project.UpdateSharing)
}

// serveOpenAPISpec serves the raw OpenAPI YAML spec.
func serveOpenAPISpec(c echo.Context) error {
	// Try working directory first, then relative to executable
	for _, path := range []string{"api/openapi.yaml", "backend/api/openapi.yaml"} {
		if _, err := os.Stat(path); err == nil {
			return c.File(path)
		}
	}
	return c.JSON(http.StatusNotFound, dto.Err("OpenAPI spec not found"))
}

// serveDocsUI serves the Scalar API documentation UI.
func serveDocsUI(c echo.Context) error {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Thask API Documentation</title>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
	<script id="api-reference" data-url="/api/v1/openapi.yaml"></script>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.25.98" crossorigin="anonymous"></script>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}
