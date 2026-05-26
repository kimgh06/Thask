package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
	"github.com/thask/backend/internal/service"
)

type EdgeHandler struct {
	edgeRepo *repository.EdgeRepo
	hub      *service.Hub
	audit    *audit.Logger
}

func NewEdgeHandler(edgeRepo *repository.EdgeRepo, hub *service.Hub, auditLogger *audit.Logger) *EdgeHandler {
	return &EdgeHandler{edgeRepo: edgeRepo, hub: hub, audit: auditLogger}
}

func (h *EdgeHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

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

		edges, hasMore, err := h.edgeRepo.FindByProjectIDPaginated(ctx, projectID, limit, afterTime, afterID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.V1Err(500, "Failed to fetch edges"))
		}
		if edges == nil {
			edges = []model.Edge{}
		}

		var nextCursor *string
		if hasMore && len(edges) > 0 {
			last := edges[len(edges)-1]
			c := dto.EncodeCursor(last.CreatedAt, last.ID)
			nextCursor = &c
		}
		return c.JSON(http.StatusOK, dto.PaginatedResponse{
			Data:       edges,
			Pagination: dto.PaginationMeta{Limit: limit, HasMore: hasMore, NextCursor: nextCursor},
		})
	}

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch edges"))
	}
	if edges == nil {
		edges = []model.Edge{}
	}

	return c.JSON(http.StatusOK, dto.OK(edges))
}

func (h *EdgeHandler) Create(c echo.Context) error {
	var req dto.CreateEdgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	if req.SourceID == req.TargetID {
		return c.JSON(http.StatusBadRequest, dto.Err("Self-referencing edges are not allowed"))
	}

	// Edges are structural — they describe graph topology, not claims.
	if err := h.audit.RequirePermission(c, audit.MutationStructural, "write", audit.Event{
		ProjectID: projectID, EntityType: "edge", Action: audit.ActionCreated,
	}); err != nil {
		return err
	}

	edgeType := model.EdgeType(req.EdgeType)
	if edgeType == "" {
		edgeType = model.EdgeTypeRelated
	}

	var edge *model.Edge
	var err error
	if req.SourcePort != "" || req.TargetPort != "" || req.Waypoints != nil {
		edge, err = h.edgeRepo.CreateWithRouting(ctx, projectID, req.SourceID, req.TargetID, edgeType, req.Label, req.SourcePort, req.TargetPort, req.Waypoints)
	} else {
		edge, err = h.edgeRepo.Create(ctx, projectID, req.SourceID, req.TargetID, edgeType, req.Label)
	}
	if err != nil {
		return c.JSON(http.StatusConflict, dto.Err("Edge already exists or invalid"))
	}

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "edge", EntityID: edge.ID,
		Action: audit.ActionCreated, MutationKind: audit.MutationStructural,
		NewValue: req.SourceID + "→" + req.TargetID + " (" + string(edgeType) + ")",
	})
	h.hub.Publish(service.Event{Type: service.EventEdgeCreated, ProjectID: projectID, Data: edge, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusCreated, dto.OK(edge))
}

func (h *EdgeHandler) Update(c echo.Context) error {
	var req dto.UpdateEdgeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	edgeID := c.Param("edgeId")

	// Verify edge belongs to this project
	if err := h.edgeRepo.VerifyProject(ctx, edgeID, projectID); err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Edge not found"))
	}

	// Routing / label / type are all structural concerns for the graph.
	if err := h.audit.RequirePermission(c, audit.MutationStructural, "write", audit.Event{
		ProjectID: projectID, EntityType: "edge", EntityID: edgeID, Action: audit.ActionUpdated,
	}); err != nil {
		return err
	}

	var edgeType *model.EdgeType
	if req.EdgeType != nil {
		et := model.EdgeType(*req.EdgeType)
		edgeType = &et
	}

	edge, err := h.edgeRepo.Update(ctx, edgeID, edgeType, req.Label)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update edge"))
	}

	// Update routing if provided
	if req.SourcePort != nil || req.TargetPort != nil || req.Waypoints != nil {
		edge, err = h.edgeRepo.UpdateRouting(ctx, edgeID, req.SourcePort, req.TargetPort, req.Waypoints)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to update edge routing"))
		}
	}

	h.audit.Log(c, audit.Event{
		ProjectID: edge.ProjectID, EntityType: "edge", EntityID: edge.ID,
		Action: audit.ActionUpdated, MutationKind: audit.MutationStructural,
	})
	h.hub.Publish(service.Event{Type: service.EventEdgeUpdated, ProjectID: edge.ProjectID, Data: edge, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(edge))
}

func (h *EdgeHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	edgeID := c.Param("edgeId")

	if err := h.audit.RequirePermission(c, audit.MutationStructural, "delete", audit.Event{
		ProjectID: projectID, EntityType: "edge", EntityID: edgeID, Action: audit.ActionDeleted,
	}); err != nil {
		return err
	}

	if err := h.edgeRepo.DeleteScoped(ctx, edgeID, projectID); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete edge"))
	}

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "edge", EntityID: edgeID,
		Action: audit.ActionDeleted, MutationKind: audit.MutationStructural,
	})
	h.hub.Publish(service.Event{Type: service.EventEdgeDeleted, ProjectID: projectID, Data: edgeID, UserID: mw.GetUserID(c)})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// BatchCreate inserts up to 500 edges in one transaction. Permission-gated
// atomically. Per-item skip reasons returned for self-references, duplicates,
// and edges referencing missing nodes — response is 207 when any are skipped.
func (h *EdgeHandler) BatchCreate(c echo.Context) error {
	var req dto.BatchEdgeCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	if err := h.audit.RequirePermission(c, audit.MutationStructural, "write", audit.Event{
		ProjectID: projectID, EntityType: "edge", Action: audit.ActionCreated,
	}); err != nil {
		return err
	}

	tx, err := h.edgeRepo.Pool().Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to start transaction"))
	}
	defer tx.Rollback(ctx)

	type createdEdge struct {
		ID       string `json:"id"`
		SourceID string `json:"sourceId"`
		TargetID string `json:"targetId"`
		EdgeType string `json:"edgeType"`
	}
	type skipped struct {
		SourceID string `json:"sourceId"`
		TargetID string `json:"targetId"`
		Reason   string `json:"reason"`
	}
	var ups []createdEdge
	var skips []skipped

	for _, item := range req.Edges {
		if item.SourceID == item.TargetID {
			skips = append(skips, skipped{item.SourceID, item.TargetID, "self_reference"})
			continue
		}
		edgeType := item.EdgeType
		if edgeType == "" {
			edgeType = string(model.EdgeTypeRelated)
		}
		// Per-edge insert under tx; UNIQUE(source_id, target_id, edge_type)
		// raises on duplicate, FK fails on missing node — both reported as skip.
		var id string
		err := tx.QueryRow(ctx,
			`INSERT INTO edges (project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (source_id, target_id, edge_type) DO NOTHING
			 RETURNING id`,
			projectID, item.SourceID, item.TargetID, edgeType, item.Label,
			nullableString(item.SourcePort, "auto"), nullableString(item.TargetPort, "auto"), item.Waypoints,
		).Scan(&id)
		if err != nil {
			// pgx.ErrNoRows fires when ON CONFLICT DO NOTHING skips the row;
			// anything else (FK violation on source/target, etc.) is a payload
			// problem we surface as "invalid_endpoint".
			if errors.Is(err, pgx.ErrNoRows) {
				skips = append(skips, skipped{item.SourceID, item.TargetID, "duplicate"})
				continue
			}
			skips = append(skips, skipped{item.SourceID, item.TargetID, "invalid_endpoint"})
			continue
		}
		ups = append(ups, createdEdge{id, item.SourceID, item.TargetID, edgeType})
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to commit batch"))
	}

	batchID := newBatchID()
	for _, e := range ups {
		h.audit.Log(c, audit.Event{
			ProjectID: projectID, EntityType: "edge", EntityID: e.ID,
			Action: audit.ActionCreated, MutationKind: audit.MutationStructural,
			NewValue: e.SourceID + "→" + e.TargetID + " (" + e.EdgeType + ")",
			BatchID:  batchID, Trigger: "batch",
		})
	}
	h.hub.Publish(service.Event{Type: service.EventEdgeCreated, ProjectID: projectID, UserID: mw.GetUserID(c)})

	resp := map[string]any{"created": ups, "skipped": skips, "batchId": batchID}
	status := http.StatusOK
	if len(skips) > 0 {
		status = http.StatusMultiStatus
	}
	return c.JSON(status, dto.OK(resp))
}

// BatchDelete removes up to 500 edges by id under one transaction.
// Missing ids appear in skipped[]; permission errors reject the batch.
func (h *EdgeHandler) BatchDelete(c echo.Context) error {
	var req dto.BatchEdgeDeleteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	if err := h.audit.RequirePermission(c, audit.MutationStructural, "delete", audit.Event{
		ProjectID: projectID, EntityType: "edge", Action: audit.ActionDeleted,
	}); err != nil {
		return err
	}

	tx, err := h.edgeRepo.Pool().Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to start transaction"))
	}
	defer tx.Rollback(ctx)

	type skipped struct {
		EdgeID string `json:"edgeId"`
		Reason string `json:"reason"`
	}
	deleted := []string{}
	var skips []skipped

	for _, id := range req.EdgeIDs {
		tag, err := tx.Exec(ctx, `DELETE FROM edges WHERE id = $1 AND project_id = $2`, id, projectID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to delete edge "+id))
		}
		if tag.RowsAffected() == 0 {
			skips = append(skips, skipped{id, "not_found"})
			continue
		}
		deleted = append(deleted, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to commit batch"))
	}

	batchID := newBatchID()
	for _, id := range deleted {
		h.audit.Log(c, audit.Event{
			ProjectID: projectID, EntityType: "edge", EntityID: id,
			Action: audit.ActionDeleted, MutationKind: audit.MutationStructural,
			BatchID: batchID, Trigger: "batch",
		})
	}
	h.hub.Publish(service.Event{Type: service.EventEdgeDeleted, ProjectID: projectID, UserID: mw.GetUserID(c)})

	resp := map[string]any{"deleted": deleted, "skipped": skips, "batchId": batchID}
	status := http.StatusOK
	if len(skips) > 0 {
		status = http.StatusMultiStatus
	}
	return c.JSON(status, dto.OK(resp))
}

// nullableString returns fallback when v is empty; lets BatchEdgeCreate keep
// the existing "auto" port default without repeating the literal everywhere.
func nullableString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// newBatchID is defined in node.go and reused here for edge batches so all
// audit_log rows from a single bulk request share the same group id.
