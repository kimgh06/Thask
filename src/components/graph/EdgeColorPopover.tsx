'use client';

import { useEffect, useRef } from 'react';
import type { EdgeType } from '@/types/graph';

const EDGE_TYPE_OPTIONS: { value: EdgeType; label: string; color: string }[] = [
  { value: 'depends_on', label: 'Depends On', color: '#f97316' },
  { value: 'blocks', label: 'Blocks', color: '#ef4444' },
  { value: 'related', label: 'Related', color: '#6b7280' },
  { value: 'parent_child', label: 'Parent / Child', color: '#8b5cf6' },
  { value: 'triggers', label: 'Triggers', color: '#3b82f6' },
];

interface EdgeColorPopoverProps {
  position: { x: number; y: number };
  onSelect: (edgeType: EdgeType) => void;
  onDelete: () => void;
  onCancel: () => void;
}

export function EdgeColorPopover({ position, onSelect, onDelete, onCancel }: EdgeColorPopoverProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onCancel();
      }
    }

    function handleEsc(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel();
    }

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEsc);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEsc);
    };
  }, [onCancel]);

  return (
    <div
      ref={ref}
      className="fixed z-50 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg"
      style={{ left: position.x, top: position.y }}
    >
      <div className="px-3 py-1.5 text-xs font-semibold text-gray-400 uppercase">
        Edge Color
      </div>
      {EDGE_TYPE_OPTIONS.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onSelect(opt.value)}
          className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
        >
          <span
            className="inline-block h-3 w-3 rounded-full"
            style={{ backgroundColor: opt.color }}
          />
          {opt.label}
        </button>
      ))}
      <div className="mx-2 my-1 border-t border-gray-200" />
      <button
        onClick={onDelete}
        className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
      >
        <svg className="h-3 w-3" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5">
          <path d="M2 3h8M4.5 3V2a.5.5 0 0 1 .5-.5h2a.5.5 0 0 1 .5.5v1M9 3v6.5a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V3" />
        </svg>
        Delete
      </button>
    </div>
  );
}
