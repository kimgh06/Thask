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
	}

	let { nodes, edges, projectId }: Props = $props();

	let container: HTMLDivElement;
	let cy: cytoscape.Core | null = $state(null);
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	let eh: any = null;
	let initialLayoutDone = false;

	function buildNodeData(node: GraphNode): Record<string, unknown> {
		const data: Record<string, unknown> = {
			id: node.id,
			label: node.title,
			title: node.title,
			nodeType: node.type,
			status: node.status,
			parentId: node.parentId || null,
		};
		if (node.type === 'GROUP') {
			const childCount = nodes.filter((n) => n.parentId === node.id).length;
			data.childCount = childCount;
			data.width = Math.max(node.width ?? 160, 160);
			data.height = Math.max(node.height ?? 100, 100);
		}
		return data;
	}

	function syncElements() {
		if (!cy) return;

		const hasPositions = nodes.some((n) => n.positionX !== 0 || n.positionY !== 0);

		cy.batch(() => {
			if (!cy) return;
			const newNodeIds = new Set(nodes.map((n) => n.id));
			const newEdgeIds = new Set(edges.map((e) => e.id));

			// Remove stale elements
			cy.nodes().forEach((n) => {
				if (!newNodeIds.has(n.id())) n.remove();
			});
			cy.edges().forEach((e) => {
				if (!newEdgeIds.has(e.id())) e.remove();
			});

			// Add or update nodes (parents first)
			const sorted = [...nodes].sort((a, b) => {
				if (!a.parentId && b.parentId) return -1;
				if (a.parentId && !b.parentId) return 1;
				return 0;
			});

			sorted.forEach((node) => {
				if (!cy) return;
				const existing = cy.getElementById(node.id);
				const data = buildNodeData(node);

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

	async function createEdge(sourceId: string, targetId: string) {
		await api.post(`/api/projects/${projectId}/edges`, {
			sourceId,
			targetId,
			edgeType: 'related',
		});
	}

	export function runLayout() {
		if (!cy || cy.nodes().length === 0) return;
		cy.layout(getFcoseLayout()).run();
		setTimeout(() => savePositions(), 1500);
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
		setTimeout(() => node.removeClass('search-highlight'), 2000);
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
		const overlay = portOverlay;
		let portSource: cytoscape.NodeSingular | null = null;
		let portHideTimer: ReturnType<typeof setTimeout> | null = null;

		function showPorts(node: cytoscape.NodeSingular) {
			if (!overlay) return;
			portSource = node;
			if (portHideTimer) { clearTimeout(portHideTimer); portHideTimer = null; }
			positionPorts(node);
			overlay.style.display = 'block';
		}

		function hidePorts(delay = 150) {
			portHideTimer = setTimeout(() => {
				if (overlay) overlay.style.display = 'none';
				portSource = null;
			}, delay);
		}

		function positionPorts(node: cytoscape.NodeSingular) {
			if (!overlay) return;
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
				const el = overlay.querySelector(`.${p.cls}`) as HTMLElement | null;
				if (el) {
					el.style.left = `${p.x}px`;
					el.style.top = `${p.y}px`;
				}
			}
		}

		if (overlay) {
			overlay.addEventListener('mousedown', (e: MouseEvent) => {
				if (!(e.target as HTMLElement).classList.contains('port-dot')) return;
				e.preventDefault();
				e.stopPropagation();
				if (portSource) {
					overlay.style.display = 'none';
					(eh as { start: (node: cytoscape.NodeSingular) => void }).start(portSource);
					const onMouseUp = () => {
						document.removeEventListener('mouseup', onMouseUp);
						(eh as { stop: () => void }).stop();
					};
					document.addEventListener('mouseup', onMouseUp);
				}
			});
			overlay.addEventListener('mouseenter', () => {
				if (portHideTimer) { clearTimeout(portHideTimer); portHideTimer = null; }
			});
			overlay.addEventListener('mouseleave', () => {
				if (!(eh as { active: boolean }).active) hidePorts(100);
			});
		}

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
			if (portSource && overlay?.style.display === 'block') positionPorts(portSource);
		});

		cy.on('ehstop ehcancel', () => {
			if (overlay) overlay.style.display = 'none';
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
			if (overlay) overlay.style.display = 'none';
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
			createEdge(
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

		let dragTimeout: ReturnType<typeof setTimeout>;
		cy.on('dragfree', 'node', (evt: cytoscape.EventObject) => {
			cy!.nodes('.drop-target').removeClass('drop-target');
			const node = evt.target as cytoscape.NodeSingular;

			if (groupDragState && groupDragState.groupId === node.id()) {
				groupDragState = null;
			}
			dragDescendantIds = null;

			clearTimeout(dragTimeout);
			dragTimeout = setTimeout(() => savePositions(), 500);
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
			resizeTarget.ungrabify();
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
				cy!.nodes('[nodeType="GROUP"]').forEach((g) => {
					if (foundZone) return;
					const zone = detectResizeZone(e, g);
					if (zone) {
						foundTarget = g;
						foundZone = zone;
					}
				});
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
		(eh as { destroy: () => void } | null)?.destroy();
		cy?.destroy();
		cy = null;
	});

	// React to data changes after mount
	$effect(() => {
		// Access reactive values to track them
		const _nodes = nodes;
		const _edges = edges;
		if (!cy || _nodes === undefined || _edges === undefined) return;
		syncElements();
	});

	let portOverlay: HTMLDivElement;
</script>

<div class="relative h-full w-full" style="min-height: 400px">
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
					width: 20px;
					height: 20px;
					border-radius: 50%;
					background: #3b82f6;
					border: 2px solid #1d4ed8;
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
