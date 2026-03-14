package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type HistoryRepo struct {
	pool *pgxpool.Pool
}

func NewHistoryRepo(pool *pgxpool.Pool) *HistoryRepo {
	return &HistoryRepo{pool: pool}
}

func (r *HistoryRepo) Create(ctx context.Context, nodeID, projectID, userID string, action model.HistoryAction, fieldName, oldValue, newValue *string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO node_history (node_id, project_id, user_id, action, field_name, old_value, new_value)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		nodeID, projectID, userID, action, fieldName, oldValue, newValue,
	)
	return err
}

func (r *HistoryRepo) FindByNodeID(ctx context.Context, nodeID string, limit int) ([]model.NodeHistoryEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT nh.id, nh.action, nh.field_name, nh.old_value, nh.new_value, nh.created_at, u.display_name
		 FROM node_history nh
		 INNER JOIN users u ON nh.user_id = u.id
		 WHERE nh.node_id = $1
		 ORDER BY nh.created_at DESC
		 LIMIT $2`, nodeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.NodeHistoryEntry
	for rows.Next() {
		var e model.NodeHistoryEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.FieldName, &e.OldValue, &e.NewValue, &e.CreatedAt, &e.UserName); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
