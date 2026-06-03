import type cytoscape from 'cytoscape';
import { getStoredRoutes, type Point, type RoutedEdgePath, type RoutedSegment } from '$lib/cytoscape/edgeRouter';

const SVG_NS = 'http://www.w3.org/2000/svg';
const BRIDGE_RADIUS = 10;
const BRIDGE_GAP_PADDING = 2;
const STITCH_OVERLAP = 5;
const BRIDGE_BUCKET_SIZE = 240;
const BRIDGE_GROUP_ATTR = 'data-edge-bridge-overlay';

interface RenderedPoint {
	x: number;
	y: number;
}

interface BridgeCrossing {
	segment: RoutedSegment;
	other: RoutedSegment;
	point: RenderedPoint;
	color: string;
	width: number;
	radius: number;
	opacity: number;
	style: 'bridge' | 'soft-bypass';
}

interface SegmentIntersection {
	point: Point;
	aT: number;
	bT: number;
}

interface BridgeDisplayOptions {
	hideAll: boolean;
	showSoftBypass: boolean;
	radius: number;
	opacity: number;
	softRadius: number;
	softOpacity: number;
}

interface SegmentBounds {
	minX: number;
	minY: number;
	maxX: number;
	maxY: number;
}

interface EdgeVisual {
	color: string;
	width: number;
	opacity: number;
}

function clamp(min: number, value: number, max: number): number {
	return Math.max(min, Math.min(max, value));
}

function bridgeDisplayOptions(zoom: number): BridgeDisplayOptions {
	if (zoom < 0.45) {
		return {
			hideAll: false,
			showSoftBypass: true,
			radius: clamp(1.6, BRIDGE_RADIUS * zoom * 0.72, 3.4),
			opacity: clamp(0.42, zoom * 1.35, 0.62),
			softRadius: clamp(1.2, BRIDGE_RADIUS * zoom * 0.52, 2.4),
			softOpacity: clamp(0.28, zoom * 1.1, 0.44),
		};
	}
	if (zoom < 0.65) {
		return {
			hideAll: false,
			showSoftBypass: true,
			radius: clamp(2.6, BRIDGE_RADIUS * zoom * 0.85, 5.4),
			opacity: 0.72,
			softRadius: clamp(1.8, BRIDGE_RADIUS * zoom * 0.62, 4),
			softOpacity: 0.58,
		};
	}
	if (zoom < 0.9) {
		return {
			hideAll: false,
			showSoftBypass: true,
			radius: clamp(4, BRIDGE_RADIUS * zoom, 8),
			opacity: 0.84,
			softRadius: clamp(3.2, BRIDGE_RADIUS * zoom * 0.78, 7),
			softOpacity: 0.76,
		};
	}
	return {
		hideAll: false,
		showSoftBypass: true,
		radius: BRIDGE_RADIUS,
		opacity: 1,
		softRadius: BRIDGE_RADIUS,
		softOpacity: 1,
	};
}

function isTransparent(color: string): boolean {
	return color === 'transparent' || color === 'rgba(0, 0, 0, 0)' || color === 'rgba(0,0,0,0)';
}

function resolveBackdropColor(host: HTMLElement): string {
	let current: HTMLElement | null = host;
	while (current) {
		const color = getComputedStyle(current).backgroundColor;
		if (!isTransparent(color)) return color;
		current = current.parentElement;
	}
	return 'rgb(17, 17, 20)';
}

function modelToRendered(cy: cytoscape.Core, point: Point): RenderedPoint {
	const zoom = cy.zoom();
	const pan = cy.pan();
	return {
		x: point.x * zoom + pan.x,
		y: point.y * zoom + pan.y,
	};
}

function isStrictInteriorParam(value: number): boolean {
	return value > 0.025 && value < 0.975;
}

function isLooseSegmentParam(value: number): boolean {
	return value > -0.01 && value < 1.01;
}

function segmentIntersection(a: RoutedSegment, b: RoutedSegment): SegmentIntersection | null {
	const p = a.from;
	const r = { x: a.to.x - a.from.x, y: a.to.y - a.from.y };
	const q = b.from;
	const s = { x: b.to.x - b.from.x, y: b.to.y - b.from.y };
	const denom = r.x * s.y - r.y * s.x;

	if (Math.abs(denom) < 0.001) return null;

	const qp = { x: q.x - p.x, y: q.y - p.y };
	const t = (qp.x * s.y - qp.y * s.x) / denom;
	const u = (qp.x * r.y - qp.y * r.x) / denom;

	if (!isLooseSegmentParam(t) || !isLooseSegmentParam(u)) return null;

	return {
		point: {
			x: p.x + t * r.x,
			y: p.y + t * r.y,
		},
		aT: t,
		bT: u,
	};
}

function segmentBounds(segment: RoutedSegment): SegmentBounds {
	return {
		minX: Math.min(segment.from.x, segment.to.x),
		minY: Math.min(segment.from.y, segment.to.y),
		maxX: Math.max(segment.from.x, segment.to.x),
		maxY: Math.max(segment.from.y, segment.to.y),
	};
}

function boundsOverlap(left: SegmentBounds, right: SegmentBounds): boolean {
	return left.minX <= right.maxX && left.maxX >= right.minX && left.minY <= right.maxY && left.maxY >= right.minY;
}

function segmentBucketKeys(bounds: SegmentBounds): string[] {
	const keys: string[] = [];
	const startX = Math.floor(bounds.minX / BRIDGE_BUCKET_SIZE);
	const endX = Math.floor(bounds.maxX / BRIDGE_BUCKET_SIZE);
	const startY = Math.floor(bounds.minY / BRIDGE_BUCKET_SIZE);
	const endY = Math.floor(bounds.maxY / BRIDGE_BUCKET_SIZE);

	for (let x = startX; x <= endX; x += 1) {
		for (let y = startY; y <= endY; y += 1) {
			keys.push(`${x}:${y}`);
		}
	}

	return keys;
}

function chooseCrossingStyle(
	a: RoutedSegment,
	b: RoutedSegment,
): { segment: RoutedSegment; other: RoutedSegment; style: 'bridge' | 'soft-bypass' } | null {
	if (a.axis === 'horizontal' && b.axis !== 'horizontal') {
		return { segment: a, other: b, style: 'bridge' };
	}
	if (b.axis === 'horizontal' && a.axis !== 'horizontal') {
		return { segment: b, other: a, style: 'bridge' };
	}
	if (a.axis === 'diagonal' && b.axis === 'vertical') {
		return { segment: a, other: b, style: 'soft-bypass' };
	}
	if (b.axis === 'diagonal' && a.axis === 'vertical') {
		return { segment: b, other: a, style: 'soft-bypass' };
	}
	if (a.axis === 'diagonal' && b.axis === 'diagonal') {
		const aDx = a.to.x - a.from.x;
		const aDy = a.to.y - a.from.y;
		const bDx = b.to.x - b.from.x;
		const bDy = b.to.y - b.from.y;
		const aTopLeftToBottomRight = aDx * aDy > 0;
		const bTopLeftToBottomRight = bDx * bDy > 0;

		if (aTopLeftToBottomRight && !bTopLeftToBottomRight) {
			return { segment: a, other: b, style: 'soft-bypass' };
		}
		if (bTopLeftToBottomRight && !aTopLeftToBottomRight) {
			return { segment: b, other: a, style: 'soft-bypass' };
		}
	}
	return null;
}

function compactCrossings(crossings: BridgeCrossing[]): BridgeCrossing[] {
	const grouped = new Map<string, BridgeCrossing[]>();
	for (const crossing of crossings) {
		const key = `${crossing.segment.edgeId}:${crossing.segment.index}`;
		const bucket = grouped.get(key) ?? [];
		bucket.push(crossing);
		grouped.set(key, bucket);
	}

	const result: BridgeCrossing[] = [];
	for (const bucket of grouped.values()) {
		bucket.sort((left, right) => {
			const leftProjection =
				(left.point.x - left.segment.from.x) * (left.segment.to.x - left.segment.from.x) +
				(left.point.y - left.segment.from.y) * (left.segment.to.y - left.segment.from.y);
			const rightProjection =
				(right.point.x - right.segment.from.x) * (right.segment.to.x - right.segment.from.x) +
				(right.point.y - right.segment.from.y) * (right.segment.to.y - right.segment.from.y);
			return leftProjection - rightProjection;
		});
		let previous: BridgeCrossing | null = null;
		for (const crossing of bucket) {
			if (!previous) {
				result.push(crossing);
				previous = crossing;
				continue;
			}
			const dx = crossing.point.x - previous.point.x;
			const dy = crossing.point.y - previous.point.y;
			const distance = Math.sqrt(dx * dx + dy * dy);
			const minDistance = crossing.style === 'soft-bypass' ? 56 : previous.radius + crossing.radius + 6;
			if (distance < minDistance) continue;
			result.push(crossing);
			previous = crossing;
		}
	}

	return result;
}

function buildBridgeCrossings(
	cy: cytoscape.Core,
	routes: RoutedEdgePath[],
	options: BridgeDisplayOptions,
): BridgeCrossing[] {
	if (options.hideAll) return [];

	const allSegments = routes.flatMap((route) => route.segments);
	const boundsCache = allSegments.map(segmentBounds);
	const bucketKeys = boundsCache.map(segmentBucketKeys);
	const buckets = new Map<string, number[]>();
	const edgeVisualCache = new Map<string, EdgeVisual | null>();
	const crossings: BridgeCrossing[] = [];

	function getEdgeVisual(edgeId: string): EdgeVisual | null {
		if (edgeVisualCache.has(edgeId)) return edgeVisualCache.get(edgeId) ?? null;

		const edge = cy.getElementById(edgeId);
		if (!edge.length) {
			edgeVisualCache.set(edgeId, null);
			return null;
		}

		const color = String(edge.style('line-color') || '#8b7fd9');
		const modelWidth = parseFloat(String(edge.style('width') || '1.5'));
		const modelOpacity = parseFloat(String(edge.style('opacity') || '1'));
		const visual = {
			color,
			width: Math.max(1, modelWidth * cy.zoom()),
			opacity: Number.isFinite(modelOpacity) ? clamp(0, modelOpacity, 1) : 1,
		};
		edgeVisualCache.set(edgeId, visual);
		return visual;
	}

	for (let i = 0; i < allSegments.length; i += 1) {
		const left = allSegments[i];
		const compared = new Set<number>();

		for (const key of bucketKeys[i]) {
			const candidates = buckets.get(key);
			if (!candidates) continue;

			for (const j of candidates) {
				if (compared.has(j)) continue;
				compared.add(j);
				const right = allSegments[j];

				if (left.edgeId === right.edgeId) continue;
				if (!boundsOverlap(boundsCache[i], boundsCache[j])) continue;

				const chosen = chooseCrossingStyle(left, right);
				if (!chosen) continue;
				if (chosen.style === 'soft-bypass' && !options.showSoftBypass) continue;

				const intersection = segmentIntersection(left, right);
				if (!intersection) continue;

				const chosenT = chosen.segment === left ? intersection.aT : intersection.bT;
				const otherT = chosen.segment === left ? intersection.bT : intersection.aT;
				if (chosen.style === 'bridge') {
					// Horizontal jumps should still render near bends, but the crossed
					// segment should be a real pass-through rather than an endpoint touch.
					if (!isLooseSegmentParam(chosenT) || !isStrictInteriorParam(otherT)) continue;
				} else if (!isStrictInteriorParam(chosenT) || !isStrictInteriorParam(otherT)) {
					continue;
				}

				const visual = getEdgeVisual(chosen.segment.edgeId);
				if (!visual) continue;

				const renderedPoint = modelToRendered(cy, intersection.point);

				crossings.push({
					segment: chosen.segment,
					other: chosen.other,
					point: renderedPoint,
					color: visual.color,
					width: visual.width,
					radius: chosen.style === 'soft-bypass' ? options.softRadius : options.radius,
					opacity: visual.opacity * (chosen.style === 'soft-bypass' ? options.softOpacity : options.opacity),
					style: chosen.style,
				});
			}
		}

		for (const key of bucketKeys[i]) {
			const bucket = buckets.get(key);
			if (bucket) bucket.push(i);
			else buckets.set(key, [i]);
		}
	}

	return compactCrossings(crossings);
}

function createPathElement(d: string, stroke: string, width: number): SVGPathElement {
	const path = document.createElementNS(SVG_NS, 'path');
	path.setAttribute('d', d);
	path.setAttribute('fill', 'none');
	path.setAttribute('stroke', stroke);
	path.setAttribute('stroke-width', `${width}`);
	path.setAttribute('stroke-linecap', 'round');
	path.setAttribute('stroke-linejoin', 'round');
	return path;
}

function setOpacity(path: SVGPathElement, opacity: number): SVGPathElement {
	path.setAttribute('opacity', `${opacity}`);
	return path;
}

function renderBridges(
	svg: SVGSVGElement,
	crossings: BridgeCrossing[],
	backdrop: string,
): void {
	svg.replaceChildren();

	for (const crossing of crossings) {
		const { point, color, radius, width, style, segment, other, opacity } = crossing;
		const eraseWidth = width + 5;

		if (style === 'bridge') {
			const startX = point.x - radius - BRIDGE_GAP_PADDING;
			const endX = point.x + radius + BRIDGE_GAP_PADDING;
			const baseY = point.y;
			const liftY = baseY - radius * 1.9;
			const arcPath = `M ${startX} ${baseY} Q ${point.x} ${liftY} ${endX} ${baseY}`;

			svg.appendChild(setOpacity(createPathElement(`M ${startX} ${baseY} L ${endX} ${baseY}`, backdrop, eraseWidth), opacity));
			svg.appendChild(setOpacity(createPathElement(arcPath, backdrop, eraseWidth), opacity));
			svg.appendChild(setOpacity(createPathElement(arcPath, color, width), opacity));
			continue;
		}

		const from = segment.from;
		const to = segment.to;
		const segDx = to.x - from.x;
		const segDy = to.y - from.y;
		const segLen = Math.sqrt(segDx * segDx + segDy * segDy) || 1;
		const ux = segDx / segLen;
		const uy = segDy / segLen;
		const nx = -uy;
		const ny = ux;

		const renderedFrom = point;
		const start = {
			x: renderedFrom.x - ux * (radius * 1.8 + STITCH_OVERLAP),
			y: renderedFrom.y - uy * (radius * 1.8 + STITCH_OVERLAP),
		};
		const end = {
			x: renderedFrom.x + ux * (radius * 1.8 + STITCH_OVERLAP),
			y: renderedFrom.y + uy * (radius * 1.8 + STITCH_OVERLAP),
		};
		const otherDx = other.to.x - other.from.x;
		const otherDy = other.to.y - other.from.y;
		let sign = (nx * otherDx + ny * otherDy) >= 0 ? -1 : 1;
		if (Math.abs(ny) > 0.001) {
			// Screen coordinates grow downward, so a visual "jump" should always
			// bias its shoulder toward negative Y.
			sign = ny < 0 ? 1 : -1;
		}
		const control1 = {
			x: point.x - ux * (radius * 0.36) + nx * radius * 1.9 * sign,
			y: point.y - uy * (radius * 0.36) + ny * radius * 1.9 * sign,
		};
		const control2 = {
			x: point.x + ux * (radius * 0.36) + nx * radius * 1.9 * sign,
			y: point.y + uy * (radius * 0.36) + ny * radius * 1.9 * sign,
		};
		const eraseStart = {
			x: renderedFrom.x - ux * (radius * 1.8),
			y: renderedFrom.y - uy * (radius * 1.8),
		};
		const eraseEnd = {
			x: renderedFrom.x + ux * (radius * 1.8),
			y: renderedFrom.y + uy * (radius * 1.8),
		};
		const erasePath = `M ${eraseStart.x} ${eraseStart.y} L ${eraseEnd.x} ${eraseEnd.y}`;
		const curvePath = `M ${start.x} ${start.y} C ${control1.x} ${control1.y}, ${control2.x} ${control2.y}, ${end.x} ${end.y}`;
		svg.appendChild(setOpacity(createPathElement(erasePath, backdrop, eraseWidth), opacity));
		svg.appendChild(setOpacity(createPathElement(curvePath, color, width), opacity));
	}
}

export function attachEdgeBridgeOverlay(cy: cytoscape.Core): () => void {
	const cyContainer = cy.container();
	if (!cyContainer) return () => {};

	const host = (cyContainer.parentElement ?? cyContainer) as HTMLElement;
	if (getComputedStyle(host).position === 'static') host.style.position = 'relative';

	const svg = document.createElementNS(SVG_NS, 'svg');
	svg.setAttribute(BRIDGE_GROUP_ATTR, 'true');
	svg.setAttribute('aria-hidden', 'true');
	svg.style.position = 'absolute';
	svg.style.inset = '0';
	svg.style.width = '100%';
	svg.style.height = '100%';
	svg.style.pointerEvents = 'none';
	svg.style.overflow = 'hidden';
	svg.style.zIndex = '5';
	host.appendChild(svg);

	let rafId: number | null = null;
	let routesDirty = true;
	let cachedRoutes: RoutedEdgePath[] = [];
	const resizeObserver = new ResizeObserver(() => scheduleRender());

	function getCachedRoutes(): RoutedEdgePath[] {
		if (routesDirty) {
			cachedRoutes = getStoredRoutes(cy);
			routesDirty = false;
		}
		return cachedRoutes;
	}

	function render() {
		rafId = null;
		const rect = host.getBoundingClientRect();
		svg.setAttribute('viewBox', `0 0 ${rect.width} ${rect.height}`);
		const backdrop = resolveBackdropColor(host);
		const options = bridgeDisplayOptions(cy.zoom());
		const routes = getCachedRoutes();
		const crossings = buildBridgeCrossings(cy, routes, options);
		renderBridges(svg, crossings, backdrop);
	}

	function scheduleRender() {
		if (rafId !== null) return;
		rafId = requestAnimationFrame(render);
	}

	function markRoutesDirty() {
		routesDirty = true;
		scheduleRender();
	}

	resizeObserver.observe(host);
	cy.on('layoutstop', markRoutesDirty);
	cy.on('add remove data', markRoutesDirty);
	cy.on('thask:filter', scheduleRender);
	cy.on('pan zoom', scheduleRender);

	scheduleRender();

	return () => {
		cy.off('layoutstop', markRoutesDirty);
		cy.off('add remove data', markRoutesDirty);
		cy.off('thask:filter', scheduleRender);
		cy.off('pan zoom', scheduleRender);
		resizeObserver.disconnect();
		if (rafId !== null) cancelAnimationFrame(rafId);
		svg.remove();
	};
}
