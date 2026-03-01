export const NODE_TYPES = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP'] as const;
export type NodeType = (typeof NODE_TYPES)[number];

export const NODE_STATUSES = ['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED'] as const;
export type NodeStatus = (typeof NODE_STATUSES)[number];

export const EDGE_TYPES = [
  'depends_on',
  'blocks',
  'related',
  'parent_child',
  'triggers',
] as const;
export type EdgeType = (typeof EDGE_TYPES)[number];

export interface GraphNode {
  id: string;
  projectId: string;
  type: NodeType;
  title: string;
  description: string | null;
  status: NodeStatus;
  assigneeId: string | null;
  assigneeName?: string;
  tags: string[];
  metadata: Record<string, unknown>;
  parentId: string | null;
  positionX: number;
  positionY: number;
  width: number | null;
  height: number | null;
  createdAt: string;
  updatedAt: string;
}

export interface GraphEdge {
  id: string;
  projectId: string;
  sourceId: string;
  targetId: string;
  edgeType: EdgeType;
  label: string | null;
  createdAt: string;
}

export interface ProjectGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface NodeHistoryEntry {
  id: string;
  nodeId: string;
  userId: string;
  userName?: string;
  action: 'created' | 'updated' | 'deleted' | 'status_changed';
  fieldName: string | null;
  oldValue: string | null;
  newValue: string | null;
  createdAt: string;
}

export interface ImpactResult {
  changedNodes: GraphNode[];
  impactedNodes: GraphNode[];
  failNodes: GraphNode[];
  impactEdges: GraphEdge[];
}
