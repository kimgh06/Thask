<script lang="ts">
	import type { GraphNode, NodeType, NodeStatus, NodeHistoryEntry } from '$lib/types';

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

	const NODE_TYPES: NodeType[] = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP'];
	const STATUS_OPTIONS: NodeStatus[] = ['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED'];

	const TYPE_COLORS: Record<NodeType, string> = {
		FLOW: '#6366f1',
		BRANCH: '#8b5cf6',
		TASK: '#3b82f6',
		BUG: '#ef4444',
		API: '#22c55e',
		UI: '#f59e0b',
		GROUP: '#64748b',
	};

	const STATUS_COLORS: Record<NodeStatus, string> = {
		PASS: '#22c55e',
		FAIL: '#ef4444',
		IN_PROGRESS: '#6366f1',
		BLOCKED: '#f59e0b',
	};

	const STATUS_LABELS: Record<NodeStatus, string> = {
		PASS: 'Pass',
		FAIL: 'Fail',
		IN_PROGRESS: 'In Progress',
		BLOCKED: 'Blocked',
	};

	type Tab = 'details' | 'relations' | 'history';
	let activeTab = $state<Tab>('details');

	// Local editable state, synced from node prop
	let localTitle = $state('');
	let localDescription = $state('');
	let localTags = $state<string[]>([]);
	let newTag = $state('');
	let showTypeDropdown = $state(false);
	let showStatusDropdown = $state(false);

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
			class="flex-shrink-0 p-4 flex flex-col gap-2 border-b"
			style="border-color: var(--color-border);"
		>
			<div class="flex items-start justify-between gap-2">
				<!-- Title (editable) -->
				<input
					bind:value={localTitle}
					onblur={saveTitle}
					onkeydown={(e) => { if (e.key === 'Enter') (e.currentTarget as HTMLInputElement).blur(); }}
					class="flex-1 text-base font-semibold bg-transparent outline-none rounded px-1 -mx-1"
					style="color: var(--color-text); min-width: 0;"
				/>
				<!-- Close button -->
				<button
					onclick={onclose}
					class="flex-shrink-0 w-7 h-7 flex items-center justify-center rounded text-sm transition-colors"
					style="color: var(--color-text-muted);"
					onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text)'; }}
					onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-text-muted)'; }}
					aria-label="Close panel"
				>
					✕
				</button>
			</div>

			<!-- Type + Status badges -->
			<div class="flex items-center gap-2">
				<!-- Type dropdown -->
				<div class="relative">
					<button
						onclick={() => { showTypeDropdown = !showTypeDropdown; showStatusDropdown = false; }}
						class="px-2 py-0.5 rounded text-xs font-medium"
						style="background: {TYPE_COLORS[node.type]}22; color: {TYPE_COLORS[node.type]}; border: 1px solid {TYPE_COLORS[node.type]}44;"
					>
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
						onclick={() => { showStatusDropdown = !showStatusDropdown; showTypeDropdown = false; }}
						class="px-2 py-0.5 rounded text-xs font-medium"
						style="background: {STATUS_COLORS[node.status]}22; color: {STATUS_COLORS[node.status]}; border: 1px solid {STATUS_COLORS[node.status]}44;"
					>
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
			{#each (['details', 'relations', 'history'] as Tab[]) as tab}
				<button
					onclick={() => (activeTab = tab)}
					class="flex-1 py-2 text-xs font-medium capitalize transition-colors"
					style="
						color: {activeTab === tab ? 'var(--color-primary)' : 'var(--color-text-muted)'};
						border-bottom: 2px solid {activeTab === tab ? 'var(--color-primary)' : 'transparent'};
					"
				>
					{tab}
				</button>
			{/each}
		</div>

		<!-- Tab content (scrollable) -->
		<div class="flex-1 overflow-y-auto p-4 flex flex-col gap-4">
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
						class="px-3 py-2 rounded-lg text-sm outline-none resize-none"
						style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
					></textarea>
				</div>

				<!-- Tags -->
				<div class="flex flex-col gap-2">
					<span class="text-xs font-medium" style="color: var(--color-text-muted);">Tags</span>
					{#if localTags.length > 0}
						<div class="flex flex-wrap gap-1">
							{#each localTags as tag}
								<span
									class="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
									style="background: var(--color-surface-hover); color: var(--color-text);"
								>
									{tag}
									<button
										onclick={() => removeTag(tag)}
										class="ml-0.5 leading-none"
										style="color: var(--color-text-muted);"
										aria-label="Remove tag {tag}"
									>
										✕
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

				<!-- Metadata -->
				<div class="flex flex-col gap-1">
					<span class="text-xs font-medium" style="color: var(--color-text-muted);">Created</span>
					<span class="text-xs" style="color: var(--color-text-muted);">{formatDate(node.createdAt)}</span>
				</div>
				<div class="flex flex-col gap-1">
					<span class="text-xs font-medium" style="color: var(--color-text-muted);">Updated</span>
					<span class="text-xs" style="color: var(--color-text-muted);">{formatDate(node.updatedAt)}</span>
				</div>

			{:else if activeTab === 'relations'}
				<!-- Connected nodes -->
				<div class="flex flex-col gap-2">
					<span class="text-xs font-medium" style="color: var(--color-text-muted);">
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
								<span class="text-xs" style="color: var(--color-text-muted);">{formatDate(entry.createdAt)}</span>
							</div>
						{/each}
					</div>
				{/if}
			{/if}
		</div>

		<!-- Delete button -->
		<div class="flex-shrink-0 p-4 border-t" style="border-color: var(--color-border);">
			<button
				onclick={handleDelete}
				class="w-full py-2 px-4 rounded-lg text-sm font-medium transition-colors"
				style="background: #ef444411; color: #ef4444; border: 1px solid #ef444433;"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = '#ef444422'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = '#ef444411'; }}
			>
				Delete Node
			</button>
		</div>
	</aside>
{/if}
