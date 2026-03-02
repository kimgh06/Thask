import type { StylesheetStyle } from 'cytoscape';

const STATUS_COLORS = {
  PASS: '#22c55e',
  FAIL: '#ef4444',
  IN_PROGRESS: '#eab308',
  BLOCKED: '#9ca3af',
};

const NODE_SHAPES: Record<string, string> = {
  FLOW: 'round-rectangle',
  BRANCH: 'diamond',
  TASK: 'rectangle',
  BUG: 'hexagon',
  API: 'barrel',
  UI: 'ellipse',
  GROUP: 'round-rectangle',
};

const NODE_COLORS: Record<string, string> = {
  FLOW: '#3b82f6',
  BRANCH: '#8b5cf6',
  TASK: '#06b6d4',
  BUG: '#ef4444',
  API: '#f97316',
  UI: '#10b981',
  GROUP: '#64748b',
};

export const cytoscapeStylesheet: StylesheetStyle[] = [
  // Base node
  {
    selector: 'node',
    style: {
      label: 'data(label)',
      'text-valign': 'center',
      'text-halign': 'center',
      'font-size': 13,
      'font-weight': 500,
      'font-family': 'Inter, system-ui, sans-serif',
      'text-wrap': 'wrap',
      'text-max-width': '110px',
      color: '#1f2937',
      'text-outline-color': '#ffffff',
      'text-outline-width': 2,
      width: 80,
      height: 80,
      'border-width': 3,
      'border-color': '#d1d5db',
      'background-color': '#f9fafb',
      'overlay-padding': 4,
      'z-index': 10,
    },
  },

  // Node type shapes and colors
  ...Object.entries(NODE_SHAPES).map(([type, shape]) => ({
    selector: `node[nodeType="${type}"]`,
    style: {
      shape,
      'border-color': NODE_COLORS[type] ?? '#d1d5db',
      'background-color': `${NODE_COLORS[type]}15`,
    } as Record<string, string>,
  })),

  // GROUP node — regular node with explicit dimensions (no compound parent)
  {
    selector: 'node[nodeType="GROUP"]',
    style: {
      width: 'data(width)',
      height: 'data(height)',
      shape: 'round-rectangle',
      'border-style': 'dashed',
      'border-color': '#94a3b8',
      'border-width': 2,
      'background-opacity': 0.08,
      'background-color': '#64748b',
      'text-valign': 'top',
      'text-halign': 'center',
      'font-size': 14,
      'font-weight': 600,
      'z-index': 0,
    } as Record<string, string | number>,
  },

  // Status colors (override border) — exclude GROUP nodes to keep their dashed style
  {
    selector: 'node[status="PASS"][nodeType!="GROUP"]',
    style: {
      'border-color': STATUS_COLORS.PASS,
      'background-color': '#f0fdf4',
    },
  },
  {
    selector: 'node[status="FAIL"][nodeType!="GROUP"]',
    style: {
      'border-color': STATUS_COLORS.FAIL,
      'background-color': '#fef2f2',
    },
  },
  {
    selector: 'node[status="IN_PROGRESS"][nodeType!="GROUP"]',
    style: {
      'border-color': STATUS_COLORS.IN_PROGRESS,
      'background-color': '#fefce8',
    },
  },
  {
    selector: 'node[status="BLOCKED"][nodeType!="GROUP"]',
    style: {
      'border-color': STATUS_COLORS.BLOCKED,
      'background-color': '#f3f4f6',
    },
  },

  // Selected node
  {
    selector: 'node:selected',
    style: {
      'border-width': 4,
      'border-color': '#3b82f6',
      'overlay-color': '#3b82f6',
      'overlay-opacity': 0.15,
    },
  },

  // Search highlight (zoom-to-node pulse)
  {
    selector: 'node.search-highlight',
    style: {
      'border-width': 5,
      'border-color': '#f59e0b',
      'overlay-color': '#f59e0b',
      'overlay-opacity': 0.25,
      'overlay-padding': 10,
      'z-index': 999,
    } as Record<string, string | number>,
  },

  // Multi-selected nodes
  {
    selector: 'node.multi-selected',
    style: {
      'border-width': 4,
      'border-color': '#6366f1',
      'overlay-color': '#6366f1',
      'overlay-opacity': 0.12,
    },
  },

  // Drop target highlight when dragging a node over a GROUP
  {
    selector: 'node.drop-target',
    style: {
      'border-color': '#3b82f6',
      'border-width': 3,
      'background-opacity': 0.15,
      'background-color': '#3b82f6',
    } as Record<string, string | number>,
  },

  // Active resizing feedback
  {
    selector: 'node.resizing',
    style: {
      'border-color': '#3b82f6',
      'border-width': 3,
      'border-style': 'solid',
    } as Record<string, string | number>,
  },

  // Base edge
  {
    selector: 'edge',
    style: {
      width: 2,
      'line-color': '#d1d5db',
      'target-arrow-color': '#d1d5db',
      'target-arrow-shape': 'triangle',
      'curve-style': 'bezier',
      'arrow-scale': 1.2,
      label: 'data(label)',
      'font-size': 10,
      'text-rotation': 'autorotate',
      color: '#9ca3af',
      'text-outline-color': '#ffffff',
      'text-outline-width': 1.5,
    },
  },

  // Edge types
  {
    selector: 'edge[edgeType="blocks"]',
    style: { 'line-color': '#ef4444', 'target-arrow-color': '#ef4444', 'line-style': 'dashed' },
  },
  {
    selector: 'edge[edgeType="depends_on"]',
    style: { 'line-color': '#f97316', 'target-arrow-color': '#f97316' },
  },
  {
    selector: 'edge[edgeType="triggers"]',
    style: { 'line-color': '#3b82f6', 'target-arrow-color': '#3b82f6' },
  },
  {
    selector: 'edge[edgeType="related"]',
    style: { 'line-color': '#6b7280', 'target-arrow-color': '#6b7280' },
  },
  {
    selector: 'edge[edgeType="parent_child"]',
    style: { 'line-color': '#8b5cf6', 'target-arrow-color': '#8b5cf6', 'line-style': 'dashed' },
  },

  // Custom hover handle dot for edge creation
  {
    selector: '.eh-custom-handle',
    style: {
      'background-color': '#3b82f6',
      width: 16,
      height: 16,
      shape: 'ellipse',
      'overlay-opacity': 0,
      'border-width': 3,
      'border-color': '#1d4ed8',
      label: '',
      'z-index': 999,
    } as Record<string, string | number>,
  },
  // Edgehandles: ghost edge (follows cursor during drag)
  {
    selector: '.eh-ghost-edge',
    style: {
      'line-color': '#93c5fd',
      'target-arrow-color': '#93c5fd',
      'target-arrow-shape': 'triangle',
      'line-style': 'dashed',
      width: 2,
      'curve-style': 'bezier',
      opacity: 0.8,
      label: '',
    },
  },
  // Edgehandles: preview edge (snapped to valid target)
  {
    selector: '.eh-preview',
    style: {
      'line-color': '#3b82f6',
      'target-arrow-color': '#3b82f6',
      'target-arrow-shape': 'triangle',
      'line-style': 'solid',
      width: 2,
      'curve-style': 'bezier',
      opacity: 0.9,
      label: '',
    },
  },
  // Edgehandles: hide ghost when preview is active (prevent duplicate lines)
  {
    selector: '.eh-ghost-edge.eh-preview-active',
    style: {
      opacity: 0,
    },
  },
  // Edgehandles: source node highlight
  {
    selector: '.eh-source',
    style: {
      'border-width': 4,
      'border-color': '#3b82f6',
      'overlay-color': '#3b82f6',
      'overlay-opacity': 0.12,
      'overlay-padding': 6,
    },
  },
  // Edgehandles: target node highlight
  {
    selector: '.eh-target',
    style: {
      'border-width': 4,
      'border-color': '#22c55e',
      'overlay-color': '#22c55e',
      'overlay-opacity': 0.12,
    },
  },

  // Collapsed GROUP: compact with expand indicator
  {
    selector: 'node.group-collapsed',
    style: {
      width: 130,
      height: 44,
      padding: 0,
      'font-size': 12,
      'text-valign': 'center',
      'border-style': 'solid',
      'border-width': 2,
      'border-color': '#94a3b8',
      'background-opacity': 0.2,
      'background-color': '#64748b',
    } as Record<string, string | number>,
  },

  // Filtered out by type/status filter
  {
    selector: 'node.filter-hidden',
    style: {
      display: 'none',
    },
  },

  // Children of collapsed group: hidden
  {
    selector: 'node.collapsed-child',
    style: {
      display: 'none',
    },
  },

  // Edges connected to collapsed children: hidden
  {
    selector: 'edge.collapsed-edge',
    style: {
      display: 'none',
    },
  },

  // QA Impact Mode: dimmed
  {
    selector: 'node.impact-dimmed',
    style: { opacity: 0.15 },
  },
  {
    selector: 'edge.impact-dimmed',
    style: { opacity: 0.08 },
  },

  // QA Impact Mode: highlighted (changed)
  {
    selector: 'node.impact-changed',
    style: {
      'border-width': 5,
      'border-color': '#f97316',
      'overlay-color': '#f97316',
      'overlay-opacity': 0.2,
      'overlay-padding': 8,
    },
  },

  // QA Impact Mode: impacted (connected)
  {
    selector: 'node.impact-affected',
    style: {
      'border-width': 4,
      'border-color': '#fb923c',
      'overlay-color': '#fb923c',
      'overlay-opacity': 0.12,
    },
  },
];
