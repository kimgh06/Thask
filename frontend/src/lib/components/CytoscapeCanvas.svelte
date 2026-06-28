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
	import { applyDynamicRouting, attachDynamicRouting } from '$lib/cytoscape/edgeRouter';
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
		panMode?: boolean;
		apiBasePath?: string;
	}

	let {
		nodes,
		edges,
		projectId,
		onUpdateNodeParent,
		onZoomChange,
		onCreateEdge,
		onNodeTap = undefined,
		readonly = false,
		panMode = false,
		apiBasePath,
	}: Props = $props();

	const resolvedApiBase = $derived(apiBasePath || (projectId ? `/api/projects/${projectId}` : ''));
	const SERVER_LAYOUT_ANIMATION_NODE_LIMIT = 40;
	let autoLayoutRoutingSuspended = false;

	let ehInstance: { start: (n: cytoscape.NodeSingular) => void; stop: () => void } | null = null;

	let container: HTMLDivElement;
	let cy: cytoscape.Core | null = $state(null);
	let initialLayoutDone = false;
	let activeTimeouts: ReturnType<typeof setTimeout>[] = [];
	let lastMouseModelPos = { x: 0, y: 0 };
	let statusDotsHandle: { update: () => void } | null = null;

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
		revealPendingEdges();
		applyInteractionMode();
		statusDotsHandle?.update();
		cy.trigger('thask:filter');
	}

	function revealPendingEdges() {
		if (!cy) return;
		const pendingEdges = cy.edges('[?routePending]');
		if (pendingEdges.length === 0) return;
		const edgeIds = new Set(pendingEdges.map((edge) => edge.id()));
		applyDynamicRouting(cy, { useGrid: true, edgeIds });
		pendingEdges.data('routePending', false);
	}

	function finishAutoLayoutRouting(delayMs: number) {
		trackTimeout(() => {
			if (!cy) return;
			applyDynamicRouting(cy, { useGrid: true });
			cy.edges().data('routePending', false);
			autoLayoutRoutingSuspended = false;
			cy.fit(undefined, 50);
		}, delayMs);
	}

	function applyInteractionMode() {
		if (!cy) return;
		cy.boxSelectionEnabled(!readonly && !panMode);
		if (panMode) {
			cy.elements().addClass('pan-interaction-disabled');
			cy.elements().unselectify();
			cy.$(':selected').unselect();
		} else {
			cy.elements().removeClass('pan-interaction-disabled');
			cy.elements().selectify();
		}
		if (readonly || panMode) {
			cy.nodes().ungrabify();
			ehInstance?.stop();
			if (portOverlay) portOverlay.style.display = 'none';
		} else {
			cy.nodes().grabify();
		}
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
				const shouldAnimate = serverNodes.length <= SERVER_LAYOUT_ANIMATION_NODE_LIMIT;
				autoLayoutRoutingSuspended = true;
				cy.edges().data('routePending', true);
				if (shouldAnimate) {
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
					finishAutoLayoutRouting(450);
				} else {
					cy.batch(() => {
						serverNodes.forEach((sn) => {
							const ele = cy!.getElementById(sn.id);
							if (!ele.length) return;
							ele.position({ x: sn.positionX, y: sn.positionY });
							if (sn.width) ele.data('width', sn.width);
							if (sn.height) ele.data('height', sn.height);
						});
					});
					finishAutoLayoutRouting(80);
				}
			}
		} catch {
			autoLayoutRoutingSuspended = false;
			cy.edges().data('routePending', false);
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

	export function focusNode(nodeId: string, options: { zoom?: boolean } = {}) {
		if (!cy) return;
		const node = cy.getElementById(nodeId);
		if (!node.length) return;
		cy.stop();
		const animation: cytoscape.AnimationOptions = { center: { eles: node } };
		if (options.zoom) {
			animation.zoom = Math.min(Math.max(cy.zoom(), 1.15), cy.maxZoom());
		}
		cy.animate(animation, { duration: 220, easing: 'ease-out-cubic' });
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
		const selectionHandlers = attachSelectionHandlers(cy, {
			onNodeTap,
			isSelectionSuspended: () => panMode,
		});
		cleanups.push(selectionHandlers);

		// Status indicator dots (bottom-right corner of each node)
		const statusDots = attachStatusDots(cy);
		statusDotsHandle = statusDots;
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
				isInteractionSuspended: () => panMode,
			});
			cleanups.push(resizeHandlers);

			// Port overlay for edge creation
			const ehTyped = eh as { active: boolean; start: (n: cytoscape.NodeSingular) => void; stop: () => void };
			const portHandlers = attachPortOverlay(cy, portOverlay, ehTyped, {
				isResizing: () => resizeHandlers.isResizing(),
				isPanMode: () => panMode,
				isOnGroupBorder,
			});
			cleanups.push(portHandlers);

			// Group drag handlers
			const groupDragHandlers = attachGroupDragHandlers(cy, {
				getProjectId: () => projectId,
				onUpdateNodeParent,
				savePositions,
				isResizing: () => resizeHandlers.isResizing(),
				isInteractionSuspended: () => panMode,
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
		}

		// Visual renderers — active in readonly too so shared iframes match the editor
		const cleanupRouting = attachDynamicRouting(cy, {
			routeImmediately: false,
			isRoutingSuspended: () => autoLayoutRoutingSuspended,
		});
		cleanups.push({ cleanup: cleanupRouting });

		const cleanupBridgeOverlay = attachEdgeBridgeOverlay(cy);
		cleanups.push({ cleanup: cleanupBridgeOverlay });

		// Zoom tracking + dot-grid follows pan/zoom.
		// Pan = transform only (GPU-composited, no repaint). Grid is periodic, so we
		// wrap translate within one tile — visually identical, never exposes an edge.
		// backgroundSize (the only repaint) changes on zoom, which is far rarer than pan.
		let lastBgTile = -1;
		const syncBgGrid = () => {
			if (!cy) return;
			const z = cy.zoom(), p = cy.pan();
			const tile = 24 * z;
			if (tile !== lastBgTile) {
				lastBgTile = tile;
				bgEl.style.backgroundSize = `${tile}px ${tile}px`;
				const m = Math.ceil(tile) + 4;
				bgEl.style.inset = `${-m}px`;
			}
			const tx = (((p.x % tile) + tile) % tile) - tile;
			const ty = (((p.y % tile) + tile) % tile) - tile;
			bgEl.style.transform = `translate3d(${tx}px, ${ty}px, 0)`;
		};
		syncBgGrid();
		cy.on('pan zoom', () => { onZoomChange?.(cy!.zoom()); syncBgGrid(); });

		// Mouse position tracking
		cy.on('mousemove', (e) => { lastMouseModelPos = e.position; });

		// Initial data load
		syncElements();
		applyInteractionMode();

		return () => {
			for (const h of cleanups) h.cleanup();
			statusDotsHandle = null;
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

	$effect(() => {
		const _panMode = panMode;
		void _panMode;
		applyInteractionMode();
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
	let bgEl: HTMLDivElement;
</script>

<div class="relative h-full w-full overflow-hidden" style="min-height: 400px; background-color: var(--color-bg)">
	<!-- Dot grid: dedicated layer, moved via GPU transform on pan (no repaint) -->
	<div bind:this={bgEl} class="dot-grid-layer"></div>
	<div bind:this={container} class="relative h-full w-full" class:pan-mode={panMode}></div>
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

<style>
	.pan-mode,
	.pan-mode :global(*) {
		cursor: grab !important;
	}

	.pan-mode:active,
	.pan-mode:active :global(*) {
		cursor: grabbing !important;
	}
</style>
