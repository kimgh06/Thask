import type { NodeType, NodeStatus } from '$lib/types';

class GraphStore {
	selectedNodeId = $state<string | null>(null);
	selectedEdgeId = $state<string | null>(null);
	selectedNodeIds = $state<Set<string>>(new Set());
	typeFilter = $state<NodeType | null>(null);
	statusFilter = $state<NodeStatus | null>(null);
	impactMode = $state(false);
	collapsedGroups = $state<Set<string>>(new Set());
	searchQuery = $state('');

	get isMultiSelect() {
		return this.selectedNodeIds.size > 1;
	}

	selectNode(id: string | null) {
		this.selectedNodeId = id;
		this.selectedEdgeId = null;
		this.selectedNodeIds = id ? new Set([id]) : new Set();
	}

	toggleNodeSelection(id: string) {
		const next = new Set(this.selectedNodeIds);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		this.selectedNodeIds = next;
		// Keep selectedNodeId in sync for detail panel (last toggled or first in set)
		this.selectedNodeId = next.size === 1 ? [...next][0] : next.size > 1 ? id : null;
		this.selectedEdgeId = null;
	}

	selectNodes(ids: string[]) {
		this.selectedNodeIds = new Set(ids);
		this.selectedNodeId = ids.length === 1 ? ids[0] : null;
		this.selectedEdgeId = null;
	}

	selectEdge(id: string | null) {
		this.selectedEdgeId = id;
		this.selectedNodeId = null;
		this.selectedNodeIds = new Set();
	}

	clearSelection() {
		this.selectedNodeId = null;
		this.selectedEdgeId = null;
		this.selectedNodeIds = new Set();
	}

	toggleImpactMode() {
		this.impactMode = !this.impactMode;
	}

	toggleCollapsed(groupId: string) {
		const next = new Set(this.collapsedGroups);
		if (next.has(groupId)) {
			next.delete(groupId);
		} else {
			next.add(groupId);
		}
		this.collapsedGroups = next;
	}

	setTypeFilter(type: NodeType | null) {
		this.typeFilter = type;
	}

	setStatusFilter(status: NodeStatus | null) {
		this.statusFilter = status;
	}

	setSearchQuery(query: string) {
		this.searchQuery = query;
	}
}

export const graphStore = new GraphStore();
