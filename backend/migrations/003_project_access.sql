ALTER TABLE projects ADD COLUMN IF NOT EXISTS link_sharing TEXT NOT NULL DEFAULT 'off';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS share_token TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS share_token_hash TEXT UNIQUE;

CREATE TABLE IF NOT EXISTS project_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('editor', 'viewer')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, user_id)
);
