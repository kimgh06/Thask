'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Plus, ArrowLeft, Pencil, Trash2 } from 'lucide-react';
import Link from 'next/link';
import type { Project } from '@/types/auth';
import { DropdownMenu } from '@/components/ui/DropdownMenu';
import { EditProjectModal } from '@/components/ui/EditProjectModal';
import { DangerConfirmDialog } from '@/components/ui/DangerConfirmDialog';

export default function TeamProjectsPage() {
  const params = useParams<{ teamSlug: string }>();
  const router = useRouter();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [projectName, setProjectName] = useState('');
  const [projectDesc, setProjectDesc] = useState('');
  const [creating, setCreating] = useState(false);

  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [deletingProject, setDeletingProject] = useState<Project | null>(null);
  const [saving, setSaving] = useState(false);

  const fetchProjects = useCallback(async () => {
    try {
      const res = await fetch(`/api/teams/${params.teamSlug}/projects`);
      const json = await res.json();
      if (res.ok) setProjects(json.data || []);
    } finally {
      setLoading(false);
    }
  }, [params.teamSlug]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      const res = await fetch(`/api/teams/${params.teamSlug}/projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: projectName, description: projectDesc || undefined }),
      });
      if (res.ok) {
        const json = await res.json();
        router.push(`/dashboard/${params.teamSlug}/${json.data.id}`);
      }
    } finally {
      setCreating(false);
    }
  }

  async function handleEditProject(data: { name: string; description?: string }) {
    if (!editingProject) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/teams/${params.teamSlug}/projects/${editingProject.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (res.ok) {
        setEditingProject(null);
        await fetchProjects();
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteProject() {
    if (!deletingProject) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/teams/${params.teamSlug}/projects/${deletingProject.id}`, {
        method: 'DELETE',
      });
      if (res.ok) {
        setDeletingProject(null);
        await fetchProjects();
      }
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-thask-primary" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-6">
        <Link href="/dashboard" className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700">
          <ArrowLeft className="h-4 w-4" />
          Back to Dashboard
        </Link>
      </div>

      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Projects</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white hover:bg-blue-600"
        >
          <Plus className="h-4 w-4" />
          New Project
        </button>
      </div>

      {showCreate && (
        <div className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Create Project</h2>
          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Project Name</label>
              <input
                type="text"
                value={projectName}
                onChange={(e) => setProjectName(e.target.value)}
                required
                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
                placeholder="My Project"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Description (optional)</label>
              <textarea
                value={projectDesc}
                onChange={(e) => setProjectDesc(e.target.value)}
                rows={2}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
                placeholder="Brief description"
              />
            </div>
            <div className="flex gap-3">
              <button type="submit" disabled={creating} className="rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white hover:bg-blue-600 disabled:opacity-50">
                {creating ? 'Creating...' : 'Create Project'}
              </button>
              <button type="button" onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {projects.length === 0 && !showCreate ? (
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center">
          <h2 className="mb-2 text-lg font-semibold text-gray-900">No projects yet</h2>
          <p className="mb-4 text-gray-500">Create a project to start building your graph.</p>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white hover:bg-blue-600"
          >
            <Plus className="h-4 w-4" />
            Create First Project
          </button>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {projects.map((project) => (
            <div
              key={project.id}
              className="group relative rounded-lg border border-gray-200 bg-white p-5 transition-all hover:border-thask-primary hover:shadow-md"
            >
              <Link href={`/dashboard/${params.teamSlug}/${project.id}`} className="block">
                <div className="pr-8 text-lg font-medium text-gray-900">{project.name}</div>
                {project.description && (
                  <div className="mt-1 text-sm text-gray-500">{project.description}</div>
                )}
                <div className="mt-3 text-xs text-gray-400">
                  Created {new Date(project.createdAt).toLocaleDateString()}
                </div>
              </Link>
              <div className="absolute right-4 top-4">
                <DropdownMenu
                  items={[
                    {
                      label: 'Edit',
                      icon: <Pencil className="h-4 w-4" />,
                      onClick: () => setEditingProject(project),
                    },
                    {
                      label: 'Delete',
                      icon: <Trash2 className="h-4 w-4" />,
                      variant: 'danger',
                      separator: true,
                      onClick: () => setDeletingProject(project),
                    },
                  ]}
                />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Edit Project Modal */}
      {editingProject && (
        <EditProjectModal
          name={editingProject.name}
          description={editingProject.description}
          onSave={handleEditProject}
          onCancel={() => setEditingProject(null)}
          saving={saving}
        />
      )}

      {/* Delete Project Dialog */}
      {deletingProject && (
        <DangerConfirmDialog
          title="Delete Project"
          message={`This will permanently delete "${deletingProject.name}" and all its nodes, edges, and history.`}
          confirmText={deletingProject.name}
          onConfirm={handleDeleteProject}
          onCancel={() => setDeletingProject(null)}
        />
      )}
    </div>
  );
}
