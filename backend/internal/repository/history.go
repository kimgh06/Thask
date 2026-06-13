package repository

import (
	"context"
	"fmt"
	"strings"

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

func (r *HistoryRepo) Create(ctx context.Context, nodeID, projectID, userID string, action model.HistoryAction, fieldName, oldValue, newValue *string) error {
	return r.q.HistoryCreate(ctx, dbgen.HistoryCreateParams{
		NodeID:    nodeID,
		ProjectID: projectID,
		UserID:    userID,
		Action:    action,
		FieldName: fieldName,
		OldValue:  oldValue,
		NewValue:  newValue,
	})
}

// BatchCreateStatusChanges uses a dynamic multi-row INSERT and stays as raw pgx.
func (r *HistoryRepo) BatchCreateStatusChanges(ctx context.Context, projectID, userID string, changes []service.StatusChange) {
	if len(changes) == 0 {
		return
	}
	values := make([]string, len(changes))
	args := []any{projectID, userID, string(model.HistoryActionStatusChanged), "status"}
	for i, wc := range changes {
		base := len(args)
		values[i] = fmt.Sprintf("($%d, $1, $2, $3, $4, $%d, $%d)", base+1, base+2, base+3)
		args = append(args, wc.NodeID, string(wc.OldStatus), string(wc.NewStatus))
	}
	query := `INSERT INTO node_history (node_id, project_id, user_id, action, field_name, old_value, new_value) VALUES ` + strings.Join(values, ", ")
	r.pool.Exec(ctx, query, args...)
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
			Action:    historyActionFromIface(row.Action),
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
		NodeID: nodeID,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	entries := make([]model.NodeHistoryEntry, len(rows))
	for i, row := range rows {
		entries[i] = model.NodeHistoryEntry{
			ID:        row.ID,
			Action:    historyActionFromIface(row.Action),
			FieldName: row.FieldName,
			OldValue:  row.OldValue,
			NewValue:  row.NewValue,
			CreatedAt: row.CreatedAt,
			UserName:  row.DisplayName,
		}
	}
	return entries, nil
}

// historyActionFromIface converts the interface{} returned by sqlc for the
// history_action enum column to model.HistoryAction.
func historyActionFromIface(v interface{}) model.HistoryAction {
	switch s := v.(type) {
	case string:
		return model.HistoryAction(s)
	case []byte:
		return model.HistoryAction(s)
	default:
		return model.HistoryAction(fmt.Sprintf("%v", v))
	}
}
