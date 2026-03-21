<script lang="ts">
	import { page } from '$app/state';
	import type { GraphNode, GraphEdge, NodeDetail } from '$lib/types';
	import { graphStore } from '$lib/stores/graph.svelte';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import DetailSidePanel from '$lib/components/DetailSidePanel.svelte';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let loading = $state(true);
	let error = $state('');
	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);
	let selectedNodeDetail = $state<NodeDetail | null>(null);

	const shareToken = $derived(page.params.shareToken ?? '');
	const noop = () => {};

	const panelMode = $derived.by(() => {
		if (graphStore.selectedNodeId) return 'node' as const;
		if (graphStore.selectedEdgeId) return 'edge' as const;
		return null;
	});

	const selectedEdge = $derived(
		edges.find((e) => e.id === graphStore.selectedEdgeId) ?? null
	);

	function buildNodeDetail(nodeId: string): NodeDetail | null {
		const node = nodes.find((n) => n.id === nodeId);
		if (!node) return null;
		const connectedEdges = edges.filter((e) => e.sourceId === nodeId || e.targetId === nodeId);
		const connectedNodeIds = [...new Set(connectedEdges.map((e) => e.sourceId === nodeId ? e.targetId : e.sourceId))];
		return { ...node, connectedEdges, connectedNodeIds, history: [] };
	}

	$effect(() => {
		const nodeId = graphStore.selectedNodeId;
		selectedNodeDetail = nodeId ? buildNodeDetail(nodeId) : null;
	});

	$effect(() => {
		if (!shareToken) return;
		loading = true;

		fetch(`/api/shared/${shareToken}/graph`)
			.then((r) => r.json())
			.then((res) => {
				if (res.error || !res.data) {
					error = res.error || 'Not found';
					loading = false;
					return;
				}
				nodes = res.data.nodes ?? [];
				edges = res.data.edges ?? [];
				loading = false;
				requestAnimationFrame(() => {
					setTimeout(() => canvas?.fitView(), 200);
				});
			})
			.catch(() => {
				error = 'Failed to load';
				loading = false;
			});

		// SSE realtime
		const source = new EventSource(`/api/shared/${shareToken}/events`);
		let debounce: ReturnType<typeof setTimeout> | null = null;
		const types = ['node.created','node.updated','node.deleted','edge.created','edge.updated','edge.deleted','graph.layout','graph.import'];
		for (const t of types) {
			source.addEventListener(t, () => {
				if (debounce) clearTimeout(debounce);
				debounce = setTimeout(() => {
					fetch(`/api/shared/${shareToken}/graph`)
						.then((r) => r.json())
						.then((res) => {
							if (res.data) {
								nodes = res.data.nodes ?? [];
								edges = res.data.edges ?? [];
							}
						});
				}, 300);
			});
		}

		return () => {
			if (debounce) clearTimeout(debounce);
			source.close();
		};
	});
</script>

<svelte:head>
	<style>
		body { margin: 0; overflow: hidden; }
	</style>
</svelte:head>

<div class="h-screen w-screen overflow-hidden flex" style="background: var(--color-bg);">
	{#if loading}
		<div class="flex-1 flex items-center justify-center">
			<p style="color: var(--color-text-muted);">Loading...</p>
		</div>
	{:else if error}
		<div class="flex-1 flex items-center justify-center">
			<p style="color: var(--color-text-muted);">{error}</p>
		</div>
	{:else}
		<div class="flex-1 min-w-0 relative">
			<CytoscapeCanvas
				bind:this={canvas}
				{nodes}
				{edges}
				projectId=""
				readonly={true}
				onNodeTap={(nodeId) => graphStore.selectNode(nodeId)}
			/>
			<a
				href="/shared/{shareToken}"
				target="_blank"
				class="absolute bottom-2 right-3 text-[10px] no-underline transition-opacity hover:opacity-80"
				style="color: #555;"
			>
				Powered by Thask
			</a>
		</div>

		{#if panelMode}
			<DetailSidePanel
				{panelMode}
				node={selectedNodeDetail}
				allNodes={nodes}
				{selectedEdge}
				selectedNodes={[]}
				readonly={true}
				onclose={() => graphStore.clearSelection()}
				onUpdateNode={noop}
				onDeleteNode={noop}
				onSelectNode={(id) => graphStore.selectNode(id)}
				onEdgeTypeChange={noop}
				onEdgeLabelUpdate={noop}
				onDeleteEdge={noop}
				onBatchDelete={noop}
				onBatchStatus={noop}
				onBatchType={noop}
				onBatchAddTag={noop}
				onCreateGroupFromSelection={noop}
			/>
		{/if}
	{/if}
</div>
