package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/service"
)

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

// BatchUpdate applies partial updates to up to 200 nodes in a single transaction.
// Atomic on permission denial and on cycle creation; best-effort on per-item
// not-found and no-change (those land in skipped[]). Responds 200 when every
// update applied, 207 when some were skipped.
//
// Use this instead of looping thask.node.update — saves agent context overhead
// and audit_log gets one batch_id grouping the whole pass.
func (h *NodeHandler) BatchUpdate(c echo.Context) error {
	var req dto.BatchNodeUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	userID := mw.GetUserID(c)

	// 1) Pre-flight: which mutation kinds does this batch need, and does any
	//    update actually touch parent_id (drives the cycle check decision)?
	needSemantic, needStructural, needMeta, anyParentChange := false, false, false, false
	for _, u := range req.Updates {
		fields := nonNilFields(u)
		for f := range fields {
			switch mutationKindForNodeField(f) {
			case audit.MutationSemantic:
				needSemantic = true
			case audit.MutationStructural:
				needStructural = true
				if f == "parent_id" {
					anyParentChange = true
				}
			default:
				needMeta = true
			}
		}
	}
	batchEvt := audit.Event{ProjectID: projectID, EntityType: "node", Action: audit.ActionUpdated}
	if needSemantic {
		if err := h.audit.RequirePermission(c, audit.MutationSemantic, "write", batchEvt); err != nil {
			return err
		}
	}
	if needStructural {
		if err := h.audit.RequirePermission(c, audit.MutationStructural, "write", batchEvt); err != nil {
			return err
		}
	}
	if needMeta {
		if err := h.audit.RequirePermission(c, audit.MutationMeta, "write", batchEvt); err != nil {
			return err
		}
	}

	// 2) Open transaction. Every UPDATE + cycle scan + provenance refresh runs
	//    under it; rollback on any failure.
	tx, err := h.nodeRepo.Pool().Begin(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to start transaction"))
	}
	defer tx.Rollback(ctx)

	// Pre-load existing nodes so we can detect not-found and compute delta
	// rows for audit_log without an extra round-trip per item.
	ids := make([]string, 0, len(req.Updates))
	for _, u := range req.Updates {
		ids = append(ids, u.NodeID)
	}
	existing, err := h.fetchNodesForBatch(ctx, tx, projectID, ids)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to load existing nodes"))
	}

	type updated struct {
		NodeID        string   `json:"nodeId"`
		FieldsChanged []string `json:"fieldsChanged"`
	}
	type skipped struct {
		NodeID string `json:"nodeId"`
		Reason string `json:"reason"`
	}
	var ups []updated
	var skips []skipped
	type pendingAudit struct {
		nodeID, field, oldVal, newVal, mutationKind string
	}
	var pending []pendingAudit
	descriptionTouched := []string{}

	for _, u := range req.Updates {
		ex, ok := existing[u.NodeID]
		if !ok {
			skips = append(skips, skipped{NodeID: u.NodeID, Reason: "not_found"})
			continue
		}

		fields := nonNilFields(u)
		// parent_id self / cycle precheck — same rules as singleton Update.
		if v, ok := fields["parent_id"]; ok {
			pid, _ := v.(string)
			if pid == u.NodeID {
				return c.JSON(http.StatusBadRequest, dto.Err("Node cannot be its own parent: "+u.NodeID))
			}
		}

		if len(fields) == 0 {
			skips = append(skips, skipped{NodeID: u.NodeID, Reason: "no_change"})
			continue
		}

		setClauses, args := buildSetClauses(fields)
		args = append(args, u.NodeID)
		query := fmt.Sprintf(
			`UPDATE nodes SET %s WHERE id = $%d`,
			joinClauses(setClauses), len(args),
		)
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			slog.Error("batch_update tx.Exec failed", "nodeId", u.NodeID, "error", err)
			return c.JSON(http.StatusInternalServerError, dto.Err("Failed to apply update for "+u.NodeID))
		}

		changed := make([]string, 0, len(fields))
		exPtr := ex // copy escapes; OK
		for field, val := range fields {
			changed = append(changed, field)
			oldVal := getOldValue(&exPtr, field)
			newVal := fmt.Sprintf("%v", val)
			old := ""
			if oldVal != nil {
				old = *oldVal
			}
			pending = append(pending, pendingAudit{
				nodeID: u.NodeID, field: field, oldVal: old, newVal: newVal,
				mutationKind: mutationKindForNodeField(field),
			})
			if field == "description" {
				descriptionTouched = append(descriptionTouched, u.NodeID)
			}
		}
		ups = append(ups, updated{NodeID: u.NodeID, FieldsChanged: changed})
	}

	// 3) Cycle check — only if anyone touched parent_id. Walks current state
	//    after every applied update inside the tx; rollback on cycle.
	if anyParentChange {
		if cycleNodeID, err := detectProjectCycleTx(ctx, tx, projectID); err != nil {
			return c.JSON(http.StatusInternalServerError, dto.Err("Cycle scan failed"))
		} else if cycleNodeID != "" {
			return c.JSON(http.StatusBadRequest, dto.Err(
				"Batch rejected: parent_id changes would create a cycle involving "+cycleNodeID))
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to commit batch"))
	}

	// 4) Refresh description provenance for non-anonymous human writes and emit
	//    audit_log rows under one shared batch_id. Both run after commit so a
	//    crash here loses traceability but not data.
	if userID != mw.AnonymousUserID {
		src := descriptionSourceForActor(c)
		model := mw.GetAgentModel(c)
		for _, nodeID := range descriptionTouched {
			_ = h.nodeRepo.UpdateDescriptionProvenance(ctx, nodeID, userID, src, model)
		}
	}
	batchID := newBatchID()
	for _, p := range pending {
		auditAction := audit.ActionUpdated
		if p.field == "status" {
			auditAction = audit.ActionStatusChanged
		}
		h.audit.Log(c, audit.Event{
			ProjectID: projectID, EntityType: "node", EntityID: p.nodeID,
			Action: auditAction, FieldName: p.field, MutationKind: p.mutationKind,
			OldValue: p.oldVal, NewValue: p.newVal,
			BatchID: batchID, Trigger: "batch",
		})
	}

	h.hub.Publish(service.Event{Type: service.EventNodeUpdated, ProjectID: projectID, UserID: userID})

	resp := map[string]any{
		"updated": ups,
		"skipped": skips,
		"batchId": batchID,
	}
	status := http.StatusOK
	if len(skips) > 0 {
		status = http.StatusMultiStatus
	}
	return c.JSON(status, dto.OK(resp))
}

// fetchNodesForBatch loads the current row state for every id touched by the
// batch — used to compute audit deltas and the not-found skip set in one
// round-trip. Returns map keyed by node id.
func (h *NodeHandler) fetchNodesForBatch(ctx context.Context, tx pgx.Tx, projectID string, ids []string) (map[string]model.Node, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at
		 FROM nodes WHERE project_id = $1 AND id = ANY($2)`,
		projectID, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]model.Node{}
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out[n.ID] = n
	}
	return out, rows.Err()
}
