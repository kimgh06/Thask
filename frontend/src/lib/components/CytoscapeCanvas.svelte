<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import cytoscape from 'cytoscape';
	import fcose from 'cytoscape-fcose';
	import edgehandles from 'cytoscape-edgehandles';
	import { getGraphStyles } from '$lib/cytoscape/styles';
	import { getFcoseLayout, getPresetLayout } from '$lib/cytoscape/layouts';
	import { getChildNodes, getDescendantNodes, getDescendantIdSet } from '$lib/cytoscape/groupHelpers';
	import { graphStore } from '$lib/stores/graph.svelte';
	import { api } from '$lib/api';
	import type { GraphNode, GraphEdge } from '$lib/types';

	// Register extensions once at module level
	let extensionsRegistered = false;
	if (!extensionsRegistered) {
		cytoscape.use(fcose);
		cytoscape.use(edgehandles);
		extensionsRegistered = true;
	}

	interface Props {
		nodes: GraphNode[];
		edges: GraphEdge[];
		projectId: string;
		onUpdateNodeParent?: (nodeId: string, parentId: string | null) => void;
		onZoomChange?: (zoom: number) => void;
		onCreateEdge?: (sourceId: string, targetId: string) => void;
	}

	let { nodes, edges, projectId, onUpdateNodeParent, onZoomChange, onCreateEdge }: Props = $props();

	let container: HTMLDivElement;
	let cy: cytoscape.Core | null = $state(null);
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let eh: any = null;
	let initialLayoutDone = false;
	let activeTimeouts: ReturnType<typeof setTimeout>[] = [];

	function trackTimeout(fn: () => void, ms: number): ReturnType<typeof setTimeout> {
		const id = setTimeout(() => {
			activeTimeouts = activeTimeouts.filter((t) => t !== id);
			fn();
		}, ms);
		activeTimeouts.push(id);
		return id;
	}

	function computeDepth(nodeId: string | null, nodeMap: Map<string, GraphNode>, visited = new Set<string>()): number {
		if (!nodeId) return 0;
		if (visited.has(nodeId)) return 0;
		visited.add(nodeId);
		const n = nodeMap.get(nodeId);
		if (!n || !n.parentId) return 0;
		return computeDepth(n.parentId, nodeMap, visited) + 1;
	}

	function buildNodeData(node: GraphNode, nodeList: GraphNode[]): Record<string, unknown> {
		const nodeMap = new Map(nodeList.map((n) => [n.id, n]));
		const depth = computeDepth(node.id, nodeMap);
		const data: Record<string, unknown> = {
			id: node.id,
			label: node.title,
			title: node.title,
			nodeType: node.type,
			status: node.status,
			parentId: node.parentId || null,
			depth,
		};
		if (node.type === 'GROUP') {
			const childCount = nodeList.filter((n) => n.parentId === node.id).length;
			data.childCount = childCount;
			data.width = Math.max(node.width ?? 160, 160);
			data.height = Math.max(node.height ?? 100, 100);
		}
		return data;
	}

	/** Detect and break circular parentId references. Returns cleaned list + IDs that were fixed. */
	function breakParentCycles(nodeList: GraphNode[]): { cleaned: GraphNode[]; brokenIds: string[] } {
		const nodeMap = new Map(nodeList.map((n) => [n.id, n]));
		const brokenIds: string[] = [];
		const cleaned = nodeList.map((n) => {
			if (!n.parentId) return n;
			const visited = new Set<string>();
			let cur: string | null = n.parentId;
			while (cur) {
				if (cur === n.id) {
					brokenIds.push(n.id);
					return { ...n, parentId: null };
				}
				if (visited.has(cur)) break;
				visited.add(cur);
				const parent = nodeMap.get(cur);
				cur = parent?.parentId ?? null;
			}
			return n;
		});
		return { cleaned, brokenIds };
	}

	function syncElements() {
		if (!cy) return;

		// Auto-break circular parent references and fix DB
		const { cleaned: safeNodes, brokenIds } = breakParentCycles(nodes);
		if (brokenIds.length > 0) {
			console.warn('[Thask] Broke circular parentId for nodes:', brokenIds);
			for (const id of brokenIds) {
				onUpdateNodeParent?.(id, null);
			}
		}

		const hasPositions = safeNodes.some((n) => n.positionX !== 0 || n.positionY !== 0);

		cy.batch(() => {
			if (!cy) return;
			const newNodeIds = new Set(safeNodes.map((n) => n.id));
			const newEdgeIds = new Set(edges.map((e) => e.id));

			// Remove stale elements
			cy.nodes().forEach((n) => {
				if (!newNodeIds.has(n.id())) n.remove();
			});
			cy.edges().forEach((e) => {
				if (!newEdgeIds.has(e.id())) e.remove();
			});

			// Add or update nodes (parents first, depth-based for nested groups)
			const nodeMap = new Map(safeNodes.map((n) => [n.id, n]));
			const depthCache = new Map<string, number>();
			const depthOf = (n: GraphNode, visited = new Set<string>()): number => {
				if (!n.parentId) return 0;
				if (visited.has(n.id)) return 0; // cycle detected
				if (depthCache.has(n.id)) return depthCache.get(n.id)!;
				visited.add(n.id);
				const parent = nodeMap.get(n.parentId);
				const d = parent ? depthOf(parent, visited) + 1 : 0;
				depthCache.set(n.id, d);
				return d;
			};
			const sorted = [...safeNodes].sort((a, b) => depthOf(a) - depthOf(b));

			sorted.forEach((node) => {
				if (!cy) return;
				const existing = cy.getElementById(node.id);
				const data = buildNodeData(node, safeNodes);

				if (existing.length) {
					existing.data(data);
				} else {
					cy.add({
						group: 'nodes',
						data: { ...data },
						position: hasPositions ? { x: node.positionX, y: node.positionY } : undefined,
					});
				}
			});

			// Add or update edges
			edges.forEach((edge) => {
				if (!cy) return;
				const existing = cy.getElementById(edge.id);
				const data = {
					id: edge.id,
					source: edge.sourceId,
					target: edge.targetId,
					label: edge.label ?? '',
					edgeType: edge.edgeType,
				};

				if (existing.length) {
					existing.data(data);
				} else {
					cy.add({ group: 'edges', data });
				}
			});
		});

		// Apply collapse state
		const collapsedGroupIds = [...graphStore.collapsedGroups];
		cy.nodes('[nodeType="GROUP"]').forEach((g) => {
			const childCount = (g.data('childCount') as number) ?? 0;
			const title = (g.data('title') as string) ?? '';
			if (collapsedGroupIds.includes(g.id())) {
				g.addClass('group-collapsed');
				g.data('label', childCount > 0 ? `${title} (${childCount})` : title);
			} else {
				g.removeClass('group-collapsed');
				g.data('label', title);
			}
		});
		cy.nodes().forEach((n) => {
			const parentId = n.data('parentId') as string | null;
			if (parentId && collapsedGroupIds.includes(parentId)) {
				n.addClass('collapsed-child');
			} else {
				n.removeClass('collapsed-child');
			}
		});
		cy.edges().forEach((e) => {
			if (e.source().hasClass('collapsed-child') || e.target().hasClass('collapsed-child')) {
				e.addClass('collapsed-edge');
			} else {
				e.removeClass('collapsed-edge');
			}
		});

		// Run layout on first load
		if (!initialLayoutDone && nodes.length > 0) {
			if (hasPositions) {
				cy.layout(getPresetLayout()).run();
			} else {
				const layout = cy.layout(getFcoseLayout());
				layout.on('layoutstop', () => {
					if (!cy) return;
					cy.nodes('[nodeType="GROUP"]').forEach((g) => {
						if (g.hasClass('group-collapsed')) return;
						const children = cy!.nodes().filter(
							(n: cytoscape.NodeSingular) => n.data('parentId') === g.id(),
						);
						if (children.length === 0) return;
						const PAD = 40;
						const NODE_HALF = 40;
						let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
						children.forEach((c: cytoscape.NodeSingular) => {
							const pos = c.position();
							minX = Math.min(minX, pos.x - NODE_HALF);
							maxX = Math.max(maxX, pos.x + NODE_HALF);
							minY = Math.min(minY, pos.y - NODE_HALF);
							maxY = Math.max(maxY, pos.y + NODE_HALF);
						});
						const w = Math.max(160, maxX - minX + PAD * 2);
						const h = Math.max(100, maxY - minY + PAD * 2);
						g.data('width', w);
						g.data('height', h);
						g.position({ x: (minX + maxX) / 2, y: (minY + maxY) / 2 });
					});
				});
				layout.run();
			}
			initialLayoutDone = true;
		}
	}

	async function savePositions() {
		if (!cy) return;
		const positions: Array<{ id: string; x: number; y: number; width?: number; height?: number }> =
			[];
		cy.nodes().forEach((n: cytoscape.NodeSingular) => {
			const pos = n.position();
			const entry: { id: string; x: number; y: number; width?: number; height?: number } = {
				id: n.id(),
				x: pos.x,
				y: pos.y,
			};
			const w = n.data('width') as number | undefined;
			const h = n.data('height') as number | undefined;
			if (w !== undefined) entry.width = w;
			if (h !== undefined) entry.height = h;
			positions.push(entry);
		});
		await api.patch(`/api/projects/${projectId}/nodes/positions`, { positions });
	}

	export function runLayout() {
		if (!cy || cy.nodes().length === 0) return;
		cy.layout(getFcoseLayout()).run();
		trackTimeout(() => savePositions(), 1500);
	}

	export function fitView() {
		cy?.fit(undefined, 50);
	}

	export function zoomIn() {
		if (!cy) return;
		cy.zoom({ level: cy.zoom() * 1.2, renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } });
	}

	export function zoomOut() {
		if (!cy) return;
		cy.zoom({ level: cy.zoom() / 1.2, renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } });
	}

	export function focusNode(nodeId: string) {
		if (!cy) return;
		const node = cy.getElementById(nodeId);
		if (!node.length) return;
		cy.animate({ center: { eles: node }, zoom: 1.5 }, { duration: 400 });
		cy.nodes().removeClass('search-highlight');
		node.addClass('search-highlight');
		trackTimeout(() => { if (node.inside()) node.removeClass('search-highlight'); }, 2000);
	}

	export function getCy(): cytoscape.Core | null {
		return cy;
	}

	onMount(() => {
		cy = cytoscape({
			container,
			style: getGraphStyles(),
			layout: { name: 'preset' },
			minZoom: 0.2,
			maxZoom: 4,
			wheelSensitivity: 0.3,
		});

		// Initialize edgehandles
		eh = (cy as cytoscape.Core & { edgehandles: (opts: unknown) => unknown }).edgehandles({
			canConnect: (sourceNode: cytoscape.NodeSingular, targetNode: cytoscape.NodeSingular) =>
				!sourceNode.same(targetNode),
			edgeParams: () => ({}),
			hoverDelay: 0,
			snap: true,
			snapThreshold: 50,
			snapFrequency: 15,
			noEdgeEventsInDraw: true,
			disableBrowserGestures: true,
			handleNodes: 'DONOTMATCHANYTHING',
		});
		(eh as { enable: () => void }).enable();

		// Port overlay for edge creation
		const PORT_SIZE = 20;
		let portSource: cytoscape.NodeSingular | null = null;
		let portHideTimer: ReturnType<typeof setTimeout> | null = null;

		function showPorts(node: cytoscape.NodeSingular) {
			if (!portOverlay) return;
			portSource = node;
			if (portHideTimer) { clearTimeout(portHideTimer); portHideTimer = null; }
			positionPorts(node);
			portOverlay.style.display = 'block';
		}

		function hidePorts(delay = 150) {
			portHideTimer = setTimeout(() => {
				if (portOverlay) portOverlay.style.display = 'none';
				portSource = null;
			}, delay);
		}

		function positionPorts(node: cytoscape.NodeSingular) {
			if (!portOverlay) return;
			const rbb = node.renderedBoundingBox({ includeLabels: false, includeOverlays: false });
			const cxR = (rbb.x1 + rbb.x2) / 2;
			const cyR = (rbb.y1 + rbb.y2) / 2;
			const half = PORT_SIZE / 2;

			const positions = [
				{ cls: 'port-top', x: cxR - half, y: rbb.y1 - PORT_SIZE },
				{ cls: 'port-right', x: rbb.x2, y: cyR - half },
				{ cls: 'port-bottom', x: cxR - half, y: rbb.y2 },
				{ cls: 'port-left', x: rbb.x1 - PORT_SIZE, y: cyR - half },
			];

			for (const p of positions) {
				const el = portOverlay.querySelector(`.${p.cls}`) as HTMLElement | null;
				if (el) {
					el.style.left = `${p.x}px`;
					el.style.top = `${p.y}px`;
				}
			}
		}

		portOverlay.addEventListener('mousedown', (e: MouseEvent) => {
			if (!(e.target as HTMLElement).classList.contains('port-dot')) return;
			e.preventDefault();
			e.stopPropagation();
			if (portSource) {
				portOverlay.style.display = 'none';
				(eh as { start: (node: cytoscape.NodeSingular) => void }).start(portSource);
				const onMouseUp = () => {
					document.removeEventListener('mouseup', onMouseUp);
					(eh as { stop: () => void }).stop();
				};
				document.addEventListener('mouseup', onMouseUp);
			}
		});
		portOverlay.addEventListener('mouseenter', () => {
			if (portHideTimer) { clearTimeout(portHideTimer); portHideTimer = null; }
		});
		portOverlay.addEventListener('mouseleave', () => {
			if (!(eh as { active: boolean }).active) hidePorts(100);
		});

		cy.on('mouseover', 'node', (e) => {
			const node = e.target as cytoscape.NodeSingular;
			if ((eh as { active: boolean }).active || node.grabbed() || isResizing) return;
			if (node.data('nodeType') === 'GROUP' && !node.hasClass('group-collapsed')) return;
			showPorts(node);
		});

		cy.on('mouseout', 'node', () => {
			if (!(eh as { active: boolean }).active) hidePorts();
		});

		cy.on('pan zoom', () => {
			if (portSource && portOverlay?.style.display === 'block') positionPorts(portSource);
			onZoomChange?.(cy!.zoom());
		});

		cy.on('ehstop ehcancel', () => {
			if (portOverlay) portOverlay.style.display = 'none';
			portSource = null;
		});

		// GROUP drag state
		let groupDragState: {
			groupId: string;
			childOffsets: Map<string, { dx: number; dy: number }>;
		} | null = null;
		let dragDescendantIds: Set<string> | null = null;

		cy.on('grab', 'node', (e) => {
			const node = e.target as cytoscape.NodeSingular;
			if (isResizing) return;
			if (portOverlay) portOverlay.style.display = 'none';
			portSource = null;

			if (node.data('nodeType') === 'GROUP') {
				const groupPos = node.position();
				const descendants = getDescendantNodes(cy!, node.id());
				const childOffsets = new Map<string, { dx: number; dy: number }>();
				descendants.forEach((d: cytoscape.NodeSingular) => {
					const dPos = d.position();
					childOffsets.set(d.id(), {
						dx: dPos.x - groupPos.x,
						dy: dPos.y - groupPos.y,
					});
				});
				groupDragState = { groupId: node.id(), childOffsets };
				dragDescendantIds = getDescendantIdSet(cy!, node.id());
			} else {
				groupDragState = null;
				dragDescendantIds = null;
			}
		});

		cy.on('ehcomplete', (_event, sourceNode, targetNode, addedEdge) => {
			(addedEdge as cytoscape.EdgeSingular).remove();
			onCreateEdge?.(
				(sourceNode as cytoscape.NodeSingular).id(),
				(targetNode as cytoscape.NodeSingular).id(),
			);
		});

		cy.on('tap', 'node', (evt: cytoscape.EventObject) => {
			graphStore.selectNode(evt.target.id());
		});

		cy.on('tap', (evt: cytoscape.EventObject) => {
			if (evt.target === cy) {
				graphStore.clearSelection();
			}
		});

		cy.on('tap', 'edge', (evt: cytoscape.EventObject) => {
			graphStore.selectEdge(evt.target.id());
		});

		cy.on('dbltap', 'node[nodeType="GROUP"]', (evt: cytoscape.EventObject) => {
			graphStore.toggleCollapsed(evt.target.id());
		});

		let currentDropTarget: string | null = null;

		cy.on('drag', 'node', (evt: cytoscape.EventObject) => {
			const node = evt.target as cytoscape.NodeSingular;
			const cursorPos = evt.position;

			let innerTarget: string | null = null;
			let innerTargetArea = Infinity;
			cy!.nodes('[nodeType="GROUP"]').forEach((g) => {
				g.removeClass('drop-target');
				if (g.id() === node.id() || g.hasClass('group-collapsed') || dragDescendantIds?.has(g.id())) return;
				const bb = g.boundingBox({});
				if (
					cursorPos.x >= bb.x1 && cursorPos.x <= bb.x2 &&
					cursorPos.y >= bb.y1 && cursorPos.y <= bb.y2
				) {
					const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
					if (area < innerTargetArea) {
						innerTarget = g.id();
						innerTargetArea = area;
					}
				}
			});
			currentDropTarget = innerTarget;
			if (innerTarget) {
				cy!.getElementById(innerTarget).addClass('drop-target');
			}

			if (groupDragState && groupDragState.groupId === node.id()) {
				const groupPos = node.position();
				groupDragState.childOffsets.forEach((offset, childId) => {
					const child = cy!.getElementById(childId);
					if (child.length) {
						child.position({ x: groupPos.x + offset.dx, y: groupPos.y + offset.dy });
					}
				});
			}
		});

		let dragTimeout: ReturnType<typeof setTimeout> | null = null;
		cy.on('dragfree', 'node', (evt: cytoscape.EventObject) => {
			cy!.nodes('.drop-target').removeClass('drop-target');
			const node = evt.target as cytoscape.NodeSingular;

			// Drop on group: update parentId (with cycle prevention)
			const oldParentId = (node.data('parentId') as string | null) ?? null;
			let newParentId = currentDropTarget;

			// Prevent cycles: ensure the drop target is not a descendant of the dragged node
			if (newParentId && node.data('nodeType') === 'GROUP') {
				const descendants = getDescendantIdSet(cy!, node.id());
				if (descendants.has(newParentId) || newParentId === node.id()) {
					newParentId = null; // reject drop — would create cycle
				}
			}

			if (newParentId !== oldParentId) {
				node.data('parentId', newParentId);
				onUpdateNodeParent?.(node.id(), newParentId);
			}
			currentDropTarget = null;

			if (groupDragState && groupDragState.groupId === node.id()) {
				groupDragState = null;
			}
			dragDescendantIds = null;

			if (dragTimeout) clearTimeout(dragTimeout);
			dragTimeout = trackTimeout(() => savePositions(), 500);
		});

		// 8-directional resize for GROUP nodes
		type ResizeZone = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw';
		const ZONE_CURSORS: Record<ResizeZone, string> = {
			nw: 'nwse-resize', se: 'nwse-resize',
			ne: 'nesw-resize', sw: 'nesw-resize',
			n: 'ns-resize', s: 'ns-resize',
			e: 'ew-resize', w: 'ew-resize',
		};

		let resizeTarget: cytoscape.NodeSingular | null = null;
		let ungrabifiedForResize: cytoscape.NodeSingular | null = null;
		let isResizing = false;
		let resizeZone: ResizeZone | null = null;
		let resizeStartPos = { x: 0, y: 0 };
		let resizeStartBB = { x1: 0, y1: 0, x2: 0, y2: 0 };

		function detectResizeZone(e: MouseEvent, node: cytoscape.NodeSingular): ResizeZone | null {
			if (node.hasClass('group-collapsed') || node.hasClass('filter-hidden')) return null;
			const bb = node.renderedBoundingBox({});
			const mx = e.offsetX;
			const my = e.offsetY;
			const T = 10;

			const inX = mx >= bb.x1 - T && mx <= bb.x2 + T;
			const inY = my >= bb.y1 - T && my <= bb.y2 + T;
			if (!inX || !inY) return null;

			const nearTop = Math.abs(my - bb.y1) <= T;
			const nearBottom = Math.abs(my - bb.y2) <= T;
			const nearLeft = Math.abs(mx - bb.x1) <= T;
			const nearRight = Math.abs(mx - bb.x2) <= T;

			if (nearTop && nearLeft) return 'nw';
			if (nearTop && nearRight) return 'ne';
			if (nearBottom && nearLeft) return 'sw';
			if (nearBottom && nearRight) return 'se';
			if (nearTop) return 'n';
			if (nearBottom) return 's';
			if (nearLeft) return 'w';
			if (nearRight) return 'e';
			return null;
		}

		const cyContainer = cy.container()!;

		function onResizeMouseDown(e: MouseEvent) {
			if (!resizeTarget || (eh as { active: boolean }).active) return;
			const zone = detectResizeZone(e, resizeTarget);
			if (!zone) return;

			e.preventDefault();
			e.stopPropagation();
			isResizing = true;
			resizeZone = zone;
			cyContainer.style.cursor = ZONE_CURSORS[zone];
			cy!.panningEnabled(false);
			cy!.boxSelectionEnabled(false);
			// Node is already ungrabified from hover detection
			resizeTarget.addClass('resizing');

			const bb = resizeTarget.boundingBox({ includeLabels: false, includeOverlays: false });
			resizeStartBB = { x1: bb.x1, y1: bb.y1, x2: bb.x2, y2: bb.y2 };
			resizeStartPos = { x: e.clientX, y: e.clientY };
		}

		function onResizeMouseMove(e: MouseEvent) {
			if (!isResizing) {
				if ((eh as { active: boolean }).active) return;
				let foundTarget: cytoscape.NodeSingular | null = null;
				let foundZone: ResizeZone | null = null;
				let foundArea = Infinity;
				cy!.nodes('[nodeType="GROUP"]').forEach((g) => {
					const zone = detectResizeZone(e, g);
					if (zone) {
						// Prefer innermost (smallest area) group when zones overlap
						const bb = g.boundingBox({});
						const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
						if (area < foundArea) {
							foundTarget = g;
							foundZone = zone;
							foundArea = area;
						}
					}
				});

				// Preemptively ungrabify when entering resize zone, restore when leaving
				if (foundTarget && foundZone) {
					const ft = foundTarget as cytoscape.NodeSingular;
					if (ungrabifiedForResize && ungrabifiedForResize.id() !== ft.id()) {
						ungrabifiedForResize.grabify();
						ungrabifiedForResize = null;
					}
					if (!ungrabifiedForResize) {
						ft.ungrabify();
						ungrabifiedForResize = ft;
					}
				} else if (ungrabifiedForResize) {
					ungrabifiedForResize.grabify();
					ungrabifiedForResize = null;
				}

				resizeTarget = foundTarget;
				cyContainer.style.cursor = foundZone ? ZONE_CURSORS[foundZone] : '';
				return;
			}
			if (!resizeTarget || !resizeZone) return;

			const zoom = cy!.zoom();
			const deltaX = (e.clientX - resizeStartPos.x) / zoom;
			const deltaY = (e.clientY - resizeStartPos.y) / zoom;

			const moveLeft = resizeZone.includes('w');
			const moveRight = resizeZone.includes('e');
			const moveTop = resizeZone.includes('n');
			const moveBottom = resizeZone.includes('s');

			let { x1, y1, x2, y2 } = resizeStartBB;
			if (moveLeft) x1 += deltaX;
			if (moveRight) x2 += deltaX;
			if (moveTop) y1 += deltaY;
			if (moveBottom) y2 += deltaY;

			let minW = 80;
			let minH = 50;
			if (resizeTarget.data('nodeType') === 'GROUP' && !resizeTarget.hasClass('group-collapsed')) {
				const children = getChildNodes(cy!, resizeTarget.id());
				if (children.length > 0) {
					const PAD = 30;
					let cxMin = Infinity, cyMin = Infinity, cxMax = -Infinity, cyMax = -Infinity;
					children.forEach((c: cytoscape.NodeSingular) => {
						const pos = c.position();
						const hw = c.width() / 2;
						const hh = c.height() / 2;
						cxMin = Math.min(cxMin, pos.x - hw);
						cyMin = Math.min(cyMin, pos.y - hh);
						cxMax = Math.max(cxMax, pos.x + hw);
						cyMax = Math.max(cyMax, pos.y + hh);
					});
					minW = Math.max(minW, cxMax - cxMin + PAD * 2);
					minH = Math.max(minH, cyMax - cyMin + PAD * 2);
				}
			}
			if (x2 - x1 < minW) {
				if (moveLeft) x1 = x2 - minW;
				else x2 = x1 + minW;
			}
			if (y2 - y1 < minH) {
				if (moveTop) y1 = y2 - minH;
				else y2 = y1 + minH;
			}

			const newW = x2 - x1;
			const newH = y2 - y1;
			const newCenterX = (x1 + x2) / 2;
			const newCenterY = (y1 + y2) / 2;

			resizeTarget.data('width', newW);
			resizeTarget.data('height', newH);
			resizeTarget.position({ x: newCenterX, y: newCenterY });
		}

		function onResizeEnd() {
			if (!isResizing || !resizeTarget) return;
			isResizing = false;
			resizeZone = null;
			cyContainer.style.cursor = '';
			cy!.panningEnabled(true);
			cy!.boxSelectionEnabled(true);
			resizeTarget.grabify();
			resizeTarget.removeClass('resizing');
			ungrabifiedForResize = null;
			savePositions();
		}

		cyContainer.addEventListener('mousedown', onResizeMouseDown);
		cyContainer.addEventListener('mousemove', onResizeMouseMove);
		cyContainer.addEventListener('mouseup', onResizeEnd);
		cyContainer.addEventListener('mouseleave', onResizeEnd);

		// Initial data load
		syncElements();

		return () => {
			if (portHideTimer) clearTimeout(portHideTimer);
			cyContainer.removeEventListener('mousedown', onResizeMouseDown);
			cyContainer.removeEventListener('mousemove', onResizeMouseMove);
			cyContainer.removeEventListener('mouseup', onResizeEnd);
			cyContainer.removeEventListener('mouseleave', onResizeEnd);
		};
	});

	onDestroy(() => {
		activeTimeouts.forEach(clearTimeout);
		activeTimeouts = [];
		(eh as { destroy: () => void } | null)?.destroy();
		cy?.destroy();
		cy = null;
	});

	// React to data changes after mount
	$effect(() => {
		// Access reactive values to track them explicitly
		const _nodes = nodes;
		const _edges = edges;
		const _collapsed = graphStore.collapsedGroups;
		if (!cy || _nodes === undefined || _edges === undefined) return;
		void _collapsed;
		syncElements();
	});

	let portOverlay: HTMLDivElement;
</script>

<div class="relative h-full w-full cytoscape-canvas-bg" style="min-height: 400px">
	<div bind:this={container} class="h-full w-full"></div>
	<!-- Port overlay for edge creation — 4 dots around hovered node -->
	<div
		bind:this={portOverlay}
		style="display: none; position: absolute; top: 0; left: 0; pointer-events: none;"
	>
		{#each ['port-top', 'port-right', 'port-bottom', 'port-left'] as cls}
			<div
				class="port-dot {cls}"
				style="
					position: absolute;
					width: 18px;
					height: 18px;
					border-radius: 50%;
					background: #818cf8;
					border: 2px solid #6366f1;
					cursor: crosshair;
					pointer-events: auto;
					transition: transform 0.1s ease;
				"
				role="button"
				tabindex="-1"
				aria-label="Create edge from this port"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.transform = 'scale(1.3)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.transform = 'scale(1)'; }}
			></div>
		{/each}
	</div>
</div>
