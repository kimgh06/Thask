-- ============================================================================
-- Node comment queries (v0.6.0 B-1). Threaded discussion attached to any node.
-- parent_id is self-referential for one-level threading; deeper nesting is
-- out of scope for v0.6.0.
-- ============================================================================

-- name: CommentCreate :one
INSERT INTO node_comments (node_id, project_id, author_id, parent_id, body)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, node_id, project_id, author_id, parent_id, body,
          resolved_at, resolved_by, created_at, updated_at;

-- name: CommentFindByNode :many
SELECT id, node_id, project_id, author_id, parent_id, body,
       resolved_at, resolved_by, created_at, updated_at
FROM node_comments
WHERE node_id = $1
ORDER BY created_at ASC;

-- name: CommentFindByProjectUnresolved :many
SELECT id, node_id, project_id, author_id, parent_id, body,
       resolved_at, resolved_by, created_at, updated_at
FROM node_comments
WHERE project_id = $1 AND resolved_at IS NULL
ORDER BY created_at DESC
LIMIT $2;

-- name: CommentUpdateBody :one
UPDATE node_comments
SET body = $1, updated_at = now()
WHERE id = $2 AND project_id = $3 AND author_id = $4
RETURNING id, node_id, project_id, author_id, parent_id, body,
          resolved_at, resolved_by, created_at, updated_at;

-- name: CommentResolve :one
UPDATE node_comments
SET resolved_at = now(), resolved_by = $1, updated_at = now()
WHERE id = $2 AND project_id = $3 AND resolved_at IS NULL
RETURNING id, node_id, project_id, author_id, parent_id, body,
          resolved_at, resolved_by, created_at, updated_at;

-- name: CommentDelete :execrows
DELETE FROM node_comments WHERE id = $1 AND project_id = $2 AND author_id = $3;
