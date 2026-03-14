import type cytoscape from 'cytoscape';
import type { GraphNode, GraphEdge, NodeType, NodeStatus } from '$lib/types';
import { getFcoseLayout, getPresetLayout } from './layouts';

/** Compute nesting depth of a node via parentId chain. */
export function computeDepth(
	nodeId: string | null,
	nodeMap: Map<string, GraphNode>,
	visited = new Set<string>(),
): number {
	if (!nodeId) return 0;
	if (visited.has(nodeId)) return 0;
	visited.add(nodeId);
	const n = nodeMap.get(nodeId);
	if (!n || !n.parentId) return 0;
	return computeDepth(n.parentId, nodeMap, visited) + 1;
}

/** Build Cytoscape node data object from a GraphNode. */
export function buildNodeData(node: GraphNode, nodeList: GraphNode[]): Record<string, unknown> {
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
export function breakParentCycles(nodeList: GraphNode[]): { cleaned: GraphNode[]; brokenIds: string[] } {
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

export interface SyncContext {
	cy: cytoscape.Core;
	nodes: GraphNode[];
	edges: GraphEdge[];
	collapsedGroups: string[];
	typeFilter: NodeType | null;
	statusFilter: NodeStatus | null;
	initialLayoutDone: boolean;
	onUpdateNodeParent?: (nodeId: string, parentId: string | null) => void;
}

/**
 * Synchronize Cytoscape elements with the current nodes/edges data.
 * Returns whether the initial layout has been performed.
 */
export function syncElements(ctx: SyncContext): boolean {
	const { cy, nodes, edges, collapsedGroups } = ctx;
	let { initialLayoutDone } = ctx;

	// Auto-break circular parent references and fix DB
	const { cleaned: safeNodes, brokenIds } = breakParentCycles(nodes);
	if (brokenIds.length > 0) {
		console.warn('[Thask] Broke circular parentId for nodes:', brokenIds);
		for (const id of brokenIds) {
			ctx.onUpdateNodeParent?.(id, null);
		}
	}

	const hasPositions = safeNodes.some((n) => n.positionX !== 0 || n.positionY !== 0);

	cy.batch(() => {
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
			if (visited.has(n.id)) return 0;
			if (depthCache.has(n.id)) return depthCache.get(n.id)!;
			visited.add(n.id);
			const parent = nodeMap.get(n.parentId);
			const d = parent ? depthOf(parent, visited) + 1 : 0;
			depthCache.set(n.id, d);
			return d;
		};
		const sorted = [...safeNodes].sort((a, b) => depthOf(a) - depthOf(b));

		sorted.forEach((node) => {
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
			const existing = cy.getElementById(edge.id);
			const data: Record<string, unknown> = {
				id: edge.id,
				source: edge.sourceId,
				target: edge.targetId,
				label: edge.label ?? '',
				edgeType: edge.edgeType,
				sourceIsGroup: nodeMap.get(edge.sourceId)?.type === 'GROUP',
				targetIsGroup: nodeMap.get(edge.targetId)?.type === 'GROUP',
			};

			if (existing.length) {
				existing.data(data);
			} else {
				cy.add({ group: 'edges', data });
			}
		});
	});

	// Apply collapse state
	cy.nodes('[nodeType="GROUP"]').forEach((g) => {
		const childCount = (g.data('childCount') as number) ?? 0;
		const title = (g.data('title') as string) ?? '';
		if (collapsedGroups.includes(g.id())) {
			g.addClass('group-collapsed');
			g.data('label', childCount > 0 ? `${title} (${childCount})` : title);
		} else {
			g.removeClass('group-collapsed');
			g.data('label', title);
		}
	});
	cy.nodes().forEach((n) => {
		const parentId = n.data('parentId') as string | null;
		if (parentId && collapsedGroups.includes(parentId)) {
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

	// Apply type/status filters
	const { typeFilter, statusFilter } = ctx;
	cy.nodes().forEach((n) => {
		const nodeType = n.data('nodeType') as NodeType;
		const status = n.data('status') as NodeStatus;
		const hidden =
			(typeFilter !== null && nodeType !== typeFilter) ||
			(statusFilter !== null && status !== statusFilter);
		if (hidden) {
			n.addClass('filter-hidden');
		} else {
			n.removeClass('filter-hidden');
		}
	});
	cy.edges().forEach((e) => {
		if (e.source().hasClass('filter-hidden') || e.target().hasClass('filter-hidden')) {
			e.addClass('filter-hidden');
		} else {
			e.removeClass('filter-hidden');
		}
	});

	// Run layout on first load
	if (!initialLayoutDone && nodes.length > 0) {
		if (hasPositions) {
			cy.layout(getPresetLayout()).run();
		} else {
			cy.layout(getFcoseLayout()).run();
		}
		initialLayoutDone = true;
	}

	return initialLayoutDone;
}
