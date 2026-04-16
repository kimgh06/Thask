<script lang="ts">
	import type { GraphNode, GraphEdge, NodeType, NodeStatus, NodeHistoryEntry, NodeDetail } from '$lib/types';
	import { NODE_TYPES, STATUS_OPTIONS, TYPE_COLORS, STATUS_COLORS, STATUS_LABELS } from '$lib/constants';
	import { Trash2, Clock, Tag, Link2, History, FileText, MoreHorizontal, Cable, ArrowUpToLine, ArrowDownToLine } from 'lucide-svelte';
	import MarkdownEditor from '$lib/components/MarkdownEditor.svelte';

	interface Props {
		node: NodeDetail;
		allNodes: GraphNode[];
		allEdges?: GraphEdge[];
		onupdate: (nodeId: string, data: Record<string, unknown>) => void;
		ondelete: (nodeId: string) => void;
		onselectnode: (nodeId: string) => void;
		onstartedge?: (nodeId: string) => void;
		readonly?: boolean;
	}

	let { node, allNodes, allEdges, onupdate, ondelete, onselectnode, onstartedge, readonly = false }: Props = $props();

	type Tab = 'details' | 'relations' | 'history';
	let activeTab = $state<Tab>('details');
	let lastNodeId = $state('');

	let localTitle = $state('');
	let localDescription = $state('');
	let localTags = $state<string[]>([]);
	let newTag = $state('');
	let showTypeDropdown = $state(false);
	let showStatusDropdown = $state(false);

	$effect(() => {
		if (node) {
			localTitle = node.title;
			localDescription = node.description ?? '';
			localTags = [...node.tags];
		}
	});

	$effect(() => {
		if (node && node.id !== lastNodeId) {
			lastNodeId = node.id;
			activeTab = 'details';
			showTypeDropdown = false;
			showStatusDropdown = false;
		}
	});

	function saveTitle() {
		if (!node || localTitle.trim() === node.title) return;
		const trimmed = localTitle.trim();
		if (!trimmed) { localTitle = node.title; return; }
		onupdate(node.id, { title: trimmed });
	}

	function saveDescription() {
		if (!node || localDescription === (node.description ?? '')) return;
		onupdate(node.id, { description: localDescription });
	}

	function setType(type: NodeType) {
		if (!node) return;
		showTypeDropdown = false;
		onupdate(node.id, { type });
	}

	function setStatus(status: NodeStatus) {
		if (!node) return;
		showStatusDropdown = false;
		onupdate(node.id, { status });
	}

	function addTag() {
		const tag = newTag.trim();
		if (!tag || !node || localTags.includes(tag)) return;
		const updated = [...localTags, tag];
		localTags = updated;
		newTag = '';
		onupdate(node.id, { tags: updated });
	}

	function removeTag(tag: string) {
		if (!node) return;
		const updated = localTags.filter((t) => t !== tag);
		localTags = updated;
		onupdate(node.id, { tags: updated });
	}

	function handleTagKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') { e.preventDefault(); addTag(); }
	}

	function handleDelete() {
		if (!node) return;
		if (confirm(`Delete node "${node.title}"? This cannot be undone.`)) {
			ondelete(node.id);
		}
	}

	function formatDate(iso: string) {
		try { return new Date(iso).toLocaleString(); } catch { return iso; }
	}

	let connectedNodes = $derived(
		allNodes.filter((n) => node.connectedNodeIds.includes(n.id) && n.id !== node.id),
	);
	let childNodes = $derived(allNodes.filter((n) => n.parentId === node.id));

	// Use connectedEdges from node detail (always available) or fall back to allEdges prop
	let edgeSource = $derived(node.connectedEdges?.length > 0 ? node.connectedEdges : (allEdges ?? []));
	let hasEdgeDirection = $derived(edgeSource.length > 0);

	// Upstream: edges where this node is the target (this node depends on the source)
	let upstreamNodes = $derived(
		hasEdgeDirection
			? allNodes.filter((n) =>
					edgeSource.some((e) => e.sourceId === n.id && e.targetId === node.id) && n.id !== node.id
				)
			: [],
	);

	// Downstream: edges where this node is the source (downstream nodes depend on this node)
	let downstreamNodes = $derived(
		hasEdgeDirection
			? allNodes.filter((n) =>
					edgeSource.some((e) => e.sourceId === node.id && e.targetId === n.id) && n.id !== node.id
				)
			: [],
	);
</script>

<!-- Header: Type + Status badges + actions -->
<div class="flex flex-col gap-2 pb-3 border-b" style="border-color: var(--color-border);">
	{#if readonly}
		<p class="text-sm font-semibold px-1 -mx-1" style="color: var(--color-text);">{node.title}</p>
	{:else}
		<input
			bind:value={localTitle}
			onblur={saveTitle}
			onkeydown={(e) => { if (e.key === 'Enter') (e.currentTarget as HTMLInputElement).blur(); }}
			class="text-sm font-semibold bg-transparent outline-none rounded px-1 -mx-1 w-full"
			style="color: var(--color-text);"
		/>
	{/if}
	<div class="flex items-center gap-1.5 flex-wrap">
		<!-- Type dropdown -->
		<div class="relative">
			<button
				onclick={() => { if (!readonly) { showTypeDropdown = !showTypeDropdown; showStatusDropdown = false; } }}
				class="flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium"
				style="background: {TYPE_COLORS[node.type]}22; color: {TYPE_COLORS[node.type]}; border: 1px solid {TYPE_COLORS[node.type]}44; cursor: {readonly ? 'default' : 'pointer'};"
			>
				<span class="w-1.5 h-1.5 rounded-full flex-shrink-0" style="background: {TYPE_COLORS[node.type]};"></span>
				{node.type}
			</button>
			{#if showTypeDropdown}
				<div class="absolute left-0 top-full mt-1 rounded-lg shadow-xl z-50 py-1 min-w-[120px]" style="background: var(--color-surface); border: 1px solid var(--color-border);">
					{#each NODE_TYPES as t}
						<button
							onclick={() => setType(t)}
							class="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left transition-colors"
							style="color: {node.type === t ? TYPE_COLORS[t] : 'var(--color-text)'};"
							onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
							onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = ''; }}
						>
							<span class="w-2 h-2 rounded-full" style="background: {TYPE_COLORS[t]};"></span>
							{t}
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Status dropdown -->
		<div class="relative">
			<button
				onclick={() => { if (!readonly) { showStatusDropdown = !showStatusDropdown; showTypeDropdown = false; } }}
				class="flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium"
				style="background: {STATUS_COLORS[node.status]}22; color: {STATUS_COLORS[node.status]}; border: 1px solid {STATUS_COLORS[node.status]}44; cursor: {readonly ? 'default' : 'pointer'};"
			>
				<span class="w-1.5 h-1.5 rounded-full flex-shrink-0" style="background: {STATUS_COLORS[node.status]};"></span>
				{STATUS_LABELS[node.status]}
			</button>
			{#if showStatusDropdown}
				<div class="absolute left-0 top-full mt-1 rounded-lg shadow-xl z-50 py-1 min-w-[130px]" style="background: var(--color-surface); border: 1px solid var(--color-border);">
					{#each STATUS_OPTIONS as s}
						<button
							onclick={() => setStatus(s)}
							class="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left transition-colors"
							style="color: {node.status === s ? STATUS_COLORS[s] : 'var(--color-text)'};"
							onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
							onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = ''; }}
						>
							<span class="w-2 h-2 rounded-full" style="background: {STATUS_COLORS[s]};"></span>
							{STATUS_LABELS[s]}
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Connect from button -->
		{#if !readonly && onstartedge && node.type !== 'GROUP'}
			<button
				onclick={() => onstartedge?.(node.id)}
				class="flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors"
				style="background: var(--color-surface-hover); color: var(--color-text-muted);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text-muted)'; }}
				title="Draw edge from this node"
			>
				<Cable size={12} />
				Connect
			</button>
		{/if}
	</div>
</div>

<!-- Tabs -->
<div class="flex border-b -mx-3" style="border-color: var(--color-border);">
	<button
		onclick={() => (activeTab = 'details')}
		class="flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors"
		style="color: {activeTab === 'details' ? 'var(--color-primary)' : 'var(--color-text-muted)'}; border-bottom: 2px solid {activeTab === 'details' ? 'var(--color-primary)' : 'transparent'};"
	>
		<FileText size={12} /> Details
	</button>
	<button
		onclick={() => (activeTab = 'relations')}
		class="flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors"
		style="color: {activeTab === 'relations' ? 'var(--color-primary)' : 'var(--color-text-muted)'}; border-bottom: 2px solid {activeTab === 'relations' ? 'var(--color-primary)' : 'transparent'};"
	>
		<Link2 size={12} /> Relations
		{#if connectedNodes.length > 0}
			<span class="px-1.5 py-0.5 rounded-full text-[10px] font-medium leading-none" style="background: var(--color-surface-hover); color: var(--color-text-muted);">{connectedNodes.length}</span>
		{/if}
	</button>
	<button
		onclick={() => (activeTab = 'history')}
		class="flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors"
		style="color: {activeTab === 'history' ? 'var(--color-primary)' : 'var(--color-text-muted)'}; border-bottom: 2px solid {activeTab === 'history' ? 'var(--color-primary)' : 'transparent'};"
	>
		<History size={12} /> History
		{#if node.history.length > 0}
			<span class="px-1.5 py-0.5 rounded-full text-[10px] font-medium leading-none" style="background: var(--color-surface-hover); color: var(--color-text-muted);">{node.history.length}</span>
		{/if}
	</button>
</div>

<!-- Tab content -->
<div class="flex-1 overflow-y-auto flex flex-col gap-3 pt-3 -mx-3 px-3">
	{#if activeTab === 'details'}
		<div class="flex flex-col gap-1">
			<span class="text-xs font-medium" style="color: var(--color-text-muted);">Description</span>
			<MarkdownEditor
				bind:value={localDescription}
				onsave={(v) => { localDescription = v; saveDescription(); }}
				{readonly}
				placeholder={readonly ? 'No description' : 'Add a description...'}
			/>
		</div>

		<div class="flex flex-col gap-1.5">
			<span class="text-xs font-medium flex items-center gap-1" style="color: var(--color-text-muted);">
				<Tag size={11} /> Tags
			</span>
			{#if localTags.length > 0}
				<div class="flex flex-wrap gap-1">
					{#each localTags as tag}
						<span class="flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[11px] font-medium" style="background: var(--color-surface-hover); color: var(--color-text);">
							{tag}
							{#if !readonly}<button onclick={() => removeTag(tag)} class="leading-none opacity-60 hover:opacity-100" style="color: var(--color-text-muted);" aria-label="Remove tag {tag}">×</button>{/if}
						</span>
					{/each}
				</div>
			{/if}
			{#if !readonly}
				<div class="flex gap-1">
					<input bind:value={newTag} onkeydown={handleTagKeydown} placeholder="Add tag..." class="flex-1 px-2 py-1 rounded text-xs outline-none" style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);" />
					<button onclick={addTag} class="px-2 py-1 rounded text-xs font-medium transition-colors" style="background: var(--color-primary); color: white;">Add</button>
				</div>
			{/if}
		</div>

		<div class="grid grid-cols-2 gap-2">
			<div class="flex flex-col gap-0.5">
				<span class="text-[10px] font-medium flex items-center gap-1" style="color: var(--color-text-muted);"><Clock size={10} /> Created</span>
				<span class="text-[11px]" style="color: var(--color-text-muted);">{formatDate(node.createdAt)}</span>
			</div>
			<div class="flex flex-col gap-0.5">
				<span class="text-[10px] font-medium flex items-center gap-1" style="color: var(--color-text-muted);"><Clock size={10} /> Updated</span>
				<span class="text-[11px]" style="color: var(--color-text-muted);">{formatDate(node.updatedAt)}</span>
			</div>
		</div>

		{#if !readonly}
			<button
				onclick={handleDelete}
				class="mt-1 px-3 py-1.5 rounded text-xs font-medium transition-colors"
				style="background: rgba(196,64,64,0.13); color: var(--color-danger); border: 1px solid rgba(196,64,64,0.27);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(196,64,64,0.2)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(196,64,64,0.13)'; }}
			>
				<Trash2 size={12} class="inline mr-1" />Delete Node
			</button>
		{/if}

	{:else if activeTab === 'relations'}
		{#if hasEdgeDirection}
			<!-- Upstream group -->
			<div class="flex flex-col gap-2">
				<span class="text-xs font-medium flex items-center gap-1.5" style="color: var(--color-text-muted);">
					<ArrowUpToLine size={12} /> Upstream ({upstreamNodes.length})
				</span>
				{#if upstreamNodes.length === 0}
					<p class="text-xs" style="color: var(--color-text-muted);">No upstream nodes.</p>
				{:else}
					{#each upstreamNodes as n}
						<button
							onclick={() => onselectnode(n.id)}
							class="flex items-center gap-2 px-3 py-2 rounded-lg text-left transition-colors"
							style="background: var(--color-bg); border: 1px solid var(--color-border);"
							onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
							onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
						>
							<span class="w-2 h-2 rounded-full flex-shrink-0" style="background: {TYPE_COLORS[n.type]};"></span>
							<span class="text-xs truncate flex-1" style="color: var(--color-text);">{n.title}</span>
							<span class="text-xs flex-shrink-0" style="color: var(--color-text-muted);">{n.type}</span>
						</button>
					{/each}
				{/if}
			</div>
			<!-- Downstream group -->
			<div class="flex flex-col gap-2">
				<span class="text-xs font-medium flex items-center gap-1.5" style="color: var(--color-text-muted);">
					<ArrowDownToLine size={12} /> Downstream ({downstreamNodes.length})
				</span>
				{#if downstreamNodes.length === 0}
					<p class="text-xs" style="color: var(--color-text-muted);">No downstream nodes.</p>
				{:else}
					{#each downstreamNodes as n}
						<button
							onclick={() => onselectnode(n.id)}
							class="flex items-center gap-2 px-3 py-2 rounded-lg text-left transition-colors"
							style="background: var(--color-bg); border: 1px solid var(--color-border);"
							onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
							onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
						>
							<span class="w-2 h-2 rounded-full flex-shrink-0" style="background: {TYPE_COLORS[n.type]};"></span>
							<span class="text-xs truncate flex-1" style="color: var(--color-text);">{n.title}</span>
							<span class="text-xs flex-shrink-0" style="color: var(--color-text-muted);">{n.type}</span>
						</button>
					{/each}
				{/if}
			</div>
		{:else}
			<!-- Fallback: flat connected list when no edge direction info -->
			<div class="flex flex-col gap-2">
				<span class="text-xs font-medium flex items-center gap-1.5" style="color: var(--color-text-muted);">
					<Link2 size={12} /> Connected Nodes ({connectedNodes.length})
				</span>
				{#if connectedNodes.length === 0}
					<p class="text-xs" style="color: var(--color-text-muted);">No connected nodes.</p>
				{:else}
					{#each connectedNodes as n}
						<button
							onclick={() => onselectnode(n.id)}
							class="flex items-center gap-2 px-3 py-2 rounded-lg text-left transition-colors"
							style="background: var(--color-bg); border: 1px solid var(--color-border);"
							onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
							onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
						>
							<span class="w-2 h-2 rounded-full flex-shrink-0" style="background: {TYPE_COLORS[n.type]};"></span>
							<span class="text-xs truncate flex-1" style="color: var(--color-text);">{n.title}</span>
							<span class="text-xs flex-shrink-0" style="color: var(--color-text-muted);">{n.type}</span>
						</button>
					{/each}
				{/if}
			</div>
		{/if}

		{#if node.type === 'GROUP' && childNodes.length > 0}
			<div class="flex flex-col gap-2">
				<span class="text-xs font-medium" style="color: var(--color-text-muted);">Children ({childNodes.length})</span>
				{#each childNodes as n}
					<button
						onclick={() => onselectnode(n.id)}
						class="flex items-center gap-2 px-3 py-2 rounded-lg text-left transition-colors"
						style="background: var(--color-bg); border: 1px solid var(--color-border);"
						onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-primary)'; }}
						onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
					>
						<span class="w-2 h-2 rounded-full flex-shrink-0" style="background: {TYPE_COLORS[n.type]};"></span>
						<span class="text-xs truncate flex-1" style="color: var(--color-text);">{n.title}</span>
						<span class="text-xs flex-shrink-0" style="color: var(--color-text-muted);">{n.type}</span>
					</button>
				{/each}
			</div>
		{/if}

	{:else if activeTab === 'history'}
		<div class="flex items-center gap-1.5 mb-1">
			<History size={12} style="color: var(--color-text-muted);" />
			<span class="text-xs font-medium" style="color: var(--color-text-muted);">Change History</span>
		</div>
		{#if node.history.length === 0}
			<p class="text-xs" style="color: var(--color-text-muted);">No history available.</p>
		{:else}
			<div class="flex flex-col gap-2">
				{#each node.history as entry}
					<div class="px-3 py-2 rounded-lg flex flex-col gap-1" style="background: var(--color-bg); border: 1px solid var(--color-border);">
						<div class="flex items-center justify-between gap-2">
							<span class="text-xs font-medium" style="color: var(--color-text);">
								{entry.action}
								{#if entry.fieldName}
									<span style="color: var(--color-text-muted);">· {entry.fieldName}</span>
								{/if}
							</span>
							<span class="text-xs flex-shrink-0" style="color: var(--color-text-muted);">{entry.userName}</span>
						</div>
						{#if entry.fieldName === 'description'}
							<div class="text-xs" style="color: var(--color-text-muted);">updated description</div>
						{:else if entry.oldValue !== null || entry.newValue !== null}
							<div class="flex items-center gap-1 text-xs" style="color: var(--color-text-muted);">
								{#if entry.oldValue !== null}<span class="line-through">{entry.oldValue}</span><span>→</span>{/if}
								{#if entry.newValue !== null}<span style="color: var(--color-text);">{entry.newValue}</span>{/if}
							</div>
						{/if}
						<span class="text-[11px] flex items-center gap-1" style="color: var(--color-text-muted);">
							<Clock size={10} /> {formatDate(entry.createdAt)}
						</span>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>
