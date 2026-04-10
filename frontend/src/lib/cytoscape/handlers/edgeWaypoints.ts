import type cytoscape from 'cytoscape';
import { api } from '$lib/api';

interface WaypointOptions {
	getApiBase: () => string;
}

const MARKER_SIZE = 10;

const SNAP_PIXEL_THRESHOLD = 8;
const SNAP_ANGLE_DEG = 3;

interface SnapGuide {
	type: 'vertical' | 'horizontal' | 'angle';
	anchor: { x: number; y: number };
	angle?: number;
}

interface AxisSnap {
	x: number;
	y: number;
	guides: SnapGuide[];
}

/**
 * Combined axis + 8-direction snapping.
 * - X/Y axis: snap if point.x or point.y is within pixel threshold of any anchor
 * - 8-direction: snap if angle from anchor to point is within degree threshold of 45° increment
 * All can apply simultaneously.
 */
function axisSnap(
	anchors: Array<{ x: number; y: number }>,
	point: { x: number; y: number },
): AxisSnap {
	let resultX = point.x;
	let resultY = point.y;
	let bestDiffX = SNAP_PIXEL_THRESHOLD;
	let bestDiffY = SNAP_PIXEL_THRESHOLD;
	const guides: SnapGuide[] = [];
	let axisSnappedX = false;
	let axisSnappedY = false;

	// 1. X/Y axis snap
	for (const anchor of anchors) {
		const diffX = Math.abs(point.x - anchor.x);
		const diffY = Math.abs(point.y - anchor.y);

		if (diffX < bestDiffX) {
			bestDiffX = diffX;
			resultX = anchor.x;
			axisSnappedX = true;
		}
		if (diffY < bestDiffY) {
			bestDiffY = diffY;
			resultY = anchor.y;
			axisSnappedY = true;
		}
	}

	// Record axis guides
	if (axisSnappedX) {
		const anchor = anchors.find(a => Math.abs(resultX - a.x) < 0.1);
		if (anchor) guides.push({ type: 'vertical', anchor });
	}
	if (axisSnappedY) {
		const anchor = anchors.find(a => Math.abs(resultY - a.y) < 0.1);
		if (anchor) guides.push({ type: 'horizontal', anchor });
	}

	// 2. 8-direction angle snap (only if axis snap didn't fire for that dimension)
	if (!axisSnappedX || !axisSnappedY) {
		let bestAngleDiff = SNAP_ANGLE_DEG;
		let bestAngleSnap: { pos: { x: number; y: number }; anchor: { x: number; y: number }; angle: number } | null = null;

		for (const anchor of anchors) {
			const dx = point.x - anchor.x;
			const dy = point.y - anchor.y;
			const dist = Math.sqrt(dx * dx + dy * dy);
			if (dist < 1) continue;

			const angle = Math.atan2(dy, dx);
			// Only check diagonal angles (45°, 135°, 225°, 315°)
			for (const snapAngle of [Math.PI / 4, 3 * Math.PI / 4, -Math.PI / 4, -3 * Math.PI / 4]) {
				let diff = angle - snapAngle;
				while (diff > Math.PI) diff -= 2 * Math.PI;
				while (diff < -Math.PI) diff += 2 * Math.PI;
				const diffDeg = Math.abs(diff * 180 / Math.PI);

				if (diffDeg < bestAngleDiff) {
					bestAngleDiff = diffDeg;
					bestAngleSnap = {
						pos: {
							x: anchor.x + dist * Math.cos(snapAngle),
							y: anchor.y + dist * Math.sin(snapAngle),
						},
						anchor,
						angle: snapAngle,
					};
				}
			}
		}

		if (bestAngleSnap) {
			if (!axisSnappedX) resultX = bestAngleSnap.pos.x;
			if (!axisSnappedY) resultY = bestAngleSnap.pos.y;
			guides.push({ type: 'angle', anchor: bestAngleSnap.anchor, angle: bestAngleSnap.angle });
		}
	}

	return { x: resultX, y: resultY, guides };
}


export function attachWaypointHandlers(
	cy: cytoscape.Core,
	options: WaypointOptions,
): { cleanup: () => void } {
	const debounceTimers = new Map<string, ReturnType<typeof setTimeout>>();
	let activeEdgeId: string | null = null;
	let markerContainer: HTMLDivElement | null = null;
	let isDragging = false;

	// --- Save ---

	function saveWaypoints(edgeId: string, waypoints: Array<{ x: number; y: number }>) {
		const existing = debounceTimers.get(edgeId);
		if (existing) clearTimeout(existing);
		debounceTimers.set(edgeId, setTimeout(async () => {
			debounceTimers.delete(edgeId);
			const base = options.getApiBase();
			if (!base) return;
			await api.patch(`${base}/edges/${edgeId}`, { waypoints });
		}, 500));
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

	// --- Snap guideline ---
	let snapGuideEl: HTMLDivElement | null = null;

	function showSnapGuideRaw(svgContent: string) {
		if (!snapGuideEl) {
			const cyEl = cy.container();
			if (!cyEl) return;
			snapGuideEl = document.createElement('div');
			snapGuideEl.style.cssText = 'position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:3;';
			cyEl.appendChild(snapGuideEl);
		}
		snapGuideEl.innerHTML = `<svg style="position:absolute;top:0;left:0;width:100%;height:100%;overflow:visible;pointer-events:none;">${svgContent}</svg>`;
		snapGuideEl.style.display = '';
	}

	function hideSnapGuide() {
		if (snapGuideEl) snapGuideEl.style.display = 'none';
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
					const z = cy.zoom();
					const p = cy.pan();
					const dx = (me.clientX - startX) / z;
					const dy = (me.clientY - startY) / z;
					const rawX = origX + dx;
					const rawY = origY + dy;

					// Axis snap — independent X/Y snapping against prev + next anchors
					const wpsLive: Array<{ x: number; y: number }> = [...(edge.data('waypoints') || [])];
					const prevAnchor = i === 0 ? edge.source().position() : wpsLive[i - 1];
					const nextAnchor = i === wpsLive.length - 1 ? edge.target().position() : wpsLive[i + 1];

					const snap = axisSnap([prevAnchor, nextAnchor], { x: rawX, y: rawY });

					marker.style.left = `${snap.x * z + p.x - half}px`;
					marker.style.top = `${snap.y * z + p.y - half}px`;

					// Show guidelines
					if (snap.guides.length > 0) {
						let svgLines = '';
						for (const guide of snap.guides) {
							if (guide.type === 'vertical') {
								const rx = guide.anchor.x * z + p.x;
								svgLines += `<line x1="${rx}" y1="0" x2="${rx}" y2="9999" stroke="#c9a84c" stroke-width="1" stroke-dasharray="4 4" opacity="0.4"/>`;
							} else if (guide.type === 'horizontal') {
								const ry = guide.anchor.y * z + p.y;
								svgLines += `<line x1="0" y1="${ry}" x2="9999" y2="${ry}" stroke="#c9a84c" stroke-width="1" stroke-dasharray="4 4" opacity="0.4"/>`;
							} else if (guide.type === 'angle' && guide.angle !== undefined) {
								const ax = guide.anchor.x * z + p.x;
								const ay = guide.anchor.y * z + p.y;
								const len = 2000;
								svgLines += `<line x1="${ax - len * Math.cos(guide.angle)}" y1="${ay - len * Math.sin(guide.angle)}" x2="${ax + len * Math.cos(guide.angle)}" y2="${ay + len * Math.sin(guide.angle)}" stroke="#c9a84c" stroke-width="1" stroke-dasharray="4 4" opacity="0.4"/>`;
							}
						}
						showSnapGuideRaw(svgLines);
					} else {
						hideSnapGuide();
					}

					// Update edge in real-time
					wpsLive[i] = { x: snap.x, y: snap.y };
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
					hideSnapGuide();

					const z = cy.zoom();
					const dx = (me.clientX - startX) / z;
					const dy = (me.clientY - startY) / z;

					setTimeout(() => { isDragging = false; }, 50);

					if (Math.abs(dx) < 1 && Math.abs(dy) < 1) return;

					const wps: Array<{ x: number; y: number }> = [...(edge.data('waypoints') || [])];
					const prevA = i === 0 ? edge.source().position() : wps[i - 1];
					const nextA = i === wps.length - 1 ? edge.target().position() : wps[i + 1];
					const finalSnap = axisSnap([prevA, nextA], { x: origX + dx, y: origY + dy });
					wps[i] = { x: finalSnap.x, y: finalSnap.y };
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

	// --- SVG overlay for waypoint edges ---

	let svgOverlay: SVGSVGElement | null = null;

	function ensureSvgOverlay(): SVGSVGElement | null {
		if (svgOverlay) return svgOverlay;
		const cyEl = cy.container();
		if (!cyEl) return null;
		svgOverlay = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
		svgOverlay.style.cssText = 'position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none;z-index:4;overflow:hidden;';
		cyEl.appendChild(svgOverlay);
		return svgOverlay;
	}

	function renderWaypointEdges() {
		const svg = ensureSvgOverlay();
		if (!svg) return;
		svg.innerHTML = '';

		const pan = cy.pan();
		const zoom = cy.zoom();

		cy.edges().forEach((edge: cytoscape.EdgeSingular) => {
			const wps: Array<{ x: number; y: number }> = edge.data('waypoints') || [];
			if (wps.length === 0) return;
			if (edge.hidden()) return;

			const srcPos = edge.source().position();
			const tgtPos = edge.target().position();
			const edgeType = (edge.data('edgeType') || 'related') as string;

			// Edge colors by type
			const colorMap: Record<string, string> = {
				depends_on: '#a78bfa', blocks: '#e05252', related: '#7c7570',
				parent_child: '#7ca3c4', triggers: '#e2b340',
			};
			const color = colorMap[edgeType] || '#7c7570';

			const points = [srcPos, ...wps, tgtPos];
			const rendered = points.map(p => ({
				x: p.x * zoom + pan.x,
				y: p.y * zoom + pan.y,
			}));

			const polyline = document.createElementNS('http://www.w3.org/2000/svg', 'polyline');
			polyline.setAttribute('points', rendered.map(p => `${p.x},${p.y}`).join(' '));
			polyline.setAttribute('fill', 'none');
			polyline.setAttribute('stroke', color);
			polyline.setAttribute('stroke-width', String(1.5 * zoom));
			polyline.setAttribute('opacity', '0.75');

			if (edgeType === 'blocks') {
				polyline.setAttribute('stroke-dasharray', `${8 * zoom} ${4 * zoom}`);
				polyline.setAttribute('stroke-width', String(2 * zoom));
			} else if (edgeType === 'parent_child' || edgeType === 'related') {
				polyline.setAttribute('stroke-dasharray', `${4 * zoom} ${4 * zoom}`);
				polyline.setAttribute('stroke-width', String(1 * zoom));
			}

			svg.appendChild(polyline);

			// Arrow at end
			const last = rendered[rendered.length - 1];
			const prev = rendered[rendered.length - 2];
			const angle = Math.atan2(last.y - prev.y, last.x - prev.x);
			const sz = 8 * zoom;
			const arrow = document.createElementNS('http://www.w3.org/2000/svg', 'polygon');
			const px = sz * 0.4 * Math.cos(angle + Math.PI / 2);
			const py = sz * 0.4 * Math.sin(angle + Math.PI / 2);
			arrow.setAttribute('points', `${last.x},${last.y} ${last.x - sz * Math.cos(angle) + px},${last.y - sz * Math.sin(angle) + py} ${last.x - sz * Math.cos(angle) - px},${last.y - sz * Math.sin(angle) - py}`);
			arrow.setAttribute('fill', color);
			arrow.setAttribute('opacity', '0.75');
			svg.appendChild(arrow);
		});
	}

	function updateEdgeSegments(_edge: cytoscape.EdgeSingular) {
		renderWaypointEdges();
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
		renderWaypointEdges();
		if (!activeEdgeId) return;
		const edge = cy.getElementById(activeEdgeId);
		if (edge.length && edge.isEdge()) {
			showMarkers(edge as cytoscape.EdgeSingular);
		}
	}

	// Initial SVG render
	requestAnimationFrame(() => renderWaypointEdges());

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
	cy.on('add remove data', 'edge', () => renderWaypointEdges());

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
			preDragPositions.clear();
			for (const timer of debounceTimers.values()) clearTimeout(timer);
			debounceTimers.clear();
			if (markerContainer) {
				markerContainer.remove();
				markerContainer = null;
			}
			cy.off('add remove data', 'edge');
			if (svgOverlay) {
				svgOverlay.remove();
				svgOverlay = null;
			}
			if (snapGuideEl) {
				snapGuideEl.remove();
				snapGuideEl = null;
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
