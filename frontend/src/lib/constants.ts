import type { NodeType, NodeStatus, EdgeType } from '$lib/types';

export const NODE_TYPES: NodeType[] = [
	'FLOW',
	'BRANCH',
	'TASK',
	'BUG',
	'API',
	'UI',
	'GROUP',
	'REQUIREMENT',
	'DECISION',
	'EXPERIMENT',
	'PERSON',
];
export const NODE_TYPES_NO_GROUP: NodeType[] = [
	'FLOW',
	'BRANCH',
	'TASK',
	'BUG',
	'API',
	'UI',
	'REQUIREMENT',
	'DECISION',
	'EXPERIMENT',
	'PERSON',
];
export const STATUS_OPTIONS: NodeStatus[] = ['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED'];

export const TYPE_COLORS: Record<NodeType, string> = {
	FLOW: '#e2b340',
	BRANCH: '#a78bfa',
	TASK: '#7ca3c4',
	BUG: '#e05252',
	API: '#5ea87a',
	UI: '#d4915a',
	GROUP: '#7c7570',
	// v0.6.0 Knowledge OS entities. Palette picks lean on the "Amber Precision"
	// spectrum: teal for REQUIREMENT (specifies), amber for DECISION (chosen),
	// violet for EXPERIMENT (probing), warm neutral for PERSON (human).
	REQUIREMENT: '#4a9fb0',
	DECISION: '#c9942e',
	EXPERIMENT: '#9370c4',
	PERSON: '#c48a4d',
};

export const STATUS_COLORS: Record<NodeStatus, string> = {
	PASS: '#5ea87a',
	FAIL: '#e05252',
	IN_PROGRESS: '#e2b340',
	BLOCKED: '#c26a3d',
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
	REQUIREMENT: 'round-rectangle',
	DECISION: 'diamond',
	EXPERIMENT: 'octagon',
	PERSON: 'ellipse',
};

export const EDGE_COLORS: Record<EdgeType, string> = {
	depends_on: '#a78bfa',
	blocks: '#e05252',
	related: '#7c7570',
	parent_child: '#7ca3c4',
	triggers: '#e2b340',
	// v0.6.0 semantic edges reuse category hues from TYPE_COLORS so viewers
	// can trace at a glance which entity kind an edge belongs to.
	realizes: '#4a9fb0',
	conflicts: '#e05252',
	drives: '#c9942e',
	supersedes: '#c9942e',
	tests: '#9370c4',
	produced: '#9370c4',
	owns: '#c48a4d',
	decided: '#c48a4d',
	reported: '#c48a4d',
};

/** Design system color constants for use in Cytoscape styles (which cannot use CSS variables) */
export const COLORS = {
	bg: '#131214',
	surface: '#1b1a1e',
	surfaceHover: '#26252a',
	border: '#26252a',
	borderSubtle: '#1f1e22',
	text: '#ededec',
	textMuted: '#9e9c97',
	textDim: '#6b6966',
	accent: '#c9a84c',
	accentHover: '#a8893a',
	accentMuted: '#7d6b35',
	danger: '#c44040',
	success: '#5ea87a',
	warning: '#c26a3d',
	analysisCycle: '#e07a5f',
	analysisCriticalPath: '#7ca3c4',
} as const;

export const LIGHT_COLORS = {
	bg: '#f5f4f0',
	surface: '#ffffff',
	surfaceHover: '#f0eeea',
	border: '#d4d0c8',
	borderSubtle: '#e8e5de',
	text: '#1a1918',
	textMuted: '#5c5a56',
	textDim: '#8a8780',
	accent: '#b8942f',
	accentHover: '#a6842a',
	accentMuted: 'rgba(184, 148, 47, 0.15)',
	danger: '#c44040',
	success: '#3d8a55',
	warning: '#b8942f',
	analysisCycle: '#e07a5f',
	analysisCriticalPath: '#7ca3c4',
} as const;

export function getThemeColors(theme: 'dark' | 'light') {
	return theme === 'light' ? LIGHT_COLORS : COLORS;
}
