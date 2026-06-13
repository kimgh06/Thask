-- ============================================================================
-- AuditLog static queries (sqlc-generated)
-- ============================================================================

-- name: AuditLogInsert :exec
INSERT INTO audit_log (
    project_id, entity_type, entity_id, action, field_name,
    old_value, new_value, mutation_kind, user_id, api_key_id,
    actor_kind, client_type, client_version, agent_model, agent_session_id,
    trigger, batch_id, code_commit, source_path, confidence, metadata
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15,
    $16, $17, $18, $19, $20, $21
);

-- name: AuditLogListByEntity :many
SELECT id, project_id, entity_type, entity_id, action, field_name,
       old_value, new_value, mutation_kind, user_id, api_key_id,
       actor_kind, client_type, client_version, agent_model, agent_session_id,
       trigger, batch_id, code_commit, source_path, confidence, metadata, created_at
FROM audit_log
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC
LIMIT $3;
