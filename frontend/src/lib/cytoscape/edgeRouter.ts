import type cytoscape from 'cytoscape';

interface Point { x: number; y: number }

/**
 * Compute a single 8-direction waypoint between src and tgt.
 * Returns null when the edge is already axis-aligned or perfectly diagonal.
 */
function compute8DirWaypoint(src: Point, tgt: Point): Point | null {
	const dx = tgt.x - src.x;
	const dy = tgt.y - src.y;
	const absDx = Math.abs(dx);
	const absDy = Math.abs(dy);

	if (absDx < 0.5 || absDy < 0.5 || Math.abs(absDx - absDy) < 0.5) {
		return null;
	}

	if (absDx >= absDy) {
		return { x: src.x + Math.sign(dx) * absDy, y: tgt.y };
	}
	return { x: tgt.x, y: src.y + Math.sign(dy) * absDx };
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
		const wp = compute8DirWaypoint(src, tgt);

		if (wp) {
			const { distances, weights } = waypointsToSegments(src, tgt, [wp]);
			edge.data({
				curveStyle: 'segments',
				segmentDistances: distances,
				segmentWeights: weights,
			});
		} else {
			edge.removeData('curveStyle');
			edge.removeData('segmentDistances');
			edge.removeData('segmentWeights');
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

	// Initial application
	applyDynamicRouting(cy);

	return () => {
		cy.off('position', 'node', scheduleRouting);
		cy.off('layoutstop', scheduleRouting);
		if (rafId !== null) cancelAnimationFrame(rafId);
	};
}
