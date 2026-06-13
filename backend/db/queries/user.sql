-- ============================================================================
-- User static queries (sqlc-generated)
-- ============================================================================

-- name: UserCreate :one
INSERT INTO users (email, display_name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, display_name, password_hash, created_at, updated_at;

-- name: UserFindByEmail :one
SELECT id, email, display_name, password_hash, created_at, updated_at
FROM users
WHERE email = $1;

-- name: UserFindByID :one
SELECT id, email, display_name, password_hash, created_at, updated_at
FROM users
WHERE id = $1;
