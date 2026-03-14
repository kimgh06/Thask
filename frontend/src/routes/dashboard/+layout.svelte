<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth.svelte';
	import { api } from '$lib/api';
	import type { Team } from '$lib/types';
	import { LayoutDashboard, Users, FolderOpen, LogOut, ChevronDown, ChevronRight } from 'lucide-svelte';

	let { children } = $props();
	let teams = $state<Team[]>([]);
	let collapsedTeams = $state<Set<string>>(new Set());

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

	function toggleTeam(teamId: string) {
		const next = new Set(collapsedTeams);
		if (next.has(teamId)) {
			next.delete(teamId);
		} else {
			next.add(teamId);
		}
		collapsedTeams = next;
	}

	function isActive(path: string) {
		return page.url.pathname === path || (path !== '/dashboard' && page.url.pathname.startsWith(path));
	}

	const avatarLetter = $derived(
		authStore.user?.displayName?.charAt(0)?.toUpperCase() ?? '?'
	);
</script>

{#if authStore.loading}
	<div class="flex items-center justify-center min-h-screen">
		<p class="text-[var(--color-text-muted)]">Loading...</p>
	</div>
{:else if authStore.isAuthenticated}
	<div class="flex h-screen">
		<!-- Sidebar -->
		<aside class="w-56 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col">
			<!-- Header -->
			<div class="p-4 border-b border-[var(--color-border)] flex items-center gap-3">
				<div class="w-8 h-8 rounded-full bg-[var(--color-primary)] flex items-center justify-center text-white text-sm font-semibold shrink-0">
					{avatarLetter}
				</div>
				<div class="min-w-0">
					<h1 class="text-sm font-bold leading-tight truncate">Thask</h1>
					<p class="text-xs text-[var(--color-text-muted)] truncate">{authStore.user?.displayName}</p>
				</div>
			</div>

			<nav class="flex-1 overflow-y-auto p-2 space-y-0.5">
				<!-- Dashboard link -->
				<a
					href="/dashboard"
					class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm transition-colors"
					style="background: {isActive('/dashboard') ? 'var(--color-primary)' : 'transparent'}; color: {isActive('/dashboard') ? 'white' : 'var(--color-text)'};"
					onmouseenter={(e) => { if (!isActive('/dashboard')) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
					onmouseleave={(e) => { if (!isActive('/dashboard')) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
				>
					<LayoutDashboard size={15} class="shrink-0" />
					<span>Dashboard</span>
				</a>

				<!-- Teams -->
				{#each teams as team}
					<div class="pt-2">
						<button
							onclick={() => toggleTeam(team.id)}
							class="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider hover:bg-[var(--color-surface-hover)] transition-colors"
						>
							<Users size={12} class="shrink-0" />
							<span class="flex-1 text-left truncate">{team.name}</span>
							{#if (team.projects?.length ?? 0) > 0}
								<span class="bg-[var(--color-border)] text-[var(--color-text-muted)] rounded-full px-1.5 py-0.5 text-[10px] font-medium leading-none">
									{team.projects?.length}
								</span>
							{/if}
							{#if collapsedTeams.has(team.id)}
								<ChevronRight size={12} class="shrink-0" />
							{:else}
								<ChevronDown size={12} class="shrink-0" />
							{/if}
						</button>

						{#if !collapsedTeams.has(team.id) && team.projects}
							{#each team.projects as project}
								{@const href = `/dashboard/${team.slug}/${project.id}`}
								<a
									{href}
									class="flex items-center gap-2.5 px-3 py-1.5 rounded-lg text-sm transition-colors truncate ml-1"
									style="background: {isActive(href) ? 'var(--color-primary)' : 'transparent'}; color: {isActive(href) ? 'white' : 'var(--color-text)'};"
									onmouseenter={(e) => { if (!isActive(href)) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
									onmouseleave={(e) => { if (!isActive(href)) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
								>
									<FolderOpen size={14} class="shrink-0" />
									<span class="truncate">{project.name}</span>
								</a>
							{/each}
						{/if}
					</div>
				{/each}
			</nav>

			<div class="p-2 border-t border-[var(--color-border)]">
				<button
					onclick={handleLogout}
					class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] transition-colors"
				>
					<LogOut size={15} class="shrink-0" />
					<span>Sign out</span>
				</button>
			</div>
		</aside>

		<!-- Main -->
		<main class="flex-1 overflow-hidden">
			{@render children()}
		</main>
	</div>
{/if}
