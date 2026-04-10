import type cytoscape from 'cytoscape';
import { getChildNodes } from './groupHelpers';

export function getFcoseLayout() {
  return {
    name: 'fcose' as const,
    quality: 'default' as const,
    randomize: false,
    animate: true,
    animationDuration: 400,
    animationEasing: 'ease-out-cubic',
    fit: true,
    padding: 50,
    nodeRepulsion: 8000,
    idealEdgeLength: 120,
    edgeElasticity: 0.45,
    gravity: 0.25,
    gravityRange: 3.8,
    numIter: 2500,
    tile: true,
  };
}

export function getPresetLayout() {
  return {
    name: 'preset' as const,
    fit: true,
    padding: 50,
  };
}

/** Calculate group dimensions based on child count. */
function calcGroupSize(childCount: number): { w: number; h: number } {
  const cols = Math.ceil(Math.sqrt(childCount));
  const rows = Math.ceil(childCount / cols);
  const cellW = 72 + 40;
  const cellH = 72 + 40;
  const padSide = 30;
  const padTop = 45;
  const padBot = 30;
  const w = Math.max(cols * cellW + padSide * 2, 160);
  const h = Math.max(rows * cellH + padTop + padBot, 100);
  return { w, h };
}

/** BFS topological layering for a set of nodes and edges between them. */
function bfsLayers(
  children: cytoscape.NodeSingular[],
  childEdges: cytoscape.EdgeCollection,
): cytoscape.NodeSingular[][] {
  const childIds = new Set(children.map((c) => c.id()));
  const inDegree = new Map<string, number>();
  const successors = new Map<string, string[]>();
  children.forEach((c) => {
    inDegree.set(c.id(), 0);
    successors.set(c.id(), []);
  });
  childEdges.forEach((e: cytoscape.EdgeSingular) => {
    const src = e.source().id();
    const tgt = e.target().id();
    if (childIds.has(src) && childIds.has(tgt)) {
      inDegree.set(tgt, (inDegree.get(tgt) ?? 0) + 1);
      successors.get(src)?.push(tgt);
    }
  });
  const layers: cytoscape.NodeSingular[][] = [];
  const nodeById = new Map(children.map((c) => [c.id(), c]));
  let queue = children.filter((c) => (inDegree.get(c.id()) ?? 0) === 0);
  while (queue.length > 0) {
    layers.push(queue);
    const next: cytoscape.NodeSingular[] = [];
    queue.forEach((node) => {
      (successors.get(node.id()) ?? []).forEach((succId) => {
        const deg = (inDegree.get(succId) ?? 1) - 1;
        inDegree.set(succId, deg);
        if (deg === 0) {
          const succNode = nodeById.get(succId);
          if (succNode) next.push(succNode);
        }
      });
    });
    queue = next;
  }
  // Handle cycles: remaining nodes go in last layer
  const placed = new Set(layers.flat().map((n) => n.id()));
  const remaining = children.filter((c) => !placed.has(c.id()));
  if (remaining.length > 0) layers.push(remaining);
  return layers;
}

/**
 * Run a group-aware layout:
 * 0. Pre-size GROUP nodes so fcose allocates correct space
 * 1. Layout top-level nodes with fcose
 * 2. Position children inside each group (edge-aware)
 */
export function runGroupAwareLayout(cy: cytoscape.Core, onComplete?: () => void) {
  // Collect top-level nodes (no parentId)
  const topLevel = cy.nodes().filter((n: cytoscape.NodeSingular) => !n.data('parentId'));

  if (topLevel.length === 0) {
    onComplete?.();
    return;
  }

  // Pass 0: pre-size GROUP nodes so fcose allocates correct space
  cy.nodes('[nodeType="GROUP"]').forEach((group: cytoscape.NodeSingular) => {
    if (group.hasClass('group-collapsed')) return;
    const children = getChildNodes(cy, group.id());
    if (children.length === 0) return;
    const { w, h } = calcGroupSize(children.length);
    group.style({ width: w, height: h });
    group.data('width', w);
    group.data('height', h);
  });

  // Also include edges between top-level nodes for layout
  const topLevelIds = new Set<string>();
  topLevel.forEach((n: cytoscape.NodeSingular) => { topLevelIds.add(n.id()); });
  const topEdges = cy.edges().filter((e: cytoscape.EdgeSingular) =>
    topLevelIds.has(e.source().id()) && topLevelIds.has(e.target().id()),
  );

  const layoutEles = topLevel.union(topEdges);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const layout = (layoutEles as any).layout({
    ...getFcoseLayout(),
    fit: false,
  });

  layout.on('layoutstop', () => {
    // Pass 2: position children inside each group
    arrangeChildrenInGroups(cy);
    cy.fit(undefined, 50);
    onComplete?.();
  });

  layout.run();
}

/** Arrange children inside their parent GROUP bounding box. Edge-aware with grid fallback. */
function arrangeChildrenInGroups(cy: cytoscape.Core) {
  const groups = cy.nodes('[nodeType="GROUP"]');

  // Process shallowest groups first (depth-order)
  const sorted = groups.sort((a: cytoscape.NodeSingular, b: cytoscape.NodeSingular) => {
    const dA = (a.data('depth') as number) ?? 0;
    const dB = (b.data('depth') as number) ?? 0;
    return dA - dB;
  });

  sorted.forEach((group: cytoscape.NodeSingular) => {
    if (group.hasClass('group-collapsed')) return;

    const children = getChildNodes(cy, group.id());
    if (children.length === 0) return;

    const gPos = group.position();
    const count = children.length;
    const cols = Math.ceil(Math.sqrt(count));
    const rows = Math.ceil(count / cols);

    // Fixed cell size based on node dimensions + spacing
    const cellW = 72 + 40;
    const cellH = 72 + 40;

    // Auto-size GROUP to fit children
    const padSide = 30;
    const padTop = 45; // title space
    const padBot = 30;
    const newW = Math.max(cols * cellW + padSide * 2, 160);
    const newH = Math.max(rows * cellH + padTop + padBot, 100);
    group.data('width', newW);
    group.data('height', newH);

    const innerW = newW - padSide * 2;
    const innerH = newH - padTop - padBot;

    // Top-left of inner area, offset for title
    const startX = gPos.x - innerW / 2 + cellW / 2;
    const startY = gPos.y - (newH / 2) + padTop + cellH / 2;

    // Check for edges between children
    const childIds = new Set(children.map((c: cytoscape.NodeSingular) => c.id()));
    const childEdges = cy.edges().filter((e: cytoscape.EdgeSingular) =>
      childIds.has(e.source().id()) && childIds.has(e.target().id()),
    );

    if (childEdges.length > 0) {
      // Edge-aware: arrange by BFS layers (left-to-right columns)
      const layers = bfsLayers(children.toArray(), childEdges);
      const numLayers = layers.length;
      const layerSpacingX = innerW / Math.max(numLayers, 1);
      layers.forEach((layer, layerIdx) => {
        const numInLayer = layer.length;
        const layerSpacingY = innerH / Math.max(numInLayer, 1);
        layer.forEach((node, nodeIdx) => {
          node.position({
            x: startX + layerIdx * layerSpacingX,
            y: startY - cellH / 2 + (nodeIdx + 0.5) * layerSpacingY,
          });
        });
      });
    } else {
      // No edges: fallback to grid layout
      let i = 0;
      children.forEach((child: cytoscape.NodeSingular) => {
        const col = i % cols;
        const row = Math.floor(i / cols);
        child.position({
          x: startX + col * cellW,
          y: startY + row * cellH,
        });
        i++;
      });
    }

    // Nested groups are handled by the outer sorted.forEach (depth-order)
  });
}
