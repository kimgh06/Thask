import type cytoscape from 'cytoscape';

export interface Point { x: number; y: number }

export type RouteAxis = 'vertical' | 'horizontal' | 'diagonal';

export interface RoutedSegment {
	edgeId: string;
	index: number;
	from: Point;
	to: Point;
	axis: RouteAxis;
}

export interface RoutedEdgePath {
	edgeId: string;
	points: Point[];
	segments: RoutedSegment[];
}

const ROUTE_SCRATCH_KEY = '_thaskRoute';
const NODE_AVOIDANCE_MARGIN = 18;
const MAX_AVOIDANCE_PASSES = 80;

interface Rect {
	x1: number;
	y1: number;
	x2: number;
	y2: number;
	nodeId: string;
}

/**
 * Compute 8-direction waypoints between src and tgt.
 * Returns 0, 1, or 2 waypoints ensuring all segments align to 8 directions.
 * Never places waypoints inside nodes (MIN_BEND = half node width).
 */
function compute8DirWaypoints(src: Point, tgt: Point): Point[] {
	const dx = tgt.x - src.x;
	const dy = tgt.y - src.y;
	const absDx = Math.abs(dx);
	const absDy = Math.abs(dy);

	const MIN_BEND = 36; // half node width

	// Nearly vertical: Z-path (vertical → horizontal → vertical)
	if (absDx < MIN_BEND && absDy >= MIN_BEND) {
		const midY = (src.y + tgt.y) / 2;
		return [
			{ x: src.x, y: midY },
			{ x: tgt.x, y: midY },
		];
	}

	// Nearly horizontal: Z-path (horizontal → vertical → horizontal)
	if (absDy < MIN_BEND && absDx >= MIN_BEND) {
		const midX = (src.x + tgt.x) / 2;
		return [
			{ x: midX, y: src.y },
			{ x: midX, y: tgt.y },
		];
	}

	// Nearly diagonal or very close: straight line
	if (Math.abs(absDx - absDy) < MIN_BEND) {
		return [];
	}

	// Normal 8-direction: single diagonal-first waypoint
	if (absDx >= absDy) {
		return [{ x: src.x + Math.sign(dx) * absDy, y: tgt.y }];
	}
	return [{ x: tgt.x, y: src.y + Math.sign(dy) * absDx }];
}

function classifyAxis(from: Point, to: Point): RouteAxis {
	const dx = Math.abs(to.x - from.x);
	const dy = Math.abs(to.y - from.y);
	const EPS = 0.75;

	if (dx <= EPS) return 'vertical';
	if (dy <= EPS) return 'horizontal';
	return 'diagonal';
}

function distance(a: Point, b: Point): number {
	const dx = b.x - a.x;
	const dy = b.y - a.y;
	return Math.sqrt(dx * dx + dy * dy);
}

function rectContainsPoint(rect: Rect, point: Point): boolean {
	return point.x >= rect.x1 && point.x <= rect.x2 && point.y >= rect.y1 && point.y <= rect.y2;
}

function segmentIntersectsRect(from: Point, to: Point, rect: Rect): boolean {
	if (rectContainsPoint(rect, from) || rectContainsPoint(rect, to)) return true;

	const dx = to.x - from.x;
	const dy = to.y - from.y;
	let t0 = 0;
	let t1 = 1;

	const clip = (p: number, q: number): boolean => {
		if (Math.abs(p) < 0.000001) return q >= 0;
		const r = q / p;
		if (p < 0) {
			if (r > t1) return false;
			if (r > t0) t0 = r;
		} else {
			if (r < t0) return false;
			if (r < t1) t1 = r;
		}
		return true;
	};

	return (
		clip(-dx, from.x - rect.x1) &&
		clip(dx, rect.x2 - from.x) &&
		clip(-dy, from.y - rect.y1) &&
		clip(dy, rect.y2 - from.y) &&
		t0 <= t1
	);
}

function pathLength(points: Point[]): number {
	let total = 0;
	for (let i = 0; i < points.length - 1; i += 1) total += distance(points[i], points[i + 1]);
	return total;
}

function countObstacleHits(points: Point[], obstacles: Rect[]): number {
	let hits = 0;
	for (let i = 0; i < points.length - 1; i += 1) {
		for (const obstacle of obstacles) {
			if (segmentIntersectsRect(points[i], points[i + 1], obstacle)) hits += 1;
		}
	}
	return hits;
}

function isDescendantOf(node: cytoscape.NodeSingular, maybeAncestorId: string): boolean {
	let parentId = node.data('parentId') as string | null | undefined;
	const seen = new Set<string>();
	while (parentId) {
		if (parentId === maybeAncestorId) return true;
		if (seen.has(parentId)) return false;
		seen.add(parentId);
		const parent = node.cy().getElementById(parentId);
		parentId = parent?.data('parentId') as string | null | undefined;
	}
	return false;
}

function shouldAvoidNode(
	node: cytoscape.NodeSingular,
	source: cytoscape.NodeSingular,
	target: cytoscape.NodeSingular,
): boolean {
	if (!node.data('nodeType')) return false;
	if (node.id() === source.id() || node.id() === target.id()) return false;

	const isGroup = node.data('nodeType') === 'GROUP';
	if (!isGroup) return true;

	if (node.hasClass('group-collapsed')) return true;

	// Expanded groups are visual containers. Avoiding their whole interior would
	// force huge detours and is the main source of false positives in diagnostics.
	if (isDescendantOf(source, node.id()) || isDescendantOf(target, node.id())) return false;
	return false;
}

function nodeObstacle(node: cytoscape.NodeSingular): Rect {
	const box = node.boundingBox({
		includeLabels: true,
		includeOverlays: true,
	});
	return {
		x1: box.x1 - NODE_AVOIDANCE_MARGIN,
		y1: box.y1 - NODE_AVOIDANCE_MARGIN,
		x2: box.x2 + NODE_AVOIDANCE_MARGIN,
		y2: box.y2 + NODE_AVOIDANCE_MARGIN,
		nodeId: node.id(),
	};
}

function buildObstacles(
	cy: cytoscape.Core,
	source: cytoscape.NodeSingular,
	target: cytoscape.NodeSingular,
): Rect[] {
	const obstacles: Rect[] = [];
	cy.nodes().forEach((node) => {
		if (shouldAvoidNode(node, source, target)) obstacles.push(nodeObstacle(node));
	});
	return obstacles;
}

function detourCandidates(from: Point, to: Point, rect: Rect): Point[][] {
	const lanes = [
		[
			{ x: from.x, y: rect.y1 },
			{ x: to.x, y: rect.y1 },
		],
		[
			{ x: from.x, y: rect.y2 },
			{ x: to.x, y: rect.y2 },
		],
		[
			{ x: rect.x1, y: from.y },
			{ x: rect.x1, y: to.y },
		],
		[
			{ x: rect.x2, y: from.y },
			{ x: rect.x2, y: to.y },
		],
	];

	const cornerLanes = [
		[
			{ x: rect.x1, y: from.y },
			{ x: rect.x1, y: rect.y1 },
			{ x: to.x, y: rect.y1 },
		],
		[
			{ x: rect.x2, y: from.y },
			{ x: rect.x2, y: rect.y1 },
			{ x: to.x, y: rect.y1 },
		],
		[
			{ x: rect.x1, y: from.y },
			{ x: rect.x1, y: rect.y2 },
			{ x: to.x, y: rect.y2 },
		],
		[
			{ x: rect.x2, y: from.y },
			{ x: rect.x2, y: rect.y2 },
			{ x: to.x, y: rect.y2 },
		],
	];

	return [...lanes, ...cornerLanes];
}

function chooseDetour(from: Point, to: Point, rect: Rect, obstacles: Rect[]): Point[] {
	let best: Point[] | null = null;
	let bestScore = Number.POSITIVE_INFINITY;

	for (const candidate of detourCandidates(from, to, rect)) {
		const points = [from, ...candidate, to];
		const sameRectHits = countObstacleHits(points, [rect]);
		const allHits = countObstacleHits(points, obstacles);
		const score = sameRectHits * 1_000_000 + allHits * 10_000 + pathLength(points);
		if (score < bestScore) {
			bestScore = score;
			best = candidate;
		}
	}

	return best ?? [];
}

function removeConsecutiveDuplicates(points: Point[]): Point[] {
	return points.filter((point, index) => {
		if (index === 0) return true;
		const prev = points[index - 1];
		return Math.abs(point.x - prev.x) > 0.5 || Math.abs(point.y - prev.y) > 0.5;
	});
}

function avoidNodeObstacles(points: Point[], obstacles: Rect[]): Point[] {
	let routed = removeConsecutiveDuplicates(points);

	for (let pass = 0; pass < MAX_AVOIDANCE_PASSES; pass += 1) {
		let changed = false;

		for (let i = 0; i < routed.length - 1 && !changed; i += 1) {
			const from = routed[i];
			const to = routed[i + 1];
			const obstacle = obstacles.find((rect) => segmentIntersectsRect(from, to, rect));
			if (!obstacle) continue;

			const detour = chooseDetour(from, to, obstacle, obstacles);
			if (detour.length === 0) continue;

			routed = removeConsecutiveDuplicates([
				...routed.slice(0, i + 1),
				...detour,
				...routed.slice(i + 1),
			]);
			changed = true;
		}

		if (!changed) break;
	}

	return routed;
}

function buildSegments(edgeId: string, points: Point[]): RoutedSegment[] {
	const segments: RoutedSegment[] = [];
	for (let i = 0; i < points.length - 1; i += 1) {
		const from = points[i];
		const to = points[i + 1];
		segments.push({
			edgeId,
			index: i,
			from,
			to,
			axis: classifyAxis(from, to),
		});
	}
	return segments;
}

function buildShiftedRoutePoints(src: Point, tgt: Point, waypoints: Point[], offset: number): Point[] {
	if (waypoints.length === 0) {
		if (Math.abs(offset) < 0.001) return [src, tgt];
		const dx = tgt.x - src.x;
		const dy = tgt.y - src.y;
		const len = Math.sqrt(dx * dx + dy * dy) || 1;
		const nx = -dy / len;
		const ny = dx / len;
		const mid = {
			x: (src.x + tgt.x) / 2 + nx * offset,
			y: (src.y + tgt.y) / 2 + ny * offset,
		};
		return [src, mid, tgt];
	}

	const dx = tgt.x - src.x;
	const dy = tgt.y - src.y;
	const len = Math.sqrt(dx * dx + dy * dy) || 1;
	const nx = -dy / len;
	const ny = dx / len;

	return [
		src,
		...waypoints.map((wp) => ({
			x: wp.x + nx * offset,
			y: wp.y + ny * offset,
		})),
		tgt,
	];
}

function storeRoute(edge: cytoscape.EdgeSingular, points: Point[]): void {
	edge.scratch(ROUTE_SCRATCH_KEY, {
		edgeId: edge.id(),
		points,
		segments: buildSegments(edge.id(), points),
	} satisfies RoutedEdgePath);
}

export function getStoredRoute(edge: cytoscape.EdgeSingular): RoutedEdgePath | null {
	const value = edge.scratch(ROUTE_SCRATCH_KEY) as RoutedEdgePath | undefined;
	if (!value || value.points.length < 2) return null;
	return value;
}

export function getStoredRoutes(cy: cytoscape.Core): RoutedEdgePath[] {
	const routes: RoutedEdgePath[] = [];
	cy.edges().forEach((edge) => {
		const route = getStoredRoute(edge);
		if (route) routes.push(route);
	});
	return routes;
}

/**
 * Convert absolute waypoints to Cytoscape segment-distances/weights.
 */
function waypointsToSegments(
	src: Point, tgt: Point, waypoints: Point[]
): { distances: number[]; weights: number[] } {
	const dx = tgt.x - src.x;
	const dy = tgt.y - src.y;
	const len = Math.sqrt(dx * dx + dy * dy) || 1;
	const nx = -dy / len;
	const ny = dx / len;

	const distances: number[] = [];
	const weights: number[] = [];

	for (const wp of waypoints) {
		const wx = wp.x - src.x;
		const wy = wp.y - src.y;
		const weight = (wx * dx + wy * dy) / (len * len);
		const dist = wx * nx + wy * ny;
		weights.push(Math.max(0.01, Math.min(0.99, isFinite(weight) ? weight : 0.5)));
		distances.push(isFinite(dist) ? dist : 0);
	}

	return { distances, weights };
}

/**
 * Recompute 8-direction edge routing for all edges based on current node positions.
 */
export function applyDynamicRouting(cy: cytoscape.Core): void {
	// Count parallel edges per node pair for offset calculation
	const pairCount = new Map<string, number>();
	const pairIndex = new Map<string, number>();
	cy.edges().forEach((edge) => {
		const srcId = edge.source().id();
		const tgtId = edge.target().id();
		const key = srcId < tgtId ? `${srcId}|${tgtId}` : `${tgtId}|${srcId}`;
		pairCount.set(key, (pairCount.get(key) || 0) + 1);
	});

	cy.edges().forEach((edge) => {
		const srcId = edge.source().id();
		const tgtId = edge.target().id();
		const key = srcId < tgtId ? `${srcId}|${tgtId}` : `${tgtId}|${srcId}`;
		const total = pairCount.get(key) || 1;
		const idx = pairIndex.get(key) || 0;
		pairIndex.set(key, idx + 1);

		const src = edge.source().position();
		const tgt = edge.target().position();
		const wps = compute8DirWaypoints(src, tgt);

		// Parallel edge offset: spread overlapping edges apart
		const offset = total > 1 ? (idx - (total - 1) / 2) * 12 : 0;
		const shiftedRoutePoints = buildShiftedRoutePoints(src, tgt, wps, offset);
		const obstacles = buildObstacles(cy, edge.source(), edge.target());
		const routePoints = avoidNodeObstacles(shiftedRoutePoints, obstacles);
		const routedWaypoints = routePoints.slice(1, -1);
		storeRoute(edge, routePoints);

		if (routedWaypoints.length > 0) {
			const { distances, weights } = waypointsToSegments(src, tgt, routedWaypoints);
			edge.data({
				curveStyle: 'segments',
				segmentDistances: distances,
				segmentWeights: weights,
			});
		} else {
			// Straight line with parallel offset
			edge.data({
				curveStyle: 'segments',
				segmentDistances: [offset || 0.1],
				segmentWeights: [0.5],
			});
		}
	});
}

/**
 * Attach dynamic 8-direction routing to a Cytoscape instance.
 * Returns a cleanup function.
 */
export function attachDynamicRouting(cy: cytoscape.Core): () => void {
	let rafId: number | null = null;

	function scheduleRouting() {
		if (rafId !== null) return;
		rafId = requestAnimationFrame(() => {
			rafId = null;
			applyDynamicRouting(cy);
		});
	}

	cy.on('position', 'node', scheduleRouting);
	cy.on('layoutstop', scheduleRouting);
	cy.on('add', 'edge', scheduleRouting);
	cy.on('data', 'node', scheduleRouting); // re-route when node data (size) changes

	// Initial application
	applyDynamicRouting(cy);

	return () => {
		cy.off('position', 'node', scheduleRouting);
		cy.off('layoutstop', scheduleRouting);
		cy.off('add', 'edge', scheduleRouting);
		cy.off('data', 'node', scheduleRouting);
		if (rafId !== null) cancelAnimationFrame(rafId);
	};
}
