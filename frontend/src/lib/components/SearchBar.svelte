<script lang="ts">
	import { Search, X } from 'lucide-svelte';
	import type { GraphNode } from '$lib/types';

	interface Props {
		nodes: GraphNode[];
		onFocusNode: (nodeId: string) => void;
	}

	let { nodes, onFocusNode }: Props = $props();

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

	export function open() {
		showSearch = true;
		setTimeout(() => searchInput?.focus(), 0);
	}

	function close() {
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

{#if showSearch}
	<div class="flex items-center gap-1 ml-0.5">
		<input
			bind:this={searchInput}
			bind:value={searchQuery}
			onkeydown={handleKeydown}
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
			onclick={close}
			class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
			data-tooltip="Close (Esc)"
		>
			<X size={16} />
		</button>
	</div>
{:else}
	<button
		onclick={open}
		class="toolbar-btn w-8 h-8 flex items-center justify-center rounded-lg transition-colors btn-muted"
		data-tooltip="Search (⌘F)"
	>
		<Search size={16} />
	</button>
{/if}
