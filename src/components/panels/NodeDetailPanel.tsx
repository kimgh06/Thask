'use client';

import { useState, useEffect, useLayoutEffect } from 'react';
import { createPortal } from 'react-dom';
import { X, Save, Trash2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { GraphNode, NodeType, NodeStatus, NodeHistoryEntry } from '@/types/graph';
import { NODE_TYPES, NODE_STATUSES } from '@/types/graph';

interface NodeDetailPanelProps {
  node: GraphNode | null;
  allNodes: GraphNode[];
  history: NodeHistoryEntry[];
  connectedNodeIds: string[];
  isOpen: boolean;
  onClose: () => void;
  onUpdate: (nodeId: string, data: Partial<GraphNode>) => void;
  onDelete: (nodeId: string) => void;
  onSelectNode?: (nodeId: string) => void;
}

const TYPE_LABELS: Record<NodeType, string> = {
  FLOW: 'Flow', BRANCH: 'Branch', TASK: 'Task', BUG: 'Bug', API: 'API', UI: 'UI', GROUP: 'Group',
};

const STATUS_LABELS: Record<NodeStatus, string> = {
  PASS: 'Pass', FAIL: 'Fail', IN_PROGRESS: 'In Progress', BLOCKED: 'Blocked',
};

const STATUS_COLORS: Record<NodeStatus, string> = {
  PASS: 'bg-green-100 text-green-700',
  FAIL: 'bg-red-100 text-red-700',
  IN_PROGRESS: 'bg-yellow-100 text-yellow-700',
  BLOCKED: 'bg-gray-100 text-gray-600',
};

export function NodeDetailPanel({
  node,
  allNodes,
  history,
  connectedNodeIds,
  isOpen,
  onClose,
  onUpdate,
  onDelete,
  onSelectNode,
}: NodeDetailPanelProps) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState<NodeType>('TASK');
  const [status, setStatus] = useState<NodeStatus>('IN_PROGRESS');
  const [dirty, setDirty] = useState(false);
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);

  useEffect(() => {
    if (isOpen) {
      const el = document.getElementById('sidebar-detail-slot');
      setPortalTarget(el);
    } else {
      setPortalTarget(null);
    }
  }, [isOpen]);

  useEffect(() => {
    if (node) {
      setTitle(node.title);
      setDescription(node.description ?? '');
      setType(node.type);
      setStatus(node.status);
      setDirty(false);
    }
  }, [node]);

  function handleSave() {
    if (!node) return;
    onUpdate(node.id, { title, description, type, status });
    setDirty(false);
  }

  function markDirty() {
    setDirty(true);
  }

  const isGroup = node?.type === 'GROUP';
  const children = node ? allNodes.filter((n) => n.parentId === node.id) : [];
  const parentGroup = node?.parentId ? allNodes.find((n) => n.id === node.parentId) : null;

  if (!isOpen || !node || !portalTarget) return null;

  return createPortal(
    <div className="flex h-full flex-col">
      {/* Detail header with close button */}
      <div className="flex items-center justify-between border-b border-gray-100 px-4 py-2">
        <h3 className="text-sm font-semibold text-gray-900">
          {isGroup ? 'Group Details' : 'Node Details'}
        </h3>
        <button onClick={onClose} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Title */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => { setTitle(e.target.value); markDirty(); }}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
          />
        </div>

        {/* Type */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Type</label>
          {isGroup && children.length > 0 ? (
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-500">
              Group <span className="ml-1 text-[10px] text-gray-400">(has children)</span>
            </div>
          ) : (
            <select
              value={type}
              onChange={(e) => { setType(e.target.value as NodeType); markDirty(); }}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none"
            >
              {NODE_TYPES.map((t) => (
                <option key={t} value={t}>{TYPE_LABELS[t]}</option>
              ))}
            </select>
          )}
        </div>

        {/* Status */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Status</label>
          <div className="flex flex-wrap gap-2">
            {NODE_STATUSES.map((s) => (
              <button
                key={s}
                onClick={() => { setStatus(s); markDirty(); }}
                className={cn(
                  'rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  status === s ? STATUS_COLORS[s] : 'bg-gray-50 text-gray-400 hover:bg-gray-100',
                )}
              >
                {STATUS_LABELS[s]}
              </button>
            ))}
          </div>
        </div>

        {/* Group membership */}
        {node.parentId && (
          <div className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2">
            <span className="text-xs text-gray-500">
              In group{' '}
              <button
                onClick={() => parentGroup && onSelectNode?.(parentGroup.id)}
                className="font-medium text-gray-700 hover:text-thask-primary"
              >
                {parentGroup?.title ?? node.parentId.slice(0, 8)}
              </button>
            </span>
            <button
              onClick={() => { onUpdate(node.id, { parentId: null }); }}
              className="rounded px-2 py-0.5 text-xs font-medium text-red-600 hover:bg-red-50"
            >
              Remove
            </button>
          </div>
        )}

        {/* Group children */}
        {isGroup && (
          <div>
            <label className="mb-2 block text-xs font-medium text-gray-500">
              Children ({children.length})
            </label>
            {children.length > 0 ? (
              <div className="space-y-1">
                {children.map((child) => (
                  <button
                    key={child.id}
                    onClick={() => onSelectNode?.(child.id)}
                    className="flex w-full items-center gap-2 rounded-lg bg-gray-50 px-3 py-1.5 text-left text-xs hover:bg-gray-100"
                  >
                    <span className="rounded bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium text-slate-600">
                      {child.type}
                    </span>
                    <span className="truncate text-gray-700">{child.title}</span>
                  </button>
                ))}
              </div>
            ) : (
              <p className="text-xs text-gray-400">Drag nodes into this group.</p>
            )}
          </div>
        )}

        {/* Description */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
          <textarea
            value={description}
            onChange={(e) => { setDescription(e.target.value); markDirty(); }}
            rows={3}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
            placeholder="Add a description..."
          />
        </div>

        {/* Connected nodes */}
        <div>
          <label className="mb-2 block text-xs font-medium text-gray-500">
            Connected Nodes ({connectedNodeIds.length})
          </label>
          {connectedNodeIds.length > 0 ? (
            <div className="space-y-1">
              {connectedNodeIds.map((id) => (
                <div key={id} className="rounded bg-gray-50 px-2 py-1 text-xs text-gray-600 font-mono">
                  {id.slice(0, 8)}...
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-gray-400">No connections yet.</p>
          )}
        </div>

        {/* History */}
        <div>
          <label className="mb-2 block text-xs font-medium text-gray-500">Recent Changes</label>
          {history.length > 0 ? (
            <div className="space-y-2">
              {history.map((entry) => (
                <div key={entry.id} className="rounded-lg bg-gray-50 px-3 py-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-gray-700">
                      {entry.action.replace('_', ' ')}
                      {entry.fieldName && `: ${entry.fieldName}`}
                    </span>
                    <span className="text-[10px] text-gray-400">
                      {new Date(entry.createdAt).toLocaleDateString()}
                    </span>
                  </div>
                  {entry.userName && (
                    <div className="mt-0.5 text-[10px] text-gray-400">by {entry.userName}</div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-gray-400">No changes recorded.</p>
          )}
        </div>
      </div>

      {/* Action buttons */}
      <div className="flex items-center justify-between border-t border-gray-200 px-4 py-2">
        <button
          onClick={() => onDelete(node.id)}
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
        >
          <Trash2 className="h-4 w-4" />
          Delete
        </button>
        <button
          onClick={handleSave}
          disabled={!dirty}
          className={cn(
            'flex items-center gap-1.5 rounded-lg px-4 py-1.5 text-sm font-medium',
            dirty
              ? 'bg-thask-primary text-white hover:bg-blue-600'
              : 'bg-gray-100 text-gray-400 cursor-not-allowed',
          )}
        >
          <Save className="h-4 w-4" />
          Save
        </button>
      </div>
    </div>,
    portalTarget,
  );
}
