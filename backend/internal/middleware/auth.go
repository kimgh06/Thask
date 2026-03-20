package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

const (
	ContextUserID      = "user_id"
	ContextUserEmail   = "user_email"
	ContextDisplayName = "user_display_name"
	SessionCookieName  = "thask_session"
)

func Auth(sessionRepo *repository.SessionRepo, apiKeyRepo ...*repository.APIKeyRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Try cookie auth
			cookie, err := c.Cookie(SessionCookieName)
			if err == nil && cookie.Value != "" {
				user, err := sessionRepo.ValidateToken(c.Request().Context(), cookie.Value)
				if err == nil {
					c.Set(ContextUserID, user.ID)
					c.Set(ContextUserEmail, user.Email)
					c.Set(ContextDisplayName, user.DisplayName)
					return next(c)
				}
			}

			// 2. Try Bearer token auth (API key)
			if len(apiKeyRepo) > 0 && apiKeyRepo[0] != nil {
				authHeader := c.Request().Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer thsk_") {
					token := strings.TrimPrefix(authHeader, "Bearer ")
					keyHash := service.HashAPIKey(token)
					apiKey, user, err := apiKeyRepo[0].FindByKeyHash(c.Request().Context(), keyHash)
					if err == nil {
						go func() {
						tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						if err := apiKeyRepo[0].UpdateLastUsed(tctx, apiKey.ID); err != nil {
							slog.Warn("failed to update API key last_used_at", "error", err)
						}
					}()
						c.Set(ContextUserID, user.ID)
						c.Set(ContextUserEmail, user.Email)
						c.Set(ContextDisplayName, user.DisplayName)
						return next(c)
					}
					return c.JSON(http.StatusUnauthorized, dto.Err("Invalid API key"))
				}
			}

			return c.JSON(http.StatusUnauthorized, dto.Err("Authentication required"))
		}
	}
}

func GetUserID(c echo.Context) string {
	return c.Get(ContextUserID).(string)
}
