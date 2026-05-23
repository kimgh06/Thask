-- 006_audit_log.sql
-- Unified audit log capturing all write events with 6 orthogonal dimensions
-- (actor, channel, agent, operation, trigger, evidence) to support provenance
-- queries and prevent AI hallucination from being silently promoted into the
-- graph. Replaces node_history over a deprecation window.

CREATE TABLE IF NOT EXISTS audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,

    -- What was touched
    entity_type     TEXT NOT NULL CHECK (entity_type IN ('node','edge','project','graph','suggestion')),
    entity_id       UUID,
    action          TEXT NOT NULL,
    field_name      TEXT,
    old_value       TEXT,
    new_value       TEXT,
    mutation_kind   TEXT NOT NULL DEFAULT 'meta'
                        CHECK (mutation_kind IN ('structural','semantic','meta')),

    -- Actor identity
    user_id         UUID REFERENCES users(id),
    api_key_id      UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    actor_kind      TEXT NOT NULL DEFAULT 'system'
                        CHECK (actor_kind IN ('user_interactive','agent','service','scanner','system')),

    -- Channel (how the call arrived)
    client_type     TEXT,
    client_version  TEXT,

    -- Agent context (only meaningful when actor_kind = 'agent')
    agent_model     TEXT,
    agent_session_id TEXT,

    -- Operation context
    trigger         TEXT,
    batch_id        UUID,

    -- Evidence (optional, agent-reported)
    code_commit     TEXT,
    source_path     TEXT,
    confidence      TEXT CHECK (confidence IN ('low','medium','high')),

    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_project_time
    ON audit_log(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_entity
    ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor
    ON audit_log(actor_kind, agent_model);
CREATE INDEX IF NOT EXISTS idx_audit_batch
    ON audit_log(batch_id) WHERE batch_id IS NOT NULL;
