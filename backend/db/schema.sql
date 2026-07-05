-- Aggregated schema snapshot for sqlc parsing.
--
-- Runtime migrations (migrations/*.sql) are the source of truth. This file
-- exists solely because sqlc's parser cannot resolve types declared inside
-- anonymous PL/pgSQL blocks (DO $$ ... $$), which is how 001_initial.sql
-- creates the enum types idempotently. Every time you touch an enum or a
-- table column, mirror the change here so `make sqlc-gen` sees the current
-- shape.

-- ── Enum types ──────────────────────────────────────────────────────────────
CREATE TYPE team_role AS ENUM ('owner', 'admin', 'member', 'viewer');
CREATE TYPE node_type AS ENUM (
    'FLOW','BRANCH','TASK','BUG','API','UI','GROUP',
    'REQUIREMENT','DECISION','EXPERIMENT','PERSON'
);
CREATE TYPE node_status AS ENUM ('PASS','FAIL','IN_PROGRESS','BLOCKED');
CREATE TYPE edge_type AS ENUM (
    'depends_on','blocks','related','parent_child','triggers',
    'realizes','conflicts','drives','supersedes','tests','produced',
    'owns','decided','reported'
);
CREATE TYPE history_action AS ENUM ('created','updated','deleted','status_changed');

-- ── Core tables ─────────────────────────────────────────────────────────────
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE teams (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE team_members (
    id UUID PRIMARY KEY,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role team_role NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE projects (
    id UUID PRIMARY KEY,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    link_sharing TEXT NOT NULL,
    share_token TEXT,
    share_token_hash TEXT UNIQUE
);

CREATE TABLE project_members (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE nodes (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    type node_type NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status node_status NOT NULL,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    tags TEXT[],
    metadata JSONB,
    parent_id UUID,
    position_x DOUBLE PRECISION,
    position_y DOUBLE PRECISION,
    width DOUBLE PRECISION,
    height DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    description_source TEXT NOT NULL,
    description_authored_by UUID REFERENCES users(id),
    description_authored_at TIMESTAMPTZ,
    description_agent_model TEXT,
    last_verified_at TIMESTAMPTZ,
    last_verified_by UUID REFERENCES users(id),
    last_verified_commit TEXT,
    field_provenance JSONB NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    lifecycle_state TEXT,
    lifecycle_state_changed_at TIMESTAMPTZ
);

CREATE TABLE edges (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    edge_type edge_type NOT NULL,
    label TEXT,
    source_port TEXT NOT NULL,
    target_port TEXT NOT NULL,
    waypoints JSONB NOT NULL,
    metadata JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE node_history (
    id UUID PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    action history_action NOT NULL,
    field_name TEXT,
    old_value TEXT,
    new_value TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_prefix CHAR(12) NOT NULL,
    key_hash TEXT NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    kind TEXT NOT NULL,
    permissions JSONB NOT NULL,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE idempotency_keys (
    key VARCHAR(256) NOT NULL,
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    response JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (key, api_key_id)
);

CREATE TABLE audit_log (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    action TEXT NOT NULL,
    field_name TEXT,
    old_value TEXT,
    new_value TEXT,
    mutation_kind TEXT NOT NULL,
    user_id UUID REFERENCES users(id),
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    actor_kind TEXT NOT NULL,
    client_type TEXT,
    client_version TEXT,
    agent_model TEXT,
    agent_session_id TEXT,
    trigger TEXT,
    batch_id UUID,
    code_commit TEXT,
    source_path TEXT,
    confidence TEXT,
    metadata JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE node_suggestions (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    field_name TEXT NOT NULL,
    proposed_value TEXT NOT NULL,
    current_value TEXT,
    rationale TEXT,
    evidence JSONB NOT NULL,
    proposed_by UUID NOT NULL REFERENCES users(id),
    agent_model TEXT,
    agent_session_id TEXT,
    status TEXT NOT NULL,
    decided_by UUID REFERENCES users(id),
    decided_at TIMESTAMPTZ,
    decided_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE node_comments (
    id UUID PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_id UUID REFERENCES node_comments(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE node_attachments (
    id UUID PRIMARY KEY,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL UNIQUE,
    sha256 CHAR(64) NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE project_tags (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    color TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (project_id, tag)
);
