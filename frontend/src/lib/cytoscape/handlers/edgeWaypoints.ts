import type cytoscape from 'cytoscape';
import { api } from '$lib/api';

interface WaypointOptions {
	getApiBase: () => string;
}

const MARKER_SIZE = 10;


export function attachWaypointHandlers(
	cy: cytoscape.Core,
	options: WaypointOptions,
): { cleanup: () => void } {
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;
	let activeEdgeId: string | null = null;
	let markerContainer: HTMLDivElement | null = null;
	let isDragging = false;

	// --- Save ---

	function saveWaypoints(edgeId: string, waypoints: Array<{ x: number; y: number }>) {
		if (debounceTimer) clearTimeout(debounceTimer);
		debounceTimer = setTimeout(async () => {
			const base = options.getApiBase();
			if (!base) return;
			await api.patch(`${base}/edges/${edgeId}`, { waypoints });
		}, 500);
	}

	// --- Marker container ---

	function ensureContainer(): HTMLDivElement | null {
		if (markerContainer) return markerContainer;
		const cyEl = cy.container();
		if (!cyEl) return null;
		markerContainer = document.createElement('div');
		markerContainer.style.cssText = 'position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:20;';
		cyEl.appendChild(markerContainer);
		return markerContainer;
	}

	function clearMarkers() {
		if (markerContainer) markerContainer.innerHTML = '';
		activeEdgeId = null;
	}

	// --- Show markers for selected edge ---

	function showMarkers(edge: cytoscape.EdgeSingular) {
		const container = ensureContainer();
		if (!container) return;
		container.innerHTML = '';
		activeEdgeId = edge.id();

		const waypoints: Array<{ x: number; y: number }> = edge.data('waypoints') || [];
		if (waypoints.length === 0) return;

		const pan = cy.pan();
		const zoom = cy.zoom();
		const half = MARKER_SIZE / 2;

		waypoints.forEach((wp, i) => {
			const rx = wp.x * zoom + pan.x;
			const ry = wp.y * zoom + pan.y;

			const marker = document.createElement('div');
			marker.style.cssText = `
				position:absolute;
				left:${rx - half}px;top:${ry - half}px;
				width:${MARKER_SIZE}px;height:${MARKER_SIZE}px;
				border-radius:50%;
				background:#c9a84c;
				border:2px solid #a8893a;
				cursor:grab;
				pointer-events:auto;
			`;

			// --- Drag ---
			marker.addEventListener('mousedown', (e) => {
				e.preventDefault();
				e.stopPropagation();
				isDragging = true;
				const startX = e.clientX;
				const startY = e.clientY;
				const origX = wp.x;
				const origY = wp.y;
				marker.style.cursor = 'grabbing';
				marker.style.transform = 'scale(1.4)';
				marker.style.zIndex = '100';
				cy.userPanningEnabled(false);

				const onMove = (me: MouseEvent) => {
					const dx = (me.clientX - startX) / zoom;
					const dy = (me.clientY - startY) / zoom;
					const newX = origX + dx;
					const newY = origY + dy;

					// Update marker position
					marker.style.left = `${newX * zoom + pan.x - half}px`;
					marker.style.top = `${newY * zoom + pan.y - half}px`;

					// Update edge segments in real-time
					const wpsLive: Array<{ x: number; y: number }> = [...(edge.data('waypoints') || [])];
					wpsLive[i] = { x: newX, y: newY };
					edge.data('waypoints', wpsLive);
					updateEdgeSegments(edge);
				};

				const onUp = (me: MouseEvent) => {
					document.removeEventListener('mousemove', onMove);
					document.removeEventListener('mouseup', onUp);
					cy.userPanningEnabled(true);
					marker.style.cursor = 'grab';
					marker.style.transform = '';
					marker.style.zIndex = '';

					const dx = (me.clientX - startX) / zoom;
					const dy = (me.clientY - startY) / zoom;

					setTimeout(() => { isDragging = false; }, 50);

					if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return;

					const wps: Array<{ x: number; y: number }> = [...(edge.data('waypoints') || [])];
					wps[i] = { x: origX + dx, y: origY + dy };
					edge.data('waypoints', wps);
					updateEdgeSegments(edge);
					saveWaypoints(edge.id(), wps);
					showMarkers(edge);
				};

				document.addEventListener('mousemove', onMove);
				document.addEventListener('mouseup', onUp);
			});

			// --- Delete on double-click ---
			marker.addEventListener('dblclick', (e) => {
				e.preventDefault();
				e.stopPropagation();
				const wps: Array<{ x: number; y: number }> = [...(edge.data('waypoints') || [])];
				wps.splice(i, 1);
				edge.data('waypoints', wps);
				updateEdgeSegments(edge);
				saveWaypoints(edge.id(), wps);
				showMarkers(edge);
			});

			container.appendChild(marker);
		});
	}

	// --- Segment recalculation ---

	function updateEdgeSegments(edge: cytoscape.EdgeSingular) {
		const waypoints: Array<{ x: number; y: number }> = edge.data('waypoints') || [];
		if (waypoints.length === 0) {
			edge.data('curveStyle', undefined);
			edge.data('segmentDistances', undefined);
			edge.data('segmentWeights', undefined);
			edge.removeStyle('curve-style segment-distances segment-weights');
			return;
		}

		const src = edge.source().position();
		const tgt = edge.target().position();
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
			weights.push(Math.max(0.01, Math.min(0.99, weight)));
			distances.push(dist);
		}

		edge.data('curveStyle', 'segments');
		edge.data('segmentDistances', distances);
		edge.data('segmentWeights', weights);
	}

	// --- Cytoscape events ---

	function onEdgeDblTap(e: cytoscape.EventObject) {
		if (isDragging) return;
		const edge = e.target;
		const pos = e.position;
		const waypoints: Array<{ x: number; y: number }> = [...(edge.data('waypoints') || [])];

		const srcPos = edge.source().position();
		const tgtPos = edge.target().position();
		const allPoints = [srcPos, ...waypoints, tgtPos];

		let bestIdx = waypoints.length;
		let bestDist = Infinity;

		for (let i = 0; i < allPoints.length - 1; i++) {
			const dist = pointToSegmentDist(pos, allPoints[i], allPoints[i + 1]);
			if (dist < bestDist) {
				bestDist = dist;
				bestIdx = i;
			}
		}

		waypoints.splice(Math.max(0, bestIdx), 0, { x: pos.x, y: pos.y });
		edge.data('waypoints', waypoints);
		updateEdgeSegments(edge);
		saveWaypoints(edge.id(), waypoints);

		// Select edge to show markers
		edge.select();
		showMarkers(edge);
	}

	function onEdgeSelect(e: cytoscape.EventObject) {
		const edge = e.target;
		if (edge.isEdge()) showMarkers(edge as cytoscape.EdgeSingular);
	}

	function onEdgeUnselect() {
		if (!isDragging) clearMarkers();
	}

	function onEdgeMouseOver(e: cytoscape.EventObject) {
		if (isDragging) return;
		const edge = e.target;
		if (edge.isEdge()) {
			const wps = edge.data('waypoints') || [];
			if (wps.length > 0) showMarkers(edge as cytoscape.EdgeSingular);
		}
	}

	function onEdgeMouseOut() {
		if (isDragging) return;
		// Only hide if edge is not selected
		if (activeEdgeId) {
			const edge = cy.getElementById(activeEdgeId);
			if (edge.length && edge.selected()) return;
		}
		clearMarkers();
	}

	function onViewport() {
		if (!activeEdgeId) return;
		const edge = cy.getElementById(activeEdgeId);
		if (edge.length && edge.isEdge()) {
			showMarkers(edge as cytoscape.EdgeSingular);
		}
	}

	cy.on('dbltap', 'edge', onEdgeDblTap);
	cy.on('select', 'edge', onEdgeSelect);
	cy.on('unselect', 'edge', onEdgeUnselect);
	cy.on('mouseover', 'edge', onEdgeMouseOver);
	cy.on('mouseout', 'edge', onEdgeMouseOut);
	cy.on('pan zoom', onViewport);

	// Track node positions before drag
	let preDragPositions: Map<string, { x: number; y: number }> = new Map();

	function onNodeGrab(e: cytoscape.EventObject) {
		const node = e.target;
		preDragPositions.set(node.id(), { ...node.position() });
	}

	function onNodeDragFree(e: cytoscape.EventObject) {
		const node = e.target;
		const prev = preDragPositions.get(node.id());
		if (!prev) return;
		const dx = node.position().x - prev.x;
		const dy = node.position().y - prev.y;
		preDragPositions.delete(node.id());
		if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return;

		node.connectedEdges().forEach((edge: cytoscape.EdgeSingular) => {
			const wps: Array<{ x: number; y: number }> = edge.data('waypoints') || [];
			if (wps.length === 0) return;

			const isSource = edge.source().id() === node.id();
			const isTarget = edge.target().id() === node.id();
			const count = wps.length;

			const updated = wps.map((wp, i) => {
				const ratio = (isSource && isTarget) ? 1 :
				              isSource ? 1 - (i / count) :
				              isTarget ? (i + 1) / count :
				              0;
				return { x: wp.x + dx * ratio, y: wp.y + dy * ratio };
			});

			edge.data('waypoints', updated);
			updateEdgeSegments(edge);
			saveWaypoints(edge.id(), updated);
		});
	}

	cy.on('grab', 'node', onNodeGrab);
	cy.on('dragfree', 'node', onNodeDragFree);

	return {
		cleanup: () => {
			cy.off('dbltap', 'edge', onEdgeDblTap);
			cy.off('select', 'edge', onEdgeSelect);
			cy.off('unselect', 'edge', onEdgeUnselect);
			cy.off('mouseover', 'edge', onEdgeMouseOver);
			cy.off('mouseout', 'edge', onEdgeMouseOut);
			cy.off('pan zoom', onViewport);
			cy.off('grab', 'node', onNodeGrab);
			cy.off('dragfree', 'node', onNodeDragFree);
			clearMarkers();
			if (debounceTimer) clearTimeout(debounceTimer);
			if (markerContainer) {
				markerContainer.remove();
				markerContainer = null;
			}
		},
	};
}

function pointToSegmentDist(
	p: { x: number; y: number },
	a: { x: number; y: number },
	b: { x: number; y: number },
): number {
	const dx = b.x - a.x;
	const dy = b.y - a.y;
	const lenSq = dx * dx + dy * dy;
	if (lenSq === 0) return Math.sqrt((p.x - a.x) ** 2 + (p.y - a.y) ** 2);
	let t = ((p.x - a.x) * dx + (p.y - a.y) * dy) / lenSq;
	t = Math.max(0, Math.min(1, t));
	const projX = a.x + t * dx;
	const projY = a.y + t * dy;
	return Math.sqrt((p.x - projX) ** 2 + (p.y - projY) ** 2);
}
