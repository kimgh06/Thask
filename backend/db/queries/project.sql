-- ============================================================================
-- Project static queries (sqlc-generated)
--
-- Dynamic queries:
--   GraphUpdatedAt (multi-subquery GREATEST across 3 tables) stays hand-written.
-- ============================================================================

-- name: ProjectCreate :one
INSERT INTO projects (team_id, name, description, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at;

-- name: ProjectFindByID :one
SELECT id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: ProjectFindByShareToken :one
SELECT id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at
FROM projects
WHERE share_token_hash = $1 AND link_sharing != 'off';

-- name: ProjectFindByTeamID :many
SELECT id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at
FROM projects
WHERE team_id = $1;

-- name: ProjectFindByTeamIDs :many
SELECT id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at
FROM projects
WHERE team_id = ANY(sqlc.arg(team_ids)::uuid[]);

-- name: ProjectUpdate :one
UPDATE projects
SET name        = COALESCE($1, name),
    description = COALESCE($2, description),
    updated_at  = now()
WHERE id = $3
RETURNING id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at;

-- name: ProjectUpdateLinkSharing :one
UPDATE projects
SET link_sharing    = $1,
    share_token     = $2,
    share_token_hash = $3,
    updated_at      = now()
WHERE id = $4
RETURNING id, team_id, name, description, created_by, link_sharing, share_token, share_token_hash, created_at, updated_at;

-- name: ProjectDelete :exec
DELETE FROM projects WHERE id = $1;

-- name: ProjectVerifyAccess :one
SELECT EXISTS(
  SELECT 1 FROM projects p
  INNER JOIN team_members tm ON tm.team_id = p.team_id AND tm.user_id = $2
  WHERE p.id = $1
)::boolean;
