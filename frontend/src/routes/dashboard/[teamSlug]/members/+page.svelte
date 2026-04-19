<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { membersStore } from '$lib/stores/members.svelte';
	import { teamsStore } from '$lib/stores/teams.svelte';
	import type { TeamRole } from '$lib/types';
	import { UserPlus, Crown, Shield, User, Eye, Trash2, ArrowLeft } from 'lucide-svelte';

	const teamSlug = $derived(page.params.teamSlug ?? '');

	let inviteEmail = $state('');
	let inviteRole = $state<TeamRole>('member');
	let inviteError = $state('');
	let showInvite = $state(false);
	let actionError = $state('');

	const myRole = $derived.by(() => {
		const me = membersStore.members.find((m) => m.user.id === authStore.user?.id);
		return me?.role ?? null;
	});

	const canManage = $derived(myRole === 'owner' || myRole === 'admin');
	const isOwner = $derived(myRole === 'owner');

	$effect(() => {
		if (teamSlug) {
			membersStore.load(teamSlug);
		}
	});

	async function handleInvite(e: Event) {
		e.preventDefault();
		inviteError = '';
		const err = await membersStore.invite(teamSlug, inviteEmail, inviteRole);
		if (err) {
			inviteError = err;
		} else {
			inviteEmail = '';
			inviteRole = 'member';
			showInvite = false;
		}
	}

	async function handleRoleChange(userId: string, newRole: TeamRole) {
		actionError = '';
		const err = await membersStore.updateRole(teamSlug, userId, newRole);
		if (err) actionError = err;
	}

	async function handleRemove(userId: string, displayName: string) {
		if (!confirm(`Remove ${displayName} from team?`)) return;
		actionError = '';
		const err = await membersStore.remove(teamSlug, userId);
		if (err) actionError = err;
	}

	async function handleLeave() {
		if (!confirm('Are you sure you want to leave this team?')) return;
		const err = await membersStore.leave(teamSlug);
		if (err) {
			actionError = err;
		} else {
			await teamsStore.load();
			goto('/dashboard');
		}
	}

	async function handleDeleteTeam() {
		const team = teamsStore.teams.find((t) => t.slug === teamSlug);
		const name = team?.name ?? teamSlug;
		if (!confirm(`Delete team "${name}"? This will permanently delete all projects and data. This cannot be undone.`)) return;
		actionError = '';
		const err = await membersStore.deleteTeam(teamSlug);
		if (err) {
			actionError = err;
		} else {
			await teamsStore.load();
			goto('/dashboard');
		}
	}

	async function handleTransfer(userId: string, displayName: string) {
		if (!confirm(`Transfer ownership to ${displayName}? You will become an admin.`)) return;
		actionError = '';
		const err = await membersStore.transfer(teamSlug, userId);
		if (err) actionError = err;
	}

	const ROLE_CONFIG: Record<TeamRole, { icon: typeof Crown; label: string; color: string }> = {
		owner: { icon: Crown, label: 'Owner', color: 'var(--color-accent)' },
		admin: { icon: Shield, label: 'Admin', color: '#6b8aed' },
		member: { icon: User, label: 'Member', color: 'var(--color-text-muted)' },
		viewer: { icon: Eye, label: 'Viewer', color: 'var(--color-text-muted)' },
	};

	function canChangeRole(targetRole: TeamRole): boolean {
		if (!canManage) return false;
		if (targetRole === 'owner') return false;
		if (myRole === 'admin' && targetRole === 'admin') return false;
		return true;
	}

	function canRemoveMember(targetRole: TeamRole): boolean {
		if (!canManage) return false;
		if (targetRole === 'owner') return false;
		if (myRole === 'admin' && targetRole === 'admin') return false;
		return true;
	}

	function availableRoles(targetRole: TeamRole): TeamRole[] {
		if (myRole === 'owner') return (['admin', 'member', 'viewer'] as TeamRole[]).filter((r) => r !== targetRole);
		if (myRole === 'admin') return (['member', 'viewer'] as TeamRole[]).filter((r) => r !== targetRole);
		return [];
	}
</script>

<div class="h-full overflow-y-auto" style="background: var(--color-bg);">
	<div class="max-w-3xl mx-auto px-6 py-8">
		<!-- Header -->
		<div class="flex items-center gap-3 mb-8">
			<a
				href="/dashboard/{teamSlug}"
				class="p-1.5 rounded-lg hover:bg-[var(--color-surface-hover)] transition-colors"
				style="color: var(--color-text-muted);"
			>
				<ArrowLeft size={18} />
			</a>
			<div class="flex-1">
				<h1 class="text-xl font-bold" style="color: var(--color-text);">Team Members</h1>
				<p class="text-sm" style="color: var(--color-text-muted);">{membersStore.members.length} members</p>
			</div>
			{#if canManage}
				<button
					onclick={() => { showInvite = !showInvite; }}
					class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors"
					style="background: var(--color-primary); color: white;"
				>
					<UserPlus size={15} />
					Invite
				</button>
			{/if}
		</div>

		{#if actionError}
			<div class="mb-4 px-4 py-2 rounded-lg text-sm" style="background: rgba(196,64,64,0.15); color: var(--color-danger);">
				{actionError}
			</div>
		{/if}

		<!-- Invite Form -->
		{#if showInvite}
			<form onsubmit={handleInvite} class="mb-6 p-4 rounded-xl" style="background: var(--color-surface); border: 1px solid var(--color-border);">
				<div class="flex gap-3">
					<input
						bind:value={inviteEmail}
						type="email"
						placeholder="Email address"
						required
						class="flex-1 px-3 py-2 rounded-lg text-sm"
						style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
					/>
					<select
						bind:value={inviteRole}
						class="px-3 py-2 rounded-lg text-sm"
						style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
					>
						{#if isOwner}<option value="admin">Admin</option>{/if}
						<option value="member">Member</option>
						<option value="viewer">Viewer</option>
					</select>
					<button
						type="submit"
						class="px-4 py-2 rounded-lg text-sm font-medium"
						style="background: var(--color-primary); color: white;"
					>
						Send
					</button>
				</div>
				{#if inviteError}
					<p class="mt-2 text-sm" style="color: var(--color-danger);">{inviteError}</p>
				{/if}
			</form>
		{/if}

		<!-- Member List -->
		<div class="rounded-xl overflow-hidden" style="background: var(--color-surface); border: 1px solid var(--color-border);">
			{#each membersStore.members as member, i}
				{@const config = ROLE_CONFIG[member.role]}
				{@const isMe = member.user.id === authStore.user?.id}
				<div
					class="flex items-center gap-3 px-4 py-3"
					style="{i > 0 ? `border-top: 1px solid var(--color-border);` : ''}"
				>
					<!-- Avatar -->
					<div
						class="w-9 h-9 rounded-full flex items-center justify-center text-sm font-semibold shrink-0"
						style="background: {config.color}; color: white; opacity: {member.role === 'viewer' ? 0.6 : 1};"
					>
						{member.user.displayName.charAt(0).toUpperCase()}
					</div>

					<!-- Info -->
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<span class="text-sm font-medium truncate" style="color: var(--color-text);">
								{member.user.displayName}
							</span>
							{#if isMe}
								<span class="text-[10px] px-1.5 py-0.5 rounded" style="background: var(--color-border); color: var(--color-text-muted);">you</span>
							{/if}
						</div>
						<p class="text-xs truncate" style="color: var(--color-text-muted);">{member.user.email}</p>
					</div>

					<!-- Role Badge -->
					<div class="flex items-center gap-1.5 px-2 py-1 rounded-md" style="background: var(--color-bg);">
						<svelte:component this={config.icon} size={12} style="color: {config.color};" />
						<span class="text-xs font-medium" style="color: {config.color};">{config.label}</span>
					</div>

					<!-- Actions -->
					{#if !isMe && canChangeRole(member.role)}
						<div class="flex items-center gap-1">
							{#each availableRoles(member.role) as role}
								<button
									onclick={() => handleRoleChange(member.userId, role)}
									class="px-2 py-1 rounded text-[11px] hover:opacity-80 transition-opacity"
									style="background: var(--color-bg); color: var(--color-text-muted); border: 1px solid var(--color-border);"
									title="Change to {role}"
								>
									{role}
								</button>
							{/each}
						</div>
					{/if}

					{#if !isMe && canRemoveMember(member.role)}
						<button
							onclick={() => handleRemove(member.userId, member.user.displayName)}
							class="p-1.5 rounded-lg hover:bg-[var(--color-surface-hover)] transition-colors"
							style="color: var(--color-danger);"
							title="Remove member"
						>
							<Trash2 size={14} />
						</button>
					{/if}

					{#if isOwner && !isMe}
						<button
							onclick={() => handleTransfer(member.userId, member.user.displayName)}
							class="p-1.5 rounded-lg hover:bg-[var(--color-surface-hover)] transition-colors"
							style="color: var(--color-accent);"
							title="Transfer ownership"
						>
							<Crown size={14} />
						</button>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Danger Zone -->
		<div class="mt-8 pt-6 flex items-center gap-3" style="border-top: 1px solid var(--color-border);">
			{#if !isOwner}
				<button
					onclick={handleLeave}
					class="px-4 py-2 rounded-lg text-sm font-medium transition-colors hover:opacity-90"
					style="background: rgba(196,64,64,0.15); color: var(--color-danger);"
				>
					Leave Team
				</button>
			{/if}
			{#if isOwner}
				<button
					onclick={handleDeleteTeam}
					class="px-4 py-2 rounded-lg text-sm font-medium transition-colors hover:opacity-90"
					style="background: rgba(196,64,64,0.15); color: var(--color-danger);"
				>
					Delete Team
				</button>
			{/if}
		</div>
	</div>
</div>
