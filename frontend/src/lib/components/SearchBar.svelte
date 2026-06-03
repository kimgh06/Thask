<script lang="ts">
	import { Search, X } from 'lucide-svelte';
	import type { GraphNode, NodeStatus } from '$lib/types';
	import { NODE_TYPES, STATUS_COLORS } from '$lib/constants';
	import { graphStore } from '$lib/stores/graph.svelte';

	interface Props {
		nodes: GraphNode[];
		onFocusNode: (nodeId: string) => void;
		onOpen?: () => void;
	}

	let { nodes, onFocusNode, onOpen }: Props = $props();

	let showSearch = $state(false);
	let searchQuery = $state('');
	let searchResults = $state<GraphNode[]>([]);
	let searchIndex = $state(0);
	let searchInput: HTMLInputElement | undefined = $state();
	const STATUS_ITEMS: { value: NodeStatus; color: string }[] = [
		{ value: 'PASS', color: STATUS_COLORS.PASS },
		{ value: 'FAIL', color: STATUS_COLORS.FAIL },
		{ value: 'IN_PROGRESS', color: STATUS_COLORS.IN_PROGRESS },
		{ value: 'BLOCKED', color: STATUS_COLORS.BLOCKED },
	];

	let activeTypeFilter = $derived(graphStore.typeFilter);
	let activeStatusFilter = $derived(graphStore.statusFilter);
	let activeFilterCount = $derived((activeTypeFilter ? 1 : 0) + (activeStatusFilter ? 1 : 0));

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
			(n) =>
				(!activeTypeFilter || n.type === activeTypeFilter) &&
				(!activeStatusFilter || n.status === activeStatusFilter) &&
				(n.title.toLowerCase().includes(q) || (n.description ?? '').toLowerCase().includes(q)),
		);
		searchIndex = 0;
	});

	function clearFilters() {
		graphStore.setTypeFilter(null);
		graphStore.setStatusFilter(null);
	}

	function cycleSearch() {
		if (searchResults.length === 0) return;
		onFocusNode(searchResults[searchIndex].id);
		searchIndex = (searchIndex + 1) % searchResults.length;
	}

	export function open() {
		onOpen?.();
		showSearch = true;
		setTimeout(() => searchInput?.focus(), 0);
	}

	export function close() {
		showSearch = false;
		searchQuery = '';
		searchResults = [];
		searchIndex = 0;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			cycleSearch();
		} else if (e.key === 'Escape') {
			close();
		}
	}
</script>

<div class="search-anchor">
	<button
		onclick={open}
		class="toolbar-btn search-toggle {showSearch ? 'search-active' : ''}"
		data-tooltip="Search & Filter (⌘F)"
		aria-pressed={showSearch}
	>
		<Search size={16} />
		{#if activeFilterCount > 0}
			<span class="filter-badge">{activeFilterCount}</span>
		{/if}
	</button>
	{#if showSearch}
		<div class="search-popover">
			<div class="popover-heading">
				<span>Search & Filter</span>
				{#if activeFilterCount > 0}
					<button onclick={clearFilters} class="clear-btn">Clear</button>
				{/if}
			</div>
			<div class="search-row">
				<Search size={15} />
				<input
					bind:this={searchInput}
					bind:value={searchQuery}
					onkeydown={handleKeydown}
					placeholder="Search nodes..."
					class="search-input"
				/>
				{#if searchResultsText}
					<span class="search-count">{searchResultsText}</span>
				{/if}
				<button
					onclick={close}
					class="close-btn"
					aria-label="Close search"
				>
					<X size={15} />
				</button>
			</div>
			<div class="filter-section">
				<span class="filter-label">Type</span>
				<div class="filter-options">
					{#each NODE_TYPES as type}
						<button
							onclick={() => graphStore.setTypeFilter(activeTypeFilter === type ? null : type)}
							class="filter-chip"
							style="background: {activeTypeFilter === type
								? 'var(--color-primary)'
								: 'var(--color-bg)'}; color: {activeTypeFilter === type
								? 'white'
								: 'var(--color-text-muted)'}; border-color: {activeTypeFilter === type
								? 'var(--color-primary)'
								: 'var(--color-border)'};"
						>
							{type}
						</button>
					{/each}
				</div>
			</div>
			<div class="filter-section">
				<span class="filter-label">Status</span>
				<div class="filter-options">
					{#each STATUS_ITEMS as opt}
						<button
							onclick={() =>
								graphStore.setStatusFilter(activeStatusFilter === opt.value ? null : opt.value)}
							class="filter-chip status-chip"
							style="background: {activeStatusFilter === opt.value
								? opt.color + '33'
								: 'var(--color-bg)'}; color: {activeStatusFilter === opt.value
								? opt.color
								: 'var(--color-text-muted)'}; border-color: {activeStatusFilter === opt.value
								? opt.color
								: 'var(--color-border)'};"
						>
							<span class="status-dot" style="background: {opt.color};"></span>
							{opt.value}
						</button>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.search-anchor {
		position: relative;
	}

	.search-toggle {
		position: relative;
		display: flex;
		width: 32px;
		height: 32px;
		align-items: center;
		justify-content: center;
		border-radius: 8px;
		background: var(--color-surface-hover);
		color: var(--color-text-muted);
		transition: color 0.15s ease, background 0.15s ease;
	}

	.search-toggle:hover {
		color: var(--color-text);
	}

	.search-active {
		background: var(--color-primary);
		color: white;
	}

	.search-popover {
		position: absolute;
		bottom: calc(100% + 8px);
		left: 50%;
		width: 300px;
		transform: translateX(-50%);
		padding: 8px;
		border-radius: 8px;
		background: rgba(27, 26, 30, 0.96);
		border: 1px solid var(--color-border);
		box-shadow: 0 14px 34px rgba(0, 0, 0, 0.35);
		backdrop-filter: blur(14px);
		z-index: 60;
		animation: slideDown 0.15s ease-out;
	}

	.popover-heading {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 8px;
		padding: 0 2px;
		font-size: 11px;
		font-weight: 700;
		color: var(--color-text);
	}

	.clear-btn {
		font-size: 11px;
		font-weight: 600;
		color: var(--color-text-muted);
		transition: color 0.15s ease;
	}

	.clear-btn:hover {
		color: var(--color-text);
	}

	.search-row {
		display: flex;
		height: 32px;
		align-items: center;
		gap: 7px;
		padding: 0 7px;
		border-radius: 7px;
		background: var(--color-bg);
		border: 1px solid var(--color-primary);
		color: var(--color-text-muted);
	}

	.filter-section {
		margin-top: 8px;
	}

	.filter-label {
		display: block;
		margin: 0 0 5px 2px;
		font-size: 10px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}

	.filter-options {
		display: flex;
		flex-wrap: wrap;
		gap: 5px;
	}

	.filter-chip {
		height: 24px;
		padding: 0 8px;
		border-radius: 6px;
		border: 1px solid;
		font-size: 11px;
		font-weight: 700;
		transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
	}

	.status-chip {
		display: inline-flex;
		align-items: center;
		gap: 5px;
	}

	.status-dot {
		width: 7px;
		height: 7px;
		border-radius: 999px;
	}

	.filter-badge {
		position: absolute;
		top: 3px;
		right: 3px;
		min-width: 12px;
		height: 12px;
		padding: 0 3px;
		border-radius: 999px;
		background: #ededec;
		color: #1b1a1e;
		font-size: 8px;
		font-weight: 800;
		line-height: 12px;
	}

	.search-input {
		min-width: 0;
		flex: 1;
		background: transparent;
		color: var(--color-text);
		font-size: 12px;
		outline: none;
	}

	.search-count {
		flex-shrink: 0;
		font-size: 11px;
		font-weight: 700;
		color: var(--color-text-muted);
	}

	.close-btn {
		display: flex;
		width: 22px;
		height: 22px;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		border-radius: 5px;
		color: var(--color-text-muted);
		transition: background 0.15s ease, color 0.15s ease;
	}

	.close-btn:hover {
		background: var(--color-surface-hover);
		color: var(--color-text);
	}

	.toolbar-btn {
		position: relative;
	}

	.toolbar-btn::after {
		content: attr(data-tooltip);
		position: absolute;
		bottom: calc(100% + 6px);
		left: 50%;
		transform: translateX(-50%) translateY(-2px);
		padding: 4px 8px;
		border-radius: 6px;
		font-size: 11px;
		font-weight: 500;
		white-space: nowrap;
		background: rgba(27, 26, 30, 0.95);
		color: #ededec;
		border: 1px solid rgba(38, 37, 42, 0.6);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		pointer-events: none;
		opacity: 0;
		transition: opacity 0.15s ease, transform 0.15s ease;
		z-index: 70;
	}

	.toolbar-btn:hover::after {
		opacity: 1;
		transform: translateX(-50%) translateY(0px);
	}

	@keyframes slideDown {
		from {
			opacity: 0;
			transform: translateX(-50%) translateY(4px);
		}
		to {
			opacity: 1;
			transform: translateX(-50%) translateY(0);
		}
	}
</style>
