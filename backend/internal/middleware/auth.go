package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

const (
	ContextUserID      = "user_id"
	ContextUserEmail   = "user_email"
	ContextDisplayName = "user_display_name"
	SessionCookieName  = "thask_session"

	// V1 API key context
	ContextAPIKeyID = "api_key_id"

	// Provenance / audit context (migration 006~008). Populated by Auth().
	ContextActorKind      = "actor_kind"
	ContextPermissions    = "permissions"
	ContextClientType     = "client_type"
	ContextClientVersion  = "client_version"
	ContextAgentModel     = "agent_model"
	ContextAgentSessionID = "agent_session_id"

	// ClientHeader is the request header CLI/MCP/web clients set so the server
	// can record where the call came from. Format: "name/version [k=v ...]".
	ClientHeader = "X-Thask-Client"
)

// allPermissions is the safe-by-default permission set for cookie-authenticated
// users — they go through the web UI which already gates by role, so the
// per-key gate is open.
var allPermissions = model.APIKeyPermissions{
	Read: true, WriteStructural: true, WriteSemantic: true,
	WriteMeta: true, Verify: true, Delete: true, Suggest: true,
}

func Auth(sessionRepo *repository.SessionRepo, apiKeyRepo ...*repository.APIKeyRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			injectClient(c)

			// 1. Try cookie auth
			cookie, err := c.Cookie(SessionCookieName)
			if err == nil && cookie.Value != "" {
				user, err := sessionRepo.ValidateToken(c.Request().Context(), cookie.Value)
				if err == nil {
					c.Set(ContextUserID, user.ID)
					c.Set(ContextUserEmail, user.Email)
					c.Set(ContextDisplayName, user.DisplayName)
					c.Set(ContextActorKind, string(model.APIKeyKindUserInteractive))
					c.Set(ContextPermissions, allPermissions)
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
						c.Set(ContextAPIKeyID, apiKey.ID)
						c.Set(ContextActorKind, string(apiKey.Kind))
						c.Set(ContextPermissions, apiKey.Permissions)
						return next(c)
					}
					return c.JSON(http.StatusUnauthorized, dto.Err("Invalid API key"))
				}
			}

			return c.JSON(http.StatusUnauthorized, dto.Err("Authentication required"))
		}
	}
}

// injectClient parses X-Thask-Client and stashes the parts on the echo context.
// Missing / malformed headers leave context values unset; downstream code must
// treat them as optional.
//
// Header grammar (space-separated):
//
//	"thask-cli/0.5.8"
//	"thask-mcp/0.5.8 model=claude-opus-4-7 session=ab12cd34"
//	"thask-web/abc123"
func injectClient(c echo.Context) {
	header := c.Request().Header.Get(ClientHeader)
	if header == "" {
		return
	}
	parts := strings.Fields(header)
	if len(parts) == 0 {
		return
	}
	if name, version, ok := strings.Cut(parts[0], "/"); ok {
		c.Set(ContextClientType, name)
		if version != "" {
			c.Set(ContextClientVersion, version)
		}
	} else {
		c.Set(ContextClientType, parts[0])
	}
	for _, kv := range parts[1:] {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || v == "" {
			continue
		}
		switch k {
		case "model":
			c.Set(ContextAgentModel, v)
		case "session":
			c.Set(ContextAgentSessionID, v)
		}
	}
}

func GetUserID(c echo.Context) string {
	return c.Get(ContextUserID).(string)
}

// GetActorKind returns the caller's actor_kind (e.g. "user_interactive",
// "agent"), defaulting to "system" when not set (e.g. anonymous flows).
func GetActorKind(c echo.Context) string {
	v, _ := c.Get(ContextActorKind).(string)
	if v == "" {
		return "system"
	}
	return v
}

// GetPermissions returns the per-key permission set, defaulting to all-true
// when the request was authenticated via cookie (web UI).
func GetPermissions(c echo.Context) model.APIKeyPermissions {
	if v, ok := c.Get(ContextPermissions).(model.APIKeyPermissions); ok {
		return v
	}
	return allPermissions
}

// GetClientType returns the parsed X-Thask-Client name, or "" if unset.
func GetClientType(c echo.Context) string {
	v, _ := c.Get(ContextClientType).(string)
	return v
}

// GetClientVersion returns the parsed X-Thask-Client version, or "" if unset.
func GetClientVersion(c echo.Context) string {
	v, _ := c.Get(ContextClientVersion).(string)
	return v
}

// GetAgentModel returns the agent model identifier when actor_kind is "agent".
func GetAgentModel(c echo.Context) string {
	v, _ := c.Get(ContextAgentModel).(string)
	return v
}

// GetAgentSessionID returns the opaque session identifier when actor_kind is "agent".
func GetAgentSessionID(c echo.Context) string {
	v, _ := c.Get(ContextAgentSessionID).(string)
	return v
}

// GetAPIKeyID returns the api_keys.id when authenticated via API key,
// empty string when via cookie.
func GetAPIKeyID(c echo.Context) string {
	v, _ := c.Get(ContextAPIKeyID).(string)
	return v
}
