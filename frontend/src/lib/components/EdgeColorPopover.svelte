<script lang="ts">
	import type { EdgeType } from '$lib/types';

	interface Props {
		position: { x: number; y: number };
		currentLabel: string;
		onselect: (edgeType: EdgeType) => void;
		onupdatelabel: (label: string) => void;
		ondelete: () => void;
		oncancel: () => void;
	}

	let { position, currentLabel, onselect, onupdatelabel, ondelete, oncancel }: Props = $props();

	const EDGE_TYPES: { value: EdgeType; label: string; color: string }[] = [
		{ value: 'depends_on', label: 'Depends On', color: '#6366f1' },
		{ value: 'blocks', label: 'Blocks', color: '#ef4444' },
		{ value: 'related', label: 'Related', color: '#64748b' },
		{ value: 'parent_child', label: 'Parent / Child', color: '#22c55e' },
		{ value: 'triggers', label: 'Triggers', color: '#f59e0b' },
	];

	let label = $state('');
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		label = currentLabel;
	});

	function handleLabelInput() {
		if (debounceTimer) clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			onupdatelabel(label);
		}, 400);
	}

	function handleOverlayClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			oncancel();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			oncancel();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-40"
	style="background: transparent;"
	onclick={handleOverlayClick}
>
	<div
		class="absolute rounded-xl p-4 shadow-2xl flex flex-col gap-3 w-56"
		style="
			left: {position.x}px;
			top: {position.y}px;
			background: var(--color-surface);
			border: 1px solid var(--color-border);
		"
		role="dialog"
		aria-label="Edge options"
	>
		<!-- Label input -->
		<div class="flex flex-col gap-1">
			<label class="text-xs font-medium" style="color: var(--color-text-muted);" for="edge-label">
				Label
			</label>
			<input
				id="edge-label"
				bind:value={label}
				oninput={handleLabelInput}
				placeholder="Edge label..."
				class="px-2 py-1.5 rounded text-xs outline-none"
				style="background: var(--color-bg); color: var(--color-text); border: 1px solid var(--color-border);"
			/>
		</div>

		<!-- Edge type selection -->
		<div class="flex flex-col gap-1">
			<span class="text-xs font-medium" style="color: var(--color-text-muted);">Type</span>
			<div class="flex flex-col gap-1">
				{#each EDGE_TYPES as et}
					<button
						onclick={() => onselect(et.value)}
						class="flex items-center gap-2 px-2 py-1.5 rounded text-xs font-medium text-left transition-colors"
						style="background: var(--color-bg); color: var(--color-text);"
						onmouseenter={(e) => {
							(e.currentTarget as HTMLElement).style.background = 'var(--color-surface-hover)';
						}}
						onmouseleave={(e) => {
							(e.currentTarget as HTMLElement).style.background = 'var(--color-bg)';
						}}
					>
						<span
							class="w-3 h-3 rounded-full flex-shrink-0"
							style="background: {et.color};"
						></span>
						{et.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Delete button -->
		<button
			onclick={ondelete}
			class="mt-1 px-3 py-1.5 rounded text-xs font-medium transition-colors"
			style="background: #ef444422; color: #ef4444; border: 1px solid #ef444444;"
			onmouseenter={(e) => {
				(e.currentTarget as HTMLElement).style.background = '#ef444433';
			}}
			onmouseleave={(e) => {
				(e.currentTarget as HTMLElement).style.background = '#ef444422';
			}}
		>
			Delete Edge
		</button>
	</div>
</div>
