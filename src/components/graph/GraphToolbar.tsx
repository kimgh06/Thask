'use client';

import { useState, useRef, useEffect } from 'react';
import {
  Plus,
  ZoomIn,
  ZoomOut,
  Maximize,
  LayoutGrid,
  AlertTriangle,
  X,
  Search,
  Undo2,
  Redo2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { NODE_TYPES, type NodeType, NODE_STATUSES, type NodeStatus } from '@/types/graph';
import type { GraphNode } from '@/types/graph';
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
  nodes?: GraphNode[];
  onFocusNode?: (nodeId: string) => void;
  onUndo?: () => void;
  onRedo?: () => void;
  canUndo?: boolean;
  canRedo?: boolean;
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
  nodes = [],
  onFocusNode,
  onUndo,
  onRedo,
  canUndo = false,
  canRedo = false,
}: GraphToolbarProps) {
  const { activeNodeTypeFilters, activeStatusFilters, setNodeTypeFilter, setStatusFilter } =
    useGraphStore();
  const [showFilters, setShowFilters] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchIndex, setSearchIndex] = useState(0);
  const searchInputRef = useRef<HTMLInputElement>(null);

  const searchResults = searchQuery.trim()
    ? nodes.filter((n) => n.title.toLowerCase().includes(searchQuery.toLowerCase()))
    : [];

  // Ctrl+F shortcut
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
        e.preventDefault();
        setSearchOpen(true);
        setTimeout(() => searchInputRef.current?.focus(), 50);
      }
      if (e.key === 'Escape' && searchOpen) {
        setSearchOpen(false);
        setSearchQuery('');
        setSearchIndex(0);
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [searchOpen]);

  function handleSearchSubmit() {
    if (searchResults.length === 0) return;
    const idx = searchIndex % searchResults.length;
    const target = searchResults[idx];
    if (!target) return;
    onFocusNode?.(target.id);
    setSearchIndex(idx + 1);
  }

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
            onClick={onUndo}
            disabled={!canUndo}
            className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed"
            title="Undo (Ctrl+Z)"
          >
            <Undo2 className="h-4 w-4" />
          </button>
          <button
            onClick={onRedo}
            disabled={!canRedo}
            className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 disabled:opacity-30 disabled:cursor-not-allowed"
            title="Redo (Ctrl+Shift+Z)"
          >
            <Redo2 className="h-4 w-4" />
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

          <div className="mx-2 h-6 w-px bg-gray-200" />

          {searchOpen ? (
            <div className="relative flex items-center gap-1">
              <Search className="h-4 w-4 text-gray-400" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => { setSearchQuery(e.target.value); setSearchIndex(0); }}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleSearchSubmit(); } }}
                placeholder="Search nodes..."
                className="w-44 rounded-lg border border-gray-300 px-2 py-1 text-sm focus:border-thask-primary focus:outline-none focus:ring-1 focus:ring-thask-primary/30"
                autoFocus
              />
              {searchQuery && (
                <span className="text-xs text-gray-400">
                  {searchResults.length > 0 ? `${(searchIndex % searchResults.length) + 1}/${searchResults.length}` : '0'}
                </span>
              )}
              <button
                onClick={() => { setSearchOpen(false); setSearchQuery(''); setSearchIndex(0); }}
                className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
              >
                <X className="h-3.5 w-3.5" />
              </button>
              {searchQuery && searchResults.length > 0 && (
                <div className="absolute left-0 top-full z-50 mt-1 max-h-48 w-64 overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-lg">
                  {searchResults.map((node, i) => (
                    <button
                      key={node.id}
                      onClick={() => { onFocusNode?.(node.id); setSearchIndex(i + 1); }}
                      className={cn(
                        'flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-gray-50',
                        i === (searchIndex % searchResults.length) && 'bg-blue-50',
                      )}
                    >
                      <span className="rounded bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium text-slate-600">
                        {node.type}
                      </span>
                      <span className="truncate text-gray-700">{node.title}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <button
              onClick={() => { setSearchOpen(true); setTimeout(() => searchInputRef.current?.focus(), 50); }}
              className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100"
              title="Search (Ctrl+F)"
            >
              <Search className="h-4 w-4" />
            </button>
          )}
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
