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
const MAX_AVOIDANCE_PASSES = 80;
const DIRECTION_EPS = 0.75;
const ROUTING_GRID_SIZE = 24;

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

function buildObstacleIndex(cy: cytoscape.Core): Rect[] {
	const obstacles: Rect[] = [];
	cy.nodes().forEach((node) => {
		if (!node.data('nodeType')) return;
		const isGroup = node.data('nodeType') === 'GROUP';
		if (isGroup && !node.hasClass('group-collapsed')) return;
		obstacles.push(nodeObstacle(node));
	});
	return obstacles;
}

function buildObstacles(
	obstacleIndex: Rect[],
	source: cytoscape.NodeSingular,
	target: cytoscape.NodeSingular,
): Rect[] {
	return obstacleIndex.filter((rect) => rect.nodeId !== source.id() && rect.nodeId !== target.id());
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

function isDiagonalSegment(from: Point, to: Point): boolean {
	const dx = Math.abs(to.x - from.x);
	const dy = Math.abs(to.y - from.y);
	return dx > DIRECTION_EPS && dy > DIRECTION_EPS && Math.abs(dx - dy) <= DIRECTION_EPS;
}

function smoothDiagonalDoglegs(points: Point[]): Point[] {
	let routed = removeConsecutiveDuplicates(points);

	for (let pass = 0; pass < 4; pass += 1) {
		let changed = false;
		const next: Point[] = [routed[0]];

		for (let i = 1; i < routed.length - 2; i += 1) {
			const prev = routed[i - 1];
			const bend = routed[i];
			const lane = routed[i + 1];
			const after = routed[i + 2];

			if (!isDiagonalSegment(prev, bend)) {
				next.push(bend);
				continue;
			}

			const horizontalDogleg =
				Math.abs(bend.y - lane.y) <= DIRECTION_EPS &&
				Math.abs(lane.x - after.x) <= DIRECTION_EPS;
			if (horizontalDogleg) {
				const signY = Math.sign(bend.y - prev.y) || 1;
				const moved = {
					x: lane.x,
					y: prev.y + signY * Math.abs(lane.x - prev.x),
				};
				next.push(moved);
				changed = true;
				continue;
			}

			const verticalDogleg =
				Math.abs(bend.x - lane.x) <= DIRECTION_EPS &&
				Math.abs(lane.y - after.y) <= DIRECTION_EPS;
			if (verticalDogleg) {
				const signX = Math.sign(bend.x - prev.x) || 1;
				const moved = {
					x: prev.x + signX * Math.abs(lane.y - prev.y),
					y: lane.y,
				};
				next.push(moved);
				changed = true;
				continue;
			}

			next.push(bend);
		}

		next.push(...routed.slice(Math.max(1, routed.length - 2)));
		routed = snapPolylineTo8Dir(next);
		if (!changed) break;
	}

	return routed;
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
		const d = Math.min(halfW, halfH);
		offset = { x: sx * d, y: sy * d };
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
		weights.push(isFinite(weight) ? weight : 0.5);
		distances.push(isFinite(dist) ? dist : 0);
	}

	return { distances, weights };
}

interface DynamicRoutingOptions {
	useGrid?: boolean;
	edgeIds?: Set<string>;
}

/**
 * Recompute 8-direction edge routing for all edges based on current node positions.
 */
export function applyDynamicRouting(cy: cytoscape.Core, options: DynamicRoutingOptions = {}): void {
	const useGrid = options.useGrid ?? true;
	const obstacleIndex = useGrid ? buildObstacleIndex(cy) : [];

	// Count parallel edges per node pair for offset calculation
	const pairCount = new Map<string, number>();
	const pairSeen = new Map<string, number>();
	const pairIndex = new Map<string, number>();
	cy.edges().forEach((edge) => {
		const srcId = edge.source().id();
		const tgtId = edge.target().id();
		const key = srcId < tgtId ? `${srcId}|${tgtId}` : `${tgtId}|${srcId}`;
		const idx = pairSeen.get(key) || 0;
		pairSeen.set(key, idx + 1);
		pairCount.set(key, idx + 1);
		pairIndex.set(edge.id(), idx);
	});

	const edgesToRoute = options.edgeIds
		? cy.edges().filter((edge) => options.edgeIds?.has(edge.id()) ?? false)
		: cy.edges();

	cy.batch(() => edgesToRoute.forEach((edge) => {
		const srcId = edge.source().id();
		const tgtId = edge.target().id();
		const key = srcId < tgtId ? `${srcId}|${tgtId}` : `${tgtId}|${srcId}`;
		const total = pairCount.get(key) || 1;
		const idx = pairIndex.get(edge.id()) || 0;

		const src = edge.source().position();
		const tgt = edge.target().position();
		const wps = compute8DirWaypoints(src, tgt);

		// Parallel edge offset: spread overlapping edges apart
		const offset = total > 1 ? (idx - (total - 1) / 2) * 12 : 0;
		const shiftedRoutePoints = snapPolylineTo8Dir(buildShiftedRoutePoints(src, tgt, wps, offset));
		const obstacles = useGrid ? buildObstacles(obstacleIndex, edge.source(), edge.target()) : [];
		const roughRoutePoints = shiftedRoutePoints;
		const { sourceEndpoint, targetEndpoint } = routeEndpoints(edge, roughRoutePoints);
		const gridRoutePoints = useGrid
			? routeGrid8(sourceEndpoint.point, targetEndpoint.point, obstacles, {
					gridSize: ROUTING_GRID_SIZE,
				})
			: null;
		const fallbackRoutePoints = gridRoutePoints
			? null
			: useGrid
				? smoothDiagonalDoglegs(snapPolylineTo8Dir(avoidNodeObstacles(shiftedRoutePoints, obstacles)))
				: roughRoutePoints;
		const innerRoutePoints = gridRoutePoints ?? fallbackRoutePoints ?? roughRoutePoints;
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
export function attachDynamicRouting(cy: cytoscape.Core): () => void {
	let rafId: number | null = null;
	let chunkHandle: number | null = null;
	let chunkHandleType: 'idle' | 'timeout' | null = null;
	let settleTimeout: ReturnType<typeof setTimeout> | null = null;
	let pendingFastEdgeIds = new Set<string>();
	let pendingSettledEdgeIds = new Set<string>();
	const GRID_CHUNK_SIZE = 2;
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

	function scheduleGridChunks(edgeIds?: Set<string>) {
		cancelChunkRouting();
		const ids = edgeIds ? [...edgeIds] : cy.edges().map((edge) => edge.id());
		let index = 0;

		function step() {
			const chunk = new Set(ids.slice(index, index + GRID_CHUNK_SIZE));
			index += GRID_CHUNK_SIZE;
			if (chunk.size > 0) applyDynamicRouting(cy, { useGrid: true, edgeIds: chunk });
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

	function scheduleRouting(useGrid: boolean) {
		if (rafId !== null) return;
		rafId = requestAnimationFrame(() => {
			rafId = null;
			const edgeIds = useGrid ? undefined : new Set(pendingFastEdgeIds);
			if (!useGrid) pendingFastEdgeIds.clear();
			if (useGrid) {
				scheduleGridChunks(edgeIds);
			} else {
				applyDynamicRouting(cy, { useGrid, edgeIds });
			}
		});
	}

	function scheduleFastRouting(evt: cytoscape.EventObject) {
		cancelChunkRouting();
		if (settleTimeout !== null) {
			clearTimeout(settleTimeout);
			settleTimeout = null;
		}
		const node = evt.target as cytoscape.NodeSingular;
		node.connectedEdges().forEach((edge) => {
			pendingFastEdgeIds.add(edge.id());
			pendingSettledEdgeIds.add(edge.id());
		});
		scheduleRouting(false);
	}

	function scheduleDirtySettledRouting(evt?: cytoscape.EventObject) {
		if (evt?.target?.isNode?.()) {
			const node = evt.target as cytoscape.NodeSingular;
			node.connectedEdges().forEach((edge) => {
				pendingSettledEdgeIds.add(edge.id());
			});
		}
		if (settleTimeout !== null) clearTimeout(settleTimeout);
		settleTimeout = setTimeout(() => {
			settleTimeout = null;
			const edgeIds = new Set(pendingSettledEdgeIds);
			pendingSettledEdgeIds.clear();
			if (edgeIds.size === 0) return;
			scheduleGridChunks(edgeIds);
		}, 80);
	}

	function scheduleFullSettledRouting() {
		pendingFastEdgeIds.clear();
		pendingSettledEdgeIds.clear();
		if (settleTimeout !== null) clearTimeout(settleTimeout);
		settleTimeout = setTimeout(() => {
			settleTimeout = null;
			scheduleGridChunks();
		}, 500);
	}

	cy.on('position', 'node', scheduleFastRouting);
	cy.on('dragfree', 'node', scheduleDirtySettledRouting);
	cy.on('layoutstop', scheduleFullSettledRouting);
	cy.on('add', 'edge', scheduleFullSettledRouting);
	cy.on('data', 'node', scheduleFullSettledRouting); // re-route when node data (size) changes

	// Initial application: draw immediately, then refine shortest paths incrementally.
	applyDynamicRouting(cy, { useGrid: false });
	scheduleFullSettledRouting();

	return () => {
		cy.off('position', 'node', scheduleFastRouting);
		cy.off('dragfree', 'node', scheduleDirtySettledRouting);
		cy.off('layoutstop', scheduleFullSettledRouting);
		cy.off('add', 'edge', scheduleFullSettledRouting);
		cy.off('data', 'node', scheduleFullSettledRouting);
		if (rafId !== null) cancelAnimationFrame(rafId);
		cancelChunkRouting();
		if (settleTimeout !== null) clearTimeout(settleTimeout);
	};
}
