package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

type ActivityHandler struct {
	historyRepo *repository.HistoryRepo
}

func NewActivityHandler(historyRepo *repository.HistoryRepo) *ActivityHandler {
	return &ActivityHandler{historyRepo: historyRepo}
}

func (h *ActivityHandler) List(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	ctx := c.Request().Context()

	limitStr := c.QueryParam("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	entries, err := h.historyRepo.FindByProjectID(ctx, projectID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch activity"))
	}
	if entries == nil {
		entries = []model.NodeHistoryEntry{}
	}

	return c.JSON(http.StatusOK, dto.OK(entries))
}
