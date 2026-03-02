'use client';

import { useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useUndoStore } from '@/stores/useUndoStore';
import type { GraphNode, GraphEdge } from '@/types/graph';
import type { UndoEntry } from '@/types/undo';

export function useUndoRedo(projectId: string) {
  const queryClient = useQueryClient();
  const { popUndo, popRedo, pushRedo, pushUndoFromRedo, canUndo, canRedo } =
    useUndoStore();
  const isExecutingRef = useRef(false);

  const invalidateGraph = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['nodes', projectId] });
    queryClient.invalidateQueries({ queryKey: ['edges', projectId] });
  }, [queryClient, projectId]);

  const executeEntry = useCallback(
    async (entry: UndoEntry, direction: 'undo' | 'redo') => {
      switch (entry.type) {
        // ── addNode ──
        case 'addNode': {
          if (direction === 'undo') {
            await fetch(`/api/projects/${projectId}/nodes/${entry.nodeId}`, {
              method: 'DELETE',
            });
            queryClient.setQueryData<GraphNode[]>(
              ['nodes', projectId],
              (old) => (old ?? []).filter((n) => n.id !== entry.nodeId),
            );
            queryClient.setQueryData<GraphEdge[]>(
              ['edges', projectId],
              (old) =>
                (old ?? []).filter(
                  (e) =>
                    e.sourceId !== entry.nodeId && e.targetId !== entry.nodeId,
                ),
            );
          } else {
            const res = await fetch(`/api/projects/${projectId}/nodes`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                type: entry.nodeData.type,
                title: entry.nodeData.title,
                description: entry.nodeData.description,
                status: entry.nodeData.status,
                assigneeId: entry.nodeData.assigneeId,
                tags: entry.nodeData.tags,
                metadata: entry.nodeData.metadata,
                positionX: entry.nodeData.positionX,
                positionY: entry.nodeData.positionY,
                width: entry.nodeData.width,
                height: entry.nodeData.height,
              }),
            });
            const json = await res.json();
            if (json.data) {
              entry.nodeId = json.data.id;
              entry.nodeData = json.data;
            }
            invalidateGraph();
          }
          break;
        }

        // ── deleteNode ──
        case 'deleteNode': {
          if (direction === 'undo') {
            const nodeRes = await fetch(`/api/projects/${projectId}/nodes`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                type: entry.node.type,
                title: entry.node.title,
                description: entry.node.description,
                status: entry.node.status,
                assigneeId: entry.node.assigneeId,
                tags: entry.node.tags,
                metadata: entry.node.metadata,
                positionX: entry.node.positionX,
                positionY: entry.node.positionY,
                width: entry.node.width,
                height: entry.node.height,
              }),
            });
            const nodeJson = await nodeRes.json();
            const newId = nodeJson.data?.id as string | undefined;

            if (newId && entry.edges.length > 0) {
              for (const edge of entry.edges) {
                const sourceId =
                  edge.sourceId === entry.node.id ? newId : edge.sourceId;
                const targetId =
                  edge.targetId === entry.node.id ? newId : edge.targetId;
                await fetch(`/api/projects/${projectId}/edges`, {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    sourceId,
                    targetId,
                    edgeType: edge.edgeType,
                    label: edge.label,
                  }),
                });
              }
            }
            if (newId) entry.node = { ...entry.node, id: newId };
            invalidateGraph();
          } else {
            await fetch(
              `/api/projects/${projectId}/nodes/${entry.node.id}`,
              { method: 'DELETE' },
            );
            queryClient.setQueryData<GraphNode[]>(
              ['nodes', projectId],
              (old) => (old ?? []).filter((n) => n.id !== entry.node.id),
            );
            queryClient.setQueryData<GraphEdge[]>(
              ['edges', projectId],
              (old) =>
                (old ?? []).filter(
                  (e) =>
                    e.sourceId !== entry.node.id &&
                    e.targetId !== entry.node.id,
                ),
            );
          }
          break;
        }

        // ── updateNode ──
        case 'updateNode': {
          const data = direction === 'undo' ? entry.prevData : entry.nextData;
          await fetch(`/api/projects/${projectId}/nodes/${entry.nodeId}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
          });
          queryClient.setQueryData<GraphNode[]>(
            ['nodes', projectId],
            (old) =>
              (old ?? []).map((n) =>
                n.id === entry.nodeId ? { ...n, ...data } : n,
              ),
          );
          break;
        }

        // ── addEdge ──
        case 'addEdge': {
          if (direction === 'undo') {
            await fetch(
              `/api/projects/${projectId}/edges/${entry.edgeId}`,
              { method: 'DELETE' },
            );
            queryClient.setQueryData<GraphEdge[]>(
              ['edges', projectId],
              (old) => (old ?? []).filter((e) => e.id !== entry.edgeId),
            );
          } else {
            const res = await fetch(`/api/projects/${projectId}/edges`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                sourceId: entry.edgeData.sourceId,
                targetId: entry.edgeData.targetId,
                edgeType: entry.edgeData.edgeType,
                label: entry.edgeData.label,
              }),
            });
            const json = await res.json();
            if (json.data) {
              entry.edgeId = json.data.id;
              entry.edgeData = json.data;
            }
            invalidateGraph();
          }
          break;
        }

        // ── deleteEdge ──
        case 'deleteEdge': {
          if (direction === 'undo') {
            const res = await fetch(`/api/projects/${projectId}/edges`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                sourceId: entry.edge.sourceId,
                targetId: entry.edge.targetId,
                edgeType: entry.edge.edgeType,
                label: entry.edge.label,
              }),
            });
            const json = await res.json();
            if (json.data) entry.edge = json.data;
            invalidateGraph();
          } else {
            await fetch(
              `/api/projects/${projectId}/edges/${entry.edge.id}`,
              { method: 'DELETE' },
            );
            queryClient.setQueryData<GraphEdge[]>(
              ['edges', projectId],
              (old) => (old ?? []).filter((e) => e.id !== entry.edge.id),
            );
          }
          break;
        }

        // ── updateEdgeType ──
        case 'updateEdgeType': {
          const newType =
            direction === 'undo' ? entry.prevType : entry.nextType;
          await fetch(`/api/projects/${projectId}/edges/${entry.edgeId}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ edgeType: newType }),
          });
          queryClient.setQueryData<GraphEdge[]>(
            ['edges', projectId],
            (old) =>
              (old ?? []).map((e) =>
                e.id === entry.edgeId ? { ...e, edgeType: newType } : e,
              ),
          );
          break;
        }

        // ── updateEdgeLabel ──
        case 'updateEdgeLabel': {
          const newLabel =
            direction === 'undo' ? entry.prevLabel : entry.nextLabel;
          await fetch(`/api/projects/${projectId}/edges/${entry.edgeId}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ label: newLabel }),
          });
          queryClient.setQueryData<GraphEdge[]>(
            ['edges', projectId],
            (old) =>
              (old ?? []).map((e) =>
                e.id === entry.edgeId ? { ...e, label: newLabel } : e,
              ),
          );
          break;
        }

        // ── dropOnGroup ──
        case 'dropOnGroup': {
          const parentId =
            direction === 'undo' ? entry.prevParentId : entry.nextParentId;
          await fetch(`/api/projects/${projectId}/nodes/${entry.nodeId}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ parentId }),
          });
          queryClient.setQueryData<GraphNode[]>(
            ['nodes', projectId],
            (old) =>
              (old ?? []).map((n) =>
                n.id === entry.nodeId ? { ...n, parentId } : n,
              ),
          );
          break;
        }
      }
    },
    [projectId, queryClient, invalidateGraph],
  );

  const undo = useCallback(async () => {
    if (isExecutingRef.current) return;
    isExecutingRef.current = true;
    try {
      const entry = popUndo();
      if (!entry) return;
      await executeEntry(entry, 'undo');
      pushRedo(entry);
    } finally {
      isExecutingRef.current = false;
    }
  }, [popUndo, pushRedo, executeEntry]);

  const redo = useCallback(async () => {
    if (isExecutingRef.current) return;
    isExecutingRef.current = true;
    try {
      const entry = popRedo();
      if (!entry) return;
      await executeEntry(entry, 'redo');
      pushUndoFromRedo(entry);
    } finally {
      isExecutingRef.current = false;
    }
  }, [popRedo, pushUndoFromRedo, executeEntry]);

  return { undo, redo, canUndo, canRedo };
}
