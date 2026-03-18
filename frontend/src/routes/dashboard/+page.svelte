<script lang="ts">
	import { api } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';
	import { teamsStore } from '$lib/stores/teams.svelte';
	import ProjectMenu from '$lib/components/ProjectMenu.svelte';

	let newTeamName = $state('');
	let newTeamSlug = $state('');
	let showCreateTeam = $state(false);
	let addingProjectTeamId = $state<string | null>(null);
	let newProjectName = $state('');

	$effect(() => {
		if (authStore.isAuthenticated) teamsStore.load();
	});

	async function createTeam() {
		if (!newTeamName || !newTeamSlug) return;
		await api.post('/api/teams', { name: newTeamName, slug: newTeamSlug });
		newTeamName = '';
		newTeamSlug = '';
		showCreateTeam = false;
		teamsStore.load();
	}

	async function createProject(teamSlug: string) {
		if (!newProjectName) return;
		await api.post(`/api/teams/${teamSlug}/projects`, { name: newProjectName });
		newProjectName = '';
		addingProjectTeamId = null;
		teamsStore.load();
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

	{#if teamsStore.error}
		<p class="text-sm mb-4" style="color: var(--color-danger);">{teamsStore.error}</p>
	{/if}

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
		{#each teamsStore.teams as team}
			<div class="p-4 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]">
				<div class="flex items-center justify-between mb-2">
					<h3 class="font-semibold">{team.name}</h3>
					<button
						onclick={() => {
							if (addingProjectTeamId === team.id) {
								addingProjectTeamId = null;
								newProjectName = '';
							} else {
								addingProjectTeamId = team.id;
								newProjectName = '';
							}
						}}
						class="w-6 h-6 flex items-center justify-center rounded-md text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text)] transition-colors text-lg leading-none"
						title="Add project"
					>+</button>
				</div>
				{#if addingProjectTeamId === team.id}
					<form
						onsubmit={(e) => { e.preventDefault(); createProject(team.slug); }}
						class="flex gap-2 mb-3"
					>
						<input
							bind:value={newProjectName}
							placeholder="Project name"
							class="flex-1 px-3 py-1.5 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] text-sm"
						/>
						<button type="submit" class="px-3 py-1.5 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm transition-colors">Add</button>
					</form>
				{/if}
				<p class="text-sm text-[var(--color-text-muted)] mb-3">{team.projects?.length ?? 0} projects</p>
				{#if team.projects}
					{#each team.projects as project}
						{@const href = `/dashboard/${team.slug}/${project.id}`}
						<div class="flex items-center group mb-1 rounded-lg hover:bg-[var(--color-surface-hover)]">
							<ProjectMenu projectId={project.id} projectName={project.name} projectHref={href} onUpdated={() => teamsStore.load()}>
								<a {href} class="flex-1 block px-3 py-2 text-sm">{project.name}</a>
							</ProjectMenu>
						</div>
					{/each}
				{/if}
			</div>
		{/each}
	</div>
</div>
