'use client';

import { useState, useEffect, useLayoutEffect, useRef, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { X, Trash2, Check, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { GraphNode, NodeType, NodeStatus, NodeHistoryEntry } from '@/types/graph';
import { NODE_TYPES, NODE_STATUSES } from '@/types/graph';

interface TeamMember {
  userId: string;
  displayName: string;
}

interface NodeDetailPanelProps {
  node: GraphNode | null;
  allNodes: GraphNode[];
  history: NodeHistoryEntry[];
  connectedNodeIds: string[];
  teamMembers?: TeamMember[];
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

const DEBOUNCE_MS = 500;

export function NodeDetailPanel({
  node,
  allNodes,
  history,
  connectedNodeIds,
  teamMembers = [],
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
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle');
  const [tagInput, setTagInput] = useState('');
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);

  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingFields = useRef<Partial<GraphNode> | null>(null);
  const nodeIdRef = useRef<string | null>(null);
  const savedIndicatorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Flush any pending debounced save immediately
  const flush = useCallback(() => {
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
      debounceTimer.current = null;
    }
    if (pendingFields.current && nodeIdRef.current) {
      onUpdate(nodeIdRef.current, pendingFields.current);
      pendingFields.current = null;
      showSaved();
    }
  }, [onUpdate]);

  function showSaved() {
    setSaveStatus('saved');
    if (savedIndicatorTimer.current) clearTimeout(savedIndicatorTimer.current);
    savedIndicatorTimer.current = setTimeout(() => setSaveStatus('idle'), 1500);
  }

  // Debounced save for text fields
  function debounceSave(fields: Partial<GraphNode>) {
    if (!nodeIdRef.current) return;
    pendingFields.current = { ...pendingFields.current, ...fields };
    setSaveStatus('saving');
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    debounceTimer.current = setTimeout(() => {
      if (pendingFields.current && nodeIdRef.current) {
        onUpdate(nodeIdRef.current, pendingFields.current);
        pendingFields.current = null;
        debounceTimer.current = null;
        showSaved();
      }
    }, DEBOUNCE_MS);
  }

  // Immediate save for discrete fields
  function immediateSave(fields: Partial<GraphNode>) {
    if (!nodeIdRef.current) return;
    // Flush any pending text changes first
    flush();
    onUpdate(nodeIdRef.current, fields);
    showSaved();
  }

  useLayoutEffect(() => {
    if (isOpen) {
      const el = document.getElementById('sidebar-detail-slot');
      setPortalTarget(el);
    } else {
      setPortalTarget(null);
    }
  }, [isOpen]);

  // Sync local state from node prop & flush on node switch
  useEffect(() => {
    // Flush pending changes for the previous node
    if (nodeIdRef.current && nodeIdRef.current !== node?.id) {
      flush();
    }

    if (node) {
      nodeIdRef.current = node.id;
      setTitle(node.title);
      setDescription(node.description ?? '');
      setType(node.type);
      setStatus(node.status);
      setSaveStatus('idle');
    } else {
      nodeIdRef.current = null;
    }
  }, [node, flush]);

  // Flush on unmount (panel close)
  useEffect(() => {
    return () => {
      if (debounceTimer.current) clearTimeout(debounceTimer.current);
      if (pendingFields.current && nodeIdRef.current) {
        onUpdate(nodeIdRef.current, pendingFields.current);
        pendingFields.current = null;
      }
      if (savedIndicatorTimer.current) clearTimeout(savedIndicatorTimer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function handleClose() {
    flush();
    onClose();
  }

  const isGroup = node?.type === 'GROUP';
  const children = node ? allNodes.filter((n) => n.parentId === node.id) : [];
  const parentGroup = node?.parentId ? allNodes.find((n) => n.id === node.parentId) : null;

  if (!isOpen || !node || !portalTarget) return null;

  return createPortal(
    <div className="flex h-full flex-col">
      {/* Detail header with close button */}
      <div className="flex items-center justify-between border-b border-gray-100 px-4 py-2">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-gray-900">
            {isGroup ? 'Group Details' : 'Node Details'}
          </h3>
          {saveStatus === 'saving' && (
            <span className="flex items-center gap-1 text-[10px] text-gray-400">
              <Loader2 className="h-3 w-3 animate-spin" />
            </span>
          )}
          {saveStatus === 'saved' && (
            <span className="flex items-center gap-1 text-[10px] text-green-500">
              <Check className="h-3 w-3" />
              Saved
            </span>
          )}
        </div>
        <button onClick={handleClose} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
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
            onChange={(e) => {
              setTitle(e.target.value);
              debounceSave({ title: e.target.value });
            }}
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
              onChange={(e) => {
                const newType = e.target.value as NodeType;
                setType(newType);
                immediateSave({ type: newType });
              }}
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
                onClick={() => {
                  setStatus(s);
                  immediateSave({ status: s });
                }}
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

        {/* Assignee */}
        {teamMembers.length > 0 && (
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Assignee</label>
            <select
              value={node.assigneeId ?? ''}
              onChange={(e) => {
                const val = e.target.value || null;
                immediateSave({ assigneeId: val });
              }}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none"
            >
              <option value="">Unassigned</option>
              {teamMembers.map((m) => (
                <option key={m.userId} value={m.userId}>{m.displayName}</option>
              ))}
            </select>
          </div>
        )}

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

        {/* Tags */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Tags</label>
          <div className="flex flex-wrap gap-1.5 mb-1.5">
            {(node.tags ?? []).map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-700"
              >
                {tag}
                <button
                  onClick={() => {
                    const updated = (node.tags ?? []).filter((t) => t !== tag);
                    immediateSave({ tags: updated });
                  }}
                  className="ml-0.5 text-slate-400 hover:text-red-500"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
          </div>
          <input
            type="text"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && tagInput.trim()) {
                e.preventDefault();
                const current = node.tags ?? [];
                if (!current.includes(tagInput.trim())) {
                  immediateSave({ tags: [...current, tagInput.trim()] });
                }
                setTagInput('');
              }
            }}
            placeholder="Add tag + Enter"
            className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-xs focus:border-thask-primary focus:outline-none"
          />
        </div>

        {/* Description */}
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
          <textarea
            value={description}
            onChange={(e) => {
              setDescription(e.target.value);
              debounceSave({ description: e.target.value });
            }}
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
              {connectedNodeIds.map((id) => {
                const connNode = allNodes.find((n) => n.id === id);
                return (
                  <button
                    key={id}
                    onClick={() => onSelectNode?.(id)}
                    className="flex w-full items-center gap-2 rounded-lg bg-gray-50 px-3 py-1.5 text-left text-xs hover:bg-gray-100"
                  >
                    <span className="rounded bg-slate-200 px-1.5 py-0.5 text-[10px] font-medium text-slate-600">
                      {connNode?.type ?? '?'}
                    </span>
                    <span className="truncate text-gray-700">
                      {connNode?.title ?? id.slice(0, 8)}
                    </span>
                  </button>
                );
              })}
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

      {/* Delete button */}
      <div className="flex items-center border-t border-gray-200 px-4 py-2">
        <button
          onClick={() => onDelete(node.id)}
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
        >
          <Trash2 className="h-4 w-4" />
          Delete
        </button>
      </div>
    </div>,
    portalTarget,
  );
}
