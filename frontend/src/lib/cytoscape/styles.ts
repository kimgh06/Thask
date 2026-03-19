import type { StylesheetStyle } from 'cytoscape';
import { TYPE_COLORS, STATUS_COLORS, NODE_SHAPES, EDGE_COLORS, COLORS } from '$lib/constants';

export function getGraphStyles(): StylesheetStyle[] {
  return [
    // Base node
    {
      selector: 'node',
      style: {
        'text-valign': 'center',
        'text-halign': 'center',
        'font-size': 12,
        'font-weight': 600,
        'font-family': 'JetBrains Mono, monospace',
        'text-wrap': 'wrap',
        'text-max-width': '100px',
        color: COLORS.text,
        'text-outline-color': COLORS.bg,
        'text-outline-width': 2,
        width: 72,
        height: 72,
        'border-width': 2,
        'border-color': COLORS.border,
        'background-color': COLORS.surface,
        'overlay-padding': 4,
        'z-index': 10,
      },
    },

    // Node label — only when label data exists (avoids warning on edgehandles ghost nodes)
    {
      selector: 'node[label]',
      style: {
        label: 'data(label)',
      },
    },

    // Hide edgehandles ghost/internal nodes (no nodeType data)
    {
      selector: 'node[!nodeType]',
      style: {
        opacity: 0,
        width: 0.001,
        height: 0.001,
      },
    },

    // Node type shapes and colors
    ...Object.entries(NODE_SHAPES).map(([type, shape]) => ({
      selector: `node[nodeType="${type}"]`,
      style: {
        shape,
        'border-color': TYPE_COLORS[type as keyof typeof TYPE_COLORS] ?? COLORS.border,
        'background-color': TYPE_COLORS[type as keyof typeof TYPE_COLORS] ?? COLORS.border,
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
        'border-color': COLORS.border,
        'border-width': 1.5,
        'background-opacity': 0.06,
        'background-color': TYPE_COLORS.GROUP,
        'text-valign': 'top',
        'text-halign': 'center',
        'font-size': 13,
        'font-weight': 600,
        color: COLORS.textMuted,
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

    // Selected node — brass gold glow
    {
      selector: 'node:selected',
      style: {
        'border-width': 3,
        'border-color': COLORS.accent,
        'overlay-color': COLORS.accent,
        'overlay-opacity': 0.12,
        'overlay-padding': 6,
      },
    },

    // Search highlight
    {
      selector: 'node.search-highlight',
      style: {
        'border-width': 5,
        'border-color': COLORS.accent,
        'overlay-color': COLORS.accent,
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
        'border-color': COLORS.accent,
        'overlay-color': COLORS.accent,
        'overlay-opacity': 0.1,
      },
    },

    // Drop target highlight when dragging a node over a GROUP
    {
      selector: 'node.drop-target',
      style: {
        'border-color': COLORS.accent,
        'border-width': 2.5,
        'border-style': 'solid',
        'background-opacity': 0.12,
        'background-color': COLORS.accent,
      } as Record<string, string | number>,
    },

    // Active resizing feedback
    {
      selector: 'node.resizing',
      style: {
        'border-color': COLORS.accent,
        'border-width': 3,
        'border-style': 'solid',
      } as Record<string, string | number>,
    },

    // Base edge
    {
      selector: 'edge',
      style: {
        width: 1.5,
        'line-color': '#3d3a36',
        'target-arrow-color': '#3d3a36',
        'target-arrow-shape': 'triangle',
        'curve-style': 'bezier',
        'arrow-scale': 1,
        'font-size': 10,
        'text-rotation': 'autorotate',
        color: COLORS.textDim,
        'text-outline-color': COLORS.bg,
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
      style: { 'line-color': EDGE_COLORS.blocks, 'target-arrow-color': EDGE_COLORS.blocks, 'line-style': 'dashed' },
    },
    {
      selector: 'edge[edgeType="depends_on"]',
      style: { 'line-color': EDGE_COLORS.depends_on, 'target-arrow-color': EDGE_COLORS.depends_on },
    },
    {
      selector: 'edge[edgeType="triggers"]',
      style: { 'line-color': EDGE_COLORS.triggers, 'target-arrow-color': EDGE_COLORS.triggers },
    },
    {
      selector: 'edge[edgeType="related"]',
      style: { 'line-color': EDGE_COLORS.related, 'target-arrow-color': EDGE_COLORS.related },
    },
    {
      selector: 'edge[edgeType="parent_child"]',
      style: { 'line-color': EDGE_COLORS.parent_child, 'target-arrow-color': EDGE_COLORS.parent_child, 'line-style': 'dashed' },
    },

    // Selected edge — brass gold highlight
    {
      selector: 'edge.edge-selected',
      style: {
        width: 3,
        'line-color': COLORS.accent,
        'target-arrow-color': COLORS.accent,
        'overlay-color': COLORS.accent,
        'overlay-opacity': 0.12,
        'overlay-padding': 4,
        'z-index': 999,
      } as Record<string, string | number>,
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
        'line-color': COLORS.accent,
        'target-arrow-color': COLORS.accent,
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
        'line-color': COLORS.accent,
        'target-arrow-color': COLORS.accent,
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
        'border-color': COLORS.accent,
        'overlay-color': COLORS.accent,
        'overlay-opacity': 0.1,
        'overlay-padding': 6,
      },
    },
    // Edgehandles: target node highlight
    {
      selector: '.eh-target',
      style: {
        'border-width': 3,
        'border-color': COLORS.success,
        'overlay-color': COLORS.success,
        'overlay-opacity': 0.1,
      },
    },
    // Suppress eh-target highlight on expanded GROUP (child gets custom highlight instead)
    {
      selector: 'node[nodeType="GROUP"].eh-target',
      style: {
        'border-color': COLORS.border,
        'border-width': 1.5,
        'overlay-opacity': 0,
      } as Record<string, string | number>,
    },
    // Custom target highlight for resolved child inside GROUP
    {
      selector: 'node.eh-target-resolved',
      style: {
        'border-width': 3,
        'border-color': COLORS.success,
        'overlay-color': COLORS.success,
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
        'border-color': COLORS.border,
        'background-opacity': 0.15,
        'background-color': COLORS.surface,
        color: COLORS.textMuted,
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

    // QA Impact Mode: highlighted (changed) — brass gold
    {
      selector: 'node[status].impact-changed',
      style: {
        'border-width': 3,
        'border-color': COLORS.accent,
        'background-color': COLORS.accent,
        'background-opacity': 0.15,
        'overlay-color': COLORS.accent,
        'overlay-opacity': 0.15,
        'overlay-padding': 8,
      },
    },

    // QA Impact Mode: impacted (connected) — dimmer gold
    {
      selector: 'node[status].impact-affected',
      style: {
        'border-width': 2.5,
        'border-color': COLORS.accentHover,
        'background-color': COLORS.accentHover,
        'background-opacity': 0.1,
        'overlay-color': COLORS.accentHover,
        'overlay-opacity': 0.1,
      },
    },

    // Status cascade flash — steel blue
    {
      selector: 'node[status].cascade-flash',
      style: {
        'border-width': 3,
        'border-color': '#7ca3c4',
        'overlay-color': '#7ca3c4',
        'overlay-opacity': 0.15,
        'overlay-padding': 6,
      },
    },

    // QA Impact Mode: FAIL/BUG nodes (warm red glow)
    {
      selector: 'node[status].impact-fail',
      style: {
        'border-width': 3,
        'border-color': COLORS.danger,
        'background-color': COLORS.danger,
        'background-opacity': 0.18,
        'overlay-color': COLORS.danger,
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
