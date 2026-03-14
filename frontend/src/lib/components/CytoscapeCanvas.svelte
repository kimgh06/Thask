<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import cytoscape from 'cytoscape';
	import fcose from 'cytoscape-fcose';
	import edgehandles from 'cytoscape-edgehandles';
	import { getGraphStyles } from '$lib/cytoscape/styles';
	import { getFcoseLayout } from '$lib/cytoscape/layouts';
	import { getChildNodes, getDescendantNodes, getDescendantIdSet } from '$lib/cytoscape/groupHelpers';
	import { graphStore } from '$lib/stores/graph.svelte';
	import { api } from '$lib/api';
	import type { GraphNode, GraphEdge, StatusChange } from '$lib/types';
	import { undoStack } from '$lib/stores/undo.svelte';
	import { moveNodesCmd, type NodePosition } from '$lib/commands/node';
	import { activateImpactMode, deactivateImpactMode } from '$lib/cytoscape/impact';
	import { attachResizeHandlers } from '$lib/cytoscape/resize';
	import { attachPortOverlay } from '$lib/cytoscape/portOverlay';
	import { syncElements as syncElementsCore } from '$lib/cytoscape/sync';

	// Register extensions once at module level
	let extensionsRegistered = false;
	if (!extensionsRegistered) {
		cytoscape.use(fcose);
		cytoscape.use(edgehandles);
		extensionsRegistered = true;
	}

	interface Props {
		nodes: GraphNode[];
		edges: GraphEdge[];
		projectId: string;
		onUpdateNodeParent?: (nodeId: string, parentId: string | null) => void;
		onZoomChange?: (zoom: number) => void;
		onCreateEdge?: (sourceId: string, targetId: string) => void;
	}

	let { nodes, edges, projectId, onUpdateNodeParent, onZoomChange, onCreateEdge }: Props = $props();

	let container: HTMLDivElement;
	let cy: cytoscape.Core | null = $state(null);
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let eh: any = null;
	let initialLayoutDone = false;
	let activeTimeouts: ReturnType<typeof setTimeout>[] = [];
	let lastMouseModelPos = { x: 0, y: 0 };

	function trackTimeout(fn: () => void, ms: number): ReturnType<typeof setTimeout> {
		const id = setTimeout(() => {
			activeTimeouts = activeTimeouts.filter((t) => t !== id);
			fn();
		}, ms);
		activeTimeouts.push(id);
		return id;
	}

	function syncElements() {
		if (!cy) return;
		initialLayoutDone = syncElementsCore({
			cy,
			nodes,
			edges,
			collapsedGroups: [...graphStore.collapsedGroups],
			initialLayoutDone,
			onUpdateNodeParent,
		});
	}

	async function savePositions() {
		if (!cy) return;
		const positions: Array<{ id: string; x: number; y: number; width?: number; height?: number }> =
			[];
		cy.nodes().forEach((n: cytoscape.NodeSingular) => {
			const pos = n.position();
			const entry: { id: string; x: number; y: number; width?: number; height?: number } = {
				id: n.id(),
				x: pos.x,
				y: pos.y,
			};
			const w = n.data('width') as number | undefined;
			const h = n.data('height') as number | undefined;
			if (w !== undefined) entry.width = w;
			if (h !== undefined) entry.height = h;
			positions.push(entry);
		});
		await api.patch(`/api/projects/${projectId}/nodes/positions`, { positions });
	}

	export function runLayout() {
		if (!cy || cy.nodes().length === 0) return;
		cy.layout(getFcoseLayout()).run();
		trackTimeout(() => savePositions(), 1500);
	}

	export function fitView() {
		cy?.fit(undefined, 50);
	}

	export function zoomIn() {
		if (!cy) return;
		cy.zoom({ level: cy.zoom() * 1.2, renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } });
	}

	export function zoomOut() {
		if (!cy) return;
		cy.zoom({ level: cy.zoom() / 1.2, renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } });
	}

	export function focusNode(nodeId: string) {
		if (!cy) return;
		const node = cy.getElementById(nodeId);
		if (!node.length) return;
		cy.animate({ center: { eles: node }, zoom: 1.5 }, { duration: 400 });
		cy.nodes().removeClass('search-highlight');
		node.addClass('search-highlight');
		trackTimeout(() => { if (node.inside()) node.removeClass('search-highlight'); }, 2000);
	}

	export function getCy(): cytoscape.Core | null {
		return cy;
	}

	export function animateCascade(changes: StatusChange[]) {
		if (!cy || changes.length === 0) return;
		changes.forEach((change, i) => {
			const node = cy!.getElementById(change.nodeId);
			if (!node.length) return;
			trackTimeout(() => {
				node.addClass('impact-affected');
				trackTimeout(() => node.removeClass('impact-affected'), 2000);
			}, i * 150);
		});
	}

	export function applyImpactClasses(changedIds: string[], affectedIds: string[]) {
		if (!cy) return;
		activateImpactMode(cy, changedIds, affectedIds);
	}

	export function clearImpactClasses() {
		if (!cy) return;
		deactivateImpactMode(cy);
	}

	/** Check if a model-coordinate position is near the GROUP's border (not interior) */
	function isOnGroupBorder(pos: { x: number; y: number }, node: cytoscape.NodeSingular, threshold = 20): boolean {
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
	function resolveInnermostNode(node: cytoscape.NodeSingular, dropPos: { x: number; y: number }): cytoscape.NodeSingular {
		if (!cy || node.data('nodeType') !== 'GROUP') return node;
		// If drop is on GROUP border, keep the GROUP as target
		if (isOnGroupBorder(dropPos, node)) return node;

		const children = cy.nodes().filter(
			(n: cytoscape.NodeSingular) => n.data('parentId') === node.id(),
		);
		if (children.length === 0) return node;

		// Find child whose bounding box contains drop position (smallest = innermost)
		let best: cytoscape.NodeSingular | null = null;
		let bestArea = Infinity;
		children.forEach((c: cytoscape.NodeSingular) => {
			const bb = c.boundingBox({});
			if (dropPos.x >= bb.x1 && dropPos.x <= bb.x2 && dropPos.y >= bb.y1 && dropPos.y <= bb.y2) {
				const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
				if (area < bestArea) { bestArea = area; best = c; }
			}
		});
		return best ? resolveInnermostNode(best, dropPos) : node;
	}

	function syncMultiSelectClasses() {
		if (!cy) return;
		cy.nodes().removeClass('multi-selected');
		if (graphStore.selectedNodeIds.size > 1) {
			graphStore.selectedNodeIds.forEach((id) => {
				cy!.getElementById(id).addClass('multi-selected');
			});
		}
	}

	onMount(() => {
		cy = cytoscape({
			container,
			style: getGraphStyles(),
			layout: { name: 'preset' },
			minZoom: 0.2,
			maxZoom: 4,
			wheelSensitivity: 0.3,
		});

		// Initialize edgehandles
		eh = (cy as cytoscape.Core & { edgehandles: (opts: unknown) => unknown }).edgehandles({
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

		// Resize handlers (must be before port overlay — isResizing is used there)
		const cyContainer = cy.container()!;
		const resizeHandlers = attachResizeHandlers(cy, cyContainer, {
			savePositions,
			isEdgeDrawing: () => (eh as { active: boolean }).active,
		});
		const isResizing = () => resizeHandlers.isResizing();

		// Port overlay for edge creation
		const ehTyped = eh as { active: boolean; start: (n: cytoscape.NodeSingular) => void; stop: () => void };
		const portHandlers = attachPortOverlay(cy, portOverlay, ehTyped, {
			isResizing,
			isOnGroupBorder,
		});

		cy.on('pan zoom', () => {
			onZoomChange?.(cy!.zoom());
		});

		cy.on('ehstop ehcancel', () => {
			cy!.nodes('.eh-target-resolved').removeClass('eh-target-resolved');
			cy!.edges('.eh-group-interior').removeClass('eh-group-interior');
		});

		// GROUP drag state
		let groupDragState: {
			groupId: string;
			childOffsets: Map<string, { dx: number; dy: number }>;
		} | null = null;
		let dragDescendantIds: Set<string> | null = null;
		let preDragPositions: NodePosition[] = [];

		cy.on('grab', 'node', (e) => {
			const node = e.target as cytoscape.NodeSingular;
			if (isResizing()) return;
			if (portOverlay) portOverlay.style.display = 'none';

			// Capture pre-drag positions for undo
			if (node.data('nodeType') === 'GROUP') {
				const descendants = getDescendantNodes(cy!, node.id());
				preDragPositions = [node, ...descendants].map((n: cytoscape.NodeSingular) => ({
					id: n.id(), x: n.position().x, y: n.position().y,
				}));
			} else {
				preDragPositions = [{ id: node.id(), x: node.position().x, y: node.position().y }];
			}

			if (node.data('nodeType') === 'GROUP') {
				const groupPos = node.position();
				const descendants = getDescendantNodes(cy!, node.id());
				const childOffsets = new Map<string, { dx: number; dy: number }>();
				descendants.forEach((d: cytoscape.NodeSingular) => {
					const dPos = d.position();
					childOffsets.set(d.id(), {
						dx: dPos.x - groupPos.x,
						dy: dPos.y - groupPos.y,
					});
				});
				groupDragState = { groupId: node.id(), childOffsets };
				dragDescendantIds = getDescendantIdSet(cy!, node.id());
			} else {
				groupDragState = null;
				dragDescendantIds = null;
			}
		});

		cy.on('ehcomplete', (event, sourceNode, targetNode, addedEdge) => {
			(addedEdge as cytoscape.EdgeSingular).remove();
			// Use tracked mouse position instead of event.position (which may be snap target, not actual cursor)
			const dropPos = lastMouseModelPos;
			let source = sourceNode as cytoscape.NodeSingular;
			let target = targetNode as cytoscape.NodeSingular;

			// Resolve innermost node: if source/target is a GROUP, check if a child
			// node is at the drop position (user intended the child, not the GROUP)
			source = resolveInnermostNode(source, source.position());
			target = resolveInnermostNode(target, dropPos);

			if (source.id() !== target.id()) {
				onCreateEdge?.(source.id(), target.id());
			}
		});

		cy.on('tap', 'node', (evt: cytoscape.EventObject) => {
			const node = evt.target as cytoscape.NodeSingular;
			// GROUP interior tap: skip — let child nodes receive the event
			if (node.data('nodeType') === 'GROUP' && !node.hasClass('group-collapsed')) {
				if (!isOnGroupBorder(evt.position, node)) return;
			}
			const originalEvent = evt.originalEvent as MouseEvent;
			if (originalEvent.shiftKey || originalEvent.ctrlKey || originalEvent.metaKey) {
				graphStore.toggleNodeSelection(node.id());
			} else {
				graphStore.selectNode(node.id());
			}
			syncMultiSelectClasses();
		});

		cy.on('tap', (evt: cytoscape.EventObject) => {
			if (evt.target === cy) {
				graphStore.clearSelection();
				syncMultiSelectClasses();
			}
		});

		cy.on('tap', 'edge', (evt: cytoscape.EventObject) => {
			graphStore.selectEdge(evt.target.id());
			syncMultiSelectClasses();
		});

		cy.on('boxselect', 'node', () => {
			const selected = cy!.$(':selected');
			const ids: string[] = [];
			selected.forEach((ele) => { ids.push(ele.id()); });
			graphStore.selectNodes(ids);
			selected.unselect(); // clear cytoscape's built-in selection, we use our own class
			syncMultiSelectClasses();
		});

		cy.on('dbltap', 'node[nodeType="GROUP"]', (evt: cytoscape.EventObject) => {
			graphStore.toggleCollapsed(evt.target.id());
		});

		let currentDropTarget: string | null = null;

		cy.on('drag', 'node', (evt: cytoscape.EventObject) => {
			const node = evt.target as cytoscape.NodeSingular;
			const cursorPos = evt.position;

			let innerTarget: string | null = null;
			let innerTargetArea = Infinity;
			cy!.nodes('[nodeType="GROUP"]').forEach((g) => {
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
				cy!.getElementById(innerTarget).addClass('drop-target');
			}

			if (groupDragState && groupDragState.groupId === node.id()) {
				const groupPos = node.position();
				groupDragState.childOffsets.forEach((offset, childId) => {
					const child = cy!.getElementById(childId);
					if (child.length) {
						child.position({ x: groupPos.x + offset.dx, y: groupPos.y + offset.dy });
					}
				});
			}
		});

		let dragTimeout: ReturnType<typeof setTimeout> | null = null;
		cy.on('dragfree', 'node', (evt: cytoscape.EventObject) => {
			cy!.nodes('.drop-target').removeClass('drop-target');
			const node = evt.target as cytoscape.NodeSingular;

			// Drop on group: update parentId (with cycle prevention)
			const oldParentId = (node.data('parentId') as string | null) ?? null;
			let newParentId = currentDropTarget;

			// Prevent cycles: ensure the drop target is not a descendant of the dragged node
			if (newParentId && node.data('nodeType') === 'GROUP') {
				const descendants = getDescendantIdSet(cy!, node.id());
				if (descendants.has(newParentId) || newParentId === node.id()) {
					newParentId = null; // reject drop — would create cycle
				}
			}

			if (newParentId !== oldParentId) {
				node.data('parentId', newParentId);
				onUpdateNodeParent?.(node.id(), newParentId);
			}
			currentDropTarget = null;

			if (groupDragState && groupDragState.groupId === node.id()) {
				groupDragState = null;
			}
			dragDescendantIds = null;

			// Record move for undo
			if (preDragPositions.length > 0) {
				const newPositions: NodePosition[] = preDragPositions.map((p) => {
					const n = cy!.getElementById(p.id);
					return { id: p.id, x: n.position().x, y: n.position().y };
				});
				const hasMoved = preDragPositions.some((old, i) =>
					Math.abs(old.x - newPositions[i].x) > 1 || Math.abs(old.y - newPositions[i].y) > 1
				);
				if (hasMoved) {
					undoStack.record(moveNodesCmd(
						projectId,
						[...preDragPositions],
						newPositions,
						(positions) => {
							if (!cy) return;
							positions.forEach((p) => {
								cy!.getElementById(p.id).position({ x: p.x, y: p.y });
							});
						},
						() => savePositions(),
					));
				}
				preDragPositions = [];
			}

			if (dragTimeout) clearTimeout(dragTimeout);
			dragTimeout = trackTimeout(() => savePositions(), 500);
		});

		// Track actual mouse position in model coordinates for accurate edge targeting
		cy.on('mousemove', (e) => {
			lastMouseModelPos = e.position;

			// During edge drawing: highlight resolved child instead of GROUP
			if (!(eh as { active: boolean }).active) return;
			cy!.nodes('.eh-target-resolved').removeClass('eh-target-resolved');
			const ehTarget = cy!.nodes('.eh-target');
			if (ehTarget.length === 0) {
				cy!.edges('.eh-group-interior').removeClass('eh-group-interior');
				return;
			}
			const targetNode = ehTarget.first();
			if (targetNode.data('nodeType') !== 'GROUP' || targetNode.hasClass('group-collapsed')) {
				cy!.edges('.eh-group-interior').removeClass('eh-group-interior');
				return;
			}
			const resolved = resolveInnermostNode(targetNode, lastMouseModelPos);
			if (resolved.id() !== targetNode.id()) {
				resolved.addClass('eh-target-resolved');
				// Hide preview edge to GROUP, show ghost edge following cursor
				cy!.edges('.eh-ghost-edge, .eh-preview').addClass('eh-group-interior');
			} else {
				cy!.edges('.eh-group-interior').removeClass('eh-group-interior');
			}
		});

		// Initial data load
		syncElements();

		return () => {
			portHandlers.cleanup();
			resizeHandlers.cleanup();
		};
	});

	onDestroy(() => {
		activeTimeouts.forEach(clearTimeout);
		activeTimeouts = [];
		(eh as { destroy: () => void } | null)?.destroy();
		cy?.destroy();
		cy = null;
	});

	// React to data changes after mount
	$effect(() => {
		// Access reactive values to track them explicitly
		const _nodes = nodes;
		const _edges = edges;
		const _collapsed = graphStore.collapsedGroups;
		if (!cy || _nodes === undefined || _edges === undefined) return;
		void _collapsed;
		syncElements();
	});

	let portOverlay: HTMLDivElement;
</script>

<div class="relative h-full w-full cytoscape-canvas-bg" style="min-height: 400px">
	<div bind:this={container} class="h-full w-full"></div>
	<!-- Port overlay for edge creation — 4 dots around hovered node -->
	<div
		bind:this={portOverlay}
		style="display: none; position: absolute; top: 0; left: 0; pointer-events: none;"
	>
		{#each ['port-top', 'port-right', 'port-bottom', 'port-left'] as cls}
			<div
				class="port-dot {cls}"
				style="
					position: absolute;
					width: 18px;
					height: 18px;
					border-radius: 50%;
					background: #818cf8;
					border: 2px solid #6366f1;
					cursor: crosshair;
					pointer-events: auto;
					transition: transform 0.1s ease;
				"
				role="button"
				tabindex="-1"
				aria-label="Create edge from this port"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.transform = 'scale(1.3)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.transform = 'scale(1)'; }}
			></div>
		{/each}
	</div>
</div>
