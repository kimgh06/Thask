import type { NodeType, NodeStatus } from '$lib/types';

class GraphStore {
	selectedNodeId = $state<string | null>(null);
	selectedEdgeId = $state<string | null>(null);
	typeFilter = $state<NodeType | null>(null);
	statusFilter = $state<NodeStatus | null>(null);
	impactMode = $state(false);
	collapsedGroups = $state<Set<string>>(new Set());
	searchQuery = $state('');

	selectNode(id: string | null) {
		this.selectedNodeId = id;
		this.selectedEdgeId = null;
	}

	selectEdge(id: string | null) {
		this.selectedEdgeId = id;
		this.selectedNodeId = null;
	}

	clearSelection() {
		this.selectedNodeId = null;
		this.selectedEdgeId = null;
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
