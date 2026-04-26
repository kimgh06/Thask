import http from 'node:http';
import { URL } from 'node:url';
import { chromium } from 'playwright-core';

const PORT = Number(process.env.PORT || 7241);
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://frontend:7243';
const BROWSER_WS_ENDPOINT = process.env.BROWSER_WS_ENDPOINT || 'ws://browserless:3000';
const INTERNAL_SECRET = process.env.CAPTURE_INTERNAL_SECRET || '';
const MAX_BODY_BYTES = Number(process.env.MAX_BODY_BYTES || 8 * 1024 * 1024);
const MAX_WIDTH = 4096;
const MAX_HEIGHT = 4096;
const MIN_WIDTH = 320;
const MIN_HEIGHT = 240;
const MAX_SCALE = 4;
const MAX_RENDER_ZOOM = 4;

let browserPromise;

function clampNumber(value, fallback, min, max) {
	const n = Number(value);
	if (!Number.isFinite(n)) return fallback;
	return Math.max(min, Math.min(max, Math.floor(n)));
}

function clampFloat(value, fallback, min, max) {
	const n = Number(value);
	if (!Number.isFinite(n)) return fallback;
	return Math.max(min, Math.min(max, n));
}

function json(res, status, body) {
	const data = Buffer.from(JSON.stringify(body));
	res.writeHead(status, {
		'Content-Type': 'application/json',
		'Content-Length': data.length,
	});
	res.end(data);
}

function readBody(req) {
	return new Promise((resolve, reject) => {
		let size = 0;
		const chunks = [];
		req.on('data', (chunk) => {
			size += chunk.length;
			if (size > MAX_BODY_BYTES) {
				reject(new Error('Request body too large'));
				req.destroy();
				return;
			}
			chunks.push(chunk);
		});
		req.on('end', () => {
			try {
				resolve(JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}'));
			} catch {
				reject(new Error('Invalid JSON body'));
			}
		});
		req.on('error', reject);
	});
}

async function browser() {
	if (!browserPromise) {
		browserPromise = chromium.connectOverCDP(BROWSER_WS_ENDPOINT);
	}
	return browserPromise;
}

function assertSecret(req) {
	if (!INTERNAL_SECRET) return true;
	return req.headers['x-thask-capture-secret'] === INTERNAL_SECRET;
}

function validatePayload(payload) {
	if (!Array.isArray(payload.nodes)) throw new Error('nodes must be an array');
	if (!Array.isArray(payload.edges)) throw new Error('edges must be an array');
	const crop = payload.crop !== false && payload.crop !== 'false' && payload.crop !== 0 && payload.crop !== '0';
	return {
		nodes: payload.nodes,
		edges: payload.edges,
		width: clampNumber(payload.width, 1600, MIN_WIDTH, MAX_WIDTH),
		height: clampNumber(payload.height, 1000, MIN_HEIGHT, MAX_HEIGHT),
		padding: clampNumber(payload.padding, 80, 0, 400),
		scale: clampNumber(payload.scale, 2, 1, MAX_SCALE),
		zoom: clampFloat(payload.zoom, 1, 0.05, MAX_RENDER_ZOOM),
		crop,
	};
}

function cropClip(metrics, viewport, padding) {
	const graph = metrics?.graph;
	if (!graph || graph.width <= 0 || graph.height <= 0) return undefined;
	const pad = Math.min(Math.max(padding, 24), 160);
	const x = Math.max(0, Math.floor(graph.x1 - pad));
	const y = Math.max(0, Math.floor(graph.y1 - pad));
	const right = Math.min(viewport.width, Math.ceil(graph.x2 + pad));
	const bottom = Math.min(viewport.height, Math.ceil(graph.y2 + pad));
	const width = Math.max(1, right - x);
	const height = Math.max(1, bottom - y);
	return { x, y, width, height };
}

function autoViewport(bounds, input) {
	if (!bounds || bounds.width <= 0 || bounds.height <= 0) {
		return { width: input.width, height: input.height, zoom: input.zoom };
	}
	const pad = input.padding;
	const maxContentW = Math.max(1, MAX_WIDTH - pad * 2);
	const maxContentH = Math.max(1, MAX_HEIGHT - pad * 2);
	const zoom = Math.min(input.zoom, maxContentW / bounds.width, maxContentH / bounds.height);
	return {
		width: clampNumber(Math.ceil(bounds.width * zoom + pad * 2), input.width, MIN_WIDTH, MAX_WIDTH),
		height: clampNumber(Math.ceil(bounds.height * zoom + pad * 2), input.height, MIN_HEIGHT, MAX_HEIGHT),
		zoom,
	};
}

function allowedHost(url) {
	const target = new URL(url);
	const frontend = new URL(FRONTEND_URL);
	if (target.origin === frontend.origin) return true;
	if (target.protocol === 'data:' || target.protocol === 'blob:') return true;
	return false;
}

async function captureGraph(payload) {
	const input = validatePayload(payload);
	const b = await browser();
	const context = await b.newContext({
		viewport: { width: input.width, height: input.height },
		deviceScaleFactor: input.scale,
		colorScheme: 'dark',
	});
	const page = await context.newPage();
	page.setDefaultTimeout(30_000);
	page.on('console', (msg) => {
		if (['error', 'warning'].includes(msg.type())) {
			console.error(`[capture:${msg.type()}] ${msg.text()}`);
		}
	});
	page.on('pageerror', (error) => {
		console.error('[capture:pageerror]', error);
	});
	page.on('requestfailed', (request) => {
		const failure = request.failure();
		console.error(`[capture:requestfailed] ${request.url()} ${failure?.errorText ?? ''}`);
	});
	page.on('response', (response) => {
		if (response.status() >= 400) {
			console.error(`[capture:response] ${response.status()} ${response.url()}`);
		}
	});
	await page.route('**/*', (route) => {
		const url = route.request().url();
		if (allowedHost(url)) {
			route.continue();
			return;
		}
		route.abort();
	});

	try {
		await page.goto(new URL('/capture', FRONTEND_URL).toString(), { waitUntil: 'networkidle' });
		await page.waitForFunction(() => Boolean(window.__thaskCapture?.ready));
		await page.evaluate(async (graph) => {
			return window.__thaskCapture.load(graph);
		}, input);
		let metrics;
		if (input.crop) {
			const bounds = await page.evaluate(() => window.__thaskCapture.modelBounds());
			const viewport = autoViewport(bounds, input);
			await page.setViewportSize({ width: viewport.width, height: viewport.height });
			metrics = await page.evaluate(async ({ zoom, padding }) => {
				return window.__thaskCapture.renderAtZoom(zoom, padding);
			}, { zoom: viewport.zoom, padding: input.padding });
			metrics.model = bounds;
			metrics.renderZoom = viewport.zoom;
		} else {
			metrics = await page.evaluate(async (graph) => {
				return window.__thaskCapture.fit(graph.padding);
			}, input);
		}
		const clip = input.crop ? cropClip(metrics, metrics.viewport, input.padding) : undefined;
		if (clip) {
			metrics.crop = clip;
		}
		const image = await page.screenshot({ type: 'png', fullPage: false, animations: 'disabled', clip });
		return { image, metrics };
	} finally {
		await context.close();
	}
}

async function handle(req, res) {
	const url = new URL(req.url || '/', `http://${req.headers.host || 'localhost'}`);
	if (req.method === 'GET' && url.pathname === '/health') {
		json(res, 200, { status: 'ok' });
		return;
	}
	if (req.method !== 'POST' || url.pathname !== '/capture/graph') {
		json(res, 404, { error: 'Not found' });
		return;
	}
	if (!assertSecret(req)) {
		json(res, 403, { error: 'Forbidden' });
		return;
	}

	try {
		const payload = await readBody(req);
		const { image, metrics } = await captureGraph(payload);
		res.writeHead(200, {
			'Content-Type': 'image/png',
			'Content-Length': image.length,
			'Cache-Control': 'no-store',
			'X-Thask-Capture-Metrics': Buffer.from(JSON.stringify(metrics)).toString('base64url'),
		});
		res.end(image);
	} catch (error) {
		console.error('[capture] failed', error);
		json(res, 400, { error: error instanceof Error ? error.message : 'Capture failed' });
	}
}

const server = http.createServer((req, res) => {
	handle(req, res).catch((error) => {
		json(res, 500, { error: error instanceof Error ? error.message : 'Internal error' });
	});
});

server.listen(PORT, () => {
	console.log(`thask-capture listening on :${PORT}`);
});

process.on('SIGTERM', async () => {
	server.close();
	if (browserPromise) {
		const b = await browserPromise;
		await b.close();
	}
	process.exit(0);
});
