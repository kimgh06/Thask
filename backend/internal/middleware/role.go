package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

const (
	ContextTeamID   = "team_id"
	ContextTeamRole = "team_role"
)

// TeamAccess resolves :teamSlug, verifies membership, and sets team_id + team_role in context.
func TeamAccess(teamRepo *repository.TeamRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			slug := c.Param("teamSlug")
			if slug == "" {
				return c.JSON(http.StatusBadRequest, dto.Err("Team slug required"))
			}

			ctx := c.Request().Context()
			userID := GetUserID(c)

			team, err := teamRepo.FindBySlug(ctx, slug)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
			}

			role, err := teamRepo.GetMemberRole(ctx, team.ID, userID)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Team not found"))
			}

			c.Set(ContextTeamID, team.ID)
			c.Set(ContextTeamRole, role)
			return next(c)
		}
	}
}

// RequireRole rejects requests if the user's team role is below minRole.
func RequireRole(minRole model.TeamRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := GetTeamRole(c)
			if !role.AtLeast(minRole) {
				return c.JSON(http.StatusForbidden, dto.Err("Insufficient permissions"))
			}
			return next(c)
		}
	}
}

func GetTeamID(c echo.Context) string {
	id, ok := c.Get(ContextTeamID).(string)
	if !ok {
		return ""
	}
	return id
}

func GetTeamRole(c echo.Context) model.TeamRole {
	role, ok := c.Get(ContextTeamRole).(model.TeamRole)
	if !ok {
		return ""
	}
	return role
}
