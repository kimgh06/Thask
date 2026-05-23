-- 009_suggestions.sql
-- Agent-proposed node updates land here instead of directly mutating nodes,
-- so human reviewers gate every semantic change. Provides a queue surface for
-- thask.suggestions.list and thask.suggestions.decide.

CREATE TABLE IF NOT EXISTS node_suggestions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    node_id           UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    field_name        TEXT NOT NULL DEFAULT 'description',
    proposed_value    TEXT NOT NULL,
    current_value     TEXT,
    rationale         TEXT,
    evidence          JSONB NOT NULL DEFAULT '{}',  -- { code_commit, source_paths[], confidence }

    proposed_by       UUID NOT NULL REFERENCES users(id),
    agent_model       TEXT,
    agent_session_id  TEXT,

    status            TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','accepted','rejected','expired','superseded')),
    decided_by        UUID REFERENCES users(id),
    decided_at        TIMESTAMPTZ,
    decided_reason    TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_suggestions_pending
    ON node_suggestions(project_id, status)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_suggestions_node
    ON node_suggestions(node_id, created_at DESC);
