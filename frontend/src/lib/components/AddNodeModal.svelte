<script lang="ts">
	import type { NodeType } from '$lib/types';

	interface Props {
		onsubmit: (data: { title: string; type: NodeType }) => void;
		onclose: () => void;
	}

	let { onsubmit, onclose }: Props = $props();

	const NODE_TYPES: NodeType[] = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI'];

	const TYPE_COLORS: Record<NodeType, string> = {
		FLOW: '#6366f1',
		BRANCH: '#8b5cf6',
		TASK: '#3b82f6',
		BUG: '#ef4444',
		API: '#22c55e',
		UI: '#f59e0b',
		GROUP: '#64748b',
	};

	let title = $state('');
	let selectedType = $state<NodeType>('TASK');

	function handleSubmit() {
		const trimmed = title.trim();
		if (!trimmed) return;
		onsubmit({ title: trimmed, type: selectedType });
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onclose();
		}
	}

	function handleOverlayClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onclose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 flex items-center justify-center z-50"
	style="background: rgba(0,0,0,0.6);"
	onclick={handleOverlayClick}
>
	<div
		class="w-full max-w-md rounded-xl p-6 shadow-2xl flex flex-col gap-4"
		style="background: var(--color-surface); border: 1px solid var(--color-border);"
	>
		<h2 class="text-lg font-semibold" style="color: var(--color-text);">Add Node</h2>

		<!-- Title input -->
		<div class="flex flex-col gap-1">
			<label class="text-sm font-medium" style="color: var(--color-text-muted);" for="node-title">
				Title <span style="color: var(--color-danger);">*</span>
			</label>
			<input
				id="node-title"
				bind:value={title}
				placeholder="Enter node title..."
				class="px-3 py-2 rounded-lg text-sm outline-none"
				style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
				onkeydown={(e) => { if (e.key === 'Enter') handleSubmit(); }}
			/>
		</div>

		<!-- Type selection -->
		<div class="flex flex-col gap-2">
			<span class="text-sm font-medium" style="color: var(--color-text-muted);">Type</span>
			<div class="grid grid-cols-3 gap-2">
				{#each NODE_TYPES as type}
					<button
						onclick={() => (selectedType = type)}
						class="py-2 px-3 rounded-lg text-sm font-medium transition-all flex items-center gap-2"
						style="
							background: {selectedType === type ? TYPE_COLORS[type] + '22' : 'var(--color-bg)'};
							color: {selectedType === type ? TYPE_COLORS[type] : 'var(--color-text-muted)'};
							border: 1.5px solid {selectedType === type ? TYPE_COLORS[type] : 'var(--color-border)'};
						"
					>
						<span
							class="w-2.5 h-2.5 rounded-full flex-shrink-0"
							style="background: {TYPE_COLORS[type]};"
						></span>
						{type}
					</button>
				{/each}
			</div>
		</div>

		<!-- Actions -->
		<div class="flex justify-end gap-2 pt-1">
			<button
				onclick={onclose}
				class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
				style="background: var(--color-bg); color: var(--color-text-muted); border: 1px solid var(--color-border);"
			>
				Cancel
			</button>
			<button
				onclick={handleSubmit}
				disabled={!title.trim()}
				class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
				style="background: {title.trim() ? 'var(--color-primary)' : 'var(--color-surface-hover)'}; color: {title.trim() ? 'white' : 'var(--color-text-muted)'}; opacity: {title.trim() ? '1' : '0.6'};"
			>
				Create Node
			</button>
		</div>
	</div>
</div>
