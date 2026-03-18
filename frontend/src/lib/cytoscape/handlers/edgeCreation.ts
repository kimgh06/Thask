import type cytoscape from 'cytoscape';

export interface EdgeCreationOptions {
	onCreateEdge?: (sourceId: string, targetId: string) => void;
	getMouseModelPos: () => { x: number; y: number };
}

/** Check if a model-coordinate position is near the GROUP's border (not interior) */
export function isOnGroupBorder(
	pos: { x: number; y: number },
	node: cytoscape.NodeSingular,
	threshold = 20,
): boolean {
	const bb = node.boundingBox({});
	const inX = pos.x >= bb.x1 && pos.x <= bb.x2;
	const inY = pos.y >= bb.y1 && pos.y <= bb.y2;
	if (!inX || !inY) return false;

	const nearLeft = Math.abs(pos.x - bb.x1) <= threshold;
	const nearRight = Math.abs(pos.x - bb.x2) <= threshold;
	const nearTop = Math.abs(pos.y - bb.y1) <= threshold;
	const nearBottom = Math.abs(pos.y - bb.y2) <= threshold;

	return nearLeft || nearRight || nearTop || nearBottom;
}

/** Resolve to innermost node at drop position — only pass through GROUP interior, not border */
function resolveInnermostNode(
	cy: cytoscape.Core,
	node: cytoscape.NodeSingular,
	dropPos: { x: number; y: number },
): cytoscape.NodeSingular {
	if (!cy || node.removed()) return node;
	if (node.data('nodeType') !== 'GROUP') return node;
	if (isOnGroupBorder(dropPos, node)) return node;

	const children = cy.nodes().filter(
		(n: cytoscape.NodeSingular) => n.data('parentId') === node.id(),
	);
	if (children.length === 0) return node;

	let best: cytoscape.NodeSingular | null = null;
	let bestArea = Infinity;
	children.forEach((c: cytoscape.NodeSingular) => {
		const bb = c.boundingBox({});
		if (dropPos.x >= bb.x1 && dropPos.x <= bb.x2 && dropPos.y >= bb.y1 && dropPos.y <= bb.y2) {
			const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
			if (area < bestArea) { bestArea = area; best = c; }
		}
	});
	return best ? resolveInnermostNode(cy, best, dropPos) : node;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Edgehandles = any;

export function initEdgehandles(cy: cytoscape.Core): Edgehandles {
	const eh = (cy as cytoscape.Core & { edgehandles: (opts: unknown) => unknown }).edgehandles({
		canConnect: (sourceNode: cytoscape.NodeSingular, targetNode: cytoscape.NodeSingular) =>
			!sourceNode.same(targetNode),
		edgeParams: () => ({}),
		hoverDelay: 0,
		snap: true,
		snapThreshold: 50,
		snapFrequency: 15,
		noEdgeEventsInDraw: true,
		disableBrowserGestures: true,
		handleNodes: 'DONOTMATCHANYTHING',
	});
	(eh as { enable: () => void }).enable();
	return eh;
}

export function attachEdgeCreationHandlers(
	cy: cytoscape.Core,
	eh: Edgehandles,
	options: EdgeCreationOptions,
): { cleanup: () => void } {

	function onEhComplete(
		_event: cytoscape.EventObject,
		sourceNode: cytoscape.NodeSingular,
		targetNode: cytoscape.NodeSingular,
		addedEdge: cytoscape.EdgeSingular,
	) {
		addedEdge.remove();
		const dropPos = options.getMouseModelPos();
		const source = resolveInnermostNode(cy, sourceNode, sourceNode.position());
		const target = resolveInnermostNode(cy, targetNode, dropPos);

		if (source.id() !== target.id()) {
			options.onCreateEdge?.(source.id(), target.id());
		}
	}

	function onMousemove(e: cytoscape.EventObject) {
		if (!(eh as { active: boolean }).active) return;
		cy.nodes('.eh-target-resolved').removeClass('eh-target-resolved');
		const ehTarget = cy.nodes('.eh-target');
		if (ehTarget.length === 0) {
			cy.edges('.eh-group-interior').removeClass('eh-group-interior');
			return;
		}
		const targetNode = ehTarget.first();
		if (targetNode.data('nodeType') !== 'GROUP' || targetNode.hasClass('group-collapsed')) {
			cy.edges('.eh-group-interior').removeClass('eh-group-interior');
			return;
		}
		const mousePos = options.getMouseModelPos();
		const resolved = resolveInnermostNode(cy, targetNode, mousePos);
		if (resolved.id() !== targetNode.id()) {
			resolved.addClass('eh-target-resolved');
			cy.edges('.eh-ghost-edge, .eh-preview').addClass('eh-group-interior');
		} else {
			cy.edges('.eh-group-interior').removeClass('eh-group-interior');
		}
	}

	function onEhCleanup() {
		cy.nodes('.eh-target-resolved').removeClass('eh-target-resolved');
		cy.edges('.eh-group-interior').removeClass('eh-group-interior');
	}

	cy.on('ehcomplete', onEhComplete as unknown as cytoscape.EventHandler);
	cy.on('mousemove', onMousemove);
	cy.on('ehstop ehcancel', onEhCleanup);

	return {
		cleanup: () => {
			cy.off('ehcomplete', onEhComplete as unknown as cytoscape.EventHandler);
			cy.off('mousemove', onMousemove);
			cy.off('ehstop ehcancel', onEhCleanup);
			(eh as { destroy?: () => void }).destroy?.();
		},
	};
}
