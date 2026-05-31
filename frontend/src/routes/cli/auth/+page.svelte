<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { authStore } from '$lib/stores/auth.svelte';

	let port = $derived($page.url.searchParams.get('callback_port'));
	let csrfState = $derived($page.url.searchParams.get('state'));

	let status = $state<'idle' | 'approving' | 'done' | 'error'>('idle');
	let error = $state('');

	onMount(async () => {
		if (!port || !csrfState) {
			status = 'error';
			error = 'Missing callback_port or state — open this page via `thask login`.';
			return;
		}
		// authStore is seeded by +layout.server.ts; the `seeded` flag means
		// authStore.user reflects current session state synchronously here.
		if (!authStore.user) {
			const here = $page.url.pathname + $page.url.search;
			await goto(`/login?next=${encodeURIComponent(here)}`);
		}
	});

	async function approve() {
		status = 'approving';
		error = '';
		const hostname = guessHostname();
		const today = new Date().toISOString().slice(0, 10);
		const res = await api.post<{ key: string; keyPrefix: string }>('/api/auth/api-keys', {
			name: `CLI (${hostname}) ${today}`,
			kind: 'user_interactive',
		});
		if (res.error || !res.data?.key) {
			status = 'error';
			error = res.error ?? 'Failed to create API key';
			return;
		}
		status = 'done';
		// Hand the token to the CLI's loopback server. The browser fetches the
		// URL — the CLI captures `?token` and the user can close this tab.
		const target = `http://localhost:${port}/?token=${encodeURIComponent(res.data.key)}&state=${encodeURIComponent(csrfState!)}`;
		window.location.href = target;
	}

	function cancel() {
		const target = `http://localhost:${port}/?error=denied&state=${encodeURIComponent(csrfState!)}`;
		window.location.href = target;
	}

	function guessHostname(): string {
		const ua = navigator.userAgent;
		if (ua.includes('Mac')) return 'Mac';
		if (ua.includes('Windows')) return 'Windows';
		if (ua.includes('Linux')) return 'Linux';
		return 'Unknown';
	}
</script>

<svelte:head>
	<title>Authorize Thask CLI</title>
</svelte:head>

<div class="flex items-center justify-center min-h-screen">
	<div
		class="w-full max-w-sm p-8 rounded-xl"
		style="background: var(--color-surface); border: 1px solid var(--color-border);"
	>
		<h1 class="text-2xl font-bold mb-2 text-center" style="color: var(--color-text);">Authorize CLI?</h1>
		<p class="text-sm text-center mb-6" style="color: var(--color-text-muted);">
			This creates a new API key with your full account access and sends it to the
			<code>thask</code> command listening on your machine.
		</p>

		{#if status === 'error'}
			<div class="mb-4 p-3 rounded-lg text-sm" style="background: rgba(239,68,68,0.1); color: #f87171;">
				{error}
			</div>
		{/if}

		{#if status === 'idle'}
			<div class="space-y-3">
				<button
					onclick={approve}
					class="w-full py-2 rounded-lg font-medium transition-colors"
					style="background: var(--color-primary); color: white;"
				>
					Approve
				</button>
				<button
					onclick={cancel}
					class="w-full py-2 rounded-lg font-medium transition-colors"
					style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
				>
					Cancel
				</button>
			</div>
			<p class="mt-4 text-xs text-center" style="color: var(--color-text-muted);">
				The key will appear in your settings — you can revoke it any time.
			</p>
		{:else if status === 'approving'}
			<p class="text-sm text-center" style="color: var(--color-text-muted);">Issuing key…</p>
		{:else if status === 'done'}
			<p class="text-sm text-center" style="color: var(--color-text-muted);">
				✓ Approved. Return to your terminal — this tab can be closed.
			</p>
		{/if}
	</div>
</div>
