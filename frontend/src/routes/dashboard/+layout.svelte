<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { authStore } from '$lib/stores/auth.svelte';
	import { teamsStore } from '$lib/stores/teams.svelte';
	import { LayoutDashboard, Users, FolderOpen, LogOut, ChevronDown, ChevronRight, Settings, UserCog, Sun, Moon, PanelLeftClose, PanelLeftOpen } from 'lucide-svelte';
	import ProjectMenu from '$lib/components/ProjectMenu.svelte';
	import { themeStore } from '$lib/stores/theme.svelte';

	let { children } = $props();
	let collapsedTeams = $state<Set<string>>(new Set());
	let sidebarCollapsed = $state(false);

	$effect(() => {
		if (!authStore.loading && !authStore.isAuthenticated) {
			goto('/login');
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated) {
			teamsStore.load();
		}
	});

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

	const currentTeam = $derived(
		teamsStore.teams.find((team) => team.slug === page.params.teamSlug) ?? null,
	);
	const currentProject = $derived(
		currentTeam?.projects?.find((project) => project.id === page.params.projectId) ?? null,
	);
	const browserTitle = $derived.by(() => {
		const path = page.url.pathname;
		if (path === '/dashboard/settings') return 'Settings — Thask';
		if (path.endsWith('/members')) {
			return currentTeam ? `${currentTeam.name} Members — Thask` : 'Team Members — Thask';
		}
		if (page.params.projectId) {
			return currentProject ? `${currentProject.name} — Thask` : 'Project — Thask';
		}
		if (page.params.teamSlug) {
			return currentTeam ? `${currentTeam.name} — Thask` : `${page.params.teamSlug} — Thask`;
		}
		return 'Dashboard — Thask';
	});

</script>

<svelte:head>
	<title>{browserTitle}</title>
</svelte:head>

{#if authStore.loading}
	<div class="flex items-center justify-center min-h-screen">
		<p class="text-[var(--color-text-muted)]">Loading...</p>
	</div>
{:else if !authStore.isAuthenticated}
	<div class="flex items-center justify-center min-h-screen">
		<p class="text-[var(--color-text-muted)]">Redirecting...</p>
	</div>
{:else}
	<div class="flex h-screen">
		<!-- Sidebar -->
		<aside
			class="transition-all duration-200 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col overflow-hidden"
			style="width: {sidebarCollapsed ? '48px' : '224px'};"
		>
			<!-- Header -->
			<div class="p-2 border-b border-[var(--color-border)] flex items-center overflow-hidden"
				class:justify-center={sidebarCollapsed}
				class:gap-3={!sidebarCollapsed}
				style="padding: {sidebarCollapsed ? '8px' : '16px'};"
			>
				<a
					href="/"
					aria-label="Go to Thask home"
					class="flex items-center min-w-0 rounded-lg focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] focus:ring-offset-2 focus:ring-offset-[var(--color-surface)]"
					class:gap-3={!sidebarCollapsed}
				>
					<img src="/icon.svg" alt="" class="w-8 h-8 rounded-lg shrink-0" />
					{#if !sidebarCollapsed}
						<div class="min-w-0">
							<h1 class="text-sm font-bold leading-tight truncate">Thask</h1>
							<p class="text-xs text-[var(--color-text-muted)] truncate">{authStore.user?.displayName}</p>
						</div>
					{/if}
				</a>
			</div>

			<nav class="flex-1 overflow-y-auto p-2 space-y-0.5">
				<!-- Dashboard link -->
				<a
					href="/dashboard"
					class="flex items-center py-2 rounded-lg text-sm transition-colors"
					class:gap-2.5={!sidebarCollapsed}
					class:px-3={!sidebarCollapsed}
					class:justify-center={sidebarCollapsed}
					class:px-0={sidebarCollapsed}
					style="background: {isActive('/dashboard') ? 'var(--color-primary)' : 'transparent'}; color: {isActive('/dashboard') ? 'white' : 'var(--color-text)'};"
					onmouseenter={(e) => { if (!isActive('/dashboard')) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
					onmouseleave={(e) => { if (!isActive('/dashboard')) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
					title={sidebarCollapsed ? 'Dashboard' : ''}
				>
					<LayoutDashboard size={15} class="shrink-0" />
					{#if !sidebarCollapsed}
						<span>Dashboard</span>
					{/if}
				</a>

				<!-- Teams -->
				{#each teamsStore.teams as team}
					<div class="pt-2">
						{#if !sidebarCollapsed}
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
						{:else}
							<div
								class="flex items-center justify-center py-1.5 rounded-lg text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] transition-colors cursor-default"
								title={team.name}
							>
								<Users size={12} class="shrink-0" />
							</div>
						{/if}

						{#if !sidebarCollapsed && !collapsedTeams.has(team.id)}
							{@const membersHref = `/dashboard/${team.slug}/members`}
							<a
								href={membersHref}
								class="flex items-center gap-2.5 ml-1 px-3 py-1.5 rounded-lg text-sm transition-colors"
								style="background: {isActive(membersHref) ? 'var(--color-primary)' : 'transparent'}; color: {isActive(membersHref) ? 'white' : 'var(--color-text-muted)'};"
								onmouseenter={(e) => { if (!isActive(membersHref)) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
								onmouseleave={(e) => { if (!isActive(membersHref)) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
							>
								<UserCog size={14} class="shrink-0" />
								<span class="truncate">Members</span>
							</a>
						{/if}
						{#if !sidebarCollapsed && !collapsedTeams.has(team.id) && team.projects}
							{#each team.projects as project}
								{@const href = `/dashboard/${team.slug}/${project.id}`}
								<div
									class="flex items-center group ml-1 rounded-lg"
									style="background: {isActive(href) ? 'var(--color-primary)' : 'transparent'};"
									onmouseenter={(e) => { if (!isActive(href)) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
									onmouseleave={(e) => { if (!isActive(href)) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
								>
									<ProjectMenu projectId={project.id} projectName={project.name} projectHref={href} active={isActive(href)} onUpdated={() => teamsStore.load()}>
										<a
											{href}
											class="flex-1 flex items-center gap-2.5 px-3 py-1.5 text-sm transition-colors truncate"
											style="color: {isActive(href) ? 'white' : 'var(--color-text)'};"
										>
											<FolderOpen size={14} class="shrink-0" />
											<span class="truncate">{project.name}</span>
										</a>
									</ProjectMenu>
								</div>
							{/each}
						{/if}
					</div>
				{/each}
			</nav>

			<!-- Toggle button -->
			<div class="px-2 pb-1">
				<button
					onclick={() => sidebarCollapsed = !sidebarCollapsed}
					class="w-full flex items-center py-2 rounded-lg text-sm transition-colors"
					class:gap-2.5={!sidebarCollapsed}
					class:px-3={!sidebarCollapsed}
					class:justify-center={sidebarCollapsed}
					class:px-0={sidebarCollapsed}
					style="color: var(--color-text-muted);"
					onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
					onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
					title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
				>
					{#if sidebarCollapsed}
						<PanelLeftOpen size={15} class="shrink-0" />
					{:else}
						<PanelLeftClose size={15} class="shrink-0" />
						<span>Collapse</span>
					{/if}
				</button>
			</div>

			<div class="p-2 border-t border-[var(--color-border)] space-y-0.5">
				<a
					href="/dashboard/settings"
					class="w-full flex items-center py-2 rounded-lg text-sm transition-colors"
					class:gap-2.5={!sidebarCollapsed}
					class:px-3={!sidebarCollapsed}
					class:justify-center={sidebarCollapsed}
					class:px-0={sidebarCollapsed}
					style="background: {isActive('/dashboard/settings') ? 'var(--color-primary)' : 'transparent'}; color: {isActive('/dashboard/settings') ? 'white' : 'var(--color-text-muted)'};"
					onmouseenter={(e) => { if (!isActive('/dashboard/settings')) (e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)'; }}
					onmouseleave={(e) => { if (!isActive('/dashboard/settings')) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
					title={sidebarCollapsed ? 'Settings' : ''}
				>
					<Settings size={15} class="shrink-0" />
					{#if !sidebarCollapsed}
						<span>Settings</span>
					{/if}
				</a>
				<button
					onclick={handleLogout}
					class="w-full flex items-center py-2 rounded-lg text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] transition-colors"
					class:gap-2.5={!sidebarCollapsed}
					class:px-3={!sidebarCollapsed}
					class:justify-center={sidebarCollapsed}
					class:px-0={sidebarCollapsed}
					title={sidebarCollapsed ? 'Sign out' : ''}
				>
					<LogOut size={15} class="shrink-0" />
					{#if !sidebarCollapsed}
						<span>Sign out</span>
					{/if}
				</button>
			</div>
		</aside>

		<!-- Main -->
		<main class="flex-1 overflow-hidden">
			{@render children()}
		</main>
	</div>
{/if}
