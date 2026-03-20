package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
)

func ProjectAccess(projectRepo *repository.ProjectRepo, teamRepo *repository.TeamRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			projectID := c.Param("projectId")
			userID := GetUserID(c)
			ctx := c.Request().Context()

			project, err := projectRepo.FindByID(ctx, projectID)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
			}

			role, err := teamRepo.GetMemberRole(ctx, project.TeamID, userID)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
			}

			c.Set(ContextTeamID, project.TeamID)
			c.Set(ContextTeamRole, role)
			return next(c)
		}
	}
}
