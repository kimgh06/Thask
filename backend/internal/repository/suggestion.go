package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

// SuggestionRepo persists agent-proposed updates to node_suggestions (migration 009).
// Agents call Create to enqueue a proposal; humans call Decide to accept/reject.
type SuggestionRepo struct {
	pool *pgxpool.Pool
}

func NewSuggestionRepo(pool *pgxpool.Pool) *SuggestionRepo {
	return &SuggestionRepo{pool: pool}
}

// Create enqueues a suggestion. The same agent_session_id may stack multiple
// pending suggestions on the same node; the human deciding can supersede
// older ones.
func (r *SuggestionRepo) Create(ctx context.Context, s *model.NodeSuggestion) (*model.NodeSuggestion, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO node_suggestions (
			project_id, node_id, field_name, proposed_value, current_value,
			rationale, evidence, proposed_by, agent_model, agent_session_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, status, created_at`,
		s.ProjectID, s.NodeID, s.FieldName, s.ProposedValue, s.CurrentValue,
		s.Rationale, s.Evidence, s.ProposedBy, s.AgentModel, s.AgentSessionID,
	).Scan(&s.ID, &s.Status, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListPending returns suggestions awaiting human review for a project, newest first.
func (r *SuggestionRepo) ListPending(ctx context.Context, projectID string, limit int) ([]model.NodeSuggestion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, node_id, field_name, proposed_value, current_value,
		        rationale, evidence, proposed_by, agent_model, agent_session_id,
		        status, decided_by, decided_at, decided_reason, created_at
		 FROM node_suggestions
		 WHERE project_id = $1 AND status = 'pending'
		 ORDER BY created_at DESC
		 LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSuggestions(rows)
}

// ListForNode returns suggestion history for a single node (any status).
func (r *SuggestionRepo) ListForNode(ctx context.Context, nodeID string, limit int) ([]model.NodeSuggestion, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, node_id, field_name, proposed_value, current_value,
		        rationale, evidence, proposed_by, agent_model, agent_session_id,
		        status, decided_by, decided_at, decided_reason, created_at
		 FROM node_suggestions
		 WHERE node_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, nodeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSuggestions(rows)
}

// FindByID loads a single suggestion (used to verify ownership / fetch fields
// before deciding).
func (r *SuggestionRepo) FindByID(ctx context.Context, id string) (*model.NodeSuggestion, error) {
	var s model.NodeSuggestion
	err := r.pool.QueryRow(ctx,
		`SELECT id, project_id, node_id, field_name, proposed_value, current_value,
		        rationale, evidence, proposed_by, agent_model, agent_session_id,
		        status, decided_by, decided_at, decided_reason, created_at
		 FROM node_suggestions WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.ProjectID, &s.NodeID, &s.FieldName, &s.ProposedValue, &s.CurrentValue,
		&s.Rationale, &s.Evidence, &s.ProposedBy, &s.AgentModel, &s.AgentSessionID,
		&s.Status, &s.DecidedBy, &s.DecidedAt, &s.DecidedReason, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
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
	tag, err := r.pool.Exec(ctx,
		`UPDATE node_suggestions
		 SET status = $1, decided_by = $2, decided_at = now(), decided_reason = $3
		 WHERE id = $4 AND status = 'pending'`,
		status, deciderID, reasonPtr, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("suggestion not found or already decided")
	}
	return nil
}

// SupersedePendingForNode marks any earlier pending suggestions on the same
// node+field as 'superseded' so the queue doesn't pile up duplicates.
func (r *SuggestionRepo) SupersedePendingForNode(ctx context.Context, nodeID, fieldName, exceptID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE node_suggestions
		 SET status = 'superseded', decided_at = now()
		 WHERE node_id = $1 AND field_name = $2 AND status = 'pending' AND id != $3`,
		nodeID, fieldName, exceptID)
	return err
}

func scanSuggestions(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]model.NodeSuggestion, error) {
	var out []model.NodeSuggestion
	for rows.Next() {
		var s model.NodeSuggestion
		if err := rows.Scan(
			&s.ID, &s.ProjectID, &s.NodeID, &s.FieldName, &s.ProposedValue, &s.CurrentValue,
			&s.Rationale, &s.Evidence, &s.ProposedBy, &s.AgentModel, &s.AgentSessionID,
			&s.Status, &s.DecidedBy, &s.DecidedAt, &s.DecidedReason, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
