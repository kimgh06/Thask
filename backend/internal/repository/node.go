package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type NodeRepo struct {
	pool *pgxpool.Pool
}

func NewNodeRepo(pool *pgxpool.Pool) *NodeRepo {
	return &NodeRepo{pool: pool}
}

func (r *NodeRepo) Pool() *pgxpool.Pool { return r.pool }


func (r *NodeRepo) Create(ctx context.Context, n *model.Node) (*model.Node, error) {
	var node model.Node
	err := r.pool.QueryRow(ctx,
		`INSERT INTO nodes (project_id, type, title, description, status, assignee_id, tags, parent_id, position_x, position_y, width, height)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance`,
		n.ProjectID, n.Type, n.Title, n.Description, n.Status, n.AssigneeID, n.Tags, n.ParentID, n.PositionX, n.PositionY, n.Width, n.Height,
	).Scan(&node.ID, &node.ProjectID, &node.Type, &node.Title, &node.Description, &node.Status, &node.AssigneeID, &node.Tags, &node.Metadata, &node.ParentID, &node.PositionX, &node.PositionY, &node.Width, &node.Height, &node.CreatedAt, &node.UpdatedAt, &node.DescriptionSource, &node.DescriptionAuthoredBy, &node.DescriptionAuthoredAt, &node.DescriptionAgentModel, &node.LastVerifiedAt, &node.LastVerifiedBy, &node.LastVerifiedCommit, &node.FieldProvenance)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *NodeRepo) FindByID(ctx context.Context, id, projectID string) (*model.Node, error) {
	var n model.Node
	err := r.pool.QueryRow(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance
		 FROM nodes WHERE id = $1 AND project_id = $2`,
		id, projectID,
	).Scan(&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt, &n.DescriptionSource, &n.DescriptionAuthoredBy, &n.DescriptionAuthoredAt, &n.DescriptionAgentModel, &n.LastVerifiedAt, &n.LastVerifiedBy, &n.LastVerifiedCommit, &n.FieldProvenance)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NodeRepo) FindByProjectID(ctx context.Context, projectID string, nodeType, status *string) ([]model.Node, error) {
	query := `SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance
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
		return nil, fmt.Errorf("no fields to update")
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
		 RETURNING id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance`,
		strings.Join(setClauses, ", "), idx,
	)
	var n model.Node
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt, &n.DescriptionSource, &n.DescriptionAuthoredBy, &n.DescriptionAuthoredAt, &n.DescriptionAgentModel, &n.LastVerifiedAt, &n.LastVerifiedBy, &n.LastVerifiedCommit, &n.FieldProvenance,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// MarkVerified stamps last_verified_at / last_verified_by / last_verified_commit
// on a node so reviewers know when a human last said "still correct". Commit
// may be empty (callers running outside a git repo). Returns an error when the
// node does not exist in the given project so the handler can surface 404.
func (r *NodeRepo) MarkVerified(ctx context.Context, nodeID, projectID, userID, commit string) error {
	var commitPtr *string
	if commit != "" {
		commitPtr = &commit
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE nodes
		 SET last_verified_at = now(),
		     last_verified_by = $1,
		     last_verified_commit = $2,
		     updated_at = now()
		 WHERE id = $3 AND project_id = $4`,
		userID, commitPtr, nodeID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("node not found")
	}
	return nil
}

// UpdateDescriptionProvenance refreshes the description_source / authored_by /
// authored_at / agent_model snapshot columns on a node. Called after a
// description write so future readers can answer "who wrote this and when".
// agentModel is the empty string for human writes.
func (r *NodeRepo) UpdateDescriptionProvenance(ctx context.Context, nodeID, authoredBy, source, agentModel string) error {
	var agentPtr *string
	if agentModel != "" {
		agentPtr = &agentModel
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE nodes
		 SET description_source = $1,
		     description_authored_by = $2,
		     description_authored_at = now(),
		     description_agent_model = $3
		 WHERE id = $4`,
		source, authoredBy, agentPtr, nodeID)
	return err
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
	b := &pgx.Batch{}
	for _, p := range positions {
		if p.Width != nil && p.Height != nil {
			b.Queue("UPDATE nodes SET position_x=$1, position_y=$2, width=$3, height=$4, updated_at=now() WHERE id=$5 AND project_id=$6",
				p.X, p.Y, *p.Width, *p.Height, p.ID, projectID)
		} else {
			b.Queue("UPDATE nodes SET position_x=$1, position_y=$2, updated_at=now() WHERE id=$3 AND project_id=$4",
				p.X, p.Y, p.ID, projectID)
		}
	}
	br := r.pool.SendBatch(ctx, b)
	defer br.Close()
	for range positions {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *NodeRepo) FindChangedSince(ctx context.Context, projectID string, since time.Time) ([]model.Node, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance
		 FROM nodes WHERE project_id = $1 AND updated_at >= $2`, projectID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *NodeRepo) FindFailOrBug(ctx context.Context, projectID string) ([]model.Node, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance
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
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance
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

func (r *NodeRepo) BatchDelete(ctx context.Context, projectID string, ids []string) error {
	// Unparent children
	_, _ = r.pool.Exec(ctx,
		`UPDATE nodes SET parent_id = NULL, updated_at = now() WHERE parent_id = ANY($1) AND project_id = $2`,
		ids, projectID)
	// Delete connected edges
	_, _ = r.pool.Exec(ctx,
		`DELETE FROM edges WHERE project_id = $1 AND (source_id = ANY($2) OR target_id = ANY($2))`,
		projectID, ids)
	// Delete nodes
	_, err := r.pool.Exec(ctx,
		`DELETE FROM nodes WHERE id = ANY($1) AND project_id = $2`,
		ids, projectID)
	return err
}

func (r *NodeRepo) BatchUpdateStatus(ctx context.Context, projectID string, ids []string, status model.NodeStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE nodes SET status = $1, updated_at = now() WHERE id = ANY($2) AND project_id = $3`,
		status, ids, projectID)
	return err
}

// FindByProjectIDPaginated returns nodes with cursor-based pagination.
// Uses (created_at, id) keyset for stable ordering.
// Returns up to limit+1 rows; if len(result) > limit, hasMore=true and trim the last row.
func (r *NodeRepo) FindByProjectIDPaginated(ctx context.Context, projectID string, nodeType, status *string, limit int, afterTime *time.Time, afterID *string) ([]model.Node, bool, error) {
	args := []any{projectID}
	where := "WHERE project_id = $1"
	argIdx := 2

	if nodeType != nil {
		where += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, *nodeType)
		argIdx++
	}
	if status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}
	if afterTime != nil && afterID != nil {
		where += fmt.Sprintf(" AND (created_at, id) > ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, *afterTime, *afterID)
		argIdx += 2
	}

	query := fmt.Sprintf(
		`SELECT id, project_id, type, title, description, status, assignee_id, tags, metadata, parent_id, position_x, position_y, width, height, created_at, updated_at, description_source, description_authored_by, description_authored_at, description_agent_model, last_verified_at, last_verified_by, last_verified_commit, field_provenance FROM nodes %s ORDER BY created_at ASC, id ASC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	nodes, err := scanNodes(rows)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(nodes) > limit
	if hasMore {
		nodes = nodes[:limit]
	}
	return nodes, hasMore, nil
}

// scanNodes is a helper to scan rows into []model.Node
func scanNodes(rows interface{ Next() bool; Scan(dest ...any) error }) ([]model.Node, error) {
	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status, &n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID, &n.PositionX, &n.PositionY, &n.Width, &n.Height, &n.CreatedAt, &n.UpdatedAt, &n.DescriptionSource, &n.DescriptionAuthoredBy, &n.DescriptionAuthoredAt, &n.DescriptionAgentModel, &n.LastVerifiedAt, &n.LastVerifiedBy, &n.LastVerifiedCommit, &n.FieldProvenance); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}
