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
		X,
		Zap,
		GitBranch,
		Download,
		FileJson,
		Upload,
		Share2,
	} from 'lucide-svelte';
	import type { GraphNode, NodeType, NodeStatus } from '$lib/types';
	import { NODE_TYPES, STATUS_COLORS } from '$lib/constants';
	import { graphStore } from '$lib/stores/graph.svelte';
	import SearchBar from '$lib/components/SearchBar.svelte';

	interface Props {
		onAddNode: () => void;
		onAddGroup: () => void;
		onZoomIn: () => void;
		onZoomOut: () => void;
		onFitView: () => void;
		onRunLayout: () => void;
		onToggleImpact: () => void;
		onToggleAnalysis: () => void;
		onExportPNG: () => void;
		onExportJSON: () => void;
		onImport: (mode: 'replace' | 'merge') => void;
		isImpactActive: boolean;
		isAnalysisActive: boolean;
		canImpact: boolean;
		nodes: GraphNode[];
		onFocusNode: (nodeId: string) => void;
		onUndo: () => void;
		onRedo: () => void;
		canUndo: boolean;
		canRedo: boolean;
		onShare?: () => void;
	}

	let {
		onAddNode,
		onAddGroup,
		onZoomIn,
		onZoomOut,
		onFitView,
		onRunLayout,
		onToggleImpact,
		onToggleAnalysis,
		onExportPNG,
		onExportJSON,
		onImport,
		isImpactActive,
		isAnalysisActive,
		canImpact,
		nodes,
		onFocusNode,
		onUndo,
		onRedo,
		canUndo,
		canRedo,
		onShare,
	}: Props = $props();

	const STATUS_ITEMS: { value: NodeStatus; color: string }[] = [
		{ value: 'PASS', color: STATUS_COLORS.PASS },
		{ value: 'FAIL', color: STATUS_COLORS.FAIL },
		{ value: 'IN_PROGRESS', color: STATUS_COLORS.IN_PROGRESS },
		{ value: 'BLOCKED', color: STATUS_COLORS.BLOCKED },
	];

	let showFilters = $state(false);
	let showImportMenu = $state(false);
	let activeTypeFilter = $derived(graphStore.typeFilter);
	let activeStatusFilter = $derived(graphStore.statusFilter);

	let searchBar = $state<ReturnType<typeof SearchBar> | undefined>(undefined);

	function handleGlobalKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
			e.preventDefault();
			searchBar?.open();
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
	style="background: rgba(27,26,30,0.9); backdrop-filter: blur(12px); border: 1px solid var(--color-border);"
>
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

		<SearchBar bind:this={searchBar} {nodes} {onFocusNode} />

		<button
			onclick={onToggleImpact}
			disabled={!canImpact && !isImpactActive}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors {isImpactActive ? 'impact-active' : 'btn-muted'}"
			style="opacity: {canImpact || isImpactActive ? '1' : '0.35'};"
			data-tooltip="Impact Mode (I)"
		>
			<Zap size={16} />
		</button>

		<button
			onclick={onToggleAnalysis}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors {isAnalysisActive ? 'analysis-active' : 'btn-muted'}"
			data-tooltip="Analysis Mode (A)"
		>
			<GitBranch size={16} />
		</button>

		<div class="w-px h-5 mx-1 flex-shrink-0" style="background: var(--color-border);"></div>

		<!-- Group 6: Export -->
		<button
			onclick={onExportPNG}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Export PNG"
		>
			<Download size={16} />
		</button>
		<button
			onclick={onExportJSON}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Export JSON"
		>
			<FileJson size={16} />
		</button>
		<div class="relative">
			<button
				onclick={() => (showImportMenu = !showImportMenu)}
				class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors"
				style="background: {showImportMenu ? 'var(--color-primary)' : 'var(--color-surface-hover)'}; color: {showImportMenu ? 'white' : 'var(--color-text-muted)'};"
				data-tooltip="Import JSON"
			>
				<Upload size={16} />
			</button>
			{#if showImportMenu}
				<div
					class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 py-1 rounded-lg shadow-xl min-w-[140px] filter-slide-in"
					style="background: rgba(27,26,30,0.95); border: 1px solid var(--color-border);"
				>
					<button
						onclick={() => { showImportMenu = false; onImport('replace'); }}
						class="w-full px-3 py-1.5 text-xs text-left transition-colors hover:bg-[var(--color-surface-hover)]"
						style="color: var(--color-text);"
					>
						Replace
					</button>
					<button
						onclick={() => { showImportMenu = false; onImport('merge'); }}
						class="w-full px-3 py-1.5 text-xs text-left transition-colors hover:bg-[var(--color-surface-hover)]"
						style="color: var(--color-text);"
					>
						Merge
					</button>
				</div>
			{/if}
		</div>

		{#if onShare}
			<!-- Divider -->
			<div class="w-px h-5 mx-0.5" style="background: var(--color-border);"></div>

			<button
				onclick={onShare}
				class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
				data-tooltip="Share"
			>
				<Share2 size={16} />
			</button>
		{/if}
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
		background: #c9a84c;
		color: #000;
		animation: pulse 2s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			box-shadow: 0 0 0 0 rgba(201, 168, 76, 0.4);
		}
		50% {
			box-shadow: 0 0 0 6px rgba(201, 168, 76, 0);
		}
	}

	.analysis-active {
		background: #e07a5f;
		color: var(--color-bg);
		animation: pulse-analysis 2s ease-in-out infinite;
	}

	@keyframes pulse-analysis {
		0%,
		100% {
			box-shadow: 0 0 0 0 rgba(224, 122, 95, 0.4);
		}
		50% {
			box-shadow: 0 0 0 6px rgba(224, 122, 95, 0);
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
		background: rgba(27, 26, 30, 0.95);
		color: #ededec;
		border: 1px solid rgba(38, 37, 42, 0.6);
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
