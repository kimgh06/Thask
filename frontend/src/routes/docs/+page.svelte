<script lang="ts">
	import { onMount } from 'svelte';

	const nodeTypes = [
		{ type: 'FLOW',   color: '#e2b340', desc: 'Product flows' },
		{ type: 'BRANCH', color: '#a78bfa', desc: 'Decision points' },
		{ type: 'TASK',   color: '#7ca3c4', desc: 'Work items' },
		{ type: 'BUG',    color: '#e05252', desc: 'Known issues' },
		{ type: 'API',    color: '#5ea87a', desc: 'Endpoints' },
		{ type: 'UI',     color: '#d4915a', desc: 'Screens' },
		{ type: 'GROUP',  color: '#7c7570', desc: 'Containers' },
	];

	const edgeTypes = [
		{ type: 'depends_on',  desc: 'A cannot run without B' },
		{ type: 'blocks',      desc: 'A prevents B from starting' },
		{ type: 'triggers',    desc: 'A initiates B automatically' },
		{ type: 'related',     desc: 'Loose conceptual link' },
		{ type: 'parent_child',desc: 'Hierarchy / containment' },
	];

	const commands = [
		{ cmd: 'thask node list',   alias: 'thask n ls',      desc: 'List nodes in a project' },
		{ cmd: 'thask node create', alias: 'thask n c',       desc: 'Create a new node' },
		{ cmd: 'thask edge create', alias: 'thask e c',       desc: 'Create an edge between nodes' },
		{ cmd: 'thask graph get',   alias: 'thask g',         desc: 'Get full graph (nodes + edges)' },
		{ cmd: 'thask graph layout',alias: 'thask g layout',  desc: 'Apply auto-layout algorithm' },
		{ cmd: 'thask impact',      alias: 'thask im',        desc: 'Impact analysis from a node' },
	];

	const sections = [
		{ id: 'getting-started', label: 'Getting Started' },
		{ id: 'api-reference',   label: 'API Reference' },
		{ id: 'cli',             label: 'CLI' },
		{ id: 'concepts',        label: 'Concepts' },
	];

	let activeSection = $state('getting-started');
	let mobileOpen = $state(false);

	onMount(() => {
		const observers: IntersectionObserver[] = [];

		sections.forEach(({ id }) => {
			const el = document.getElementById(id);
			if (!el) return;
			const obs = new IntersectionObserver(
				(entries) => {
					entries.forEach((e) => {
						if (e.isIntersecting) activeSection = id;
					});
				},
				{ rootMargin: '-20% 0px -70% 0px', threshold: 0 }
			);
			obs.observe(el);
			observers.push(obs);
		});

		return () => observers.forEach((o) => o.disconnect());
	});

	function scrollTo(id: string) {
		const el = document.getElementById(id);
		if (el) {
			el.scrollIntoView({ behavior: 'smooth', block: 'start' });
		}
		mobileOpen = false;
	}
</script>

<svelte:head>
	<title>Docs — Thask</title>
	<meta name="description" content="Thask documentation: getting started, API reference, CLI, and core concepts." />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet" />
</svelte:head>

<div class="docs-page">

	<!-- Nav (identical to landing) -->
	<nav class="nav">
		<a href="/" class="nav-brand">
			<img src="/icon.svg" alt="Thask" class="nav-icon" />
			<span class="nav-name">Thask</span>
		</a>
		<div class="nav-links">
			<a href="/docs" class="nav-link nav-link--active">Docs</a>
			<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener" class="nav-link">GitHub</a>
			<a href="/login" class="nav-cta">Sign in</a>
		</div>
		<!-- Mobile hamburger -->
		<button
			class="nav-hamburger"
			aria-label="Toggle navigation"
			aria-expanded={mobileOpen}
			onclick={() => (mobileOpen = !mobileOpen)}
		>
			<span class="ham-line"></span>
			<span class="ham-line"></span>
			<span class="ham-line"></span>
		</button>
	</nav>

	<!-- Mobile sidebar overlay -->
	{#if mobileOpen}
		<div
			class="mobile-overlay"
			role="button"
			tabindex="-1"
			aria-label="Close navigation"
			onclick={() => (mobileOpen = false)}
			onkeydown={(e) => e.key === 'Escape' && (mobileOpen = false)}
		></div>
		<aside class="sidebar sidebar--mobile">
			<nav class="sidebar-nav" aria-label="Documentation sections">
				{#each sections as s}
					<button
						class="sidebar-item {activeSection === s.id ? 'sidebar-item--active' : ''}"
						onclick={() => scrollTo(s.id)}
					>
						{s.label}
					</button>
				{/each}
			</nav>
		</aside>
	{/if}

	<div class="docs-layout">

		<!-- Desktop sidebar -->
		<aside class="sidebar sidebar--desktop" aria-label="Documentation sections">
			<p class="sidebar-heading">DOCUMENTATION</p>
			<nav class="sidebar-nav">
				{#each sections as s}
					<button
						class="sidebar-item {activeSection === s.id ? 'sidebar-item--active' : ''}"
						onclick={() => scrollTo(s.id)}
					>
						{s.label}
					</button>
				{/each}
			</nav>
		</aside>

		<!-- Content -->
		<main class="content" id="main-content">

			<!-- ===== GETTING STARTED ===== -->
			<section id="getting-started" class="doc-section">
				<p class="section-label">01</p>
				<h2 class="section-title">Getting Started</h2>

				<h3 class="subsection-title">Quick Setup</h3>
				<p class="body-text">Clone the repo and run one command. Postgres, backend, and frontend all start together.</p>

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

				<h3 class="subsection-title">First steps</h3>
				<ol class="step-list">
					<li>
						<span class="step-num">1</span>
						<div>
							<strong>Create an account</strong>
							<p>Register at <code>/register</code>. No email confirmation required in self-hosted mode.</p>
						</div>
					</li>
					<li>
						<span class="step-num">2</span>
						<div>
							<strong>Create a team</strong>
							<p>Teams namespace your projects. Invite members from the team settings page.</p>
						</div>
					</li>
					<li>
						<span class="step-num">3</span>
						<div>
							<strong>Create a project</strong>
							<p>Projects hold graphs. Each project has its own set of nodes and edges.</p>
						</div>
					</li>
					<li>
						<span class="step-num">4</span>
						<div>
							<strong>Open the graph editor</strong>
							<p>Click a project to open the visual editor. Right-click canvas to create nodes. Drag from a node handle to draw edges.</p>
						</div>
					</li>
				</ol>
			</section>

			<div class="section-divider"></div>

			<!-- ===== API REFERENCE ===== -->
			<section id="api-reference" class="doc-section">
				<p class="section-label">02</p>
				<h2 class="section-title">API Reference</h2>

				<div class="info-row">
					<div class="info-item">
						<span class="info-key">Base URL</span>
						<code class="info-val">/api/v1</code>
					</div>
					<div class="info-item">
						<span class="info-key">Auth header</span>
						<code class="info-val">Authorization: Bearer thsk_...</code>
					</div>
					<div class="info-item">
						<span class="info-key">Version header</span>
						<code class="info-val">X-API-Version: v1</code>
					</div>
				</div>

				<h3 class="subsection-title">Interactive docs</h3>
				<p class="body-text">
					Explore and test all endpoints in the browser:
					<a href="/api/v1/docs" class="text-link">/api/v1/docs</a> (Scalar UI).
					Raw spec: <a href="/api/v1/openapi.yaml" class="text-link">/api/v1/openapi.yaml</a>.
				</p>

				<h3 class="subsection-title">Quick examples</h3>

				<div class="code-block">
					<div class="code-block__header">
						<span class="code-block__lang">bash</span>
					</div>
					<pre class="code-block__body"><code><span class="cm"># List nodes</span>
curl -H <span class="cs">"Authorization: Bearer thsk_..."</span> \
  /api/v1/projects/<span class="cv">{`{id}`}</span>/nodes

<span class="cm"># Create node</span>
curl -X POST \
  -H <span class="cs">"Authorization: Bearer thsk_..."</span> \
  -H <span class="cs">"Content-Type: application/json"</span> \
  /api/v1/projects/<span class="cv">{`{id}`}</span>/nodes \
  -d <span class="cs">'&#123;"type":"TASK","title":"My Task"&#125;'</span></code></pre>
				</div>

				<h3 class="subsection-title">Error format</h3>
				<p class="body-text">All errors return a consistent envelope:</p>
				<div class="code-block">
					<div class="code-block__header">
						<span class="code-block__lang">json</span>
					</div>
					<pre class="code-block__body"><code>&#123; "error": &#123; "code": "NOT_FOUND", "message": "project not found" &#125; &#125;</code></pre>
				</div>

				<h3 class="subsection-title">Idempotency</h3>
				<p class="body-text">
					Pass <code>Idempotency-Key: &lt;uuid&gt;</code> on any mutating request.
					The server deduplicates responses for 24 hours. Safe to retry on network failure.
				</p>
			</section>

			<div class="section-divider"></div>

			<!-- ===== CLI ===== -->
			<section id="cli" class="doc-section">
				<p class="section-label">03</p>
				<h2 class="section-title">CLI</h2>

				<h3 class="subsection-title">Install</h3>
				<div class="code-block">
					<div class="code-block__header">
						<span class="code-block__lang">bash</span>
					</div>
					<pre class="code-block__body"><code>go install github.com/kimgh06/Thask/cli@latest</code></pre>
				</div>

				<h3 class="subsection-title">Authenticate</h3>
				<div class="code-block">
					<div class="code-block__header">
						<span class="code-block__lang">bash</span>
					</div>
					<pre class="code-block__body"><code>thask auth login</code></pre>
				</div>
				<p class="body-text">Opens a browser prompt. Token is saved to <code>~/.config/thask/token</code>.</p>

				<h3 class="subsection-title">Commands</h3>
				<div class="table-wrap">
					<table class="cmd-table">
						<thead>
							<tr>
								<th>Command</th>
								<th>Alias</th>
								<th>Description</th>
							</tr>
						</thead>
						<tbody>
							{#each commands as c}
								<tr>
									<td><code>{c.cmd}</code></td>
									<td><code class="alias">{c.alias}</code></td>
									<td class="td-desc">{c.desc}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<h3 class="subsection-title">MCP integration</h3>
				<p class="body-text">
					Start the MCP server to give Claude Code or Cursor direct access to your graph:
				</p>
				<div class="code-block">
					<div class="code-block__header">
						<span class="code-block__lang">bash</span>
					</div>
					<pre class="code-block__body"><code>thask mcp serve</code></pre>
				</div>
				<p class="body-text">
					Add to your Claude Code <code>.mcp.json</code> or Cursor MCP config, then agents can
					query and modify graphs directly from chat.
				</p>
			</section>

			<div class="section-divider"></div>

			<!-- ===== CONCEPTS ===== -->
			<section id="concepts" class="doc-section">
				<p class="section-label">04</p>
				<h2 class="section-title">Concepts</h2>

				<h3 class="subsection-title">Node types</h3>
				<p class="body-text">Every node has a type that determines its color and visual shape in the editor.</p>
				<div class="node-grid">
					{#each nodeTypes as n}
						<div class="node-row">
							<span class="node-dot" style="background:{n.color}"></span>
							<code class="node-type">{n.type}</code>
							<span class="node-desc">{n.desc}</span>
						</div>
					{/each}
				</div>

				<h3 class="subsection-title">Edge types</h3>
				<div class="table-wrap">
					<table class="cmd-table">
						<thead>
							<tr>
								<th>Type</th>
								<th>Meaning</th>
							</tr>
						</thead>
						<tbody>
							{#each edgeTypes as e}
								<tr>
									<td><code>{e.type}</code></td>
									<td class="td-desc">{e.desc}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<h3 class="subsection-title">Impact analysis</h3>
				<p class="body-text">
					Select any node and open Impact mode. Thask runs a BFS traversal following
					<code>depends_on</code> and <code>blocks</code> edges to highlight every downstream node.
					Use it before making a change to see what breaks.
				</p>

				<h3 class="subsection-title">Groups</h3>
				<p class="body-text">
					GROUP nodes are containers. Drag nodes onto a GROUP to assign them as children.
					Collapse a group to hide its contents and reduce visual noise.
					Groups can be connected with edges like any other node.
				</p>

				<h3 class="subsection-title">Sharing</h3>
				<p class="body-text">
					Any project can be shared via a public link with <strong>viewer</strong> or <strong>editor</strong> access.
					Shared graphs can also be embedded in external pages with an <code>&lt;iframe&gt;</code>:
				</p>
				<div class="code-block">
					<div class="code-block__header">
						<span class="code-block__lang">html</span>
					</div>
					<pre class="code-block__body"><code>&lt;iframe src="https://your-instance/embed/&#123;shareToken&#125;" /&gt;</code></pre>
				</div>
			</section>

		</main>
	</div>

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
	/* ===== PAGE SHELL ===== */
	.docs-page {
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
		z-index: 200;
		height: 56px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 2rem;
		background: rgba(19, 18, 20, 0.92);
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
	.nav-link:hover { color: var(--color-text); }
	.nav-link--active { color: var(--color-accent); }
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
	.nav-cta:hover { background: var(--color-accent-hover); }

	.nav-hamburger {
		display: none;
		flex-direction: column;
		justify-content: center;
		gap: 5px;
		width: 36px;
		height: 36px;
		background: none;
		border: none;
		cursor: pointer;
		padding: 6px;
	}
	.ham-line {
		display: block;
		height: 1.5px;
		background: var(--color-text-muted);
		border-radius: 2px;
		transition: background 0.15s;
	}
	.nav-hamburger:hover .ham-line { background: var(--color-text); }

	/* ===== LAYOUT ===== */
	.docs-layout {
		display: flex;
		padding-top: 56px; /* nav height */
		min-height: calc(100vh - 56px);
	}

	/* ===== SIDEBAR ===== */
	.sidebar--desktop {
		position: sticky;
		top: 56px;
		width: 240px;
		flex-shrink: 0;
		height: calc(100vh - 56px);
		overflow-y: auto;
		padding: 2.5rem 0 2.5rem 2rem;
		border-right: 1px solid var(--color-border-subtle);
	}
	.sidebar--mobile {
		position: fixed;
		top: 56px;
		left: 0;
		bottom: 0;
		width: 240px;
		z-index: 150;
		background: var(--color-surface);
		border-right: 1px solid var(--color-border);
		padding: 2rem 0 2rem 1.5rem;
		overflow-y: auto;
	}
	.mobile-overlay {
		position: fixed;
		inset: 56px 0 0 0;
		z-index: 140;
		background: rgba(13, 12, 14, 0.6);
		cursor: pointer;
	}
	.sidebar-heading {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.625rem;
		font-weight: 600;
		letter-spacing: 0.12em;
		color: var(--color-text-dim);
		margin-bottom: 1.25rem;
		padding-left: 0.75rem;
	}
	.sidebar-nav {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.sidebar-item {
		display: block;
		width: 100%;
		text-align: left;
		padding: 0.5rem 0.75rem;
		border: none;
		background: none;
		border-radius: 5px;
		font-size: 0.8125rem;
		font-weight: 500;
		color: var(--color-text-muted);
		cursor: pointer;
		transition: color 0.12s, background 0.12s;
	}
	.sidebar-item:hover {
		color: var(--color-text);
		background: var(--color-surface-hover);
	}
	.sidebar-item--active {
		color: var(--color-accent);
		background: rgba(201, 168, 76, 0.08);
	}
	.sidebar-item--active:hover {
		color: var(--color-accent);
		background: rgba(201, 168, 76, 0.12);
	}

	/* ===== CONTENT ===== */
	.content {
		flex: 1;
		min-width: 0;
		padding: 3rem 3rem 5rem;
		max-width: 720px;
	}

	/* ===== SECTIONS ===== */
	.doc-section {
		scroll-margin-top: 80px;
	}
	.section-label {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.625rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		color: var(--color-accent);
		margin-bottom: 0.5rem;
	}
	.section-title {
		font-size: 1.875rem;
		font-weight: 800;
		letter-spacing: -0.03em;
		line-height: 1.15;
		margin-bottom: 2rem;
	}
	.subsection-title {
		font-size: 0.875rem;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: var(--color-text);
		margin-top: 2rem;
		margin-bottom: 0.75rem;
		font-family: 'JetBrains Mono', monospace;
	}
	.section-divider {
		height: 1px;
		background: var(--color-border-subtle);
		margin: 3.5rem 0;
	}

	/* ===== BODY TEXT ===== */
	.body-text {
		font-size: 0.9375rem;
		line-height: 1.75;
		color: var(--color-text-muted);
		margin-bottom: 1rem;
	}
	.body-text code,
	.body-text strong {
		color: var(--color-text);
	}
	.body-text code {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		background: var(--color-surface);
		padding: 0.125em 0.375em;
		border-radius: 4px;
		border: 1px solid var(--color-border-subtle);
	}
	.text-link {
		color: var(--color-accent);
		text-decoration: none;
	}
	.text-link:hover { text-decoration: underline; }

	/* ===== INFO PILLS ===== */
	.info-row {
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
		margin-bottom: 2rem;
		padding: 1.25rem 1.5rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
	}
	.info-item {
		display: flex;
		align-items: baseline;
		gap: 1rem;
	}
	.info-key {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.6875rem;
		font-weight: 500;
		letter-spacing: 0.06em;
		color: var(--color-text-dim);
		width: 100px;
		flex-shrink: 0;
	}
	.info-val {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		color: var(--color-text);
	}

	/* ===== TERMINAL ===== */
	.terminal {
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		overflow: hidden;
		background: var(--color-surface);
		margin-bottom: 1.5rem;
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
		font-family: 'JetBrains Mono', monospace;
	}
	.terminal-body {
		padding: 1.25rem 1.5rem;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		line-height: 1.9;
	}
	.t-prompt { color: var(--color-accent); margin-right: 0.5rem; }
	.t-out { color: var(--color-text-dim); padding-left: 1.25rem; }
	.t-ok { color: var(--color-success); }
	.t-link { color: var(--color-accent); }

	/* ===== CODE BLOCKS ===== */
	.code-block {
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		overflow: hidden;
		background: var(--color-surface);
		margin-bottom: 1.5rem;
	}
	.code-block__header {
		display: flex;
		align-items: center;
		padding: 0.5rem 1rem;
		border-bottom: 1px solid var(--color-border-subtle);
	}
	.code-block__lang {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.625rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		color: var(--color-text-dim);
		text-transform: uppercase;
	}
	.code-block__body {
		margin: 0;
		padding: 1.25rem 1.5rem;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		line-height: 1.8;
		color: var(--color-text);
		overflow-x: auto;
		white-space: pre;
	}
	/* Inline syntax tints */
	.cm { color: var(--color-text-dim); }      /* comment */
	.cs { color: var(--color-success); }        /* string */
	.cv { color: var(--color-accent); }         /* variable */

	/* ===== STEP LIST ===== */
	.step-list {
		list-style: none;
		padding: 0;
		margin: 0 0 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 1.25rem;
	}
	.step-list li {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
	}
	.step-num {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.6875rem;
		font-weight: 600;
		color: var(--color-accent);
		background: rgba(201, 168, 76, 0.1);
		border: 1px solid rgba(201, 168, 76, 0.2);
		border-radius: 4px;
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		margin-top: 2px;
	}
	.step-list strong {
		display: block;
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--color-text);
		margin-bottom: 0.25rem;
	}
	.step-list p {
		font-size: 0.875rem;
		color: var(--color-text-muted);
		line-height: 1.6;
		margin: 0;
	}
	.step-list code {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		color: var(--color-accent);
	}

	/* ===== NODE GRID ===== */
	.node-grid {
		display: flex;
		flex-direction: column;
		gap: 0;
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		overflow: hidden;
		margin-bottom: 2rem;
	}
	.node-row {
		display: flex;
		align-items: center;
		gap: 0.875rem;
		padding: 0.75rem 1.25rem;
		border-bottom: 1px solid var(--color-border-subtle);
	}
	.node-row:last-child { border-bottom: none; }
	.node-dot {
		width: 10px;
		height: 10px;
		border-radius: 3px;
		flex-shrink: 0;
	}
	.node-type {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.75rem;
		font-weight: 600;
		width: 80px;
		flex-shrink: 0;
		color: var(--color-text);
	}
	.node-desc {
		font-size: 0.8125rem;
		color: var(--color-text-muted);
	}

	/* ===== TABLE ===== */
	.table-wrap {
		overflow-x: auto;
		border: 1px solid var(--color-border-subtle);
		border-radius: 8px;
		margin-bottom: 2rem;
	}
	.cmd-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8125rem;
	}
	.cmd-table thead tr {
		border-bottom: 1px solid var(--color-border-subtle);
	}
	.cmd-table th {
		padding: 0.625rem 1.25rem;
		text-align: left;
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.625rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		color: var(--color-text-dim);
		text-transform: uppercase;
		white-space: nowrap;
	}
	.cmd-table td {
		padding: 0.75rem 1.25rem;
		border-bottom: 1px solid var(--color-border-subtle);
		vertical-align: middle;
	}
	.cmd-table tbody tr:last-child td { border-bottom: none; }
	.cmd-table tbody tr:hover td { background: var(--color-surface-hover); }
	.cmd-table code {
		font-family: 'JetBrains Mono', monospace;
		font-size: 0.8125rem;
		color: var(--color-text);
	}
	.alias { color: var(--color-text-muted) !important; }
	.td-desc { color: var(--color-text-muted); }

	/* ===== FOOTER ===== */
	.footer {
		border-top: 1px solid var(--color-border-subtle);
		padding: 1.5rem 2rem;
	}
	.footer-inner {
		max-width: 960px;
		margin: 0 auto;
		display: flex;
		align-items: center;
		gap: 1.5rem;
	}
	.footer-brand { font-weight: 700; font-size: 0.8125rem; }
	.footer-copy { color: var(--color-text-dim); font-size: 0.75rem; }
	.footer-link {
		color: var(--color-text-dim);
		font-size: 0.75rem;
		text-decoration: none;
		margin-left: auto;
		transition: color 0.15s;
	}
	.footer-link:hover { color: var(--color-text); }

	/* ===== RESPONSIVE ===== */
	@media (max-width: 768px) {
		.nav-links { display: none; }
		.nav-hamburger { display: flex; }

		.docs-layout {
			display: block;
		}
		.sidebar--desktop { display: none; }

		.content {
			padding: 2rem 1.25rem 4rem;
		}
		.section-title {
			font-size: 1.5rem;
		}
		.info-item {
			flex-direction: column;
			gap: 0.25rem;
		}
		.info-key { width: auto; }
	}

	@media (min-width: 769px) {
		.nav-hamburger { display: none; }
	}

	@media (min-width: 1200px) {
		.content {
			padding: 3rem 4rem 6rem;
		}
	}
</style>
