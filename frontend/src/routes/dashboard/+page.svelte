<script lang="ts">
	import { api } from '$lib/api';
	import type { Team } from '$lib/types';
	import { authStore } from '$lib/stores/auth.svelte';

	let teams = $state<Team[]>([]);
	let newTeamName = $state('');
	let newTeamSlug = $state('');
	let showCreateTeam = $state(false);

	$effect(() => {
		if (authStore.isAuthenticated) loadTeams();
	});

	async function loadTeams() {
		const res = await api.get<Team[]>('/api/teams');
		if (res.data) teams = res.data;
	}

	async function createTeam() {
		if (!newTeamName || !newTeamSlug) return;
		await api.post('/api/teams', { name: newTeamName, slug: newTeamSlug });
		newTeamName = '';
		newTeamSlug = '';
		showCreateTeam = false;
		loadTeams();
	}
</script>

<div class="p-8">
	<div class="flex items-center justify-between mb-8">
		<h1 class="text-2xl font-bold">Dashboard</h1>
		<button
			onclick={() => showCreateTeam = !showCreateTeam}
			class="px-4 py-2 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors"
		>
			New Team
		</button>
	</div>

	{#if showCreateTeam}
		<div class="mb-6 p-4 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]">
			<div class="flex gap-3 items-end">
				<div class="flex-1">
					<label class="block text-sm mb-1 text-[var(--color-text-muted)]">Team Name</label>
					<input bind:value={newTeamName} class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)]" />
				</div>
				<div class="flex-1">
					<label class="block text-sm mb-1 text-[var(--color-text-muted)]">Slug</label>
					<input bind:value={newTeamSlug} placeholder="my-team" class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)]" />
				</div>
				<button onclick={createTeam} class="px-4 py-2 rounded-lg bg-[var(--color-primary)] text-white text-sm">Create</button>
			</div>
		</div>
	{/if}

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each teams as team}
			<div class="p-4 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]">
				<h3 class="font-semibold mb-2">{team.name}</h3>
				<p class="text-sm text-[var(--color-text-muted)] mb-3">{team.projects?.length ?? 0} projects</p>
				{#if team.projects}
					{#each team.projects as project}
						<a
							href="/dashboard/{team.slug}/{project.id}"
							class="block px-3 py-2 mb-1 rounded-lg hover:bg-[var(--color-surface-hover)] text-sm"
						>
							{project.name}
						</a>
					{/each}
				{/if}
			</div>
		{/each}
	</div>
</div>
