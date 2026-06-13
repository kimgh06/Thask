package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

// AuditRepo writes rows into audit_log (migration 006). One row per write event.
type AuditRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool, q: dbgen.New(pool)}
}

// Insert persists an AuditLog row. All optional fields may be nil; the schema
// supplies defaults for mutation_kind, actor_kind, and metadata when absent.
func (r *AuditRepo) Insert(ctx context.Context, e *model.AuditLog) error {
	return r.q.AuditLogInsert(ctx, dbgen.AuditLogInsertParams{
		ProjectID:      e.ProjectID,
		EntityType:     e.EntityType,
		EntityID:       e.EntityID,
		Action:         e.Action,
		FieldName:      e.FieldName,
		OldValue:       e.OldValue,
		NewValue:       e.NewValue,
		MutationKind:   e.MutationKind,
		UserID:         e.UserID,
		ApiKeyID:       e.APIKeyID,
		ActorKind:      e.ActorKind,
		ClientType:     e.ClientType,
		ClientVersion:  e.ClientVersion,
		AgentModel:     e.AgentModel,
		AgentSessionID: e.AgentSessionID,
		Trigger:        e.Trigger,
		BatchID:        e.BatchID,
		CodeCommit:     e.CodeCommit,
		SourcePath:     e.SourcePath,
		Confidence:     e.Confidence,
		Metadata:       marshalAuditMetadata(e.Metadata),
	})
}

// ListByEntity returns audit rows for a single node/edge in reverse chronological
// order. Used by thask.node.history and the suggestion review UI.
func (r *AuditRepo) ListByEntity(ctx context.Context, entityType, entityID string, limit int) ([]model.AuditLog, error) {
	rows, err := r.q.AuditLogListByEntity(ctx, dbgen.AuditLogListByEntityParams{
		EntityType: entityType,
		EntityID:   &entityID,
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]model.AuditLog, len(rows))
	for i, row := range rows {
		out[i] = auditLogFromRow(row)
	}
	return out, nil
}

// marshalAuditMetadata converts model.AuditLog.Metadata (any) to json.RawMessage
// for the dbgen insert param. `metadata jsonb NOT NULL DEFAULT '{}'::jsonb`
// (migration 006) — never return nil; the Logger swallows insert errors, so a
// SQL NULL here silently drops the entire audit row and breaks the Phase 11
// "every mutation writes one row" invariant.
func marshalAuditMetadata(v any) json.RawMessage {
	if v == nil {
		return json.RawMessage("{}")
	}
	if raw, ok := v.(json.RawMessage); ok {
		if len(raw) == 0 {
			return json.RawMessage("{}")
		}
		return raw
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

func auditLogFromRow(r dbgen.AuditLog) model.AuditLog {
	return model.AuditLog{
		ID:             r.ID,
		ProjectID:      r.ProjectID,
		EntityType:     r.EntityType,
		EntityID:       r.EntityID,
		Action:         r.Action,
		FieldName:      r.FieldName,
		OldValue:       r.OldValue,
		NewValue:       r.NewValue,
		MutationKind:   r.MutationKind,
		UserID:         r.UserID,
		APIKeyID:       r.ApiKeyID,
		ActorKind:      r.ActorKind,
		ClientType:     r.ClientType,
		ClientVersion:  r.ClientVersion,
		AgentModel:     r.AgentModel,
		AgentSessionID: r.AgentSessionID,
		Trigger:        r.Trigger,
		BatchID:        r.BatchID,
		CodeCommit:     r.CodeCommit,
		SourcePath:     r.SourcePath,
		Confidence:     r.Confidence,
		Metadata:       r.Metadata,
		CreatedAt:      r.CreatedAt,
	}
}
