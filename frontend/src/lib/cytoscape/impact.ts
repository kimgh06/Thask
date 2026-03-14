import type { Core } from 'cytoscape';
import type { GraphEdge } from '$lib/types';

/**
 * Direction-aware BFS from a starting node.
 * Returns { impactedIds, impactEdgeIds } — excludes the start node itself.
 */
export function computeLocalImpact(
  edges: GraphEdge[],
  startId: string,
  depth: number = 2,
): { impactedIds: string[]; impactEdgeIds: string[] } {
  const impacted = new Map<string, boolean>([[startId, true]]);
  let frontier = [startId];
  const touchedEdgeIds = new Set<string>();

  for (let d = 0; d < depth && frontier.length > 0; d++) {
    const next: string[] = [];
    for (const edge of edges) {
      for (const fid of frontier) {
        let candidate = '';
        switch (edge.edgeType) {
          case 'blocks':
          case 'triggers':
            // forward: source changed → target affected
            if (edge.sourceId === fid) candidate = edge.targetId;
            break;
          case 'depends_on':
            // backward: A depends_on B, B changed → A affected
            if (edge.targetId === fid) candidate = edge.sourceId;
            break;
          case 'related':
            // bidirectional
            if (edge.sourceId === fid) candidate = edge.targetId;
            else if (edge.targetId === fid) candidate = edge.sourceId;
            break;
          case 'parent_child':
            continue; // structural, not causal
        }
        if (candidate && !impacted.has(candidate)) {
          impacted.set(candidate, true);
          next.push(candidate);
          touchedEdgeIds.add(edge.id);
        }
      }
    }
    frontier = next;
  }

  // Also include edges between any two nodes in the impact set
  for (const edge of edges) {
    if (impacted.has(edge.sourceId) && impacted.has(edge.targetId) && edge.edgeType !== 'parent_child') {
      touchedEdgeIds.add(edge.id);
    }
  }

  const impactedIds = [...impacted.keys()].filter((id) => id !== startId);
  return { impactedIds, impactEdgeIds: [...touchedEdgeIds] };
}

export function activateImpactMode(
  cy: Core,
  changedNodeIds: string[],
  impactedNodeIds: string[],
  failNodeIds: string[],
  impactEdgeIds: string[],
) {
  const changedSet = new Set(changedNodeIds);
  const impactedSet = new Set(impactedNodeIds);
  const failSet = new Set(failNodeIds);
  const edgeSet = new Set(impactEdgeIds);

  cy.batch(() => {
    // Clear all previous impact classes before re-applying
    cy.elements().removeClass('impact-dimmed impact-changed impact-affected impact-fail impact-edge');

    // Dim everything first
    cy.nodes().addClass('impact-dimmed');
    cy.edges().addClass('impact-dimmed');

    // Highlight nodes by category
    cy.nodes().forEach((node) => {
      const id = node.id();
      if (changedSet.has(id)) {
        node.removeClass('impact-dimmed');
        node.addClass('impact-changed');
      } else if (impactedSet.has(id)) {
        node.removeClass('impact-dimmed');
        node.addClass('impact-affected');
      }
      // FAIL/BUG nodes get red glow (additive — can combine with changed/affected)
      if (failSet.has(id)) {
        node.removeClass('impact-dimmed');
        node.addClass('impact-fail');
      }
    });

    // Highlight impact path edges
    cy.edges().forEach((edge) => {
      if (edgeSet.has(edge.id())) {
        edge.removeClass('impact-dimmed');
        edge.addClass('impact-edge');
      }
    });
  });
}

export function deactivateImpactMode(cy: Core) {
  cy.batch(() => {
    cy.elements().removeClass('impact-dimmed impact-changed impact-affected impact-fail impact-edge');
  });
}
