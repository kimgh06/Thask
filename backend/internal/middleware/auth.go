package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
)

const (
	ContextUserID      = "user_id"
	ContextUserEmail   = "user_email"
	ContextDisplayName = "user_display_name"
	SessionCookieName  = "thask_session"
)

func Auth(sessionRepo *repository.SessionRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				return c.JSON(http.StatusUnauthorized, dto.Err("Authentication required"))
			}

			user, err := sessionRepo.ValidateToken(c.Request().Context(), cookie.Value)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, dto.Err("Session expired"))
			}

			c.Set(ContextUserID, user.ID)
			c.Set(ContextUserEmail, user.Email)
			c.Set(ContextDisplayName, user.DisplayName)
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) string {
	return c.Get(ContextUserID).(string)
}
