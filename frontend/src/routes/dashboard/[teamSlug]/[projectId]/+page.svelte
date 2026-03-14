<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { graphStore } from '$lib/stores/graph.svelte';
	import { undoStack } from '$lib/stores/undo.svelte';
	import { createNodeCmd, deleteNodeCmd, createEdgeCmd, deleteEdgeCmd, batchDeleteCmd } from '$lib/commands/node';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import GraphToolbar from '$lib/components/GraphToolbar.svelte';
	import AddNodeModal from '$lib/components/AddNodeModal.svelte';
	import EdgeColorPopover from '$lib/components/EdgeColorPopover.svelte';
	import NodeDetailPanel from '$lib/components/NodeDetailPanel.svelte';
	import type { GraphNode, GraphEdge, NodeDetail, NodeType, NodeStatus, EdgeType, NodeUpdateResult, StatusChange, ImpactResult } from '$lib/types';
	import { createKeydownHandler } from '$lib/shortcuts';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let loading = $state(true);

	const projectId = $derived(page.params.projectId ?? '');

	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);

	// Modal / panel state
	let showAddNodeModal = $state(false);

	// Node detail state
	let selectedNodeDetail = $state<NodeDetail | null>(null);
	let detailLoading = $state(false);

	// Edge popover state
	let selectedEdge = $state<GraphEdge | null>(null);
	let edgePopoverPos = $state({ x: 0, y: 0 });

	// Mutation context for undo commands
	const mutCtx = {
		get projectId() { return projectId; },
		getNodes: () => nodes,
		setNodes: (v: GraphNode[]) => { nodes = v; },
		getEdges: () => edges,
		setEdges: (v: GraphEdge[]) => { edges = v; },
	};

	// Zoom level for status bar
	let zoomLevel = $state(1);

	$effect(() => {
		const currentProjectId = projectId;
		if (!currentProjectId) return;
		loading = true;
		Promise.all([
			api.get<GraphNode[]>(`/api/projects/${currentProjectId}/nodes`),
			api.get<GraphEdge[]>(`/api/projects/${currentProjectId}/edges`),
		]).then(([nodeRes, edgeRes]) => {
			if (projectId !== currentProjectId) return; // stale response
			nodes = nodeRes.data ?? [];
			edges = edgeRes.data ?? [];
			loading = false;
		}).catch(() => {
			if (projectId !== currentProjectId) return;
			loading = false;
		});
	});

	// React to node selection from graphStore
	$effect(() => {
		const nodeId = graphStore.selectedNodeId;
		if (nodeId) {
			fetchNodeDetail(nodeId);
		} else {
			selectedNodeDetail = null;
		}
	});

	// React to edge selection from graphStore
	$effect(() => {
		const edgeId = graphStore.selectedEdgeId;
		if (edgeId) {
			const edge = edges.find((e) => e.id === edgeId) ?? null;
			selectedEdge = edge;
			if (edge) {
				updateEdgePopoverPosition(edgeId);
			}
		} else {
			selectedEdge = null;
		}
	});

	// React to impact mode toggle
	$effect(() => {
		const active = graphStore.impactMode;
		if (active && projectId) {
			fetchImpactData();
		} else {
			canvas?.clearImpactClasses();
		}
	});

	async function fetchImpactData() {
		const res = await api.get<ImpactResult>(`/api/projects/${projectId}/impact?depth=2`);
		if (res.data && graphStore.impactMode) {
			const changedIds = res.data.changedNodes.map((n) => n.id);
			const affectedIds = res.data.impactedNodes.map((n) => n.id);
			canvas?.applyImpactClasses(changedIds, affectedIds);
		}
	}

	async function fetchNodeDetail(nodeId: string) {
		detailLoading = true;
		const res = await api.get<NodeDetail>(`/api/projects/${projectId}/nodes/${nodeId}`);
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

	// --- Node CRUD ---
	async function handleAddNode(data: { title: string; type: NodeType }) {
		showAddNodeModal = false;
		await undoStack.run(createNodeCmd(mutCtx, { title: data.title, type: data.type }));
	}

	async function handleAddGroup() {
		await undoStack.run(createNodeCmd(mutCtx, { title: 'New Group', type: 'GROUP' }));
	}

	async function handleUpdateNode(nodeId: string, data: Record<string, unknown>) {
		const res = await api.patch<NodeUpdateResult>(`/api/projects/${projectId}/nodes/${nodeId}`, data);
		if (res.data) {
			const updated = res.data.node;
			const propagated = res.data.propagated ?? [];

			nodes = nodes.map((n) => (n.id === nodeId ? updated : n));
			if (selectedNodeDetail?.id === nodeId) {
				selectedNodeDetail = { ...selectedNodeDetail, ...updated };
			}

			// Apply cascaded status changes to local nodes
			if (propagated.length > 0) {
				const changeMap = new Map(propagated.map((c) => [c.nodeId, c.newStatus]));
				nodes = nodes.map((n) => changeMap.has(n.id) ? { ...n, status: changeMap.get(n.id)! } : n);
				canvas?.animateCascade(propagated);
			}
		}
	}

	async function handleDeleteNode(nodeId: string) {
		const node = nodes.find((n) => n.id === nodeId);
		if (!node) return;
		const connectedEdges = edges.filter((e) => e.sourceId === nodeId || e.targetId === nodeId);
		await undoStack.run(deleteNodeCmd(mutCtx, node, connectedEdges));
		graphStore.clearSelection();
	}

	async function handleBatchDelete() {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length === 0) return;
		const idSet = new Set(ids);
		const deletedNodes = nodes.filter((n) => idSet.has(n.id));
		const deletedEdges = edges.filter((e) => idSet.has(e.sourceId) || idSet.has(e.targetId));
		await undoStack.run(batchDeleteCmd(mutCtx, deletedNodes, deletedEdges));
		graphStore.clearSelection();
	}

	async function handleBatchStatus(status: NodeStatus) {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length === 0) return;
		const res = await api.patch(`/api/projects/${projectId}/nodes/batch-status`, { ids, status });
		if (!res.error) {
			const idSet = new Set(ids);
			nodes = nodes.map((n) => idSet.has(n.id) ? { ...n, status } : n);
		}
	}

	// --- Edge CRUD ---
	async function handleCreateEdge(sourceId: string, targetId: string) {
		const duplicate = edges.some(
			(e) => e.sourceId === sourceId && e.targetId === targetId && e.edgeType === 'related',
		);
		if (duplicate) return;
		await undoStack.run(createEdgeCmd(mutCtx, { sourceId, targetId }));
	}

	async function handleEdgeTypeChange(edgeType: EdgeType) {
		if (!selectedEdge) return;
		const res = await api.patch<GraphEdge>(
			`/api/projects/${projectId}/edges/${selectedEdge.id}`,
			{ edgeType },
		);
		if (res.data) {
			edges = edges.map((e) => (e.id === res.data!.id ? res.data! : e));
			selectedEdge = res.data;
		}
	}

	async function handleEdgeLabelUpdate(label: string) {
		if (!selectedEdge) return;
		const res = await api.patch<GraphEdge>(
			`/api/projects/${projectId}/edges/${selectedEdge.id}`,
			{ label },
		);
		if (res.data) {
			edges = edges.map((e) => (e.id === res.data!.id ? res.data! : e));
			selectedEdge = res.data;
		}
	}

	async function handleDeleteEdge() {
		if (!selectedEdge) return;
		const edge = selectedEdge;
		await undoStack.run(deleteEdgeCmd(mutCtx, edge));
		graphStore.clearSelection();
	}

	async function handleUpdateNodeParent(nodeId: string, parentId: string | null) {
		const res = await api.patch<GraphNode>(`/api/projects/${projectId}/nodes/${nodeId}`, { parentId });
		if (res.data) {
			nodes = nodes.map((n) => (n.id === nodeId ? res.data! : n));
		}
	}

	function handleSelectNodeFromPanel(nodeId: string) {
		graphStore.selectNode(nodeId);
		canvas?.focusNode(nodeId);
	}

	const handleKeydown = createKeydownHandler({
		deleteSelection: () => {
			if (graphStore.selectedNodeIds.size > 1) handleBatchDelete();
			else if (graphStore.selectedNodeId) handleDeleteNode(graphStore.selectedNodeId);
			else if (graphStore.selectedEdgeId) handleDeleteEdge();
		},
		escape: () => {
			if (showAddNodeModal) { showAddNodeModal = false; return; }
			if (selectedEdge || graphStore.selectedNodeId) graphStore.clearSelection();
		},
		undo: () => undoStack.undo(),
		redo: () => undoStack.redo(),
		selectAll: () => graphStore.selectNodes(nodes.map((n) => n.id)),
		addNode: () => { showAddNodeModal = true; },
		addGroup: () => handleAddGroup(),
		zoomIn: () => canvas?.zoomIn(),
		zoomOut: () => canvas?.zoomOut(),
		fitView: () => canvas?.fitView(),
		runLayout: () => canvas?.runLayout(),
		toggleImpact: () => graphStore.toggleImpactMode(),
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
		{:else}
			<CytoscapeCanvas
				bind:this={canvas}
				{nodes}
				{edges}
				{projectId}
				onUpdateNodeParent={handleUpdateNodeParent}
				onCreateEdge={handleCreateEdge}
				onZoomChange={(z) => (zoomLevel = z)}
			/>

			<!-- Floating toolbar -->
			<div class="absolute top-3 left-3 z-10">
				<GraphToolbar
					onAddNode={() => (showAddNodeModal = true)}
					onAddGroup={handleAddGroup}
					onZoomIn={() => canvas?.zoomIn()}
					onZoomOut={() => canvas?.zoomOut()}
					onFitView={() => canvas?.fitView()}
					onRunLayout={() => canvas?.runLayout()}
					onToggleImpact={() => graphStore.toggleImpactMode()}
					isImpactActive={graphStore.impactMode}
					{nodes}
					onFocusNode={(id) => canvas?.focusNode(id)}
					onUndo={() => undoStack.undo()}
					onRedo={() => undoStack.redo()}
					canUndo={undoStack.canUndo}
					canRedo={undoStack.canRedo}
					selectedCount={graphStore.selectedNodeIds.size}
					onBatchDelete={handleBatchDelete}
					onBatchStatus={handleBatchStatus}
					onDeselectAll={() => graphStore.clearSelection()}
				/>
			</div>

			<!-- Status bar -->
			<div class="absolute bottom-3 left-3 z-10 flex items-center gap-3 text-xs px-3 py-1.5 rounded-lg"
				style="background: rgba(30,41,59,0.85); backdrop-filter: blur(12px); color: var(--color-text-muted); border: 1px solid var(--color-border);">
				<span>{nodes.length} nodes</span>
				<span style="color: var(--color-border);">&middot;</span>
				<span>{edges.length} edges</span>
				<span style="color: var(--color-border);">&middot;</span>
				<span>{Math.round(zoomLevel * 100)}%</span>
			</div>
		{/if}
	</div>

	<!-- Add Node Modal -->
	{#if showAddNodeModal}
		<AddNodeModal
			onsubmit={handleAddNode}
			onclose={() => (showAddNodeModal = false)}
		/>
	{/if}

	<!-- Edge Color Popover -->
	{#if selectedEdge}
		<EdgeColorPopover
			position={edgePopoverPos}
			currentLabel={selectedEdge.label ?? ''}
			onselect={handleEdgeTypeChange}
			onupdatelabel={handleEdgeLabelUpdate}
			ondelete={handleDeleteEdge}
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
		onclose={() => graphStore.clearSelection()}
		onupdate={handleUpdateNode}
		ondelete={handleDeleteNode}
		onselectnode={handleSelectNodeFromPanel}
	/>
</div>
