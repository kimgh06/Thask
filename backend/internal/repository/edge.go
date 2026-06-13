package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type EdgeRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewEdgeRepo(pool *pgxpool.Pool) *EdgeRepo {
	return &EdgeRepo{pool: pool, q: dbgen.New(pool)}
}

// Pool exposes the underlying pool so handlers can run batched work inside
// their own transaction without bouncing every statement through the repo.
func (r *EdgeRepo) Pool() *pgxpool.Pool { return r.pool }

// ============================================================================
// model.Edge converters (per sqlc-generated row type).
// waypoints is []byte (JSONB) in dbgen rows; model.Edge uses `any`.
// ============================================================================

func edgeFromCreateRow(row dbgen.EdgeCreateRow) *model.Edge {
	return &model.Edge{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		SourceID:   row.SourceID,
		TargetID:   row.TargetID,
		EdgeType:   model.EdgeType(row.EdgeType),
		Label:      row.Label,
		SourcePort: row.SourcePort,
		TargetPort: row.TargetPort,
		Waypoints:  json.RawMessage(row.Waypoints),
		CreatedAt:  row.CreatedAt,
	}
}

func edgeFromCreateWithRoutingRow(row dbgen.EdgeCreateWithRoutingRow) *model.Edge {
	return &model.Edge{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		SourceID:   row.SourceID,
		TargetID:   row.TargetID,
		EdgeType:   model.EdgeType(row.EdgeType),
		Label:      row.Label,
		SourcePort: row.SourcePort,
		TargetPort: row.TargetPort,
		Waypoints:  json.RawMessage(row.Waypoints),
		CreatedAt:  row.CreatedAt,
	}
}

func edgeFromFindByProjectIDRow(row dbgen.EdgeFindByProjectIDRow) *model.Edge {
	return &model.Edge{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		SourceID:   row.SourceID,
		TargetID:   row.TargetID,
		EdgeType:   model.EdgeType(row.EdgeType),
		Label:      row.Label,
		SourcePort: row.SourcePort,
		TargetPort: row.TargetPort,
		Waypoints:  json.RawMessage(row.Waypoints),
		CreatedAt:  row.CreatedAt,
	}
}

func edgeFromFindConnectedRow(row dbgen.EdgeFindConnectedRow) *model.Edge {
	return &model.Edge{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		SourceID:   row.SourceID,
		TargetID:   row.TargetID,
		EdgeType:   model.EdgeType(row.EdgeType),
		Label:      row.Label,
		SourcePort: row.SourcePort,
		TargetPort: row.TargetPort,
		Waypoints:  json.RawMessage(row.Waypoints),
		CreatedAt:  row.CreatedAt,
	}
}

// edgeCols is kept for the dynamic FindByProjectIDPaginated raw query.
const edgeCols = `id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, created_at`

// scanEdge is kept for the dynamic raw-pgx query paths.
func scanEdge(row interface{ Scan(...any) error }) (*model.Edge, error) {
	var e model.Edge
	err := row.Scan(&e.ID, &e.ProjectID, &e.SourceID, &e.TargetID, &e.EdgeType, &e.Label, &e.SourcePort, &e.TargetPort, &e.Waypoints, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ============================================================================
// Repo methods — static = sqlc-generated wrappers, dynamic = hand-written pgx
// ============================================================================

func (r *EdgeRepo) Create(ctx context.Context, projectID, sourceID, targetID string, edgeType model.EdgeType, label *string) (*model.Edge, error) {
	row, err := r.q.EdgeCreate(ctx, dbgen.EdgeCreateParams{
		ProjectID: projectID,
		SourceID:  sourceID,
		TargetID:  targetID,
		EdgeType:  string(edgeType),
		Label:     label,
	})
	if err != nil {
		return nil, err
	}
	return edgeFromCreateRow(row), nil
}

func (r *EdgeRepo) CreateWithRouting(ctx context.Context, projectID, sourceID, targetID string, edgeType model.EdgeType, label *string, sourcePort, targetPort string, waypoints any) (*model.Edge, error) {
	if sourcePort == "" {
		sourcePort = "auto"
	}
	if targetPort == "" {
		targetPort = "auto"
	}
	var wpBytes []byte
	if waypoints == nil {
		wpBytes = []byte("[]")
	} else {
		b, err := json.Marshal(waypoints)
		if err != nil {
			return nil, err
		}
		wpBytes = b
	}
	row, err := r.q.EdgeCreateWithRouting(ctx, dbgen.EdgeCreateWithRoutingParams{
		ProjectID:  projectID,
		SourceID:   sourceID,
		TargetID:   targetID,
		EdgeType:   string(edgeType),
		Label:      label,
		SourcePort: sourcePort,
		TargetPort: targetPort,
		Waypoints:  wpBytes,
	})
	if err != nil {
		return nil, err
	}
	return edgeFromCreateWithRoutingRow(row), nil
}

func (r *EdgeRepo) FindByProjectID(ctx context.Context, projectID string) ([]model.Edge, error) {
	rows, err := r.q.EdgeFindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]model.Edge, len(rows))
	for i, row := range rows {
		out[i] = *edgeFromFindByProjectIDRow(row)
	}
	return out, nil
}

func (r *EdgeRepo) FindConnected(ctx context.Context, nodeID string) ([]model.Edge, error) {
	rows, err := r.q.EdgeFindConnected(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	out := make([]model.Edge, len(rows))
	for i, row := range rows {
		out[i] = *edgeFromFindConnectedRow(row)
	}
	return out, nil
}

// Update keeps COALESCE nullable-param semantics as raw pgx — the sqlc-generated
// EdgeUpdateParams.EdgeType is non-pointer string, which can't express "no change".
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

// UpdateRouting keeps COALESCE nullable-param semantics as raw pgx — the sqlc-generated
// EdgeUpdateRoutingParams uses non-pointer string for ports and []byte for waypoints,
// which can't express "no change" for nullable inputs.
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
	return r.q.EdgeResetWaypoints(ctx, projectID)
}

func (r *EdgeRepo) VerifyProject(ctx context.Context, edgeID, projectID string) error {
	exists, err := r.q.EdgeVerifyProject(ctx, dbgen.EdgeVerifyProjectParams{ID: edgeID, ProjectID: projectID})
	if err != nil || !exists {
		return fmt.Errorf("edge not found in project")
	}
	return nil
}

// VerifyNodesInProject confirms that every nodeID belongs to projectID. Returns
// nil iff all supplied IDs are present in the project's nodes table. Used by
// edge create paths to reject cross-project source/target — without this
// check, a caller can store an edge whose project_id doesn't match its
// endpoints' project_ids, corrupting the graph.
func (r *EdgeRepo) VerifyNodesInProject(ctx context.Context, projectID string, nodeIDs []string) error {
	if len(nodeIDs) == 0 {
		return nil
	}
	got, err := r.q.EdgeVerifyNodesInProject(ctx, dbgen.EdgeVerifyNodesInProjectParams{
		ProjectID: projectID,
		NodeIds:   nodeIDs,
	})
	if err != nil {
		return err
	}
	// COUNT ignores duplicates; dedupe before comparing so passing the same
	// id twice (e.g. self-edge check upstream) doesn't false-pass here.
	seen := make(map[string]struct{}, len(nodeIDs))
	for _, id := range nodeIDs {
		seen[id] = struct{}{}
	}
	if int(got) != len(seen) {
		return fmt.Errorf("one or more node IDs do not belong to this project")
	}
	return nil
}

// FilterNodesInProject returns the subset of nodeIDs that actually belong to
// projectID. Used by edge BatchCreate to per-item skip cross-project endpoints
// while still applying every legitimate edge in the same transaction.
func (r *EdgeRepo) FilterNodesInProject(ctx context.Context, projectID string, nodeIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{}, len(nodeIDs))
	if len(nodeIDs) == 0 {
		return out, nil
	}
	rows, err := r.q.EdgeListNodeIDsInProject(ctx, dbgen.EdgeListNodeIDsInProjectParams{
		ProjectID: projectID,
		NodeIds:   nodeIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, id := range rows {
		out[id] = struct{}{}
	}
	return out, nil
}

func (r *EdgeRepo) Delete(ctx context.Context, id string) error {
	return r.q.EdgeDelete(ctx, id)
}

func (r *EdgeRepo) DeleteScoped(ctx context.Context, id, projectID string) error {
	rows, err := r.q.EdgeDeleteScoped(ctx, dbgen.EdgeDeleteScopedParams{ID: id, ProjectID: projectID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("edge not found")
	}
	return nil
}

// WaypointUpdate holds an edge ID and its JSON-encoded waypoints.
type WaypointUpdate struct {
	ID        string
	Waypoints []byte
}

// BatchUpdateWaypoints saves computed waypoints for multiple edges.
// Kept as raw pgx loop — no pgx.Batch but iterative pool.Exec per update.
func (r *EdgeRepo) BatchUpdateWaypoints(ctx context.Context, updates []WaypointUpdate) error {
	for _, u := range updates {
		if _, err := r.pool.Exec(ctx,
			`UPDATE edges SET waypoints = $1 WHERE id = $2`,
			u.Waypoints, u.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

// FindByProjectIDPaginated returns edges with cursor-based pagination.
// Kept as hand-written pgx — dynamic WHERE (cursor keyset).
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
