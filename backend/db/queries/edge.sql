-- ============================================================================
-- Edge static queries (sqlc-generated)
--
-- Dynamic queries (FindByProjectIDPaginated with cursor, BatchUpdateWaypoints
-- loop) stay as hand-written pgx in internal/repository/edge.go.
-- ============================================================================

-- name: EdgeCreate :one
INSERT INTO edges (project_id, source_id, target_id, edge_type, label)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, metadata, created_at;

-- name: EdgeCreateWithRouting :one
INSERT INTO edges (project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, metadata, created_at;

-- name: EdgeFindByProjectID :many
SELECT id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, metadata, created_at
FROM edges
WHERE project_id = $1;

-- name: EdgeFindConnected :many
SELECT id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, metadata, created_at
FROM edges
WHERE source_id = $1 OR target_id = $1;

-- name: EdgeUpdate :one
UPDATE edges SET
  edge_type = COALESCE($1, edge_type),
  label = COALESCE($2, label)
WHERE id = $3
RETURNING id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, metadata, created_at;

-- name: EdgeUpdateRouting :one
UPDATE edges SET
  source_port = COALESCE($1, source_port),
  target_port = COALESCE($2, target_port),
  waypoints = COALESCE($3, waypoints)
WHERE id = $4
RETURNING id, project_id, source_id, target_id, edge_type, label, source_port, target_port, waypoints, metadata, created_at;

-- name: EdgeResetWaypoints :exec
UPDATE edges SET waypoints = '[]', source_port = 'auto', target_port = 'auto'
WHERE project_id = $1;

-- name: EdgeVerifyProject :one
SELECT EXISTS(SELECT 1 FROM edges WHERE id = $1 AND project_id = $2);

-- name: EdgeVerifyNodesInProject :one
-- Returns the count of the given node IDs that actually live in the specified
-- project. Used by edge create paths to reject cross-project source/target —
-- without this check, an authenticated caller can pass IDs from another
-- project they have access to and end up with corrupt edge.project_id rows.
SELECT COUNT(*) FROM nodes
WHERE project_id = sqlc.arg(project_id)
  AND id = ANY(sqlc.arg(node_ids)::uuid[]);

-- name: EdgeListNodeIDsInProject :many
-- Returns the subset of supplied IDs that actually live in the project.
-- Used by edge BatchCreate to filter cross-project endpoints into the
-- 207-style skipped[] response instead of corrupting the graph.
SELECT id FROM nodes
WHERE project_id = sqlc.arg(project_id)
  AND id = ANY(sqlc.arg(node_ids)::uuid[]);

-- name: EdgeDelete :exec
DELETE FROM edges WHERE id = $1;

-- name: EdgeDeleteScoped :execrows
DELETE FROM edges WHERE id = $1 AND project_id = $2;
