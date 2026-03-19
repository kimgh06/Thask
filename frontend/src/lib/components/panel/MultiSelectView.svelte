<script lang="ts">
	import type { GraphNode, NodeType, NodeStatus } from '$lib/types';
	import { NODE_TYPES_NO_GROUP, STATUS_OPTIONS, TYPE_COLORS, STATUS_COLORS, STATUS_LABELS } from '$lib/constants';
	import { Trash2, Tag, Group } from 'lucide-svelte';

	interface Props {
		selectedNodes: GraphNode[];
		onbatchdelete: () => void;
		onbatchstatus: (status: NodeStatus) => void;
		onbatchtype: (type: NodeType) => void;
		onbatchaddtag: (tag: string) => void;
		oncreategroup: () => void;
	}

	let { selectedNodes, onbatchdelete, onbatchstatus, onbatchtype, onbatchaddtag, oncreategroup }: Props = $props();

	let newTag = $state('');

	// Type distribution
	let typeDistribution = $derived.by(() => {
		const counts = new Map<NodeType, number>();
		for (const n of selectedNodes) {
			counts.set(n.type, (counts.get(n.type) ?? 0) + 1);
		}
		return [...counts.entries()].sort((a, b) => b[1] - a[1]);
	});

	// Status distribution
	let statusDistribution = $derived.by(() => {
		const counts = new Map<NodeStatus, number>();
		for (const n of selectedNodes) {
			counts.set(n.status, (counts.get(n.status) ?? 0) + 1);
		}
		return [...counts.entries()].sort((a, b) => b[1] - a[1]);
	});

	function handleAddTag() {
		const tag = newTag.trim();
		if (!tag) return;
		onbatchaddtag(tag);
		newTag = '';
	}

	function handleTagKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') { e.preventDefault(); handleAddTag(); }
	}
</script>

<!-- Summary -->
<div class="pb-3 border-b" style="border-color: var(--color-border);">
	<span class="text-sm font-semibold" style="color: var(--color-text);">{selectedNodes.length} nodes selected</span>
</div>

<!-- Distribution -->
<div class="flex flex-col gap-3">
	<!-- Type distribution -->
	<div class="flex flex-col gap-1.5">
		<span class="text-xs font-medium" style="color: var(--color-text-muted);">Types</span>
		<div class="flex flex-wrap gap-1">
			{#each typeDistribution as [type, count]}
				<span class="flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium"
					style="background: {TYPE_COLORS[type]}22; color: {TYPE_COLORS[type]};"
				>
					<span class="w-1.5 h-1.5 rounded-full" style="background: {TYPE_COLORS[type]};"></span>
					{type} ×{count}
				</span>
			{/each}
		</div>
	</div>

	<!-- Status distribution -->
	<div class="flex flex-col gap-1.5">
		<span class="text-xs font-medium" style="color: var(--color-text-muted);">Status</span>
		<div class="flex flex-wrap gap-1">
			{#each statusDistribution as [status, count]}
				<span class="flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium"
					style="background: {STATUS_COLORS[status]}22; color: {STATUS_COLORS[status]};"
				>
					<span class="w-1.5 h-1.5 rounded-full" style="background: {STATUS_COLORS[status]};"></span>
					{STATUS_LABELS[status]} ×{count}
				</span>
			{/each}
		</div>
	</div>
</div>

<!-- Batch actions -->
<div class="flex flex-col gap-3 pt-3 border-t" style="border-color: var(--color-border);">
	<!-- Set status -->
	<div class="flex flex-col gap-1.5">
		<span class="text-xs font-medium" style="color: var(--color-text-muted);">Set Status</span>
		<div class="flex gap-1">
			{#each STATUS_OPTIONS as status}
				<button
					onclick={() => onbatchstatus(status)}
					class="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 rounded text-[11px] font-medium transition-colors"
					style="background: var(--color-bg); color: {STATUS_COLORS[status]}; border: 1px solid var(--color-border);"
					onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = STATUS_COLORS[status] + '22'; (e.currentTarget as HTMLElement).style.borderColor = STATUS_COLORS[status]; }}
					onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--color-bg)'; (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
				>
					<span class="w-2 h-2 rounded-full" style="background: {STATUS_COLORS[status]};"></span>
					{STATUS_LABELS[status]}
				</button>
			{/each}
		</div>
	</div>

	<!-- Set type -->
	<div class="flex flex-col gap-1.5">
		<span class="text-xs font-medium" style="color: var(--color-text-muted);">Set Type</span>
		<div class="flex flex-wrap gap-1">
			{#each NODE_TYPES_NO_GROUP as type}
				<button
					onclick={() => onbatchtype(type)}
					class="flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium transition-colors"
					style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
					onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = TYPE_COLORS[type] + '22'; (e.currentTarget as HTMLElement).style.borderColor = TYPE_COLORS[type]; (e.currentTarget as HTMLElement).style.color = TYPE_COLORS[type]; }}
					onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--color-bg)'; (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; (e.currentTarget as HTMLElement).style.color = 'var(--color-text)'; }}
				>
					<span class="w-2 h-2 rounded-full" style="background: {TYPE_COLORS[type]};"></span>
					{type}
				</button>
			{/each}
		</div>
	</div>

	<!-- Add tag -->
	<div class="flex flex-col gap-1.5">
		<span class="text-xs font-medium flex items-center gap-1" style="color: var(--color-text-muted);">
			<Tag size={11} /> Add Tag
		</span>
		<div class="flex gap-1">
			<input bind:value={newTag} onkeydown={handleTagKeydown} placeholder="Tag name..." class="flex-1 px-2 py-1 rounded text-xs outline-none" style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);" />
			<button onclick={handleAddTag} class="px-2 py-1 rounded text-xs font-medium transition-colors" style="background: var(--color-primary); color: white;">Add</button>
		</div>
	</div>

	<!-- Create group -->
	<button
		onclick={oncreategroup}
		class="flex items-center justify-center gap-1.5 px-3 py-1.5 rounded text-xs font-medium transition-colors"
		style="background: var(--color-surface-hover); color: var(--color-text); border: 1px solid var(--color-border);"
		onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
		onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
	>
		<Group size={14} /> Create Group from Selection
	</button>

	<!-- Delete -->
	<button
		onclick={onbatchdelete}
		class="px-3 py-1.5 rounded text-xs font-medium transition-colors"
		style="background: rgba(196,64,64,0.13); color: var(--color-danger); border: 1px solid rgba(196,64,64,0.27);"
		onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(196,64,64,0.2)'; }}
		onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(196,64,64,0.13)'; }}
	>
		<Trash2 size={12} class="inline mr-1" />Delete {selectedNodes.length} Nodes
	</button>
</div>
