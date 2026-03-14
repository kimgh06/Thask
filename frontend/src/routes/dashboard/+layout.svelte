<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { api } from '$lib/api';
	import type { Team } from '$lib/types';

	let { children } = $props();
	let teams = $state<Team[]>([]);

	$effect(() => {
		if (!authStore.loading && !authStore.isAuthenticated) {
			goto('/login');
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated) {
			loadTeams();
		}
	});

	async function loadTeams() {
		const res = await api.get<Team[]>('/api/teams');
		if (res.data) teams = res.data;
	}

	async function handleLogout() {
		await authStore.logout();
		goto('/login');
	}
</script>

{#if authStore.loading}
	<div class="flex items-center justify-center min-h-screen">
		<p class="text-[var(--color-text-muted)]">Loading...</p>
	</div>
{:else if authStore.isAuthenticated}
	<div class="flex h-screen">
		<!-- Sidebar -->
		<aside class="w-64 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col">
			<div class="p-4 border-b border-[var(--color-border)]">
				<h1 class="text-lg font-bold">Thask</h1>
				<p class="text-xs text-[var(--color-text-muted)]">{authStore.user?.displayName}</p>
			</div>

			<nav class="flex-1 overflow-y-auto p-3 space-y-1">
				<a href="/dashboard" class="block px-3 py-2 rounded-lg hover:bg-[var(--color-surface-hover)] text-sm">
					Dashboard
				</a>

				{#each teams as team}
					<div class="mt-3">
						<p class="px-3 py-1 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">{team.name}</p>
						{#if team.projects}
							{#each team.projects as project}
								<a
									href="/dashboard/{team.slug}/{project.id}"
									class="block px-3 py-1.5 rounded-lg hover:bg-[var(--color-surface-hover)] text-sm truncate"
								>
									{project.name}
								</a>
							{/each}
						{/if}
					</div>
				{/each}
			</nav>

			<div class="p-3 border-t border-[var(--color-border)]">
				<button
					onclick={handleLogout}
					class="w-full px-3 py-2 rounded-lg text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] text-left"
				>
					Sign out
				</button>
			</div>
		</aside>

		<!-- Main -->
		<main class="flex-1 overflow-hidden">
			{@render children()}
		</main>
	</div>
{/if}
