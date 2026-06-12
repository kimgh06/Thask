-- Migration 011: Add created_by to nodes
-- Backfill from node_history (first 'created' action per node).
-- Nodes with no history entry get NULL (legacy data without tracking).

ALTER TABLE nodes
  ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Backfill: set created_by from the earliest node_history row with action = 'created'
UPDATE nodes n
SET created_by = (
  SELECT nh.user_id
  FROM node_history nh
  WHERE nh.node_id = n.id
    AND nh.action = 'created'
  ORDER BY nh.created_at ASC
  LIMIT 1
)
WHERE n.created_by IS NULL;

CREATE INDEX IF NOT EXISTS idx_nodes_created_by ON nodes(created_by);
