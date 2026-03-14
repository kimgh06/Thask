import type { NodeType, NodeStatus } from '$lib/types';

export const NODE_TYPES: NodeType[] = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP'];
export const NODE_TYPES_NO_GROUP: NodeType[] = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI'];
export const STATUS_OPTIONS: NodeStatus[] = ['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED'];

export const TYPE_COLORS: Record<NodeType, string> = {
	FLOW: '#6366f1',
	BRANCH: '#8b5cf6',
	TASK: '#3b82f6',
	BUG: '#ef4444',
	API: '#22c55e',
	UI: '#f59e0b',
	GROUP: '#64748b',
};

export const STATUS_COLORS: Record<NodeStatus, string> = {
	PASS: '#22c55e',
	FAIL: '#ef4444',
	IN_PROGRESS: '#6366f1',
	BLOCKED: '#f59e0b',
};

export const STATUS_LABELS: Record<NodeStatus, string> = {
	PASS: 'Pass',
	FAIL: 'Fail',
	IN_PROGRESS: 'In Progress',
	BLOCKED: 'Blocked',
};

export const NODE_SHAPES: Record<string, string> = {
	FLOW: 'round-rectangle',
	BRANCH: 'diamond',
	TASK: 'rectangle',
	BUG: 'hexagon',
	API: 'barrel',
	UI: 'ellipse',
	GROUP: 'round-rectangle',
};
