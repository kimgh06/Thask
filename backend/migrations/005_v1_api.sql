-- 005_v1_api.sql
-- Indexes for cursor-based pagination and idempotency support for v1 API.

-- Pagination support indexes (keyset on created_at, id)
CREATE INDEX IF NOT EXISTS idx_nodes_pagination ON nodes(project_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_edges_pagination ON edges(project_id, created_at, id);

-- Idempotency keys for deduplicating POST/PATCH/DELETE requests
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key         VARCHAR(256) NOT NULL,
    api_key_id  UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    method      TEXT NOT NULL,
    path        TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    response    JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '24 hours',
    PRIMARY KEY (key, api_key_id)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expiry ON idempotency_keys(expires_at);
