-- ============================================================================
-- Node attachment queries (v0.6.0 B-2). Filesystem-backed for now; storage_key
-- is a relative path inside THASK_ATTACHMENT_DIR (see cmd/server/main.go and
-- docker-compose.yml). MinIO/S3 swap-in later reuses the same rows.
-- ============================================================================

-- name: AttachmentCreate :one
INSERT INTO node_attachments
    (node_id, project_id, filename, mime_type, size_bytes,
     storage_key, sha256, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, node_id, project_id, filename, mime_type, size_bytes,
          storage_key, sha256, uploaded_by, created_at;

-- name: AttachmentFindByNode :many
SELECT id, node_id, project_id, filename, mime_type, size_bytes,
       storage_key, sha256, uploaded_by, created_at
FROM node_attachments
WHERE node_id = $1
ORDER BY created_at DESC;

-- name: AttachmentFindByID :one
SELECT id, node_id, project_id, filename, mime_type, size_bytes,
       storage_key, sha256, uploaded_by, created_at
FROM node_attachments
WHERE id = $1;

-- name: AttachmentDelete :execrows
DELETE FROM node_attachments WHERE id = $1 AND project_id = $2;
