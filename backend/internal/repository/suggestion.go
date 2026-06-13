package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

// SuggestionRepo persists agent-proposed updates to node_suggestions (migration 009).
// Agents call Create to enqueue a proposal; humans call Decide to accept/reject.
type SuggestionRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewSuggestionRepo(pool *pgxpool.Pool) *SuggestionRepo {
	return &SuggestionRepo{pool: pool, q: dbgen.New(pool)}
}

// suggestionFromRow converts a dbgen.NodeSuggestion to model.NodeSuggestion.
// Evidence is stored as JSONB ([]byte in dbgen) but typed as any in the model.
func suggestionFromRow(r dbgen.NodeSuggestion) model.NodeSuggestion {
	s := model.NodeSuggestion{
		ID:             r.ID,
		ProjectID:      r.ProjectID,
		NodeID:         r.NodeID,
		FieldName:      r.FieldName,
		ProposedValue:  r.ProposedValue,
		CurrentValue:   r.CurrentValue,
		Rationale:      r.Rationale,
		ProposedBy:     r.ProposedBy,
		AgentModel:     r.AgentModel,
		AgentSessionID: r.AgentSessionID,
		Status:         r.Status,
		DecidedBy:      r.DecidedBy,
		DecidedAt:      r.DecidedAt,
		DecidedReason:  r.DecidedReason,
		CreatedAt:      r.CreatedAt,
	}
	if len(r.Evidence) > 0 {
		var ev any
		if err := json.Unmarshal(r.Evidence, &ev); err == nil {
			s.Evidence = ev
		} else {
			s.Evidence = r.Evidence
		}
	}
	return s
}

// evidenceBytes marshals the model Evidence (any) to []byte for storage.
// `evidence` is JSONB NOT NULL DEFAULT '{}' in the schema (migration 009),
// so nil / unmarshal error must map to `{}` not NULL to avoid 23502.
func evidenceBytes(ev any) []byte {
	if ev == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// Create enqueues a suggestion. The same agent_session_id may stack multiple
// pending suggestions on the same node; the human deciding can supersede
// older ones.
func (r *SuggestionRepo) Create(ctx context.Context, s *model.NodeSuggestion) (*model.NodeSuggestion, error) {
	row, err := r.q.SuggestionCreate(ctx, dbgen.SuggestionCreateParams{
		ProjectID:      s.ProjectID,
		NodeID:         s.NodeID,
		FieldName:      s.FieldName,
		ProposedValue:  s.ProposedValue,
		CurrentValue:   s.CurrentValue,
		Rationale:      s.Rationale,
		Evidence:       evidenceBytes(s.Evidence),
		ProposedBy:     s.ProposedBy,
		AgentModel:     s.AgentModel,
		AgentSessionID: s.AgentSessionID,
	})
	if err != nil {
		return nil, err
	}
	s.ID = row.ID
	s.Status = row.Status
	s.CreatedAt = row.CreatedAt
	return s, nil
}

// ListPending returns suggestions awaiting human review for a project, newest first.
func (r *SuggestionRepo) ListPending(ctx context.Context, projectID string, limit int) ([]model.NodeSuggestion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.q.SuggestionListPending(ctx, dbgen.SuggestionListPendingParams{
		ProjectID: projectID,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.NodeSuggestion, len(rows))
	for i, row := range rows {
		out[i] = suggestionFromRow(row)
	}
	return out, nil
}

// ListForNode returns suggestion history for a single node (any status).
func (r *SuggestionRepo) ListForNode(ctx context.Context, nodeID string, limit int) ([]model.NodeSuggestion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.q.SuggestionListForNode(ctx, dbgen.SuggestionListForNodeParams{
		NodeID: nodeID,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.NodeSuggestion, len(rows))
	for i, row := range rows {
		out[i] = suggestionFromRow(row)
	}
	return out, nil
}

// FindByID loads a single suggestion (used to verify ownership / fetch fields
// before deciding).
func (r *SuggestionRepo) FindByID(ctx context.Context, id string) (*model.NodeSuggestion, error) {
	row, err := r.q.SuggestionFindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s := suggestionFromRow(row)
	return &s, nil
}

// Decide marks a suggestion accepted or rejected. status must be 'accepted'
// or 'rejected'. Only pending suggestions can be decided; deciding twice is
// a no-op (returns the existing row).
func (r *SuggestionRepo) Decide(ctx context.Context, id, status, deciderID, reason string) error {
	if status != "accepted" && status != "rejected" {
		return fmt.Errorf("invalid status: %s", status)
	}
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	n, err := r.q.SuggestionDecide(ctx, dbgen.SuggestionDecideParams{
		Status:        status,
		DecidedBy:     &deciderID,
		DecidedReason: reasonPtr,
		ID:            id,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("suggestion not found or already decided")
	}
	return nil
}

// SupersedePendingForNode marks any earlier pending suggestions on the same
// node+field as 'superseded' so the queue doesn't pile up duplicates.
func (r *SuggestionRepo) SupersedePendingForNode(ctx context.Context, nodeID, fieldName, exceptID string) error {
	return r.q.SuggestionSupersedePendingForNode(ctx, dbgen.SuggestionSupersedePendingForNodeParams{
		NodeID:    nodeID,
		FieldName: fieldName,
		ID:        exceptID,
	})
}
