'use client';

import { useState, useEffect, useLayoutEffect, useRef, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { X, Trash2, Check } from 'lucide-react';
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

const TYPE_BADGE_COLORS: Record<NodeType, string> = {
  FLOW: 'bg-blue-100 text-blue-700',
  BRANCH: 'bg-violet-100 text-violet-700',
  TASK: 'bg-cyan-100 text-cyan-700',
  BUG: 'bg-red-100 text-red-700',
  API: 'bg-orange-100 text-orange-700',
  UI: 'bg-emerald-100 text-emerald-700',
  GROUP: 'bg-slate-100 text-slate-600',
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

const STATUS_DOT: Record<NodeStatus, string> = {
  PASS: 'bg-green-500',
  FAIL: 'bg-red-500',
  IN_PROGRESS: 'bg-yellow-500',
  BLOCKED: 'bg-gray-400',
};

type TabId = 'details' | 'relations' | 'history';

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
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saved'>('idle');
  const [tagInput, setTagInput] = useState('');
  const [portalTarget, setPortalTarget] = useState<HTMLElement | null>(null);
  const [activeTab, setActiveTab] = useState<TabId>('details');
  const [showTypeDropdown, setShowTypeDropdown] = useState(false);
  const [showStatusDropdown, setShowStatusDropdown] = useState(false);

  const nodeIdRef = useRef<string | null>(null);
  const savedTitleRef = useRef('');
  const savedDescRef = useRef('');
  const titleRef = useRef('');
  titleRef.current = title;
  const descRef = useRef('');
  descRef.current = description;
  const savedIndicatorTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const typeDropdownRef = useRef<HTMLDivElement>(null);
  const statusDropdownRef = useRef<HTMLDivElement>(null);

  function showSaved() {
    setSaveStatus('saved');
    if (savedIndicatorTimer.current) clearTimeout(savedIndicatorTimer.current);
    savedIndicatorTimer.current = setTimeout(() => setSaveStatus('idle'), 1500);
  }

  // Save dirty text fields — called on blur, node switch, panel close
  const flushText = useCallback(() => {
    if (!nodeIdRef.current) return;
    const dirty: Partial<GraphNode> = {};
    if (savedTitleRef.current !== titleRef.current) dirty.title = titleRef.current;
    if (savedDescRef.current !== descRef.current) dirty.description = descRef.current;
    if (Object.keys(dirty).length > 0) {
      onUpdate(nodeIdRef.current, dirty);
      if (dirty.title != null) savedTitleRef.current = dirty.title;
      if (dirty.description != null) savedDescRef.current = dirty.description ?? '';
      showSaved();
    }
  }, [onUpdate]);

  // Immediate save for discrete fields (status, type, tags, assignee)
  function immediateSave(fields: Partial<GraphNode>) {
    if (!nodeIdRef.current) return;
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

  // Flush & reset all local state when selecting a DIFFERENT node
  useEffect(() => {
    if (nodeIdRef.current && nodeIdRef.current !== node?.id) {
      flushText();
    }

    if (node) {
      nodeIdRef.current = node.id;
      setTitle(node.title);
      setDescription(node.description ?? '');
      setType(node.type);
      setStatus(node.status);
      savedTitleRef.current = node.title;
      savedDescRef.current = node.description ?? '';
      setSaveStatus('idle');
      setActiveTab('details');
    } else {
      nodeIdRef.current = null;
    }
    // Only reset when the selected node ID changes (not on every data update)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [node?.id, flushText]);

  // Sync discrete fields (status, type) from external changes
  // (e.g. waterfall propagation, undo/redo) without resetting text fields
  useEffect(() => {
    if (node && node.id === nodeIdRef.current) {
      setStatus(node.status);
      setType(node.type);
    }
  }, [node?.status, node?.type, node?.id]);

  // Flush on unmount
  useEffect(() => {
    return () => {
      // Save any unsaved text on unmount
      if (nodeIdRef.current) {
        const dirty: Partial<GraphNode> = {};
        if (savedTitleRef.current !== titleRef.current) dirty.title = titleRef.current;
        if (savedDescRef.current !== descRef.current) dirty.description = descRef.current;
        if (Object.keys(dirty).length > 0) {
          onUpdate(nodeIdRef.current, dirty);
        }
      }
      if (savedIndicatorTimer.current) clearTimeout(savedIndicatorTimer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Close dropdowns on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (typeDropdownRef.current && !typeDropdownRef.current.contains(e.target as Node)) {
        setShowTypeDropdown(false);
      }
      if (statusDropdownRef.current && !statusDropdownRef.current.contains(e.target as Node)) {
        setShowStatusDropdown(false);
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  function handleClose() {
    flushText();
    onClose();
  }

  const isGroup = node?.type === 'GROUP';
  const children = node ? allNodes.filter((n) => n.parentId === node.id) : [];
  const parentGroup = node?.parentId ? allNodes.find((n) => n.id === node.parentId) : null;

  if (!isOpen || !node || !portalTarget) return null;

  const tabs: { id: TabId; label: string; count?: number }[] = [
    { id: 'details', label: 'Details' },
    { id: 'relations', label: 'Relations', count: connectedNodeIds.length + children.length },
    { id: 'history', label: 'History', count: history.length },
  ];

  return createPortal(
    <div className="flex h-full flex-col">
      {/* ── Header: close + save indicator ── */}
      <div className="flex items-center justify-between border-b border-gray-100 px-4 py-2">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-gray-900">
            {isGroup ? 'Group' : 'Node'}
          </h3>
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

      {/* ── Fixed top: Title + Type/Status row + Assignee ── */}
      <div className="space-y-3 border-b border-gray-100 px-4 py-3">
        {/* Title — inline edit */}
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          onBlur={() => {
            if (title !== savedTitleRef.current) {
              immediateSave({ title });
              savedTitleRef.current = title;
            }
          }}
          className="w-full bg-transparent text-sm font-semibold text-gray-900 placeholder-gray-400 outline-none focus:border-b focus:border-thask-primary"
          placeholder="Untitled"
        />

        {/* Type + Status badges — single row */}
        <div className="flex items-center gap-2">
          {/* Type badge/dropdown */}
          <div className="relative" ref={typeDropdownRef}>
            {isGroup && children.length > 0 ? (
              <span className={cn('rounded-full px-2.5 py-0.5 text-[11px] font-medium', TYPE_BADGE_COLORS.GROUP)}>
                Group
              </span>
            ) : (
              <>
                <button
                  onClick={() => { setShowTypeDropdown(!showTypeDropdown); setShowStatusDropdown(false); }}
                  className={cn('rounded-full px-2.5 py-0.5 text-[11px] font-medium transition-colors hover:opacity-80', TYPE_BADGE_COLORS[type])}
                >
                  {TYPE_LABELS[type]}
                </button>
                {showTypeDropdown && (
                  <div className="absolute left-0 top-full z-50 mt-1 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                    {NODE_TYPES.map((t) => (
                      <button
                        key={t}
                        onClick={() => {
                          setType(t);
                          immediateSave({ type: t });
                          setShowTypeDropdown(false);
                        }}
                        className={cn(
                          'flex w-full items-center gap-2 px-3 py-1.5 text-xs hover:bg-gray-50',
                          type === t && 'bg-gray-50 font-medium',
                        )}
                      >
                        <span className={cn('h-2 w-2 rounded-full', TYPE_BADGE_COLORS[t].replace('text-', 'bg-').split(' ')[0])} />
                        {TYPE_LABELS[t]}
                      </button>
                    ))}
                  </div>
                )}
              </>
            )}
          </div>

          <span className="text-gray-300">|</span>

          {/* Status badge/dropdown */}
          <div className="relative" ref={statusDropdownRef}>
            <button
              onClick={() => { setShowStatusDropdown(!showStatusDropdown); setShowTypeDropdown(false); }}
              className={cn('flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium transition-colors hover:opacity-80', STATUS_COLORS[status])}
            >
              <span className={cn('h-1.5 w-1.5 rounded-full', STATUS_DOT[status])} />
              {STATUS_LABELS[status]}
            </button>
            {showStatusDropdown && (
              <div className="absolute left-0 top-full z-50 mt-1 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                {NODE_STATUSES.map((s) => (
                  <button
                    key={s}
                    onClick={() => {
                      setStatus(s);
                      immediateSave({ status: s });
                      setShowStatusDropdown(false);
                    }}
                    className={cn(
                      'flex w-full items-center gap-2 px-3 py-1.5 text-xs hover:bg-gray-50',
                      status === s && 'bg-gray-50 font-medium',
                    )}
                  >
                    <span className={cn('h-2 w-2 rounded-full', STATUS_DOT[s])} />
                    {STATUS_LABELS[s]}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Assignee — compact */}
        {teamMembers.length > 0 && (
          <select
            value={node.assigneeId ?? ''}
            onChange={(e) => {
              const val = e.target.value || null;
              immediateSave({ assigneeId: val });
            }}
            className="w-full rounded-lg border border-gray-200 bg-gray-50 px-2.5 py-1.5 text-xs text-gray-700 focus:border-thask-primary focus:outline-none"
          >
            <option value="">Unassigned</option>
            {teamMembers.map((m) => (
              <option key={m.userId} value={m.userId}>{m.displayName}</option>
            ))}
          </select>
        )}

        {/* Group membership banner */}
        {node.parentId && (
          <div className="flex items-center justify-between rounded-lg bg-slate-50 px-2.5 py-1.5">
            <span className="text-[11px] text-gray-500">
              In{' '}
              <button
                onClick={() => parentGroup && onSelectNode?.(parentGroup.id)}
                className="font-medium text-gray-700 hover:text-thask-primary"
              >
                {parentGroup?.title ?? node.parentId.slice(0, 8)}
              </button>
            </span>
            <button
              onClick={() => { onUpdate(node.id, { parentId: null }); }}
              className="rounded px-1.5 py-0.5 text-[10px] font-medium text-red-500 hover:bg-red-50"
            >
              Remove
            </button>
          </div>
        )}
      </div>

      {/* ── Tab bar ── */}
      <div className="flex border-b border-gray-100 px-4">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={cn(
              'relative px-3 py-2 text-xs font-medium transition-colors',
              activeTab === tab.id
                ? 'text-thask-primary'
                : 'text-gray-400 hover:text-gray-600',
            )}
          >
            {tab.label}
            {tab.count != null && tab.count > 0 && (
              <span className="ml-1 text-[10px] text-gray-400">{tab.count}</span>
            )}
            {activeTab === tab.id && (
              <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-thask-primary rounded-full" />
            )}
          </button>
        ))}
      </div>

      {/* ── Tab content (scrollable) ── */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Details tab */}
        {activeTab === 'details' && (
          <>
            {/* Description */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                onBlur={() => {
                  if (description !== savedDescRef.current) {
                    immediateSave({ description });
                    savedDescRef.current = description;
                  }
                }}
                rows={3}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
                placeholder="Add a description..."
              />
            </div>

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
                className="w-full rounded-lg border border-gray-200 px-3 py-1.5 text-xs focus:border-thask-primary focus:outline-none"
              />
            </div>
          </>
        )}

        {/* Relations tab */}
        {activeTab === 'relations' && (
          <>
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
                        <span className={cn('rounded px-1.5 py-0.5 text-[10px] font-medium', TYPE_BADGE_COLORS[child.type])}>
                          {child.type}
                        </span>
                        <span className="truncate text-gray-700">{child.title}</span>
                        <span className={cn('ml-auto h-1.5 w-1.5 rounded-full', STATUS_DOT[child.status])} />
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="text-xs text-gray-400">Drag nodes into this group.</p>
                )}
              </div>
            )}

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
                        <span className={cn('rounded px-1.5 py-0.5 text-[10px] font-medium', TYPE_BADGE_COLORS[connNode?.type ?? 'TASK'])}>
                          {connNode?.type ?? '?'}
                        </span>
                        <span className="truncate text-gray-700">
                          {connNode?.title ?? id.slice(0, 8)}
                        </span>
                        {connNode && (
                          <span className={cn('ml-auto h-1.5 w-1.5 rounded-full', STATUS_DOT[connNode.status])} />
                        )}
                      </button>
                    );
                  })}
                </div>
              ) : (
                <p className="text-xs text-gray-400">No connections yet.</p>
              )}
            </div>
          </>
        )}

        {/* History tab */}
        {activeTab === 'history' && (
          <div>
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
        )}
      </div>

      {/* ── Delete button ── */}
      <div className="flex items-center border-t border-gray-200 px-4 py-2">
        <button
          onClick={() => onDelete(node.id)}
          className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs text-red-500 hover:bg-red-50"
        >
          <Trash2 className="h-3.5 w-3.5" />
          Delete
        </button>
      </div>
    </div>,
    portalTarget,
  );
}
