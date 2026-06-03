import type cytoscape from 'cytoscape';
import { STATUS_COLORS } from '$lib/constants';
import type { NodeStatus } from '$lib/types';

const DOT_SIZE = 10;
const VIEWPORT_PADDING = DOT_SIZE + 12;

/**
 * Renders a colored status dot at the bottom-right corner of each non-GROUP node.
 * Updates on data change, pan, zoom.
 */
export function attachStatusDots(
	cy: cytoscape.Core,
): { update: () => void; cleanup: () => void } {
	let container: HTMLDivElement | null = null;
	let dots: Map<string, HTMLDivElement> = new Map();
	let rafId: number | null = null;
	let pendingFullUpdate = false;
	let pendingNodeIds = new Set<string>();

	function ensureContainer(): HTMLDivElement | null {
		if (container) return container;
		const cyEl = cy.container();
		if (!cyEl) return null;
		container = document.createElement('div');
		container.style.cssText = 'position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:5;overflow:hidden;';
		cyEl.appendChild(container);
		return container;
	}

	function removeDot(id: string) {
		const dot = dots.get(id);
		if (!dot) return;
		dot.remove();
		dots.delete(id);
	}

	function renderedNodeBox(node: cytoscape.NodeSingular) {
		const zoom = cy.zoom();
		const pan = cy.pan();
		const pos = node.position();
		const halfW = node.outerWidth() / 2;
		const halfH = node.outerHeight() / 2;
		return {
			x1: (pos.x - halfW) * zoom + pan.x,
			y1: (pos.y - halfH) * zoom + pan.y,
			x2: (pos.x + halfW) * zoom + pan.x,
			y2: (pos.y + halfH) * zoom + pan.y,
		};
	}

	function isRenderedBoxVisible(box: { x1: number; y1: number; x2: number; y2: number }, host: HTMLElement): boolean {
		return (
			box.x2 >= -VIEWPORT_PADDING &&
			box.y2 >= -VIEWPORT_PADDING &&
			box.x1 <= host.clientWidth + VIEWPORT_PADDING &&
			box.y1 <= host.clientHeight + VIEWPORT_PADDING
		);
	}

	function updateNode(node: cytoscape.NodeSingular) {
		const c = ensureContainer();
		if (!c) return;

		const half = DOT_SIZE / 2;

		const id = node.id();
		const nodeType = node.data('nodeType') as string;
		const status = node.data('status') as NodeStatus;
		if (nodeType === 'GROUP' || node.hidden() || node.hasClass('filter-hidden') || !status || !STATUS_COLORS[status]) {
			removeDot(id);
			return;
		}

		const rbb = renderedNodeBox(node);
		if (!isRenderedBoxVisible(rbb, c)) {
			removeDot(id);
			return;
		}
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
	}

	function update() {
		const activeIds = new Set<string>();

		cy.nodes('[nodeType]').forEach((node: cytoscape.NodeSingular) => {
			activeIds.add(node.id());
			updateNode(node);
		});

		// Remove dots for nodes that no longer exist
		for (const [id, dot] of dots) {
			if (!activeIds.has(id)) {
				dot.remove();
				dots.delete(id);
			}
		}
	}

	function flushScheduledUpdate() {
		rafId = null;
		if (pendingFullUpdate) {
			pendingFullUpdate = false;
			pendingNodeIds.clear();
			update();
			return;
		}

		const ids = [...pendingNodeIds];
		pendingNodeIds.clear();
		ids.forEach((id) => {
			const node = cy.getElementById(id);
			if (node.length && node.isNode()) updateNode(node);
			else removeDot(id);
		});
	}

	function scheduleFlush() {
		if (rafId !== null) return;
		rafId = requestAnimationFrame(flushScheduledUpdate);
	}

	function scheduleFullUpdate() {
		pendingFullUpdate = true;
		scheduleFlush();
	}

	function scheduleNodeUpdate(evt: cytoscape.EventObject) {
		const node = evt.target as cytoscape.NodeSingular;
		pendingNodeIds.add(node.id());
		scheduleFlush();
	}

	// Update on relevant events
	cy.on('pan zoom', scheduleFullUpdate);
	cy.on('add data position', 'node', scheduleNodeUpdate);
	cy.on('remove', 'node', scheduleFullUpdate);

	// Initial render
	scheduleFullUpdate();

	return {
		update,
		cleanup: () => {
			cy.off('pan zoom', scheduleFullUpdate);
			cy.off('add data position', 'node', scheduleNodeUpdate);
			cy.off('remove', 'node', scheduleFullUpdate);
			if (rafId !== null) cancelAnimationFrame(rafId);
			if (container) {
				container.remove();
				container = null;
			}
			dots.clear();
		},
	};
}
