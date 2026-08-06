<script lang="ts">

	import { goto } from '$app/navigation';
	import { page } from '$app/state';

	import { authStore } from '$lib/stores/auth.svelte';
	import { teamsStore } from '$lib/stores/teams.svelte';
	import { themeStore } from '$lib/stores/theme.svelte';

	import {
		LayoutDashboard,
		Users,
		FolderOpen,
		LogOut,
		ChevronDown,
		ChevronRight,
		Settings,
		UserCog,
		Sun,
		Moon,
		PanelLeftClose,
		PanelLeftOpen
	} from 'lucide-svelte';


	import ProjectMenu from '$lib/components/ProjectMenu.svelte';



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





	function toggleTeam(teamId:string){

		const next = new Set(collapsedTeams);


		if(next.has(teamId)){

			next.delete(teamId);

		}else{

			next.add(teamId);

		}


		collapsedTeams = next;

	}





	function isActive(path:string){

		return (
			page.url.pathname === path ||
			(path !== '/dashboard' &&
			page.url.pathname.startsWith(path))
		);

	}





	const currentTeam = $derived(

		teamsStore.teams.find(
			(team)=>team.slug === page.params.teamSlug
		) ?? null

	);





	const currentProject = $derived(

		currentTeam?.projects?.find(
			(project)=>project.id === page.params.projectId
		) ?? null

	);





	const browserTitle = $derived.by(()=>{

		const path = page.url.pathname;


		if(path === '/dashboard/settings')
			return 'Settings — Thask';


		if(path.endsWith('/members')){

			return currentTeam
			? `${currentTeam.name} Members — Thask`
			: 'Team Members — Thask';

		}



		if(page.params.projectId){

			return currentProject
			? `${currentProject.name} — Thask`
			: 'Project — Thask';

		}



		if(page.params.teamSlug){

			return currentTeam
			? `${currentTeam.name} — Thask`
			: `${page.params.teamSlug} — Thask`;

		}



		return 'Dashboard — Thask';


	});


</script>
{#if authStore.loading}

	<div class="flex items-center justify-center min-h-screen">
		<p class="text-[var(--color-text-muted)]">
			Loading...
		</p>
	</div>


{:else if !authStore.isAuthenticated}


	<div class="flex items-center justify-center min-h-screen">
		<p class="text-[var(--color-text-muted)]">
			Redirecting...
		</p>
	</div>



{:else}


<div class="flex h-screen">


	<!-- Sidebar -->

	<aside
		class="transition-all duration-200 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col overflow-hidden"
		style="width: {sidebarCollapsed ? '48px' : '224px'};"
	>


		<!-- Header -->

		<div
			class="p-2 border-b border-[var(--color-border)] flex items-center overflow-hidden"
			class:justify-center={sidebarCollapsed}
			class:gap-3={!sidebarCollapsed}
			style="padding:{sidebarCollapsed ? '8px':'16px'};"
		>


			<a
				href="/"
				class="flex items-center min-w-0 rounded-lg"
				class:gap-3={!sidebarCollapsed}
			>


				<img 
					src="/icon.svg"
					alt=""
					class="w-8 h-8 rounded-lg shrink-0"
				/>


				{#if !sidebarCollapsed}

					<div class="min-w-0">

						<h1 class="text-sm font-bold">
							Thask
						</h1>

						<p class="text-xs text-[var(--color-text-muted)] truncate">
							{authStore.user?.displayName}
						</p>

					</div>

				{/if}


			</a>


		</div>




		<!-- Navigation -->

		<nav class="flex-1 overflow-y-auto p-2 space-y-0.5">


			<a
				href="/dashboard"
				class="flex items-center py-2 rounded-lg text-sm transition-colors"
				class:gap-2.5={!sidebarCollapsed}
				class:px-3={!sidebarCollapsed}
				class:justify-center={sidebarCollapsed}
				class:px-0={sidebarCollapsed}
				style="background:{isActive('/dashboard') ? 'var(--color-primary)' : 'transparent'}; color:{isActive('/dashboard') ? 'white':'var(--color-text)'};"
			>


				<LayoutDashboard 
					size={15}
					class="shrink-0"
				/>


				{#if !sidebarCollapsed}

					<span>
						Dashboard
					</span>

				{/if}


			</a>




			{#each teamsStore.teams as team}


				<div class="pt-2">


					{#if !sidebarCollapsed}


					<button
						onclick={() => toggleTeam(team.id)}
						class="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider hover:bg-[var(--color-surface-hover)]"
					>

						<Users size={12}/>

						<span class="flex-1 text-left">
							{team.name}
						</span>


					</button>


					{/if}



				</div>


			{/each}


		</nav>




		<!-- Bottom Actions -->

		<div class="p-2 border-t border-[var(--color-border)] space-y-1">


			<!-- Theme Toggle Added -->

			<button

				onclick={() => themeStore.toggle()}

				class="w-full flex items-center py-2 rounded-lg text-sm transition-colors text-[var(--color-text-muted)]"

				class:gap-2.5={!sidebarCollapsed}

				class:px-3={!sidebarCollapsed}

				class:justify-center={sidebarCollapsed}

				class:px-0={sidebarCollapsed}

				hover:bg-[var(--color-surface-hover)]

			>


				{#if themeStore.theme === 'dark'}


					<Sun size={15}/>


					{#if !sidebarCollapsed}
						<span>
							Light Mode
						</span>
					{/if}



				{:else}


					<Moon size={15}/>


					{#if !sidebarCollapsed}
						<span>
							Dark Mode
						</span>
					{/if}



				{/if}



			</button>
						<!-- Settings -->

			<a
				href="/dashboard/settings"
				class="w-full flex items-center py-2 rounded-lg text-sm transition-colors"
				class:gap-2.5={!sidebarCollapsed}
				class:px-3={!sidebarCollapsed}
				class:justify-center={sidebarCollapsed}
				class:px-0={sidebarCollapsed}
				style="color: var(--color-text-muted);"
			>

				<Settings 
					size={15}
					class="shrink-0"
				/>


				{#if !sidebarCollapsed}

					<span>
						Settings
					</span>

				{/if}


			</a>




			<!-- Logout -->

			<button

				onclick={handleLogout}

				class="w-full flex items-center py-2 rounded-lg text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-surface-hover)] transition-colors"

				class:gap-2.5={!sidebarCollapsed}

				class:px-3={!sidebarCollapsed}

				class:justify-center={sidebarCollapsed}

				class:px-0={sidebarCollapsed}

			>


				<LogOut
					size={15}
					class="shrink-0"
				/>


				{#if !sidebarCollapsed}

					<span>
						Sign out
					</span>

				{/if}


			</button>



		</div>



	</aside>





	<!-- Main Content -->


	<main class="flex-1 overflow-hidden">

		{@render children()}

	</main>



</div>


{/if}