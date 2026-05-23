package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type APIKeyHandler struct {
	apiKeyRepo *repository.APIKeyRepo
}

func NewAPIKeyHandler(apiKeyRepo *repository.APIKeyRepo) *APIKeyHandler {
	return &APIKeyHandler{apiKeyRepo: apiKeyRepo}
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

	apiKey, err := h.apiKeyRepo.Create(ctx, userID, req.Name, keyPrefix, keyHash, kind, perms, expiresAt)
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
