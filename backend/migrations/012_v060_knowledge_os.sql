-- 012_v060_knowledge_os.sql
-- v0.6.0 "Knowledge OS Foundation" schema bundle.
--
-- Adds 4 first-class entity types (REQUIREMENT/DECISION/EXPERIMENT/PERSON)
-- with a domain lifecycle field, 9 new edge types, per-edge metadata, plus
-- adjacent tables for comments, attachments, project-scoped tags and API
-- keys. node_history writes are being retired in favor of audit_log — the
-- table itself is kept for read compatibility and dropped in v0.7.0.

-- ── Node type ────────────────────────────────────────────────────────────────
ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'REQUIREMENT';
ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'DECISION';
ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'EXPERIMENT';
ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'PERSON';

-- ── Edge type ────────────────────────────────────────────────────────────────
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'realizes';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'conflicts';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'drives';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'supersedes';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'tests';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'produced';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'owns';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'decided';
ALTER TYPE edge_type ADD VALUE IF NOT EXISTS 'reported';

-- ── Node lifecycle ───────────────────────────────────────────────────────────
ALTER TABLE nodes
    ADD COLUMN IF NOT EXISTS lifecycle_state            TEXT,
    ADD COLUMN IF NOT EXISTS lifecycle_state_changed_at TIMESTAMPTZ;

-- ── Edge metadata ────────────────────────────────────────────────────────────
ALTER TABLE edges
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

-- ── Comments (threaded discussion on nodes) ─────────────────────────────────
CREATE TABLE IF NOT EXISTS node_comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id     UUID NOT NULL REFERENCES nodes(id)    ON DELETE CASCADE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    author_id   UUID NOT NULL REFERENCES users(id)    ON DELETE RESTRICT,
    parent_id   UUID REFERENCES node_comments(id)     ON DELETE CASCADE,
    body        TEXT NOT NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_comments_node
    ON node_comments(node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_project_unresolved
    ON node_comments(project_id, resolved_at)
    WHERE resolved_at IS NULL;

-- ── Attachments (files backed by local FS volume; MinIO/S3 later) ───────────
CREATE TABLE IF NOT EXISTS node_attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_id      UUID NOT NULL REFERENCES nodes(id)    ON DELETE CASCADE,
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    filename     TEXT NOT NULL,
    mime_type    TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL CHECK (size_bytes >= 0),
    storage_key  TEXT NOT NULL UNIQUE,
    sha256       CHAR(64) NOT NULL,
    uploaded_by  UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_attachments_node
    ON node_attachments(node_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_attachments_project
    ON node_attachments(project_id);

-- ── Project tags (canonical metadata; nodes.tags[] stays for storage) ───────
CREATE TABLE IF NOT EXISTS project_tags (
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag         TEXT NOT NULL CHECK (length(tag) BETWEEN 1 AND 64),
    color       TEXT,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (project_id, tag)
);

-- ── API key project scope ───────────────────────────────────────────────────
-- NULL = user-scoped (existing behavior, all of the user's projects).
-- Set   = key can only touch this one project.
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_api_keys_project
    ON api_keys(project_id)
    WHERE project_id IS NOT NULL;
