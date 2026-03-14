<script lang="ts">
	import type { GraphNode, NodeType, NodeStatus, NodeHistoryEntry } from '$lib/types';
	import { NODE_TYPES, STATUS_OPTIONS, TYPE_COLORS, STATUS_COLORS, STATUS_LABELS } from '$lib/constants';
	import { X, Trash2, Clock, Tag, Link2, History, FileText, MoreHorizontal } from 'lucide-svelte';

	interface Props {
		node: GraphNode | null;
		allNodes: GraphNode[];
		history: NodeHistoryEntry[];
		connectedNodeIds: string[];
		isOpen: boolean;
		onclose: () => void;
		onupdate: (nodeId: string, data: Record<string, unknown>) => void;
		ondelete: (nodeId: string) => void;
		onselectnode: (nodeId: string) => void;
	}

	let {
		node,
		allNodes,
		history,
		connectedNodeIds,
		isOpen,
		onclose,
		onupdate,
		ondelete,
		onselectnode,
	}: Props = $props();

	type Tab = 'details' | 'relations' | 'history';
	let activeTab = $state<Tab>('details');

	// Local editable state, synced from node prop
	let localTitle = $state('');
	let localDescription = $state('');
	let localTags = $state<string[]>([]);
	let newTag = $state('');
	let showTypeDropdown = $state(false);
	let showStatusDropdown = $state(false);
	let showMoreMenu = $state(false);

	// Sync local state when node changes
	$effect(() => {
		if (node) {
			localTitle = node.title;
			localDescription = node.description ?? '';
			localTags = [...node.tags];
		}
	});

	// Reset tab to details when panel opens with a new node
	$effect(() => {
		if (node) {
			activeTab = 'details';
			showTypeDropdown = false;
			showStatusDropdown = false;
			showMoreMenu = false;
		}
	});

	function saveTitle() {
		if (!node || localTitle.trim() === node.title) return;
		const trimmed = localTitle.trim();
		if (!trimmed) {
			localTitle = node.title;
			return;
		}
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
		if (e.key === 'Enter') {
			e.preventDefault();
			addTag();
		}
	}

	function handleDelete() {
		if (!node) return;
		showMoreMenu = false;
		if (confirm(`Delete node "${node.title}"? This cannot be undone.`)) {
			ondelete(node.id);
		}
	}

	function formatDate(iso: string) {
		try {
			return new Date(iso).toLocaleString();
		} catch {
			return iso;
		}
	}

	let connectedNodes = $derived(
		allNodes.filter((n) => connectedNodeIds.includes(n.id) && n.id !== node?.id),
	);

	let childNodes = $derived(allNodes.filter((n) => n.parentId === node?.id));
</script>

<!-- Panel overlay backdrop (clicking outside closes) -->
{#if isOpen && node}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-30"
		style="background: transparent;"
		onclick={(e) => {
			if (e.target === e.currentTarget) onclose();
		}}
	></div>

	<!-- Slide-out panel -->
	<aside
		class="fixed right-0 top-0 h-full z-40 flex flex-col shadow-2xl"
		style="
			width: 360px;
			background: var(--color-surface);
			border-left: 1px solid var(--color-border);
			overflow: hidden;
		"
	>
		<!-- Header -->
		<div
			class="flex-shrink-0 px-3 py-2.5 flex flex-col gap-2 border-b"
			style="border-color: var(--color-border);"
		>
			<div class="flex items-center gap-1.5">
				<!-- Title (editable) -->
				<input
					bind:value={localTitle}
					onblur={saveTitle}
					onkeydown={(e) => { if (e.key === 'Enter') (e.currentTarget as HTMLInputElement).blur(); }}
					class="flex-1 text-sm font-semibold bg-transparent outline-none rounded px-1 -mx-1"
					style="color: var(--color-text); min-width: 0;"
				/>

				<!-- More menu button -->
				<div class="relative flex-shrink-0">
					<button
						onclick={() => { showMoreMenu = !showMoreMenu; showTypeDropdown = false; showStatusDropdown = false; }}
						class="w-6 h-6 flex items-center justify-center rounded transition-colors"
						style="color: var(--color-text-muted);"
						onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text)'; (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
						onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text-muted)'; (e.currentTarget as HTMLElement).style.background = ''; }}
						aria-label="More options"
					>
						<MoreHorizontal size={15} />
					</button>
					{#if showMoreMenu}
						<div
							class="absolute right-0 top-full mt-1 rounded-lg shadow-xl z-50 py-1 min-w-[140px]"
							style="background: var(--color-surface); border: 1px solid var(--color-border);"
						>
							<button
								onclick={handleDelete}
								class="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left transition-colors"
								style="color: #ef4444;"
								onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
								onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = ''; }}
							>
								<Trash2 size={13} />
								Delete Node
							</button>
						</div>
					{/if}
				</div>

				<!-- Close button -->
				<button
					onclick={onclose}
					class="flex-shrink-0 w-6 h-6 flex items-center justify-center rounded transition-colors"
					style="color: var(--color-text-muted);"
					onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text)'; (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
					onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text-muted)'; (e.currentTarget as HTMLElement).style.background = ''; }}
					aria-label="Close panel"
				>
					<X size={16} />
				</button>
			</div>

			<!-- Type + Status badges -->
			<div class="flex items-center gap-1.5">
				<!-- Type dropdown -->
				<div class="relative">
					<button
						onclick={() => { showTypeDropdown = !showTypeDropdown; showStatusDropdown = false; showMoreMenu = false; }}
						class="flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium"
						style="background: {TYPE_COLORS[node.type]}22; color: {TYPE_COLORS[node.type]}; border: 1px solid {TYPE_COLORS[node.type]}44;"
					>
						<span class="w-1.5 h-1.5 rounded-full flex-shrink-0" style="background: {TYPE_COLORS[node.type]};"></span>
						{node.type}
					</button>
					{#if showTypeDropdown}
						<div
							class="absolute left-0 top-full mt-1 rounded-lg shadow-xl z-50 py-1 min-w-[120px]"
							style="background: var(--color-surface); border: 1px solid var(--color-border);"
						>
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
						onclick={() => { showStatusDropdown = !showStatusDropdown; showTypeDropdown = false; showMoreMenu = false; }}
						class="flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium"
						style="background: {STATUS_COLORS[node.status]}22; color: {STATUS_COLORS[node.status]}; border: 1px solid {STATUS_COLORS[node.status]}44;"
					>
						<span class="w-1.5 h-1.5 rounded-full flex-shrink-0" style="background: {STATUS_COLORS[node.status]};"></span>
						{STATUS_LABELS[node.status]}
					</button>
					{#if showStatusDropdown}
						<div
							class="absolute left-0 top-full mt-1 rounded-lg shadow-xl z-50 py-1 min-w-[130px]"
							style="background: var(--color-surface); border: 1px solid var(--color-border);"
						>
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
			</div>
		</div>

		<!-- Tabs -->
		<div class="flex-shrink-0 flex border-b" style="border-color: var(--color-border);">
			<button
				onclick={() => (activeTab = 'details')}
				class="flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors"
				style="
					color: {activeTab === 'details' ? 'var(--color-primary)' : 'var(--color-text-muted)'};
					border-bottom: 2px solid {activeTab === 'details' ? 'var(--color-primary)' : 'transparent'};
				"
			>
				<FileText size={12} />
				Details
			</button>
			<button
				onclick={() => (activeTab = 'relations')}
				class="flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors"
				style="
					color: {activeTab === 'relations' ? 'var(--color-primary)' : 'var(--color-text-muted)'};
					border-bottom: 2px solid {activeTab === 'relations' ? 'var(--color-primary)' : 'transparent'};
				"
			>
				<Link2 size={12} />
				Relations
				{#if connectedNodes.length > 0}
					<span
						class="px-1.5 py-0.5 rounded-full text-[10px] font-medium leading-none"
						style="background: var(--color-surface-hover); color: var(--color-text-muted);"
					>{connectedNodes.length}</span>
				{/if}
			</button>
			<button
				onclick={() => (activeTab = 'history')}
				class="flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors"
				style="
					color: {activeTab === 'history' ? 'var(--color-primary)' : 'var(--color-text-muted)'};
					border-bottom: 2px solid {activeTab === 'history' ? 'var(--color-primary)' : 'transparent'};
				"
			>
				<History size={12} />
				History
				{#if history.length > 0}
					<span
						class="px-1.5 py-0.5 rounded-full text-[10px] font-medium leading-none"
						style="background: var(--color-surface-hover); color: var(--color-text-muted);"
					>{history.length}</span>
				{/if}
			</button>
		</div>

		<!-- Tab content (scrollable) -->
		<div class="flex-1 overflow-y-auto p-3 flex flex-col gap-3">
			{#if activeTab === 'details'}
				<!-- Description -->
				<div class="flex flex-col gap-1">
					<label class="text-xs font-medium" style="color: var(--color-text-muted);" for="node-description">
						Description
					</label>
					<textarea
						id="node-description"
						bind:value={localDescription}
						onblur={saveDescription}
						placeholder="Add a description..."
						rows="4"
						class="px-3 py-2 rounded-lg text-xs outline-none resize-none"
						style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
					></textarea>
				</div>

				<!-- Tags -->
				<div class="flex flex-col gap-1.5">
					<span class="text-xs font-medium flex items-center gap-1" style="color: var(--color-text-muted);">
						<Tag size={11} />
						Tags
					</span>
					{#if localTags.length > 0}
						<div class="flex flex-wrap gap-1">
							{#each localTags as tag}
								<span
									class="flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[11px] font-medium"
									style="background: var(--color-surface-hover); color: var(--color-text);"
								>
									{tag}
									<button
										onclick={() => removeTag(tag)}
										class="leading-none opacity-60 hover:opacity-100"
										style="color: var(--color-text-muted);"
										aria-label="Remove tag {tag}"
									>
										<X size={10} />
									</button>
								</span>
							{/each}
						</div>
					{/if}
					<div class="flex gap-1">
						<input
							bind:value={newTag}
							onkeydown={handleTagKeydown}
							placeholder="Add tag..."
							class="flex-1 px-2 py-1 rounded text-xs outline-none"
							style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
						/>
						<button
							onclick={addTag}
							class="px-2 py-1 rounded text-xs font-medium transition-colors"
							style="background: var(--color-primary); color: white;"
						>
							Add
						</button>
					</div>
				</div>

				<!-- Metadata: 2-column grid -->
				<div class="grid grid-cols-2 gap-2">
					<div class="flex flex-col gap-0.5">
						<span class="text-[10px] font-medium flex items-center gap-1" style="color: var(--color-text-muted);">
							<Clock size={10} />
							Created
						</span>
						<span class="text-[11px]" style="color: var(--color-text-muted);">{formatDate(node.createdAt)}</span>
					</div>
					<div class="flex flex-col gap-0.5">
						<span class="text-[10px] font-medium flex items-center gap-1" style="color: var(--color-text-muted);">
							<Clock size={10} />
							Updated
						</span>
						<span class="text-[11px]" style="color: var(--color-text-muted);">{formatDate(node.updatedAt)}</span>
					</div>
				</div>

			{:else if activeTab === 'relations'}
				<!-- Connected nodes -->
				<div class="flex flex-col gap-2">
					<span class="text-xs font-medium flex items-center gap-1.5" style="color: var(--color-text-muted);">
						<Link2 size={12} />
						Connected Nodes ({connectedNodes.length})
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

				<!-- Group children -->
				{#if node.type === 'GROUP' && childNodes.length > 0}
					<div class="flex flex-col gap-2">
						<span class="text-xs font-medium" style="color: var(--color-text-muted);">
							Children ({childNodes.length})
						</span>
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
				{#if history.length === 0}
					<p class="text-xs" style="color: var(--color-text-muted);">No history available.</p>
				{:else}
					<div class="flex flex-col gap-2">
						{#each history as entry}
							<div
								class="px-3 py-2 rounded-lg flex flex-col gap-1"
								style="background: var(--color-bg); border: 1px solid var(--color-border);"
							>
								<div class="flex items-center justify-between gap-2">
									<span class="text-xs font-medium" style="color: var(--color-text);">
										{entry.action}
										{#if entry.fieldName}
											<span style="color: var(--color-text-muted);">· {entry.fieldName}</span>
										{/if}
									</span>
									<span class="text-xs flex-shrink-0" style="color: var(--color-text-muted);">{entry.userName}</span>
								</div>
								{#if entry.oldValue !== null || entry.newValue !== null}
									<div class="flex items-center gap-1 text-xs" style="color: var(--color-text-muted);">
										{#if entry.oldValue !== null}
											<span class="line-through">{entry.oldValue}</span>
											<span>→</span>
										{/if}
										{#if entry.newValue !== null}
											<span style="color: var(--color-text);">{entry.newValue}</span>
										{/if}
									</div>
								{/if}
								<span class="text-[11px] flex items-center gap-1" style="color: var(--color-text-muted);">
									<Clock size={10} />
									{formatDate(entry.createdAt)}
								</span>
							</div>
						{/each}
					</div>
				{/if}
			{/if}
		</div>
	</aside>
{/if}
