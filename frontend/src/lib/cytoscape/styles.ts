import type { StylesheetStyle } from 'cytoscape';
import { TYPE_COLORS, STATUS_COLORS, NODE_SHAPES } from '$lib/constants';

export function getGraphStyles(): StylesheetStyle[] {
  return [
    // Base node
    {
      selector: 'node',
      style: {
        label: 'data(label)',
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': 12,
        'font-weight': 600,
        'font-family': 'Inter, system-ui, sans-serif',
        'text-wrap': 'wrap',
        'text-max-width': '100px',
        color: '#e2e8f0',
        'text-outline-color': '#1e293b',
        'text-outline-width': 2,
        width: 72,
        height: 72,
        'border-width': 2,
        'border-color': '#475569',
        'background-color': '#1e293b',
        'overlay-padding': 4,
        'z-index': 10,
      },
    },

    // Node type shapes and colors
    ...Object.entries(NODE_SHAPES).map(([type, shape]) => ({
      selector: `node[nodeType="${type}"]`,
      style: {
        shape,
        'border-color': TYPE_COLORS[type as keyof typeof TYPE_COLORS] ?? '#475569',
        'background-color': TYPE_COLORS[type as keyof typeof TYPE_COLORS] ?? '#475569',
        'background-opacity': 0.125,
      } as Record<string, string | number>,
    })),

    // GROUP node — regular node with explicit dimensions (no compound parent)
    {
      selector: 'node[nodeType="GROUP"]',
      style: {
        width: 'data(width)',
        height: 'data(height)',
        shape: 'round-rectangle',
        'border-style': 'dashed',
        'border-color': '#475569',
        'border-width': 1.5,
        'background-opacity': 0.06,
        'background-color': '#94a3b8',
        'text-valign': 'top',
        'text-halign': 'center',
        'font-size': 13,
        'font-weight': 600,
        color: '#94a3b8',
        'z-index': 'data(depth)',
      } as Record<string, string | number>,
    },

    // Status colors (override border) — exclude GROUP nodes to keep their dashed style
    {
      selector: 'node[status="PASS"][nodeType!="GROUP"]',
      style: {
        'border-color': STATUS_COLORS.PASS,
        'background-color': STATUS_COLORS.PASS,
        'background-opacity': 0.094,
      },
    },
    {
      selector: 'node[status="FAIL"][nodeType!="GROUP"]',
      style: {
        'border-color': STATUS_COLORS.FAIL,
        'background-color': STATUS_COLORS.FAIL,
        'background-opacity': 0.094,
      },
    },
    {
      selector: 'node[status="IN_PROGRESS"][nodeType!="GROUP"]',
      style: {
        'border-color': STATUS_COLORS.IN_PROGRESS,
        'background-color': STATUS_COLORS.IN_PROGRESS,
        'background-opacity': 0.094,
      },
    },
    {
      selector: 'node[status="BLOCKED"][nodeType!="GROUP"]',
      style: {
        'border-color': STATUS_COLORS.BLOCKED,
        'background-color': STATUS_COLORS.BLOCKED,
        'background-opacity': 0.094,
      },
    },

    // Selected node — glow effect
    {
      selector: 'node:selected',
      style: {
        'border-width': 3,
        'border-color': '#818cf8',
        'overlay-color': '#818cf8',
        'overlay-opacity': 0.12,
        'overlay-padding': 6,
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
        'border-width': 3,
        'border-color': '#a78bfa',
        'overlay-color': '#a78bfa',
        'overlay-opacity': 0.1,
      },
    },

    // Drop target highlight when dragging a node over a GROUP
    {
      selector: 'node.drop-target',
      style: {
        'border-color': '#818cf8',
        'border-width': 2.5,
        'border-style': 'solid',
        'background-opacity': 0.12,
        'background-color': '#818cf8',
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
        width: 1.5,
        'line-color': '#475569',
        'target-arrow-color': '#475569',
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier',
        'arrow-scale': 1,
        'font-size': 10,
        'text-rotation': 'autorotate',
        color: '#64748b',
        'text-outline-color': '#0f172a',
        'text-outline-width': 2,
      },
    },
    // Edge label — only when label data exists
    {
      selector: 'edge[label]',
      style: {
        label: 'data(label)',
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

    // Edges connecting to GROUP nodes — snap to outline
    {
      selector: 'edge[?targetIsGroup]',
      style: {
        'target-endpoint': 'outside-to-node',
        'target-distance-from-node': 2,
      } as Record<string, string | number>,
    },
    {
      selector: 'edge[?sourceIsGroup]',
      style: {
        'source-endpoint': 'outside-to-node',
        'source-distance-from-node': 2,
      } as Record<string, string | number>,
    },

    // Edgehandles: ghost edge (follows cursor during drag)
    {
      selector: '.eh-ghost-edge',
      style: {
        'line-color': '#818cf8',
        'target-arrow-color': '#818cf8',
        'target-arrow-shape': 'triangle',
        'line-style': 'dashed',
        width: 1.5,
        'curve-style': 'bezier',
        opacity: 0.7,
        label: '',
      },
    },
    // Edgehandles: preview edge (snapped to valid target)
    {
      selector: 'edge.eh-preview',
      style: {
        'line-color': '#818cf8',
        'target-arrow-color': '#818cf8',
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
    // When edge targets GROUP interior: hide preview, show ghost instead
    {
      selector: 'edge.eh-preview.eh-group-interior',
      style: {
        opacity: 0,
      },
    },
    {
      selector: '.eh-ghost-edge.eh-group-interior',
      style: {
        opacity: 0.7,
      },
    },
    // Edgehandles: source node highlight
    {
      selector: '.eh-source',
      style: {
        'border-width': 3,
        'border-color': '#818cf8',
        'overlay-color': '#818cf8',
        'overlay-opacity': 0.1,
        'overlay-padding': 6,
      },
    },
    // Edgehandles: target node highlight
    {
      selector: '.eh-target',
      style: {
        'border-width': 3,
        'border-color': '#34d399',
        'overlay-color': '#34d399',
        'overlay-opacity': 0.1,
      },
    },
    // Suppress eh-target highlight on expanded GROUP (child gets custom highlight instead)
    {
      selector: 'node[nodeType="GROUP"].eh-target',
      style: {
        'border-color': '#475569',
        'border-width': 1.5,
        'overlay-opacity': 0,
      } as Record<string, string | number>,
    },
    // Custom target highlight for resolved child inside GROUP
    {
      selector: 'node.eh-target-resolved',
      style: {
        'border-width': 3,
        'border-color': '#34d399',
        'overlay-color': '#34d399',
        'overlay-opacity': 0.1,
      },
    },

    // Collapsed GROUP: compact with expand indicator
    {
      selector: 'node.group-collapsed',
      style: {
        width: 130,
        height: 44,
        padding: 0,
        'font-size': 11,
        'text-valign': 'center',
        'border-style': 'solid',
        'border-width': 1.5,
        'border-color': '#475569',
        'background-opacity': 0.15,
        'background-color': '#334155',
        color: '#94a3b8',
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

    // QA Impact Mode: highlighted (changed) — [status] raises specificity to override status colors
    {
      selector: 'node[status].impact-changed',
      style: {
        'border-width': 3,
        'border-color': '#fb923c',
        'background-color': '#fb923c',
        'background-opacity': 0.15,
        'overlay-color': '#fb923c',
        'overlay-opacity': 0.15,
        'overlay-padding': 8,
      },
    },

    // QA Impact Mode: impacted (connected)
    {
      selector: 'node[status].impact-affected',
      style: {
        'border-width': 2.5,
        'border-color': '#fdba74',
        'background-color': '#fdba74',
        'background-opacity': 0.1,
        'overlay-color': '#fdba74',
        'overlay-opacity': 0.1,
      },
    },

    // Status cascade flash (sky-blue, separate from impact mode orange)
    {
      selector: 'node[status].cascade-flash',
      style: {
        'border-width': 3,
        'border-color': '#38bdf8',
        'overlay-color': '#38bdf8',
        'overlay-opacity': 0.15,
        'overlay-padding': 6,
      },
    },

    // QA Impact Mode: FAIL/BUG nodes (red glow)
    {
      selector: 'node[status].impact-fail',
      style: {
        'border-width': 3,
        'border-color': '#ef4444',
        'background-color': '#ef4444',
        'background-opacity': 0.18,
        'overlay-color': '#ef4444',
        'overlay-opacity': 0.18,
        'overlay-padding': 8,
      },
    },

    // QA Impact Mode: impact path edges (un-dimmed + thicker)
    {
      selector: 'edge.impact-edge',
      style: {
        opacity: 0.85,
        width: 2,
      },
    },
  ];
}
