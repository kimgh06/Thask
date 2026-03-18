import { api } from '$lib/api';
import { graphStore } from '$lib/stores/graph.svelte';
import { undoStack } from '$lib/stores/undo.svelte';
import { createEdgeCmd, deleteEdgeCmd } from '$lib/commands/node';
import type { GraphEdge, EdgeType } from '$lib/types';

interface EdgeMutationContext {
	projectId: string;
	getNodes: () => import('$lib/types').GraphNode[];
	setNodes: (nodes: import('$lib/types').GraphNode[]) => void;
	getEdges: () => GraphEdge[];
	setEdges: (edges: GraphEdge[]) => void;
}

export interface EdgeCrudDeps {
	getProjectId: () => string;
	getMutCtx: () => EdgeMutationContext;
	getEdges: () => GraphEdge[];
	setEdges: (edges: GraphEdge[]) => void;
	getSelectedEdge: () => GraphEdge | null;
	setSelectedEdge: (e: GraphEdge | null) => void;
}

export function createEdgeCrud(deps: EdgeCrudDeps) {
	async function handleCreateEdge(sourceId: string, targetId: string) {
		const duplicate = deps.getEdges().some(
			(e) => e.sourceId === sourceId && e.targetId === targetId && e.edgeType === 'related',
		);
		if (duplicate) return;
		await undoStack.run(createEdgeCmd(deps.getMutCtx(), { sourceId, targetId }));
	}

	async function handleEdgeTypeChange(edgeType: EdgeType) {
		const selectedEdge = deps.getSelectedEdge();
		if (!selectedEdge) return;
		const projectId = deps.getProjectId();
		const res = await api.patch<GraphEdge>(
			`/api/projects/${projectId}/edges/${selectedEdge.id}`,
			{ edgeType },
		);
		if (res.error) { console.error('Failed to update edge type:', res.error); return; }
		if (res.data) {
			deps.setEdges(deps.getEdges().map((e) => (e.id === res.data!.id ? res.data! : e)));
			deps.setSelectedEdge(res.data);
		}
	}

	async function handleEdgeLabelUpdate(label: string) {
		const selectedEdge = deps.getSelectedEdge();
		if (!selectedEdge) return;
		const projectId = deps.getProjectId();
		const res = await api.patch<GraphEdge>(
			`/api/projects/${projectId}/edges/${selectedEdge.id}`,
			{ label },
		);
		if (res.error) { console.error('Failed to update edge label:', res.error); return; }
		if (res.data) {
			deps.setEdges(deps.getEdges().map((e) => (e.id === res.data!.id ? res.data! : e)));
			deps.setSelectedEdge(res.data);
		}
	}

	async function handleDeleteEdge() {
		const selectedEdge = deps.getSelectedEdge();
		if (!selectedEdge) return;
		await undoStack.run(deleteEdgeCmd(deps.getMutCtx(), selectedEdge));
		graphStore.clearSelection();
	}

	return {
		handleCreateEdge,
		handleEdgeTypeChange,
		handleEdgeLabelUpdate,
		handleDeleteEdge,
	};
}
