import type cytoscape from 'cytoscape';
import { graphStore } from '$lib/stores/graph.svelte';

export interface SelectionOptions {
	onNodeTap?: (nodeId: string, position: { x: number; y: number }) => void;
	isSelectionSuspended?: () => boolean;
}

export function attachSelectionHandlers(
	cy: cytoscape.Core,
	options: SelectionOptions,
): { cleanup: () => void } {

	function syncMultiSelectClasses() {
		cy.nodes().removeClass('multi-selected');
		if (graphStore.selectedNodeIds.size >= 1) {
			graphStore.selectedNodeIds.forEach((id) => {
				cy.getElementById(id).addClass('multi-selected');
			});
		}
	}

	function onTapNode(evt: cytoscape.EventObject) {
		if (options.isSelectionSuspended?.()) return;
		const node = evt.target as cytoscape.NodeSingular;
		const originalEvent = evt.originalEvent as MouseEvent | undefined;
		if (originalEvent?.shiftKey || originalEvent?.ctrlKey || originalEvent?.metaKey) {
			graphStore.toggleNodeSelection(node.id());
		} else {
			graphStore.selectNode(node.id());
			if (originalEvent) {
				options.onNodeTap?.(node.id(), { x: originalEvent.clientX, y: originalEvent.clientY });
			}
		}
		cy.edges().removeClass('edge-selected');
		syncMultiSelectClasses();
	}

	function onTapBackground(evt: cytoscape.EventObject) {
		if (options.isSelectionSuspended?.()) return;
		if (evt.target === cy) {
			graphStore.clearSelection();
			cy.edges().removeClass('edge-selected');
			syncMultiSelectClasses();
		}
	}

	function onTapEdge(evt: cytoscape.EventObject) {
		if (options.isSelectionSuspended?.()) return;
		graphStore.selectEdge(evt.target.id());
		cy.edges().removeClass('edge-selected');
		evt.target.addClass('edge-selected');
		syncMultiSelectClasses();
	}

	function onBoxSelect() {
		if (options.isSelectionSuspended?.()) {
			cy.$(':selected').unselect();
			return;
		}
		const selected = cy.$(':selected');
		const ids: string[] = [];
		selected.forEach((ele) => { ids.push(ele.id()); });
		graphStore.selectNodes(ids);
		selected.unselect(); // clear cytoscape's built-in selection, we use our own class
		syncMultiSelectClasses();
	}

	function onDblTapGroup(evt: cytoscape.EventObject) {
		if (options.isSelectionSuspended?.()) return;
		graphStore.toggleCollapsed(evt.target.id());
	}

	// Edge hover — highlight connected nodes
	function onEdgeMouseOver(e: cytoscape.EventObject) {
		if (options.isSelectionSuspended?.()) return;
		const edge = e.target;
		edge.source().addClass('edge-hover-connected');
		edge.target().addClass('edge-hover-connected');
		edge.style({ opacity: 1, width: 2.5 });
	}
	function onEdgeMouseOut(e: cytoscape.EventObject) {
		if (options.isSelectionSuspended?.()) return;
		const edge = e.target;
		edge.source().removeClass('edge-hover-connected');
		edge.target().removeClass('edge-hover-connected');
		// Reset to base style (let stylesheet handle it)
		edge.removeStyle('opacity width');
	}

	cy.on('tap', 'node', onTapNode);
	cy.on('tap', onTapBackground);
	cy.on('tap', 'edge', onTapEdge);
	cy.on('boxselect', 'node', onBoxSelect);
	cy.on('dbltap', 'node[nodeType="GROUP"]', onDblTapGroup);
	cy.on('mouseover', 'edge', onEdgeMouseOver);
	cy.on('mouseout', 'edge', onEdgeMouseOut);

	return {
		cleanup: () => {
			cy.off('tap', 'node', onTapNode);
			cy.off('tap', onTapBackground);
			cy.off('tap', 'edge', onTapEdge);
			cy.off('boxselect', 'node', onBoxSelect);
			cy.off('dbltap', 'node[nodeType="GROUP"]', onDblTapGroup);
			cy.off('mouseover', 'edge', onEdgeMouseOver);
			cy.off('mouseout', 'edge', onEdgeMouseOut);
		},
	};
}
