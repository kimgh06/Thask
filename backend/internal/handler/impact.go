package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type ImpactHandler struct {
	nodeRepo *repository.NodeRepo
	edgeRepo *repository.EdgeRepo
}

func NewImpactHandler(nodeRepo *repository.NodeRepo, edgeRepo *repository.EdgeRepo) *ImpactHandler {
	return &ImpactHandler{nodeRepo: nodeRepo, edgeRepo: edgeRepo}
}

func (h *ImpactHandler) Analyze(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	since := time.Now().Add(-7 * 24 * time.Hour)
	if s := c.QueryParam("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}

	depth := 2
	if d := c.QueryParam("depth"); d != "" {
		if v, err := strconv.Atoi(d); err == nil {
			depth = v
		}
	}

	changedNodes, err := h.nodeRepo.FindChangedSince(ctx, projectID, since)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch changed nodes"))
	}

	failNodes, _ := h.nodeRepo.FindFailOrBug(ctx, projectID)
	if failNodes == nil {
		failNodes = []model.Node{}
	}

	if len(changedNodes) == 0 {
		return c.JSON(http.StatusOK, dto.OK(model.ImpactResult{
			ChangedNodes:  []model.Node{},
			ImpactedNodes: []model.Node{},
			FailNodes:     failNodes,
			ImpactEdges:   []model.Edge{},
		}))
	}

	allEdges, _ := h.edgeRepo.FindByProjectID(ctx, projectID)
	if allEdges == nil {
		allEdges = []model.Edge{}
	}

	changedIDs := make([]string, len(changedNodes))
	for i, n := range changedNodes {
		changedIDs[i] = n.ID
	}

	impactedIDs := service.ComputeImpact(changedIDs, allEdges, depth)
	impactedNodes, _ := h.nodeRepo.FindByIDs(ctx, impactedIDs)
	if impactedNodes == nil {
		impactedNodes = []model.Node{}
	}

	// Filter edges within impact subgraph
	allImpacted := make(map[string]bool)
	for _, id := range changedIDs {
		allImpacted[id] = true
	}
	for _, id := range impactedIDs {
		allImpacted[id] = true
	}
	var impactEdges []model.Edge
	for _, e := range allEdges {
		if allImpacted[e.SourceID] && allImpacted[e.TargetID] {
			impactEdges = append(impactEdges, e)
		}
	}
	if impactEdges == nil {
		impactEdges = []model.Edge{}
	}

	return c.JSON(http.StatusOK, dto.OK(model.ImpactResult{
		ChangedNodes:  changedNodes,
		ImpactedNodes: impactedNodes,
		FailNodes:     failNodes,
		ImpactEdges:   impactEdges,
	}))
}
