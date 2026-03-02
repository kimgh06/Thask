import type { NodeStatus, EdgeType } from '@/types/graph';

export interface WaterfallNode {
  id: string;
  status: NodeStatus;
  parentId: string | null;
}

export interface WaterfallEdge {
  sourceId: string;
  targetId: string;
  edgeType: EdgeType;
}

export interface StatusChange {
  nodeId: string;
  oldStatus: NodeStatus;
  newStatus: NodeStatus;
}

const MAX_DEPTH = 10;

/**
 * Compute cascading status changes triggered by a node's status update.
 *
 * Pure function — does not touch the DB.  The caller is responsible for
 * persisting the returned changes and recording history.
 */
export function computeWaterfall(
  changedNodeId: string,
  newStatus: NodeStatus,
  allNodes: WaterfallNode[],
  allEdges: WaterfallEdge[],
): StatusChange[] {
  // Only PASS and FAIL trigger propagation
  if (newStatus !== 'PASS' && newStatus !== 'FAIL') return [];

  const nodeMap = new Map(allNodes.map((n) => [n.id, { ...n }]));
  // Apply the triggering change first
  const trigger = nodeMap.get(changedNodeId);
  if (trigger) trigger.status = newStatus;

  const changes: StatusChange[] = [];
  const visited = new Set<string>([changedNodeId]);

  // BFS queue: [nodeId, depth]
  const queue: [string, number][] = [[changedNodeId, 0]];

  while (queue.length > 0) {
    const [currentId, depth] = queue.shift()!;
    if (depth >= MAX_DEPTH) continue;

    const current = nodeMap.get(currentId);
    if (!current) continue;

    const derived = deriveChanges(currentId, current.status, nodeMap, allEdges);

    for (const change of derived) {
      if (visited.has(change.nodeId)) continue;
      visited.add(change.nodeId);

      const target = nodeMap.get(change.nodeId);
      if (!target) continue;
      if (target.status === change.newStatus) continue;

      change.oldStatus = target.status;
      target.status = change.newStatus;
      changes.push(change);
      queue.push([change.nodeId, depth + 1]);
    }

    // Parent-child aggregation: if current node has a parent, re-evaluate the parent
    if (current.parentId && !visited.has(current.parentId)) {
      const parentChange = evaluateParent(current.parentId, nodeMap, allNodes);
      if (parentChange) {
        visited.add(parentChange.nodeId);
        const parent = nodeMap.get(parentChange.nodeId);
        if (parent) {
          parent.status = parentChange.newStatus;
          changes.push(parentChange);
          queue.push([parentChange.nodeId, depth + 1]);
        }
      }
    }
  }

  return changes;
}

/** Derive direct status changes from a single node's new status. */
function deriveChanges(
  nodeId: string,
  status: NodeStatus,
  nodeMap: Map<string, WaterfallNode>,
  allEdges: WaterfallEdge[],
): StatusChange[] {
  const results: StatusChange[] = [];

  for (const edge of allEdges) {
    if (edge.edgeType === 'related' || edge.edgeType === 'parent_child') continue;

    // ── blocks: A --blocks--> B ──
    if (edge.edgeType === 'blocks' && edge.sourceId === nodeId) {
      const target = nodeMap.get(edge.targetId);
      if (!target) continue;

      if (status === 'PASS' && target.status === 'BLOCKED') {
        // Check all other blockers of this target
        const otherBlockers = allEdges.filter(
          (e) => e.edgeType === 'blocks' && e.targetId === edge.targetId && e.sourceId !== nodeId,
        );
        const allResolved = otherBlockers.every((e) => nodeMap.get(e.sourceId)?.status === 'PASS');
        if (allResolved) {
          results.push({ nodeId: edge.targetId, oldStatus: target.status, newStatus: 'IN_PROGRESS' });
        }
      } else if (status === 'FAIL' && target.status === 'IN_PROGRESS') {
        results.push({ nodeId: edge.targetId, oldStatus: target.status, newStatus: 'BLOCKED' });
      }
    }

    // ── depends_on: A --depends_on--> B  (A depends on B) ──
    if (edge.edgeType === 'depends_on' && edge.targetId === nodeId) {
      const dependent = nodeMap.get(edge.sourceId);
      if (!dependent) continue;

      if (status === 'PASS' && dependent.status === 'BLOCKED') {
        // Check all dependencies of this dependent
        const allDeps = allEdges.filter(
          (e) => e.edgeType === 'depends_on' && e.sourceId === edge.sourceId,
        );
        const allMet = allDeps.every((e) => nodeMap.get(e.targetId)?.status === 'PASS');
        if (allMet) {
          results.push({ nodeId: edge.sourceId, oldStatus: dependent.status, newStatus: 'IN_PROGRESS' });
        }
      } else if (status === 'FAIL' && dependent.status === 'IN_PROGRESS') {
        results.push({ nodeId: edge.sourceId, oldStatus: dependent.status, newStatus: 'BLOCKED' });
      }
    }

    // ── triggers: A --triggers--> B ──
    if (edge.edgeType === 'triggers' && edge.sourceId === nodeId) {
      const target = nodeMap.get(edge.targetId);
      if (!target) continue;

      if (status === 'PASS' && target.status === 'BLOCKED') {
        results.push({ nodeId: edge.targetId, oldStatus: target.status, newStatus: 'IN_PROGRESS' });
      }
    }
  }

  return results;
}

/** Evaluate GROUP parent status from children. */
function evaluateParent(
  parentId: string,
  nodeMap: Map<string, WaterfallNode>,
  allNodes: WaterfallNode[],
): StatusChange | null {
  const parent = nodeMap.get(parentId);
  if (!parent) return null;

  const children = allNodes.filter((n) => n.parentId === parentId);
  if (children.length === 0) return null;

  const statuses = children.map((c) => nodeMap.get(c.id)?.status ?? c.status);

  let newStatus: NodeStatus;
  if (statuses.every((s) => s === 'PASS')) {
    newStatus = 'PASS';
  } else if (statuses.some((s) => s === 'FAIL')) {
    newStatus = 'FAIL';
  } else {
    newStatus = 'IN_PROGRESS';
  }

  if (newStatus === parent.status) return null;

  return { nodeId: parentId, oldStatus: parent.status, newStatus };
}
