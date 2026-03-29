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
	}

	let { projectId }: Props = $props();
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

<div class="activity-feed">
	<div class="activity-header">
		<Clock size={14} />
		<span>Activity</span>
	</div>

	{#if loading}
		<p class="activity-empty">Loading...</p>
	{:else if entries.length === 0}
		<p class="activity-empty">No activity yet</p>
	{:else}
		<div class="activity-list">
			{#each entries as entry (entry.id)}
				<div class="activity-item">
					<div class="activity-actor">{entry.userName}</div>
					<div class="activity-action">{formatAction(entry)}</div>
					<div class="activity-time">{timeAgo(entry.createdAt)}</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.activity-feed {
		border-top: 1px solid var(--color-border-subtle);
		padding: 0.75rem 0;
	}
	.activity-header {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0 1rem 0.5rem;
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}
	.activity-empty {
		padding: 1rem;
		text-align: center;
		font-size: 0.75rem;
		color: var(--color-text-dim);
	}
	.activity-list {
		max-height: 300px;
		overflow-y: auto;
	}
	.activity-item {
		padding: 0.5rem 1rem;
		border-bottom: 1px solid var(--color-border-subtle);
	}
	.activity-item:last-child {
		border-bottom: none;
	}
	.activity-actor {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-text);
	}
	.activity-action {
		font-size: 0.6875rem;
		color: var(--color-text-dim);
		margin-top: 0.125rem;
	}
	.activity-time {
		font-size: 0.625rem;
		color: var(--color-text-dim);
		margin-top: 0.125rem;
		opacity: 0.7;
	}
</style>
