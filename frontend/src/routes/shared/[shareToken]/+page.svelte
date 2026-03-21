<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import type { GraphNode, GraphEdge, NodeDetail, EdgeType, NodeStatus, NodeType } from '$lib/types';
	import { graphStore } from '$lib/stores/graph.svelte';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import DetailSidePanel from '$lib/components/DetailSidePanel.svelte';
	import GraphToolbar from '$lib/components/GraphToolbar.svelte';
	import AddNodeModal from '$lib/components/AddNodeModal.svelte';
	import { exportPNG, exportJSON } from '$lib/export';
	import { Eye, Pencil } from 'lucide-svelte';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let projectName = $state('');
	let projectDescription = $state('');
	let loading = $state(true);
	let error = $state('');
	let sseConnected = $state(false);
	let linkSharing = $state<'viewer' | 'editor'>('viewer');
	let showAddNode = $state(false);
	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);
	let selectedNodeDetail = $state<NodeDetail | null>(null);

	const shareToken = $derived(page.params.shareToken ?? '');
	const isEditor = $derived(linkSharing === 'editor');
	const sharedApiBase = $derived(`/api/shared/${shareToken}`);

	// Panel mode derived from graphStore
	const panelMode = $derived.by(() => {
		if (graphStore.selectedNodeIds.size > 1) return 'multi-select' as const;
		if (graphStore.selectedNodeId) return 'node' as const;
		if (graphStore.selectedEdgeId) return 'edge' as const;
		return null;
	});

	const selectedEdge = $derived(
		edges.find((e) => e.id === graphStore.selectedEdgeId) ?? null
	);

	const selectedNodes = $derived(
		nodes.filter((n) => graphStore.selectedNodeIds.has(n.id))
	);

	let graphAbort: AbortController | null = null;

	function loadGraph() {
		graphAbort?.abort();
		graphAbort = new AbortController();
		fetch(`${sharedApiBase}/graph`, { signal: graphAbort.signal })
			.then((r) => r.json())
			.then((res) => {
				if (res.error || !res.data) {
					error = res.error || 'Failed to load graph';
					loading = false;
					return;
				}
				nodes = res.data.nodes ?? [];
				edges = res.data.edges ?? [];
				loading = false;
			})
			.catch((e) => {
				if (e?.name === 'AbortError') return;
				error = 'Failed to load graph';
				loading = false;
			});
	}

	// Build node detail from local data (no separate API call needed for shared)
	function buildNodeDetail(nodeId: string): NodeDetail | null {
		const node = nodes.find((n) => n.id === nodeId);
		if (!node) return null;
		const connectedEdges = edges.filter((e) => e.sourceId === nodeId || e.targetId === nodeId);
		const connectedNodeIds = [...new Set(connectedEdges.map((e) => e.sourceId === nodeId ? e.targetId : e.sourceId))];
		return { ...node, connectedEdges, connectedNodeIds, history: [] };
	}

	$effect(() => {
		const nodeId = graphStore.selectedNodeId;
		if (nodeId) {
			selectedNodeDetail = buildNodeDetail(nodeId);
		} else {
			selectedNodeDetail = null;
		}
	});

	$effect(() => {
		if (!shareToken) return;
		loading = true;
		error = '';

		fetch(`${sharedApiBase}`)
			.then((r) => {
				if (!r.ok) throw new Error('Not found');
				return r.json();
			})
			.then((res) => {
				if (res.data) {
					projectName = res.data.name;
					projectDescription = res.data.description ?? '';
					linkSharing = res.data.linkSharing;
				}
			})
			.catch(() => {});

		loadGraph();

		// SSE
		let debounceTimer: ReturnType<typeof setTimeout> | null = null;
		const source = new EventSource(`${sharedApiBase}/events`);
		source.addEventListener('connected', () => { sseConnected = true; });
		const eventTypes = [
			'node.created', 'node.updated', 'node.deleted',
			'edge.created', 'edge.updated', 'edge.deleted',
			'graph.layout', 'graph.import',
		];
		for (const type of eventTypes) {
			source.addEventListener(type, () => {
				if (debounceTimer) clearTimeout(debounceTimer);
				debounceTimer = setTimeout(() => loadGraph(), 300);
			});
		}
		source.onerror = () => { sseConnected = false; };

		return () => {
			if (debounceTimer) clearTimeout(debounceTimer);
			graphAbort?.abort();
			source.close();
		};
	});

	// noop for viewer
	const noop = () => {};

	// Editor callbacks
	async function handleCreateNode(data: { type: string; title: string }) {
		const res = await api.post<GraphNode>(`${sharedApiBase}/nodes`, {
			...data,
			positionX: 100 + Math.random() * 200,
			positionY: 100 + Math.random() * 200,
		});
		if (res.data) {
			nodes = [...nodes, res.data];
			showAddNode = false;
		}
	}

	async function handleCreateEdge(sourceId: string, targetId: string) {
		const res = await api.post<GraphEdge>(`${sharedApiBase}/edges`, {
			sourceId, targetId, edgeType: 'related',
		});
		if (res.data) {
			edges = [...edges, res.data];
		}
	}

	async function handleUpdateNodeParent(nodeId: string, parentId: string | null) {
		if (!isEditor) return;
		await api.patch(`${sharedApiBase}/nodes/${nodeId}`, { parentId });
		loadGraph();
	}

	async function handleUpdateNode(nodeId: string, data: Record<string, unknown>) {
		if (!isEditor) return;
		const res = await api.patch<{ node: GraphNode }>(`${sharedApiBase}/nodes/${nodeId}`, data);
		if (res.data) {
			nodes = nodes.map((n) => n.id === nodeId ? res.data!.node : n);
			if (selectedNodeDetail?.id === nodeId) {
				selectedNodeDetail = buildNodeDetail(nodeId);
			}
		}
	}

	async function handleDeleteNode(nodeId: string) {
		if (!isEditor) return;
		await api.delete(`${sharedApiBase}/nodes/${nodeId}`);
		nodes = nodes.filter((n) => n.id !== nodeId);
		edges = edges.filter((e) => e.sourceId !== nodeId && e.targetId !== nodeId);
		graphStore.clearSelection();
	}

	async function handleEdgeTypeChange(edgeType: EdgeType) {
		if (!isEditor || !graphStore.selectedEdgeId) return;
		await api.patch(`${sharedApiBase}/edges/${graphStore.selectedEdgeId}`, { edgeType });
		loadGraph();
	}

	async function handleEdgeLabelUpdate(label: string) {
		if (!isEditor || !graphStore.selectedEdgeId) return;
		await api.patch(`${sharedApiBase}/edges/${graphStore.selectedEdgeId}`, { label });
		loadGraph();
	}

	async function handleDeleteEdge() {
		if (!isEditor || !graphStore.selectedEdgeId) return;
		await api.delete(`${sharedApiBase}/edges/${graphStore.selectedEdgeId}`);
		edges = edges.filter((e) => e.id !== graphStore.selectedEdgeId);
		graphStore.clearSelection();
	}

	function handleSelectNode(nodeId: string) {
		graphStore.selectNode(nodeId);
	}
</script>

<svelte:head>
	<title>{projectName || 'Shared Graph'} - Thask</title>
	<meta property="og:title" content={projectName || 'Shared Graph'} />
	<meta property="og:description" content={projectDescription || 'Interactive graph on Thask'} />
	<meta property="og:image" content="{sharedApiBase}/og-image" />
	<meta property="og:type" content="website" />
</svelte:head>

<div class="flex flex-col h-screen" style="background: var(--color-bg);">
	<header
		class="flex items-center justify-between px-4 py-3 shrink-0"
		style="background: var(--color-surface); border-bottom: 1px solid var(--color-border);"
	>
		<div class="flex items-center gap-3">
			<div>
				<h1 class="text-base font-semibold" style="color: var(--color-text);">
					{projectName || 'Shared Project'}
				</h1>
				{#if projectDescription}
					<p class="text-xs" style="color: var(--color-text-muted);">{projectDescription}</p>
				{/if}
			</div>
			<span
				class="flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
				style="background: var(--color-bg); color: var(--color-text-muted); border: 1px solid var(--color-border);"
			>
				{#if isEditor}
					<Pencil size={12} />
					Editor
				{:else}
					<Eye size={12} />
					View only
				{/if}
			</span>
			{#if sseConnected}
				<span class="w-2 h-2 rounded-full" style="background: var(--color-success);" title="Live"></span>
			{/if}
		</div>
		<div class="flex items-center gap-2">
			<a
				href="/login"
				class="px-3 py-1.5 rounded-lg text-sm font-medium"
				style="background: var(--color-bg); color: var(--color-text-muted); border: 1px solid var(--color-border);"
			>
				Sign in
			</a>
		</div>
	</header>

	<div class="flex-1 flex overflow-hidden">
		<div class="flex-1 min-w-0 relative">
			{#if loading}
				<div class="flex items-center justify-center h-full">
					<p style="color: var(--color-text-muted);">Loading...</p>
				</div>
			{:else if error}
				<div class="flex items-center justify-center h-full">
					<div class="text-center">
						<p class="text-lg font-medium mb-2" style="color: var(--color-text);">Access denied</p>
						<p style="color: var(--color-text-muted);">{error}</p>
					</div>
				</div>
			{:else}
				{#if isEditor}
					<div class="absolute bottom-6 left-1/2 -translate-x-1/2 z-40">
						<GraphToolbar
							onAddNode={() => { showAddNode = true; }}
							onAddGroup={() => { handleCreateNode({ type: 'GROUP', title: 'New Group' }); }}
							onZoomIn={() => canvas?.zoomIn()}
							onZoomOut={() => canvas?.zoomOut()}
							onFitView={() => canvas?.fitView()}
							onRunLayout={() => canvas?.runLayout()}
							onToggleImpact={noop}
							onExportPNG={() => { const cy = canvas?.getCy(); if (cy) exportPNG(cy); }}
							onExportJSON={() => exportJSON(nodes, edges)}
							onImport={noop}
							isImpactActive={false}
							canImpact={false}
							{nodes}
							onFocusNode={(id) => canvas?.focusNode(id)}
							onUndo={noop}
							onRedo={noop}
							canUndo={false}
							canRedo={false}
						/>
					</div>
				{/if}
				<CytoscapeCanvas
					bind:this={canvas}
					{nodes}
					{edges}
					projectId=""
					readonly={!isEditor}
					apiBasePath={isEditor ? sharedApiBase : undefined}
					onCreateEdge={isEditor ? handleCreateEdge : undefined}
					onUpdateNodeParent={isEditor ? handleUpdateNodeParent : undefined}
					onNodeTap={(nodeId) => graphStore.selectNode(nodeId)}
				/>
			{/if}
		</div>

		<!-- Side Panel -->
		{#if panelMode}
			<DetailSidePanel
				{panelMode}
				node={selectedNodeDetail}
				allNodes={nodes}
				{selectedEdge}
				{selectedNodes}
				readonly={!isEditor}
				onclose={() => graphStore.clearSelection()}
				onUpdateNode={isEditor ? handleUpdateNode : noop}
				onDeleteNode={isEditor ? handleDeleteNode : noop}
				onSelectNode={handleSelectNode}
				onEdgeTypeChange={isEditor ? handleEdgeTypeChange : noop}
				onEdgeLabelUpdate={isEditor ? handleEdgeLabelUpdate : noop}
				onDeleteEdge={isEditor ? handleDeleteEdge : noop}
				onBatchDelete={noop}
				onBatchStatus={noop}
				onBatchType={noop}
				onBatchAddTag={noop}
				onCreateGroupFromSelection={noop}
			/>
		{/if}
	</div>
</div>

{#if showAddNode && isEditor}
	<AddNodeModal
		onsubmit={handleCreateNode}
		onclose={() => { showAddNode = false; }}
	/>
{/if}
