'use client';

import { useState, useEffect, useRef } from 'react';

interface EditProjectModalProps {
  name: string;
  description: string | null;
  onSave: (data: { name: string; description?: string }) => void;
  onCancel: () => void;
  saving?: boolean;
}

export function EditProjectModal({ name, description, onSave, onCancel, saving }: EditProjectModalProps) {
  const [editName, setEditName] = useState(name);
  const [editDesc, setEditDesc] = useState(description ?? '');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel();
    }
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, [onCancel]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!editName.trim()) return;
    onSave({ name: editName.trim(), description: editDesc.trim() || undefined });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-xl bg-white p-5 shadow-2xl">
        <h3 className="mb-4 text-sm font-semibold text-gray-900">Edit Project</h3>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-700">Project Name</label>
            <input
              ref={inputRef}
              type="text"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              required
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-700">Description</label>
            <textarea
              value={editDesc}
              onChange={(e) => setEditDesc(e.target.value)}
              rows={2}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
            />
          </div>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onCancel}
              className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving || !editName.trim()}
              className="rounded-lg bg-thask-primary px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-600 disabled:opacity-50"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
