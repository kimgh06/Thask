package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
)

func ProjectAccess(projectRepo *repository.ProjectRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			projectID := c.Param("projectId")
			userID := GetUserID(c)

			ok, err := projectRepo.VerifyAccess(c.Request().Context(), projectID, userID)
			if err != nil || !ok {
				return c.JSON(http.StatusNotFound, dto.Err("Project not found"))
			}

			return next(c)
		}
	}
}
