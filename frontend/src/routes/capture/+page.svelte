<script lang="ts">
	import { onMount, tick } from 'svelte';
	import CytoscapeCanvas from '$lib/components/CytoscapeCanvas.svelte';
	import type { GraphEdge, GraphNode } from '$lib/types';

	type CapturePayload = {
		nodes: GraphNode[];
		edges: GraphEdge[];
		padding?: number;
	};

	type CaptureMetrics = {
		nodeCount: number;
		edgeCount: number;
		zoom: number;
		fitsViewport: boolean;
		viewport: { width: number; height: number };
		graph: { x1: number; y1: number; x2: number; y2: number; width: number; height: number };
	};

	let nodes = $state<GraphNode[]>([]);
	let edges = $state<GraphEdge[]>([]);
	let loaded = $state(false);
	let canvas = $state<ReturnType<typeof CytoscapeCanvas> | undefined>(undefined);

	function wait(ms: number) {
		return new Promise((resolve) => setTimeout(resolve, ms));
	}

	async function waitForCy() {
		for (let i = 0; i < 100; i++) {
			const cy = canvas?.getCy();
			if (cy) return cy;
			await wait(25);
		}
		throw new Error('Cytoscape did not initialize');
	}

	async function loadGraph(payload: CapturePayload) {
		nodes = payload.nodes ?? [];
		edges = payload.edges ?? [];
		loaded = true;
		await tick();
		await fit(payload.padding);
		return metrics();
	}

	async function fit(padding = 80) {
		const cy = await waitForCy();
		cy.minZoom(0.01);
		cy.maxZoom(4);
		cy.resize();
		await wait(50);
		if (cy.elements().length > 0) {
			cy.fit(cy.elements(), padding);
		}
		await wait(150);
		cy.resize();
		if (cy.elements().length > 0) {
			cy.fit(cy.elements(), padding);
		}
		await wait(100);
		return metrics();
	}

	function metrics(): CaptureMetrics {
		const cy = canvas?.getCy();
		if (!cy) {
			return {
				nodeCount: 0,
				edgeCount: 0,
				zoom: 1,
				fitsViewport: false,
				viewport: { width: 0, height: 0 },
				graph: { x1: 0, y1: 0, x2: 0, y2: 0, width: 0, height: 0 },
			};
		}
		const bb = cy.elements().length > 0
			? cy.elements().renderedBoundingBox({ includeLabels: true, includeOverlays: true })
			: { x1: 0, y1: 0, x2: 0, y2: 0, w: 0, h: 0 };
		const viewport = { width: cy.width(), height: cy.height() };
		const fitsViewport = bb.x1 >= -1 && bb.y1 >= -1 && bb.x2 <= viewport.width + 1 && bb.y2 <= viewport.height + 1;
		return {
			nodeCount: cy.nodes('[nodeType]').length,
			edgeCount: cy.edges().length,
			zoom: cy.zoom(),
			fitsViewport,
			viewport,
			graph: { x1: bb.x1, y1: bb.y1, x2: bb.x2, y2: bb.y2, width: bb.w, height: bb.h },
		};
	}

	function modelBounds() {
		const cy = canvas?.getCy();
		if (!cy || cy.elements().length === 0) {
			return { x1: 0, y1: 0, x2: 0, y2: 0, width: 0, height: 0 };
		}
		const bb = cy.elements().boundingBox({ includeLabels: true, includeOverlays: true });
		return { x1: bb.x1, y1: bb.y1, x2: bb.x2, y2: bb.y2, width: bb.w, height: bb.h };
	}

	async function renderAtZoom(zoom = 1, padding = 80) {
		const cy = await waitForCy();
		cy.minZoom(0.01);
		cy.maxZoom(4);
		cy.resize();
		await wait(50);
		const bb = modelBounds();
		if (bb.width > 0 && bb.height > 0) {
			cy.zoom(zoom);
			cy.pan({
				x: Math.round(padding - bb.x1 * zoom),
				y: Math.round(padding - bb.y1 * zoom),
			});
		}
		await wait(150);
		return metrics();
	}

	onMount(() => {
		document.documentElement.dataset.theme = 'dark';
		const api = {
			ready: true,
			load: loadGraph,
			fit,
			modelBounds,
			renderAtZoom,
			metrics,
		};
		(window as unknown as { __thaskCapture: typeof api }).__thaskCapture = api;
	});
</script>

<svelte:head>
	<title>Thask Capture</title>
	<style>
		html, body {
			margin: 0;
			width: 100%;
			height: 100%;
			overflow: hidden;
			background: #131214;
		}
	</style>
</svelte:head>

<div class="h-screen w-screen overflow-hidden cytoscape-canvas-bg" data-capture-root>
	<CytoscapeCanvas
		bind:this={canvas}
		{nodes}
		{edges}
		projectId=""
		readonly={false}
		onUpdateNodeParent={() => {}}
		onCreateEdge={() => {}}
	/>
	{#if !loaded}
		<div class="fixed inset-0 flex items-center justify-center text-xs" style="color: var(--color-text-muted); background: var(--color-bg);">
			Ready
		</div>
	{:else}
		<div class="thask-watermark">
			<img src="/icon.svg" alt="" />
			<span>Thask</span>
		</div>
	{/if}
</div>

<style>
	.thask-watermark {
		position: fixed;
		right: 20px;
		bottom: 16px;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 12px 6px 8px;
		background: rgba(19, 18, 20, 0.72);
		backdrop-filter: blur(6px);
		border: 1px solid rgba(201, 168, 76, 0.18);
		border-radius: 8px;
		pointer-events: none;
	}
	.thask-watermark img {
		width: 22px;
		height: 22px;
		border-radius: 4px;
	}
	.thask-watermark span {
		font-family: 'JetBrains Mono', ui-monospace, monospace;
		font-size: 13px;
		font-weight: 700;
		letter-spacing: -0.01em;
		color: #f0e7d6;
	}
</style>
