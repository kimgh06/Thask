import type cytoscape from 'cytoscape';

interface Point { x: number; y: number }

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
	cy.edges().forEach((edge) => {
		const src = edge.source().position();
		const tgt = edge.target().position();
		const wps = compute8DirWaypoints(src, tgt);

		if (wps.length > 0) {
			const { distances, weights } = waypointsToSegments(src, tgt, wps);
			edge.data({
				curveStyle: 'segments',
				segmentDistances: distances,
				segmentWeights: weights,
			});
		} else {
			// Straight diagonal: keep segments style with near-zero offset
			edge.data({
				curveStyle: 'segments',
				segmentDistances: [0.1],
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
