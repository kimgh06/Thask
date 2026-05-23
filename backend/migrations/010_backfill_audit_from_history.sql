-- 010_backfill_audit_from_history.sql
-- One-time backfill: copy node_history rows into audit_log so we can deprecate
-- node_history without losing historical context. Only runs when audit_log is
-- empty (i.e., on the first deployment after migrations 006~009 land).
--
-- Provenance for historical rows is partial: pre-migration writes happened
-- before X-Thask-Client / api_keys.kind existed, so actor_kind defaults to
-- 'user_interactive' and channel fields stay NULL. Acceptable trade-off —
-- backfilled data marks itself as "unknown channel" via NULL client_type.

INSERT INTO audit_log (
    project_id, entity_type, entity_id, action, field_name,
    old_value, new_value, mutation_kind, user_id, actor_kind,
    metadata, created_at
)
SELECT
    nh.project_id,
    'node',
    nh.node_id,
    nh.action::text,
    nh.field_name,
    nh.old_value,
    nh.new_value,
    CASE
        WHEN nh.field_name = 'description'               THEN 'semantic'
        WHEN nh.field_name IN ('type', 'parent_id')      THEN 'structural'
        WHEN nh.action::text = 'created'                 THEN 'structural'
        ELSE 'meta'
    END,
    nh.user_id,
    'user_interactive',
    jsonb_build_object('backfilled_from', 'node_history'),
    nh.created_at
FROM node_history nh
WHERE NOT EXISTS (SELECT 1 FROM audit_log LIMIT 1);
