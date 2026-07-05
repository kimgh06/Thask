-- ============================================================================
-- NodeHistory static queries (sqlc-generated)
--
-- BatchCreateStatusChanges stays as hand-written pgx (dynamic multi-row INSERT
-- with fmt.Sprintf).
-- ============================================================================

-- name: HistoryCreate :exec
INSERT INTO node_history (node_id, project_id, user_id, action, field_name, old_value, new_value)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- v0.6.0: reads now come from audit_log (node_history is deprecated and its
-- writes were disabled in v0.6.0). Migration 010 backfilled every prior
-- node_history row into audit_log, so callers see one continuous timeline
-- covering both pre- and post-v0.6.0 activity without stitching two tables.
-- Only node entity events are surfaced here — edges/projects/graph events
-- live in their own audit views.

-- name: HistoryFindByProjectID :many
SELECT a.id, a.action, a.field_name, a.old_value, a.new_value, a.created_at,
       COALESCE(u.display_name, '')::text AS display_name
FROM audit_log a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.project_id = $1 AND a.entity_type = 'node'
ORDER BY a.created_at DESC
LIMIT $2;

-- name: HistoryFindByNodeID :many
SELECT a.id, a.action, a.field_name, a.old_value, a.new_value, a.created_at,
       COALESCE(u.display_name, '')::text AS display_name
FROM audit_log a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.entity_id = $1 AND a.entity_type = 'node'
ORDER BY a.created_at DESC
LIMIT $2;
