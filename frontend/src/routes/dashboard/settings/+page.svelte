<script lang="ts">
	import { api } from '$lib/api';
	import type { APIKey, APIKeyCreated } from '$lib/types';
	import { Key, Trash2, Copy, Check, ArrowLeft } from 'lucide-svelte';

	let keys = $state<APIKey[]>([]);
	let loading = $state(true);
	let keyName = $state('');
	let expiresIn = $state<string>('');
	let createError = $state('');
	let createdKey = $state<string | null>(null);
	let copied = $state(false);

	$effect(() => {
		loadKeys();
	});

	async function loadKeys() {
		loading = true;
		const res = await api.get<APIKey[]>('/api/auth/api-keys');
		if (res.data) keys = res.data;
		loading = false;
	}

	async function handleCreate(e: Event) {
		e.preventDefault();
		createError = '';
		createdKey = null;

		const body: { name: string; expiresIn?: number } = { name: keyName };
		if (expiresIn) {
			const parsed = parseInt(expiresIn, 10);
			if (!isNaN(parsed)) body.expiresIn = parsed;
		}

		const res = await api.post<APIKeyCreated>('/api/auth/api-keys', body);
		if (res.error) {
			createError = res.error;
			return;
		}
		if (res.data) {
			createdKey = res.data.key;
			keyName = '';
			expiresIn = '';
			await loadKeys();
		}
	}

	async function handleDelete(keyId: string) {
		if (!confirm('Revoke this API key? This cannot be undone.')) return;
		await api.delete(`/api/auth/api-keys/${keyId}`);
		await loadKeys();
	}

	async function copyKey() {
		if (createdKey) {
			await navigator.clipboard.writeText(createdKey);
			copied = true;
			setTimeout(() => { copied = false; }, 2000);
		}
	}

	function formatDate(dateStr: string | null): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleDateString('ko-KR', {
			year: 'numeric', month: 'short', day: 'numeric',
		});
	}
</script>

<div class="h-full overflow-y-auto" style="background: var(--color-bg);">
	<div class="max-w-3xl mx-auto px-6 py-8">
		<!-- Header -->
		<div class="flex items-center gap-3 mb-8">
			<a
				href="/dashboard"
				class="p-1.5 rounded-lg hover:bg-[var(--color-surface-hover)] transition-colors"
				style="color: var(--color-text-muted);"
			>
				<ArrowLeft size={18} />
			</a>
			<div>
				<h1 class="text-xl font-bold" style="color: var(--color-text);">Settings</h1>
				<p class="text-sm" style="color: var(--color-text-muted);">Manage your API keys for CLI and agent access</p>
			</div>
		</div>

		<!-- Created Key Display -->
		{#if createdKey}
			<div class="mb-6 p-4 rounded-xl" style="background: rgba(94,168,122,0.1); border: 1px solid var(--color-success);">
				<p class="text-sm font-medium mb-2" style="color: var(--color-success);">API key created. Copy it now — it won't be shown again.</p>
				<div class="flex items-center gap-2">
					<code class="flex-1 px-3 py-2 rounded-lg text-xs font-mono break-all" style="background: var(--color-bg); color: var(--color-text);">
						{createdKey}
					</code>
					<button
						onclick={copyKey}
						class="p-2 rounded-lg shrink-0 transition-colors"
						style="background: var(--color-surface); color: {copied ? 'var(--color-success)' : 'var(--color-text-muted)'};"
					>
						{#if copied}
							<Check size={16} />
						{:else}
							<Copy size={16} />
						{/if}
					</button>
				</div>
			</div>
		{/if}

		<!-- Create Key Form -->
		<form onsubmit={handleCreate} class="mb-6 p-4 rounded-xl" style="background: var(--color-surface); border: 1px solid var(--color-border);">
			<h2 class="text-sm font-semibold mb-3" style="color: var(--color-text);">Create API Key</h2>
			<div class="flex gap-3">
				<input
					bind:value={keyName}
					type="text"
					placeholder="Key name (e.g. Claude Agent)"
					required
					class="flex-1 px-3 py-2 rounded-lg text-sm"
					style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
				/>
				<select
					bind:value={expiresIn}
					class="px-3 py-2 rounded-lg text-sm"
					style="background: var(--color-bg); border: 1px solid var(--color-border); color: var(--color-text);"
				>
					<option value="">No expiry</option>
					<option value="30">30 days</option>
					<option value="90">90 days</option>
					<option value="180">180 days</option>
					<option value="365">365 days</option>
				</select>
				<button
					type="submit"
					class="px-4 py-2 rounded-lg text-sm font-medium"
					style="background: var(--color-primary); color: white;"
				>
					Create
				</button>
			</div>
			{#if createError}
				<p class="mt-2 text-sm" style="color: var(--color-danger);">{createError}</p>
			{/if}
		</form>

		<!-- Key List -->
		<div class="rounded-xl overflow-hidden" style="background: var(--color-surface); border: 1px solid var(--color-border);">
			<div class="px-4 py-3" style="border-bottom: 1px solid var(--color-border);">
				<h2 class="text-sm font-semibold" style="color: var(--color-text);">Active Keys</h2>
			</div>

			{#if loading}
				<div class="px-4 py-8 text-center">
					<p class="text-sm" style="color: var(--color-text-muted);">Loading...</p>
				</div>
			{:else if keys.length === 0}
				<div class="px-4 py-8 text-center">
					<Key size={24} style="color: var(--color-text-muted); margin: 0 auto 8px;" />
					<p class="text-sm" style="color: var(--color-text-muted);">No API keys yet</p>
				</div>
			{:else}
				{#each keys as key, i}
					<div
						class="flex items-center gap-3 px-4 py-3"
						style="{i > 0 ? `border-top: 1px solid var(--color-border);` : ''}"
					>
						<Key size={14} style="color: var(--color-text-muted); flex-shrink: 0;" />
						<div class="flex-1 min-w-0">
							<p class="text-sm font-medium truncate" style="color: var(--color-text);">{key.name}</p>
							<p class="text-xs" style="color: var(--color-text-muted);">
								{key.keyPrefix}... · Created {formatDate(key.createdAt)}
								{#if key.lastUsedAt} · Last used {formatDate(key.lastUsedAt)}{/if}
								{#if key.expiresAt} · Expires {formatDate(key.expiresAt)}{/if}
							</p>
						</div>
						<button
							onclick={() => handleDelete(key.id)}
							class="p-1.5 rounded-lg hover:bg-[var(--color-surface-hover)] transition-colors"
							style="color: var(--color-danger);"
							title="Revoke key"
						>
							<Trash2 size={14} />
						</button>
					</div>
				{/each}
			{/if}
		</div>
	</div>
</div>
