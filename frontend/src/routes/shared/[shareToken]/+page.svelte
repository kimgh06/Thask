<script lang="ts">
	import { page } from '$app/state';
	import type { GraphNode, GraphEdge, GraphData } from '$lib/types';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import { Eye } from 'lucide-svelte';

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let projectName = $state('');
	let loading = $state(true);
	let error = $state('');
	let linkSharing = $state('viewer');

	const shareToken = $derived(page.params.shareToken ?? '');
	const API_URL = '';

	$effect(() => {
		if (!shareToken) return;
		loading = true;
		error = '';

		// Fetch project info
		fetch(`${API_URL}/api/shared/${shareToken}`)
			.then((r) => {
				if (!r.ok) throw new Error('Not found');
				return r.json();
			})
			.then((res) => {
				if (res.data) {
					projectName = res.data.name;
					linkSharing = res.data.linkSharing;
				}
			})
			.catch(() => { /* project info is non-critical, graph fetch handles errors */ });

		// Fetch graph
		fetch(`${API_URL}/api/shared/${shareToken}/graph`)
			.then((r) => r.json())
			.then((res) => {
				if (res.error || !res.data) {
					error = res.error || 'Failed to load graph';
					loading = false;
					return;
				}
				nodes = res.data.nodes ?? [];
				edges = res.data.edges ?? [];
				loading = false;
			})
			.catch(() => {
				error = 'Failed to load graph';
				loading = false;
			});
	});
</script>

<div class="flex flex-col h-screen" style="background: var(--color-bg);">
	<!-- Header -->
	<header
		class="flex items-center justify-between px-4 py-3 shrink-0"
		style="background: var(--color-surface); border-bottom: 1px solid var(--color-border);"
	>
		<div class="flex items-center gap-3">
			<h1 class="text-base font-semibold" style="color: var(--color-text);">
				{projectName || 'Shared Project'}
			</h1>
			<span
				class="flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium"
				style="background: var(--color-bg); color: var(--color-text-muted); border: 1px solid var(--color-border);"
			>
				<Eye size={12} />
				{linkSharing === 'editor' ? 'Editable' : 'View only'}
			</span>
		</div>
		<a
			href="/login"
			class="px-3 py-1.5 rounded-lg text-sm font-medium"
			style="background: var(--color-primary); color: white;"
		>
			Sign in
		</a>
	</header>

	<!-- Canvas -->
	<div class="flex-1 overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center h-full">
				<p style="color: var(--color-text-muted);">Loading...</p>
			</div>
		{:else if error}
			<div class="flex items-center justify-center h-full">
				<div class="text-center">
					<p class="text-lg font-medium mb-2" style="color: var(--color-text);">Access denied</p>
					<p style="color: var(--color-text-muted);">{error}</p>
				</div>
			</div>
		{:else}
			<CytoscapeCanvas
				{nodes}
				{edges}
				projectId=""
			/>
		{/if}
	</div>
</div>
