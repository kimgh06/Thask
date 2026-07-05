-- ============================================================================
-- Project tag queries (v0.6.0 B-3). Canonical metadata (color, description)
-- for tags used across a project. nodes.tags TEXT[] is still the primary
-- storage — project_tags augments it for autocomplete and consistent styling.
-- ============================================================================

-- name: ProjectTagUpsert :one
INSERT INTO project_tags (project_id, tag, color, description, created_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, tag) DO UPDATE
   SET color       = EXCLUDED.color,
       description = EXCLUDED.description
RETURNING project_id, tag, color, description, created_at, created_by;

-- name: ProjectTagList :many
SELECT project_id, tag, color, description, created_at, created_by
FROM project_tags
WHERE project_id = $1
ORDER BY tag ASC;

-- name: ProjectTagDelete :execrows
DELETE FROM project_tags WHERE project_id = $1 AND tag = $2;
