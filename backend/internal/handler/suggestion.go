package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/repository"
)

// SuggestionHandler implements the human-in-the-loop queue for agent-proposed
// node updates. Agents POST to /suggestions; humans PATCH to decide.
type SuggestionHandler struct {
	repo     *repository.SuggestionRepo
	nodeRepo *repository.NodeRepo
	audit    *audit.Logger
}

func NewSuggestionHandler(repo *repository.SuggestionRepo, nodeRepo *repository.NodeRepo, auditLogger *audit.Logger) *SuggestionHandler {
	return &SuggestionHandler{repo: repo, nodeRepo: nodeRepo, audit: auditLogger}
}

// Suggest accepts a proposed update. Allowed for keys with permissions.suggest.
// The queue rather than the node is what gets touched, so this never affects
// trust state until a human decides.
func (h *SuggestionHandler) Suggest(c echo.Context) error {
	var req dto.SuggestNodeUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if req.FieldName == "" {
		req.FieldName = "description"
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")
	userID := mw.GetUserID(c)
	if userID == mw.AnonymousUserID {
		return c.JSON(http.StatusForbidden, dto.Err("Anonymous shared access cannot queue suggestions"))
	}

	if err := h.audit.RequirePermission(c, audit.MutationSemantic, "suggest", audit.Event{
		ProjectID: projectID, EntityType: "suggestion", Action: audit.ActionSuggested, FieldName: req.FieldName,
	}); err != nil {
		return err
	}

	existing, ferr := h.nodeRepo.FindByID(ctx, nodeID, projectID)
	if ferr != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Node not found"))
	}

	var currentValue *string
	switch req.FieldName {
	case "description":
		currentValue = existing.Description
	case "title":
		v := existing.Title
		currentValue = &v
	}

	var rationale *string
	if req.Rationale != "" {
		rationale = &req.Rationale
	}
	agentModel := mw.GetAgentModel(c)
	sessionID := mw.GetAgentSessionID(c)
	var agentModelPtr, sessionPtr *string
	if agentModel != "" {
		agentModelPtr = &agentModel
	}
	if sessionID != "" {
		sessionPtr = &sessionID
	}

	row, err := h.repo.Create(ctx, &model.NodeSuggestion{
		ProjectID:      projectID,
		NodeID:         nodeID,
		FieldName:      req.FieldName,
		ProposedValue:  req.ProposedValue,
		CurrentValue:   currentValue,
		Rationale:      rationale,
		Evidence:       req.Evidence,
		ProposedBy:     userID,
		AgentModel:     agentModelPtr,
		AgentSessionID: sessionPtr,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to queue suggestion"))
	}

	// Newer suggestion supersedes older pending ones on the same field so the
	// reviewer never sees duplicate stacks.
	_ = h.repo.SupersedePendingForNode(ctx, nodeID, req.FieldName, row.ID)

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "suggestion", EntityID: row.ID,
		Action: audit.ActionSuggested, MutationKind: audit.MutationSemantic,
		FieldName: req.FieldName, NewValue: req.ProposedValue,
		Metadata: map[string]any{"node_id": nodeID, "agent_model": agentModel},
	})
	return c.JSON(http.StatusCreated, dto.OK(row))
}

// List returns pending suggestions for review.
func (h *SuggestionHandler) List(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	rows, err := h.repo.ListPending(ctx, projectID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to list suggestions"))
	}
	if rows == nil {
		rows = []model.NodeSuggestion{}
	}
	return c.JSON(http.StatusOK, dto.OK(rows))
}

// Decide accepts or rejects a suggestion. Accepting copies proposed_value into
// the target node and stamps provenance to the deciding human; rejecting just
// marks the row and leaves the node untouched.
func (h *SuggestionHandler) Decide(c echo.Context) error {
	var req dto.DecideSuggestionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err("Invalid request body"))
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, dto.Err(err.Error()))
	}

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	suggestionID := c.Param("suggestionId")
	userID := mw.GetUserID(c)
	if userID == mw.AnonymousUserID {
		return c.JSON(http.StatusForbidden, dto.Err("Anonymous shared access cannot decide suggestions"))
	}

	// Deciding is a meta operation on the queue, but accepting flips a
	// semantic field on the underlying node — gate accordingly.
	mk := audit.MutationMeta
	if req.Status == "accepted" {
		mk = audit.MutationSemantic
	}
	if err := h.audit.RequirePermission(c, mk, "write", audit.Event{
		ProjectID: projectID, EntityType: "suggestion", EntityID: suggestionID, Action: audit.ActionSuggestDecided,
	}); err != nil {
		return err
	}

	// Policy: only user_interactive actors may accept. The whole point of the
	// queue is human-in-the-loop — a service or agent key with write_semantic
	// enabled is still not the human reviewer the design promises.
	if req.Status == "accepted" && mw.GetActorKind(c) != string(model.APIKeyKindUserInteractive) {
		h.audit.Log(c, audit.Event{
			ProjectID: projectID, EntityType: "suggestion", EntityID: suggestionID,
			Action: audit.ActionWriteDenied, MutationKind: audit.MutationSemantic,
			Metadata: map[string]any{
				"reason":     "accept reserved for user_interactive actors",
				"actor_kind": mw.GetActorKind(c),
			},
		})
		return c.JSON(http.StatusForbidden, dto.Err(
			"Accepting a suggestion is reserved for user_interactive API keys (humans). "+
				"Use a browser session or a user_interactive key to review."))
	}

	existing, err := h.repo.FindByID(ctx, suggestionID)
	if err != nil {
		return c.JSON(http.StatusNotFound, dto.Err("Suggestion not found"))
	}
	if existing.ProjectID != projectID {
		return c.JSON(http.StatusNotFound, dto.Err("Suggestion not found"))
	}

	if err := h.repo.Decide(ctx, suggestionID, req.Status, userID, req.Reason); err != nil {
		return c.JSON(http.StatusConflict, dto.Err(err.Error()))
	}

	// Accept = apply the change. The human is the author of record, not the
	// agent — provenance reflects who pressed the button, with the agent
	// only credited in audit_log metadata.
	if req.Status == "accepted" && existing.FieldName == "description" {
		if _, uerr := h.nodeRepo.Update(ctx, existing.NodeID, map[string]any{
			"description": existing.ProposedValue,
		}); uerr == nil {
			source := "human"
			agentModel := ""
			if existing.AgentModel != nil {
				agentModel = *existing.AgentModel
			}
			_ = h.nodeRepo.UpdateDescriptionProvenance(ctx, existing.NodeID, userID, source, agentModel)
		}
	}

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "suggestion", EntityID: suggestionID,
		Action: audit.ActionSuggestDecided, MutationKind: mk,
		FieldName: existing.FieldName, NewValue: req.Status,
		Metadata: map[string]any{
			"node_id":         existing.NodeID,
			"reason":          req.Reason,
			"original_author": existing.ProposedBy,
			"agent_model":     existing.AgentModel,
		},
	})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}

// Verify is the human-only "still correct" 1-click for a node. Updates
// last_verified_at / by / commit. Permission gate: requires verify=true.
func (h *SuggestionHandler) Verify(c echo.Context) error {
	var req dto.VerifyNodeRequest
	_ = c.Bind(&req) // body is optional

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)
	nodeID := c.Param("nodeId")
	userID := mw.GetUserID(c)
	if userID == mw.AnonymousUserID {
		return c.JSON(http.StatusForbidden, dto.Err("Anonymous shared access cannot verify nodes"))
	}

	if err := h.audit.RequirePermission(c, audit.MutationMeta, "verify", audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID, Action: audit.ActionVerified,
	}); err != nil {
		return err
	}

	if err := h.nodeRepo.MarkVerified(ctx, nodeID, projectID, userID, req.Commit); err != nil {
		if err.Error() == "node not found" {
			return c.JSON(http.StatusNotFound, dto.Err("Node not found"))
		}
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to verify node"))
	}

	h.audit.Log(c, audit.Event{
		ProjectID: projectID, EntityType: "node", EntityID: nodeID,
		Action: audit.ActionVerified, MutationKind: audit.MutationMeta,
		CodeCommit: req.Commit,
	})
	return c.JSON(http.StatusOK, dto.OK(dto.SuccessResponse{Success: true}))
}
