'use client';

import { useState } from 'react';
import { X } from 'lucide-react';
import type { NodeType } from '@/types/graph';

interface AddNodeModalProps {
  onSubmit: (data: { title: string; type: NodeType }) => void;
  onClose: () => void;
}

const MODAL_NODE_TYPES = ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI'] as const;

const TYPE_LABELS: Record<NodeType, string> = {
  FLOW: 'Flow',
  BRANCH: 'Branch',
  TASK: 'Task',
  BUG: 'Bug',
  API: 'API',
  UI: 'UI',
  GROUP: 'Group',
};

const TYPE_DESCRIPTIONS: Record<NodeType, string> = {
  FLOW: 'A feature flow or user journey',
  BRANCH: 'A decision branch or business rule',
  TASK: 'A development task or work item',
  BUG: 'A bug report or known issue',
  API: 'An API endpoint or integration',
  UI: 'A UI component or screen',
  GROUP: 'A group container for related nodes',
};

export function AddNodeModal({ onSubmit, onClose }: AddNodeModalProps) {
  const [title, setTitle] = useState('');
  const [type, setType] = useState<NodeType>('TASK');

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    onSubmit({ title: title.trim(), type });
    onClose();
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Add Node</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
              autoFocus
              className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
              placeholder="Enter node title"
            />
          </div>

          <div>
            <label className="mb-2 block text-sm font-medium text-gray-700">Type</label>
            <div className="grid grid-cols-3 gap-2">
              {MODAL_NODE_TYPES.map((t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setType(t)}
                  className={`rounded-lg border p-2 text-center transition-colors ${
                    type === t
                      ? 'border-thask-primary bg-blue-50 text-thask-primary'
                      : 'border-gray-200 text-gray-600 hover:border-gray-300'
                  }`}
                >
                  <div className="text-sm font-medium">{TYPE_LABELS[t]}</div>
                  <div className="mt-0.5 text-[10px] text-gray-400">{TYPE_DESCRIPTIONS[t]}</div>
                </button>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white hover:bg-blue-600"
            >
              Add Node
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
