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
		// Only update parentId locally — position stays as-is to prevent syncElements from resetting
		deps.setNodes(deps.getNodes().map((n) => (n.id === nodeId ? { ...n, parentId } : n)));
		const res = await api.patch<NodeUpdateResult>(`/api/projects/${projectId}/nodes/${nodeId}`, { parentId: parentId ?? '' });
		if (res.error) { console.error('Failed to update node parent:', res.error); }
	}

	async function handleBatchType(type: NodeType) {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length === 0) return;
		const projectId = deps.getProjectId();
		const idSet = new Set(ids);
		for (const id of ids) {
			const res = await api.patch(`/api/projects/${projectId}/nodes/${id}`, { type });
			if (res.error) { console.error('Failed to update node type:', res.error); return; }
		}
		deps.setNodes(deps.getNodes().map((n) => idSet.has(n.id) ? { ...n, type } : n));
	}

	async function handleBatchAddTag(tag: string) {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length === 0 || !tag) return;
		const projectId = deps.getProjectId();
		const idSet = new Set(ids);
		const currentNodes = deps.getNodes();
		for (const id of ids) {
			const node = currentNodes.find((n) => n.id === id);
			if (!node || node.tags.includes(tag)) continue;
			const res = await api.patch(`/api/projects/${projectId}/nodes/${id}`, { tags: [...node.tags, tag] });
			if (res.error) { console.error('Failed to add tag:', res.error); return; }
		}
		deps.setNodes(deps.getNodes().map((n) =>
			idSet.has(n.id) && !n.tags.includes(tag) ? { ...n, tags: [...n.tags, tag] } : n,
		));
	}

	async function handleCreateGroupFromSelection() {
		const ids = [...graphStore.selectedNodeIds];
		if (ids.length < 2) return;
		const projectId = deps.getProjectId();
		const selected = deps.getNodes().filter((n) => ids.includes(n.id));
		const xs = selected.map((n) => n.positionX);
		const ys = selected.map((n) => n.positionY);
		const padding = 60;
		const minX = Math.min(...xs) - padding;
		const minY = Math.min(...ys) - padding;
		const maxX = Math.max(...xs) + padding;
		const maxY = Math.max(...ys) + padding;
		const cx = (minX + maxX) / 2;
		const cy = (minY + maxY) / 2;
		const w = maxX - minX;
		const h = maxY - minY;

		// Create group via direct API call
		const createRes = await api.post<GraphNode>(`/api/projects/${projectId}/nodes`, {
			title: 'New Group', type: 'GROUP', status: 'IN_PROGRESS', positionX: cx, positionY: cy,
		});
		if (createRes.error || !createRes.data) { console.error('Failed to create group:', createRes.error); return; }
		const group = createRes.data;
		deps.setNodes([...deps.getNodes(), group]);

		// Set dimensions
		await api.patch(`/api/projects/${projectId}/nodes/${group.id}`, { width: w, height: h });
		deps.setNodes(deps.getNodes().map((n) =>
			n.id === group.id ? { ...n, width: w, height: h } : n,
		));

		// Reparent selected nodes (capture old parents for undo)
		const oldParents = new Map(selected.map((n) => [n.id, n.parentId ?? null]));
		for (const id of ids) {
			await api.patch(`/api/projects/${projectId}/nodes/${id}`, { parentId: group.id });
		}
		const idSet = new Set(ids);
		deps.setNodes(deps.getNodes().map((n) =>
			idSet.has(n.id) ? { ...n, parentId: group.id } : n,
		));

		// Record compound undo for the full operation
		undoStack.record({
			description: `Create group from ${ids.length} nodes`,
			async execute() {
				const r = await api.post<GraphNode>(`/api/projects/${projectId}/nodes`, {
					title: 'New Group', type: 'GROUP', status: 'IN_PROGRESS', positionX: cx, positionY: cy,
				});
				if (!r.data) return;
				const newGroup = r.data;
				deps.setNodes([...deps.getNodes(), newGroup]);
				await api.patch(`/api/projects/${projectId}/nodes/${newGroup.id}`, { width: w, height: h });
				deps.setNodes(deps.getNodes().map((n) => n.id === newGroup.id ? { ...n, width: w, height: h } : n));
				for (const id of ids) {
					await api.patch(`/api/projects/${projectId}/nodes/${id}`, { parentId: newGroup.id });
				}
				deps.setNodes(deps.getNodes().map((n) => idSet.has(n.id) ? { ...n, parentId: newGroup.id } : n));
			},
			async undo() {
				for (const [id, oldParent] of oldParents) {
					await api.patch(`/api/projects/${projectId}/nodes/${id}`, { parentId: oldParent ?? '' });
				}
				deps.setNodes(deps.getNodes().map((n) =>
					oldParents.has(n.id) ? { ...n, parentId: oldParents.get(n.id) ?? null } : n,
				));
				await api.delete(`/api/projects/${projectId}/nodes/${group.id}`);
				deps.setNodes(deps.getNodes().filter((n) => n.id !== group.id));
			},
		});

		graphStore.selectNode(group.id);
	}

	return {
		handleAddNode,
		handleAddGroup,
		handleUpdateNode,
		handleDeleteNode,
		handleBatchDelete,
		handleBatchStatus,
		handleBatchType,
		handleBatchAddTag,
		handleCreateGroupFromSelection,
		handleUpdateNodeParent,
	};
}
