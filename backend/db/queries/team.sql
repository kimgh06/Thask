-- ============================================================================
-- Team static queries (sqlc-generated)
--
-- Dynamic / multi-statement queries kept as hand-written pgx in
-- internal/repository/team.go:
--   GetMembers     — JOIN team_members + users into a flat struct
--   TransferOwnership — two-statement tx (demote old owner, promote new)
-- ============================================================================

-- name: TeamCreate :one
INSERT INTO teams (name, slug, created_by)
VALUES ($1, $2, $3)
RETURNING id, name, slug, created_by, created_at, updated_at;

-- name: TeamFindBySlug :one
SELECT id, name, slug, created_by, created_at, updated_at
FROM teams
WHERE slug = $1;

-- name: TeamFindByUserID :many
SELECT t.id, t.name, t.slug, t.created_by, t.created_at, t.updated_at
FROM teams t
INNER JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = $1;

-- name: TeamDelete :exec
DELETE FROM teams WHERE id = $1;

-- name: TeamUpdate :one
UPDATE teams
SET name = $1, updated_at = now()
WHERE id = $2
RETURNING id, name, slug, created_by, created_at, updated_at;

-- name: TeamMemberAdd :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3);

-- name: TeamMemberExists :one
SELECT EXISTS(
  SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2
)::boolean;

-- name: TeamMemberRemove :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: TeamMemberGetRole :one
SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: TeamMemberUpdateRole :execrows
UPDATE team_members
SET role = $3
WHERE team_id = $1 AND user_id = $2;

-- name: TeamMemberCountByRole :one
SELECT COUNT(*)::int FROM team_members WHERE team_id = $1 AND role = $2;
