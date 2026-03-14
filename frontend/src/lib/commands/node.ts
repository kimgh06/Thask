import { api } from '$lib/api';
import type { Command } from '$lib/stores/undo.svelte';
import type { GraphNode, GraphEdge, NodeType, NodeStatus } from '$lib/types';

interface NodeMutationContext {
	projectId: string;
	getNodes: () => GraphNode[];
	setNodes: (nodes: GraphNode[]) => void;
	getEdges: () => GraphEdge[];
	setEdges: (edges: GraphEdge[]) => void;
}

export function createNodeCmd(
	ctx: NodeMutationContext,
	data: { title: string; type: NodeType; status?: NodeStatus },
): Command {
	let createdNode: GraphNode | null = null;

	return {
		description: `Create node "${data.title}"`,
		async execute() {
			const res = await api.post<GraphNode>(`/api/projects/${ctx.projectId}/nodes`, {
				title: data.title,
				type: data.type,
				status: data.status ?? 'IN_PROGRESS',
			});
			if (res.data) {
				createdNode = res.data;
				ctx.setNodes([...ctx.getNodes(), res.data]);
			}
		},
		async undo() {
			if (!createdNode) return;
			const res = await api.delete(`/api/projects/${ctx.projectId}/nodes/${createdNode.id}`);
			if (!res.error) {
				ctx.setNodes(ctx.getNodes().filter((n) => n.id !== createdNode!.id));
			}
		},
	};
}

export function deleteNodeCmd(
	ctx: NodeMutationContext,
	node: GraphNode,
	connectedEdges: GraphEdge[],
): Command {
	return {
		description: `Delete node "${node.title}"`,
		async execute() {
			const res = await api.delete(`/api/projects/${ctx.projectId}/nodes/${node.id}`);
			if (!res.error) {
				ctx.setNodes(ctx.getNodes().filter((n) => n.id !== node.id));
				ctx.setEdges(ctx.getEdges().filter((e) => e.sourceId !== node.id && e.targetId !== node.id));
			}
		},
		async undo() {
			// Recreate node
			const nodeRes = await api.post<GraphNode>(`/api/projects/${ctx.projectId}/nodes`, {
				title: node.title,
				type: node.type,
				status: node.status,
				description: node.description,
				tags: node.tags,
				positionX: node.positionX,
				positionY: node.positionY,
				width: node.width,
				height: node.height,
			});
			if (nodeRes.data) {
				ctx.setNodes([...ctx.getNodes(), nodeRes.data]);
				// Recreate connected edges
				for (const edge of connectedEdges) {
					const src = edge.sourceId === node.id ? nodeRes.data.id : edge.sourceId;
					const tgt = edge.targetId === node.id ? nodeRes.data.id : edge.targetId;
					const edgeRes = await api.post<GraphEdge>(`/api/projects/${ctx.projectId}/edges`, {
						sourceId: src,
						targetId: tgt,
						edgeType: edge.edgeType,
						label: edge.label,
					});
					if (edgeRes.data) {
						ctx.setEdges([...ctx.getEdges(), edgeRes.data]);
					}
				}
			}
		},
	};
}

export function updateNodeCmd(
	ctx: NodeMutationContext,
	nodeId: string,
	oldData: Record<string, unknown>,
	newData: Record<string, unknown>,
): Command {
	return {
		description: `Update node`,
		async execute() {
			const res = await api.patch<{ node: GraphNode }>(`/api/projects/${ctx.projectId}/nodes/${nodeId}`, newData);
			if (res.data) {
				ctx.setNodes(ctx.getNodes().map((n) => (n.id === nodeId ? res.data!.node : n)));
			}
		},
		async undo() {
			const res = await api.patch<{ node: GraphNode }>(`/api/projects/${ctx.projectId}/nodes/${nodeId}`, oldData);
			if (res.data) {
				ctx.setNodes(ctx.getNodes().map((n) => (n.id === nodeId ? res.data!.node : n)));
			}
		},
	};
}

export function createEdgeCmd(
	ctx: NodeMutationContext,
	data: { sourceId: string; targetId: string; edgeType?: string },
): Command {
	let createdEdge: GraphEdge | null = null;

	return {
		description: `Create edge`,
		async execute() {
			const res = await api.post<GraphEdge>(`/api/projects/${ctx.projectId}/edges`, {
				sourceId: data.sourceId,
				targetId: data.targetId,
				edgeType: data.edgeType ?? 'related',
			});
			if (res.data) {
				createdEdge = res.data;
				ctx.setEdges([...ctx.getEdges(), res.data]);
			}
		},
		async undo() {
			if (!createdEdge) return;
			const res = await api.delete(`/api/projects/${ctx.projectId}/edges/${createdEdge.id}`);
			if (!res.error) {
				ctx.setEdges(ctx.getEdges().filter((e) => e.id !== createdEdge!.id));
			}
		},
	};
}

export function deleteEdgeCmd(
	ctx: NodeMutationContext,
	edge: GraphEdge,
): Command {
	return {
		description: `Delete edge`,
		async execute() {
			const res = await api.delete(`/api/projects/${ctx.projectId}/edges/${edge.id}`);
			if (!res.error) {
				ctx.setEdges(ctx.getEdges().filter((e) => e.id !== edge.id));
			}
		},
		async undo() {
			const res = await api.post<GraphEdge>(`/api/projects/${ctx.projectId}/edges`, {
				sourceId: edge.sourceId,
				targetId: edge.targetId,
				edgeType: edge.edgeType,
				label: edge.label,
			});
			if (res.data) {
				ctx.setEdges([...ctx.getEdges(), res.data]);
			}
		},
	};
}

export function batchDeleteCmd(
	ctx: NodeMutationContext,
	deletedNodes: GraphNode[],
	deletedEdges: GraphEdge[],
): Command {
	return {
		description: `Delete ${deletedNodes.length} nodes`,
		async execute() {
			const ids = deletedNodes.map((n) => n.id);
			const res = await api.post(`/api/projects/${ctx.projectId}/nodes/batch-delete`, { ids });
			if (!res.error) {
				const idSet = new Set(ids);
				ctx.setNodes(ctx.getNodes().filter((n) => !idSet.has(n.id)));
				ctx.setEdges(ctx.getEdges().filter((e) => !idSet.has(e.sourceId) && !idSet.has(e.targetId)));
			}
		},
		async undo() {
			// Recreate all deleted nodes
			for (const node of deletedNodes) {
				const res = await api.post<GraphNode>(`/api/projects/${ctx.projectId}/nodes`, {
					title: node.title,
					type: node.type,
					status: node.status,
					description: node.description,
					tags: node.tags,
					positionX: node.positionX,
					positionY: node.positionY,
					width: node.width,
					height: node.height,
				});
				if (res.data) {
					ctx.setNodes([...ctx.getNodes(), res.data]);
				}
			}
			// Recreate edges (best effort — node IDs may differ)
			for (const edge of deletedEdges) {
				const res = await api.post<GraphEdge>(`/api/projects/${ctx.projectId}/edges`, {
					sourceId: edge.sourceId,
					targetId: edge.targetId,
					edgeType: edge.edgeType,
					label: edge.label,
				});
				if (res.data) {
					ctx.setEdges([...ctx.getEdges(), res.data]);
				}
			}
		},
	};
}

export interface NodePosition {
	id: string;
	x: number;
	y: number;
}

export function moveNodesCmd(
	projectId: string,
	oldPositions: NodePosition[],
	newPositions: NodePosition[],
	applyPositions: (positions: NodePosition[]) => void,
	persistPositions: () => void,
): Command {
	return {
		description: `Move ${oldPositions.length} node(s)`,
		async execute() {
			applyPositions(newPositions);
			persistPositions();
		},
		async undo() {
			applyPositions(oldPositions);
			persistPositions();
		},
	};
}
