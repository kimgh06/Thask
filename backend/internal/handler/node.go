package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type NodeHandler struct {
	nodeRepo    *repository.NodeRepo
	edgeRepo    *repository.EdgeRepo
	historyRepo *repository.HistoryRepo
}

func NewNodeHandler(nodeRepo *repository.NodeRepo, edgeRepo *repository.EdgeRepo, historyRepo *repository.HistoryRepo) *NodeHandler {
	return &NodeHandler{nodeRepo: nodeRepo, edgeRepo: edgeRepo, historyRepo: historyRepo}
}

func (h *NodeHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	var nodeType, status *string
	if t := c.QueryParam("type"); t != "" {
		nodeType = &t
	}
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}

	nodes, err := h.nodeRepo.FindByProjectID(ctx, projectID, nodeType, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch nodes"))
	}
	if nodes == nil {
		nodes = []model.Node{}
	}

	return c.JSON(http.StatusOK, dto.OK(nodes))
}

func (h *NodeHandler) Get(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	nodeID := c.Param("nodeId")

	node, err := h.nodeRepo.FindByID(ctx, nodeID, projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Node not found"))
	}

	connectedEdges, err := h.edgeRepo.FindConnected(ctx, nodeID)
	if err != nil {
		connectedEdges = []model.Edge{}
	}

	connectedIDs := make(map[string]bool)
	for _, e := range connectedEdges {
		if e.SourceID != nodeID {
			connectedIDs[e.SourceID] = true
		}
		if e.TargetID != nodeID {
			connectedIDs[e.TargetID] = true
		}
	}

	idList := make([]string, 0, len(connectedIDs))
	for id := range connectedIDs {
		idList = append(idList, id)
	}

	history, _ := h.historyRepo.FindByNodeID(ctx, nodeID, 20)
	if history == nil {
		history = []model.NodeHistoryEntry{}
	}

	return c.JSON(http.StatusOK, dto.OK(model.NodeDetail{
		Node:             *node,
		ConnectedEdges:   connectedEdges,
		ConnectedNodeIDs: idList,
		History:          history,
	}))
}

func (h *NodeHandler) Create(c echo.Context) error {
	var req dto.CreateNodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	userID := mw.GetUserID(c)

	status := model.NodeStatus(req.Status)
	if status == "" {
		status = model.NodeStatusInProgress
	}

	node := &model.Node{
		ProjectID:   projectID,
		Type:        model.NodeType(req.Type),
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		AssigneeID:  req.AssigneeID,
		Tags:        req.Tags,
		PositionX:   req.PositionX,
		PositionY:   req.PositionY,
		Width:       req.Width,
		Height:      req.Height,
	}
	if node.Tags == nil {
		node.Tags = []string{}
	}

	created, err := h.nodeRepo.Create(ctx, node)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create node"))
	}

	// Record history
	title := created.Title
	_ = h.historyRepo.Create(ctx, created.ID, projectID, userID, model.HistoryActionCreated, strPtr("title"), nil, &title)

	return c.JSON(http.StatusCreated, dto.OK(created))
}

func (h *NodeHandler) Update(c echo.Context) error {
	var req dto.UpdateNodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	nodeID := c.Param("nodeId")
	userID := mw.GetUserID(c)

	existing, err := h.nodeRepo.FindByID(ctx, nodeID, projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Node not found"))
	}

	fields := make(map[string]any)
	if req.Type != nil {
		fields["type"] = *req.Type
	}
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.AssigneeID != nil {
		fields["assignee_id"] = *req.AssigneeID
	}
	if req.Tags != nil {
		fields["tags"] = req.Tags
	}
	if req.ParentID != nil {
		newParentID := *req.ParentID
		if newParentID != "" {
			// Prevent circular parent references
			if newParentID == nodeID {
				return c.JSON(http.StatusBadRequest, dto.Err("Node cannot be its own parent"))
			}
			if err := h.detectParentCycle(ctx, projectID, nodeID, newParentID); err != nil {
				return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
			}
		}
		fields["parent_id"] = newParentID
	}
	if req.Width != nil {
		fields["width"] = *req.Width
	}
	if req.Height != nil {
		fields["height"] = *req.Height
	}

	updated, err := h.nodeRepo.Update(ctx, nodeID, fields)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update node"))
	}

	// Record history
	for field, val := range fields {
		action := model.HistoryActionUpdated
		if field == "status" {
			action = model.HistoryActionStatusChanged
		}
		oldVal := getOldValue(existing, field)
		newVal := fmt.Sprintf("%v", val)
		_ = h.historyRepo.Create(ctx, nodeID, projectID, userID, action, &field, oldVal, &newVal)
	}

	// Waterfall propagation
	var propagated []service.StatusChange
	if req.Status != nil && model.NodeStatus(*req.Status) != existing.Status {
		newStatus := model.NodeStatus(*req.Status)
		if newStatus == model.NodeStatusPass || newStatus == model.NodeStatusFail {
			allNodes, _ := h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
			allEdges, _ := h.edgeRepo.FindByProjectID(ctx, projectID)

			wNodes := make([]service.WaterfallNode, len(allNodes))
			for i, n := range allNodes {
				wNodes[i] = service.WaterfallNode{ID: n.ID, Status: n.Status, ParentID: n.ParentID}
			}
			wEdges := make([]service.WaterfallEdge, len(allEdges))
			for i, e := range allEdges {
				wEdges[i] = service.WaterfallEdge{SourceID: e.SourceID, TargetID: e.TargetID, EdgeType: e.EdgeType}
			}

			propagated = service.ComputeWaterfall(nodeID, newStatus, wNodes, wEdges)
			for _, wc := range propagated {
				_ = h.nodeRepo.UpdateStatus(ctx, wc.NodeID, wc.NewStatus)
				field := "status"
				old := string(wc.OldStatus)
				nw := string(wc.NewStatus)
				_ = h.historyRepo.Create(ctx, wc.NodeID, projectID, userID, model.HistoryActionStatusChanged, &field, &old, &nw)
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]any{"data": updated, "propagated": propagated})
}

func (h *NodeHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := c.Param("projectId")
	nodeID := c.Param("nodeId")

	if err := h.nodeRepo.Delete(ctx, nodeID, projectID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete node"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *NodeHandler) BatchUpdatePositions(c echo.Context) error {
	var req dto.BatchPositionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := c.Param("projectId")

	positions := make([]struct {
		ID     string
		X, Y   float64
		Width  *float64
		Height *float64
	}, len(req.Positions))
	for i, p := range req.Positions {
		positions[i] = struct {
			ID     string
			X, Y   float64
			Width  *float64
			Height *float64
		}{p.ID, p.X, p.Y, p.Width, p.Height}
	}

	if err := h.nodeRepo.BatchUpdatePositions(ctx, projectID, positions); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update positions"))
	}

	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// detectParentCycle walks up the parent chain from targetParentID.
// If it reaches nodeID, setting nodeID's parent to targetParentID would create a cycle.
func (h *NodeHandler) detectParentCycle(ctx context.Context, projectID, nodeID, targetParentID string) error {
	visited := map[string]bool{nodeID: true}
	cur := targetParentID
	for cur != "" {
		if visited[cur] {
			return fmt.Errorf("Cannot set parent: would create circular reference")
		}
		visited[cur] = true
		parent, err := h.nodeRepo.FindByID(ctx, cur, projectID)
		if err != nil || parent.ParentID == nil {
			break
		}
		cur = *parent.ParentID
	}
	return nil
}

func strPtr(s string) *string { return &s }

func getOldValue(n *model.Node, field string) *string {
	var v string
	switch field {
	case "type":
		v = string(n.Type)
	case "title":
		v = n.Title
	case "description":
		if n.Description != nil {
			v = *n.Description
		}
	case "status":
		v = string(n.Status)
	case "assignee_id":
		if n.AssigneeID != nil {
			v = *n.AssigneeID
		}
	case "parent_id":
		if n.ParentID != nil {
			v = *n.ParentID
		}
	default:
		return nil
	}
	return &v
}
