package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

const edgeCols = `id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, created_at`

type EdgeRepo struct {
	pool *pgxpool.Pool
}

func NewEdgeRepo(pool *pgxpool.Pool) *EdgeRepo {
	return &EdgeRepo{pool: pool}
}

func scanEdge(row interface{ Scan(...any) error }) (*model.Edge, error) {
	var e model.Edge
	err := row.Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.TargetID, &e.EdgeType, &e.Label, &e.SourcePort, &e.TargetPort, &e.Waypoints, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EdgeRepo) Create(ctx context.Context, projectID, sourceID, targetID string, edgeType model.EdgeType, label *string) (*model.Edge, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO edges (project_id, source_id, target_id, edge_type, label)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+edgeCols,
		projectID, sourceID, targetID, edgeType, label,
	)
	return scanEdge(row)
}

func (r *EdgeRepo) CreateWithRouting(ctx context.Context, projectID, sourceID, targetID string, edgeType model.EdgeType, label *string, sourcePort, targetPort string, waypoints any) (*model.Edge, error) {
	if sourcePort == "" {
		sourcePort = "auto"
	}
	if targetPort == "" {
		targetPort = "auto"
	}
	if waypoints == nil {
		waypoints = []any{}
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO edges (project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+edgeCols,
		projectID, sourceID, targetID, edgeType, label, sourcePort, targetPort, waypoints,
	)
	return scanEdge(row)
}

func (r *EdgeRepo) FindByProjectID(ctx context.Context, projectID string) ([]model.Edge, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+edgeCols+` FROM edges WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []model.Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, *e)
	}
	return edges, nil
}

func (r *EdgeRepo) FindConnected(ctx context.Context, nodeID string) ([]model.Edge, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+edgeCols+` FROM edges WHERE source_id = $1 OR target_id = $1`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []model.Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, *e)
	}
	return edges, nil
}

func (r *EdgeRepo) Update(ctx context.Context, id string, edgeType *model.EdgeType, label *string) (*model.Edge, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE edges SET
		   edge_type = COALESCE($1, edge_type),
		   label = COALESCE($2, label)
		 WHERE id = $3
		 RETURNING `+edgeCols,
		edgeType, label, id,
	)
	return scanEdge(row)
}

func (r *EdgeRepo) UpdateRouting(ctx context.Context, id string, sourcePort, targetPort *string, waypoints any) (*model.Edge, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE edges SET
		   source_port = COALESCE($1, source_port),
		   target_port = COALESCE($2, target_port),
		   waypoints = COALESCE($3, waypoints)
		 WHERE id = $4
		 RETURNING `+edgeCols,
		sourcePort, targetPort, waypoints, id,
	)
	return scanEdge(row)
}

func (r *EdgeRepo) ResetWaypoints(ctx context.Context, projectID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE edges SET waypoints = '[]', source_port = 'auto', target_port = 'auto' WHERE project_id = $1`,
		projectID,
	)
	return err
}

func (r *EdgeRepo) VerifyProject(ctx context.Context, edgeID, projectID string) error {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM edges WHERE id = $1 AND project_id = $2)`,
		edgeID, projectID,
	).Scan(&exists)
	if err != nil || !exists {
		return fmt.Errorf("edge not found in project")
	}
	return nil
}

func (r *EdgeRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM edges WHERE id = $1`, id)
	return err
}

func (r *EdgeRepo) DeleteScoped(ctx context.Context, id, projectID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM edges WHERE id = $1 AND project_id = $2`, id, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("edge not found")
	}
	return nil
}

// FindByProjectIDPaginated returns edges with cursor-based pagination.
func (r *EdgeRepo) FindByProjectIDPaginated(ctx context.Context, projectID string, limit int, afterTime *time.Time, afterID *string) ([]model.Edge, bool, error) {
	args := []any{projectID}
	where := "WHERE project_id = $1"
	argIdx := 2

	if afterTime != nil && afterID != nil {
		where += fmt.Sprintf(" AND (created_at, id) > ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, *afterTime, *afterID)
		argIdx += 2
	}

	query := fmt.Sprintf(
		`SELECT %s FROM edges %s ORDER BY created_at ASC, id ASC LIMIT $%d`,
		edgeCols, where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var edges []model.Edge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, false, err
		}
		edges = append(edges, *e)
	}

	hasMore := len(edges) > limit
	if hasMore {
		edges = edges[:limit]
	}
	return edges, hasMore, nil
}
