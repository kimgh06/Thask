package handler

import (
	"context"
	cryptoRand "crypto/rand"
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

// graphResponse is the shared implementation for Graph and SharedGraph.
func (h *NodeHandler) graphResponse(c echo.Context, projectID string) error {
	ctx := c.Request().Context()

	nodes, err := h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch nodes"))
	}
	if nodes == nil {
		nodes = []model.Node{}
	}

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch edges"))
	}
	if edges == nil {
		edges = []model.Edge{}
	}

	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"nodes": nodes,
		"edges": edges,
	}))
}

// Graph returns nodes and edges together in a single response to ensure consistency.
func (h *NodeHandler) Graph(c echo.Context) error {
	return h.graphResponse(c, mw.ResolveProjectID(c))
}

func (h *NodeHandler) SharedGraph(c echo.Context) error {
	return h.graphResponse(c, mw.GetProjectID(c))
}

func (h *NodeHandler) Layout(c echo.Context) error {
	var req dto.LayoutRequest
	if err := c.Bind(&req); err != nil {
		req.Algorithm = ""
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	algorithm := req.Algorithm
	if algorithm == "" {
		algorithm = "dagre"
	}

	nodes, err := h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch nodes"))
	}
	if len(nodes) == 0 {
		return c.JSON(http.StatusOK, dto.OK(map[string]any{"nodes": []any{}, "edges": []any{}}))
	}

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch edges"))
	}
	if edges == nil {
		edges = []model.Edge{}
	}

	layoutResult := service.CalculateLayout(nodes, edges, algorithm)

	positions := make([]struct {
		ID     string
		X, Y   float64
		Width  *float64
		Height *float64
	}, len(layoutResult.Positions))
	for i, lp := range layoutResult.Positions {
		positions[i] = struct {
			ID     string
			X, Y   float64
			Width  *float64
			Height *float64
		}{lp.ID, lp.X, lp.Y, lp.Width, lp.Height}
	}

	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "graph", Action: audit.ActionLayoutComputed,
	}); err != nil {
		return err
	}

	if err := h.nodeRepo.BatchUpdatePositions(ctx, projectID, positions); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to save layout"))
	}

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "graph",
		Action: audit.ActionLayoutComputed, MutationKind: audit.MutationMeta,
		Trigger: "layout",
		Metadata: map[string]any{"algorithm": algorithm, "count": len(positions)},
	})

	nodes, _ = h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
	if nodes == nil {
		nodes = []model.Node{}
	}
	edges, _ = h.edgeRepo.FindByProjectID(ctx, projectID)
	if edges == nil {
		edges = []model.Edge{}
	}

	h.hub.Publish(service.Event{Type: service.EventGraphLayout, ProjectID: projectID, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(map[string]any{
		"nodes": nodes,
		"edges": edges,
	}))
}

func (h *NodeHandler) Import(c echo.Context) error {
	var req dto.ImportGraphRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	// Classify the import: any node with a description means the payload
	// carries semantic claims, so it must clear write_semantic.
	importKind := audit.MutationStructural
	for _, item := range req.Nodes {
		if item.Description != nil && *item.Description != "" {
			importKind = audit.MutationSemantic
			break
		}
	}
	if err := h.audit.RequirePermission(c, importKind, "write", audit.Event{
		ProjectID: projectID, EntityType: "graph", Action: audit.ActionImported,
	}); err != nil {
		return err
	}

	tx, err := h.nodeRepo.Pool().Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to start transaction"))
	}
	defer tx.Rollback(ctx)

	// Replace mode: delete all existing data
	if req.Mode == "replace" {
		if _, err := tx.Exec(ctx, `DELETE FROM edges WHERE project_id = $1`, projectID); err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to clear edges"))
		}
		if _, err := tx.Exec(ctx, `DELETE FROM nodes WHERE project_id = $1`, projectID); err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to clear nodes"))
		}
	}

	// Create nodes with new IDs, build old→new mapping
	idMap := make(map[string]string, len(req.Nodes))
	createdNodes := make([]model.Node, 0, len(req.Nodes))

	for _, item := range req.Nodes {
		status := model.NodeStatus(item.Status)
		if status == "" {
			status = model.NodeStatusInProgress
		}
		tags := item.Tags
		if tags == nil {
			tags = []string{}
		}

		var node model.Node
		err := tx.QueryRow(ctx,
			`INSERT INTO nodes (project_id, type, title, description, status, tags, position_x, position_y, width, height)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 RETURNING id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at`,
			projectID, item.Type, item.Title, item.Description, status, tags, item.PositionX, item.PositionY, item.Width, item.Height,
		).Scan(&node.ID, &node.ProjectID, &node.Type, &node.Title, &node.Description, &node.Status, &node.AssigneeID, &node.Tags, &node.Metadata, &node.ParentID, &node.PositionX, &node.PositionY, &node.Width, &node.Height, &node.CreatedAt, &node.UpdatedAt)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err(fmt.Sprintf("Failed to create node: %s", item.Title)))
		}

		if item.ID != "" {
			idMap[item.ID] = node.ID
		}
		createdNodes = append(createdNodes, node)
	}

	// Update parentId using the mapping
	for i, item := range req.Nodes {
		if item.ParentID == nil || *item.ParentID == "" {
			continue
		}
		newParentID, ok := idMap[*item.ParentID]
		if !ok {
			continue
		}
		_, err := tx.Exec(ctx,
			`UPDATE nodes SET parent_id = $1, updated_at = now() WHERE id = $2`,
			newParentID, createdNodes[i].ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to set parent"))
		}
		createdNodes[i].ParentID = &newParentID
	}

	// Create edges with remapped IDs
	createdEdges := make([]model.Edge, 0, len(req.Edges))
	for _, item := range req.Edges {
		srcID, srcOK := idMap[item.SourceID]
		tgtID, tgtOK := idMap[item.TargetID]
		if !srcOK || !tgtOK {
			continue // skip edges referencing unknown nodes
		}

		edgeType := model.EdgeType(item.EdgeType)
		if edgeType == "" {
			edgeType = model.EdgeTypeDependsOn
		}

		var edge model.Edge
		err := tx.QueryRow(ctx,
			`INSERT INTO edges (project_id, source_id, target_id, edge_type, label)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id, project_id, source_id, target_id, edge_type, label, created_at`,
			projectID, srcID, tgtID, edgeType, item.Label,
		).Scan(&edge.ID, &edge.ProjectID, &edge.SourceID, &edge.TargetID, &edge.EdgeType, &edge.Label, &edge.CreatedAt)
		if err != nil {
			slog.Warn("import: skipping edge", "source", srcID, "target", tgtID, "error", err)
			continue // skip duplicate or invalid edges
		}
		createdEdges = append(createdEdges, edge)
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to commit import"))
	}

	// One batch event for the import — per-node rows would explode audit_log
	// without adding much investigative value. The batch_id ties any future
	// "what did this import do" queries together.
	batchID := newBatchID()
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "graph",
		Action: audit.ActionImported, MutationKind: importKind,
		BatchID: batchID, Trigger: "import",
		Metadata: map[string]any{
			"mode":  req.Mode,
			"nodes": len(createdNodes),
			"edges": len(createdEdges),
		},
	})

	// For replace mode, return only imported data
	// For merge mode, return full graph
	if req.Mode == "merge" {
		allNodes, _ := h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
		allEdges, _ := h.edgeRepo.FindByProjectID(ctx, projectID)
		if allNodes == nil {
			allNodes = []model.Node{}
		}
		if allEdges == nil {
			allEdges = []model.Edge{}
		}
		return c.JSON(http.StatusOK, dto.OK(map[string]any{"nodes": allNodes, "edges": allEdges}))
	}

	h.hub.Publish(service.Event{Type: service.EventGraphImport, ProjectID: projectID, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(map[string]any{"nodes": createdNodes, "edges": createdEdges}))
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

func (h *NodeHandler) BatchUpdatePositions(c echo.Context) error {
	var req dto.BatchPositionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

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

	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "graph", Action: audit.ActionLayoutComputed,
	}); err != nil {
		return err
	}

	if err := h.nodeRepo.BatchUpdatePositions(ctx, projectID, positions); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update positions"))
	}

	// One batch event for the whole drag/layout pass — emitting N rows per
	// position update would flood audit_log with low-value churn.
	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "graph",
		Action: audit.ActionLayoutComputed, MutationKind: audit.MutationMeta,
		Trigger: "layout", Metadata: map[string]any{"count": len(positions)},
	})
	h.hub.Publish(service.Event{Type: service.EventNodeUpdated, ProjectID: projectID, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *NodeHandler) BatchDelete(c echo.Context) error {
	var req dto.BatchDeleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	if err := h.audit.RequirePermission(c, audit.MutationStructural, "delete", audit.Event{
		ProjectID: projectID, EntityType: "node", Action: audit.ActionDeleted,
	}); err != nil {
		return err
	}

	if err := h.nodeRepo.BatchDelete(ctx, projectID, req.IDs); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete nodes"))
	}

	for _, id := range req.IDs {
		h.audit.Log(c, audit.Event{
			ProjectID: projectID, EntityType: "node", EntityID: id,
			Action: audit.ActionDeleted, MutationKind: audit.MutationStructural, Trigger: "batch",
		})
	}
	h.hub.Publish(service.Event{Type: service.EventNodeDeleted, ProjectID: projectID, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

func (h *NodeHandler) BatchUpdateStatus(c echo.Context) error {
	var req dto.BatchStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", audit.Event{
		ProjectID: projectID, EntityType: "node", Action: audit.ActionStatusChanged, FieldName: "status",
	}); err != nil {
		return err
	}

	if err := h.nodeRepo.BatchUpdateStatus(ctx, projectID, req.IDs, model.NodeStatus(req.Status)); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update status"))
	}

	for _, id := range req.IDs {
		h.audit.Log(c, audit.Event{
			ProjectID: projectID, EntityType: "node", EntityID: id,
			Action: audit.ActionStatusChanged, MutationKind: audit.MutationMeta,
			FieldName: "status", NewValue: req.Status, Trigger: "batch",
		})
	}
	h.hub.Publish(service.Event{Type: service.EventNodeUpdated, ProjectID: projectID, UserID: mw.GetUserID(c)})
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

// mutationKindForNodeField maps a node field to the permission class that
// gates writes to it. Keep in sync with audit.MutationKind constants.
//
//	semantic    — claims about reality (description, "why" content)
//	structural  — graph topology / typing (type, parent_id)
//	meta        — operational state (status, position, tags, assignee)
func mutationKindForNodeField(field string) string {
	switch field {
	case "description":
		return audit.MutationSemantic
	case "type", "parent_id":
		return audit.MutationStructural
	default:
		return audit.MutationMeta
	}
}

// newBatchID returns a UUID-format string suitable for audit_log.batch_id. We
// avoid adding a new dependency by composing 16 random bytes into the standard
// 8-4-4-4-12 format. Collision risk is negligible for batch grouping.
func newBatchID() string {
	var b [16]byte
	if _, err := cryptoRand.Read(b[:]); err != nil {
		// Extremely unlikely; the worst-case is duplicate batch IDs, not an
		// auth/security issue, so we silently fall back to a static value.
		return "00000000-0000-0000-0000-000000000000"
	}
	// RFC 4122 variant + version 4
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
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
