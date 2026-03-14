<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Plus,
		Group,
		ZoomIn,
		ZoomOut,
		Maximize,
		LayoutGrid,
		Undo2,
		Redo2,
		Filter,
		Search,
		X,
		Zap,
		Trash2,
	} from 'lucide-svelte';
	import type { GraphNode, NodeType, NodeStatus } from '$lib/types';
	import { NODE_TYPES, STATUS_COLORS } from '$lib/constants';
	import { graphStore } from '$lib/stores/graph.svelte';

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
		selectedCount?: number;
		onBatchDelete?: () => void;
		onBatchStatus?: (status: NodeStatus) => void;
		onDeselectAll?: () => void;
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
		selectedCount = 0,
		onBatchDelete,
		onBatchStatus,
		onDeselectAll,
	}: Props = $props();

	const STATUS_ITEMS: { value: NodeStatus; color: string }[] = [
		{ value: 'PASS', color: STATUS_COLORS.PASS },
		{ value: 'FAIL', color: STATUS_COLORS.FAIL },
		{ value: 'IN_PROGRESS', color: STATUS_COLORS.IN_PROGRESS },
		{ value: 'BLOCKED', color: STATUS_COLORS.BLOCKED },
	];

	let showFilters = $state(false);
	let activeTypeFilter = $derived(graphStore.typeFilter);
	let activeStatusFilter = $derived(graphStore.statusFilter);

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
	});

	function cycleSearch() {
		if (searchResults.length === 0) return;
		onFocusNode(searchResults[searchIndex].id);
		searchIndex = (searchIndex + 1) % searchResults.length;
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
	class="flex flex-col gap-1 p-2 rounded-xl shadow-xl"
	style="background: rgba(30,41,59,0.85); backdrop-filter: blur(12px); border: 1px solid var(--color-border);"
>
	<!-- Batch context bar -->
	{#if selectedCount > 1}
		<div
			class="flex items-center gap-1.5 pb-1.5 mb-0.5 border-b filter-slide-in"
			style="border-color: var(--color-border);"
		>
			<span class="text-xs font-semibold px-1" style="color: var(--color-text);">
				{selectedCount} selected
			</span>
			<button
				onclick={onBatchDelete}
				class="flex items-center gap-1 px-2 h-7 rounded-md text-xs font-medium transition-colors"
				style="background: rgba(239,68,68,0.15); color: #f87171;"
				title="Delete selected"
			>
				<Trash2 size={14} />
				Delete
			</button>
			{#each STATUS_ITEMS as opt}
				<button
					onclick={() => onBatchStatus?.(opt.value)}
					class="w-3 h-3 rounded-full transition-transform hover:scale-125"
					style="background: {opt.color}; border: 2px solid rgba(255,255,255,0.2);"
					title="Set {opt.value}"
				></button>
			{/each}
			<button
				onclick={onDeselectAll}
				class="w-7 h-7 flex items-center justify-center rounded-md transition-colors btn-muted ml-auto"
				title="Deselect all"
			>
				<X size={14} />
			</button>
		</div>
	{/if}

	<!-- Main toolbar row -->
	<div class="flex items-center gap-0.5">

		<!-- Group 1: Add Node, Add Group -->
		<button
			onclick={onAddNode}
			class="toolbar-btn flex items-center gap-1.5 px-3 h-8 rounded-lg text-xs font-semibold transition-colors"
			style="background: var(--color-primary); color: white;"
			data-tooltip="Add Node (N)"
		>
			<Plus size={16} />
			Node
		</button>
		<button
			onclick={onAddGroup}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Add Group (G)"
		>
			<Group size={16} />
		</button>

		<div class="w-px h-5 mx-1 flex-shrink-0" style="background: var(--color-border);"></div>

		<!-- Group 2: Zoom In, Zoom Out, Fit View, Layout -->
		<button
			onclick={onZoomIn}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Zoom In (+)"
		>
			<ZoomIn size={16} />
		</button>
		<button
			onclick={onZoomOut}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Zoom Out (-)"
		>
			<ZoomOut size={16} />
		</button>
		<button
			onclick={onFitView}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Fit View (0)"
		>
			<Maximize size={16} />
		</button>
		<button
			onclick={onRunLayout}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Auto Layout (L)"
		>
			<LayoutGrid size={16} />
		</button>

		<div class="w-px h-5 mx-1 flex-shrink-0" style="background: var(--color-border);"></div>

		<!-- Group 3: Undo, Redo -->
		<button
			onclick={onUndo}
			disabled={!canUndo}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			style="opacity: {canUndo ? '1' : '0.35'};"
			data-tooltip="Undo (⌘Z)"
		>
			<Undo2 size={16} />
		</button>
		<button
			onclick={onRedo}
			disabled={!canRedo}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			style="opacity: {canRedo ? '1' : '0.35'};"
			data-tooltip="Redo (⌘⇧Z)"
		>
			<Redo2 size={16} />
		</button>

		<div class="w-px h-5 mx-1 flex-shrink-0" style="background: var(--color-border);"></div>

		<!-- Group 4: Filter, Search, Impact -->
		<button
			onclick={() => (showFilters = !showFilters)}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors relative"
			style="background: {showFilters ? 'var(--color-primary)' : 'var(--color-surface-hover)'}; color: {showFilters ? 'white' : 'var(--color-text-muted)'};"
			data-tooltip="Filter"
		>
			<Filter size={16} />
			{#if activeTypeFilter || activeStatusFilter}
				<span
					class="absolute top-1 right-1 w-1.5 h-1.5 rounded-full"
					style="background: var(--color-primary);"
				></span>
			{/if}
		</button>

		{#if showSearch}
			<div class="flex items-center gap-1 ml-0.5">
				<input
					bind:this={searchInput}
					bind:value={searchQuery}
					onkeydown={handleSearchKeydown}
					placeholder="Search nodes..."
					class="px-2 py-1 rounded-lg text-xs outline-none transition-all"
					style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-primary); width: 150px; height: 32px;"
				/>
				{#if searchResultsText}
					<span class="text-xs whitespace-nowrap" style="color: var(--color-text-muted);"
						>{searchResultsText}</span
					>
				{/if}
				<button
					onclick={closeSearch}
					class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
					data-tooltip="Close (Esc)"
				>
					<X size={16} />
				</button>
			</div>
		{:else}
			<button
				onclick={openSearch}
				class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
				data-tooltip="Search (⌘F)"
			>
				<Search size={16} />
			</button>
		{/if}

		<button
			onclick={onToggleImpact}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors {isImpactActive ? 'impact-active' : 'btn-muted'}"
			data-tooltip="Impact Mode (I)"
		>
			<Zap size={16} />
		</button>
	</div>

	<!-- Filter bar (slides in smoothly) -->
	{#if showFilters}
		<div
			class="flex flex-col gap-1 pt-1.5 mt-0.5 border-t filter-slide-in"
			style="border-color: var(--color-border);"
		>
			<!-- Node type filters -->
			<div class="flex items-center gap-1 flex-wrap">
				<span class="text-xs mr-1 flex-shrink-0" style="color: var(--color-text-muted);">Type:</span
				>
				{#each NODE_TYPES as type}
					<button
						onclick={() => graphStore.setTypeFilter(activeTypeFilter === type ? null : type)}
						class="px-2 py-0.5 rounded-md text-xs font-medium transition-colors"
						style="background: {activeTypeFilter === type
							? 'var(--color-primary)'
							: 'var(--color-bg)'}; color: {activeTypeFilter === type
							? 'white'
							: 'var(--color-text-muted)'}; border: 1px solid {activeTypeFilter === type
							? 'var(--color-primary)'
							: 'var(--color-border)'};"
					>
						{type}
					</button>
				{/each}
			</div>
			<!-- Status filters -->
			<div class="flex items-center gap-1 flex-wrap">
				<span class="text-xs mr-1 flex-shrink-0" style="color: var(--color-text-muted);">Status:</span
				>
				{#each STATUS_ITEMS as opt}
					<button
						onclick={() =>
							graphStore.setStatusFilter(activeStatusFilter === opt.value ? null : opt.value)}
						class="px-2 py-0.5 rounded-md text-xs font-medium transition-colors flex items-center gap-1"
						style="background: {activeStatusFilter === opt.value
							? opt.color + '33'
							: 'var(--color-bg)'}; color: {activeStatusFilter === opt.value
							? opt.color
							: 'var(--color-text-muted)'}; border: 1px solid {activeStatusFilter === opt.value
							? opt.color
							: 'var(--color-border)'};"
					>
						<span class="w-2 h-2 rounded-full inline-block" style="background: {opt.color};"></span>
						{opt.value}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>

<style>
	.btn-muted {
		background: var(--color-surface-hover);
		color: var(--color-text-muted);
	}

	.btn-muted:hover {
		color: var(--color-text);
	}

	.impact-active {
		background: #f59e0b;
		color: #000;
		animation: pulse 2s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.4);
		}
		50% {
			box-shadow: 0 0 0 6px rgba(245, 158, 11, 0);
		}
	}

	/* Tooltip */
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
		background: rgba(15, 23, 42, 0.95);
		color: #e2e8f0;
		border: 1px solid rgba(148, 163, 184, 0.15);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		pointer-events: none;
		opacity: 0;
		transition: opacity 0.15s ease, transform 0.15s ease;
		z-index: 50;
	}

	.toolbar-btn:hover::after {
		opacity: 1;
		transform: translateX(-50%) translateY(0px);
	}

	.filter-slide-in {
		animation: slideDown 0.15s ease-out;
	}

	@keyframes slideDown {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>
