'use client';

import { useEffect, useRef } from 'react';
import { useParams } from 'next/navigation';
import { CytoscapeCanvas, type CytoscapeCanvasHandle } from '@/components/graph/CytoscapeCanvas';
import { GraphToolbar } from '@/components/graph/GraphToolbar';
import { AddNodeModal } from '@/components/graph/AddNodeModal';
import { EdgeColorPopover } from '@/components/graph/EdgeColorPopover';
import { GraphMinimap } from '@/components/graph/GraphMinimap';
import { NodeDetailPanel } from '@/components/panels/NodeDetailPanel';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { useGraphStore } from '@/stores/useGraphStore';
import { useGraphData } from '@/hooks/useGraphData';
import { useNodeDetail } from '@/hooks/useNodeDetail';
import { useEdgePopover } from '@/hooks/useEdgePopover';
import { useUndoRedo } from '@/hooks/useUndoRedo';
import { activateImpactMode, deactivateImpactMode } from '@/lib/cytoscape/impact';
import type { GraphNode, EdgeType } from '@/types/graph';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

export default function ProjectGraphPage() {
  const params = useParams<{ teamSlug: string; projectId: string }>();
  const { teamSlug, projectId } = params;
  const graphRef = useRef<CytoscapeCanvasHandle>(null);
  const [showAddModal, setShowAddModal] = useState(false);

  const { data: teamMembers = [] } = useQuery({
    queryKey: ['teamMembers', teamSlug],
    queryFn: async () => {
      const res = await fetch(`/api/teams/${teamSlug}/members`);
      if (!res.ok) return [];
      const json = await res.json();
      return json.data ?? [];
    },
  });

  const {
    selectedNodeId, selectedNodeIds, isDetailPanelOpen, isImpactModeActive,
    selectNode, toggleNodeSelection, selectAllNodes, clearSelection,
    toggleDetailPanel,
    activateImpactMode: storeActivateImpact,
    deactivateImpactMode: storeDeactivateImpact,
    collapsedGroupIds, toggleGroupCollapsed,
    activeNodeTypeFilters, activeStatusFilters,
  } = useGraphStore();

  const {
    nodes, edges, loading, filteredNodes,
    addNode, addGroup, updateNode,
    requestDeleteNode, executeDeleteNode, confirmDelete, setConfirmDelete,
    savePositions, resizeNode, dropNodeOnGroup,
    connectNodes, updateEdgeType, updateEdgeLabel, deleteEdge,
  } = useGraphData(projectId);

  const { selectedNodeDetail, nodeHistory, connectedNodeIds } = useNodeDetail(
    projectId, selectedNodeId, nodes, edges,
  );

  const { edgeColorPopover, showPopover, hidePopover } = useEdgePopover(graphRef);
  const { undo, redo, canUndo, canRedo } = useUndoRedo(projectId);

  // Keyboard shortcuts
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement)?.tagName;
      const isInput = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';

      // Ctrl+Z: undo, Ctrl+Shift+Z / Ctrl+Y: redo
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey && !isInput) {
        e.preventDefault();
        undo();
        return;
      }
      if ((e.ctrlKey || e.metaKey) && ((e.key === 'z' && e.shiftKey) || e.key === 'y') && !isInput) {
        e.preventDefault();
        redo();
        return;
      }

      if (e.key === 'Escape') {
        if (edgeColorPopover.visible) { hidePopover(); return; }
        if (selectedNodeIds.length > 0) { clearSelection(); return; }
      }

      // Ctrl+A: select all nodes
      if ((e.ctrlKey || e.metaKey) && e.key === 'a' && !isInput) {
        e.preventDefault();
        selectAllNodes(filteredNodes.map((n) => n.id));
        return;
      }

      if (e.key === 'Delete' || e.key === 'Backspace') {
        if (isInput) return;
        e.preventDefault();

        if (edgeColorPopover.visible && edgeColorPopover.edgeId) {
          deleteEdge(edgeColorPopover.edgeId);
          hidePopover();
          return;
        }

        // Bulk delete for multi-select
        if (selectedNodeIds.length > 1) {
          selectedNodeIds.forEach((id) => requestDeleteNode(id));
          clearSelection();
          return;
        }

        if (selectedNodeId) {
          requestDeleteNode(selectedNodeId);
        }
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedNodeId, selectedNodeIds, edgeColorPopover, filteredNodes, undo, redo]);

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

  // Impact mode
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

  function handleAddGroup() {
    const cy = graphRef.current?.getCy();
    const center = cy ? cy.extent() : null;
    const positionX = center ? (center.x1 + center.x2) / 2 : 0;
    const positionY = center ? (center.y1 + center.y2) / 2 : 0;
    addGroup(positionX, positionY);
  }

  function handleEdgeColorSelect(edgeType: EdgeType) {
    if (!edgeColorPopover.edgeId) return;
    updateEdgeType(edgeColorPopover.edgeId, edgeType);
    hidePopover();
  }

  function handleEdgeDelete() {
    if (!edgeColorPopover.edgeId) return;
    deleteEdge(edgeColorPopover.edgeId);
    hidePopover();
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
        nodes={nodes}
        onFocusNode={(nodeId) => graphRef.current?.focusNode(nodeId)}
        onUndo={undo}
        onRedo={redo}
        canUndo={canUndo}
        canRedo={canRedo}
      />

      <div className="relative flex-1">
        <CytoscapeCanvas
          ref={graphRef}
          nodes={filteredNodes}
          edges={edges}
          collapsedGroupIds={collapsedGroupIds}
          selectedNodeIds={selectedNodeIds}
          onNodeSelect={selectNode}
          onNodeToggleSelect={toggleNodeSelection}
          onNodeDragEnd={savePositions}
          onConnectEnd={connectNodes}
          onEdgeClick={showPopover}
          onNodeDropOnGroup={dropNodeOnGroup}
          onToggleGroupCollapse={toggleGroupCollapsed}
          onNodeResize={resizeNode}
        />
        <GraphMinimap getCy={() => graphRef.current?.getCy() ?? null} />
      </div>

      {edgeColorPopover.visible && (
        <EdgeColorPopover
          position={edgeColorPopover.position}
          currentLabel={edges.find((e) => e.id === edgeColorPopover.edgeId)?.label ?? ''}
          onSelect={handleEdgeColorSelect}
          onUpdateLabel={(label) => edgeColorPopover.edgeId && updateEdgeLabel(edgeColorPopover.edgeId, label)}
          onDelete={handleEdgeDelete}
          onCancel={hidePopover}
        />
      )}

      <NodeDetailPanel
        node={selectedNodeDetail}
        allNodes={nodes}
        history={nodeHistory}
        connectedNodeIds={connectedNodeIds}
        teamMembers={teamMembers}
        isOpen={isDetailPanelOpen}
        onClose={() => toggleDetailPanel(false)}
        onUpdate={updateNode}
        onDelete={requestDeleteNode}
        onSelectNode={(id) => selectNode(id)}
      />

      {showAddModal && (
        <AddNodeModal
          onSubmit={addNode}
          onClose={() => setShowAddModal(false)}
        />
      )}

      {confirmDelete && (
        <ConfirmDialog
          title={confirmDelete.title}
          message={confirmDelete.message}
          confirmLabel="Delete"
          variant="danger"
          onConfirm={() => executeDeleteNode(confirmDelete.nodeId)}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  );
}
