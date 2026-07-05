package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
	"github.com/thask/backend/internal/service"
)

type HistoryRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewHistoryRepo(pool *pgxpool.Pool) *HistoryRepo {
	return &HistoryRepo{pool: pool, q: dbgen.New(pool)}
}

// Create used to INSERT into node_history. As of v0.6.0 (A-2) it is a no-op:
// audit_log is the sole write target for provenance, and node_history is
// dropping in v0.7.0. Signature is kept so handlers don't churn; callers can
// keep their `_ = h.historyRepo.Create(...)` fire-and-forget sites and the
// audit logger picks up the semantics via mutation_kind.
func (r *HistoryRepo) Create(ctx context.Context, nodeID, projectID, userID string, action model.HistoryAction, fieldName, oldValue, newValue *string) error {
	return nil
}

// BatchCreateStatusChanges was a raw-pgx multi-row INSERT into node_history.
// Same deprecation as Create — retained as no-op until v0.7.0 removes the
// call sites.
func (r *HistoryRepo) BatchCreateStatusChanges(ctx context.Context, projectID, userID string, changes []service.StatusChange) {
}

func (r *HistoryRepo) FindByProjectID(ctx context.Context, projectID string, limit int) ([]model.NodeHistoryEntry, error) {
	rows, err := r.q.HistoryFindByProjectID(ctx, dbgen.HistoryFindByProjectIDParams{
		ProjectID: projectID,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	entries := make([]model.NodeHistoryEntry, len(rows))
	for i, row := range rows {
		entries[i] = model.NodeHistoryEntry{
			ID:        row.ID,
			Action:    row.Action,
			FieldName: row.FieldName,
			OldValue:  row.OldValue,
			NewValue:  row.NewValue,
			CreatedAt: row.CreatedAt,
			UserName:  row.DisplayName,
		}
	}
	return entries, nil
}

func (r *HistoryRepo) FindByNodeID(ctx context.Context, nodeID string, limit int) ([]model.NodeHistoryEntry, error) {
	rows, err := r.q.HistoryFindByNodeID(ctx, dbgen.HistoryFindByNodeIDParams{
		EntityID: &nodeID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	entries := make([]model.NodeHistoryEntry, len(rows))
	for i, row := range rows {
		entries[i] = model.NodeHistoryEntry{
			ID:        row.ID,
			Action:    row.Action,
			FieldName: row.FieldName,
			OldValue:  row.OldValue,
			NewValue:  row.NewValue,
			CreatedAt: row.CreatedAt,
			UserName:  row.DisplayName,
		}
	}
	return entries, nil
}

