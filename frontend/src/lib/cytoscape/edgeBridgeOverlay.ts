import type cytoscape from 'cytoscape';
import { getStoredRoutes, type Point, type RoutedEdgePath, type RoutedSegment } from '$lib/cytoscape/edgeRouter';

const SVG_NS = 'http://www.w3.org/2000/svg';
const BRIDGE_RADIUS = 10;
const BRIDGE_GAP_PADDING = 2;
const STITCH_OVERLAP = 5;
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
	const crossings: BridgeCrossing[] = [];

	for (let i = 0; i < allSegments.length; i += 1) {
		for (let j = i + 1; j < allSegments.length; j += 1) {
			const left = allSegments[i];
			const right = allSegments[j];

			if (left.edgeId === right.edgeId) continue;

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

			const edge = cy.getElementById(chosen.segment.edgeId);
			if (!edge.length) continue;

			const renderedPoint = modelToRendered(cy, intersection.point);
			const color = String(edge.style('line-color') || '#8b7fd9');
			const modelWidth = parseFloat(String(edge.style('width') || '1.5'));
			const width = Math.max(1, modelWidth * cy.zoom());

			crossings.push({
				segment: chosen.segment,
				other: chosen.other,
				point: renderedPoint,
				color,
				width,
				radius: chosen.style === 'soft-bypass' ? options.softRadius : options.radius,
				opacity: chosen.style === 'soft-bypass' ? options.softOpacity : options.opacity,
				style: chosen.style,
			});
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

			svg.appendChild(createPathElement(`M ${startX} ${baseY} L ${endX} ${baseY}`, backdrop, eraseWidth));
			svg.appendChild(createPathElement(arcPath, backdrop, eraseWidth));
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
		svg.appendChild(createPathElement(erasePath, backdrop, eraseWidth));
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
	const resizeObserver = new ResizeObserver(() => scheduleRender());

	function render() {
		rafId = null;
		const rect = host.getBoundingClientRect();
		svg.setAttribute('viewBox', `0 0 ${rect.width} ${rect.height}`);
		const backdrop = resolveBackdropColor(host);
		const options = bridgeDisplayOptions(cy.zoom());
		const routes = getStoredRoutes(cy);
		const crossings = buildBridgeCrossings(cy, routes, options);
		renderBridges(svg, crossings, backdrop);
	}

	function scheduleRender() {
		if (rafId !== null) return;
		rafId = requestAnimationFrame(render);
	}

	resizeObserver.observe(host);
	cy.on('render', scheduleRender);
	cy.on('layoutstop', scheduleRender);
	cy.on('position', 'node', scheduleRender);
	cy.on('add remove data', scheduleRender);

	scheduleRender();

	return () => {
		cy.off('render', scheduleRender);
		cy.off('layoutstop', scheduleRender);
		cy.off('position', 'node', scheduleRender);
		cy.off('add remove data', scheduleRender);
		resizeObserver.disconnect();
		if (rafId !== null) cancelAnimationFrame(rafId);
		svg.remove();
	};
}
