<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { authStore } from '$lib/stores/auth.svelte';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		authStore.fetchUser();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
