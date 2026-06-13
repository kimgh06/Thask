-- ============================================================================
-- NodeHistory static queries (sqlc-generated)
--
-- BatchCreateStatusChanges stays as hand-written pgx (dynamic multi-row INSERT
-- with fmt.Sprintf).
-- ============================================================================

-- name: HistoryCreate :exec
INSERT INTO node_history (node_id, project_id, user_id, action, field_name, old_value, new_value)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: HistoryFindByProjectID :many
SELECT nh.id, nh.action, nh.field_name, nh.old_value, nh.new_value, nh.created_at, u.display_name
FROM node_history nh
INNER JOIN users u ON nh.user_id = u.id
WHERE nh.project_id = $1
ORDER BY nh.created_at DESC
LIMIT $2;

-- name: HistoryFindByNodeID :many
SELECT nh.id, nh.action, nh.field_name, nh.old_value, nh.new_value, nh.created_at, u.display_name
FROM node_history nh
INNER JOIN users u ON nh.user_id = u.id
WHERE nh.node_id = $1
ORDER BY nh.created_at DESC
LIMIT $2;
