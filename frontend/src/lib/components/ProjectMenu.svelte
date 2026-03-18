<script lang="ts">
	import type { Snippet } from 'svelte';
	import { api } from '$lib/api';

	interface Props {
		projectId: string;
		projectName: string;
		projectHref: string;
		active?: boolean;
		onUpdated: () => void;
		children: Snippet;
	}

	let { projectId, projectName, projectHref, active = false, onUpdated, children }: Props = $props();

	let showMenu = $state(false);
	let editing = $state(false);
	let editName = $state('');

	function toggleMenu(e: MouseEvent) {
		e.stopPropagation();
		e.preventDefault();
		showMenu = !showMenu;
	}

	function startRename() {
		editName = projectName;
		editing = true;
		showMenu = false;
	}

	async function submitRename(e: Event) {
		e.preventDefault();
		if (!editName) return;
		await api.patch(`/api/projects/${projectId}`, { name: editName });
		editing = false;
		editName = '';
		onUpdated();
	}

	function cancelRename() {
		editing = false;
		editName = '';
	}

	async function copyLink() {
		const url = `${window.location.origin}${projectHref}`;
		await navigator.clipboard.writeText(url);
		showMenu = false;
	}

	async function deleteProject() {
		await api.delete(`/api/projects/${projectId}`);
		showMenu = false;
		onUpdated();
	}

	function closeMenu() {
		showMenu = false;
	}
</script>

<svelte:window onclick={closeMenu} />

{#if editing}
	<form onsubmit={submitRename} class="flex-1 flex gap-1.5 px-3 py-1.5">
		<input
			bind:value={editName}
			onkeydown={(e) => { if (e.key === 'Escape') cancelRename(); }}
			class="flex-1 px-2 py-1 rounded-md bg-[var(--color-bg)] border border-[var(--color-primary)] text-[var(--color-text)] text-sm outline-none"
			autofocus
		/>
		<button type="submit" class="px-2 py-1 rounded-md bg-[var(--color-primary)] text-white text-xs shrink-0 cursor-pointer">OK</button>
		<button type="button" onclick={cancelRename} class="px-2 py-1 rounded-md bg-[var(--color-surface-hover)] text-[var(--color-text-muted)] text-xs shrink-0 cursor-pointer">Cancel</button>
	</form>
{:else}
	{@render children()}
	<div class="relative">
		<button
			onclick={toggleMenu}
			class="w-7 h-7 flex items-center justify-center rounded-md opacity-0 group-hover:opacity-100 transition-all text-sm shrink-0 cursor-pointer"
			style="color: {active ? 'rgba(255,255,255,0.7)' : 'var(--color-text-muted)'};"
		>⋯</button>
		{#if showMenu}
			<div class="absolute right-0 top-full z-50 mt-1 py-1 min-w-[130px] rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] shadow-lg">
				<button
					onclick={startRename}
					class="w-full text-left px-3 py-1.5 text-sm hover:bg-[var(--color-surface-hover)]"
				>Rename</button>
				<button
					onclick={copyLink}
					class="w-full text-left px-3 py-1.5 text-sm hover:bg-[var(--color-surface-hover)]"
				>Copy link</button>
				<button
					onclick={deleteProject}
					class="w-full text-left px-3 py-1.5 text-sm hover:bg-[var(--color-surface-hover)]" style="color: var(--color-danger);"
				>Delete</button>
			</div>
		{/if}
	</div>
{/if}
