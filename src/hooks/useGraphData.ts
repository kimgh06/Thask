'use client';

import { useCallback, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useGraphStore } from '@/stores/useGraphStore';
import { useUndoStore } from '@/stores/useUndoStore';
import type { GraphNode, GraphEdge, NodeType, EdgeType } from '@/types/graph';

interface ConfirmDeleteState {
  nodeId: string;
  title: string;
  message: string;
}

// -- Fetch helpers --

async function fetchNodes(projectId: string): Promise<GraphNode[]> {
  const res = await fetch(`/api/projects/${projectId}/nodes`);
  if (!res.ok) return [];
  const json = await res.json();
  return json.data || [];
}

async function fetchEdges(projectId: string): Promise<GraphEdge[]> {
  const res = await fetch(`/api/projects/${projectId}/edges`);
  if (!res.ok) return [];
  const json = await res.json();
  return json.data || [];
}

export function useGraphData(projectId: string) {
  const queryClient = useQueryClient();
  const { selectNode, activeNodeTypeFilters, activeStatusFilters } = useGraphStore();
  const pushUndo = useUndoStore((s) => s.pushUndo);

  // -- Queries --

  const { data: nodes = [], isPending: nodesLoading } = useQuery({
    queryKey: ['nodes', projectId],
    queryFn: () => fetchNodes(projectId),
  });

  const { data: edges = [], isPending: edgesLoading } = useQuery({
    queryKey: ['edges', projectId],
    queryFn: () => fetchEdges(projectId),
  });

  const loading = nodesLoading || edgesLoading;

  const invalidateGraph = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['nodes', projectId] });
    queryClient.invalidateQueries({ queryKey: ['edges', projectId] });
  }, [queryClient, projectId]);

  // Filtered nodes for display
  const filteredNodes = nodes.filter((n) => {
    if (activeNodeTypeFilters.length > 0 && !activeNodeTypeFilters.includes(n.type)) return false;
    if (activeStatusFilters.length > 0 && !activeStatusFilters.includes(n.status)) return false;
    return true;
  });

  // -- Confirm delete state --
  const [confirmDelete, setConfirmDelete] = useState<ConfirmDeleteState | null>(null);

  // -- Node mutations --

  const addNodeMutation = useMutation({
    mutationFn: async (data: { title: string; type: NodeType }) => {
      const res = await fetch(`/api/projects/${projectId}/nodes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!res.ok) throw new Error('Failed to create node');
      return res.json();
    },
    onSuccess: (json) => {
      if (json.data) pushUndo({ type: 'addNode', nodeId: json.data.id, nodeData: json.data });
      invalidateGraph();
    },
  });

  const addGroupMutation = useMutation({
    mutationFn: async ({ positionX, positionY }: { positionX: number; positionY: number }) => {
      const res = await fetch(`/api/projects/${projectId}/nodes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'New Group', type: 'GROUP', positionX, positionY }),
      });
      if (!res.ok) throw new Error('Failed to create group');
      return res.json();
    },
    onSuccess: (json) => {
      if (json.data) pushUndo({ type: 'addNode', nodeId: json.data.id, nodeData: json.data });
      invalidateGraph();
      if (json.data?.id) selectNode(json.data.id);
    },
  });

  const updateNodeMutation = useMutation({
    mutationFn: async ({ nodeId, data }: { nodeId: string; data: Partial<GraphNode> }) => {
      const res = await fetch(`/api/projects/${projectId}/nodes/${nodeId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!res.ok) throw new Error('Failed to update node');
      return res.json();
    },
    onMutate: async ({ nodeId, data }) => {
      await queryClient.cancelQueries({ queryKey: ['nodes', projectId] });
      const prev = queryClient.getQueryData<GraphNode[]>(['nodes', projectId]);
      queryClient.setQueryData<GraphNode[]>(['nodes', projectId], (old) =>
        (old ?? []).map((n) => (n.id === nodeId ? { ...n, ...data } : n)),
      );
      return { prev };
    },
    onSuccess: (json) => {
      // Apply waterfall-propagated status changes to cache
      const propagated: { nodeId: string; newStatus: string }[] = json.propagated ?? [];
      if (propagated.length > 0) {
        queryClient.setQueryData<GraphNode[]>(['nodes', projectId], (old) =>
          (old ?? []).map((n) => {
            const change = propagated.find((p) => p.nodeId === n.id);
            return change ? { ...n, status: change.newStatus as GraphNode['status'] } : n;
          }),
        );
      }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(['nodes', projectId], ctx.prev);
    },
  });

  const deleteNodeMutation = useMutation({
    mutationFn: async (nodeId: string) => {
      const res = await fetch(`/api/projects/${projectId}/nodes/${nodeId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete node');
    },
    onMutate: async (nodeId) => {
      await queryClient.cancelQueries({ queryKey: ['nodes', projectId] });
      await queryClient.cancelQueries({ queryKey: ['edges', projectId] });
      const prevNodes = queryClient.getQueryData<GraphNode[]>(['nodes', projectId]);
      const prevEdges = queryClient.getQueryData<GraphEdge[]>(['edges', projectId]);
      queryClient.setQueryData<GraphNode[]>(['nodes', projectId], (old) =>
        (old ?? []).filter((n) => n.id !== nodeId).map((n) => (n.parentId === nodeId ? { ...n, parentId: null } : n)),
      );
      queryClient.setQueryData<GraphEdge[]>(['edges', projectId], (old) =>
        (old ?? []).filter((e) => e.sourceId !== nodeId && e.targetId !== nodeId),
      );
      return { prevNodes, prevEdges };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prevNodes) queryClient.setQueryData(['nodes', projectId], ctx.prevNodes);
      if (ctx?.prevEdges) queryClient.setQueryData(['edges', projectId], ctx.prevEdges);
    },
  });

  const dropOnGroupMutation = useMutation({
    mutationFn: async ({ nodeId, groupId }: { nodeId: string; groupId: string | null }) => {
      const res = await fetch(`/api/projects/${projectId}/nodes/${nodeId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ parentId: groupId }),
      });
      if (!res.ok) throw new Error('Failed to update parent');
    },
    onMutate: async ({ nodeId, groupId }) => {
      await queryClient.cancelQueries({ queryKey: ['nodes', projectId] });
      const prev = queryClient.getQueryData<GraphNode[]>(['nodes', projectId]);
      queryClient.setQueryData<GraphNode[]>(['nodes', projectId], (old) =>
        (old ?? []).map((n) => (n.id === nodeId ? { ...n, parentId: groupId } : n)),
      );
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(['nodes', projectId], ctx.prev);
    },
  });

  // -- Edge mutations --

  const connectNodesMutation = useMutation({
    mutationFn: async ({ sourceId, targetId }: { sourceId: string; targetId: string }) => {
      const res = await fetch(`/api/projects/${projectId}/edges`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sourceId, targetId }),
      });
      if (!res.ok) throw new Error('Failed to create edge');
      return res.json();
    },
    onSuccess: (json) => {
      if (json.data) pushUndo({ type: 'addEdge', edgeId: json.data.id, edgeData: json.data });
      invalidateGraph();
    },
  });

  const updateEdgeMutation = useMutation({
    mutationFn: async ({ edgeId, edgeType }: { edgeId: string; edgeType: EdgeType }) => {
      const res = await fetch(`/api/projects/${projectId}/edges/${edgeId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ edgeType }),
      });
      if (!res.ok) throw new Error('Failed to update edge');
    },
    onMutate: async ({ edgeId, edgeType }) => {
      await queryClient.cancelQueries({ queryKey: ['edges', projectId] });
      const prev = queryClient.getQueryData<GraphEdge[]>(['edges', projectId]);
      queryClient.setQueryData<GraphEdge[]>(['edges', projectId], (old) =>
        (old ?? []).map((e) => (e.id === edgeId ? { ...e, edgeType } : e)),
      );
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(['edges', projectId], ctx.prev);
    },
  });

  const deleteEdgeMutation = useMutation({
    mutationFn: async (edgeId: string) => {
      const res = await fetch(`/api/projects/${projectId}/edges/${edgeId}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to delete edge');
    },
    onMutate: async (edgeId) => {
      await queryClient.cancelQueries({ queryKey: ['edges', projectId] });
      const prev = queryClient.getQueryData<GraphEdge[]>(['edges', projectId]);
      queryClient.setQueryData<GraphEdge[]>(['edges', projectId], (old) =>
        (old ?? []).filter((e) => e.id !== edgeId),
      );
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) queryClient.setQueryData(['edges', projectId], ctx.prev);
    },
  });

  // -- Stable API (same signatures as before) --

  function addNode(data: { title: string; type: NodeType }) {
    addNodeMutation.mutate(data);
  }

  function addGroup(positionX: number, positionY: number) {
    addGroupMutation.mutate({ positionX, positionY });
  }

  function updateNode(nodeId: string, data: Partial<GraphNode>) {
    const prev = nodes.find((n) => n.id === nodeId);
    if (prev) {
      const prevData: Partial<GraphNode> = {};
      for (const key of Object.keys(data) as (keyof GraphNode)[]) {
        (prevData as Record<string, unknown>)[key] = prev[key];
      }
      pushUndo({ type: 'updateNode', nodeId, prevData, nextData: data });
    }
    selectNode(nodeId);
    updateNodeMutation.mutate({ nodeId, data });
  }

  function requestDeleteNode(nodeId: string) {
    const node = nodes.find((n) => n.id === nodeId);
    if (!node) return;

    const edgeCount = edges.filter((e) => e.sourceId === nodeId || e.targetId === nodeId).length;
    const childCount = nodes.filter((n) => n.parentId === nodeId).length;

    if (node.type === 'GROUP' && childCount > 0) {
      const edgeNote = edgeCount > 0 ? ` ${edgeCount} connected edge(s) will also be deleted.` : '';
      setConfirmDelete({
        nodeId,
        title: 'Delete Group',
        message: `This group contains ${childCount} node(s). They will be removed from the group (preserved).${edgeNote}`,
      });
      return;
    }

    if (edgeCount > 0) {
      setConfirmDelete({
        nodeId,
        title: `Delete ${node.type === 'GROUP' ? 'Group' : 'Node'}`,
        message: `"${node.title}" and ${edgeCount} connected edge(s) will be deleted.`,
      });
      return;
    }

    executeDeleteNode(nodeId);
  }

  function executeDeleteNode(nodeId: string) {
    const node = nodes.find((n) => n.id === nodeId);
    const nodeEdges = edges.filter((e) => e.sourceId === nodeId || e.targetId === nodeId);
    if (node) pushUndo({ type: 'deleteNode', node, edges: nodeEdges });
    setConfirmDelete(null);
    selectNode(null);
    deleteNodeMutation.mutate(nodeId);
  }

  function savePositions(positions: Array<{ id: string; x: number; y: number }>) {
    fetch(`/api/projects/${projectId}/nodes/positions`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ positions }),
    });
  }

  function resizeNode(nodeId: string, width: number, height: number) {
    queryClient.setQueryData<GraphNode[]>(['nodes', projectId], (old) =>
      (old ?? []).map((n) => (n.id === nodeId ? { ...n, width, height } : n)),
    );
    const node = nodes.find((n) => n.id === nodeId);
    if (!node) return;
    fetch(`/api/projects/${projectId}/nodes/positions`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ positions: [{ id: nodeId, x: node.positionX, y: node.positionY, width, height }] }),
    });
  }

  function dropNodeOnGroup(nodeId: string, groupId: string | null) {
    const prevParentId = nodes.find((n) => n.id === nodeId)?.parentId ?? null;
    pushUndo({ type: 'dropOnGroup', nodeId, prevParentId, nextParentId: groupId });
    dropOnGroupMutation.mutate({ nodeId, groupId });
  }

  function connectNodes(sourceId: string, targetId: string) {
    connectNodesMutation.mutate({ sourceId, targetId });
  }

  function updateEdgeType(edgeId: string, edgeType: EdgeType) {
    const prev = edges.find((e) => e.id === edgeId);
    if (prev) pushUndo({ type: 'updateEdgeType', edgeId, prevType: prev.edgeType, nextType: edgeType });
    updateEdgeMutation.mutate({ edgeId, edgeType });
  }

  function updateEdgeLabel(edgeId: string, label: string) {
    const prev = edges.find((e) => e.id === edgeId);
    if (prev) pushUndo({ type: 'updateEdgeLabel', edgeId, prevLabel: prev.label, nextLabel: label || null });
    // Optimistic update
    queryClient.setQueryData<GraphEdge[]>(['edges', projectId], (old) =>
      (old ?? []).map((e) => (e.id === edgeId ? { ...e, label } : e)),
    );
    fetch(`/api/projects/${projectId}/edges/${edgeId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ label: label || null }),
    }).then((r) => { if (!r.ok) invalidateGraph(); });
  }

  function deleteEdge(edgeId: string) {
    const edge = edges.find((e) => e.id === edgeId);
    if (edge) pushUndo({ type: 'deleteEdge', edge });
    deleteEdgeMutation.mutate(edgeId);
  }

  return {
    nodes, edges, loading, filteredNodes, fetchGraph: invalidateGraph,
    addNode, addGroup, updateNode,
    requestDeleteNode, executeDeleteNode, confirmDelete, setConfirmDelete,
    savePositions, resizeNode, dropNodeOnGroup,
    connectNodes, updateEdgeType, updateEdgeLabel, deleteEdge,
  };
}
