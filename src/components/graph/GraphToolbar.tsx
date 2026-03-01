'use client';

import { useState } from 'react';
import {
  Plus,
  ZoomIn,
  ZoomOut,
  Maximize,
  LayoutGrid,
  AlertTriangle,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { NODE_TYPES, type NodeType, NODE_STATUSES, type NodeStatus } from '@/types/graph';
import { useGraphStore } from '@/stores/useGraphStore';

interface GraphToolbarProps {
  onAddNode: () => void;
  onAddGroup: () => void;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onFitView: () => void;
  onRunLayout: () => void;
  onToggleImpact: () => void;
  isImpactActive: boolean;
}

const TYPE_LABELS: Record<NodeType, string> = {
  FLOW: 'Flow',
  BRANCH: 'Branch',
  TASK: 'Task',
  BUG: 'Bug',
  API: 'API',
  UI: 'UI',
  GROUP: 'Group',
};

const STATUS_LABELS: Record<NodeStatus, string> = {
  PASS: 'Pass',
  FAIL: 'Fail',
  IN_PROGRESS: 'In Progress',
  BLOCKED: 'Blocked',
};

const STATUS_COLORS: Record<NodeStatus, string> = {
  PASS: 'bg-green-100 text-green-700 border-green-300',
  FAIL: 'bg-red-100 text-red-700 border-red-300',
  IN_PROGRESS: 'bg-yellow-100 text-yellow-700 border-yellow-300',
  BLOCKED: 'bg-gray-100 text-gray-600 border-gray-300',
};

export function GraphToolbar({
  onAddNode,
  onAddGroup,
  onZoomIn,
  onZoomOut,
  onFitView,
  onRunLayout,
  onToggleImpact,
  isImpactActive,
}: GraphToolbarProps) {
  const { activeNodeTypeFilters, activeStatusFilters, setNodeTypeFilter, setStatusFilter } =
    useGraphStore();
  const [showFilters, setShowFilters] = useState(false);

  function toggleTypeFilter(type: NodeType) {
    if (activeNodeTypeFilters.includes(type)) {
      setNodeTypeFilter(activeNodeTypeFilters.filter((t) => t !== type));
    } else {
      setNodeTypeFilter([...activeNodeTypeFilters, type]);
    }
  }

  function toggleStatusFilter(status: NodeStatus) {
    if (activeStatusFilters.includes(status)) {
      setStatusFilter(activeStatusFilters.filter((s) => s !== status));
    } else {
      setStatusFilter([...activeStatusFilters, status]);
    }
  }

  return (
    <div className="flex flex-col gap-2 border-b border-gray-200 bg-white px-4 py-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1">
          <button
            onClick={onAddNode}
            className="flex items-center gap-1.5 rounded-lg bg-thask-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-600"
          >
            <Plus className="h-4 w-4" />
            Add Node
          </button>
          <button
            onClick={onAddGroup}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            <Plus className="h-4 w-4" />
            Add Group
          </button>

          <div className="mx-2 h-6 w-px bg-gray-200" />

          <button onClick={onZoomIn} className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100" title="Zoom In">
            <ZoomIn className="h-4 w-4" />
          </button>
          <button onClick={onZoomOut} className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100" title="Zoom Out">
            <ZoomOut className="h-4 w-4" />
          </button>
          <button onClick={onFitView} className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100" title="Fit to View">
            <Maximize className="h-4 w-4" />
          </button>
          <button onClick={onRunLayout} className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100" title="Re-run Layout">
            <LayoutGrid className="h-4 w-4" />
          </button>

          <div className="mx-2 h-6 w-px bg-gray-200" />

          <button
            onClick={() => setShowFilters(!showFilters)}
            className={cn(
              'rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
              showFilters || activeNodeTypeFilters.length > 0 || activeStatusFilters.length > 0
                ? 'bg-blue-50 text-thask-primary'
                : 'text-gray-600 hover:bg-gray-100',
            )}
          >
            Filters
            {(activeNodeTypeFilters.length > 0 || activeStatusFilters.length > 0) && (
              <span className="ml-1.5 inline-flex h-5 w-5 items-center justify-center rounded-full bg-thask-primary text-xs text-white">
                {activeNodeTypeFilters.length + activeStatusFilters.length}
              </span>
            )}
          </button>
        </div>

        <button
          onClick={onToggleImpact}
          className={cn(
            'flex items-center gap-2 rounded-lg px-4 py-1.5 text-sm font-medium transition-all',
            isImpactActive
              ? 'bg-orange-500 text-white shadow-md hover:bg-orange-600'
              : 'border border-orange-300 bg-orange-50 text-orange-700 hover:bg-orange-100',
          )}
        >
          {isImpactActive ? <X className="h-4 w-4" /> : <AlertTriangle className="h-4 w-4" />}
          {isImpactActive ? 'Exit Impact Mode' : 'View Change Impact'}
        </button>
      </div>

      {showFilters && (
        <div className="flex flex-wrap items-center gap-2 pb-1">
          <span className="text-xs font-medium text-gray-500">Type:</span>
          {NODE_TYPES.map((type) => (
            <button
              key={type}
              onClick={() => toggleTypeFilter(type)}
              className={cn(
                'rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
                activeNodeTypeFilters.includes(type)
                  ? 'border-thask-primary bg-blue-50 text-thask-primary'
                  : 'border-gray-200 text-gray-500 hover:border-gray-300',
              )}
            >
              {TYPE_LABELS[type]}
            </button>
          ))}

          <div className="mx-1 h-4 w-px bg-gray-200" />

          <span className="text-xs font-medium text-gray-500">Status:</span>
          {NODE_STATUSES.map((status) => (
            <button
              key={status}
              onClick={() => toggleStatusFilter(status)}
              className={cn(
                'rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
                activeStatusFilters.includes(status)
                  ? STATUS_COLORS[status]
                  : 'border-gray-200 text-gray-500 hover:border-gray-300',
              )}
            >
              {STATUS_LABELS[status]}
            </button>
          ))}

          {(activeNodeTypeFilters.length > 0 || activeStatusFilters.length > 0) && (
            <button
              onClick={() => {
                setNodeTypeFilter([]);
                setStatusFilter([]);
              }}
              className="ml-2 text-xs text-gray-400 hover:text-gray-600"
            >
              Clear all
            </button>
          )}
        </div>
      )}
    </div>
  );
}
