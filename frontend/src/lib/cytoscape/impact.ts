import type { Core } from 'cytoscape';

export function activateImpactMode(
  cy: Core,
  changedNodeIds: string[],
  impactedNodeIds: string[],
) {
  const changedSet = new Set(changedNodeIds);
  const impactedSet = new Set(impactedNodeIds);
  const allAffected = new Set([...changedNodeIds, ...impactedNodeIds]);

  cy.batch(() => {
    // Dim everything first
    cy.nodes().addClass('impact-dimmed');
    cy.edges().addClass('impact-dimmed');

    // Highlight changed nodes
    cy.nodes().forEach((node) => {
      const id = node.id();
      if (changedSet.has(id)) {
        node.removeClass('impact-dimmed');
        node.addClass('impact-changed');
      } else if (impactedSet.has(id)) {
        node.removeClass('impact-dimmed');
        node.addClass('impact-affected');
      }
    });

    // Show edges between affected nodes
    cy.edges().forEach((edge) => {
      if (allAffected.has(edge.source().id()) && allAffected.has(edge.target().id())) {
        edge.removeClass('impact-dimmed');
      }
    });
  });
}

export function deactivateImpactMode(cy: Core) {
  cy.batch(() => {
    cy.elements().removeClass('impact-dimmed impact-changed impact-affected');
  });
}
