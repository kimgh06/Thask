package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type GraphAnalysisHandler struct {
	edgeRepo *repository.EdgeRepo
}

func NewGraphAnalysisHandler(edgeRepo *repository.EdgeRepo) *GraphAnalysisHandler {
	return &GraphAnalysisHandler{edgeRepo: edgeRepo}
}

func (h *GraphAnalysisHandler) Analyze(c echo.Context) error {
	projectID := mw.ResolveProjectID(c)
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch edges"))
	}

	if len(edges) > service.MaxAnalysisEdges {
		return c.JSON(http.StatusRequestEntityTooLarge, dto.Err("Graph too large for analysis"))
	}

	cycles := service.DetectCycles(edges)
	result := service.CriticalPath(edges)

	if cycles == nil {
		cycles = [][]string{}
	}
	criticalPath := result.Path
	if criticalPath == nil {
		criticalPath = []string{}
	}
	skippedCycleNodes := result.SkippedCycleNodes
	if skippedCycleNodes == nil {
		skippedCycleNodes = []string{}
	}

	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"cycles":            cycles,
		"criticalPath":      criticalPath,
		"skippedCycleNodes": skippedCycleNodes,
	}))
}
