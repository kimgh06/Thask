<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import cytoscape from 'cytoscape';
	import fcose from 'cytoscape-fcose';
	import edgehandles from 'cytoscape-edgehandles';
	import { getGraphStyles } from '$lib/cytoscape/styles';
	import { getFcoseLayout } from '$lib/cytoscape/layouts';
	import { graphStore } from '$lib/stores/graph.svelte';
	import { themeStore } from '$lib/stores/theme.svelte';
	import { api } from '$lib/api';
	import type { GraphNode, GraphEdge, StatusChange } from '$lib/types';
	import { activateImpactMode, deactivateImpactMode } from '$lib/cytoscape/impact';
	import { attachResizeHandlers } from '$lib/cytoscape/resize';
	import { attachPortOverlay } from '$lib/cytoscape/portOverlay';
	import { syncElements as syncElementsCore } from '$lib/cytoscape/sync';
	import { attachGroupDragHandlers } from '$lib/cytoscape/handlers/groupDrag';
	import { initEdgehandles, attachEdgeCreationHandlers, isOnGroupBorder } from '$lib/cytoscape/handlers/edgeCreation';
	import { attachSelectionHandlers } from '$lib/cytoscape/handlers/selection';
	import { attachDynamicRouting } from '$lib/cytoscape/edgeRouter';
	import { attachEdgeBridgeOverlay } from '$lib/cytoscape/edgeBridgeOverlay';
	import { attachStatusDots } from '$lib/cytoscape/statusDot';

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
		onNodeTap?: (nodeId: string, position: { x: number; y: number }) => void;
		readonly?: boolean;
		apiBasePath?: string;
	}

	let { nodes, edges, projectId, onUpdateNodeParent, onZoomChange, onCreateEdge, onNodeTap = undefined, readonly = false, apiBasePath }: Props = $props();

	const resolvedApiBase = $derived(apiBasePath || (projectId ? `/api/projects/${projectId}` : ''));

	let ehInstance: { start: (n: cytoscape.NodeSingular) => void; stop: () => void } | null = null;

	let container: HTMLDivElement;
	let cy: cytoscape.Core | null = $state(null);
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
			typeFilter: graphStore.typeFilter,
			statusFilter: graphStore.statusFilter,
			initialLayoutDone,
			onUpdateNodeParent,
		});
	}

	async function savePositions() {
		if (!cy || readonly || !resolvedApiBase) return;
		const positions: Array<{ id: string; x: number; y: number; width?: number; height?: number }> = [];
		cy.nodes().forEach((n: cytoscape.NodeSingular) => {
			const pos = n.position();
			const entry: { id: string; x: number; y: number; width?: number; height?: number } = {
				id: n.id(), x: pos.x, y: pos.y,
			};
			const w = n.data('width') as number | undefined;
			const h = n.data('height') as number | undefined;
			if (w !== undefined) entry.width = w;
			if (h !== undefined) entry.height = h;
			positions.push(entry);
		});
		await api.patch(`${resolvedApiBase}/nodes/positions`, { positions });
	}

	export async function runLayout(algorithm: string = 'dagre') {
		if (!cy || cy.nodes().length === 0) return;
		if (readonly || !resolvedApiBase) {
			// Fallback to client-side layout for readonly/shared views
			runClientLayout();
			return;
		}
		try {
			const res = await api.post(`${resolvedApiBase}/graph/layout`, { algorithm });
			if (res.data) {
				const serverData = res.data as { nodes?: Array<{ id: string; positionX: number; positionY: number; width?: number; height?: number }> };
				const serverNodes = serverData.nodes || [];
				serverNodes.forEach((sn) => {
					const ele = cy!.getElementById(sn.id);
					if (ele.length) {
						ele.animate({
							position: { x: sn.positionX, y: sn.positionY },
							duration: 400,
							easing: 'ease-out-cubic' as any,
						});
						if (sn.width) ele.data('width', sn.width);
						if (sn.height) ele.data('height', sn.height);
					}
				});
				setTimeout(() => cy?.fit(undefined, 50), 450);
			}
		} catch {
			runClientLayout();
		}
	}

	function runClientLayout() {
		if (!cy) return;
		cy.layout(getFcoseLayout()).run();
		savePositions();
	}

	export function fitView() { cy?.fit(undefined, 50); }
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
		cy.stop();
		cy.animate({ center: { eles: node } }, { duration: 200 });
		cy.nodes().removeClass('search-highlight');
		node.addClass('search-highlight');
		trackTimeout(() => { if (node.inside()) node.removeClass('search-highlight'); }, 2000);
	}

	export function getCy(): cytoscape.Core | null { return cy; }
	export function getMousePosition(): { x: number; y: number } { return { ...lastMouseModelPos }; }

	export function animateCascade(changes: StatusChange[]) {
		if (!cy || changes.length === 0) return;
		changes.forEach((change, i) => {
			const node = cy!.getElementById(change.nodeId);
			if (!node.length) return;
			trackTimeout(() => {
				node.addClass('cascade-flash');
				trackTimeout(() => node.removeClass('cascade-flash'), 2000);
			}, i * 150);
		});
	}

	export function applyImpactClasses(changedIds: string[], affectedIds: string[], failIds: string[], edgeIds: string[]) {
		if (!cy) return;
		activateImpactMode(cy, changedIds, affectedIds, failIds, edgeIds);
	}

	export function clearImpactClasses() {
		if (!cy) return;
		deactivateImpactMode(cy);
	}

	export function applyAnalysisClasses(
		cycleNodeIds: string[],
		cycleEdgeIds: string[],
		criticalPathNodeIds: string[],
		criticalPathEdgeIds: string[],
	) {
		if (!cy) return;
		// Dim everything first
		cy.elements().addClass('analysis-dimmed');

		// Highlight cycle nodes/edges
		cycleNodeIds.forEach(id => {
			const node = cy!.getElementById(id);
			if (node.length) {
				node.removeClass('analysis-dimmed');
				node.addClass('cycle-node');
			}
		});
		cycleEdgeIds.forEach(id => {
			const edge = cy!.getElementById(id);
			if (edge.length) {
				edge.removeClass('analysis-dimmed');
				edge.addClass('cycle-edge');
			}
		});

		// Highlight critical path nodes/edges
		criticalPathNodeIds.forEach(id => {
			const node = cy!.getElementById(id);
			if (node.length) {
				node.removeClass('analysis-dimmed');
				node.addClass('critical-path-node');
			}
		});
		criticalPathEdgeIds.forEach(id => {
			const edge = cy!.getElementById(id);
			if (edge.length) {
				edge.removeClass('analysis-dimmed');
				edge.addClass('critical-path-edge');
			}
		});
	}

	export function clearAnalysisClasses() {
		if (!cy) return;
		cy.elements().removeClass('analysis-dimmed cycle-node cycle-edge critical-path-node critical-path-edge');
	}

	export function startEdgeDrawingFromNode(nodeId: string) {
		if (!cy || !ehInstance) return;
		const node = cy.getElementById(nodeId);
		if (!node.length) return;
		ehInstance.start(node);
	}

	onMount(() => {
		cy = cytoscape({
			container,
			style: getGraphStyles(themeStore.current),
			layout: { name: 'preset' },
			minZoom: 0.2,
			maxZoom: 4,
			boxSelectionEnabled: !readonly,
			selectionType: 'additive',
			autoungrabify: readonly,
			userPanningEnabled: true,
			userZoomingEnabled: true,
		});
		if (import.meta.env.DEV && typeof window !== 'undefined') (window as unknown as { __cy: unknown }).__cy = cy;

		const cleanups: Array<{ cleanup: () => void }> = [];

		// Selection handlers (always active — needed for detail panel in readonly too)
		const selectionHandlers = attachSelectionHandlers(cy, { onNodeTap });
		cleanups.push(selectionHandlers);

		// Status indicator dots (bottom-right corner of each node)
		const statusDots = attachStatusDots(cy);
		cleanups.push(statusDots);

		if (!readonly) {
			// Initialize edgehandles
			const eh = initEdgehandles(cy);
			ehInstance = eh as typeof ehInstance;

			// Resize handlers
			const cyContainer = cy.container()!;
			const resizeHandlers = attachResizeHandlers(cy, cyContainer, {
				savePositions,
				isEdgeDrawing: () => (eh as { active: boolean }).active,
			});
			cleanups.push(resizeHandlers);

			// Port overlay for edge creation
			const ehTyped = eh as { active: boolean; start: (n: cytoscape.NodeSingular) => void; stop: () => void };
			const portHandlers = attachPortOverlay(cy, portOverlay, ehTyped, {
				isResizing: () => resizeHandlers.isResizing(),
				isOnGroupBorder,
			});
			cleanups.push(portHandlers);

			// Group drag handlers
			const groupDragHandlers = attachGroupDragHandlers(cy, {
				getProjectId: () => projectId,
				onUpdateNodeParent,
				savePositions,
				isResizing: () => resizeHandlers.isResizing(),
				hidePortOverlay: () => { portOverlay.style.display = 'none'; },
				trackTimeout,
			});
			cleanups.push(groupDragHandlers);

			// Edge creation handlers
			const edgeCreationHandlers = attachEdgeCreationHandlers(cy, eh, {
				onCreateEdge,
				getMouseModelPos: () => lastMouseModelPos,
			});
			cleanups.push(edgeCreationHandlers);

			// Dynamic 8-direction edge routing
			const cleanupRouting = attachDynamicRouting(cy);
			cleanups.push({ cleanup: cleanupRouting });

			// Circuit-style bridge overlay for horizontal underpasses
			const cleanupBridgeOverlay = attachEdgeBridgeOverlay(cy);
			cleanups.push({ cleanup: cleanupBridgeOverlay });
		}

		// Zoom tracking
		cy.on('pan zoom', () => { onZoomChange?.(cy!.zoom()); });

		// Mouse position tracking
		cy.on('mousemove', (e) => { lastMouseModelPos = e.position; });

		// Initial data load
		syncElements();

		return () => {
			for (const h of cleanups) h.cleanup();
		};
	});

	onDestroy(() => {
		activeTimeouts.forEach(clearTimeout);
		activeTimeouts = [];
		cy?.destroy();
		cy = null;
	});

	// Re-apply Cytoscape styles when theme changes
	$effect(() => {
		const theme = themeStore.current;
		if (cy) {
			cy.style().clear().fromJson(getGraphStyles(theme)).update();
		}
	});

	// React to data changes after mount
	$effect(() => {
		const _nodes = nodes;
		const _edges = edges;
		const _collapsed = graphStore.collapsedGroups;
		const _typeFilter = graphStore.typeFilter;
		const _statusFilter = graphStore.statusFilter;
		if (!cy || _nodes === undefined || _edges === undefined) return;
		void _collapsed;
		void _typeFilter;
		void _statusFilter;
		syncElements();
	});

	let portOverlay: HTMLDivElement;
</script>

<div class="relative h-full w-full cytoscape-canvas-bg" style="min-height: 400px">
	<div bind:this={container} class="h-full w-full"></div>
	<!-- Port overlay for edge creation — 8 dots around hovered node -->
	<div
		bind:this={portOverlay}
		style="display: none; position: absolute; top: 0; left: 0; pointer-events: none;"
	>
		{#each ['port-top', 'port-top-right', 'port-right', 'port-bottom-right', 'port-bottom', 'port-bottom-left', 'port-left', 'port-top-left'] as cls}
			<div
				class="port-dot {cls}"
				style="
					position: absolute;
					width: 12px;
					height: 12px;
					border-radius: 50%;
					background: #c9a84c;
					border: 2px solid #a8893a;
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
