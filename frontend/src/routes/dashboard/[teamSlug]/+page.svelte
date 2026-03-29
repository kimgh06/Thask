<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api';
	import type { Project } from '$lib/types';
	import TemplateSelector from '$lib/components/TemplateSelector.svelte';

	let projects = $state<Project[]>([]);
	let newProjectName = $state('');
	let loadError = $state('');
	let showTemplates = $state(false);
	let newProjectId = $state('');

	const teamSlug = $derived(page.params.teamSlug);

	$effect(() => {
		loadProjects();
	});

	async function loadProjects() {
		loadError = '';
		const res = await api.get<Project[]>(`/api/teams/${teamSlug}/projects`);
		if (res.data) { projects = res.data; }
		else { loadError = 'Failed to load projects.'; }
	}

	async function createProject() {
		if (!newProjectName) return;
		const res = await api.post<Project>(`/api/teams/${teamSlug}/projects`, { name: newProjectName });
		newProjectName = '';
		if (res.data) {
			newProjectId = res.data.id;
			showTemplates = true;
			loadProjects();
		} else {
			loadProjects();
		}
	}
</script>

<div class="p-8">
	<h1 class="text-2xl font-bold mb-6">Team: {teamSlug}</h1>

	<div class="mb-6 flex gap-3">
		<input bind:value={newProjectName} placeholder="New project name" class="px-3 py-2 rounded-lg bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-text)]" />
		<button onclick={createProject} class="px-4 py-2 rounded-lg bg-[var(--color-primary)] text-white text-sm">Create Project</button>
	</div>

	{#if loadError}
		<p class="text-sm mb-4" style="color: var(--color-danger);">{loadError}</p>
	{/if}

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

{#if showTemplates}
	<TemplateSelector
		projectId={newProjectId}
		onApplied={() => { showTemplates = false; goto(`/dashboard/${teamSlug}/${newProjectId}`); }}
		onclose={() => { showTemplates = false; goto(`/dashboard/${teamSlug}/${newProjectId}`); }}
	/>
{/if}
