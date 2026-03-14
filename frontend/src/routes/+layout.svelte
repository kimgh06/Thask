<script lang="ts">
	import '../app.css';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		authStore.fetchUser();
	});
</script>

{@render children()}
