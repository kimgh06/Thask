package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

const ContextProjectID = "project_id"

// SharedAccess validates share token and grants role based on link_sharing setting.
// Used for public /api/shared/:shareToken routes (no auth required).
func SharedAccess(projectRepo *repository.ProjectRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Param("shareToken")
			if token == "" {
				return c.JSON(http.StatusNotFound, dto.Err("Not found"))
			}

			ctx := c.Request().Context()
			project, err := projectRepo.FindByShareToken(ctx, token)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Not found"))
			}

			// Public share links are always read-only (viewer) regardless of link_sharing setting.
			// Authenticated users with project/team membership get write access through ProjectAccess.
			var role model.TeamRole
			switch project.LinkSharing {
			case "editor", "viewer":
				role = model.TeamRoleViewer
			default:
				return c.JSON(http.StatusNotFound, dto.Err("Not found"))
			}

			c.Set(ContextProjectID, project.ID)
			c.Set(ContextTeamID, project.TeamID)
			c.Set(ContextTeamRole, role)
			return next(c)
		}
	}
}
