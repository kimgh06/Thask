package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type AuthHandler struct {
	userRepo    *repository.UserRepo
	sessionRepo *repository.SessionRepo
}

func NewAuthHandler(userRepo *repository.UserRepo, sessionRepo *repository.SessionRepo) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()

	// Check existing
	if existing, _ := h.userRepo.FindByEmail(ctx, req.Email); existing != nil {
		return c.JSON(http.StatusConflict, dto.Err("Email already registered"))
	}

	hash, err := service.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to hash password"))
	}

	user, err := h.userRepo.Create(ctx, req.Email, req.DisplayName, hash)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create user"))
	}

	token, err := service.GenerateToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to generate session"))
	}

	expiresAt := service.SessionExpiresAt()
	if err := h.sessionRepo.Create(ctx, user.ID, token, expiresAt); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create session"))
	}

	setSessionCookie(c, token, expiresAt)

	return c.JSON(http.StatusCreated, dto.OK(map[string]any{
		"id":          user.ID,
		"email":       user.Email,
		"displayName": user.DisplayName,
	}))
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()

	user, err := h.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, dto.Err("Invalid email or password"))
	}

	if !service.VerifyPassword(req.Password, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, dto.Err("Invalid email or password"))
	}

	// Invalidate previous sessions (session rotation)
	_ = h.sessionRepo.DeleteByUserID(ctx, user.ID)

	token, err := service.GenerateToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to generate session"))
	}

	expiresAt := service.SessionExpiresAt()
	if err := h.sessionRepo.Create(ctx, user.ID, token, expiresAt); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create session"))
	}

	setSessionCookie(c, token, expiresAt)

	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"id":          user.ID,
		"email":       user.Email,
		"displayName": user.DisplayName,
	}))
}

func (h *AuthHandler) Me(c echo.Context) error {
	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"id":          c.Get(mw.ContextUserID),
		"email":       c.Get(mw.ContextUserEmail),
		"displayName": c.Get(mw.ContextDisplayName),
	}))
}

func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(mw.SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.sessionRepo.DeleteByToken(c.Request().Context(), cookie.Value)
	}

	c.SetCookie(&http.Cookie{
		Name:     mw.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func setSessionCookie(c echo.Context, token string, expiresAt time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     mw.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false, // set true in production
		SameSite: http.SameSiteLaxMode,
	})
}
