<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import type { GraphNode, GraphEdge, GraphData } from '$lib/types';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import GraphToolbar from '$lib/components/GraphToolbar.svelte';
	import AddNodeModal from '$lib/components/AddNodeModal.svelte';
	import { exportPNG, exportJSON } from '$lib/export';
	import { Eye, Pencil } from 'lucide-svelte';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let projectName = $state('');
	let loading = $state(true);
	let error = $state('');
	let sseConnected = $state(false);
	let linkSharing = $state<'viewer' | 'editor'>('viewer');
	let showAddNode = $state(false);
	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);

	const shareToken = $derived(page.params.shareToken ?? '');
	const isEditor = $derived(linkSharing === 'editor');
	const sharedApiBase = $derived(`/api/shared/${shareToken}`);

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

	// Editor callbacks (use shared API base)
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
			sourceId,
			targetId,
			edgeType: 'related',
		});
		if (res.data) {
			edges = [...edges, res.data];
		}
	}

	async function handleUpdateNodeParent(nodeId: string, parentId: string | null) {
		await api.patch(`${sharedApiBase}/nodes/${nodeId}`, { parentId });
		loadGraph();
	}
</script>

<div class="flex flex-col h-screen" style="background: var(--color-bg);">
	<header
		class="flex items-center justify-between px-4 py-3 shrink-0"
		style="background: var(--color-surface); border-bottom: 1px solid var(--color-border);"
	>
		<div class="flex items-center gap-3">
			<h1 class="text-base font-semibold" style="color: var(--color-text);">
				{projectName || 'Shared Project'}
			</h1>
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

	<div class="flex-1 overflow-hidden relative">
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
						onToggleImpact={() => {}}
						onExportPNG={() => { const cy = canvas?.getCy(); if (cy) exportPNG(cy); }}
						onExportJSON={() => exportJSON(nodes, edges)}
						onImport={() => {}}
						isImpactActive={false}
						canImpact={false}
						{nodes}
						onFocusNode={(id) => canvas?.focusNode(id)}
						onUndo={() => {}}
						onRedo={() => {}}
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
