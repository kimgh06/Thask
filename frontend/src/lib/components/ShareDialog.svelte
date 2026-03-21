<script lang="ts">
	import { api } from '$lib/api';
	import type { SharingInfo, ProjectRole } from '$lib/types';
	import { X, Link, Lock, Globe, Copy, Check, Trash2 } from 'lucide-svelte';

	interface Props {
		projectId: string;
		projectName: string;
		onClose: () => void;
	}

	let { projectId, projectName, onClose }: Props = $props();

	let sharing = $state<SharingInfo | null>(null);
	let loading = $state(true);
	let inviteEmail = $state('');
	let inviteRole = $state<ProjectRole>('viewer');
	let inviteError = $state('');
	let copied = $state(false);

	$effect(() => {
		loadSharing();
	});

	async function loadSharing() {
		loading = true;
		const res = await api.get<SharingInfo>(`/api/projects/${projectId}/sharing`);
		if (res.data) sharing = res.data;
		loading = false;
	}

	async function updateLinkSharing(mode: 'off' | 'viewer' | 'editor') {
		await api.put(`/api/projects/${projectId}/sharing`, { linkSharing: mode });
		await loadSharing();
	}

	async function handleInvite(e: Event) {
		e.preventDefault();
		inviteError = '';
		const res = await api.post(`/api/projects/${projectId}/sharing/members`, {
			email: inviteEmail,
			role: inviteRole,
		});
		if (res.error) {
			inviteError = res.error;
			return;
		}
		inviteEmail = '';
		await loadSharing();
	}

	async function handleRoleChange(userId: string, role: ProjectRole) {
		await api.patch(`/api/projects/${projectId}/sharing/members/${userId}`, { role });
		await loadSharing();
	}

	async function handleRemoveMember(userId: string) {
		await api.delete(`/api/projects/${projectId}/sharing/members/${userId}`);
		await loadSharing();
	}

	async function copyLink() {
		if (sharing?.shareUrl) {
			const url = window.location.origin + sharing.shareUrl;
			await navigator.clipboard.writeText(url);
			copied = true;
			setTimeout(() => { copied = false; }, 2000);
		}
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center"
	style="background: rgba(0,0,0,0.5);"
	onclick={(e) => { if (e.target === e.currentTarget) onClose(); }}
>
	<div
		class="w-[480px] max-h-[85vh] overflow-y-auto rounded-2xl"
		style="background: var(--color-surface); border: 1px solid var(--color-border);"
	>
		<!-- Header -->
		<div class="flex items-center justify-between px-5 py-4" style="border-bottom: 1px solid var(--color-border);">
			<h2 class="text-base font-semibold" style="color: var(--color-text);">Share "{projectName}"</h2>
			<button onclick={onClose} class="p-1 rounded-lg hover:bg-[var(--color-surface-hover)]" style="color: var(--color-text-muted);">
				<X size={18} />
			</button>
		</div>

		{#if loading}
			<div class="px-5 py-8 text-center" style="color: var(--color-text-muted);">Loading...</div>
		{:else if sharing}
			<div class="px-5 py-4 space-y-5">
				<!-- Invite -->
				<form onsubmit={handleInvite} class="flex gap-2">
					<input
						bind:value={inviteEmail}
						type="email"
						placeholder="Enter email address..."
						required
						class="flex-1 px-3 py-2 rounded-lg text-sm"
						style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
					/>
					<select
						bind:value={inviteRole}
						class="px-2 py-2 rounded-lg text-sm"
						style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
					>
						<option value="viewer">Viewer</option>
						<option value="editor">Editor</option>
					</select>
					<button type="submit" class="px-3 py-2 rounded-lg text-sm font-medium" style="background: var(--color-primary); color: white;">
						Invite
					</button>
				</form>
				{#if inviteError}
					<p class="text-sm" style="color: var(--color-danger);">{inviteError}</p>
				{/if}

				<!-- Members -->
				{#if sharing.members.length > 0}
					<div>
						<h3 class="text-xs font-semibold uppercase tracking-wider mb-2" style="color: var(--color-text-muted);">People with access</h3>
						<div class="space-y-1">
							{#each sharing.members as member}
								<div class="flex items-center gap-3 px-2 py-2 rounded-lg">
									<div
										class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold shrink-0"
										style="background: var(--color-primary); color: white;"
									>
										{member.user.displayName.charAt(0).toUpperCase()}
									</div>
									<div class="flex-1 min-w-0">
										<p class="text-sm truncate" style="color: var(--color-text);">{member.user.displayName}</p>
										<p class="text-xs truncate" style="color: var(--color-text-muted);">{member.user.email}</p>
									</div>
									<select
										value={member.role}
										onchange={(e) => handleRoleChange(member.userId, (e.target as HTMLSelectElement).value as ProjectRole)}
										class="px-2 py-1 rounded text-xs"
										style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
									>
										<option value="viewer">Viewer</option>
										<option value="editor">Editor</option>
									</select>
									<button
										onclick={() => handleRemoveMember(member.userId)}
										class="p-1 rounded hover:bg-[var(--color-surface-hover)]"
										style="color: var(--color-danger);"
									>
										<Trash2 size={14} />
									</button>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Link Sharing -->
				<div>
					<h3 class="text-xs font-semibold uppercase tracking-wider mb-2" style="color: var(--color-text-muted);">General access</h3>
					<div class="p-3 rounded-xl" style="background: var(--color-bg); border: 1px solid var(--color-border);">
						<div class="flex items-center gap-3">
							<div
								class="w-8 h-8 rounded-full flex items-center justify-center shrink-0"
								style="background: {sharing.linkSharing === 'off' ? 'var(--color-border)' : 'var(--color-primary)'}; color: white;"
							>
								{#if sharing.linkSharing === 'off'}
									<Lock size={14} />
								{:else}
									<Globe size={14} />
								{/if}
							</div>
							<div class="flex-1">
								<select
									value={sharing.linkSharing}
									onchange={(e) => updateLinkSharing((e.target as HTMLSelectElement).value as 'off' | 'viewer' | 'editor')}
									class="px-2 py-1 rounded text-sm font-medium w-full"
									style="background: transparent; border: none; color: var(--color-text); cursor: pointer;"
								>
									<option value="off">Restricted</option>
									<option value="viewer">Anyone with the link (Viewer)</option>
									<option value="editor">Anyone with the link (Editor)</option>
								</select>
								<p class="text-xs mt-0.5" style="color: var(--color-text-muted);">
									{#if sharing.linkSharing === 'off'}
										Only people with access can open this project
									{:else if sharing.linkSharing === 'viewer'}
										Anyone on the internet with the link can view
									{:else}
										Anyone on the internet with the link can edit
									{/if}
								</p>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div class="flex items-center justify-between px-5 py-4" style="border-top: 1px solid var(--color-border);">
				{#if sharing.linkSharing !== 'off' && sharing.shareUrl}
					<button
						onclick={copyLink}
						class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors"
						style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
					>
						{#if copied}
							<Check size={14} style="color: var(--color-success);" />
							Copied
						{:else}
							<Link size={14} />
							Copy link
						{/if}
					</button>
				{:else}
					<div></div>
				{/if}
				<button
					onclick={onClose}
					class="px-4 py-2 rounded-lg text-sm font-medium"
					style="background: var(--color-primary); color: white;"
				>
					Done
				</button>
			</div>
		{/if}
	</div>
</div>
