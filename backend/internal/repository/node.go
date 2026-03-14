package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type NodeRepo struct {
	pool *pgxpool.Pool
}

func NewNodeRepo(pool *pgxpool.Pool) *NodeRepo {
	return &NodeRepo{pool: pool}
}

func (r *NodeRepo) Create(ctx context.Context, n *model.Node) (*model.Node, error) {
	var node model.Node
	err := r.pool.QueryRow(ctx,
		`INSERT INTO nodes (project_id, type, title, description, status, assignee_id, tags, parent_id, position_x, position_y, width, height)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at`,
		n.ProjectID, n.Type, n.Title, n.Description, n.Status, n.AssigneeID, n.Tags, n.ParentID, n.PositionX, n.PositionY, n.Width, n.Height,
	).Scan(&node.ID, &node.ProjectID, &node.Type, &node.Title, &node.Description, &node.Status, &node.AssigneeID, &node.Tags, &node.Metadata, &node.ParentID, &node.PositionX, &node.PositionY, &node.Width, &node.Height, &node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *NodeRepo) FindByID(ctx context.Context, id, projectID string) (*model.Node, error) {
	var n model.Node
	err := r.pool.QueryRow(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at
		 FROM nodes WHERE id = $1 AND project_id = $2`,
		id, projectID,
	).Scan(&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NodeRepo) FindByProjectID(ctx context.Context, projectID string, nodeType, status *string) ([]model.Node, error) {
	query := `SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at
		 FROM nodes WHERE project_id = $1`
	args := []any{projectID}
	idx := 2

	if nodeType != nil {
		query += fmt.Sprintf(" AND type = $%d", idx)
		args = append(args, *nodeType)
		idx++
	}
	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, *status)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *NodeRepo) Update(ctx context.Context, id string, fields map[string]any) (*model.Node, error) {
	if len(fields) == 0 {
		return r.FindByID(ctx, id, "")
	}
	setClauses := []string{"updated_at = now()"}
	args := []any{}
	idx := 1
	for col, val := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE nodes SET %s WHERE id = $%d
		 RETURNING id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at`,
		strings.Join(setClauses, ", "), idx,
	)
	var n model.Node
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NodeRepo) Delete(ctx context.Context, id, projectID string) error {
	// Unparent children
	_, _ = r.pool.Exec(ctx,
		`UPDATE nodes SET parent_id = NULL, updated_at = now() WHERE parent_id = $1 AND project_id = $2`,
		id, projectID)
	// Delete connected edges
	_, _ = r.pool.Exec(ctx,
		`DELETE FROM edges WHERE project_id = $1 AND (source_id = $2 OR target_id = $2)`,
		projectID, id)
	// Delete node
	_, err := r.pool.Exec(ctx, `DELETE FROM nodes WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}

func (r *NodeRepo) BatchUpdatePositions(ctx context.Context, projectID string, positions []struct {
	ID     string
	X, Y   float64
	Width  *float64
	Height *float64
}) error {
	batch := &strings.Builder{}
	batch.WriteString("BEGIN;")
	for _, p := range positions {
		if p.Width != nil && p.Height != nil {
			fmt.Fprintf(batch,
				"UPDATE nodes SET position_x=%f, position_y=%f, width=%f, height=%f, updated_at=now() WHERE id='%s' AND project_id='%s';",
				p.X, p.Y, *p.Width, *p.Height, p.ID, projectID)
		} else {
			fmt.Fprintf(batch,
				"UPDATE nodes SET position_x=%f, position_y=%f, updated_at=now() WHERE id='%s' AND project_id='%s';",
				p.X, p.Y, p.ID, projectID)
		}
	}
	batch.WriteString("COMMIT;")
	_, err := r.pool.Exec(ctx, batch.String())
	return err
}

func (r *NodeRepo) FindChangedSince(ctx context.Context, projectID string, since time.Time) ([]model.Node, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at
		 FROM nodes WHERE project_id = $1 AND updated_at >= $2`, projectID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *NodeRepo) FindFailOrBug(ctx context.Context, projectID string) ([]model.Node, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at
		 FROM nodes WHERE project_id = $1 AND (status = 'FAIL' OR type = 'BUG')`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *NodeRepo) FindByIDs(ctx context.Context, ids []string) ([]model.Node, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at
		 FROM nodes WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *NodeRepo) UpdateStatus(ctx context.Context, id string, status model.NodeStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE nodes SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}

// scanNodes is a helper to scan rows into []model.Node
func scanNodes(rows interface{ Next() bool; Scan(dest ...any) error }) ([]model.Node, error) {
	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}
