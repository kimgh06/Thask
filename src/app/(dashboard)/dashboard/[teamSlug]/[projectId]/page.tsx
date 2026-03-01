'use client';

import { useEffect, useState, useRef, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { CytoscapeCanvas, type CytoscapeCanvasHandle } from '@/components/graph/CytoscapeCanvas';
import { GraphToolbar } from '@/components/graph/GraphToolbar';
import { AddNodeModal } from '@/components/graph/AddNodeModal';
import { EdgeColorPopover } from '@/components/graph/EdgeColorPopover';
import { NodeDetailPanel } from '@/components/panels/NodeDetailPanel';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { useGraphStore } from '@/stores/useGraphStore';
import { activateImpactMode, deactivateImpactMode } from '@/lib/cytoscape/impact';
import type { GraphNode, GraphEdge, NodeType, EdgeType, NodeHistoryEntry } from '@/types/graph';

export default function ProjectGraphPage() {
  const params = useParams<{ teamSlug: string; projectId: string }>();
  const projectId = params.projectId;
  const graphRef = useRef<CytoscapeCanvasHandle>(null);

  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [edges, setEdges] = useState<GraphEdge[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);

  // Detail panel state
  const [selectedNodeDetail, setSelectedNodeDetail] = useState<GraphNode | null>(null);
  const [nodeHistory, setNodeHistory] = useState<NodeHistoryEntry[]>([]);
  const [connectedNodeIds, setConnectedNodeIds] = useState<string[]>([]);

  // Confirm dialog state
  const [confirmDelete, setConfirmDelete] = useState<{ nodeId: string; childCount: number } | null>(null);

  // Edge color popover state (for changing existing edge color)
  const [edgeColorPopover, setEdgeColorPopover] = useState<{
    visible: boolean;
    position: { x: number; y: number };
    edgeId: string;
  }>({ visible: false, position: { x: 0, y: 0 }, edgeId: '' });

  const {
    selectedNodeId,
    isDetailPanelOpen,
    isImpactModeActive,
    selectNode,
    toggleDetailPanel,
    activeNodeTypeFilters,
    activeStatusFilters,
    activateImpactMode: storeActivateImpact,
    deactivateImpactMode: storeDeactivateImpact,
    collapsedGroupIds,
    toggleGroupCollapsed,
  } = useGraphStore();

  // Fetch graph data
  const fetchGraph = useCallback(async () => {
    try {
      const [nodesRes, edgesRes] = await Promise.all([
        fetch(`/api/projects/${projectId}/nodes`),
        fetch(`/api/projects/${projectId}/edges`),
      ]);
      const nodesJson = await nodesRes.json();
      const edgesJson = await edgesRes.json();
      if (nodesRes.ok) setNodes(nodesJson.data || []);
      if (edgesRes.ok) setEdges(edgesJson.data || []);
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchGraph();
  }, [fetchGraph]);

  // ESC key to close edge color popover
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && edgeColorPopover.visible) {
        setEdgeColorPopover((p) => ({ ...p, visible: false }));
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [edgeColorPopover.visible]);

  // Filter nodes for display
  const filteredNodes = nodes.filter((n) => {
    if (activeNodeTypeFilters.length > 0 && !activeNodeTypeFilters.includes(n.type)) return false;
    if (activeStatusFilters.length > 0 && !activeStatusFilters.includes(n.status)) return false;
    return true;
  });

  // Apply filters to cytoscape visibility
  useEffect(() => {
    const cy = graphRef.current?.getCy();
    if (!cy) return;

    cy.batch(() => {
      cy.nodes().forEach((cyNode) => {
        const nodeData = nodes.find((n) => n.id === cyNode.id());
        if (!nodeData) return;

        const typeMatch = activeNodeTypeFilters.length === 0 || activeNodeTypeFilters.includes(nodeData.type);
        const statusMatch = activeStatusFilters.length === 0 || activeStatusFilters.includes(nodeData.status);

        if (typeMatch && statusMatch) {
          cyNode.style('display', 'element');
        } else {
          cyNode.style('display', 'none');
        }
      });
    });
  }, [activeNodeTypeFilters, activeStatusFilters, nodes]);

  // Node selection - fetch detail
  useEffect(() => {
    if (!selectedNodeId) {
      setSelectedNodeDetail(null);
      setNodeHistory([]);
      setConnectedNodeIds([]);
      return;
    }

    async function fetchDetail() {
      const res = await fetch(`/api/projects/${projectId}/nodes/${selectedNodeId}`);
      if (res.ok) {
        const json = await res.json();
        setSelectedNodeDetail(json.data);
        setNodeHistory(json.data.history || []);
        setConnectedNodeIds(json.data.connectedNodeIds || []);
      }
    }

    fetchDetail();
  }, [selectedNodeId, projectId]);

  // Handlers
  function handleNodeSelect(nodeId: string | null) {
    selectNode(nodeId);
  }

  async function handleAddNode(data: { title: string; type: NodeType }) {
    const res = await fetch(`/api/projects/${projectId}/nodes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (res.ok) {
      await fetchGraph();
    }
  }

  async function handleAddGroup() {
    const cy = graphRef.current?.getCy();
    const center = cy ? cy.extent() : null;
    const positionX = center ? (center.x1 + center.x2) / 2 : 0;
    const positionY = center ? (center.y1 + center.y2) / 2 : 0;

    const res = await fetch(`/api/projects/${projectId}/nodes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'New Group', type: 'GROUP', positionX, positionY }),
    });
    if (res.ok) {
      const json = await res.json();
      await fetchGraph();
      if (json.data?.id) selectNode(json.data.id);
    }
  }

  function handleUpdateNode(nodeId: string, data: Partial<GraphNode>) {
    // Optimistic update
    setNodes((prev) => prev.map((n) => (n.id === nodeId ? { ...n, ...data } : n)));
    selectNode(nodeId);
    fetch(`/api/projects/${projectId}/nodes/${nodeId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
  }

  function handleDeleteNode(nodeId: string) {
    const node = nodes.find((n) => n.id === nodeId);
    if (node?.type === 'GROUP') {
      const childCount = nodes.filter((n) => n.parentId === nodeId).length;
      if (childCount > 0) {
        setConfirmDelete({ nodeId, childCount });
        return;
      }
    }
    executeDeleteNode(nodeId);
  }

  function executeDeleteNode(nodeId: string) {
    setConfirmDelete(null);
    selectNode(null);
    // Optimistic: remove node, unparent children, remove connected edges
    setNodes((prev) =>
      prev
        .filter((n) => n.id !== nodeId)
        .map((n) => (n.parentId === nodeId ? { ...n, parentId: null } : n)),
    );
    setEdges((prev) => prev.filter((e) => e.sourceId !== nodeId && e.targetId !== nodeId));
    fetch(`/api/projects/${projectId}/nodes/${nodeId}`, { method: 'DELETE' });
  }

  function handleNodeDragEnd(nodeId: string, x: number, y: number) {
    fetch(`/api/projects/${projectId}/nodes/positions`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ positions: [{ id: nodeId, x, y }] }),
    });
  }

  function handleNodeResize(nodeId: string, width: number, height: number) {
    setNodes((prev) => prev.map((n) => (n.id === nodeId ? { ...n, width, height } : n)));
    const node = nodes.find((n) => n.id === nodeId);
    if (!node) return;
    fetch(`/api/projects/${projectId}/nodes/positions`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ positions: [{ id: nodeId, x: node.positionX, y: node.positionY, width, height }] }),
    });
  }

  function handleNodeDropOnGroup(nodeId: string, groupId: string | null) {
    // Optimistic update
    setNodes((prev) => prev.map((n) => (n.id === nodeId ? { ...n, parentId: groupId } : n)));
    fetch(`/api/projects/${projectId}/nodes/${nodeId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ parentId: groupId }),
    });
  }

  async function handleToggleImpact() {
    const cy = graphRef.current?.getCy();
    if (!cy) return;

    if (isImpactModeActive) {
      deactivateImpactMode(cy);
      storeDeactivateImpact();
    } else {
      const res = await fetch(`/api/projects/${projectId}/impact`);
      if (res.ok) {
        const json = await res.json();
        const changedIds = (json.data.changedNodes as GraphNode[]).map((n) => n.id);
        const impactedIds = (json.data.impactedNodes as GraphNode[]).map((n) => n.id);
        activateImpactMode(cy, changedIds, impactedIds);
        storeActivateImpact(changedIds, [...changedIds, ...impactedIds]);
      }
    }
  }

  // Called by edgehandles ehcomplete — instantly create edge with default type
  async function handleConnectEnd(sourceId: string, targetId: string) {
    const res = await fetch(`/api/projects/${projectId}/edges`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sourceId, targetId }),
    });
    if (res.ok) {
      await fetchGraph();
    }
  }

  // Edge click — show color popover to change edge type
  function handleEdgeClick(edgeId: string, renderedPosition: { x: number; y: number }) {
    const container = graphRef.current?.getCy()?.container();
    const rect = container?.getBoundingClientRect();
    const pageX = (rect?.left ?? 0) + renderedPosition.x;
    const pageY = (rect?.top ?? 0) + renderedPosition.y;

    setEdgeColorPopover({
      visible: true,
      position: { x: pageX + 10, y: pageY - 10 },
      edgeId,
    });
  }

  function handleEdgeColorSelect(edgeType: EdgeType) {
    if (!edgeColorPopover.edgeId) return;
    const edgeId = edgeColorPopover.edgeId;
    setEdgeColorPopover((p) => ({ ...p, visible: false }));
    // Optimistic update
    setEdges((prev) => prev.map((e) => (e.id === edgeId ? { ...e, edgeType } : e)));
    fetch(`/api/projects/${projectId}/edges/${edgeId}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ edgeType }),
    });
  }

  function handleEdgeDelete() {
    if (!edgeColorPopover.edgeId) return;
    const edgeId = edgeColorPopover.edgeId;
    setEdgeColorPopover((p) => ({ ...p, visible: false }));
    // Optimistic update
    setEdges((prev) => prev.filter((e) => e.id !== edgeId));
    fetch(`/api/projects/${projectId}/edges/${edgeId}`, { method: 'DELETE' });
  }

  function handleEdgeColorCancel() {
    setEdgeColorPopover((p) => ({ ...p, visible: false }));
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-thask-primary" />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col -m-6">
      <GraphToolbar
        onAddNode={() => setShowAddModal(true)}
        onAddGroup={handleAddGroup}
        onZoomIn={() => graphRef.current?.zoomIn()}
        onZoomOut={() => graphRef.current?.zoomOut()}
        onFitView={() => graphRef.current?.fitToView()}
        onRunLayout={() => graphRef.current?.runLayout()}
        onToggleImpact={handleToggleImpact}
        isImpactActive={isImpactModeActive}
      />

      <div className="relative flex-1">
        <CytoscapeCanvas
          ref={graphRef}
          nodes={filteredNodes}
          edges={edges}
          collapsedGroupIds={collapsedGroupIds}
          onNodeSelect={handleNodeSelect}
          onNodeDragEnd={handleNodeDragEnd}
          onConnectEnd={handleConnectEnd}
          onEdgeClick={handleEdgeClick}
          onNodeDropOnGroup={handleNodeDropOnGroup}
          onToggleGroupCollapse={toggleGroupCollapsed}
          onNodeResize={handleNodeResize}
        />
      </div>

      {edgeColorPopover.visible && (
        <EdgeColorPopover
          position={edgeColorPopover.position}
          onSelect={handleEdgeColorSelect}
          onDelete={handleEdgeDelete}
          onCancel={handleEdgeColorCancel}
        />
      )}

      <NodeDetailPanel
        node={selectedNodeDetail}
        allNodes={nodes}
        history={nodeHistory}
        connectedNodeIds={connectedNodeIds}
        isOpen={isDetailPanelOpen}
        onClose={() => toggleDetailPanel(false)}
        onUpdate={handleUpdateNode}
        onDelete={handleDeleteNode}
        onSelectNode={(id) => selectNode(id)}
      />

      {showAddModal && (
        <AddNodeModal
          onSubmit={handleAddNode}
          onClose={() => setShowAddModal(false)}
        />
      )}

      {confirmDelete && (
        <ConfirmDialog
          title="Delete Group"
          message={`This group contains ${confirmDelete.childCount} node(s). Deleting will remove them from the group (nodes are preserved).`}
          confirmLabel="Delete Group"
          variant="danger"
          onConfirm={() => executeDeleteNode(confirmDelete.nodeId)}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  );
}
