'use client';

import { create } from 'zustand';
import type { NodeType, NodeStatus } from '@/types/graph';

interface GraphState {
  selectedNodeId: string | null;
  isDetailPanelOpen: boolean;
  activeNodeTypeFilters: NodeType[];
  activeStatusFilters: NodeStatus[];
  isImpactModeActive: boolean;
  impactNodeIds: string[];
  impactSubgraphIds: string[];
  collapsedGroupIds: string[];
  selectNode: (id: string | null) => void;
  toggleDetailPanel: (open?: boolean) => void;
  setNodeTypeFilter: (types: NodeType[]) => void;
  setStatusFilter: (statuses: NodeStatus[]) => void;
  activateImpactMode: (changedIds: string[], subgraphIds: string[]) => void;
  deactivateImpactMode: () => void;
  toggleGroupCollapsed: (groupId: string) => void;
}

export const useGraphStore = create<GraphState>()((set) => ({
  selectedNodeId: null,
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
      isDetailPanelOpen: id !== null,
    }),

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
