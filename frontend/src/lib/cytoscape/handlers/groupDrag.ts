import type cytoscape from 'cytoscape';
import { getDescendantNodes, getDescendantIdSet } from '$lib/cytoscape/groupHelpers';
import { undoStack } from '$lib/stores/undo.svelte';
import { moveNodesCmd, type NodePosition } from '$lib/commands/node';

export interface GroupDragOptions {
	getProjectId: () => string;
	onUpdateNodeParent?: (nodeId: string, parentId: string | null) => void;
	savePositions: () => void;
	isResizing: () => boolean;
	isInteractionSuspended?: () => boolean;
	hidePortOverlay: () => void;
	trackTimeout: (fn: () => void, ms: number) => ReturnType<typeof setTimeout>;
}

export function attachGroupDragHandlers(
	cy: cytoscape.Core,
	options: GroupDragOptions,
): { cleanup: () => void } {
	let groupDragState: {
		groupId: string;
		childOffsets: Map<string, { dx: number; dy: number }>;
	} | null = null;
	let dragDescendantIds: Set<string> | null = null;
	let preDragPositions: NodePosition[] = [];
	let currentDropTarget: string | null = null;
	let previousDropTarget: string | null = null;
	let dragTimeout: ReturnType<typeof setTimeout> | null = null;
	let dropTargetRaf: number | null = null;
	let pendingCursorPos: { x: number; y: number } | null = null;
	let cachedDropGroups: Array<{ id: string; x1: number; y1: number; x2: number; y2: number; area: number }> = [];

	function clearDropTarget() {
		if (previousDropTarget) {
			cy.getElementById(previousDropTarget).removeClass('drop-target');
			previousDropTarget = null;
		}
		currentDropTarget = null;
	}

	function buildDropGroupCache(draggedNode: cytoscape.NodeSingular) {
		cachedDropGroups = [];
		cy.nodes('[nodeType="GROUP"]').forEach((g) => {
			if (g.id() === draggedNode.id() || g.hasClass('group-collapsed') || dragDescendantIds?.has(g.id())) return;
			const bb = g.boundingBox({});
			cachedDropGroups.push({
				id: g.id(),
				x1: bb.x1,
				y1: bb.y1,
				x2: bb.x2,
				y2: bb.y2,
				area: (bb.x2 - bb.x1) * (bb.y2 - bb.y1),
			});
		});
	}

	function computeDropTarget(cursorPos: { x: number; y: number }): string | null {
		let innerTarget: string | null = null;
		let innerTargetArea = Infinity;
		for (const g of cachedDropGroups) {
			if (
				cursorPos.x >= g.x1 && cursorPos.x <= g.x2 &&
				cursorPos.y >= g.y1 && cursorPos.y <= g.y2 &&
				g.area < innerTargetArea
			) {
				innerTarget = g.id;
				innerTargetArea = g.area;
			}
		}
		return innerTarget;
	}

	function applyDropTarget(nextTarget: string | null) {
		if (nextTarget === previousDropTarget) {
			currentDropTarget = nextTarget;
			return;
		}
		if (previousDropTarget) cy.getElementById(previousDropTarget).removeClass('drop-target');
		if (nextTarget) cy.getElementById(nextTarget).addClass('drop-target');
		previousDropTarget = nextTarget;
		currentDropTarget = nextTarget;
	}

	function scheduleDropTarget(cursorPos: { x: number; y: number }) {
		pendingCursorPos = cursorPos;
		if (dropTargetRaf !== null) return;
		dropTargetRaf = requestAnimationFrame(() => {
			dropTargetRaf = null;
			if (!pendingCursorPos) return;
			applyDropTarget(computeDropTarget(pendingCursorPos));
		});
	}

	function onGrab(e: cytoscape.EventObject) {
		if (options.isInteractionSuspended?.()) return;
		const node = e.target as cytoscape.NodeSingular;
		if (options.isResizing()) return;
		options.hidePortOverlay();

		// Capture pre-drag positions for undo
		if (node.data('nodeType') === 'GROUP') {
			const descendants = getDescendantNodes(cy, node.id());
			preDragPositions = [node, ...descendants].map((n: cytoscape.NodeSingular) => ({
				id: n.id(), x: n.position().x, y: n.position().y,
			}));
		} else {
			preDragPositions = [{ id: node.id(), x: node.position().x, y: node.position().y }];
		}

		if (node.data('nodeType') === 'GROUP') {
			const groupPos = node.position();
			const descendants = getDescendantNodes(cy, node.id());
			const childOffsets = new Map<string, { dx: number; dy: number }>();
			descendants.forEach((d: cytoscape.NodeSingular) => {
				const dPos = d.position();
				childOffsets.set(d.id(), {
					dx: dPos.x - groupPos.x,
					dy: dPos.y - groupPos.y,
				});
			});
			groupDragState = { groupId: node.id(), childOffsets };
			dragDescendantIds = getDescendantIdSet(cy, node.id());
		} else {
			groupDragState = null;
			dragDescendantIds = null;
		}
		buildDropGroupCache(node);
	}

	function onDrag(evt: cytoscape.EventObject) {
		if (options.isInteractionSuspended?.()) return;
		const node = evt.target as cytoscape.NodeSingular;
		const cursorPos = evt.position;

		scheduleDropTarget(cursorPos);

		if (groupDragState && groupDragState.groupId === node.id()) {
			const groupPos = node.position();
			cy.batch(() => {
				groupDragState?.childOffsets.forEach((offset, childId) => {
					const child = cy.getElementById(childId);
					if (child.length) {
						child.position({ x: groupPos.x + offset.dx, y: groupPos.y + offset.dy });
					}
				});
			});
		}
	}

	function onDragfree(evt: cytoscape.EventObject) {
		if (options.isInteractionSuspended?.()) return;
		if (dropTargetRaf !== null) {
			cancelAnimationFrame(dropTargetRaf);
			dropTargetRaf = null;
		}
		if (pendingCursorPos) applyDropTarget(computeDropTarget(pendingCursorPos));
		pendingCursorPos = null;
		const node = evt.target as cytoscape.NodeSingular;

		// Snap dropped position(s) to the routing grid (24px). Edge avoidance rasterises
		// obstacles onto this grid, so off-grid positions re-phase against it and routes wiggle
		// when a node/group later moves. Grid-aligned positions keep every move in-phase.
		const GRID = 24;
		const snapToGrid = (n: cytoscape.NodeSingular) => {
			const p = n.position();
			n.position({ x: Math.round(p.x / GRID) * GRID, y: Math.round(p.y / GRID) * GRID });
		};
		snapToGrid(node);
		if (node.data('nodeType') === 'GROUP') getDescendantNodes(cy, node.id()).forEach(snapToGrid);

		// Drop on group: update parentId (with cycle prevention)
		const oldParentId = (node.data('parentId') as string | null) ?? null;
		let newParentId = currentDropTarget;

		// Prevent cycles
		if (newParentId && node.data('nodeType') === 'GROUP') {
			const descendants = getDescendantIdSet(cy, node.id());
			if (descendants.has(newParentId) || newParentId === node.id()) {
				newParentId = null;
			}
		}

		if (newParentId !== oldParentId) {
			const pos = node.position();
			node.data('parentId', newParentId);
			node.position(pos);
			// Mark as recently dragged so syncElements won't reset position
			node.scratch('_lastDragTime', Date.now());
			// Save parent + positions concurrently (non-blocking)
			options.onUpdateNodeParent?.(node.id(), newParentId);
			requestAnimationFrame(() => options.savePositions());
		}
		clearDropTarget();
		cachedDropGroups = [];

		if (groupDragState && groupDragState.groupId === node.id()) {
			groupDragState = null;
		}
		dragDescendantIds = null;

		// Record move for undo
		if (preDragPositions.length > 0) {
			const newPositions: NodePosition[] = preDragPositions.map((p) => {
				const n = cy.getElementById(p.id);
				return { id: p.id, x: n.position().x, y: n.position().y };
			});
			const hasMoved = preDragPositions.some((old, i) =>
				Math.abs(old.x - newPositions[i].x) > 1 || Math.abs(old.y - newPositions[i].y) > 1
			);
			if (hasMoved) {
				undoStack.record(moveNodesCmd(
					options.getProjectId(),
					[...preDragPositions],
					newPositions,
					(positions) => {
						positions.forEach((p) => {
							cy.getElementById(p.id).position({ x: p.x, y: p.y });
						});
					},
					() => options.savePositions(),
				));
			}
			preDragPositions = [];
		}

		// Mark node as recently dragged (prevents syncElements from resetting position)
		node.scratch('_lastDragTime', Date.now());

		if (dragTimeout) clearTimeout(dragTimeout);
		dragTimeout = options.trackTimeout(() => options.savePositions(), 500);
	}

	cy.on('grab', 'node', onGrab);
	cy.on('drag', 'node', onDrag);
	cy.on('dragfree', 'node', onDragfree);

	return {
		cleanup: () => {
			if (dragTimeout) clearTimeout(dragTimeout);
			if (dropTargetRaf !== null) cancelAnimationFrame(dropTargetRaf);
			clearDropTarget();
			cy.off('grab', 'node', onGrab);
			cy.off('drag', 'node', onDrag);
			cy.off('dragfree', 'node', onDragfree);
		},
	};
}
