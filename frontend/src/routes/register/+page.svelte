<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';

	let email = $state('');
	let password = $state('');
	let displayName = $state('');
	let error = $state('');
	let submitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		submitting = true;
		error = '';
		const err = await authStore.register(email, password, displayName);
		if (err) {
			error = err;
			submitting = false;
		} else {
			goto('/dashboard');
		}
	}
</script>

<div class="flex items-center justify-center min-h-screen">
	<div class="w-full max-w-sm p-8 rounded-xl bg-[var(--color-surface)] border border-[var(--color-border)]">
		<h1 class="text-2xl font-bold mb-6 text-center">Thask</h1>
		<p class="text-sm text-[var(--color-text-muted)] text-center mb-6">Create your account</p>

		{#if error}
			<div class="mb-4 p-3 rounded-lg bg-red-500/10 text-red-400 text-sm">{error}</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-4">
			<div>
				<label for="name" class="block text-sm mb-1 text-[var(--color-text-muted)]">Display Name</label>
				<input
					id="name"
					type="text"
					bind:value={displayName}
					required
					class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] focus:outline-none focus:border-[var(--color-primary)]"
				/>
			</div>
			<div>
				<label for="email" class="block text-sm mb-1 text-[var(--color-text-muted)]">Email</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] focus:outline-none focus:border-[var(--color-primary)]"
				/>
			</div>
			<div>
				<label for="password" class="block text-sm mb-1 text-[var(--color-text-muted)]">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					required
					minlength="8"
					class="w-full px-3 py-2 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] text-[var(--color-text)] focus:outline-none focus:border-[var(--color-primary)]"
				/>
			</div>
			<button
				type="submit"
				disabled={submitting}
				class="w-full py-2 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white font-medium disabled:opacity-50 transition-colors"
			>
				{submitting ? 'Creating account...' : 'Create account'}
			</button>
		</form>

		<p class="mt-4 text-center text-sm text-[var(--color-text-muted)]">
			Already have an account? <a href="/login" class="text-[var(--color-primary)] hover:underline">Sign in</a>
		</p>
	</div>
</div>
