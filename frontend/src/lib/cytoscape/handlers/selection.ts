import type cytoscape from 'cytoscape';
import { graphStore } from '$lib/stores/graph.svelte';

export interface SelectionOptions {
	onNodeTap?: (nodeId: string, position: { x: number; y: number }) => void;
}

export function attachSelectionHandlers(
	cy: cytoscape.Core,
	options: SelectionOptions,
): { cleanup: () => void } {

	function syncMultiSelectClasses() {
		cy.nodes().removeClass('multi-selected');
		if (graphStore.selectedNodeIds.size > 1) {
			graphStore.selectedNodeIds.forEach((id) => {
				cy.getElementById(id).addClass('multi-selected');
			});
		}
	}

	function onTapNode(evt: cytoscape.EventObject) {
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
		if (evt.target === cy) {
			graphStore.clearSelection();
			cy.edges().removeClass('edge-selected');
			syncMultiSelectClasses();
		}
	}

	function onTapEdge(evt: cytoscape.EventObject) {
		graphStore.selectEdge(evt.target.id());
		cy.edges().removeClass('edge-selected');
		evt.target.addClass('edge-selected');
		syncMultiSelectClasses();
	}

	function onBoxSelect() {
		const selected = cy.$(':selected');
		const ids: string[] = [];
		selected.forEach((ele) => { ids.push(ele.id()); });
		graphStore.selectNodes(ids);
		selected.unselect(); // clear cytoscape's built-in selection, we use our own class
		syncMultiSelectClasses();
	}

	function onDblTapGroup(evt: cytoscape.EventObject) {
		graphStore.toggleCollapsed(evt.target.id());
	}

	cy.on('tap', 'node', onTapNode);
	cy.on('tap', onTapBackground);
	cy.on('tap', 'edge', onTapEdge);
	cy.on('boxselect', 'node', onBoxSelect);
	cy.on('dbltap', 'node[nodeType="GROUP"]', onDblTapGroup);

	return {
		cleanup: () => {
			cy.off('tap', 'node', onTapNode);
			cy.off('tap', onTapBackground);
			cy.off('tap', 'edge', onTapEdge);
			cy.off('boxselect', 'node', onBoxSelect);
			cy.off('dbltap', 'node[nodeType="GROUP"]', onDblTapGroup);
		},
	};
}
