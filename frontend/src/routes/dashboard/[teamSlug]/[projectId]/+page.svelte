<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import { graphStore } from '$lib/stores/graph.svelte';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import GraphToolbar from '$lib/components/GraphToolbar.svelte';
	import AddNodeModal from '$lib/components/AddNodeModal.svelte';
	import EdgeColorPopover from '$lib/components/EdgeColorPopover.svelte';
	import NodeDetailPanel from '$lib/components/NodeDetailPanel.svelte';
	import type { GraphNode, GraphEdge, NodeDetail, NodeType, EdgeType } from '$lib/types';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let loading = $state(true);

	const projectId = $derived(page.params.projectId ?? '');

	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);

	// Modal / panel state
	let showAddNodeModal = $state(false);
	let showAddGroupModal = $state(false);

	// Node detail state
	let selectedNodeDetail = $state<NodeDetail | null>(null);
	let detailLoading = $state(false);

	// Edge popover state
	let selectedEdge = $state<GraphEdge | null>(null);
	let edgePopoverPos = $state({ x: 0, y: 0 });

	// Undo/redo stubs (to be implemented later)
	let canUndo = $state(false);
	let canRedo = $state(false);

	$effect(() => {
		loadGraph();
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

	async function loadGraph() {
		if (!projectId) return;
		loading = true;
		const [nodeRes, edgeRes] = await Promise.all([
			api.get<GraphNode[]>(`/api/projects/${projectId}/nodes`),
			api.get<GraphEdge[]>(`/api/projects/${projectId}/edges`),
		]);
		nodes = nodeRes.data ?? [];
		edges = edgeRes.data ?? [];
		loading = false;
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
		const res = await api.post<GraphNode>(`/api/projects/${projectId}/nodes`, {
			title: data.title,
			type: data.type,
			status: 'IN_PROGRESS',
		});
		if (res.data) {
			nodes = [...nodes, res.data];
		}
	}

	async function handleAddGroup() {
		const res = await api.post<GraphNode>(`/api/projects/${projectId}/nodes`, {
			title: 'New Group',
			type: 'GROUP',
			status: 'IN_PROGRESS',
		});
		if (res.data) {
			nodes = [...nodes, res.data];
		}
	}

	async function handleUpdateNode(nodeId: string, data: Record<string, unknown>) {
		const res = await api.patch<GraphNode>(`/api/projects/${projectId}/nodes/${nodeId}`, data);
		if (res.data) {
			nodes = nodes.map((n) => (n.id === nodeId ? res.data! : n));
			if (selectedNodeDetail?.id === nodeId) {
				selectedNodeDetail = { ...selectedNodeDetail, ...res.data };
			}
		}
	}

	async function handleDeleteNode(nodeId: string) {
		const res = await api.delete(`/api/projects/${projectId}/nodes/${nodeId}`);
		if (!res.error) {
			nodes = nodes.filter((n) => n.id !== nodeId);
			edges = edges.filter((e) => e.sourceId !== nodeId && e.targetId !== nodeId);
			graphStore.clearSelection();
		}
	}

	// --- Edge CRUD ---
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
		const edgeId = selectedEdge.id;
		const res = await api.delete(`/api/projects/${projectId}/edges/${edgeId}`);
		if (!res.error) {
			edges = edges.filter((e) => e.id !== edgeId);
			graphStore.clearSelection();
		}
	}

	function handleSelectNodeFromPanel(nodeId: string) {
		graphStore.selectNode(nodeId);
		canvas?.focusNode(nodeId);
	}
</script>

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
					onUndo={() => {}}
					onRedo={() => {}}
					{canUndo}
					{canRedo}
				/>
			</div>

			<!-- Node count badge -->
			<div class="absolute bottom-3 left-3 z-10 text-xs px-2 py-1 rounded"
				style="background: var(--color-surface); color: var(--color-text-muted); border: 1px solid var(--color-border);">
				{nodes.length} nodes, {edges.length} edges
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
