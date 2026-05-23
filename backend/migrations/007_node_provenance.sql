-- 007_node_provenance.sql
-- Snapshot provenance on the nodes table so callers can answer
-- "who/when/which model wrote this description" without joining audit_log.
-- audit_log is the movie; these columns are the freeze-frame.

ALTER TABLE nodes
    ADD COLUMN IF NOT EXISTS description_source       TEXT NOT NULL DEFAULT 'unknown'
        CHECK (description_source IN ('human','agent','scanner','import','unknown')),
    ADD COLUMN IF NOT EXISTS description_authored_by  UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS description_authored_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS description_agent_model  TEXT,
    ADD COLUMN IF NOT EXISTS last_verified_at         TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_verified_by         UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS last_verified_commit     TEXT,
    ADD COLUMN IF NOT EXISTS field_provenance         JSONB NOT NULL DEFAULT '{}';

-- Index for "show me unverified-agent description nodes" stale review queries.
CREATE INDEX IF NOT EXISTS idx_nodes_unverified
    ON nodes(project_id, last_verified_at)
    WHERE description_source = 'agent';
