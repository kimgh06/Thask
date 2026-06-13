-- ============================================================================
-- Idempotency static queries (sqlc-generated)
-- ============================================================================

-- name: IdempotencyFind :one
SELECT key, api_key_id, method, path, status_code, response, created_at, expires_at
FROM idempotency_keys
WHERE key = $1 AND api_key_id = $2 AND expires_at > now();

-- name: IdempotencyTryClaim :one
INSERT INTO idempotency_keys (key, api_key_id, method, path, status_code, response)
VALUES ($1, $2, $3, $4, 0, '{}'::jsonb)
ON CONFLICT (key, api_key_id) DO UPDATE SET key = idempotency_keys.key
RETURNING key, api_key_id, method, path, status_code, response, created_at, expires_at;

-- name: IdempotencyUpdateResponse :exec
UPDATE idempotency_keys SET status_code = $3, response = $4 WHERE key = $1 AND api_key_id = $2;

-- name: IdempotencyCleanup :execrows
DELETE FROM idempotency_keys WHERE expires_at < now();
