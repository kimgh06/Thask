<script lang="ts">
	const nodeTypes = [
		{ type: 'FLOW', color: '#e2b340', desc: 'Product flows' },
		{ type: 'BRANCH', color: '#a78bfa', desc: 'Decision points' },
		{ type: 'TASK', color: '#7ca3c4', desc: 'Work items' },
		{ type: 'BUG', color: '#e05252', desc: 'Known issues' },
		{ type: 'API', color: '#5ea87a', desc: 'Endpoints' },
		{ type: 'UI', color: '#d4915a', desc: 'Screens' },
		{ type: 'GROUP', color: '#7c7570', desc: 'Containers' },
	];

	const features = [
		{ title: '7 Node Types', desc: 'Each with distinct shape and color for instant recognition.' },
		{ title: '5 Edge Types', desc: 'blocks, depends_on, triggers, related, parent_child.' },
		{ title: 'Impact Mode', desc: 'BFS-powered blast radius analysis from any node.' },
		{ title: 'CLI & MCP', desc: 'Terminal-first workflow. Claude Code and Cursor integration.' },
		{ title: 'Project Sharing', desc: 'Share graphs via link. Viewer or editor access.' },
		{ title: 'Self-Hosted', desc: 'One command. Your data stays on your server.' },
	];
</script>

<svelte:head>
	<title>Thask — Map dependencies visually. Break nothing.</title>
	<meta
		name="description"
		content="Dependency visualization for AI-assisted development. Map flows, tasks, and bugs as a graph. CLI + MCP for Claude Code & Cursor. Self-hosted, open-source."
	/>
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet" />
</svelte:head>

<div class="landing">
	<!-- Nav -->
	<nav class="nav">
		<a href="/" class="nav-brand">
			<img src="/icon.svg" alt="Thask" class="nav-icon" />
			<span class="nav-name">Thask</span>
		</a>
		<div class="nav-links">
			<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener" class="nav-link">GitHub</a>
			<a href="/login" class="nav-cta">Sign in</a>
		</div>
	</nav>

	<!-- Hero -->
	<section class="hero">
		<div class="hero-row">
			<div class="hero-content">
				<p class="hero-label">DEPENDENCY VISUALIZATION FOR AI-ASSISTED DEV</p>
				<h1 class="hero-title">
					Map dependencies visually.<br /><span class="hero-accent">Break nothing.</span>
				</h1>
				<p class="hero-desc">
					Nodes for every flow, task, and bug. Edges for every dependency.<br />
					Works with Claude Code & Cursor via MCP.
				</p>
				<div class="hero-actions">
					<a href="/login" class="btn-primary">Open Editor</a>
					<a href="https://thask.kimgh06.com/shared/562a734b85200bbdd65a65e18e066dc377b7dcfd3c86edb1eab4e6aece9a9bbf" target="_blank" rel="noopener" class="btn-ghost">
						Live Demo
					</a>
					<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener" class="btn-ghost">
						GitHub
					</a>
				</div>
			</div>
			<div class="hero-visual">
				<img src="/mascot.png" alt="Thask mascot" class="hero-mascot" />
			</div>
		</div>

		<!-- Live embeds -->
		<div class="hero-embeds">
			<div class="hero-embed">
				<h3 class="embed-title">Documentation Graph</h3>
				<div class="embed-frame">
					<iframe
						src="https://thask.kimgh06.com/shared/562a734b85200bbdd65a65e18e066dc377b7dcfd3c86edb1eab4e6aece9a9bbf"
						title="Thask Documentation Graph"
						loading="lazy"
					></iframe>
				</div>
				<p class="embed-caption">
					Docs dependency map — built with <code>thask</code> CLI.
					<a href="https://thask.kimgh06.com/shared/562a734b85200bbdd65a65e18e066dc377b7dcfd3c86edb1eab4e6aece9a9bbf" target="_blank" rel="noopener">Full view</a>
				</p>
			</div>
			<div class="hero-embed">
				<h3 class="embed-title">Architecture Graph</h3>
				<div class="embed-frame">
					<iframe
						src="https://thask.kimgh06.com/shared/de5cb3d3587d2479d6c875f873bd6479f031187e4054fd8cc93eafca8c840691"
						title="Thask Architecture Graph"
						loading="lazy"
					></iframe>
				</div>
				<p class="embed-caption">
					Full service flow — Frontend → Middleware → Handlers → Services → DB.
					<a href="https://thask.kimgh06.com/shared/de5cb3d3587d2479d6c875f873bd6479f031187e4054fd8cc93eafca8c840691" target="_blank" rel="noopener">Full view</a>
				</p>
			</div>
		</div>

		<!-- Node type strip -->
		<div class="node-strip">
			{#each nodeTypes as n}
				<div class="node-chip">
					<span class="node-dot" style="background:{n.color}"></span>
					<span class="node-type">{n.type}</span>
					<span class="node-desc">{n.desc}</span>
				</div>
			{/each}
		</div>
	</section>

	<!-- Features grid -->
	<section class="features">
		<div class="features-grid">
			{#each features as f}
				<div class="feature-card">
					<h3>{f.title}</h3>
					<p>{f.desc}</p>
				</div>
			{/each}
		</div>
	</section>

	<!-- Quick start -->
	<section class="quickstart">
		<div class="terminal">
			<div class="terminal-bar">
				<span class="terminal-dot" style="background:#e05252"></span>
				<span class="terminal-dot" style="background:#e2b340"></span>
				<span class="terminal-dot" style="background:#5ea87a"></span>
				<span class="terminal-title">terminal</span>
			</div>
			<div class="terminal-body">
				<p><span class="t-prompt">$</span> make up</p>
				<p class="t-out t-ok">&#10003; postgres  :7242</p>
				<p class="t-out t-ok">&#10003; backend   :7244</p>
				<p class="t-out t-ok">&#10003; frontend  :7243</p>
				<p class="t-out">&nbsp;</p>
				<p class="t-out">Open <span class="t-link">http://localhost:7243</span></p>
			</div>
		</div>
	</section>

	<!-- Footer -->
	<footer class="footer">
		<div class="footer-inner">
			<span class="footer-brand">Thask</span>
			<span class="footer-copy">MIT License</span>
			<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener" class="footer-link">GitHub</a>
		</div>
	</footer>
</div>

<style>
	/* ===== BASE ===== */
	.landing {
		min-height: 100vh;
		background: var(--color-bg);
		color: var(--color-text);
	}

	/* ===== NAV ===== */
	.nav {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 100;
		height: 56px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 2rem;
		background: rgba(19, 18, 20, 0.9);
		backdrop-filter: blur(8px);
		border-bottom: 1px solid var(--color-border-subtle);
	}
	.nav-brand {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		text-decoration: none;
		color: var(--color-text);
	}
	.nav-icon {
		width: 28px;
		height: 28px;
		border-radius: 6px;
	}
	.nav-name {
		font-weight: 700;
		font-size: 1rem;
		letter-spacing: -0.01em;
	}
	.nav-links {
		display: flex;
		align-items: center;
		gap: 1.5rem;
	}
	.nav-link {
		color: var(--color-text-muted);
		text-decoration: none;
		font-size: 0.8125rem;
		font-weight: 500;
		transition: color 0.15s;
	}
	.nav-link:hover {
		color: var(--color-text);
	}
	.nav-cta {
		color: var(--color-bg);
		background: var(--color-accent);
		padding: 0.375rem 1rem;
		border-radius: 6px;
		font-size: 0.8125rem;
		font-weight: 600;
		text-decoration: none;
		transition: background 0.15s;
	}
	.nav-cta:hover {
		background: var(--color-accent-hover);
	}

	/* ===== HERO ===== */
	.hero {
		padding: 8rem 2rem 4rem;
		max-width: 960px;
		margin: 0 auto;
	}
	.hero-row {
		display: flex;
		align-items: center;
		gap: 3rem;
		margin-bottom: 4rem;
	}
	.hero-content {
		flex: 1;
		min-width: 0;
	}
	.hero-visual {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.hero-mascot {
		width: 200px;
		height: auto;
		filter: drop-shadow(0 0 28px rgba(201, 168, 76, 0.12));
		animation: slideIn 0.8s ease-out both;
	}
	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateX(40px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}
	.hero-label {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.6875rem;
		font-weight: 500;
		letter-spacing: 0.12em;
		color: var(--color-accent);
		margin-bottom: 1.25rem;
		text-transform: uppercase;
	}
	.hero-title {
		font-size: 3.25rem;
		font-weight: 800;
		line-height: 1.1;
		letter-spacing: -0.03em;
		margin-bottom: 1.5rem;
	}
	.hero-accent {
		color: var(--color-accent);
	}
	.hero-desc {
		font-size: 1.0625rem;
		line-height: 1.7;
		color: var(--color-text-muted);
		max-width: 520px;
	}
	.hero-actions {
		display: flex;
		gap: 0.75rem;
		margin-top: 2.5rem;
	}

	/* Buttons */
	.btn-primary {
		display: inline-flex;
		align-items: center;
		padding: 0.625rem 1.5rem;
		background: var(--color-accent);
		color: var(--color-bg);
		font-weight: 600;
		font-size: 0.875rem;
		border-radius: 6px;
		text-decoration: none;
		transition: background 0.15s, transform 80ms;
	}
	.btn-primary:hover {
		background: var(--color-accent-hover);
	}
	.btn-primary:active {
		transform: scale(0.97);
	}
	.btn-ghost {
		display: inline-flex;
		align-items: center;
		padding: 0.625rem 1.5rem;
		background: transparent;
		color: var(--color-text-muted);
		font-weight: 500;
		font-size: 0.875rem;
		border-radius: 6px;
		border: 1px solid var(--color-border);
		text-decoration: none;
		transition: color 0.15s, border-color 0.15s;
	}
	.btn-ghost:hover {
		color: var(--color-text);
		border-color: var(--color-text-dim);
	}

	/* Live embeds */
	.hero-embeds {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.5rem;
		margin-bottom: 3rem;
	}
	.hero-embed {
	}
	.embed-title {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-text-muted);
		letter-spacing: 0.04em;
		margin-bottom: 0.5rem;
	}
	.embed-frame {
		border: 1px solid var(--color-border);
		border-radius: 8px;
		overflow: hidden;
		aspect-ratio: 16 / 9;
	}
	.embed-frame iframe {
		width: 100%;
		height: 100%;
		border: none;
		background: var(--color-bg);
	}
	.embed-caption {
		margin-top: 0.75rem;
		font-size: 0.8125rem;
		color: var(--color-text-dim);
		text-align: center;
	}
	.embed-caption code {
		font-family: 'JetBrains Mono', monospace;
		color: var(--color-accent);
		font-size: 0.75rem;
	}
	.embed-caption a {
		color: var(--color-accent);
		text-decoration: none;
	}
	.embed-caption a:hover {
		text-decoration: underline;
	}

	/* Node strip */
	.node-strip {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
	.node-chip {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.875rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border-subtle);
		border-radius: 6px;
	}
	.node-dot {
		width: 8px;
		height: 8px;
		border-radius: 2px;
		flex-shrink: 0;
	}
	.node-type {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		color: var(--color-text);
	}
	.node-desc {
		font-size: 0.75rem;
		color: var(--color-text-dim);
	}

	/* ===== FEATURES ===== */
	.features {
		padding: 2rem 2rem 5rem;
		max-width: 860px;
		margin: 0 auto;
	}
	.features-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1px;
		background: var(--color-border-subtle);
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		overflow: hidden;
	}
	.feature-card {
		padding: 1.5rem;
		background: var(--color-bg);
	}
	.feature-card h3 {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--color-text);
		margin-bottom: 0.375rem;
		letter-spacing: -0.01em;
	}
	.feature-card p {
		font-size: 0.8125rem;
		color: var(--color-text-dim);
		line-height: 1.5;
	}

	/* ===== QUICKSTART ===== */
	.quickstart {
		padding: 0 2rem 5rem;
		max-width: 560px;
		margin: 0 auto;
	}
	.terminal {
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		overflow: hidden;
		background: var(--color-surface);
	}
	.terminal-bar {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--color-border-subtle);
	}
	.terminal-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
	}
	.terminal-title {
		font-size: 0.6875rem;
		color: var(--color-text-dim);
		margin-left: 0.5rem;
	}
	.terminal-body {
		padding: 1.25rem 1.5rem;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		line-height: 1.9;
	}
	.t-prompt {
		color: var(--color-accent);
		margin-right: 0.5rem;
	}
	.t-out {
		color: var(--color-text-dim);
		padding-left: 1.25rem;
	}
	.t-ok {
		color: var(--color-success);
	}
	.t-link {
		color: var(--color-accent);
	}

	/* ===== FOOTER ===== */
	.footer {
		border-top: 1px solid var(--color-border-subtle);
		padding: 1.5rem 2rem;
	}
	.footer-inner {
		max-width: 860px;
		margin: 0 auto;
		display: flex;
		align-items: center;
		gap: 1.5rem;
	}
	.footer-brand {
		font-weight: 700;
		font-size: 0.8125rem;
	}
	.footer-copy {
		color: var(--color-text-dim);
		font-size: 0.75rem;
	}
	.footer-link {
		color: var(--color-text-dim);
		font-size: 0.75rem;
		text-decoration: none;
		margin-left: auto;
		transition: color 0.15s;
	}
	.footer-link:hover {
		color: var(--color-text);
	}

	/* ===== RESPONSIVE ===== */
	@media (max-width: 768px) {
		.hero {
			padding: 6rem 1.25rem 3rem;
		}
		.hero-row {
			flex-direction: column-reverse;
			gap: 2rem;
			text-align: center;
		}
		.hero-mascot {
			width: 140px;
		}
		.hero-actions {
			justify-content: center;
		}
		.hero-title {
			font-size: 2.25rem;
		}
		.hero-desc br {
			display: none;
		}
		.hero-embeds {
			grid-template-columns: 1fr;
		}
		.features-grid {
			grid-template-columns: 1fr;
		}
		.node-strip {
			gap: 0.375rem;
		}
		.node-desc {
			display: none;
		}
		.features,
		.quickstart {
			padding-left: 1.25rem;
			padding-right: 1.25rem;
		}
	}

	@media (max-width: 480px) {
		.hero-title {
			font-size: 1.75rem;
		}
		.hero-actions {
			flex-direction: column;
		}
		.btn-primary,
		.btn-ghost {
			width: 100%;
			justify-content: center;
		}
		.features-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
