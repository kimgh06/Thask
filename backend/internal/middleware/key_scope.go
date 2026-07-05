package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
)

// RejectScopedKey blocks project-scoped API keys from accessing routes that
// live outside a specific project. Without this, a scoped `agent` key issued
// for project A could POST /api/auth/api-keys and mint a NEW key with a
// different (or no) scope — self-escalation. Same problem on team CRUD.
//
// Attach to any route group under `authed` that isn't already wrapped by
// ProjectAccess middleware (which enforces the same rule per-project).
// Cookie-authenticated calls have no ContextAPIKeyProjectID and pass through
// unchanged.
func RejectScopedKey() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if scope, _ := c.Get(ContextAPIKeyProjectID).(string); scope != "" {
				return c.JSON(http.StatusForbidden, dto.Err("API key is scoped to a project; cannot access account/team routes"))
			}
			return next(c)
		}
	}
}
