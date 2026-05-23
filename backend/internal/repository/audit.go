package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

// AuditRepo writes rows into audit_log (migration 006). One row per write event.
type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

// Insert persists an AuditLog row. All optional fields may be nil; the schema
// supplies defaults for mutation_kind, actor_kind, and metadata when absent.
func (r *AuditRepo) Insert(ctx context.Context, e *model.AuditLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_log (
			project_id, entity_type, entity_id, action, field_name,
			old_value, new_value, mutation_kind, user_id, api_key_id,
			actor_kind, client_type, client_version, agent_model, agent_session_id,
			trigger, batch_id, code_commit, source_path, confidence, metadata
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)`,
		e.ProjectID, e.EntityType, e.EntityID, e.Action, e.FieldName,
		e.OldValue, e.NewValue, e.MutationKind, e.UserID, e.APIKeyID,
		e.ActorKind, e.ClientType, e.ClientVersion, e.AgentModel, e.AgentSessionID,
		e.Trigger, e.BatchID, e.CodeCommit, e.SourcePath, e.Confidence, e.Metadata,
	)
	return err
}

// ListByEntity returns audit rows for a single node/edge in reverse chronological
// order. Used by thask.node.history and the suggestion review UI.
func (r *AuditRepo) ListByEntity(ctx context.Context, entityType, entityID string, limit int) ([]model.AuditLog, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, entity_type, entity_id, action, field_name,
		        old_value, new_value, mutation_kind, user_id, api_key_id,
		        actor_kind, client_type, client_version, agent_model, agent_session_id,
		        trigger, batch_id, code_commit, source_path, confidence, metadata, created_at
		 FROM audit_log
		 WHERE entity_type = $1 AND entity_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3`, entityType, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.AuditLog
	for rows.Next() {
		var e model.AuditLog
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.EntityType, &e.EntityID, &e.Action, &e.FieldName,
			&e.OldValue, &e.NewValue, &e.MutationKind, &e.UserID, &e.APIKeyID,
			&e.ActorKind, &e.ClientType, &e.ClientVersion, &e.AgentModel, &e.AgentSessionID,
			&e.Trigger, &e.BatchID, &e.CodeCommit, &e.SourcePath, &e.Confidence, &e.Metadata, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
