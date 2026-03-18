import type cytoscape from 'cytoscape';
import { getDescendantNodes, getDescendantIdSet } from '$lib/cytoscape/groupHelpers';
import { undoStack } from '$lib/stores/undo.svelte';
import { moveNodesCmd, type NodePosition } from '$lib/commands/node';

export interface GroupDragOptions {
	getProjectId: () => string;
	onUpdateNodeParent?: (nodeId: string, parentId: string | null) => void;
	savePositions: () => void;
	isResizing: () => boolean;
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
	let dragTimeout: ReturnType<typeof setTimeout> | null = null;

	function onGrab(e: cytoscape.EventObject) {
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
	}

	function onDrag(evt: cytoscape.EventObject) {
		const node = evt.target as cytoscape.NodeSingular;
		const cursorPos = evt.position;

		let innerTarget: string | null = null;
		let innerTargetArea = Infinity;
		cy.nodes('[nodeType="GROUP"]').forEach((g) => {
			g.removeClass('drop-target');
			if (g.id() === node.id() || g.hasClass('group-collapsed') || dragDescendantIds?.has(g.id())) return;
			const bb = g.boundingBox({});
			if (
				cursorPos.x >= bb.x1 && cursorPos.x <= bb.x2 &&
				cursorPos.y >= bb.y1 && cursorPos.y <= bb.y2
			) {
				const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
				if (area < innerTargetArea) {
					innerTarget = g.id();
					innerTargetArea = area;
				}
			}
		});
		currentDropTarget = innerTarget;
		if (innerTarget) {
			cy.getElementById(innerTarget).addClass('drop-target');
		}

		if (groupDragState && groupDragState.groupId === node.id()) {
			const groupPos = node.position();
			groupDragState.childOffsets.forEach((offset, childId) => {
				const child = cy.getElementById(childId);
				if (child.length) {
					child.position({ x: groupPos.x + offset.dx, y: groupPos.y + offset.dy });
				}
			});
		}
	}

	function onDragfree(evt: cytoscape.EventObject) {
		cy.nodes('.drop-target').removeClass('drop-target');
		const node = evt.target as cytoscape.NodeSingular;

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
			node.data('parentId', newParentId);
			options.onUpdateNodeParent?.(node.id(), newParentId);
		}
		currentDropTarget = null;

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

		if (dragTimeout) clearTimeout(dragTimeout);
		dragTimeout = options.trackTimeout(() => options.savePositions(), 500);
	}

	cy.on('grab', 'node', onGrab);
	cy.on('drag', 'node', onDrag);
	cy.on('dragfree', 'node', onDragfree);

	return {
		cleanup: () => {
			if (dragTimeout) clearTimeout(dragTimeout);
			cy.off('grab', 'node', onGrab);
			cy.off('drag', 'node', onDrag);
			cy.off('dragfree', 'node', onDragfree);
		},
	};
}
