-- ============================================================================
-- ProjectMember static queries (sqlc-generated)
--
-- Dynamic queries:
--   ListWithUsers (JOIN projects_members + users producing flat
--   ProjectMemberWithUser) stays hand-written pgx — sqlc can't split one row
--   into two destination structs cleanly.
-- ============================================================================

-- name: ProjectMemberGetRole :one
SELECT role FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: ProjectMemberAdd :exec
INSERT INTO project_members (project_id, user_id, role)
VALUES ($1, $2, $3);

-- name: ProjectMemberUpdateRole :execrows
UPDATE project_members
SET role = $3
WHERE project_id = $1 AND user_id = $2;

-- name: ProjectMemberRemove :execrows
DELETE FROM project_members
WHERE project_id = $1 AND user_id = $2;
