<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { GraphNode, NodeType, NodeStatus } from '$lib/types';

	interface Props {
		onAddNode: () => void;
		onAddGroup: () => void;
		onZoomIn: () => void;
		onZoomOut: () => void;
		onFitView: () => void;
		onRunLayout: () => void;
		onToggleImpact: () => void;
		isImpactActive: boolean;
		nodes: GraphNode[];
		onFocusNode: (nodeId: string) => void;
		onUndo: () => void;
		onRedo: () => void;
		canUndo: boolean;
		canRedo: boolean;
	}

	let {
		onAddNode,
		onAddGroup,
		onZoomIn,
		onZoomOut,
		onFitView,
		onRunLayout,
		onToggleImpact,
		isImpactActive,
		nodes,
		onFocusNode,
		onUndo,
		onRedo,
		canUndo,
		canRedo,
	}: Props = $props();

	const NODE_TYPES: NodeType[] = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP'];
	const STATUS_OPTIONS: { value: NodeStatus; color: string }[] = [
		{ value: 'PASS', color: '#22c55e' },
		{ value: 'FAIL', color: '#ef4444' },
		{ value: 'IN_PROGRESS', color: '#6366f1' },
		{ value: 'BLOCKED', color: '#f59e0b' },
	];

	let showFilters = $state(false);
	let activeTypeFilter = $state<NodeType | null>(null);
	let activeStatusFilter = $state<NodeStatus | null>(null);

	let showSearch = $state(false);
	let searchQuery = $state('');
	let searchResults = $state<GraphNode[]>([]);
	let searchIndex = $state(0);
	let searchInput: HTMLInputElement | undefined = $state();

	let searchResultsText = $derived(
		searchResults.length > 0
			? `${searchIndex + 1}/${searchResults.length}`
			: searchQuery.length > 0
				? '0/0'
				: '',
	);

	$effect(() => {
		if (!searchQuery) {
			searchResults = [];
			searchIndex = 0;
			return;
		}
		const q = searchQuery.toLowerCase();
		searchResults = nodes.filter(
			(n) => n.title.toLowerCase().includes(q) || (n.description ?? '').toLowerCase().includes(q),
		);
		searchIndex = 0;
		if (searchResults.length > 0) {
			onFocusNode(searchResults[0].id);
		}
	});

	function cycleSearch() {
		if (searchResults.length === 0) return;
		searchIndex = (searchIndex + 1) % searchResults.length;
		onFocusNode(searchResults[searchIndex].id);
	}

	function openSearch() {
		showSearch = true;
		setTimeout(() => searchInput?.focus(), 0);
	}

	function closeSearch() {
		showSearch = false;
		searchQuery = '';
		searchResults = [];
		searchIndex = 0;
	}

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			cycleSearch();
		} else if (e.key === 'Escape') {
			closeSearch();
		}
	}

	function handleGlobalKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
			e.preventDefault();
			openSearch();
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleGlobalKeydown);
	});

	onDestroy(() => {
		window.removeEventListener('keydown', handleGlobalKeydown);
	});
</script>

<div
	class="flex flex-col gap-1 p-2 rounded-lg border shadow-lg"
	style="background: var(--color-surface); border-color: var(--color-border);"
>
	<!-- Main toolbar row -->
	<div class="flex items-center gap-1 flex-wrap">
		<!-- Add buttons -->
		<button
			onclick={onAddNode}
			class="px-2 py-1 rounded text-xs font-medium transition-colors"
			style="background: var(--color-primary); color: white;"
			title="Add Node"
		>
			+ Node
		</button>
		<button
			onclick={onAddGroup}
			class="px-2 py-1 rounded text-xs font-medium transition-colors"
			style="background: var(--color-surface-hover); color: var(--color-text); border: 1px solid var(--color-border);"
			title="Add Group"
		>
			+ Group
		</button>

		<div class="w-px h-5 mx-1" style="background: var(--color-border);"></div>

		<!-- Zoom controls -->
		<button
			onclick={onZoomIn}
			class="w-7 h-7 flex items-center justify-center rounded text-sm font-medium transition-colors"
			style="background: var(--color-surface-hover); color: var(--color-text);"
			title="Zoom In"
		>
			+
		</button>
		<button
			onclick={onZoomOut}
			class="w-7 h-7 flex items-center justify-center rounded text-sm font-medium transition-colors"
			style="background: var(--color-surface-hover); color: var(--color-text);"
			title="Zoom Out"
		>
			−
		</button>
		<button
			onclick={onFitView}
			class="px-2 py-1 rounded text-xs font-medium transition-colors"
			style="background: var(--color-surface-hover); color: var(--color-text);"
			title="Fit View"
		>
			Fit
		</button>
		<button
			onclick={onRunLayout}
			class="px-2 py-1 rounded text-xs font-medium transition-colors"
			style="background: var(--color-surface-hover); color: var(--color-text);"
			title="Auto Layout"
		>
			Layout
		</button>

		<div class="w-px h-5 mx-1" style="background: var(--color-border);"></div>

		<!-- Undo/Redo -->
		<button
			onclick={onUndo}
			disabled={!canUndo}
			class="w-7 h-7 flex items-center justify-center rounded text-sm transition-colors"
			style="background: var(--color-surface-hover); color: {canUndo ? 'var(--color-text)' : 'var(--color-text-muted)'}; opacity: {canUndo ? '1' : '0.5'};"
			title="Undo (Ctrl+Z)"
		>
			↩
		</button>
		<button
			onclick={onRedo}
			disabled={!canRedo}
			class="w-7 h-7 flex items-center justify-center rounded text-sm transition-colors"
			style="background: var(--color-surface-hover); color: {canRedo ? 'var(--color-text)' : 'var(--color-text-muted)'}; opacity: {canRedo ? '1' : '0.5'};"
			title="Redo (Ctrl+Y)"
		>
			↪
		</button>

		<div class="w-px h-5 mx-1" style="background: var(--color-border);"></div>

		<!-- Filter toggle -->
		<button
			onclick={() => (showFilters = !showFilters)}
			class="px-2 py-1 rounded text-xs font-medium transition-colors"
			style="background: {showFilters ? 'var(--color-primary)' : 'var(--color-surface-hover)'}; color: {showFilters ? 'white' : 'var(--color-text)'};"
			title="Toggle Filters"
		>
			Filter {activeTypeFilter || activeStatusFilter ? '●' : ''}
		</button>

		<!-- Search -->
		{#if showSearch}
			<div class="flex items-center gap-1">
				<input
					bind:this={searchInput}
					bind:value={searchQuery}
					onkeydown={handleSearchKeydown}
					placeholder="Search nodes..."
					class="px-2 py-1 rounded text-xs outline-none"
					style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-primary); width: 150px;"
				/>
				{#if searchResultsText}
					<span class="text-xs" style="color: var(--color-text-muted);">{searchResultsText}</span>
				{/if}
				<button
					onclick={closeSearch}
					class="w-5 h-5 flex items-center justify-center rounded text-xs"
					style="color: var(--color-text-muted);"
					title="Close Search (Esc)"
				>
					✕
				</button>
			</div>
		{:else}
			<button
				onclick={openSearch}
				class="px-2 py-1 rounded text-xs font-medium transition-colors"
				style="background: var(--color-surface-hover); color: var(--color-text-muted);"
				title="Search (Ctrl+F)"
			>
				Search
			</button>
		{/if}

		<!-- Impact Mode -->
		<button
			onclick={onToggleImpact}
			class="px-2 py-1 rounded text-xs font-medium transition-colors"
			style="background: {isImpactActive ? '#f59e0b' : 'var(--color-surface-hover)'}; color: {isImpactActive ? '#000' : 'var(--color-text)'};"
			title="Toggle Impact Mode"
		>
			Impact
		</button>
	</div>

	<!-- Filter bar -->
	{#if showFilters}
		<div
			class="flex flex-col gap-1 pt-1 mt-1 border-t"
			style="border-color: var(--color-border);"
		>
			<!-- Node type filters -->
			<div class="flex items-center gap-1 flex-wrap">
				<span class="text-xs mr-1" style="color: var(--color-text-muted);">Type:</span>
				{#each NODE_TYPES as type}
					<button
						onclick={() => (activeTypeFilter = activeTypeFilter === type ? null : type)}
						class="px-2 py-0.5 rounded text-xs font-medium transition-colors"
						style="background: {activeTypeFilter === type ? 'var(--color-primary)' : 'var(--color-bg)'}; color: {activeTypeFilter === type ? 'white' : 'var(--color-text-muted)'}; border: 1px solid var(--color-border);"
					>
						{type}
					</button>
				{/each}
			</div>
			<!-- Status filters -->
			<div class="flex items-center gap-1 flex-wrap">
				<span class="text-xs mr-1" style="color: var(--color-text-muted);">Status:</span>
				{#each STATUS_OPTIONS as opt}
					<button
						onclick={() => (activeStatusFilter = activeStatusFilter === opt.value ? null : opt.value)}
						class="px-2 py-0.5 rounded text-xs font-medium transition-colors flex items-center gap-1"
						style="background: {activeStatusFilter === opt.value ? opt.color + '33' : 'var(--color-bg)'}; color: {activeStatusFilter === opt.value ? opt.color : 'var(--color-text-muted)'}; border: 1px solid {activeStatusFilter === opt.value ? opt.color : 'var(--color-border)'};"
					>
						<span
							class="w-2 h-2 rounded-full inline-block"
							style="background: {opt.color};"
						></span>
						{opt.value}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
