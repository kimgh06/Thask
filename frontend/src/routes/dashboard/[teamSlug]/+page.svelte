<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import type { Project } from '$lib/types';

	let projects = $state<Project[]>([]);
	let newProjectName = $state('');

	const teamSlug = $derived(page.params.teamSlug);

	$effect(() => {
		loadProjects();
	});

	async function loadProjects() {
		const res = await api.get<Project[]>(`/api/teams/${teamSlug}/projects`);
		if (res.data) projects = res.data;
	}

	async function createProject() {
		if (!newProjectName) return;
		await api.post(`/api/teams/${teamSlug}/projects`, { name: newProjectName });
		newProjectName = '';
		loadProjects();
	}
</script>

<div class="p-8">
	<h1 class="text-2xl font-bold mb-6">Team: {teamSlug}</h1>

	<div class="mb-6 flex gap-3">
		<input bind:value={newProjectName} placeholder="New project name" class="px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-text)]" />
		<button onclick={createProject} class="px-4 py-2 rounded-lg bg-[var(--color-primary)] text-white text-sm">Create Project</button>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
		{#each projects as project}
			<a
				href="/dashboard/{teamSlug}/{project.id}"
				class="p-4 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)] hover:border-[var(--color-primary)] transition-colors"
			>
				<h3 class="font-semibold">{project.name}</h3>
				<p class="text-sm text-[var(--color-text-muted)] mt-1">{project.description || 'No description'}</p>
			</a>
		{/each}
	</div>
</div>
