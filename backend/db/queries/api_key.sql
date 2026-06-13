-- ============================================================================
-- APIKey static queries (sqlc-generated)
--
-- FindByKeyHash (JOIN that splits into two separate model structs) stays as
-- hand-written pgx in internal/repository/api_key.go.
-- ============================================================================

-- name: APIKeyCreate :one
INSERT INTO api_keys (user_id, name, key_prefix, key_hash, kind, permissions, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, user_id, name, key_prefix, kind, permissions, last_used_at, expires_at, created_at;

-- name: APIKeyFindByUserID :many
SELECT id, user_id, name, key_prefix, kind, permissions, last_used_at, expires_at, created_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: APIKeyDelete :execrows
DELETE FROM api_keys WHERE id = $1 AND user_id = $2;

-- name: APIKeyUpdateLastUsed :exec
UPDATE api_keys SET last_used_at = now() WHERE id = $1;

-- name: APIKeyCountByUserID :one
SELECT COUNT(*) FROM api_keys WHERE user_id = $1;
