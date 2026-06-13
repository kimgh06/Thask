-- ============================================================================
-- Session static queries (sqlc-generated)
-- ============================================================================

-- name: SessionCreate :exec
INSERT INTO sessions (user_id, token, expires_at)
VALUES ($1, $2, $3);

-- name: SessionValidateToken :one
SELECT u.id, u.email, u.display_name, u.password_hash, u.created_at, u.updated_at
FROM sessions s
INNER JOIN users u ON s.user_id = u.id
WHERE s.token = $1 AND s.expires_at > now();

-- name: SessionDeleteByToken :exec
DELETE FROM sessions WHERE token = $1;

-- name: SessionDeleteByUserID :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: SessionDeleteExpired :exec
DELETE FROM sessions WHERE expires_at < now();
