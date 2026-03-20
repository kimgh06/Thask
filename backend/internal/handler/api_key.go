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

	apiKey, err := h.apiKeyRepo.Create(ctx, userID, req.Name, keyPrefix, keyHash, expiresAt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create API key"))
	}

	return c.JSON(http.StatusCreated, dto.OK(map[string]any{
		"id":        apiKey.ID,
		"name":      apiKey.Name,
		"keyPrefix": apiKey.KeyPrefix,
		"key":       plainKey, // one-time display
		"expiresAt": apiKey.ExpiresAt,
		"createdAt": apiKey.CreatedAt,
	}))
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
