<script lang="ts">
	import { tick } from 'svelte';
	import { Check, Pencil, Bold, Code, SquareCode, List, Heading2, Link } from 'lucide-svelte';
	import { renderMarkdown } from '$lib/markdown';

	interface Props {
		value: string;
		onsave: (value: string) => void;
		readonly?: boolean;
		placeholder?: string;
	}

	let { value = $bindable(), onsave, readonly = false, placeholder = '' }: Props = $props();

	let editing = $state(false);
	let textareaEl = $state<HTMLTextAreaElement | undefined>();

	function handleBlur() {
		onsave(value);
		editing = false;
	}

	function handleKeydown(e: KeyboardEvent) {
		// ⌘/Ctrl + Enter saves and exits to preview
		if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
			e.preventDefault();
			onsave(value);
			editing = false;
		}
	}

	function startEditing() {
		if (readonly) return;
		editing = true;
	}

	// --- Markdown authoring helpers -------------------------------------------
	// Buttons use onmousedown+preventDefault so the textarea keeps focus (no blur/save).
	async function restoreSelection(start: number, end: number) {
		await tick();
		textareaEl?.focus();
		textareaEl?.setSelectionRange(start, end);
	}

	function wrapSelection(before: string, after = before) {
		const ta = textareaEl;
		if (!ta) return;
		const s = ta.selectionStart;
		const e = ta.selectionEnd;
		const sel = value.slice(s, e);
		value = value.slice(0, s) + before + sel + after + value.slice(e);
		restoreSelection(s + before.length, e + before.length);
	}

	function prefixLine(prefix: string) {
		const ta = textareaEl;
		if (!ta) return;
		const s = ta.selectionStart;
		const lineStart = value.lastIndexOf('\n', s - 1) + 1;
		value = value.slice(0, lineStart) + prefix + value.slice(lineStart);
		restoreSelection(s + prefix.length, ta.selectionEnd + prefix.length);
	}

	function codeBlock() {
		const ta = textareaEl;
		if (!ta) return;
		const s = ta.selectionStart;
		const e = ta.selectionEnd;
		const sel = value.slice(s, e) || 'schema';
		const block = `\n\`\`\`\n${sel}\n\`\`\`\n`;
		value = value.slice(0, s) + block + value.slice(e);
		restoreSelection(s + 5, s + 5 + sel.length); // inside the fence, over the selection
	}

	let rendered = $derived(value ? renderMarkdown(value) : '');
</script>

<div class="md-editor">
	{#if editing}
		<div class="md-toolbar">
			<div class="md-format">
				<button class="md-toggle" title="Heading" onmousedown={(e) => e.preventDefault()} onclick={() => prefixLine('## ')}><Heading2 size={12} /></button>
				<button class="md-toggle" title="Bold" onmousedown={(e) => e.preventDefault()} onclick={() => wrapSelection('**')}><Bold size={12} /></button>
				<button class="md-toggle" title="Inline code" onmousedown={(e) => e.preventDefault()} onclick={() => wrapSelection('`')}><Code size={12} /></button>
				<button class="md-toggle" title="Code block" onmousedown={(e) => e.preventDefault()} onclick={codeBlock}><SquareCode size={12} /></button>
				<button class="md-toggle" title="List" onmousedown={(e) => e.preventDefault()} onclick={() => prefixLine('- ')}><List size={12} /></button>
				<button class="md-toggle" title="Link" onmousedown={(e) => e.preventDefault()} onclick={() => wrapSelection('[', '](url)')}><Link size={12} /></button>
			</div>
			<button
				class="md-done"
				onmousedown={(e) => e.preventDefault()}
				onclick={() => { onsave(value); editing = false; }}
				title="Save & preview (⌘/Ctrl + Enter)"
			>
				<Check size={12} /> Done
			</button>
		</div>
		<textarea
			bind:this={textareaEl}
			bind:value
			onblur={handleBlur}
			onkeydown={handleKeydown}
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
		justify-content: space-between;
		align-items: center;
		margin-bottom: 4px;
	}

	.md-format {
		display: flex;
		gap: 3px;
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

	.md-done {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		height: 24px;
		padding: 0 10px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 600;
		border: 1px solid var(--color-accent-muted, var(--color-border));
		background: color-mix(in srgb, var(--color-accent) 14%, var(--color-surface));
		color: var(--color-accent-hover, var(--color-text));
		cursor: pointer;
		transition: background 0.15s, border-color 0.15s;
	}

	.md-done:hover {
		background: color-mix(in srgb, var(--color-accent) 24%, var(--color-surface));
		border-color: var(--color-accent);
	}

	.md-textarea {
		width: 100%;
		padding: 8px 12px;
		border-radius: 8px;
		font-size: 14px;
		font-family: 'JetBrains Mono', monospace;
		line-height: 1.7;
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
		font-size: 14px;
		line-height: 1.75;
		color: var(--color-text);
		cursor: text;
		padding: 4px 0;
		/* Long unbroken tokens (e.g. {sellerId?,canvas:{screens[]}}) wrap cleanly in the
		   narrow side panel instead of overflowing or breaking mid-glyph. */
		overflow-wrap: anywhere;
		word-break: break-word;
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
		background: color-mix(in srgb, var(--color-accent) 12%, var(--color-bg));
		border: 1px solid color-mix(in srgb, var(--color-accent) 25%, transparent);
		color: var(--color-accent-hover, var(--color-text));
		padding: 0.1em 0.35em;
		border-radius: 4px;
		font-size: 0.9em;
		white-space: nowrap;
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
