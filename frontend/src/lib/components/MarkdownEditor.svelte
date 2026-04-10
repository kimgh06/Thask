<script lang="ts">
	import { Eye, Pencil } from 'lucide-svelte';
	import { renderMarkdown } from '$lib/markdown';

	interface Props {
		value: string;
		onsave: (value: string) => void;
		readonly?: boolean;
		placeholder?: string;
	}

	let { value = $bindable(), onsave, readonly = false, placeholder = '' }: Props = $props();

	let editing = $state(false);

	function handleBlur() {
		onsave(value);
		editing = false;
	}

	function startEditing() {
		if (readonly) return;
		editing = true;
	}

	let rendered = $derived(value ? renderMarkdown(value) : '');
</script>

<div class="md-editor">
	{#if editing}
		<div class="md-toolbar">
			<button
				class="md-toggle"
				onclick={() => { editing = false; }}
				title="Preview"
			>
				<Eye size={12} />
			</button>
		</div>
		<textarea
			bind:value
			onblur={handleBlur}
			{placeholder}
			rows="6"
			class="md-textarea"
		></textarea>
	{:else}
		{#if !readonly}
			<div class="md-toolbar">
				<button
					class="md-toggle"
					onclick={startEditing}
					title="Edit"
				>
					<Pencil size={12} />
				</button>
			</div>
		{/if}
		{#if rendered}
			<div
				class="markdown-body"
				role="document"
				ondblclick={startEditing}
			>
				{@html rendered}
			</div>
		{:else}
			<button
				class="md-empty"
				ondblclick={startEditing}
				type="button"
			>{readonly ? 'No description' : placeholder || 'Double-click to add description...'}</button>
		{/if}
	{/if}
</div>

<style>
	.md-editor {
		position: relative;
	}

	.md-toolbar {
		display: flex;
		justify-content: flex-end;
		margin-bottom: 4px;
	}

	.md-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border: 1px solid var(--color-border);
		border-radius: 4px;
		background: var(--color-surface);
		color: var(--color-text-muted);
		cursor: pointer;
		transition: color 0.15s, border-color 0.15s;
	}

	.md-toggle:hover {
		color: var(--color-text);
		border-color: var(--color-text-dim);
	}

	.md-toggle:disabled {
		opacity: 0.3;
		cursor: default;
	}

	.md-textarea {
		width: 100%;
		padding: 8px 12px;
		border-radius: 8px;
		font-size: 12px;
		font-family: 'JetBrains Mono', monospace;
		line-height: 1.6;
		outline: none;
		resize: vertical;
		background: var(--color-bg);
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.md-textarea:focus {
		border-color: var(--color-accent-muted);
	}

	.md-empty {
		font-size: 12px;
		color: var(--color-text-dim);
		font-style: italic;
		padding: 4px 0;
		margin: 0;
		background: none;
		border: none;
		cursor: pointer;
		text-align: left;
		width: 100%;
	}

	/* Markdown rendered output */
	.markdown-body {
		font-size: 12px;
		line-height: 1.6;
		color: var(--color-text);
		cursor: text;
		padding: 4px 0;
	}

	.markdown-body :global(h1),
	.markdown-body :global(h2),
	.markdown-body :global(h3) {
		font-weight: 600;
		color: var(--color-text);
		margin-top: 0.75em;
		margin-bottom: 0.25em;
	}

	.markdown-body :global(h1) { font-size: 1.15em; }
	.markdown-body :global(h2) { font-size: 1.05em; }
	.markdown-body :global(h3) { font-size: 1em; }

	.markdown-body :global(p) {
		margin: 0.4em 0;
	}

	.markdown-body :global(code) {
		font-family: 'JetBrains Mono', monospace;
		background: var(--color-bg);
		padding: 0.15em 0.35em;
		border-radius: 4px;
		font-size: 0.9em;
	}

	.markdown-body :global(pre) {
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		padding: 0.75em;
		overflow-x: auto;
		margin: 0.5em 0;
	}

	.markdown-body :global(pre code) {
		background: none;
		padding: 0;
	}

	.markdown-body :global(a) {
		color: var(--color-accent);
		text-decoration: underline;
		text-decoration-color: var(--color-accent-muted);
	}

	.markdown-body :global(a:hover) {
		color: var(--color-accent-hover);
	}

	.markdown-body :global(ul),
	.markdown-body :global(ol) {
		padding-left: 1.5em;
		margin: 0.5em 0;
	}

	.markdown-body :global(blockquote) {
		border-left: 2px solid var(--color-border);
		padding-left: 0.75em;
		color: var(--color-text-muted);
		margin: 0.5em 0;
	}

	.markdown-body :global(hr) {
		border: none;
		border-top: 1px solid var(--color-border);
		margin: 0.75em 0;
	}

	.markdown-body :global(table) {
		border-collapse: collapse;
		width: 100%;
		font-size: 0.9em;
		margin: 0.5em 0;
	}

	.markdown-body :global(th),
	.markdown-body :global(td) {
		border: 1px solid var(--color-border);
		padding: 0.35em 0.6em;
	}

	.markdown-body :global(th) {
		background: var(--color-surface-hover);
	}

	.markdown-body :global(del) {
		color: var(--color-text-dim);
	}

	.markdown-body :global(strong) {
		font-weight: 600;
	}
</style>
