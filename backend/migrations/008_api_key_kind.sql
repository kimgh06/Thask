-- 008_api_key_kind.sql
-- Classify API keys by intended caller (human/agent/service) and attach
-- a per-key permissions JSONB. The middleware uses these to decide whether
-- a write attempt is allowed, and to populate audit_log.actor_kind.
-- Default for new keys (when permissions = '{}') is all-true to preserve
-- backwards compatibility; new key creation paths supply explicit values.

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS kind        TEXT NOT NULL DEFAULT 'user_interactive'
        CHECK (kind IN ('user_interactive','agent','service')),
    ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '{}';

-- Backfill: existing keys keep current behavior (all permissions on).
-- Empty '{}' is treated as "no overrides", but for clarity we set every flag.
UPDATE api_keys
SET permissions = jsonb_build_object(
        'read', true,
        'write_structural', true,
        'write_semantic', true,
        'write_meta', true,
        'verify', true,
        'delete', true,
        'suggest', true
    )
WHERE permissions = '{}'::jsonb;
