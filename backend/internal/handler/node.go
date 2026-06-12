package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type NodeHandler struct {
	nodeRepo       *repository.NodeRepo
	edgeRepo       *repository.EdgeRepo
	historyRepo    *repository.HistoryRepo
	audit          *audit.Logger
	hub            *service.Hub
	captureURL     string
	captureSecret  string
	captureTimeout time.Duration
}

func NewNodeHandler(nodeRepo *repository.NodeRepo, edgeRepo *repository.EdgeRepo, historyRepo *repository.HistoryRepo, auditLogger *audit.Logger, hub *service.Hub, captureURL, captureSecret string, captureTimeout time.Duration) *NodeHandler {
	if captureTimeout <= 0 {
		captureTimeout = 30 * time.Second
	}
	return &NodeHandler{
		nodeRepo: nodeRepo, edgeRepo: edgeRepo, historyRepo: historyRepo, audit: auditLogger, hub: hub,
		captureURL: captureURL, captureSecret: captureSecret, captureTimeout: captureTimeout,
	}
}

func (h *NodeHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	var nodeType, status *string
	if t := c.QueryParam("type"); t != "" {
		nodeType = &t
	}
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}

	// V1 paginated path
	if c.Get(mw.ContextIsV1) == true {
		limitParam, _ := strconv.Atoi(c.QueryParam("limit"))
		limit := dto.ClampLimit(limitParam, 100)

		var afterTime *time.Time
		var afterID *string
		if cursor := c.QueryParam("after"); cursor != "" {
			t, id, err := dto.DecodeCursor(cursor)
			if err != nil {
				return c.JSON(http.StatusBadRequest, dto.V1Err(400, "Invalid cursor"))
			}
			afterTime = &t
			afterID = &id
		}

		nodes, hasMore, err := h.nodeRepo.FindByProjectIDPaginated(ctx, projectID, nodeType, status, limit, afterTime, afterID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.V1Err(500, "Failed to fetch nodes"))
		}
		if nodes == nil {
			nodes = []model.Node{}
		}

		var nextCursor *string
		if hasMore && len(nodes) > 0 {
			last := nodes[len(nodes)-1]
			c := dto.EncodeCursor(last.CreatedAt, last.ID)
			nextCursor = &c
		}
		return c.JSON(http.StatusOK, dto.PaginatedResponse{
			Data:       nodes,
			Pagination: dto.PaginationMeta{Limit: limit, HasMore: hasMore, NextCursor: nextCursor},
		})
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
	projectID := mw.ResolveProjectID(c)
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
	projectID := mw.ResolveProjectID(c)
	userID := mw.GetUserID(c)

	// Permission gate: a Create that carries a description is a semantic write
	// (the description is the new claim about reality); a bare structural
	// node is fine for agents. Reject upfront so the request never mutates.
	mutationKind := audit.MutationStructural
	if req.Description != nil && *req.Description != "" {
		mutationKind = audit.MutationSemantic
	}
	if err := h.audit.RequirePermission(c, mutationKind, "write", audit.Event{
		ProjectID: projectID, EntityType: "node", Action: audit.ActionCreated,
	}); err != nil {
		return err
	}

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
	if userID != "" {
		uid := userID
		node.CreatedBy = &uid
	}
	if node.Tags == nil {
		node.Tags = []string{}
	}

	created, err := h.nodeRepo.Create(ctx, node)
	if err != nil {
		slog.Error("nodeRepo.Create failed", "error", err)
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to create node"))
	}

	// Record history (skip for anonymous shared access)
	if userID != mw.AnonymousUserID {
		title := created.Title
		_ = h.historyRepo.Create(ctx, created.ID, projectID, userID, model.HistoryActionCreated, strPtr("title"), nil, &title)
	}

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: created.ID,
		Action: audit.ActionCreated, MutationKind: mutationKind,
		FieldName: "title", NewValue: created.Title,
	})

	h.hub.Publish(service.Event{Type: service.EventNodeCreated, ProjectID: projectID, Data: created, UserID: userID})
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
	projectID := mw.ResolveProjectID(c)
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
		if *req.AssigneeID != "" {
			fields["assignee_id"] = *req.AssigneeID
		} else {
			fields["assignee_id"] = nil
		}
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
			fields["parent_id"] = newParentID
		} else {
			fields["parent_id"] = nil
		}
	}
	if req.Width != nil {
		fields["width"] = *req.Width
	}
	if req.Height != nil {
		fields["height"] = *req.Height
	}

	if len(fields) == 0 {
		return c.JSON(http.StatusOK, dto.OK(map[string]any{"node": existing, "propagated": []service.StatusChange{}}))
	}

	// Permission gate: classify every field being touched and reject the whole
	// update if the key lacks permission for any one of them. Rejection is
	// per-field so the response message points at the actual blocked flag.
	for field := range fields {
		mk := mutationKindForNodeField(field)
		if err := h.audit.RequirePermission(c, mk, "write", audit.Event{
			ProjectID: projectID, EntityType: "node", EntityID: nodeID,
			Action: audit.ActionUpdated, FieldName: field, MutationKind: mk,
		}); err != nil {
			return err
		}
	}

	updated, err := h.nodeRepo.Update(ctx, nodeID, fields)
	if err != nil {
		slog.Error("nodeRepo.Update failed", "nodeId", nodeID, "error", err, "fields", fields)
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update node"))
	}

	// Record history (skip for anonymous shared access) + parallel audit_log row.
	for field, val := range fields {
		oldVal := getOldValue(existing, field)
		newVal := fmt.Sprintf("%v", val)
		mk := mutationKindForNodeField(field)

		if userID != mw.AnonymousUserID {
			action := model.HistoryActionUpdated
			if field == "status" {
				action = model.HistoryActionStatusChanged
			}
			_ = h.historyRepo.Create(ctx, nodeID, projectID, userID, action, &field, oldVal, &newVal)
		}

		auditAction := audit.ActionUpdated
		if field == "status" {
			auditAction = audit.ActionStatusChanged
		}
		evt := audit.Event{
			ProjectID: projectID, EntityType: "node", EntityID: nodeID,
			Action: auditAction, FieldName: field, MutationKind: mk, NewValue: newVal,
		}
		if oldVal != nil {
			evt.OldValue = *oldVal
		}
		h.audit.Log(c, evt)
	}

	// If description was changed by a non-anonymous user, refresh the snapshot
	// provenance columns on the node so future readers can see who wrote it.
	if _, ok := fields["description"]; ok && userID != mw.AnonymousUserID {
		_ = h.nodeRepo.UpdateDescriptionProvenance(ctx, nodeID, userID, descriptionSourceForActor(c), mw.GetAgentModel(c))
	}

	// Waterfall propagation
	var propagated []service.StatusChange
	if req.Status != nil && model.NodeStatus(*req.Status) != existing.Status {
		propagated = h.propagateWaterfall(ctx, projectID, nodeID, model.NodeStatus(*req.Status), userID)
	}

	h.hub.Publish(service.Event{Type: service.EventNodeUpdated, ProjectID: projectID, Data: updated, UserID: userID})
	return c.JSON(http.StatusOK, dto.OK(map[string]any{"node": updated, "propagated": propagated}))
}

func (h *NodeHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")

	if err := h.audit.RequirePermission(c, audit.MutationStructural, "delete", audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID, Action: audit.ActionDeleted,
	}); err != nil {
		return err
	}

	// Snapshot the title before deletion so audit_log has a human-readable trace.
	existingTitle := ""
	if existing, ferr := h.nodeRepo.FindByID(ctx, nodeID, projectID); ferr == nil && existing != nil {
		existingTitle = existing.Title
	}

	if err := h.nodeRepo.Delete(ctx, nodeID, projectID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete node"))
	}

	userID := mw.GetUserID(c)
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID,
		Action: audit.ActionDeleted, MutationKind: audit.MutationStructural,
		FieldName: "title", OldValue: existingTitle,
	})
	h.hub.Publish(service.Event{Type: service.EventNodeDeleted, ProjectID: projectID, Data: nodeID, UserID: userID})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// propagateWaterfall computes and applies status changes to downstream nodes.
func (h *NodeHandler) propagateWaterfall(ctx context.Context, projectID, nodeID string, newStatus model.NodeStatus, userID string) []service.StatusChange {
	if newStatus != model.NodeStatusPass && newStatus != model.NodeStatusFail {
		return nil
	}

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

	propagated := service.ComputeWaterfall(nodeID, newStatus, wNodes, wEdges)
	if len(propagated) == 0 {
		return propagated
	}

	// Batch update: group by new status
	byStatus := make(map[model.NodeStatus][]string)
	for _, wc := range propagated {
		byStatus[wc.NewStatus] = append(byStatus[wc.NewStatus], wc.NodeID)
	}
	for status, ids := range byStatus {
		_ = h.nodeRepo.BatchUpdateStatus(ctx, projectID, ids, status)
	}

	// Batch history insert
	if userID != mw.AnonymousUserID {
		h.historyRepo.BatchCreateStatusChanges(ctx, projectID, userID, propagated)
	}

	return propagated
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

// descriptionSourceForActor maps the caller's actor_kind to the description_source
// enum stored on nodes (migration 007). Falls back to "unknown" when the kind
// is missing or unrecognised.
func descriptionSourceForActor(c echo.Context) string {
	switch mw.GetActorKind(c) {
	case "user_interactive":
		return "human"
	case "agent":
		return "agent"
	case "scanner":
		return "scanner"
	case "service":
		return "import"
	default:
		return "unknown"
	}
}
