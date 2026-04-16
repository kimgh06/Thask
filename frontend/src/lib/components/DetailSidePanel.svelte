<script lang="ts">
	import { X } from 'lucide-svelte';
	import type { GraphNode, GraphEdge, NodeType, NodeStatus, EdgeType, NodeDetail } from '$lib/types';
	import NodeDetailView from './panel/NodeDetailView.svelte';
	import EdgeDetailView from './panel/EdgeDetailView.svelte';
	import MultiSelectView from './panel/MultiSelectView.svelte';
	import ActivityFeed from './ActivityFeed.svelte';

	interface Props {
		panelMode: 'node' | 'edge' | 'multi-select' | 'empty';
		projectId?: string;
		// Node mode
		node: NodeDetail | null;
		allNodes: GraphNode[];
		allEdges?: GraphEdge[];
		// Edge mode
		selectedEdge: GraphEdge | null;
		// Multi-select mode
		selectedNodes: GraphNode[];
		// Callbacks
		onclose: () => void;
		onUpdateNode: (nodeId: string, data: Record<string, unknown>) => void;
		onDeleteNode: (nodeId: string) => void;
		onSelectNode: (nodeId: string) => void;
		onEdgeTypeChange: (edgeType: EdgeType) => void;
		onEdgeLabelUpdate: (label: string) => void;
		onDeleteEdge: () => void;
		onBatchDelete: () => void;
		onBatchStatus: (status: NodeStatus) => void;
		onBatchType: (type: NodeType) => void;
		onBatchAddTag: (tag: string) => void;
		onCreateGroupFromSelection: () => void;
		onStartEdgeDrawing?: (nodeId: string) => void;
		readonly?: boolean;
	}

	let {
		panelMode,
		projectId,
		node,
		allNodes,
		allEdges,
		selectedEdge,
		selectedNodes,
		onclose,
		onUpdateNode,
		onDeleteNode,
		onSelectNode,
		onEdgeTypeChange,
		onEdgeLabelUpdate,
		onDeleteEdge,
		onBatchDelete,
		onBatchStatus,
		onBatchType,
		onBatchAddTag,
		onCreateGroupFromSelection,
		onStartEdgeDrawing,
		readonly = false,
	}: Props = $props();

	let activityFeed = $state<ReturnType<typeof ActivityFeed> | undefined>(undefined);

	let panelTitle = $derived.by(() => {
		if (panelMode === 'node') return 'Node';
		if (panelMode === 'edge') return 'Edge';
		if (panelMode === 'multi-select') return 'Selection';
		return 'Activity';
	});

	export function refreshActivity() {
		activityFeed?.refresh();
	}
</script>

<aside
	class="w-[350px] flex-shrink-0 flex flex-col h-full"
	style="background: var(--color-surface); border-left: 1px solid var(--color-border);"
>
	<!-- Header -->
	<div class="flex items-center justify-between px-3 py-2.5 border-b flex-shrink-0" style="border-color: var(--color-border);">
		<span class="text-xs font-semibold uppercase tracking-wider" style="color: var(--color-text-muted);">{panelTitle}</span>
		<button
			onclick={onclose}
			class="w-6 h-6 flex items-center justify-center rounded transition-colors"
			style="color: var(--color-text-muted);"
			onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text)'; (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
			onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text-muted)'; (e.currentTarget as HTMLElement).style.background = ''; }}
			aria-label="Close panel"
		>
			<X size={16} />
		</button>
	</div>

	<!-- Content (hidden in empty mode) -->
	{#if panelMode !== 'empty'}
		<div class="flex-1 overflow-y-auto p-3 flex flex-col gap-3">
			{#if panelMode === 'node' && node}
				<NodeDetailView
					{node}
					{allNodes}
					{allEdges}
					onupdate={onUpdateNode}
					ondelete={onDeleteNode}
					onselectnode={onSelectNode}
					onstartedge={onStartEdgeDrawing}
					{readonly}
				/>
			{:else if panelMode === 'edge' && selectedEdge}
				<EdgeDetailView
					edge={selectedEdge}
					{allNodes}
					onselect={onEdgeTypeChange}
					onupdatelabel={onEdgeLabelUpdate}
					ondelete={onDeleteEdge}
					onselectnode={onSelectNode}
					{readonly}
				/>
			{:else if panelMode === 'multi-select'}
				<MultiSelectView
					{selectedNodes}
					onbatchdelete={onBatchDelete}
					onbatchstatus={onBatchStatus}
					onbatchtype={onBatchType}
					onbatchaddtag={onBatchAddTag}
					oncreategroup={onCreateGroupFromSelection}
				/>
			{/if}
		</div>
	{/if}

	<!-- Activity Feed (always visible, expands to fill space in empty mode) -->
	{#if projectId}
		<div class={panelMode === 'empty' ? 'flex-1 overflow-y-auto' : 'flex-shrink-0 max-h-[40%] overflow-y-auto'}>
			<ActivityFeed bind:this={activityFeed} {projectId} expanded={panelMode === 'empty'} />
		</div>
	{/if}
</aside>
