'use client';

import { useEffect, useMemo, useState } from 'react';
import type { GraphNode, GraphEdge, NodeHistoryEntry } from '@/types/graph';

export function useNodeDetail(
  projectId: string,
  selectedNodeId: string | null,
  nodes: GraphNode[],
  edges: GraphEdge[],
) {
  const [nodeHistory, setNodeHistory] = useState<NodeHistoryEntry[]>([]);

  // Derived directly from nodes array — always in sync with optimistic updates & waterfall
  const selectedNodeDetail = useMemo(
    () => (selectedNodeId ? nodes.find((n) => n.id === selectedNodeId) ?? null : null),
    [selectedNodeId, nodes],
  );

  const connectedNodeIds = useMemo(() => {
    if (!selectedNodeId) return [];
    const connected = new Set<string>();
    edges.forEach((e) => {
      if (e.sourceId === selectedNodeId) connected.add(e.targetId);
      if (e.targetId === selectedNodeId) connected.add(e.sourceId);
    });
    return Array.from(connected);
  }, [selectedNodeId, edges]);

  // Async: only history needs a fetch
  useEffect(() => {
    if (!selectedNodeId) {
      setNodeHistory([]);
      return;
    }

    async function fetchHistory() {
      const res = await fetch(`/api/projects/${projectId}/nodes/${selectedNodeId}`);
      if (res.ok) {
        const json = await res.json();
        setNodeHistory(json.data.history || []);
      }
    }

    fetchHistory();
  }, [selectedNodeId, projectId]);

  return { selectedNodeDetail, nodeHistory, connectedNodeIds };
}
