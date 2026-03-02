'use client';

import { useEffect, useState } from 'react';
import type { GraphNode, GraphEdge, NodeHistoryEntry } from '@/types/graph';

export function useNodeDetail(
  projectId: string,
  selectedNodeId: string | null,
  nodes: GraphNode[],
  edges: GraphEdge[],
) {
  const [selectedNodeDetail, setSelectedNodeDetail] = useState<GraphNode | null>(null);
  const [nodeHistory, setNodeHistory] = useState<NodeHistoryEntry[]>([]);
  const [connectedNodeIds, setConnectedNodeIds] = useState<string[]>([]);

  useEffect(() => {
    if (!selectedNodeId) {
      setSelectedNodeDetail(null);
      setNodeHistory([]);
      setConnectedNodeIds([]);
      return;
    }

    // Instant: basic data from already-loaded nodes array
    const basicNode = nodes.find((n) => n.id === selectedNodeId);
    if (basicNode) setSelectedNodeDetail(basicNode);

    // Instant: connected nodes from already-loaded edges array
    const connected = new Set<string>();
    edges.forEach((e) => {
      if (e.sourceId === selectedNodeId) connected.add(e.targetId);
      if (e.targetId === selectedNodeId) connected.add(e.sourceId);
    });
    setConnectedNodeIds(Array.from(connected));

    // Async: only history needs a fetch
    async function fetchHistory() {
      const res = await fetch(`/api/projects/${projectId}/nodes/${selectedNodeId}`);
      if (res.ok) {
        const json = await res.json();
        setNodeHistory(json.data.history || []);
      }
    }

    fetchHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedNodeId, projectId]);

  return { selectedNodeDetail, nodeHistory, connectedNodeIds };
}
