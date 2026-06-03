<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		Plus,
		Group,
		ZoomIn,
		ZoomOut,
		Maximize,
		HandGrab,
		LayoutGrid,
		Undo2,
		Redo2,
		Zap,
		GitBranch,
		Download,
		FileJson,
		Upload,
		Share2,
		MoreHorizontal,
	} from 'lucide-svelte';
	import type { GraphNode } from '$lib/types';
	import SearchBar from '$lib/components/SearchBar.svelte';

	interface Props {
		onAddNode: () => void;
		onAddGroup: () => void;
		onZoomIn: () => void;
		onZoomOut: () => void;
		onFitView: () => void;
		onTogglePanMode?: () => void;
		onRunLayout: (algorithm?: string) => void;
		onToggleImpact: () => void;
		onToggleAnalysis: () => void;
		onExportPNG: () => void;
		onExportJSON: () => void;
		onImport?: (mode: 'replace' | 'merge') => void;
		isImpactActive: boolean;
		isAnalysisActive: boolean;
		isPanModeActive?: boolean;
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
		onTogglePanMode,
		onRunLayout,
		onToggleImpact,
		onToggleAnalysis,
		onExportPNG,
		onExportJSON,
		onImport,
		isImpactActive,
		isAnalysisActive,
		isPanModeActive = false,
		canImpact,
		nodes,
		onFocusNode,
		onUndo,
		onRedo,
		canUndo,
		canRedo,
		onShare,
	}: Props = $props();

	let showMoreMenu = $state(false);

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
			onclick={() => onTogglePanMode?.()}
			disabled={!onTogglePanMode}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors {isPanModeActive ? 'pan-active' : 'btn-muted'}"
			style="opacity: {onTogglePanMode ? '1' : '0.35'};"
			data-tooltip="Move Canvas (V, hold Space)"
			aria-pressed={isPanModeActive}
		>
			<HandGrab size={16} />
		</button>
		<button
			onclick={() => onRunLayout()}
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

		<!-- Group 4: Search, Impact -->
		<SearchBar
			bind:this={searchBar}
			{nodes}
			{onFocusNode}
			onOpen={() => { showMoreMenu = false; }}
		/>

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
			data-tooltip="Analysis Mode (⇧A)"
		>
			<GitBranch size={16} />
		</button>

		<div class="w-px h-5 mx-1 flex-shrink-0" style="background: var(--color-border);"></div>

		<!-- Group 5: More actions -->
		<div class="relative">
			<button
				onclick={() => { showMoreMenu = !showMoreMenu; searchBar?.close(); }}
				class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors"
				style="background: {showMoreMenu ? 'var(--color-primary)' : 'var(--color-surface-hover)'}; color: {showMoreMenu ? 'white' : 'var(--color-text-muted)'};"
				data-tooltip="More"
				aria-pressed={showMoreMenu}
			>
				<MoreHorizontal size={17} />
			</button>
			{#if showMoreMenu}
				<div class="toolbar-popover more-popover filter-slide-in">
					<button
						onclick={() => { showMoreMenu = false; onExportPNG(); }}
						class="menu-item"
					>
						<Download size={14} />
						<span>Export PNG</span>
					</button>
					<button
						onclick={() => { showMoreMenu = false; onExportJSON(); }}
						class="menu-item"
					>
						<FileJson size={14} />
						<span>Export JSON</span>
					</button>
					{#if onImport}
						<div class="menu-divider"></div>
						<div class="menu-label">Import JSON</div>
						<button
							onclick={() => { showMoreMenu = false; onImport?.('replace'); }}
							class="menu-item"
						>
							<Upload size={14} />
							<span>Replace graph</span>
						</button>
						<button
							onclick={() => { showMoreMenu = false; onImport?.('merge'); }}
							class="menu-item"
						>
							<Upload size={14} />
							<span>Merge into graph</span>
						</button>
					{/if}
					{#if onShare}
						<div class="menu-divider"></div>
						<button
							onclick={() => { showMoreMenu = false; onShare?.(); }}
							class="menu-item"
						>
							<Share2 size={14} />
							<span>Share</span>
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>
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

	.pan-active {
		background: #6aa6f8;
		color: #07111f;
	}

	.toolbar-popover {
		position: absolute;
		bottom: calc(100% + 8px);
		padding: 8px;
		border-radius: 8px;
		background: rgba(27, 26, 30, 0.96);
		border: 1px solid var(--color-border);
		box-shadow: 0 14px 34px rgba(0, 0, 0, 0.35);
		backdrop-filter: blur(14px);
		z-index: 60;
	}

	.more-popover {
		right: 0;
		min-width: 180px;
	}

	.menu-label {
		display: block;
		margin: 0 0 5px 2px;
		font-size: 10px;
		font-weight: 700;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}

	.menu-item {
		display: flex;
		width: 100%;
		height: 30px;
		align-items: center;
		gap: 8px;
		padding: 0 8px;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 600;
		color: var(--color-text);
		transition: background 0.15s ease, color 0.15s ease;
	}

	.menu-item:hover {
		background: var(--color-surface-hover);
	}

	.menu-divider {
		height: 1px;
		margin: 6px 0;
		background: var(--color-border);
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
