<script lang="ts">

	const nodeTypes = [
		{ type: 'FLOW', color: '#6366f1', shape: 'Round Rect', desc: 'Product flows & user journeys' },
		{ type: 'BRANCH', color: '#8b5cf6', shape: 'Diamond', desc: 'Decision points & conditions' },
		{ type: 'TASK', color: '#3b82f6', shape: 'Rectangle', desc: 'Work items & action steps' },
		{ type: 'BUG', color: '#ef4444', shape: 'Hexagon', desc: 'Bugs & known issues' },
		{ type: 'API', color: '#22c55e', shape: 'Barrel', desc: 'API endpoints & integrations' },
		{ type: 'UI', color: '#f59e0b', shape: 'Ellipse', desc: 'UI components & screens' },
		{ type: 'GROUP', color: '#64748b', shape: 'Dashed Rect', desc: 'Collapsible containers' },
	];

	const statuses = [
		{ status: 'PASS', color: '#22c55e' },
		{ status: 'FAIL', color: '#ef4444' },
		{ status: 'IN_PROGRESS', color: '#6366f1' },
		{ status: 'BLOCKED', color: '#f59e0b' },
	];

	const edgeTypes = [
		{ type: 'depends_on', color: '#f97316', style: 'solid' },
		{ type: 'blocks', color: '#ef4444', style: 'dashed' },
		{ type: 'triggers', color: '#3b82f6', style: 'solid' },
		{ type: 'related', color: '#6b7280', style: 'solid' },
		{ type: 'parent_child', color: '#8b5cf6', style: 'dashed' },
	];

	const techStack = [
		{ name: 'Go 1.26', color: '#00ADD8' },
		{ name: 'SvelteKit', color: '#FF3E00' },
		{ name: 'Cytoscape.js', color: '#F7DF1E' },
		{ name: 'Tailwind v4', color: '#38BDF8' },
		{ name: 'PostgreSQL 17', color: '#4169E1' },
		{ name: 'Docker', color: '#2496ED' },
	];
</script>

<svelte:head>
	<title>Thask — Visualize product flows as a linked node graph</title>
	<meta
		name="description"
		content="Thask maps your product as a living graph. Visualize flows, tasks, and bugs with QA impact analysis. Self-hosted, open-source."
	/>
	<meta name="keywords" content="node graph, QA, impact analysis, task management, dependency graph, product flow, self-hosted, open source" />

	<!-- Open Graph -->
	<meta property="og:type" content="website" />
	<meta property="og:title" content="Thask — Visualize product flows as a linked node graph" />
	<meta property="og:description" content="Map your product as a living graph. 7 node types, 5 edge types, QA impact mode. Self-hosted with Docker Compose." />
	<meta property="og:image" content="/icon.svg" />
	<meta property="og:site_name" content="Thask" />

	<!-- Twitter Card -->
	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content="Thask — Visualize product flows as a linked node graph" />
	<meta name="twitter:description" content="Map your product as a living graph. 7 node types, 5 edge types, QA impact mode. Self-hosted with Docker Compose." />
	<meta name="twitter:image" content="/icon.svg" />

	<!-- Structured Data -->
	{@html `<script type="application/ld+json">${JSON.stringify({
		"@context": "https://schema.org",
		"@type": "SoftwareApplication",
		"name": "Thask",
		"applicationCategory": "DeveloperApplication",
		"operatingSystem": "Any",
		"description": "Visualize product flows, tasks, and bugs as a linked node graph. Built for QA risk management and change impact analysis.",
		"offers": { "@type": "Offer", "price": "0", "priceCurrency": "USD" },
		"license": "https://opensource.org/licenses/MIT",
		"url": "https://github.com/kimgh06/Thask",
		"codeRepository": "https://github.com/kimgh06/Thask"
	})}</script>`}
</svelte:head>

<div class="landing">
	<!-- ===== NAV ===== -->
	<nav class="nav">
		<div class="nav-inner">
			<a href="/" class="nav-logo">
				<img src="/icon.svg" alt="Thask" class="logo-icon-img" />
				<span class="logo-text">Thask</span>
			</a>
			<div class="nav-links">
				<a href="#features" class="nav-link">Features</a>
				<a href="#tech" class="nav-link">Tech Stack</a>
				<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener" class="nav-link">GitHub</a>
	<a href="/login" class="btn btn-primary btn-sm">Sign in</a>
			</div>
		</div>
	</nav>

	<!-- ===== HERO ===== -->
	<section class="hero">
		<div class="hero-grid-bg"></div>
		<div class="hero-inner">
			<div class="hero-text">
				<img src="/mascot.png" alt="Thask Mascot" class="hero-mascot" />
				<h1 class="hero-title">
					Thask it,
					<span class="hero-gradient">done.</span>
				</h1>
				<p class="hero-desc">
					Visualize product flows, tasks, and bugs as a <strong>linked node graph</strong>.<br />
					Built for QA risk management and change impact analysis.
				</p>
				<div class="hero-actions">
					<a href="/login" class="btn btn-primary btn-lg">Try Live Demo</a>
					<a
						href="https://github.com/kimgh06/Thask"
						target="_blank"
						rel="noopener"
						class="btn btn-outline btn-lg"
					>
						<svg class="icon" viewBox="0 0 24 24" fill="currentColor">
							<path
								d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"
							/>
						</svg>
						GitHub
					</a>
				</div>
			</div>
			<div class="hero-visual">
				<div class="screenshot-placeholder">
					<span>Screenshot</span>
					<span class="screenshot-sub">Graph Editor</span>
				</div>
			</div>
		</div>
	</section>

	<!-- ===== PAIN POINTS ===== -->
	<section class="section">
		<div class="container">
			<h2 class="section-title">Why Thask?</h2>
			<p class="section-subtitle">
				Existing tools fail to show the full picture of your product.
			</p>
			<div class="pain-grid">
				<div class="pain-card">
					<div class="pain-icon pain-icon-red">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<rect x="3" y="3" width="18" height="18" rx="2" />
							<path d="M3 9h18M9 3v18" />
						</svg>
					</div>
					<h3>Spreadsheets lose context</h3>
					<p>Rows and columns can't express relationships and dependencies between tasks.</p>
					<div class="pain-arrow">&darr;</div>
					<p class="pain-solution">Preserve context with a graph</p>
				</div>
				<div class="pain-card">
					<div class="pain-icon pain-icon-yellow">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M4 6h16M4 12h16M4 18h10" />
						</svg>
					</div>
					<h3>Issue trackers hide relationships</h3>
					<p>List views in Linear and Jira don't reveal dependency chains.</p>
					<div class="pain-arrow">&darr;</div>
					<p class="pain-solution">Visualize relationships with edges</p>
				</div>
				<div class="pain-card">
					<div class="pain-icon pain-icon-blue">
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10" />
							<path d="M12 6v6l4 2" />
						</svg>
					</div>
					<h3>Change impact is invisible</h3>
					<p>You fix one thing and something else breaks — discovered only after deploy.</p>
					<div class="pain-arrow">&darr;</div>
					<p class="pain-solution">Catch regressions with Impact Mode</p>
				</div>
			</div>
		</div>
	</section>

	<!-- ===== FEATURES ===== -->
	<section id="features" class="section section-alt">
		<div class="container">
			<h2 class="section-title">Core Features</h2>
			<p class="section-subtitle">Build graphs with drag and drop. Spot impact with a single click.</p>

			<!-- Feature 1: Graph Editor -->
			<div class="feature-row">
				<div class="feature-screenshot">
					<div class="screenshot-placeholder">
						<span>Screenshot</span>
						<span class="screenshot-sub">Interactive Graph Editor</span>
					</div>
				</div>
				<div class="feature-text">
					<div class="feature-badge">Graph Editor</div>
					<h3>Interactive Graph Editor</h3>
					<p>
						Place 7 node types and 5 edge types with drag and drop.
						The fCOSE physics engine automatically calculates the optimal layout.
					</p>
					<ul class="feature-list">
						<li>Hover a node to connect via edge handles</li>
						<li>fCOSE force-directed auto-layout</li>
						<li>Keyboard shortcuts for fast editing</li>
					</ul>
				</div>
			</div>

			<!-- Feature 2: Impact Mode -->
			<div class="feature-row feature-row-reverse">
				<div class="feature-screenshot">
					<div class="screenshot-placeholder">
						<span>Screenshot</span>
						<span class="screenshot-sub">QA Impact Mode</span>
					</div>
				</div>
				<div class="feature-text">
					<div class="feature-badge feature-badge-orange">Impact Mode</div>
					<h3>QA Impact Mode</h3>
					<p>
						Highlight changed nodes and their downstream dependencies in one click.
						A BFS algorithm automatically traverses the impact radius.
					</p>
					<ul class="feature-list">
						<li>Changed nodes — orange glow</li>
						<li>Affected nodes — cascading highlight</li>
						<li>Safe nodes — auto-dimmed</li>
					</ul>
				</div>
			</div>

			<!-- Feature 3: Node Detail -->
			<div class="feature-row">
				<div class="feature-screenshot">
					<div class="screenshot-placeholder">
						<span>Screenshot</span>
						<span class="screenshot-sub">Node Detail Panel</span>
					</div>
				</div>
				<div class="feature-text">
					<div class="feature-badge feature-badge-green">Detail Panel</div>
					<h3>Node Detail Panel</h3>
					<p>
						View and edit all node information in a slide-out panel.
						Every change is automatically recorded in the audit log.
					</p>
					<ul class="feature-list">
						<li>Inline editing — type, status, tags, description</li>
						<li>Connected nodes list with quick navigation</li>
						<li>Full change history audit log</li>
					</ul>
				</div>
			</div>
		</div>
	</section>

	<!-- ===== NODE TYPES ===== -->
	<section class="section">
		<div class="container">
			<h2 class="section-title">Nodes & Edges</h2>
			<p class="section-subtitle">7 node types, 4 statuses, and 5 edge types to express every relationship.</p>

			<div class="catalog-group">
				<h4 class="catalog-label">Node Types</h4>
				<div class="node-catalog">
					{#each nodeTypes as node}
						<div class="node-chip">
							<div class="node-dot" style="background-color: {node.color}"></div>
							<div class="node-chip-info">
								<span class="node-chip-type">{node.type}</span>
								<span class="node-chip-desc">{node.desc}</span>
							</div>
						</div>
					{/each}
				</div>
			</div>

			<div class="catalog-row">
				<div class="catalog-group">
					<h4 class="catalog-label">Statuses</h4>
					<div class="status-catalog">
						{#each statuses as s}
							<div class="status-chip">
								<div class="status-dot" style="background-color: {s.color}"></div>
								<span>{s.status.replace('_', ' ')}</span>
							</div>
						{/each}
					</div>
				</div>

				<div class="catalog-group">
					<h4 class="catalog-label">Edge Types</h4>
					<div class="edge-catalog">
						{#each edgeTypes as e}
							<div class="edge-chip">
								<svg width="32" height="8" viewBox="0 0 32 8">
									<line
										x1="0"
										y1="4"
										x2="24"
										y2="4"
										stroke={e.color}
										stroke-width="2"
										stroke-dasharray={e.style === 'dashed' ? '4,3' : 'none'}
									/>
									<polygon points="24,0 32,4 24,8" fill={e.color} />
								</svg>
								<span>{e.type}</span>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	</section>

	<!-- ===== TECH STACK ===== -->
	<section id="tech" class="section section-alt">
		<div class="container">
			<h2 class="section-title">Tech Stack</h2>
			<div class="tech-badges">
				{#each techStack as tech}
					<span class="tech-badge" style="border-color: {tech.color}; color: {tech.color}">
						{tech.name}
					</span>
				{/each}
			</div>
		</div>
	</section>

	<!-- ===== QUICK START ===== -->
	<section class="section">
		<div class="container">
			<h2 class="section-title">Get Started</h2>
			<p class="section-subtitle">Self-hosted. One command is all you need.</p>
			<div class="terminal">
				<div class="terminal-bar">
					<div class="terminal-dots">
						<span class="dot dot-red"></span>
						<span class="dot dot-yellow"></span>
						<span class="dot dot-green"></span>
					</div>
					<span class="terminal-title">terminal</span>
				</div>
				<div class="terminal-body">
					<p><span class="terminal-prompt">$</span> make up</p>
					<p class="terminal-output">Created .env with generated SESSION_SECRET</p>
					<p class="terminal-output terminal-success">&#10003; postgres &rarr; :7242</p>
					<p class="terminal-output terminal-success">&#10003; backend  &rarr; :7244</p>
					<p class="terminal-output terminal-success">&#10003; frontend &rarr; :7243</p>
					<p class="terminal-output">&nbsp;</p>
					<p class="terminal-output">
						Open <span class="terminal-link">http://localhost:7243</span>
					</p>
				</div>
			</div>
		</div>
	</section>

	<!-- ===== CTA ===== -->
	<section class="cta-section">
		<div class="container">
			<h2 class="cta-title">See it in action</h2>
			<p class="cta-desc">
				Map your product as a living graph and spot the blast radius of every change at a glance.
			</p>
			<div class="cta-actions">
				<a href="/login" class="btn btn-primary btn-lg">Try Live Demo</a>
				<a
					href="https://github.com/kimgh06/Thask"
					target="_blank"
					rel="noopener"
					class="btn btn-outline btn-lg"
				>
					View on GitHub
				</a>
			</div>
		</div>
	</section>

	<!-- ===== FOOTER ===== -->
	<footer class="footer">
		<div class="footer-inner">
			<div class="footer-brand">
				<img src="/icon.svg" alt="Thask" class="footer-icon" />
				<span>Thask</span>
			</div>
			<p class="footer-copy">MIT License &middot; Thask Contributors</p>
			<div class="footer-links">
				<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener">GitHub</a>
				<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener">Report Bug</a>
				<a href="https://github.com/kimgh06/Thask" target="_blank" rel="noopener">Request Feature</a>
			</div>
		</div>
	</footer>
</div>

<style>
	/* ===== RESET & BASE ===== */
	.landing {
		min-height: 100vh;
		overflow-x: hidden;
	}

	/* ===== NAV ===== */
	.nav {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 100;
		background: rgba(15, 23, 42, 0.85);
		backdrop-filter: blur(12px);
		border-bottom: 1px solid var(--color-border);
	}
	.nav-inner {
		max-width: 1200px;
		margin: 0 auto;
		padding: 0 1.5rem;
		height: 64px;
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.nav-logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		text-decoration: none;
		color: var(--color-text);
	}
	.logo-icon-img {
		width: 36px;
		height: 36px;
		border-radius: 10px;
	}
	.footer-icon {
		width: 28px;
		height: 28px;
		border-radius: 8px;
	}
	.logo-text {
		font-weight: 700;
		font-size: 1.25rem;
	}
	.nav-links {
		display: flex;
		align-items: center;
		gap: 1.5rem;
	}
	.nav-link {
		color: var(--color-text-muted);
		text-decoration: none;
		font-size: 0.875rem;
		font-weight: 500;
		transition: color 0.15s;
	}
	.nav-link:hover {
		color: var(--color-text);
	}

	/* ===== BUTTONS ===== */
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		border-radius: 10px;
		font-weight: 600;
		text-decoration: none;
		transition: all 0.15s;
		cursor: pointer;
		border: none;
		white-space: nowrap;
	}
	.btn-primary {
		background: var(--color-primary);
		color: white;
	}
	.btn-primary:hover {
		background: var(--color-primary-hover);
	}
	.btn-outline {
		background: transparent;
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}
	.btn-outline:hover {
		border-color: var(--color-text-muted);
		background: rgba(255, 255, 255, 0.03);
	}
	.btn-sm {
		padding: 0.4rem 1rem;
		font-size: 0.875rem;
	}
	.btn-lg {
		padding: 0.75rem 1.75rem;
		font-size: 1rem;
	}
	.icon {
		width: 18px;
		height: 18px;
	}

	/* ===== HERO ===== */
	.hero {
		position: relative;
		padding: 8rem 1.5rem 4rem;
		overflow: hidden;
	}
	.hero-grid-bg {
		position: absolute;
		inset: 0;
		background-image: radial-gradient(circle, #1e293b 1px, transparent 1px);
		background-size: 24px 24px;
		opacity: 0.7;
		mask-image: radial-gradient(ellipse 70% 60% at 50% 40%, black 30%, transparent 100%);
		-webkit-mask-image: radial-gradient(ellipse 70% 60% at 50% 40%, black 30%, transparent 100%);
	}
	.hero-inner {
		position: relative;
		max-width: 1200px;
		margin: 0 auto;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 3rem;
		align-items: center;
	}
	.hero-mascot {
		width: 200px;
		height: auto;
		margin-bottom: 1.5rem;
		filter: drop-shadow(0 0 20px rgba(99, 102, 241, 0.15));
	}
	.hero-title {
		font-size: 3.5rem;
		font-weight: 800;
		line-height: 1.1;
		margin-bottom: 1.25rem;
		letter-spacing: -0.025em;
	}
	.hero-gradient {
		background: linear-gradient(135deg, var(--color-primary), #a78bfa);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	.hero-desc {
		font-size: 1.125rem;
		line-height: 1.7;
		color: var(--color-text-muted);
		margin-bottom: 2rem;
	}
	.hero-desc strong {
		color: var(--color-text);
	}
	.hero-actions {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.hero-visual {
		display: flex;
		justify-content: center;
		align-items: center;
	}

	/* ===== SCREENSHOT PLACEHOLDER ===== */
	.screenshot-placeholder {
		width: 100%;
		max-width: 560px;
		aspect-ratio: 16 / 10;
		background: #ffffff;
		border-radius: 12px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.25rem;
		border: 1px solid rgba(255, 255, 255, 0.1);
		box-shadow: 0 0 60px rgba(99, 102, 241, 0.08);
	}
	.screenshot-placeholder span {
		color: #94a3b8;
		font-size: 0.875rem;
		font-weight: 500;
	}
	.screenshot-sub {
		font-size: 0.75rem !important;
		color: #64748b !important;
	}

	/* ===== SECTIONS ===== */
	.section {
		padding: 5rem 1.5rem;
	}
	.section-alt {
		background: rgba(30, 41, 59, 0.3);
	}
	.container {
		max-width: 1200px;
		margin: 0 auto;
	}
	.section-title {
		font-size: 2rem;
		font-weight: 800;
		text-align: center;
		margin-bottom: 0.75rem;
		letter-spacing: -0.02em;
	}
	.section-subtitle {
		text-align: center;
		color: var(--color-text-muted);
		font-size: 1.05rem;
		margin-bottom: 3rem;
		max-width: 600px;
		margin-left: auto;
		margin-right: auto;
	}

	/* ===== PAIN POINTS ===== */
	.pain-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1.5rem;
	}
	.pain-card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 12px;
		padding: 2rem 1.5rem;
		text-align: center;
	}
	.pain-card h3 {
		font-size: 1.05rem;
		font-weight: 700;
		margin-bottom: 0.5rem;
	}
	.pain-card p {
		font-size: 0.875rem;
		color: var(--color-text-muted);
		line-height: 1.6;
	}
	.pain-icon {
		width: 48px;
		height: 48px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		margin: 0 auto 1rem;
	}
	.pain-icon svg {
		width: 24px;
		height: 24px;
	}
	.pain-icon-red {
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}
	.pain-icon-yellow {
		background: rgba(245, 158, 11, 0.1);
		color: #f59e0b;
	}
	.pain-icon-blue {
		background: rgba(99, 102, 241, 0.1);
		color: #6366f1;
	}
	.pain-arrow {
		color: var(--color-primary);
		font-size: 1.25rem;
		margin: 0.75rem 0;
	}
	.pain-solution {
		font-weight: 600;
		color: var(--color-primary) !important;
		font-size: 0.9rem !important;
	}

	/* ===== FEATURE ROWS ===== */
	.feature-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 3rem;
		align-items: center;
		margin-bottom: 4rem;
	}
	.feature-row:last-child {
		margin-bottom: 0;
	}
	.feature-row-reverse {
		direction: rtl;
	}
	.feature-row-reverse > * {
		direction: ltr;
	}
	.feature-screenshot {
		display: flex;
		justify-content: center;
	}
	.feature-text h3 {
		font-size: 1.5rem;
		font-weight: 700;
		margin-bottom: 0.75rem;
	}
	.feature-text p {
		color: var(--color-text-muted);
		line-height: 1.7;
		margin-bottom: 1rem;
	}
	.feature-badge {
		display: inline-block;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--color-primary);
		background: rgba(99, 102, 241, 0.1);
		padding: 0.25rem 0.6rem;
		border-radius: 6px;
		margin-bottom: 0.75rem;
	}
	.feature-badge-orange {
		color: #fb923c;
		background: rgba(251, 146, 60, 0.1);
	}
	.feature-badge-green {
		color: #22c55e;
		background: rgba(34, 197, 94, 0.1);
	}
	.feature-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.feature-list li {
		position: relative;
		padding-left: 1.25rem;
		color: var(--color-text-muted);
		font-size: 0.9rem;
		line-height: 1.8;
	}
	.feature-list li::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0.65em;
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--color-primary);
	}

	/* ===== NODE CATALOG ===== */
	.catalog-label {
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--color-text-muted);
		margin-bottom: 1rem;
	}
	.catalog-group {
		margin-bottom: 2rem;
	}
	.catalog-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2rem;
	}
	.node-catalog {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 0.75rem;
	}
	.node-chip {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		padding: 0.75rem 1rem;
	}
	.node-dot {
		width: 12px;
		height: 12px;
		border-radius: 3px;
		flex-shrink: 0;
	}
	.node-chip-info {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
	}
	.node-chip-type {
		font-weight: 700;
		font-size: 0.8rem;
		letter-spacing: 0.03em;
	}
	.node-chip-desc {
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}
	.status-catalog {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
	.status-chip {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 0.5rem 0.875rem;
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.02em;
	}
	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}
	.edge-catalog {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
	.edge-chip {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		padding: 0.5rem 0.875rem;
		font-size: 0.8rem;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	/* ===== TECH BADGES ===== */
	.tech-badges {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.75rem;
	}
	.tech-badge {
		border: 1px solid;
		border-radius: 999px;
		padding: 0.5rem 1.25rem;
		font-size: 0.875rem;
		font-weight: 600;
	}

	/* ===== TERMINAL ===== */
	.terminal {
		max-width: 560px;
		margin: 0 auto;
		border-radius: 12px;
		overflow: hidden;
		border: 1px solid var(--color-border);
		background: #0c1222;
	}
	.terminal-bar {
		background: var(--color-surface);
		padding: 0.75rem 1rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		border-bottom: 1px solid var(--color-border);
	}
	.terminal-dots {
		display: flex;
		gap: 6px;
	}
	.dot {
		width: 12px;
		height: 12px;
		border-radius: 50%;
	}
	.dot-red {
		background: #ef4444;
	}
	.dot-yellow {
		background: #f59e0b;
	}
	.dot-green {
		background: #22c55e;
	}
	.terminal-title {
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}
	.terminal-body {
		padding: 1.25rem 1.5rem;
		font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
		font-size: 0.875rem;
		line-height: 1.8;
	}
	.terminal-prompt {
		color: var(--color-success);
		margin-right: 0.5rem;
	}
	.terminal-output {
		color: var(--color-text-muted);
		padding-left: 1.25rem;
	}
	.terminal-success {
		color: var(--color-success);
	}
	.terminal-link {
		color: var(--color-primary);
		text-decoration: underline;
	}

	/* ===== CTA ===== */
	.cta-section {
		padding: 5rem 1.5rem;
		text-align: center;
		background: linear-gradient(
			180deg,
			transparent 0%,
			rgba(99, 102, 241, 0.04) 50%,
			transparent 100%
		);
	}
	.cta-title {
		font-size: 2rem;
		font-weight: 800;
		margin-bottom: 0.75rem;
	}
	.cta-desc {
		color: var(--color-text-muted);
		font-size: 1.05rem;
		margin-bottom: 2rem;
		max-width: 500px;
		margin-left: auto;
		margin-right: auto;
	}
	.cta-actions {
		display: flex;
		justify-content: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	/* ===== FOOTER ===== */
	.footer {
		border-top: 1px solid var(--color-border);
		padding: 2rem 1.5rem;
	}
	.footer-inner {
		max-width: 1200px;
		margin: 0 auto;
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: 1rem;
	}
	.footer-brand {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-weight: 700;
		font-size: 0.9rem;
	}
	.footer-copy {
		color: var(--color-text-muted);
		font-size: 0.8rem;
	}
	.footer-links {
		display: flex;
		gap: 1.25rem;
	}
	.footer-links a {
		color: var(--color-text-muted);
		text-decoration: none;
		font-size: 0.8rem;
		transition: color 0.15s;
	}
	.footer-links a:hover {
		color: var(--color-text);
	}

	/* ===== RESPONSIVE ===== */
	@media (max-width: 768px) {
		.hero-inner {
			grid-template-columns: 1fr;
			text-align: center;
		}
		.hero-title {
			font-size: 2.5rem;
		}
		.hero-actions {
			justify-content: center;
		}
		.pain-grid {
			grid-template-columns: 1fr;
		}
		.feature-row,
		.feature-row-reverse {
			grid-template-columns: 1fr;
			direction: ltr;
		}
		.catalog-row {
			grid-template-columns: 1fr;
		}
		.nav-links {
			gap: 0.75rem;
		}
		.nav-link {
			display: none;
		}
	}
</style>