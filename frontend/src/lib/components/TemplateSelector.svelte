<script lang="ts">
	import { api } from '$lib/api';

	interface Template {
		name: string;
		description: string;
		nodes: any[];
		edges: any[];
	}

	interface Props {
		projectId: string;
		onApplied: () => void;
		onclose: () => void;
	}

	let { projectId, onApplied, onclose }: Props = $props();

	const templates = [
		{ file: 'sprint-board.json', name: 'Sprint Board', description: 'Kanban-style sprint with tasks, bugs, and milestones' },
		{ file: 'microservice-map.json', name: 'Microservice Map', description: 'Service dependency map with API gateway and data layer' },
		{ file: 'api-flow.json', name: 'API Flow', description: 'Request lifecycle from client to database and back' },
		{ file: 'handoff-flow.json', name: 'Team Handoff Starter', description: 'Self-documenting template with Handoff Convention examples for onboarding and knowledge transfer' },
	];

	let loading = $state(false);

	async function apply(file: string) {
		loading = true;
		try {
			const resp = await fetch(`/templates/${file}`);
			const template: Template = await resp.json();

			await api.post(`/api/projects/${projectId}/graph/import`, {
				mode: 'merge',
				nodes: template.nodes,
				edges: template.edges,
			});

			onApplied();
		} catch (e) {
			console.error('Failed to apply template:', e);
		} finally {
			loading = false;
		}
	}
</script>

<div class="template-overlay" role="button" tabindex="-1" onclick={onclose} onkeydown={(e) => e.key === 'Escape' && onclose()}>
	<div class="template-modal" onclick={(e) => e.stopPropagation()}>
		<h2 class="template-title">Choose a Template</h2>
		<p class="template-subtitle">Start with a pre-built graph structure</p>

		<div class="template-grid">
			{#each templates as t}
				<button
					class="template-card"
					onclick={() => apply(t.file)}
					disabled={loading}
				>
					<h3>{t.name}</h3>
					<p>{t.description}</p>
				</button>
			{/each}
		</div>

		<div class="template-actions">
			<button class="template-skip" onclick={onclose}>Skip</button>
		</div>
	</div>
</div>

<style>
	.template-overlay {
		position: fixed;
		inset: 0;
		z-index: 200;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(0, 0, 0, 0.6);
		backdrop-filter: blur(4px);
	}
	.template-modal {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 12px;
		padding: 2rem;
		max-width: 560px;
		width: 90vw;
	}
	.template-title {
		font-size: 1.125rem;
		font-weight: 700;
		color: var(--color-text);
		margin-bottom: 0.25rem;
	}
	.template-subtitle {
		font-size: 0.8125rem;
		color: var(--color-text-muted);
		margin-bottom: 1.5rem;
	}
	.template-grid {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.template-card {
		text-align: left;
		padding: 1rem 1.25rem;
		background: var(--color-bg);
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s;
	}
	.template-card:hover:not(:disabled) {
		border-color: var(--color-accent);
		background: var(--color-surface-hover);
	}
	.template-card:disabled {
		opacity: 0.5;
		cursor: wait;
	}
	.template-card h3 {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--color-text);
		margin-bottom: 0.25rem;
	}
	.template-card p {
		font-size: 0.75rem;
		color: var(--color-text-dim);
	}
	.template-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 1.25rem;
	}
	.template-skip {
		padding: 0.5rem 1rem;
		border-radius: 6px;
		border: 1px solid var(--color-border);
		background: transparent;
		color: var(--color-text-muted);
		font-size: 0.8125rem;
		cursor: pointer;
		transition: color 0.15s;
	}
	.template-skip:hover {
		color: var(--color-text);
	}
</style>
