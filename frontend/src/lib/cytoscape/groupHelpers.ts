import type cytoscape from 'cytoscape';

/**
 * Find direct children of a GROUP node by querying parentId data field.
 * Replaces compound API: node.children()
 */
export function getChildNodes(cy: cytoscape.Core, parentId: string): cytoscape.NodeCollection {
  return cy.nodes().filter((n: cytoscape.NodeSingular) => n.data('parentId') === parentId);
}

/**
 * Find all descendants of a GROUP node recursively by parentId data.
 * Replaces compound API: node.descendants()
 */
export function getDescendantNodes(cy: cytoscape.Core, groupId: string): cytoscape.NodeCollection {
  let result = cy.collection() as cytoscape.NodeCollection;
  const directChildren = getChildNodes(cy, groupId);
  result = result.union(directChildren);
  directChildren.forEach((child: cytoscape.NodeSingular) => {
    if (child.data('nodeType') === 'GROUP') {
      result = result.union(getDescendantNodes(cy, child.id()));
    }
  });
  return result;
}

/**
 * Collect descendant IDs as a Set (for cycle prevention during drag).
 */
export function getDescendantIdSet(cy: cytoscape.Core, groupId: string): Set<string> {
  const descendants = getDescendantNodes(cy, groupId);
  const ids = new Set<string>();
  descendants.forEach((d: cytoscape.NodeSingular) => { ids.add(d.id()); });
  return ids;
}
