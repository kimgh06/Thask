-- ============================================================================
-- NodeSuggestion static queries (sqlc-generated)
-- ============================================================================

-- name: SuggestionCreate :one
INSERT INTO node_suggestions (
    project_id, node_id, field_name, proposed_value, current_value,
    rationale, evidence, proposed_by, agent_model, agent_session_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, status, created_at;

-- name: SuggestionListPending :many
SELECT id, project_id, node_id, field_name, proposed_value, current_value,
       rationale, evidence, proposed_by, agent_model, agent_session_id,
       status, decided_by, decided_at, decided_reason, created_at
FROM node_suggestions
WHERE project_id = $1 AND status = 'pending'
ORDER BY created_at DESC
LIMIT $2;

-- name: SuggestionListForNode :many
SELECT id, project_id, node_id, field_name, proposed_value, current_value,
       rationale, evidence, proposed_by, agent_model, agent_session_id,
       status, decided_by, decided_at, decided_reason, created_at
FROM node_suggestions
WHERE node_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: SuggestionFindByID :one
SELECT id, project_id, node_id, field_name, proposed_value, current_value,
       rationale, evidence, proposed_by, agent_model, agent_session_id,
       status, decided_by, decided_at, decided_reason, created_at
FROM node_suggestions
WHERE id = $1;

-- name: SuggestionDecide :execrows
UPDATE node_suggestions
SET status = $1, decided_by = $2, decided_at = now(), decided_reason = $3
WHERE id = $4 AND status = 'pending';

-- name: SuggestionSupersedePendingForNode :exec
UPDATE node_suggestions
SET status = 'superseded', decided_at = now()
WHERE node_id = $1 AND field_name = $2 AND status = 'pending' AND id != $3;
