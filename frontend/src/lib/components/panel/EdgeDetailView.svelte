<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { GraphEdge, GraphNode, EdgeType } from '$lib/types';
	import { EDGE_COLORS } from '$lib/constants';
	import { Trash2, ArrowRight } from 'lucide-svelte';

	interface Props {
		edge: GraphEdge;
		allNodes: GraphNode[];
		onselect: (edgeType: EdgeType) => void;
		onupdatelabel: (label: string) => void;
		ondelete: () => void;
		onselectnode: (nodeId: string) => void;
		readonly?: boolean;
	}

	let { edge, allNodes, onselect, onupdatelabel, ondelete, onselectnode, readonly = false }: Props = $props();

	const EDGE_TYPES: { value: EdgeType; label: string; color: string }[] = [
		{ value: 'depends_on', label: 'Depends On', color: EDGE_COLORS.depends_on },
		{ value: 'blocks', label: 'Blocks', color: EDGE_COLORS.blocks },
		{ value: 'related', label: 'Related', color: EDGE_COLORS.related },
		{ value: 'parent_child', label: 'Parent / Child', color: EDGE_COLORS.parent_child },
		{ value: 'triggers', label: 'Triggers', color: EDGE_COLORS.triggers },
	];

	let label = $state('');
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		label = edge.label ?? '';
	});

	function handleLabelInput() {
		if (debounceTimer) clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => { onupdatelabel(label); }, 400);
	}

	onDestroy(() => { if (debounceTimer) clearTimeout(debounceTimer); });

	let sourceNode = $derived(allNodes.find((n) => n.id === edge.sourceId));
	let targetNode = $derived(allNodes.find((n) => n.id === edge.targetId));
</script>

<!-- Source → Target display -->
<div class="flex items-center gap-2 pb-3 border-b" style="border-color: var(--color-border);">
	<button
		onclick={() => onselectnode(edge.sourceId)}
		class="flex-1 px-2 py-1.5 rounded text-xs font-medium text-left truncate transition-colors"
		style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
		onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
		onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
	>
		{sourceNode?.title ?? 'Unknown'}
	</button>
	<ArrowRight size={14} style="color: var(--color-text-muted); flex-shrink: 0;" />
	<button
		onclick={() => onselectnode(edge.targetId)}
		class="flex-1 px-2 py-1.5 rounded text-xs font-medium text-left truncate transition-colors"
		style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
		onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
		onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
	>
		{targetNode?.title ?? 'Unknown'}
	</button>
</div>

<!-- Label input -->
<div class="flex flex-col gap-1">
	<label class="text-xs font-medium" style="color: var(--color-text-muted);" for="edge-label">Label</label>
	<input
		id="edge-label"
		bind:value={label}
		oninput={readonly ? undefined : handleLabelInput}
		placeholder={readonly ? 'No label' : 'Edge label...'}
		class="px-2 py-1.5 rounded text-xs outline-none"
		style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
		{readonly}
	/>
</div>

<!-- Edge type selection -->
<div class="flex flex-col gap-1">
	<span class="text-xs font-medium" style="color: var(--color-text-muted);">Type</span>
	<div class="flex flex-col gap-1">
		{#each EDGE_TYPES as et}
			<button
				onclick={() => { if (!readonly) onselect(et.value); }}
				class="flex items-center gap-2 px-2 py-1.5 rounded text-xs font-medium text-left transition-colors"
				style="background: {edge.edgeType === et.value ? et.color + '22' : 'var(--color-bg)'}; color: {edge.edgeType === et.value ? et.color : 'var(--color-text)'}; border: 1px solid {edge.edgeType === et.value ? et.color + '44' : 'transparent'}; cursor: {readonly ? 'default' : 'pointer'};"
				onmouseenter={(e) => { if (!readonly && edge.edgeType !== et.value) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
				onmouseleave={(e) => { if (!readonly && edge.edgeType !== et.value) (e.currentTarget as HTMLElement).style.background = 'var(--color-bg)'; }}
			>
				<span class="w-3 h-3 rounded-full flex-shrink-0" style="background: {et.color};"></span>
				{et.label}
			</button>
		{/each}
	</div>
</div>

{#if !readonly}
<!-- Delete button -->
<button
	onclick={ondelete}
	class="mt-1 px-3 py-1.5 rounded text-xs font-medium transition-colors"
	style="background: rgba(196,64,64,0.13); color: var(--color-danger); border: 1px solid rgba(196,64,64,0.27);"
	onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(196,64,64,0.2)'; }}
	onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(196,64,64,0.13)'; }}
>
	<Trash2 size={12} class="inline mr-1" />Delete Edge
</button>
{/if}
