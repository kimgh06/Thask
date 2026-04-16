<script lang="ts">
	import { api } from '$lib/api';
	import { Clock } from 'lucide-svelte';

	interface ActivityEntry {
		id: string;
		action: string;
		fieldName: string | null;
		oldValue: string | null;
		newValue: string | null;
		createdAt: string;
		userName: string;
	}

	interface Props {
		projectId: string;
		expanded?: boolean;
	}

	let { projectId, expanded = false }: Props = $props();
	let entries = $state<ActivityEntry[]>([]);
	let loading = $state(true);

	$effect(() => {
		if (!projectId) return;
		loading = true;
		api.get<ActivityEntry[]>(`/api/projects/${projectId}/activity?limit=20`)
			.then((res) => {
				if (res.data) entries = res.data;
				loading = false;
			})
			.catch(() => {
				loading = false;
			});
	});

	// Refresh when SSE events arrive (called from parent)
	export function refresh() {
		api.get<ActivityEntry[]>(`/api/projects/${projectId}/activity?limit=20`).then((res) => {
			if (res.data) entries = res.data;
		});
	}

	function formatAction(entry: ActivityEntry): string {
		switch (entry.action) {
			case 'created':
				return 'created a node';
			case 'deleted':
				return 'deleted a node';
			case 'status_changed':
				return `changed status: ${entry.oldValue} → ${entry.newValue}`;
			case 'updated':
				if (entry.fieldName) return `updated ${entry.fieldName}`;
				return 'updated a node';
			case 'edge_created':
				return 'created an edge';
			case 'edge_deleted':
				return 'deleted an edge';
			case 'tag_added':
				return `added tag: ${entry.newValue}`;
			case 'tag_removed':
				return `removed tag: ${entry.oldValue}`;
			case 'type_changed':
				return `changed type: ${entry.oldValue} → ${entry.newValue}`;
			case 'title_updated':
				return `renamed: ${entry.oldValue} → ${entry.newValue}`;
			case 'description_updated':
				return 'updated description';
			case 'node_moved':
				return 'moved node';
			default:
				return entry.action;
		}
	}

	function timeAgo(dateStr: string): string {
		const diff = Date.now() - new Date(dateStr).getTime();
		const mins = Math.floor(diff / 60000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		const days = Math.floor(hrs / 24);
		return `${days}d ago`;
	}
</script>

<div class="border-t" style="border-color: var(--color-border-subtle); padding: 0.75rem 0;">
	<div class="flex items-center gap-2 px-4 pb-2" style="font-size: 0.6875rem; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; color: var(--color-text-muted);">
		<Clock size={14} />
		<span>Activity</span>
	</div>

	{#if loading}
		<p class="px-4 py-4 text-center text-xs" style="color: var(--color-text-dim);">Loading...</p>
	{:else if entries.length === 0}
		<p class="px-4 py-4 text-center text-xs" style="color: var(--color-text-dim);">No activity yet</p>
	{:else}
		<div style="{expanded ? 'overflow-y: auto;' : 'max-height: 300px; overflow-y: auto;'}">
			{#each entries as entry (entry.id)}
				<div class="px-4 py-2 border-b" style="border-color: var(--color-border-subtle);">
					<div class="text-xs font-semibold" style="color: var(--color-text);">{entry.userName}</div>
					<div class="text-[0.6875rem] mt-0.5" style="color: var(--color-text-dim);">{formatAction(entry)}</div>
					<div class="text-[0.625rem] mt-0.5" style="color: var(--color-text-dim); opacity: 0.7;">{timeAgo(entry.createdAt)}</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
