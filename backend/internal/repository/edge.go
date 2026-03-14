package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type EdgeRepo struct {
	pool *pgxpool.Pool
}

func NewEdgeRepo(pool *pgxpool.Pool) *EdgeRepo {
	return &EdgeRepo{pool: pool}
}

func (r *EdgeRepo) Create(ctx context.Context, projectID, sourceID, targetID string, edgeType model.EdgeType, label *string) (*model.Edge, error) {
	var e model.Edge
	err := r.pool.QueryRow(ctx,
		`INSERT INTO edges (project_id, source_id, target_id, edge_type, label)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, project_id, source_id, target_id, edge_type, label, created_at`,
		projectID, sourceID, targetID, edgeType, label,
	).Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.TargetID, &e.EdgeType, &e.Label, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EdgeRepo) FindByProjectID(ctx context.Context, projectID string) ([]model.Edge, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, source_id, target_id, edge_type, label, created_at
		 FROM edges WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []model.Edge
	for rows.Next() {
		var e model.Edge
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.TargetID, &e.EdgeType, &e.Label, &e.CreatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}

func (r *EdgeRepo) FindConnected(ctx context.Context, nodeID string) ([]model.Edge, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, source_id, target_id, edge_type, label, created_at
		 FROM edges WHERE source_id = $1 OR target_id = $1`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []model.Edge
	for rows.Next() {
		var e model.Edge
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.TargetID, &e.EdgeType, &e.Label, &e.CreatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}

func (r *EdgeRepo) Update(ctx context.Context, id string, edgeType *model.EdgeType, label *string) (*model.Edge, error) {
	var e model.Edge
	err := r.pool.QueryRow(ctx,
		`UPDATE edges SET
		   edge_type = COALESCE($1, edge_type),
		   label = COALESCE($2, label)
		 WHERE id = $3
		 RETURNING id, project_id, source_id, target_id, edge_type, label, created_at`,
		edgeType, label, id,
	).Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.TargetID, &e.EdgeType, &e.Label, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EdgeRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM edges WHERE id = $1`, id)
	return err
}
