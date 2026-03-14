import type cytoscape from 'cytoscape';
import { getChildNodes } from './groupHelpers';

type ResizeZone = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw';

const ZONE_CURSORS: Record<ResizeZone, string> = {
	nw: 'nwse-resize',
	se: 'nwse-resize',
	ne: 'nesw-resize',
	sw: 'nesw-resize',
	n: 'ns-resize',
	s: 'ns-resize',
	e: 'ew-resize',
	w: 'ew-resize',
};

interface ResizeOptions {
	savePositions: () => void;
	isEdgeDrawing: () => boolean;
}

export function attachResizeHandlers(
	cy: cytoscape.Core,
	container: HTMLElement,
	options: ResizeOptions,
): { cleanup: () => void; isResizing: () => boolean } {
	let resizeTarget: cytoscape.NodeSingular | null = null;
	let ungrabifiedForResize: cytoscape.NodeSingular | null = null;
	let isResizing = false;
	let resizeZone: ResizeZone | null = null;
	let resizeStartPos = { x: 0, y: 0 };
	let resizeStartBB = { x1: 0, y1: 0, x2: 0, y2: 0 };

	function detectResizeZone(e: MouseEvent, node: cytoscape.NodeSingular): ResizeZone | null {
		if (node.hasClass('group-collapsed') || node.hasClass('filter-hidden')) return null;
		const bb = node.renderedBoundingBox({});
		const mx = e.offsetX;
		const my = e.offsetY;
		const T = 10;

		const inX = mx >= bb.x1 - T && mx <= bb.x2 + T;
		const inY = my >= bb.y1 - T && my <= bb.y2 + T;
		if (!inX || !inY) return null;

		const nearTop = Math.abs(my - bb.y1) <= T;
		const nearBottom = Math.abs(my - bb.y2) <= T;
		const nearLeft = Math.abs(mx - bb.x1) <= T;
		const nearRight = Math.abs(mx - bb.x2) <= T;

		if (nearTop && nearLeft) return 'nw';
		if (nearTop && nearRight) return 'ne';
		if (nearBottom && nearLeft) return 'sw';
		if (nearBottom && nearRight) return 'se';
		if (nearTop) return 'n';
		if (nearBottom) return 's';
		if (nearLeft) return 'w';
		if (nearRight) return 'e';
		return null;
	}

	function onResizeMouseDown(e: MouseEvent) {
		if (!resizeTarget || options.isEdgeDrawing()) return;
		const zone = detectResizeZone(e, resizeTarget);
		if (!zone) return;

		e.preventDefault();
		e.stopPropagation();
		isResizing = true;
		resizeZone = zone;
		container.style.cursor = ZONE_CURSORS[zone];
		cy.panningEnabled(false);
		cy.boxSelectionEnabled(false);
		resizeTarget.addClass('resizing');

		const bb = resizeTarget.boundingBox({ includeLabels: false, includeOverlays: false });
		resizeStartBB = { x1: bb.x1, y1: bb.y1, x2: bb.x2, y2: bb.y2 };
		resizeStartPos = { x: e.clientX, y: e.clientY };
	}

	function onResizeMouseMove(e: MouseEvent) {
		if (!isResizing) {
			if (options.isEdgeDrawing()) return;
			let foundTarget: cytoscape.NodeSingular | null = null;
			let foundZone: ResizeZone | null = null;
			let foundArea = Infinity;
			cy.nodes('[nodeType="GROUP"]').forEach((g) => {
				const zone = detectResizeZone(e, g);
				if (zone) {
					const bb = g.boundingBox({});
					const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
					if (area < foundArea) {
						foundTarget = g;
						foundZone = zone;
						foundArea = area;
					}
				}
			});

			if (foundTarget && foundZone) {
				const ft = foundTarget as cytoscape.NodeSingular;
				if (ungrabifiedForResize && ungrabifiedForResize.id() !== ft.id()) {
					ungrabifiedForResize.grabify();
					ungrabifiedForResize = null;
				}
				if (!ungrabifiedForResize) {
					ft.ungrabify();
					ungrabifiedForResize = ft;
				}
			} else if (ungrabifiedForResize) {
				ungrabifiedForResize.grabify();
				ungrabifiedForResize = null;
			}

			resizeTarget = foundTarget;
			container.style.cursor = foundZone ? ZONE_CURSORS[foundZone] : '';
			return;
		}
		if (!resizeTarget || !resizeZone) return;

		const zoom = cy.zoom();
		const deltaX = (e.clientX - resizeStartPos.x) / zoom;
		const deltaY = (e.clientY - resizeStartPos.y) / zoom;

		const moveLeft = resizeZone.includes('w');
		const moveRight = resizeZone.includes('e');
		const moveTop = resizeZone.includes('n');
		const moveBottom = resizeZone.includes('s');

		let { x1, y1, x2, y2 } = resizeStartBB;
		if (moveLeft) x1 += deltaX;
		if (moveRight) x2 += deltaX;
		if (moveTop) y1 += deltaY;
		if (moveBottom) y2 += deltaY;

		let minW = 80;
		let minH = 50;
		if (resizeTarget.data('nodeType') === 'GROUP' && !resizeTarget.hasClass('group-collapsed')) {
			const children = getChildNodes(cy, resizeTarget.id());
			if (children.length > 0) {
				const PAD = 30;
				let cxMin = Infinity,
					cyMin = Infinity,
					cxMax = -Infinity,
					cyMax = -Infinity;
				children.forEach((c: cytoscape.NodeSingular) => {
					const pos = c.position();
					const hw = c.width() / 2;
					const hh = c.height() / 2;
					cxMin = Math.min(cxMin, pos.x - hw);
					cyMin = Math.min(cyMin, pos.y - hh);
					cxMax = Math.max(cxMax, pos.x + hw);
					cyMax = Math.max(cyMax, pos.y + hh);
				});
				minW = Math.max(minW, cxMax - cxMin + PAD * 2);
				minH = Math.max(minH, cyMax - cyMin + PAD * 2);
			}
		}
		if (x2 - x1 < minW) {
			if (moveLeft) x1 = x2 - minW;
			else x2 = x1 + minW;
		}
		if (y2 - y1 < minH) {
			if (moveTop) y1 = y2 - minH;
			else y2 = y1 + minH;
		}

		const newW = x2 - x1;
		const newH = y2 - y1;
		const newCenterX = (x1 + x2) / 2;
		const newCenterY = (y1 + y2) / 2;

		resizeTarget.data('width', newW);
		resizeTarget.data('height', newH);
		resizeTarget.position({ x: newCenterX, y: newCenterY });
	}

	function onResizeEnd() {
		if (!isResizing || !resizeTarget) return;
		isResizing = false;
		resizeZone = null;
		container.style.cursor = '';
		cy.panningEnabled(true);
		cy.boxSelectionEnabled(true);
		resizeTarget.grabify();
		resizeTarget.removeClass('resizing');
		ungrabifiedForResize = null;
		options.savePositions();
	}

	container.addEventListener('mousedown', onResizeMouseDown);
	container.addEventListener('mousemove', onResizeMouseMove);
	container.addEventListener('mouseup', onResizeEnd);
	container.addEventListener('mouseleave', onResizeEnd);

	return {
		cleanup: () => {
			container.removeEventListener('mousedown', onResizeMouseDown);
			container.removeEventListener('mousemove', onResizeMouseMove);
			container.removeEventListener('mouseup', onResizeEnd);
			container.removeEventListener('mouseleave', onResizeEnd);
		},
		isResizing: () => isResizing,
	};
}
