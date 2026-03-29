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
        'background-opacity': 0.35,
        'border-width': 2.5,
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

    // Status is shown via DOM overlay dots (not border color)
    // Border color stays as TYPE color for visual consistency

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
        opacity: 0.75,
        'line-color': '#3d3a36',
        'target-arrow-color': '#3d3a36',
        'target-arrow-shape': 'triangle',
        'target-arrow-fill': 'filled',
        'curve-style': 'bezier',
        'control-point-step-size': 40,
        'arrow-scale': 1.3,
        'font-size': 11,
        'min-zoomed-font-size': 8,
        'text-rotation': 'autorotate',
        color: COLORS.textDim,
        'text-outline-color': COLORS.bg,
        'text-outline-width': 3,
        'text-background-color': COLORS.bg,
        'text-background-opacity': 0.85,
        'text-background-padding': '3px',
        'text-background-shape': 'roundrectangle',
        'text-margin-y': -8,
      },
    },
    // Edge hover
    {
      selector: 'edge:active',
      style: {
        opacity: 1,
        width: 2.5,
        'z-index': 999,
      } as Record<string, string | number>,
    },
    // Edge label — only when label data exists
    {
      selector: 'edge[label]',
      style: {
        label: 'data(label)',
      },
    },

    // Edge types — differentiated by color, width, style, and arrow
    {
      selector: 'edge[edgeType="depends_on"]',
      style: { 'line-color': EDGE_COLORS.depends_on, 'target-arrow-color': EDGE_COLORS.depends_on, width: 1.5 },
    },
    {
      selector: 'edge[edgeType="blocks"]',
      style: {
        'line-color': EDGE_COLORS.blocks,
        'target-arrow-color': EDGE_COLORS.blocks,
        'line-style': 'dashed',
        'line-dash-pattern': [8, 4],
        width: 2,
        'target-arrow-shape': 'tee',
      } as Record<string, any>,
    },
    {
      selector: 'edge[edgeType="triggers"]',
      style: {
        'line-color': EDGE_COLORS.triggers,
        'target-arrow-color': EDGE_COLORS.triggers,
        width: 1.5,
        'target-arrow-shape': 'triangle-backcurve',
      },
    },
    {
      selector: 'edge[edgeType="related"]',
      style: {
        'line-color': EDGE_COLORS.related,
        'target-arrow-color': EDGE_COLORS.related,
        'line-style': 'dotted',
        'line-dash-pattern': [3, 3],
        width: 1,
        opacity: 0.4,
        'arrow-scale': 0.9,
      } as Record<string, any>,
    },
    {
      selector: 'edge[edgeType="parent_child"]',
      style: {
        'line-color': EDGE_COLORS.parent_child,
        'target-arrow-color': EDGE_COLORS.parent_child,
        'line-style': 'dashed',
        'line-dash-pattern': [5, 5],
        width: 1,
        'target-arrow-shape': 'none',
      } as Record<string, any>,
    },

    // Selected edge — brass gold highlight
    {
      selector: 'edge.edge-selected',
      style: {
        width: 3,
        opacity: 1,
        'line-color': COLORS.accent,
        'target-arrow-color': COLORS.accent,
        'overlay-color': COLORS.accent,
        'overlay-opacity': 0.12,
        'overlay-padding': 4,
        'z-index': 999,
      } as Record<string, string | number>,
    },

    // Edge hover — connected nodes highlight
    {
      selector: 'node.edge-hover-connected',
      style: {
        'border-width': 2,
        'border-color': COLORS.accent,
        'border-opacity': 0.6,
      },
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

    // Segment-based routing for edges with waypoints
    {
      selector: 'edge[curveStyle="segments"]',
      style: {
        'curve-style': 'segments',
        'segment-distances': 'data(segmentDistances)',
        'segment-weights': 'data(segmentWeights)',
        'edge-distances': 'node-position',
      } as Record<string, any>,
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

    // Analysis Mode classes
    {
      selector: 'node.analysis-dimmed',
      style: {
        'opacity': 0.15,
        'overlay-opacity': 0,
      },
    },
    {
      selector: 'edge.analysis-dimmed',
      style: {
        'opacity': 0.08,
      },
    },
    {
      selector: 'node.cycle-node',
      style: {
        'border-color': '#e07a5f',
        'border-width': 3,
        'background-opacity': 0.15,
        'overlay-color': '#e07a5f',
        'overlay-opacity': 0.15,
      },
    },
    {
      selector: 'edge.cycle-edge',
      style: {
        'line-color': '#e07a5f',
        'target-arrow-color': '#e07a5f',
        'width': 2.5,
        'line-style': 'dashed',
        'opacity': 0.85,
      },
    },
    {
      selector: 'node.critical-path-node',
      style: {
        'border-color': '#7ca3c4',
        'border-width': 3,
        'background-opacity': 0.15,
        'overlay-color': '#7ca3c4',
        'overlay-opacity': 0.15,
      },
    },
    {
      selector: 'edge.critical-path-edge',
      style: {
        'line-color': '#7ca3c4',
        'target-arrow-color': '#7ca3c4',
        'width': 2.5,
        'opacity': 0.85,
      },
    },
  ];
}
