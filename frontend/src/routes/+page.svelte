<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';

	$effect(() => {
		if (!authStore.loading) {
			goto(authStore.isAuthenticated ? '/dashboard' : '/login');
		}
	});
</script>

<div class="flex items-center justify-center min-h-screen">
	<p class="text-[var(--color-text-muted)]">Loading...</p>
</div>
