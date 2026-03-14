import type cytoscape from 'cytoscape';
import { getChildNodes } from './groupHelpers';

export function getFcoseLayout() {
  return {
    name: 'fcose' as const,
    quality: 'default' as const,
    randomize: true,
    animate: false,
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

/**
 * Run a group-aware layout:
 * 1. Layout top-level nodes with fcose
 * 2. Position children inside each group
 */
export function runGroupAwareLayout(cy: cytoscape.Core, onComplete?: () => void) {
  // Collect top-level nodes (no parentId)
  const topLevel = cy.nodes().filter((n: cytoscape.NodeSingular) => !n.data('parentId'));

  if (topLevel.length === 0) {
    onComplete?.();
    return;
  }

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

/** Arrange children inside their parent GROUP bounding box in a grid. */
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
    const gW = (group.data('width') as number) ?? 160;
    const gH = (group.data('height') as number) ?? 100;
    const pad = 25;

    const innerW = gW - pad * 2;
    const innerH = gH - pad * 2;

    const count = children.length;
    const cols = Math.ceil(Math.sqrt(count));
    const rows = Math.ceil(count / cols);

    const cellW = innerW / cols;
    const cellH = innerH / rows;

    // Top-left of inner area
    const startX = gPos.x - innerW / 2 + cellW / 2;
    const startY = gPos.y - innerH / 2 + cellH / 2;

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

    // Nested groups are handled by the outer sorted.forEach (depth-order)
  });
}
