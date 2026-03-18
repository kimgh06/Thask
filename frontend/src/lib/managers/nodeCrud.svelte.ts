import { api } from '$lib/api';
import { graphStore } from '$lib/stores/graph.svelte';
import { undoStack } from '$lib/stores/undo.svelte';
import { createNodeCmd, deleteNodeCmd, updateNodeCmd, batchDeleteCmd } from '$lib/commands/node';
import type { GraphNode, GraphEdge, NodeType, NodeStatus, NodeDetail, NodeUpdateResult, StatusChange } from '$lib/types';

interface NodeMutationContext {
	projectId: string;
	getNodes: () => GraphNode[];
	setNodes: (nodes: GraphNode[]) => void;
	getEdges: () => GraphEdge[];
	setEdges: (edges: GraphEdge[]) => void;
}

export interface NodeCrudDeps {
	getProjectId: () => string;
	getMutCtx: () => NodeMutationContext;
	getNodes: () => GraphNode[];
	setNodes: (nodes: GraphNode[]) => void;
	getEdges: () => GraphEdge[];
	getSelectedNodeDetail: () => NodeDetail | null;
	setSelectedNodeDetail: (d: NodeDetail | null) => void;
	getCanvas: () => { getMousePosition: () => { x: number; y: number }; animateCascade: (changes: StatusChange[]) => void } | undefined;
}

export function createNodeCrud(deps: NodeCrudDeps) {
	async function handleAddNode() {
		const pos = deps.getCanvas()?.getMousePosition() ?? { x: 0, y: 0 };
		await undoStack.run(createNodeCmd(deps.getMutCtx(), { title: 'New Node', type: 'TASK', positionX: pos.x, positionY: pos.y }));
	}

	async function handleAddGroup() {
		const pos = deps.getCanvas()?.getMousePosition() ?? { x: 0, y: 0 };
		await undoStack.run(createNodeCmd(deps.getMutCtx(), { title: 'New Group', type: 'GROUP', positionX: pos.x, positionY: pos.y }));
	}

	async function handleUpdateNode(nodeId: string, data: Record<string, unknown>) {
		const nodes = deps.getNodes();
		const oldNode = nodes.find((n) => n.id === nodeId);
		const oldData: Record<string, unknown> = {};
		if (oldNode) {
			for (const key of Object.keys(data)) {
				oldData[key] = (oldNode as unknown as Record<string, unknown>)[key];
			}
		}

		const projectId = deps.getProjectId();
		const res = await api.patch<NodeUpdateResult>(`/api/projects/${projectId}/nodes/${nodeId}`, data);
		if (res.error) { console.error('Failed to update node:', res.error); return; }
		if (res.data) {
			const updated = res.data.node;
			const propagated = res.data.propagated ?? [];

			deps.setNodes(deps.getNodes().map((n) => (n.id === nodeId ? updated : n)));
			const detail = deps.getSelectedNodeDetail();
			if (detail?.id === nodeId) {
				deps.setSelectedNodeDetail({ ...detail, ...updated });
			}

			if (propagated.length > 0) {
				const changeMap = new Map(propagated.map((c) => [c.nodeId, c.newStatus]));
				deps.setNodes(deps.getNodes().map((n) => changeMap.has(n.id) ? { ...n, status: changeMap.get(n.id)! } : n));
				deps.getCanvas()?.animateCascade(propagated);
			}

			undoStack.record(updateNodeCmd(deps.getMutCtx(), nodeId, oldData, data));
		}
	}

	async function handleDeleteNode(nodeId: string) {
		const node = deps.getNodes().find((n) => n.id === nodeId);
		if (!node) return;
		const connectedEdges = deps.getEdges().filter((e) => e.sourceId === nodeId || e.targetId === nodeId);
		await undoStack.run(deleteNodeCmd(deps.getMutCtx(), node, connectedEdges));
		graphStore.clearSelection();
	}

	async function handleBatchDelete() {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length === 0) return;
		const idSet = new Set(ids);
		const deletedNodes = deps.getNodes().filter((n) => idSet.has(n.id));
		const deletedEdges = deps.getEdges().filter((e) => idSet.has(e.sourceId) || idSet.has(e.targetId));
		await undoStack.run(batchDeleteCmd(deps.getMutCtx(), deletedNodes, deletedEdges));
		graphStore.clearSelection();
	}

	async function handleBatchStatus(status: NodeStatus) {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length === 0) return;
		const nodes = deps.getNodes();
		const oldStatuses = new Map(
			nodes.filter((n) => ids.includes(n.id)).map((n) => [n.id, n.status]),
		);
		const projectId = deps.getProjectId();
		const res = await api.patch(`/api/projects/${projectId}/nodes/batch-status`, { ids, status });
		if (!res.error) {
			const idSet = new Set(ids);
			deps.setNodes(deps.getNodes().map((n) => idSet.has(n.id) ? { ...n, status } : n));
			undoStack.record({
				description: `Set ${ids.length} nodes to ${status}`,
				async execute() {
					const r = await api.patch(`/api/projects/${projectId}/nodes/batch-status`, { ids, status });
					if (!r.error) {
						deps.setNodes(deps.getNodes().map((n) => idSet.has(n.id) ? { ...n, status } : n));
					}
				},
				async undo() {
					for (const [id, oldStatus] of oldStatuses) {
						await api.patch(`/api/projects/${projectId}/nodes/${id}`, { status: oldStatus });
					}
					deps.setNodes(deps.getNodes().map((n) => oldStatuses.has(n.id) ? { ...n, status: oldStatuses.get(n.id)! } : n));
				},
			});
		}
	}

	async function handleUpdateNodeParent(nodeId: string, parentId: string | null) {
		const projectId = deps.getProjectId();
		const res = await api.patch<NodeUpdateResult>(`/api/projects/${projectId}/nodes/${nodeId}`, { parentId: parentId ?? '' });
		if (res.error) { console.error('Failed to update node parent:', res.error); return; }
		if (res.data) {
			deps.setNodes(deps.getNodes().map((n) => (n.id === nodeId ? res.data!.node : n)));
		}
	}

	return {
		handleAddNode,
		handleAddGroup,
		handleUpdateNode,
		handleDeleteNode,
		handleBatchDelete,
		handleBatchStatus,
		handleUpdateNodeParent,
	};
}
