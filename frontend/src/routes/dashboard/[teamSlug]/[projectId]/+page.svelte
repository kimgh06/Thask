<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { graphStore } from '$lib/stores/graph.svelte';
	import { undoStack } from '$lib/stores/undo.svelte';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import GraphToolbar from '$lib/components/GraphToolbar.svelte';
	import EdgeColorPopover from '$lib/components/EdgeColorPopover.svelte';
	import NodeDetailPanel from '$lib/components/NodeDetailPanel.svelte';
	import { createNodeCrud } from '$lib/managers/nodeCrud.svelte';
	import { createEdgeCrud } from '$lib/managers/edgeCrud.svelte';
	import type { GraphNode, GraphEdge, GraphData, NodeDetail, NodeUpdateResult } from '$lib/types';
	import { computeLocalImpact } from '$lib/cytoscape/impact';
	import { createKeydownHandler } from '$lib/shortcuts';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let loading = $state(true);
	let loadError = $state('');

	const projectId = $derived(page.params.projectId ?? '');

	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);

	// Node detail state
	let selectedNodeDetail = $state<NodeDetail | null>(null);
	let detailLoading = $state(false);
	let detailRequestId = 0;

	// Node detail popup position
	let nodePopupPos = $state({ x: 0, y: 0 });

	// Edge popover state
	let selectedEdge = $state<GraphEdge | null>(null);
	let edgePopoverPos = $state({ x: 0, y: 0 });

	// Zoom level for status bar
	let zoomLevel = $state(1);

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
	$effect(() => {
		const currentProjectId = projectId;
		if (!currentProjectId) return;
		loading = true;
		loadError = '';
		api.get<GraphData>(`/api/projects/${currentProjectId}/graph`).then((res) => {
			if (projectId !== currentProjectId) return;
			if (res.error || !res.data) {
				loadError = res.error || 'Failed to load graph data.';
				loading = false;
				return;
			}
			nodes = res.data.nodes ?? [];
			edges = res.data.edges ?? [];
			loading = false;
		});
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
			const edge = edges.find((e) => e.id === edgeId) ?? null;
			selectedEdge = edge;
			if (edge) updateEdgePopoverPosition(edgeId);
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

	function updateEdgePopoverPosition(edgeId: string) {
		const cy = canvas?.getCy();
		if (!cy) return;
		const edgeEle = cy.getElementById(edgeId);
		if (!edgeEle.length) return;
		const rbb = edgeEle.renderedBoundingBox({});
		const containerRect = cy.container()?.getBoundingClientRect();
		if (!containerRect) return;
		edgePopoverPos = {
			x: containerRect.left + (rbb.x1 + rbb.x2) / 2,
			y: containerRect.top + (rbb.y1 + rbb.y2) / 2,
		};
	}

	function handleSelectNodeFromPanel(nodeId: string) {
		graphStore.selectNode(nodeId);
		canvas?.focusNode(nodeId);
	}

	const handleKeydown = createKeydownHandler({
		deleteSelection: () => {
			if (graphStore.selectedNodeIds.size > 1) nodeCrud.handleBatchDelete();
			else if (graphStore.selectedNodeId) nodeCrud.handleDeleteNode(graphStore.selectedNodeId);
			else if (graphStore.selectedEdgeId) edgeCrud.handleDeleteEdge();
		},
		escape: () => {
			if (selectedEdge || graphStore.selectedNodeId) graphStore.clearSelection();
		},
		undo: () => undoStack.undo(),
		redo: () => undoStack.redo(),
		selectAll: () => graphStore.selectNodes(nodes.map((n) => n.id)),
		addNode: () => nodeCrud.handleAddNode(),
		addGroup: () => nodeCrud.handleAddGroup(),
		zoomIn: () => canvas?.zoomIn(),
		zoomOut: () => canvas?.zoomOut(),
		fitView: () => canvas?.fitView(),
		runLayout: () => canvas?.runLayout(),
		toggleImpact: () => { if (graphStore.selectedNodeId || graphStore.impactMode) graphStore.toggleImpactMode(); },
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="h-full flex flex-col">
	<!-- Canvas area -->
	<div class="flex-1 relative">
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
				onNodeTap={(_id, pos) => { nodePopupPos = pos; }}
			/>

			<!-- Floating toolbar -->
			<div class="absolute bottom-6 left-1/2 -translate-x-1/2 z-40">
				<GraphToolbar
					onAddNode={nodeCrud.handleAddNode}
					onAddGroup={nodeCrud.handleAddGroup}
					onZoomIn={() => canvas?.zoomIn()}
					onZoomOut={() => canvas?.zoomOut()}
					onFitView={() => canvas?.fitView()}
					onRunLayout={() => canvas?.runLayout()}
					onToggleImpact={() => graphStore.toggleImpactMode()}
					isImpactActive={graphStore.impactMode}
					canImpact={!!graphStore.selectedNodeId}
					{nodes}
					onFocusNode={(id) => canvas?.focusNode(id)}
					onUndo={() => undoStack.undo()}
					onRedo={() => undoStack.redo()}
					canUndo={undoStack.canUndo}
					canRedo={undoStack.canRedo}
					selectedCount={graphStore.selectedNodeIds.size}
					onBatchDelete={nodeCrud.handleBatchDelete}
					onBatchStatus={nodeCrud.handleBatchStatus}
					onDeselectAll={() => graphStore.clearSelection()}
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
		{/if}
	</div>

	<!-- Edge Color Popover -->
	{#if selectedEdge}
		<EdgeColorPopover
			position={edgePopoverPos}
			currentLabel={selectedEdge.label ?? ''}
			onselect={edgeCrud.handleEdgeTypeChange}
			onupdatelabel={edgeCrud.handleEdgeLabelUpdate}
			ondelete={edgeCrud.handleDeleteEdge}
			oncancel={() => graphStore.clearSelection()}
		/>
	{/if}

	<!-- Node Detail Panel -->
	<NodeDetailPanel
		node={selectedNodeDetail}
		allNodes={nodes}
		history={selectedNodeDetail?.history ?? []}
		connectedNodeIds={selectedNodeDetail?.connectedNodeIds ?? []}
		isOpen={!!graphStore.selectedNodeId && !!selectedNodeDetail}
		position={nodePopupPos}
		onclose={() => graphStore.clearSelection()}
		onupdate={nodeCrud.handleUpdateNode}
		ondelete={nodeCrud.handleDeleteNode}
		onselectnode={handleSelectNodeFromPanel}
	/>
</div>
