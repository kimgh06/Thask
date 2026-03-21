package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
)

func ProjectAccess(projectRepo *repository.ProjectRepo, teamRepo *repository.TeamRepo, pmRepo ...*repository.ProjectMemberRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			projectID := c.Param("projectId")
			userID := GetUserID(c)
			ctx := c.Request().Context()

			project, err := projectRepo.FindByID(ctx, projectID)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
			}

			// 1. Check team membership first
			teamRole, err := teamRepo.GetMemberRole(ctx, project.TeamID, userID)
			if err == nil {
				c.Set(ContextProjectID, project.ID)
				c.Set(ContextTeamID, project.TeamID)
				c.Set(ContextTeamRole, teamRole)
				return next(c)
			}

			// 2. Fall back to project-level membership (capped to member via ProjectRole.ToTeamRole)
			if len(pmRepo) > 0 && pmRepo[0] != nil {
				pmRole, err := pmRepo[0].GetRole(ctx, projectID, userID)
				if err == nil && pmRole != "" {
					c.Set(ContextProjectID, project.ID)
					c.Set(ContextTeamID, project.TeamID)
					c.Set(ContextTeamRole, pmRole.ToTeamRole())
					return next(c)
				}
			}

			return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
		}
	}
}
