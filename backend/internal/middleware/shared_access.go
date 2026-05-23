package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

type sharedCacheEntry struct {
	project   *model.Project
	expiresAt time.Time
}

var (
	sharedCache   = make(map[string]sharedCacheEntry)
	sharedCacheMu sync.RWMutex
	sharedCacheTTL = 30 * time.Second
)

const (
	ContextProjectID = "project_id"
	AnonymousUserID  = "anonymous"
)

// SharedAccess validates share token and grants role based on link_sharing setting.
// Used for public /api/shared/:shareToken routes (no auth required).
func SharedAccess(projectRepo *repository.ProjectRepo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			injectClient(c)
			c.Set(ContextActorKind, string(model.APIKeyKindUserInteractive))
			c.Set(ContextPermissions, allPermissions)

			token := c.Param("shareToken")
			if token == "" {
				return c.JSON(http.StatusNotFound, dto.Err("Not found"))
			}

			// Check cache first
			sharedCacheMu.RLock()
			if entry, ok := sharedCache[token]; ok && time.Now().Before(entry.expiresAt) {
				sharedCacheMu.RUnlock()
				project := entry.project
				var role model.TeamRole
				switch project.LinkSharing {
				case "editor":
					role = model.TeamRoleMember
				case "viewer":
					role = model.TeamRoleViewer
				default:
					return c.JSON(http.StatusNotFound, dto.Err("Not found"))
				}
				c.Set(ContextProjectID, project.ID)
				c.Set(ContextTeamID, project.TeamID)
				c.Set(ContextTeamRole, role)
				if c.Get(ContextUserID) == nil {
					c.Set(ContextUserID, AnonymousUserID)
					c.Set(ContextUserEmail, AnonymousUserID)
					c.Set(ContextDisplayName, "Anonymous")
				}
				return next(c)
			}
			sharedCacheMu.RUnlock()

			ctx := c.Request().Context()
			project, err := projectRepo.FindByShareToken(ctx, token)
			if err != nil {
				return c.JSON(http.StatusNotFound, dto.Err("Not found"))
			}

			// Cache the result
			sharedCacheMu.Lock()
			sharedCache[token] = sharedCacheEntry{project: project, expiresAt: time.Now().Add(sharedCacheTTL)}
			sharedCacheMu.Unlock()

			var role model.TeamRole
			switch project.LinkSharing {
			case "editor":
				role = model.TeamRoleMember
			case "viewer":
				role = model.TeamRoleViewer
			default:
				return c.JSON(http.StatusNotFound, dto.Err("Not found"))
			}

			c.Set(ContextProjectID, project.ID)
			c.Set(ContextTeamID, project.TeamID)
			c.Set(ContextTeamRole, role)
			// Set anonymous user context so handlers don't panic on GetUserID
			if c.Get(ContextUserID) == nil {
				c.Set(ContextUserID, AnonymousUserID)
				c.Set(ContextUserEmail, AnonymousUserID)
				c.Set(ContextDisplayName, "Anonymous")
			}
			return next(c)
		}
	}
}
