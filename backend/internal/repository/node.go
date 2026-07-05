package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type NodeRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewNodeRepo(pool *pgxpool.Pool) *NodeRepo {
	return &NodeRepo{pool: pool, q: dbgen.New(pool)}
}

func (r *NodeRepo) Pool() *pgxpool.Pool { return r.pool }

// ============================================================================
// model.Node converters (per sqlc-generated row type).
// All "WithCreator" join rows share an identical column shape; sqlc names them
// per-query, so we have one trivial converter per row type. Keeping the field
// list explicit means adding a field to model.Node + the query SELECT lights
// up a compile error here until each converter is updated — that's the point.
// ============================================================================

func nodeFromCreateRow(r dbgen.Node) *model.Node {
	return &model.Node{
		ID:                    r.ID,
		ProjectID:             r.ProjectID,
		Type:                  model.NodeType(r.Type),
		Title:                 r.Title,
		Description:           r.Description,
		Status:                model.NodeStatus(r.Status),
		AssigneeID:            r.AssigneeID,
		Tags:                  r.Tags,
		Metadata:              r.Metadata,
		ParentID:              r.ParentID,
		PositionX:             r.PositionX,
		PositionY:             r.PositionY,
		Width:                 r.Width,
		Height:                r.Height,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
		DescriptionSource:     r.DescriptionSource,
		DescriptionAuthoredBy: r.DescriptionAuthoredBy,
		DescriptionAuthoredAt: r.DescriptionAuthoredAt,
		DescriptionAgentModel: r.DescriptionAgentModel,
		LastVerifiedAt:        r.LastVerifiedAt,
		LastVerifiedBy:        r.LastVerifiedBy,
		LastVerifiedCommit:    r.LastVerifiedCommit,
		FieldProvenance:         r.FieldProvenance,
		CreatedBy:               r.CreatedBy,
		LifecycleState:          r.LifecycleState,
		LifecycleStateChangedAt: r.LifecycleStateChangedAt,
		// CreatorEmail left empty — Create RETURNING can't JOIN users.
		// Re-fetch via FindByID if the caller needs it immediately.
	}
}

func nodeFromFindByIDRow(r dbgen.NodeFindByIDRow) *model.Node {
	return &model.Node{
		ID: r.ID, ProjectID: r.ProjectID, Type: model.NodeType(r.Type), Title: r.Title,
		Description: r.Description, Status: model.NodeStatus(r.Status), AssigneeID: r.AssigneeID,
		Tags: r.Tags, Metadata: r.Metadata, ParentID: r.ParentID,
		PositionX: r.PositionX, PositionY: r.PositionY, Width: r.Width, Height: r.Height,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		DescriptionSource: r.DescriptionSource, DescriptionAuthoredBy: r.DescriptionAuthoredBy,
		DescriptionAuthoredAt: r.DescriptionAuthoredAt, DescriptionAgentModel: r.DescriptionAgentModel,
		LastVerifiedAt: r.LastVerifiedAt, LastVerifiedBy: r.LastVerifiedBy,
		LastVerifiedCommit: r.LastVerifiedCommit, FieldProvenance: r.FieldProvenance,
		CreatedBy: r.CreatedBy, CreatorEmail: r.CreatorEmail,
		LifecycleState: r.LifecycleState, LifecycleStateChangedAt: r.LifecycleStateChangedAt,
	}
}

func nodeFromFindByIDsRow(r dbgen.NodeFindByIDsRow) *model.Node {
	return &model.Node{
		ID: r.ID, ProjectID: r.ProjectID, Type: model.NodeType(r.Type), Title: r.Title,
		Description: r.Description, Status: model.NodeStatus(r.Status), AssigneeID: r.AssigneeID,
		Tags: r.Tags, Metadata: r.Metadata, ParentID: r.ParentID,
		PositionX: r.PositionX, PositionY: r.PositionY, Width: r.Width, Height: r.Height,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		DescriptionSource: r.DescriptionSource, DescriptionAuthoredBy: r.DescriptionAuthoredBy,
		DescriptionAuthoredAt: r.DescriptionAuthoredAt, DescriptionAgentModel: r.DescriptionAgentModel,
		LastVerifiedAt: r.LastVerifiedAt, LastVerifiedBy: r.LastVerifiedBy,
		LastVerifiedCommit: r.LastVerifiedCommit, FieldProvenance: r.FieldProvenance,
		CreatedBy: r.CreatedBy, CreatorEmail: r.CreatorEmail,
		LifecycleState: r.LifecycleState, LifecycleStateChangedAt: r.LifecycleStateChangedAt,
	}
}

func nodeFromFindByProjectIDSimpleRow(r dbgen.NodeFindByProjectIDSimpleRow) *model.Node {
	return &model.Node{
		ID: r.ID, ProjectID: r.ProjectID, Type: model.NodeType(r.Type), Title: r.Title,
		Description: r.Description, Status: model.NodeStatus(r.Status), AssigneeID: r.AssigneeID,
		Tags: r.Tags, Metadata: r.Metadata, ParentID: r.ParentID,
		PositionX: r.PositionX, PositionY: r.PositionY, Width: r.Width, Height: r.Height,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		DescriptionSource: r.DescriptionSource, DescriptionAuthoredBy: r.DescriptionAuthoredBy,
		DescriptionAuthoredAt: r.DescriptionAuthoredAt, DescriptionAgentModel: r.DescriptionAgentModel,
		LastVerifiedAt: r.LastVerifiedAt, LastVerifiedBy: r.LastVerifiedBy,
		LastVerifiedCommit: r.LastVerifiedCommit, FieldProvenance: r.FieldProvenance,
		CreatedBy: r.CreatedBy, CreatorEmail: r.CreatorEmail,
		LifecycleState: r.LifecycleState, LifecycleStateChangedAt: r.LifecycleStateChangedAt,
	}
}

func nodeFromFindChangedSinceRow(r dbgen.NodeFindChangedSinceRow) *model.Node {
	return &model.Node{
		ID: r.ID, ProjectID: r.ProjectID, Type: model.NodeType(r.Type), Title: r.Title,
		Description: r.Description, Status: model.NodeStatus(r.Status), AssigneeID: r.AssigneeID,
		Tags: r.Tags, Metadata: r.Metadata, ParentID: r.ParentID,
		PositionX: r.PositionX, PositionY: r.PositionY, Width: r.Width, Height: r.Height,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		DescriptionSource: r.DescriptionSource, DescriptionAuthoredBy: r.DescriptionAuthoredBy,
		DescriptionAuthoredAt: r.DescriptionAuthoredAt, DescriptionAgentModel: r.DescriptionAgentModel,
		LastVerifiedAt: r.LastVerifiedAt, LastVerifiedBy: r.LastVerifiedBy,
		LastVerifiedCommit: r.LastVerifiedCommit, FieldProvenance: r.FieldProvenance,
		CreatedBy: r.CreatedBy, CreatorEmail: r.CreatorEmail,
		LifecycleState: r.LifecycleState, LifecycleStateChangedAt: r.LifecycleStateChangedAt,
	}
}

func nodeFromFindFailOrBugRow(r dbgen.NodeFindFailOrBugRow) *model.Node {
	return &model.Node{
		ID: r.ID, ProjectID: r.ProjectID, Type: model.NodeType(r.Type), Title: r.Title,
		Description: r.Description, Status: model.NodeStatus(r.Status), AssigneeID: r.AssigneeID,
		Tags: r.Tags, Metadata: r.Metadata, ParentID: r.ParentID,
		PositionX: r.PositionX, PositionY: r.PositionY, Width: r.Width, Height: r.Height,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		DescriptionSource: r.DescriptionSource, DescriptionAuthoredBy: r.DescriptionAuthoredBy,
		DescriptionAuthoredAt: r.DescriptionAuthoredAt, DescriptionAgentModel: r.DescriptionAgentModel,
		LastVerifiedAt: r.LastVerifiedAt, LastVerifiedBy: r.LastVerifiedBy,
		LastVerifiedCommit: r.LastVerifiedCommit, FieldProvenance: r.FieldProvenance,
		CreatedBy: r.CreatedBy, CreatorEmail: r.CreatorEmail,
		LifecycleState: r.LifecycleState, LifecycleStateChangedAt: r.LifecycleStateChangedAt,
	}
}

// nodeColsWithCreator is the SELECT list for the hand-written dynamic queries
// below (FindByProjectIDPaginated, FindByProjectID-with-filters). The alias
// "n" must be applied to the nodes table.
const nodeColsWithCreator = `n.id, n.project_id, n.type, n.title, n.description, n.status, n.assignee_id, n.tags, n.metadata, n.parent_id, n.position_x, n.position_y, n.width, n.height, n.created_at, n.updated_at, n.description_source, n.description_authored_by, n.description_authored_at, n.description_agent_model, n.last_verified_at, n.last_verified_by, n.last_verified_commit, n.field_provenance, n.created_by, COALESCE(u.email, '') AS creator_email, n.lifecycle_state, n.lifecycle_state_changed_at`

// ============================================================================
// Repo methods — static = sqlc-generated wrappers, dynamic = hand-written pgx
// ============================================================================

func (r *NodeRepo) Create(ctx context.Context, n *model.Node) (*model.Node, error) {
	row, err := r.q.NodeCreate(ctx, dbgen.NodeCreateParams{
		ProjectID:   n.ProjectID,
		Type:        string(n.Type),
		Title:       n.Title,
		Description: n.Description,
		Status:      string(n.Status),
		AssigneeID:  n.AssigneeID,
		Tags:        n.Tags,
		ParentID:    n.ParentID,
		PositionX:   n.PositionX,
		PositionY:   n.PositionY,
		Width:       n.Width,
		Height:      n.Height,
		CreatedBy:   n.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return nodeFromCreateRow(row), nil
}

func (r *NodeRepo) FindByID(ctx context.Context, id, projectID string) (*model.Node, error) {
	row, err := r.q.NodeFindByID(ctx, dbgen.NodeFindByIDParams{ID: id, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	return nodeFromFindByIDRow(row), nil
}

func (r *NodeRepo) FindByProjectID(ctx context.Context, projectID string, nodeType, status *string) ([]model.Node, error) {
	if nodeType == nil && status == nil {
		rows, err := r.q.NodeFindByProjectIDSimple(ctx, projectID)
		if err != nil {
			return nil, err
		}
		out := make([]model.Node, len(rows))
		for i, r := range rows {
			out[i] = *nodeFromFindByProjectIDSimpleRow(r)
		}
		return out, nil
	}
	// Dynamic filter path stays as hand-written pgx.
	query := `SELECT ` + nodeColsWithCreator + `
		 FROM nodes n
		 LEFT JOIN users u ON u.id = n.created_by
		 WHERE n.project_id = $1`
	args := []any{projectID}
	idx := 2
	if nodeType != nil {
		query += fmt.Sprintf(" AND n.type = $%d", idx)
		args = append(args, *nodeType)
		idx++
	}
	if status != nil {
		query += fmt.Sprintf(" AND n.status = $%d", idx)
		args = append(args, *status)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodesRaw(rows)
}

func (r *NodeRepo) Update(ctx context.Context, id string, fields map[string]any) (*model.Node, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}
	// Dynamic setClauses path stays as hand-written pgx — RETURN just the keys
	// we need to re-fetch the full row through the sqlc-generated FindByID.
	setClauses := []string{"updated_at = now()"}
	// lifecycle_state_changed_at auto-stamps on every write that carries
	// lifecycle_state. Handlers don't have to remember to bump the timestamp;
	// the repository owns that invariant so any future path (batch update,
	// MCP tool, import) inherits it for free. Idempotent identity writes will
	// over-bump the timestamp — acceptable for v0.6.0; if that becomes noisy
	// we can swap in `IS DISTINCT FROM` once fields is threaded as an ordered
	// slice with known parameter positions.
	if _, ok := fields["lifecycle_state"]; ok {
		setClauses = append(setClauses, "lifecycle_state_changed_at = now()")
	}
	args := []any{}
	idx := 1
	for col, val := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE nodes SET %s WHERE id = $%d RETURNING project_id`,
		strings.Join(setClauses, ", "), idx,
	)
	var projectID string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&projectID); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id, projectID)
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
	rows, err := r.q.NodeMarkVerified(ctx, dbgen.NodeMarkVerifiedParams{
		LastVerifiedBy:     &userID,
		LastVerifiedCommit: commitPtr,
		ID:                 nodeID,
		ProjectID:          projectID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
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
	return r.q.NodeUpdateDescriptionProvenance(ctx, dbgen.NodeUpdateDescriptionProvenanceParams{
		DescriptionSource:     source,
		DescriptionAuthoredBy: &authoredBy,
		DescriptionAgentModel: agentPtr,
		ID:                    nodeID,
	})
}

func (r *NodeRepo) Delete(ctx context.Context, id, projectID string) error {
	if err := r.q.NodeUnparentChildren(ctx, dbgen.NodeUnparentChildrenParams{
		ParentIds: []string{id}, ProjectID: projectID,
	}); err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx,
		`DELETE FROM edges WHERE project_id = $1 AND (source_id = $2 OR target_id = $2)`,
		projectID, id); err != nil {
		return err
	}
	return r.q.NodeDeleteByID(ctx, dbgen.NodeDeleteByIDParams{ID: id, ProjectID: projectID})
}

func (r *NodeRepo) BatchUpdatePositions(ctx context.Context, projectID string, positions []struct {
	ID     string
	X, Y   float64
	Width  *float64
	Height *float64
}) error {
	// Hot path that benefits from pgx.Batch pipelining — stays raw.
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
	rows, err := r.q.NodeFindChangedSince(ctx, dbgen.NodeFindChangedSinceParams{ProjectID: projectID, UpdatedAt: since})
	if err != nil {
		return nil, err
	}
	out := make([]model.Node, len(rows))
	for i, r := range rows {
		out[i] = *nodeFromFindChangedSinceRow(r)
	}
	return out, nil
}

func (r *NodeRepo) FindFailOrBug(ctx context.Context, projectID string) ([]model.Node, error) {
	rows, err := r.q.NodeFindFailOrBug(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]model.Node, len(rows))
	for i, r := range rows {
		out[i] = *nodeFromFindFailOrBugRow(r)
	}
	return out, nil
}

func (r *NodeRepo) FindByIDs(ctx context.Context, ids []string) ([]model.Node, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.q.NodeFindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]model.Node, len(rows))
	for i, r := range rows {
		out[i] = *nodeFromFindByIDsRow(r)
	}
	return out, nil
}

func (r *NodeRepo) UpdateStatus(ctx context.Context, id string, status model.NodeStatus) error {
	return r.q.NodeUpdateStatus(ctx, dbgen.NodeUpdateStatusParams{Status: string(status), ID: id})
}

func (r *NodeRepo) BatchDelete(ctx context.Context, projectID string, ids []string) error {
	if err := r.q.NodeUnparentChildren(ctx, dbgen.NodeUnparentChildrenParams{
		ParentIds: ids, ProjectID: projectID,
	}); err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx,
		`DELETE FROM edges WHERE project_id = $1 AND (source_id = ANY($2) OR target_id = ANY($2))`,
		projectID, ids); err != nil {
		return err
	}
	return r.q.NodeDeleteByIDs(ctx, dbgen.NodeDeleteByIDsParams{NodeIds: ids, ProjectID: projectID})
}

func (r *NodeRepo) BatchUpdateStatus(ctx context.Context, projectID string, ids []string, status model.NodeStatus) error {
	return r.q.NodeBatchUpdateStatus(ctx, dbgen.NodeBatchUpdateStatusParams{
		Status: string(status), NodeIds: ids, ProjectID: projectID,
	})
}

// FindByProjectIDPaginated returns nodes with cursor-based pagination.
// Uses (created_at, id) keyset for stable ordering.
// Returns up to limit+1 rows; if len(result) > limit, hasMore=true and trim the last row.
// Stays as hand-written pgx because the WHERE/ORDER clauses are dynamic.
func (r *NodeRepo) FindByProjectIDPaginated(ctx context.Context, projectID string, nodeType, status *string, limit int, afterTime *time.Time, afterID *string) ([]model.Node, bool, error) {
	args := []any{projectID}
	where := "WHERE n.project_id = $1"
	argIdx := 2

	if nodeType != nil {
		where += fmt.Sprintf(" AND n.type = $%d", argIdx)
		args = append(args, *nodeType)
		argIdx++
	}
	if status != nil {
		where += fmt.Sprintf(" AND n.status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}
	if afterTime != nil && afterID != nil {
		where += fmt.Sprintf(" AND (n.created_at, n.id) > ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, *afterTime, *afterID)
		argIdx += 2
	}

	query := fmt.Sprintf(
		`SELECT `+nodeColsWithCreator+` FROM nodes n LEFT JOIN users u ON u.id = n.created_by %s ORDER BY n.created_at ASC, n.id ASC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	nodes, err := scanNodesRaw(rows)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(nodes) > limit
	if hasMore {
		nodes = nodes[:limit]
	}
	return nodes, hasMore, nil
}

// scanArgCount is the number of pointer args scanNodesRaw passes to Scan.
// Kept in sync with the actual call below AND with nodeColsWithCreator's
// column count — the invariant is enforced by
// TestNodeReadPaths_IncludeAllPersistedFields.
const scanArgCount = 28

// scanNodesRaw is the manual scanner for dynamic-WHERE queries that can't go
// through sqlc. Field order MUST match nodeColsWithCreator.
//
// INVARIANT: this list, nodeColsWithCreator, and every nodeFrom*Row converter
// must all mention every column persisted on the `nodes` table. See
// ~/.claude memory `feedback_sql_scan_safety` for prior incidents. Adding a
// column here without updating the const or the converters is caught by
// TestNodeReadPaths_IncludeAllPersistedFields in the same package.
func scanNodesRaw(rows interface {
	Next() bool
	Scan(dest ...any) error
}) ([]model.Node, error) {
	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(
			&n.ID, &n.ProjectID, &n.Type, &n.Title, &n.Description, &n.Status,
			&n.AssigneeID, &n.Tags, &n.Metadata, &n.ParentID,
			&n.PositionX, &n.PositionY, &n.Width, &n.Height,
			&n.CreatedAt, &n.UpdatedAt,
			&n.DescriptionSource, &n.DescriptionAuthoredBy, &n.DescriptionAuthoredAt, &n.DescriptionAgentModel,
			&n.LastVerifiedAt, &n.LastVerifiedBy, &n.LastVerifiedCommit, &n.FieldProvenance,
			&n.CreatedBy, &n.CreatorEmail,
			&n.LifecycleState, &n.LifecycleStateChangedAt,
		); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}
