package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type APIKeyHandler struct {
	apiKeyRepo  *repository.APIKeyRepo
	projectRepo *repository.ProjectRepo
	teamRepo    *repository.TeamRepo
	pmRepo      *repository.ProjectMemberRepo
}

func NewAPIKeyHandler(
	apiKeyRepo *repository.APIKeyRepo,
	projectRepo *repository.ProjectRepo,
	teamRepo *repository.TeamRepo,
	pmRepo *repository.ProjectMemberRepo,
) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyRepo:  apiKeyRepo,
		projectRepo: projectRepo,
		teamRepo:    teamRepo,
		pmRepo:      pmRepo,
	}
}

func (h *APIKeyHandler) Create(c echo.Context) error {
	var req dto.CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	// v0.6.0 R3: reject cross-tenant scope claims. Users can only mint a
	// project-scoped key for projects they already have access to; otherwise
	// a dormant key would sit in api_keys and activate the moment the user
	// is invited to that project. Mirror of ProjectAccess middleware, with
	// P3 hardening: only fall back to pmRepo on pgx.ErrNoRows — transient
	// DB errors must fail closed (500) rather than being interpreted as
	// "not a team member" and possibly clearing on a stale pmRepo row.
	if req.ProjectID != nil && *req.ProjectID != "" {
		project, err := h.projectRepo.FindByID(ctx, *req.ProjectID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.JSON(http.StatusForbidden, dto.Err("No access to that project"))
			}
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to verify project"))
		}
		_, teamErr := h.teamRepo.GetMemberRole(ctx, project.TeamID, userID)
		if teamErr != nil && !errors.Is(teamErr, pgx.ErrNoRows) {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to verify project access"))
		}
		if teamErr != nil {
			hasProject := false
			if h.pmRepo != nil {
				role, pmErr := h.pmRepo.GetRole(ctx, *req.ProjectID, userID)
				if pmErr != nil && !errors.Is(pmErr, pgx.ErrNoRows) {
					return c.JSON(http.StatusInternalServerError, dto.Err("Failed to verify project access"))
				}
				if pmErr == nil && role != "" {
					hasProject = true
				}
			}
			if !hasProject {
				return c.JSON(http.StatusForbidden, dto.Err("No access to that project"))
			}
		}
	}

	// Enforce max 10 keys per user
	count, err := h.apiKeyRepo.CountByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to check key count"))
	}
	if count >= 10 {
		return c.JSON(http.StatusBadRequest, dto.Err("Maximum 10 API keys allowed"))
	}

	plainKey, err := service.GenerateAPIKey()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to generate key"))
	}

	keyHash := service.HashAPIKey(plainKey)
	keyPrefix := plainKey[:12] // "thsk_" + 7 chars

	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		t := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &t
	}

	kind := model.APIKeyKind(req.Kind)
	if kind == "" {
		kind = model.APIKeyKindUserInteractive
	}
	perms := model.DefaultPermissions(kind)
	if req.Permissions != nil {
		applyPermissionOverrides(&perms, *req.Permissions)
	}

	apiKey, err := h.apiKeyRepo.Create(ctx, userID, req.Name, keyPrefix, keyHash, kind, perms, expiresAt, req.ProjectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create API key"))
	}

	return c.JSON(http.StatusCreated, dto.OK(map[string]any{
		"id":          apiKey.ID,
		"name":        apiKey.Name,
		"keyPrefix":   apiKey.KeyPrefix,
		"kind":        apiKey.Kind,
		"permissions": apiKey.Permissions,
		"key":         plainKey, // one-time display
		"expiresAt":   apiKey.ExpiresAt,
		"createdAt":   apiKey.CreatedAt,
	}))
}

// applyPermissionOverrides mutates perms in place, applying any boolean
// fields present in the override map. Missing keys leave the preset intact.
func applyPermissionOverrides(perms *model.APIKeyPermissions, override map[string]bool) {
	if v, ok := override["read"]; ok {
		perms.Read = v
	}
	if v, ok := override["write_structural"]; ok {
		perms.WriteStructural = v
	}
	if v, ok := override["write_semantic"]; ok {
		perms.WriteSemantic = v
	}
	if v, ok := override["write_meta"]; ok {
		perms.WriteMeta = v
	}
	if v, ok := override["verify"]; ok {
		perms.Verify = v
	}
	if v, ok := override["delete"]; ok {
		perms.Delete = v
	}
	if v, ok := override["suggest"]; ok {
		perms.Suggest = v
	}
}

func (h *APIKeyHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	userID := mw.GetUserID(c)

	keys, err := h.apiKeyRepo.FindByUserID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch API keys"))
	}
	if keys == nil {
		keys = []model.APIKey{}
	}

	return c.JSON(http.StatusOK, dto.OK(keys))
}

func (h *APIKeyHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	userID := mw.GetUserID(c)
	keyID := c.Param("keyId")

	if err := h.apiKeyRepo.Delete(ctx, keyID, userID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("API key not found"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}
