'use client';

import { create } from 'zustand';
import type { NodeType, NodeStatus } from '@/types/graph';

interface GraphState {
  selectedNodeId: string | null;
  selectedNodeIds: string[];
  isDetailPanelOpen: boolean;
  activeNodeTypeFilters: NodeType[];
  activeStatusFilters: NodeStatus[];
  isImpactModeActive: boolean;
  impactNodeIds: string[];
  impactSubgraphIds: string[];
  collapsedGroupIds: string[];
  selectNode: (id: string | null) => void;
  toggleNodeSelection: (id: string) => void;
  selectAllNodes: (ids: string[]) => void;
  clearSelection: () => void;
  toggleDetailPanel: (open?: boolean) => void;
  setNodeTypeFilter: (types: NodeType[]) => void;
  setStatusFilter: (statuses: NodeStatus[]) => void;
  activateImpactMode: (changedIds: string[], subgraphIds: string[]) => void;
  deactivateImpactMode: () => void;
  toggleGroupCollapsed: (groupId: string) => void;
}

export const useGraphStore = create<GraphState>()((set) => ({
  selectedNodeId: null,
  selectedNodeIds: [],
  isDetailPanelOpen: false,
  activeNodeTypeFilters: [],
  activeStatusFilters: [],
  isImpactModeActive: false,
  impactNodeIds: [],
  impactSubgraphIds: [],
  collapsedGroupIds: [],
  selectNode: (id) =>
    set({
      selectedNodeId: id,
      selectedNodeIds: id ? [id] : [],
      isDetailPanelOpen: id !== null,
    }),

  toggleNodeSelection: (id) =>
    set((state) => {
      const ids = state.selectedNodeIds.includes(id)
        ? state.selectedNodeIds.filter((i) => i !== id)
        : [...state.selectedNodeIds, id];
      return {
        selectedNodeIds: ids,
        selectedNodeId: ids.length === 1 ? ids[0]! : state.selectedNodeId,
        isDetailPanelOpen: ids.length === 1,
      };
    }),

  selectAllNodes: (ids) =>
    set({ selectedNodeIds: ids, isDetailPanelOpen: false }),

  clearSelection: () =>
    set({ selectedNodeIds: [], selectedNodeId: null, isDetailPanelOpen: false }),

  toggleDetailPanel: (open) =>
    set((state) => ({
      isDetailPanelOpen: open ?? !state.isDetailPanelOpen,
      selectedNodeId: open === false ? null : state.selectedNodeId,
    })),

  setNodeTypeFilter: (types) => set({ activeNodeTypeFilters: types }),
  setStatusFilter: (statuses) => set({ activeStatusFilters: statuses }),

  activateImpactMode: (changedIds, subgraphIds) =>
    set({
      isImpactModeActive: true,
      impactNodeIds: changedIds,
      impactSubgraphIds: subgraphIds,
    }),

  deactivateImpactMode: () =>
    set({
      isImpactModeActive: false,
      impactNodeIds: [],
      impactSubgraphIds: [],
    }),

  toggleGroupCollapsed: (groupId) =>
    set((state) => ({
      collapsedGroupIds: state.collapsedGroupIds.includes(groupId)
        ? state.collapsedGroupIds.filter((id) => id !== groupId)
        : [...state.collapsedGroupIds, groupId],
    })),
}));
