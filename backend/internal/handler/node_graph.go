package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/service"
)

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
		// graph.import creates NEW nodes — type+title are required per item.
		// Agents trying to partial-update existing nodes via "merge" hit this
		// path; nudge them toward the right tool.
		return c.JSON(http.StatusBadRequest, dto.Err(
			err.Error()+
				" — graph.import creates new nodes (each item requires type + title)."+
				" To patch existing nodes by id use node.update or node.batch_update."+
				" To add edges between existing nodes use edge.batch_create.",
		))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	userID := mw.GetUserID(c)
	var createdBy *string
	if userID != "" {
		uid := userID
		createdBy = &uid
	}

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
			`INSERT INTO nodes (project_id, type, title, description, status, tags, position_x, position_y, width, height, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			 RETURNING id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at`,
			projectID, item.Type, item.Title, item.Description, status, tags, item.PositionX, item.PositionY, item.Width, item.Height, createdBy,
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
