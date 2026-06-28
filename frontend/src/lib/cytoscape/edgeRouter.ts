import type cytoscape from 'cytoscape';
import { routeGrid8 } from './gridRouter';

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
const DIRECTION_EPS = 0.75;
const ROUTING_GRID_SIZE = 24;
const OBSTACLE_BUCKET_SIZE = 240;
const LARGE_GRAPH_NODE_THRESHOLD = 50;
const NORMAL_GRID_MAX_EXPANSIONS = 30_000;
const LARGE_GRID_MAX_EXPANSIONS = 12_000;
const FAST_ROUTE_MIN_INTERVAL_MS = 32;
const FAST_ROUTE_FRAME_BUDGET_MS = 6; // per-frame A* budget during drag; leftovers ride next tick

interface Rect {
	x1: number;
	y1: number;
	x2: number;
	y2: number;
	nodeId: string;
}

interface ObstacleIndex {
	rects: Rect[];
	buckets: Map<string, Rect[]>;
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

	if (
		absDx < DIRECTION_EPS ||
		absDy < DIRECTION_EPS ||
		Math.abs(absDx - absDy) < DIRECTION_EPS
	) {
		return [];
	}

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

	// Normal 8-direction: single diagonal-first waypoint
	if (absDx >= absDy) {
		return [{ x: src.x + Math.sign(dx) * absDy, y: tgt.y }];
	}
	return [{ x: tgt.x, y: src.y + Math.sign(dy) * absDx }];
}

function classifyAxis(from: Point, to: Point): RouteAxis {
	const dx = Math.abs(to.x - from.x);
	const dy = Math.abs(to.y - from.y);

	if (dx <= DIRECTION_EPS) return 'vertical';
	if (dy <= DIRECTION_EPS) return 'horizontal';
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

function rectBucketKeys(rect: Pick<Rect, 'x1' | 'y1' | 'x2' | 'y2'>): string[] {
	const keys: string[] = [];
	const startX = Math.floor(rect.x1 / OBSTACLE_BUCKET_SIZE);
	const endX = Math.floor(rect.x2 / OBSTACLE_BUCKET_SIZE);
	const startY = Math.floor(rect.y1 / OBSTACLE_BUCKET_SIZE);
	const endY = Math.floor(rect.y2 / OBSTACLE_BUCKET_SIZE);

	for (let x = startX; x <= endX; x += 1) {
		for (let y = startY; y <= endY; y += 1) {
			keys.push(`${x}:${y}`);
		}
	}

	return keys;
}

function segmentRect(from: Point, to: Point): Pick<Rect, 'x1' | 'y1' | 'x2' | 'y2'> {
	return {
		x1: Math.min(from.x, to.x),
		y1: Math.min(from.y, to.y),
		x2: Math.max(from.x, to.x),
		y2: Math.max(from.y, to.y),
	};
}

function shouldUseGridRoute(
	points: Point[],
	obstacleIndex: ObstacleIndex,
	sourceId: string,
	targetId: string,
): boolean {
	if (obstacleIndex.rects.length === 0) return false;

	for (let i = 0; i < points.length - 1; i += 1) {
		const from = points[i];
		const to = points[i + 1];
		const seen = new Set<Rect>();
		for (const key of rectBucketKeys(segmentRect(from, to))) {
			const bucket = obstacleIndex.buckets.get(key);
			if (!bucket) continue;
			for (const obstacle of bucket) {
				if (seen.has(obstacle)) continue;
				seen.add(obstacle);
				if (obstacle.nodeId === sourceId || obstacle.nodeId === targetId) continue;
				if (segmentIntersectsRect(from, to, obstacle)) return true;
			}
		}
	}

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

function buildObstacleIndex(cy: cytoscape.Core): ObstacleIndex {
	const rects: Rect[] = [];
	const buckets = new Map<string, Rect[]>();
	cy.nodes().forEach((node) => {
		if (!node.data('nodeType')) return;
		const isGroup = node.data('nodeType') === 'GROUP';
		if (isGroup && !node.hasClass('group-collapsed')) return;
		const obstacle = nodeObstacle(node);
		rects.push(obstacle);
		for (const key of rectBucketKeys(obstacle)) {
			const bucket = buckets.get(key);
			if (bucket) bucket.push(obstacle);
			else buckets.set(key, [obstacle]);
		}
	});
	return { rects, buckets };
}

function buildObstacles(
	obstacleIndex: ObstacleIndex,
	sourceId: string,
	targetId: string,
): Rect[] {
	return obstacleIndex.rects.filter((rect) => rect.nodeId !== sourceId && rect.nodeId !== targetId);
}

function removeConsecutiveDuplicates(points: Point[]): Point[] {
	return points.filter((point, index) => {
		if (index === 0) return true;
		const prev = points[index - 1];
		return Math.abs(point.x - prev.x) > 0.5 || Math.abs(point.y - prev.y) > 0.5;
	});
}

function snapPolylineTo8Dir(points: Point[]): Point[] {
	const anchors = removeConsecutiveDuplicates(points);
	if (anchors.length < 2) return anchors;

	const snapped: Point[] = [anchors[0]];
	for (let i = 0; i < anchors.length - 1; i += 1) {
		const from = anchors[i];
		const to = anchors[i + 1];
		const waypoints = compute8DirWaypoints(from, to);
		snapped.push(...waypoints, to);
	}

	return removeConsecutiveDuplicates(snapped);
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

function nearlyEqual(a: number, b: number): boolean {
	return Math.abs(a - b) <= 0.001;
}

function sameNumberArray(left: unknown, right: number[]): boolean {
	if (!Array.isArray(left) || left.length !== right.length) return false;
	return left.every((value, index) => typeof value === 'number' && nearlyEqual(value, right[index]));
}

function samePoints(left: Point[], right: Point[]): boolean {
	if (left.length !== right.length) return false;
	return left.every((point, index) => nearlyEqual(point.x, right[index].x) && nearlyEqual(point.y, right[index].y));
}

function storeRouteIfChanged(edge: cytoscape.EdgeSingular, points: Point[]): void {
	const previous = getStoredRoute(edge);
	if (previous && samePoints(previous.points, points)) return;
	storeRoute(edge, points);
}

function applyEdgeDataIfChanged(
	edge: cytoscape.EdgeSingular,
	data: {
		curveStyle: string;
		segmentDistances: number[];
		segmentWeights: number[];
		sourceEndpoint: string;
		targetEndpoint: string;
	},
): void {
	if (
		edge.data('curveStyle') === data.curveStyle &&
		edge.data('sourceEndpoint') === data.sourceEndpoint &&
		edge.data('targetEndpoint') === data.targetEndpoint &&
		sameNumberArray(edge.data('segmentDistances'), data.segmentDistances) &&
		sameNumberArray(edge.data('segmentWeights'), data.segmentWeights)
	) {
		return;
	}
	edge.data(data);
}

interface Endpoint {
	point: Point;
	value: string;
}

function endpointOffset(node: cytoscape.NodeSingular, from: Point, to: Point): Endpoint {
	const dx = to.x - from.x;
	const dy = to.y - from.y;
	const sx = Math.sign(dx);
	const sy = Math.sign(dy);
	const halfW = node.outerWidth() / 2;
	const halfH = node.outerHeight() / 2;
	let offset: Point;

	if (Math.abs(dx) <= DIRECTION_EPS) {
		offset = { x: 0, y: sy * halfH };
	} else if (Math.abs(dy) <= DIRECTION_EPS) {
		offset = { x: sx * halfW, y: 0 };
	} else {
		// Diagonal approach: attach to the dominant-axis face (a perpendicular hit), never a
		// corner. Corner attachment on a rounded-rectangle node leaves the arrowhead floating
		// off the visible border and pointing askew.
		if (Math.abs(dx) >= Math.abs(dy)) offset = { x: sx * halfW, y: 0 };
		else offset = { x: 0, y: sy * halfH };
	}

	const center = node.position();
	return {
		point: { x: center.x + offset.x, y: center.y + offset.y },
		value: `${offset.x}px ${offset.y}px`,
	};
}

function routeEndpoints(edge: cytoscape.EdgeSingular, points: Point[]): {
	sourceEndpoint: Endpoint;
	targetEndpoint: Endpoint;
} {
	if (points.length < 2) {
		const sourcePosition = edge.source().position();
		const targetPosition = edge.target().position();
		return {
			sourceEndpoint: { point: sourcePosition, value: 'outside-to-node' },
			targetEndpoint: { point: targetPosition, value: 'outside-to-node' },
		};
	}

	return {
		sourceEndpoint: endpointOffset(edge.source(), points[0], points[1]),
		targetEndpoint: endpointOffset(edge.target(), points[points.length - 1], points[points.length - 2]),
	};
}

function pairKey(srcId: string, tgtId: string): string {
	return srcId < tgtId ? `${srcId}|${tgtId}` : `${tgtId}|${srcId}`;
}

export function getStoredRoute(edge: cytoscape.EdgeSingular): RoutedEdgePath | null {
	const value = edge.scratch(ROUTE_SCRATCH_KEY) as RoutedEdgePath | undefined;
	if (!value || value.points.length < 2) return null;
	return value;
}

export function getStoredRoutes(cy: cytoscape.Core): RoutedEdgePath[] {
	const routes: RoutedEdgePath[] = [];
	cy.edges().forEach((edge) => {
		if (edge.data('routePending')) return;
		const route = getStoredRoute(edge);
		if (route) routes.push(route);
	});
	return routes;
}

/**
 * Build routes from the edge's CURRENT rendered geometry (live model coords), not the stored
 * scratch route. The stored route only refreshes when routing runs (throttled + budgeted), so it
 * lags the visible edge during a drag — the bridge overlay reading it would show stale, "bobbing"
 * arcs. sourceEndpoint/segmentPoints/targetEndpoint track every frame as nodes move, so crossings
 * built from them stay glued to what's actually drawn.
 */
export function getLiveRoutes(cy: cytoscape.Core): RoutedEdgePath[] {
	const routes: RoutedEdgePath[] = [];
	cy.edges().forEach((edge) => {
		if (edge.data('routePending') || edge.hidden()) return;
		const src = edge.sourceEndpoint();
		const tgt = edge.targetEndpoint();
		if (!src || !tgt) return;
		const mids =
			edge.style('curve-style') === 'segments' &&
			typeof (edge as unknown as { segmentPoints?: () => Point[] }).segmentPoints === 'function'
				? (edge as unknown as { segmentPoints: () => Point[] }).segmentPoints()
				: [];
		const points: Point[] = [
			{ x: src.x, y: src.y },
			...mids.map((p) => ({ x: p.x, y: p.y })),
			{ x: tgt.x, y: tgt.y },
		];
		const deduped = points.filter(
			(p, i) => i === 0 || Math.abs(p.x - points[i - 1].x) > 0.01 || Math.abs(p.y - points[i - 1].y) > 0.01,
		);
		if (deduped.length < 2) return;
		routes.push({ edgeId: edge.id(), points: deduped, segments: buildSegments(edge.id(), deduped) });
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
		weights.push(isFinite(weight) ? weight : 0.5);
		distances.push(isFinite(dist) ? dist : 0);
	}

	return { distances, weights };
}

interface DynamicRoutingOptions {
	useGrid?: boolean;
	edgeIds?: Set<string>;
	obstacleIndex?: ObstacleIndex;
	maxExpansions?: number;
}

interface DynamicRoutingAttachmentOptions {
	routeImmediately?: boolean;
	isRoutingSuspended?: () => boolean;
}

/**
 * Recompute 8-direction edge routing for all edges based on current node positions.
 */
export function applyDynamicRouting(cy: cytoscape.Core, options: DynamicRoutingOptions = {}): void {
	const useGrid = options.useGrid ?? true;
	const obstacleIndex = useGrid ? (options.obstacleIndex ?? buildObstacleIndex(cy)) : null;
	const maxExpansions = useGrid
		? options.maxExpansions ?? (
				cy.nodes('[nodeType]').length >= LARGE_GRAPH_NODE_THRESHOLD
					? LARGE_GRID_MAX_EXPANSIONS
					: NORMAL_GRID_MAX_EXPANSIONS
			)
		: 0;
	const edgesToRoute: cytoscape.EdgeSingular[] = [];
	if (options.edgeIds) {
		options.edgeIds.forEach((id) => {
			cy.getElementById(id).forEach((edge) => {
				if (edge.isEdge()) edgesToRoute.push(edge);
			});
		});
	} else {
		cy.edges().forEach((edge) => {
			edgesToRoute.push(edge);
		});
	}
	if (edgesToRoute.length === 0) return;
		// Count parallel edges per node pair for offset calculation
		const pairCount = new Map<string, number>();
		const pairSeen = new Map<string, number>();
		const pairIndex = new Map<string, number>();
		const registerParallelEdge = (edge: cytoscape.EdgeSingular) => {
			const key = pairKey(edge.source().id(), edge.target().id());
			const idx = pairSeen.get(key) || 0;
			pairSeen.set(key, idx + 1);
			pairCount.set(key, idx + 1);
			pairIndex.set(edge.id(), idx);
		};

		if (options.edgeIds) {
			const visitedPairs = new Set<string>();
			edgesToRoute.forEach((edge) => {
				const source = edge.source();
				const key = pairKey(source.id(), edge.target().id());
				if (visitedPairs.has(key)) return;
				visitedPairs.add(key);
				source.connectedEdges().forEach((candidate) => {
					if (pairKey(candidate.source().id(), candidate.target().id()) === key) {
						registerParallelEdge(candidate);
					}
				});
			});
		} else {
			cy.edges().forEach(registerParallelEdge);
		}

	cy.batch(() => edgesToRoute.forEach((edge) => {
		const srcId = edge.source().id();
		const tgtId = edge.target().id();
			const key = pairKey(srcId, tgtId);
			const total = pairCount.get(key) || 1;
			const idx = pairIndex.get(edge.id()) || 0;

		const src = edge.source().position();
		const tgt = edge.target().position();
		const wps = compute8DirWaypoints(src, tgt);

		// Parallel edge offset: spread overlapping edges apart
		const offset = total > 1 ? (idx - (total - 1) / 2) * 12 : 0;
		const shiftedRoutePoints = snapPolylineTo8Dir(buildShiftedRoutePoints(src, tgt, wps, offset));
			const roughRoutePoints = shiftedRoutePoints;
			// First pass: endpoints from the rough straight route — used ONLY as A* start/goal anchors.
			const anchors = routeEndpoints(edge, roughRoutePoints);
			const routeWithGrid = useGrid && obstacleIndex
				? shouldUseGridRoute(roughRoutePoints, obstacleIndex, srcId, tgtId)
				: false;
			const obstacles = routeWithGrid && obstacleIndex ? buildObstacles(obstacleIndex, srcId, tgtId) : [];
		const gridRoutePoints = routeWithGrid
			? routeGrid8(anchors.sourceEndpoint.point, anchors.targetEndpoint.point, obstacles, {
					gridSize: ROUTING_GRID_SIZE,
					maxExpansions,
				})
			: null;
		// One routing logic: grid A* is the sole obstacle-avoidance path. If A* finds no
		// route (rare — only past maxExpansions), fall straight back to the 8-dir line.
		const innerRoutePoints = gridRoutePoints ?? roughRoutePoints;
		// Root fix: recompute attachment from the ACTUAL routed geometry (post-avoidance), so the
		// arrow meets the face the line truly enters — not the pre-avoidance straight guess. When
		// A* detours and arrives from a different side, the old endpoints (from rough) left the
		// arrowhead anchored to the wrong side / a corner. Direction is taken from the node CENTER
		// to the adjacent routed waypoint (not the corner anchor A* used), so the face matches the
		// side the route truly comes from.
		const sourceEndpoint = innerRoutePoints.length >= 2
			? endpointOffset(edge.source(), edge.source().position(), innerRoutePoints[1])
			: anchors.sourceEndpoint;
		const targetEndpoint = innerRoutePoints.length >= 2
			? endpointOffset(edge.target(), edge.target().position(), innerRoutePoints[innerRoutePoints.length - 2])
			: anchors.targetEndpoint;
		const visibleRoutePoints = snapPolylineTo8Dir([
			sourceEndpoint.point,
			...innerRoutePoints.slice(1, -1),
			targetEndpoint.point,
		]);
		const routedWaypoints = visibleRoutePoints.slice(1, -1);
		storeRouteIfChanged(edge, visibleRoutePoints);

		if (routedWaypoints.length > 0) {
			const { distances, weights } = waypointsToSegments(
				sourceEndpoint.point,
				targetEndpoint.point,
				routedWaypoints,
			);
			applyEdgeDataIfChanged(edge, {
				curveStyle: 'segments',
				segmentDistances: distances,
				segmentWeights: weights,
				sourceEndpoint: sourceEndpoint.value,
				targetEndpoint: targetEndpoint.value,
			});
		} else {
			// Straight line with parallel offset
			applyEdgeDataIfChanged(edge, {
				curveStyle: 'segments',
				segmentDistances: [offset || 0.1],
				segmentWeights: [0.5],
				sourceEndpoint: sourceEndpoint.value,
				targetEndpoint: targetEndpoint.value,
			});
		}
	}));
}

/**
 * Attach dynamic 8-direction routing to a Cytoscape instance.
 * Returns a cleanup function.
 */
export function attachDynamicRouting(cy: cytoscape.Core, options: DynamicRoutingAttachmentOptions = {}): () => void {
	let rafId: number | null = null;
	let chunkHandle: number | null = null;
	let chunkHandleType: 'idle' | 'timeout' | null = null;
	let settleTimeout: ReturnType<typeof setTimeout> | null = null;
	let fastRouteTimeout: ReturnType<typeof setTimeout> | null = null;
	let lastFastRouteAt = 0;
	let pendingFastEdgeIds = new Set<string>();
	let pendingFastNodeIds = new Set<string>();
	let pendingSettledEdgeIds = new Set<string>();
	let pendingSettledNodeIds = new Set<string>();
	const GRID_CHUNK_SIZE = 8;
	const SETTLE_ROUTE_UNIT = 2; // edges per deadline check during idle settle routing
	const isSuspended = () => options.isRoutingSuspended?.() ?? false;
	const requestIdle = (callback: () => void): { type: 'idle' | 'timeout'; id: number } => {
		const win = window as Window & {
			requestIdleCallback?: (cb: () => void, options?: { timeout: number }) => number;
		};
		if (win.requestIdleCallback) {
			return { type: 'idle', id: win.requestIdleCallback(callback, { timeout: 800 }) };
		}
		return { type: 'timeout', id: window.setTimeout(callback, 16) };
	};

	function cancelChunkRouting() {
		if (chunkHandle !== null) {
			const win = window as Window & { cancelIdleCallback?: (id: number) => void };
			if (chunkHandleType === 'idle' && win.cancelIdleCallback) {
				win.cancelIdleCallback(chunkHandle);
			} else {
				clearTimeout(chunkHandle);
			}
			chunkHandle = null;
			chunkHandleType = null;
		}
	}

	function addConnectedEdgeIdsForNodes(nodeIds: Set<string>, edgeIds: Set<string>) {
		nodeIds.forEach((nodeId) => {
			cy.getElementById(nodeId).forEach((node) => {
				if (!node.isNode()) return;
				node.connectedEdges().forEach((edge) => {
					edgeIds.add(edge.id());
				});
			});
		});
	}

	function collectPendingFastEdgeIds(): Set<string> {
		const edgeIds = new Set(pendingFastEdgeIds);
		addConnectedEdgeIdsForNodes(pendingFastNodeIds, edgeIds);
		pendingFastEdgeIds.clear();
		pendingFastNodeIds.clear();
		return edgeIds;
	}

	function collectPendingSettledEdgeIds(): Set<string> {
		const edgeIds = new Set(pendingSettledEdgeIds);
		addConnectedEdgeIdsForNodes(pendingSettledNodeIds, edgeIds);
		pendingSettledEdgeIds.clear();
		pendingSettledNodeIds.clear();
		return edgeIds;
	}

	function scheduleGridChunks(edgeIds?: Set<string>) {
		cancelChunkRouting();
		const ids = edgeIds ? [...edgeIds] : cy.edges().map((edge) => edge.id());
		const obstacleIndex = buildObstacleIndex(cy);
		const maxExpansions = cy.nodes('[nodeType]').length >= LARGE_GRAPH_NODE_THRESHOLD
			? LARGE_GRID_MAX_EXPANSIONS
			: NORMAL_GRID_MAX_EXPANSIONS;
		let index = 0;

		// Process edges while the idle callback still has frame budget, then yield.
		// Ignoring the deadline (plowing a fixed 8-edge chunk to completion) is what
		// caused the ~58ms hitch on settle — one callback ran all 8 edges' A* synchronously,
		// blocking the paint. Honouring timeRemaining() spreads the A* across idle slots.
		function step(deadline?: IdleDeadline) {
			if (isSuspended()) {
				chunkHandle = null;
				chunkHandleType = null;
				return;
			}
			const idle = !!deadline && typeof deadline.timeRemaining === 'function';
			// Small unit so we re-check the deadline often. A whole GRID_CHUNK_SIZE before the
			// first check would already overrun the frame — that was the bug.
			const unit = idle ? SETTLE_ROUTE_UNIT : GRID_CHUNK_SIZE;
			do {
				const chunk = new Set(ids.slice(index, index + unit));
				index += unit;
				if (chunk.size > 0) applyDynamicRouting(cy, { useGrid: true, edgeIds: chunk, obstacleIndex, maxExpansions });
			} while (index < ids.length && idle && deadline!.timeRemaining() > 2);
			if (index < ids.length) {
				const handle = requestIdle(step);
				chunkHandle = handle.id;
				chunkHandleType = handle.type;
			} else {
				chunkHandle = null;
				chunkHandleType = null;
			}
		}

		const handle = requestIdle(step);
		chunkHandle = handle.id;
		chunkHandleType = handle.type;
	}

	function scheduleRouting() {
		if (isSuspended()) return;
		const now = performance.now();
		const delay = Math.max(0, FAST_ROUTE_MIN_INTERVAL_MS - (now - lastFastRouteAt));
		if (delay > 0) {
			if (fastRouteTimeout === null) {
				fastRouteTimeout = setTimeout(() => {
					fastRouteTimeout = null;
					scheduleRouting();
				}, delay);
			}
			return;
		}
		if (rafId !== null) return;
		rafId = requestAnimationFrame(() => {
			rafId = null;
			if (isSuspended()) return;
			lastFastRouteAt = performance.now();
			const edgeIds = collectPendingFastEdgeIds();
			if (edgeIds.size === 0) return;
			// One drawing standard: grid A* obstacle avoidance, drag AND settle alike — edges
			// never change shape between the two states. A* is too heavy to run all moved edges
			// per frame (100ms+ spikes when crossing nodes), so we spend at most a frame budget
			// here and defer the rest to the next tick. Same maxExpansions as settle → identical
			// paths; settle (idle) just re-runs the same A* as the authoritative final pass.
			const ids = [...edgeIds];
			const obstacleIndex = buildObstacleIndex(cy);
			const maxExpansions = cy.nodes('[nodeType]').length >= LARGE_GRAPH_NODE_THRESHOLD
				? LARGE_GRID_MAX_EXPANSIONS
				: NORMAL_GRID_MAX_EXPANSIONS;
			const t0 = performance.now();
			let i = 0;
			while (i < ids.length) {
				applyDynamicRouting(cy, { useGrid: true, edgeIds: new Set([ids[i]]), obstacleIndex, maxExpansions });
				i++;
				if (performance.now() - t0 > FAST_ROUTE_FRAME_BUDGET_MS) break;
			}
			if (i < ids.length) {
				for (; i < ids.length; i++) pendingFastEdgeIds.add(ids[i]);
				scheduleRouting(); // finish leftovers next tick
			}
		});
	}

	function scheduleFastRouting(evt: cytoscape.EventObject) {
		if (isSuspended()) return;
		cancelChunkRouting();
		if (settleTimeout !== null) {
			clearTimeout(settleTimeout);
			settleTimeout = null;
		}
		const node = evt.target as cytoscape.NodeSingular;
		pendingFastNodeIds.add(node.id());
		pendingSettledNodeIds.add(node.id());
		scheduleRouting();
	}

	function scheduleDirtySettledRouting(evt?: cytoscape.EventObject) {
		if (isSuspended()) return;
		if (evt?.target?.isNode?.()) {
			const node = evt.target as cytoscape.NodeSingular;
			pendingSettledNodeIds.add(node.id());
		}
		if (settleTimeout !== null) clearTimeout(settleTimeout);
		settleTimeout = setTimeout(() => {
			settleTimeout = null;
			if (isSuspended()) return;
			const edgeIds = collectPendingSettledEdgeIds();
			if (edgeIds.size === 0) return;
			scheduleGridChunks(edgeIds);
		}, 80);
	}

	function scheduleFullSettledRouting() {
		if (isSuspended()) return;
		pendingFastEdgeIds.clear();
		pendingFastNodeIds.clear();
		pendingSettledEdgeIds.clear();
		pendingSettledNodeIds.clear();
		if (settleTimeout !== null) clearTimeout(settleTimeout);
		settleTimeout = setTimeout(() => {
			settleTimeout = null;
			if (isSuspended()) return;
			scheduleGridChunks();
		}, 500);
	}

	cy.on('position', 'node', scheduleFastRouting);
	cy.on('dragfree', 'node', scheduleDirtySettledRouting);
	cy.on('layoutstop', scheduleFullSettledRouting);
	cy.on('add', 'edge', scheduleFullSettledRouting);
	cy.on('data', 'node', scheduleFullSettledRouting); // re-route when node data (size) changes

	// Initial application: draw immediately when elements already exist, then refine shortest paths incrementally.
	if (options.routeImmediately ?? true) applyDynamicRouting(cy, { useGrid: true });
	scheduleFullSettledRouting();

	return () => {
		cy.off('position', 'node', scheduleFastRouting);
		cy.off('dragfree', 'node', scheduleDirtySettledRouting);
		cy.off('layoutstop', scheduleFullSettledRouting);
		cy.off('add', 'edge', scheduleFullSettledRouting);
		cy.off('data', 'node', scheduleFullSettledRouting);
		if (rafId !== null) cancelAnimationFrame(rafId);
		if (fastRouteTimeout !== null) clearTimeout(fastRouteTimeout);
		cancelChunkRouting();
		if (settleTimeout !== null) clearTimeout(settleTimeout);
	};
}
