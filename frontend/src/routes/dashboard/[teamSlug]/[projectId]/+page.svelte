<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { graphStore } from '$lib/stores/graph.svelte';
	import { undoStack } from '$lib/stores/undo.svelte';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import GraphToolbar from '$lib/components/GraphToolbar.svelte';
	import DetailSidePanel from '$lib/components/DetailSidePanel.svelte';
	import { createNodeCrud } from '$lib/managers/nodeCrud.svelte';
	import { createEdgeCrud } from '$lib/managers/edgeCrud.svelte';
	import type { GraphNode, GraphEdge, GraphData, NodeDetail } from '$lib/types';
	import { computeLocalImpact } from '$lib/cytoscape/impact';
	import { createKeydownHandler } from '$lib/shortcuts';
	import { exportPNG, exportJSON, importJSON } from '$lib/export';
	import { realtimeStore } from '$lib/stores/realtime.svelte';
	import ShareDialog from '$lib/components/ShareDialog.svelte';
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';


	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let loading = $state(true);
	let loadError = $state('');
	let showShareDialog = $state(false);
	let projectName = $state('');
	let panelCollapsed = $state(false);

	const projectId = $derived(page.params.projectId ?? '');

	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);
	let sidePanel = $state<ReturnType<typeof DetailSidePanel> | undefined>(undefined);

	// Node detail state
	let selectedNodeDetail = $state<NodeDetail | null>(null);
	let detailLoading = $state(false);
	let detailRequestId = 0;

	// Edge state
	let selectedEdge = $state<GraphEdge | null>(null);

	// Zoom level for status bar
	let zoomLevel = $state(1);

	// Panel mode derived from selection state
	let panelMode = $derived.by(() => {
		if (graphStore.selectedNodeIds.size > 1) return 'multi-select' as const;
		if (graphStore.selectedNodeId) return 'node' as const;
		if (graphStore.selectedEdgeId) return 'edge' as const;
		return 'empty' as const;
	});

	// Selected nodes for multi-select view
	let selectedNodes = $derived(
		nodes.filter((n) => graphStore.selectedNodeIds.has(n.id)),
	);

	// Mutation context for undo commands
	const mutCtx = {
		get projectId() { return projectId; },
		getNodes: () => nodes,
		setNodes: (v: GraphNode[]) => { nodes = v; },
		getEdges: () => edges,
		setEdges: (v: GraphEdge[]) => { edges = v; },
	};

	// CRUD managers
	const nodeCrud = createNodeCrud({
		getProjectId: () => projectId,
		getMutCtx: () => mutCtx,
		getNodes: () => nodes,
		setNodes: (v) => { nodes = v; },
		getEdges: () => edges,
		getSelectedNodeDetail: () => selectedNodeDetail,
		setSelectedNodeDetail: (d) => { selectedNodeDetail = d; },
		getCanvas: () => canvas,
	});

	const edgeCrud = createEdgeCrud({
		getProjectId: () => projectId,
		getMutCtx: () => mutCtx,
		getEdges: () => edges,
		setEdges: (v) => { edges = v; },
		getSelectedEdge: () => selectedEdge,
		setSelectedEdge: (e) => { selectedEdge = e; },
	});

	// Load graph data
	function loadGraph() {
		const pid = projectId;
		if (!pid) return;
		api.get<GraphData>(`/api/projects/${pid}/graph`).then((res) => {
			if (projectId !== pid) return;
			if (res.error || !res.data) {
				loadError = res.error || 'Failed to load graph data.';
				loading = false;
				return;
			}
			nodes = res.data.nodes ?? [];
			edges = res.data.edges ?? [];
			loading = false;
		});
	}

	$effect(() => {
		if (!projectId) return;
		loading = true;
		loadError = '';
		// Fetch project name
		api.get<{ name: string }>(`/api/projects/${projectId}`).then((res) => {
			if (res.data) projectName = res.data.name;
		});
		loadGraph();

		// Real-time sync
		realtimeStore.connect(projectId, () => {
			loadGraph();
			sidePanel?.refreshActivity();
		});
		return () => realtimeStore.disconnect();
	});

	// React to node selection
	$effect(() => {
		const nodeId = graphStore.selectedNodeId;
		if (nodeId) {
			fetchNodeDetail(nodeId);
		} else {
			selectedNodeDetail = null;
		}
	});

	// React to edge selection
	$effect(() => {
		const edgeId = graphStore.selectedEdgeId;
		if (edgeId) {
			selectedEdge = edges.find((e) => e.id === edgeId) ?? null;
		} else {
			selectedEdge = null;
		}
	});

	// Impact mode
	$effect(() => {
		const active = graphStore.impactMode;
		const selectedId = graphStore.selectedNodeId;
		if (active && selectedId) {
			const { impactedIds, impactEdgeIds } = computeLocalImpact(edges, selectedId);
			const failIds = nodes.filter((n) => n.status === 'FAIL' || n.type === 'BUG').map((n) => n.id);
			canvas?.applyImpactClasses([selectedId], impactedIds, failIds, impactEdgeIds);
		} else {
			canvas?.clearImpactClasses();
		}
	});

	// Analysis mode
	$effect(() => {
		const active = graphStore.analysisMode;
		const pid = projectId;
		if (!active) {
			canvas?.clearAnalysisClasses();
			return;
		}
		api.get<{ cycles: string[][]; criticalPath: string[]; skippedCycleNodes: string[] }>(`/api/projects/${pid}/graph/analyze`).then((res) => {
			if (!graphStore.analysisMode || projectId !== pid) return;
			const cycles: string[][] = res.data?.cycles ?? [];
			const criticalPath: string[] = res.data?.criticalPath ?? [];
			const skippedCycleNodes: string[] = res.data?.skippedCycleNodes ?? [];

			if (skippedCycleNodes.length > 0) {
				console.warn(`[Thask] ${skippedCycleNodes.length} nodes excluded from critical path due to cycles`);
			}

			const cycleNodeIds = [...new Set(cycles.flat())];
			const cycleEdgeIds: string[] = [];
			cycles.forEach((cycle) => {
				for (let i = 0; i < cycle.length; i++) {
					const src = cycle[i];
					const tgt = cycle[(i + 1) % cycle.length];
					const match = edges.find((e) => e.sourceId === src && e.targetId === tgt);
					if (match) cycleEdgeIds.push(match.id);
				}
			});

			const criticalPathEdgeIds: string[] = [];
			for (let i = 0; i < criticalPath.length - 1; i++) {
				const src = criticalPath[i];
				const tgt = criticalPath[i + 1];
				const match = edges.find((e) => e.sourceId === src && e.targetId === tgt);
				if (match) criticalPathEdgeIds.push(match.id);
			}

			canvas?.applyAnalysisClasses(cycleNodeIds, cycleEdgeIds, criticalPath, criticalPathEdgeIds);
		}).catch(() => {
			graphStore.analysisMode = false;
			canvas?.clearAnalysisClasses();
		});
	});

	async function fetchNodeDetail(nodeId: string) {
		const requestId = ++detailRequestId;
		detailLoading = true;
		const res = await api.get<NodeDetail>(`/api/projects/${projectId}/nodes/${nodeId}`);
		if (requestId !== detailRequestId) return;
		if (res.data && graphStore.selectedNodeId === nodeId) {
			selectedNodeDetail = res.data;
		}
		detailLoading = false;
	}

	function handleSelectNodeFromPanel(nodeId: string) {
		graphStore.selectNode(nodeId);
		canvas?.focusNode(nodeId);
	}

	function handleStartEdgeDrawing(nodeId: string) {
		canvas?.startEdgeDrawingFromNode(nodeId);
	}

	function handleExportPNG() {
		const cy = canvas?.getCy();
		if (cy) exportPNG(cy);
	}

	function handleExportJSON() {
		exportJSON(nodes, edges);
	}

	async function handleImport(mode: 'replace' | 'merge') {
		const result = await importJSON(projectId, mode, mode === 'merge' ? nodes : undefined);
		if (result) {
			nodes = result.nodes;
			edges = result.edges;
			graphStore.clearSelection();
		}
	}

	const handleKeydown = createKeydownHandler({
		deleteSelection: () => {
			if (graphStore.selectedNodeIds.size > 1) nodeCrud.handleBatchDelete();
			else if (graphStore.selectedNodeId) nodeCrud.handleDeleteNode(graphStore.selectedNodeId);
			else if (graphStore.selectedEdgeId) edgeCrud.handleDeleteEdge();
		},
		escape: () => {
			if (selectedEdge || graphStore.selectedNodeId || graphStore.selectedNodeIds.size > 0) graphStore.clearSelection();
		},
		undo: () => undoStack.undo(),
		redo: () => undoStack.redo(),
		selectAll: () => graphStore.selectNodes(nodes.map((n) => n.id)),
		addNode: () => nodeCrud.handleAddNode(),
		addGroup: () => nodeCrud.handleAddGroup(),
		zoomIn: () => canvas?.zoomIn(),
		zoomOut: () => canvas?.zoomOut(),
		fitView: () => canvas?.fitView(),
		runLayout: (algorithm?: string) => canvas?.runLayout(algorithm),
		toggleImpact: () => { if (graphStore.selectedNodeId || graphStore.impactMode) graphStore.toggleImpactMode(); },
		toggleAnalysis: () => graphStore.toggleAnalysisMode(),
		togglePanel: () => { panelCollapsed = !panelCollapsed; },
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="h-full flex flex-col">
	<div class="flex-1 flex overflow-hidden">
		<!-- Canvas column -->
		<div class="flex-1 relative min-w-0">
			{#if loading}
				<div class="absolute inset-0 flex items-center justify-center">
					<p class="text-[var(--color-text-muted)]">Loading graph...</p>
				</div>
			{:else if loadError}
				<div class="absolute inset-0 flex flex-col items-center justify-center gap-3">
					<p class="text-sm" style="color: var(--color-danger);">{loadError}</p>
					<button
						onclick={() => location.reload()}
						class="px-4 py-2 rounded-lg text-sm font-medium"
						style="background: var(--color-surface); color: var(--color-text); border: 1px solid var(--color-border);"
					>Retry</button>
				</div>
			{:else}
				<CytoscapeCanvas
					bind:this={canvas}
					{nodes}
					{edges}
					{projectId}
					onUpdateNodeParent={nodeCrud.handleUpdateNodeParent}
					onCreateEdge={edgeCrud.handleCreateEdge}
					onZoomChange={(z) => (zoomLevel = z)}
				/>

				<!-- Floating toolbar -->
				<div class="absolute bottom-6 left-1/2 -translate-x-1/2 z-50">
					<GraphToolbar
						onAddNode={nodeCrud.handleAddNode}
						onAddGroup={nodeCrud.handleAddGroup}
						onZoomIn={() => canvas?.zoomIn()}
						onZoomOut={() => canvas?.zoomOut()}
						onFitView={() => canvas?.fitView()}
						onRunLayout={(algorithm) => canvas?.runLayout(algorithm)}
						onToggleImpact={() => graphStore.toggleImpactMode()}
						onToggleAnalysis={() => graphStore.toggleAnalysisMode()}
						onExportPNG={handleExportPNG}
						onExportJSON={handleExportJSON}
						onImport={handleImport}
						isImpactActive={graphStore.impactMode}
						isAnalysisActive={graphStore.analysisMode}
						canImpact={!!graphStore.selectedNodeId}
						{nodes}
						onFocusNode={(id) => canvas?.focusNode(id)}
						onUndo={() => undoStack.undo()}
						onRedo={() => undoStack.redo()}
						canUndo={undoStack.canUndo}
						canRedo={undoStack.canRedo}
						onShare={() => { showShareDialog = true; }}
					/>
				</div>

				<!-- Status bar -->
				<div class="absolute top-3 right-3 z-40 flex items-center gap-3 text-xs px-3 py-1.5 rounded-lg"
					style="background: rgba(27,26,30,0.9); backdrop-filter: blur(12px); color: var(--color-text-muted); border: 1px solid var(--color-border);">
					<span>{nodes.length} nodes</span>
					<span style="color: var(--color-border);">&middot;</span>
					<span>{edges.length} edges</span>
					<span style="color: var(--color-border);">&middot;</span>
					<span>{Math.round(zoomLevel * 100)}%</span>
				</div>

				<!-- Panel toggle button -->
				<button
					onclick={() => panelCollapsed = !panelCollapsed}
					class="absolute right-0 top-1/2 -translate-y-1/2 z-50 w-4 h-12 flex items-center justify-center rounded-l transition-colors"
					style="background: var(--color-surface); border: 1px solid var(--color-border); border-right: none; color: var(--color-text-muted);"
					title="{panelCollapsed ? 'Expand panel' : 'Collapse panel'}"
				>
					{#if panelCollapsed}
						<ChevronLeft size={12} />
					{:else}
						<ChevronRight size={12} />
					{/if}
				</button>
			{/if}
		</div>

		<!-- Side Panel (always visible) -->
		{#if !loading && !loadError}
			<div
				class="transition-all duration-200 overflow-hidden flex-shrink-0"
				style="width: {panelCollapsed ? '0' : '350px'};"
			>
				<DetailSidePanel
					bind:this={sidePanel}
					{panelMode}
					{projectId}
					node={selectedNodeDetail}
					allNodes={nodes}
					allEdges={edges}
					{selectedEdge}
					{selectedNodes}
					onclose={() => graphStore.clearSelection()}
					onUpdateNode={nodeCrud.handleUpdateNode}
					onDeleteNode={nodeCrud.handleDeleteNode}
					onSelectNode={handleSelectNodeFromPanel}
					onEdgeTypeChange={edgeCrud.handleEdgeTypeChange}
					onEdgeLabelUpdate={edgeCrud.handleEdgeLabelUpdate}
					onDeleteEdge={edgeCrud.handleDeleteEdge}
					onBatchDelete={nodeCrud.handleBatchDelete}
					onBatchStatus={nodeCrud.handleBatchStatus}
					onBatchType={nodeCrud.handleBatchType}
					onBatchAddTag={nodeCrud.handleBatchAddTag}
					onCreateGroupFromSelection={nodeCrud.handleCreateGroupFromSelection}
					onStartEdgeDrawing={handleStartEdgeDrawing}
				/>
			</div>
		{/if}
	</div>
</div>

{#if showShareDialog}
	<ShareDialog
		{projectId}
		{projectName}
		onClose={() => { showShareDialog = false; }}
	/>
{/if}

