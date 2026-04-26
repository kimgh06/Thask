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
		const routePoints = buildShiftedRoutePoints(src, tgt, wps, offset);
		storeRoute(edge, routePoints);

		if (wps.length > 0) {
			const { distances, weights } = waypointsToSegments(src, tgt, wps);
			// Apply perpendicular offset to each distance
			const offsetDistances = distances.map(d => d + offset);
			edge.data({
				curveStyle: 'segments',
				segmentDistances: offsetDistances,
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
