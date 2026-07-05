-- ============================================================================
-- Node static queries (sqlc-generated)
--
-- Dynamic queries (FindByProjectID with optional filters, Update with dynamic
-- setClauses, FindByProjectIDPaginated, BatchUpdatePositions multi-row) stay
-- as hand-written pgx in internal/repository/node.go.
-- ============================================================================

-- name: NodeCreate :one
INSERT INTO nodes (
  project_id, type, title, description, status, assignee_id, tags,
  parent_id, position_x, position_y, width, height, created_by
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING id, project_id, type, title, description, status, assignee_id, tags,
  metadata, parent_id, position_x, position_y, width, height,
  created_at, updated_at,
  description_source, description_authored_by, description_authored_at,
  description_agent_model,
  last_verified_at, last_verified_by, last_verified_commit, field_provenance,
  created_by, lifecycle_state, lifecycle_state_changed_at;

-- name: NodeFindByID :one
SELECT n.id, n.project_id, n.type, n.title, n.description, n.status,
       n.assignee_id, n.tags, n.metadata, n.parent_id,
       n.position_x, n.position_y, n.width, n.height,
       n.created_at, n.updated_at,
       n.description_source, n.description_authored_by,
       n.description_authored_at, n.description_agent_model,
       n.last_verified_at, n.last_verified_by, n.last_verified_commit,
       n.field_provenance, n.created_by,
       n.lifecycle_state, n.lifecycle_state_changed_at,
       COALESCE(u.email, '')::text AS creator_email
FROM nodes n
LEFT JOIN users u ON u.id = n.created_by
WHERE n.id = $1 AND n.project_id = $2;

-- name: NodeFindChangedSince :many
SELECT n.id, n.project_id, n.type, n.title, n.description, n.status,
       n.assignee_id, n.tags, n.metadata, n.parent_id,
       n.position_x, n.position_y, n.width, n.height,
       n.created_at, n.updated_at,
       n.description_source, n.description_authored_by,
       n.description_authored_at, n.description_agent_model,
       n.last_verified_at, n.last_verified_by, n.last_verified_commit,
       n.field_provenance, n.created_by,
       n.lifecycle_state, n.lifecycle_state_changed_at,
       COALESCE(u.email, '')::text AS creator_email
FROM nodes n
LEFT JOIN users u ON u.id = n.created_by
WHERE n.project_id = $1 AND n.updated_at >= $2;

-- name: NodeFindFailOrBug :many
SELECT n.id, n.project_id, n.type, n.title, n.description, n.status,
       n.assignee_id, n.tags, n.metadata, n.parent_id,
       n.position_x, n.position_y, n.width, n.height,
       n.created_at, n.updated_at,
       n.description_source, n.description_authored_by,
       n.description_authored_at, n.description_agent_model,
       n.last_verified_at, n.last_verified_by, n.last_verified_commit,
       n.field_provenance, n.created_by,
       n.lifecycle_state, n.lifecycle_state_changed_at,
       COALESCE(u.email, '')::text AS creator_email
FROM nodes n
LEFT JOIN users u ON u.id = n.created_by
WHERE n.project_id = $1 AND (n.status = 'FAIL' OR n.type = 'BUG');

-- name: NodeFindByIDs :many
SELECT n.id, n.project_id, n.type, n.title, n.description, n.status,
       n.assignee_id, n.tags, n.metadata, n.parent_id,
       n.position_x, n.position_y, n.width, n.height,
       n.created_at, n.updated_at,
       n.description_source, n.description_authored_by,
       n.description_authored_at, n.description_agent_model,
       n.last_verified_at, n.last_verified_by, n.last_verified_commit,
       n.field_provenance, n.created_by,
       n.lifecycle_state, n.lifecycle_state_changed_at,
       COALESCE(u.email, '')::text AS creator_email
FROM nodes n
LEFT JOIN users u ON u.id = n.created_by
WHERE n.id = ANY(sqlc.arg(node_ids)::uuid[]);

-- name: NodeFindByProjectIDSimple :many
-- Used when no type/status filter is requested.
SELECT n.id, n.project_id, n.type, n.title, n.description, n.status,
       n.assignee_id, n.tags, n.metadata, n.parent_id,
       n.position_x, n.position_y, n.width, n.height,
       n.created_at, n.updated_at,
       n.description_source, n.description_authored_by,
       n.description_authored_at, n.description_agent_model,
       n.last_verified_at, n.last_verified_by, n.last_verified_commit,
       n.field_provenance, n.created_by,
       n.lifecycle_state, n.lifecycle_state_changed_at,
       COALESCE(u.email, '')::text AS creator_email
FROM nodes n
LEFT JOIN users u ON u.id = n.created_by
WHERE n.project_id = $1;

-- name: NodeUpdateStatus :exec
UPDATE nodes SET status = $1, updated_at = now() WHERE id = $2;

-- name: NodeMarkVerified :execrows
UPDATE nodes
SET last_verified_at = now(),
    last_verified_by = $1,
    last_verified_commit = $2,
    updated_at = now()
WHERE id = $3 AND project_id = $4;

-- name: NodeUpdateDescriptionProvenance :exec
UPDATE nodes
SET description_source = $1,
    description_authored_by = $2,
    description_authored_at = now(),
    description_agent_model = $3
WHERE id = $4;

-- name: NodeBatchUpdateStatus :exec
UPDATE nodes
SET status = sqlc.arg(status), updated_at = now()
WHERE id = ANY(sqlc.arg(node_ids)::uuid[]) AND project_id = sqlc.arg(project_id);

-- name: NodeUnparentChildren :exec
-- Used by Delete and BatchDelete to detach children before removing parent.
UPDATE nodes
SET parent_id = NULL, updated_at = now()
WHERE parent_id = ANY(sqlc.arg(parent_ids)::uuid[]) AND project_id = sqlc.arg(project_id);

-- name: NodeDeleteByID :exec
DELETE FROM nodes WHERE id = $1 AND project_id = $2;

-- name: NodeDeleteByIDs :exec
DELETE FROM nodes
WHERE id = ANY(sqlc.arg(node_ids)::uuid[]) AND project_id = sqlc.arg(project_id);
