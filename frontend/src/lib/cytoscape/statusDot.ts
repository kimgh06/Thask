import type cytoscape from 'cytoscape';
import { STATUS_COLORS } from '$lib/constants';
import type { NodeStatus } from '$lib/types';

const DOT_SIZE = 10;

/**
 * Renders a colored status dot at the bottom-right corner of each non-GROUP node.
 * Updates on data change, pan, zoom.
 */
export function attachStatusDots(
	cy: cytoscape.Core,
): { update: () => void; cleanup: () => void } {
	let container: HTMLDivElement | null = null;
	let dots: Map<string, HTMLDivElement> = new Map();

	function ensureContainer(): HTMLDivElement | null {
		if (container) return container;
		const cyEl = cy.container();
		if (!cyEl) return null;
		container = document.createElement('div');
		container.style.cssText = 'position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:5;overflow:hidden;';
		cyEl.appendChild(container);
		return container;
	}

	function update() {
		const c = ensureContainer();
		if (!c) return;

		const half = DOT_SIZE / 2;
		const activeIds = new Set<string>();

		cy.nodes('[nodeType]').forEach((node: cytoscape.NodeSingular) => {
			const nodeType = node.data('nodeType') as string;
			if (nodeType === 'GROUP') return;
			if (node.hidden()) return;

			const status = node.data('status') as NodeStatus;
			if (!status || !STATUS_COLORS[status]) return;

			const id = node.id();
			activeIds.add(id);

			const rbb = node.renderedBoundingBox({ includeLabels: false, includeOverlays: false });
			const x = rbb.x2 - half - 2;
			const y = rbb.y2 - half - 2;

			let dot = dots.get(id);
			if (!dot) {
				dot = document.createElement('div');
				dot.style.cssText = `
					position:absolute;
					width:${DOT_SIZE}px;height:${DOT_SIZE}px;
					border-radius:50%;
					border:1.5px solid rgba(0,0,0,0.3);
					pointer-events:none;
				`;
				c.appendChild(dot);
				dots.set(id, dot);
			}

			dot.style.left = `${x}px`;
			dot.style.top = `${y}px`;
			dot.style.background = STATUS_COLORS[status];
			dot.style.display = '';
		});

		// Remove dots for nodes that no longer exist
		for (const [id, dot] of dots) {
			if (!activeIds.has(id)) {
				dot.remove();
				dots.delete(id);
			}
		}
	}

	// Update on relevant events
	cy.on('pan zoom', update);
	cy.on('add remove data position', 'node', update);

	// Initial render
	requestAnimationFrame(update);

	return {
		update,
		cleanup: () => {
			cy.off('pan zoom', update);
			cy.off('add remove data position', 'node', update);
			if (container) {
				container.remove();
				container = null;
			}
			dots.clear();
		},
	};
}
