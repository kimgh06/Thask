'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import { Plus, FolderOpen, Pencil, Trash2, Bug, Box, AlertTriangle, Clock } from 'lucide-react';
import type { Team, Project } from '@/types/auth';
import { DropdownMenu } from '@/components/ui/DropdownMenu';
import { EditProjectModal } from '@/components/ui/EditProjectModal';
import { DangerConfirmDialog } from '@/components/ui/DangerConfirmDialog';

interface ProjectWithStats extends Project {
  nodeCount: number;
  bugCount: number;
}

interface TeamWithProjects extends Team {
  projects: ProjectWithStats[];
}

interface SummaryStats {
  totalNodes: number;
  totalBugs: number;
  failCount: number;
  blockedCount: number;
}

export default function DashboardPage() {
  const [teams, setTeams] = useState<TeamWithProjects[]>([]);
  const [stats, setStats] = useState<SummaryStats>({ totalNodes: 0, totalBugs: 0, failCount: 0, blockedCount: 0 });
  const [loading, setLoading] = useState(true);
  const [showCreateTeam, setShowCreateTeam] = useState(false);
  const [teamName, setTeamName] = useState('');
  const [teamSlug, setTeamSlug] = useState('');
  const [creating, setCreating] = useState(false);

  // Edit/Delete state
  const [editingProject, setEditingProject] = useState<{ project: ProjectWithStats; teamSlug: string } | null>(null);
  const [deletingProject, setDeletingProject] = useState<{ project: ProjectWithStats; teamSlug: string } | null>(null);
  const [deletingTeam, setDeletingTeam] = useState<Team | null>(null);
  const [saving, setSaving] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const res = await fetch('/api/projects/summary');
      const json = await res.json();
      if (res.ok) {
        setTeams(json.data?.teams || []);
        setStats(json.data?.stats || { totalNodes: 0, totalBugs: 0, failCount: 0, blockedCount: 0 });
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleCreateTeam(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      const res = await fetch('/api/teams', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: teamName, slug: teamSlug }),
      });
      if (res.ok) {
        setTeamName('');
        setTeamSlug('');
        setShowCreateTeam(false);
        await fetchData();
      }
    } finally {
      setCreating(false);
    }
  }

  async function handleEditProject(data: { name: string; description?: string }) {
    if (!editingProject) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/teams/${editingProject.teamSlug}/projects/${editingProject.project.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (res.ok) {
        setEditingProject(null);
        await fetchData();
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteProject() {
    if (!deletingProject) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/teams/${deletingProject.teamSlug}/projects/${deletingProject.project.id}`, {
        method: 'DELETE',
      });
      if (res.ok) {
        setDeletingProject(null);
        await fetchData();
      }
    } finally {
      setSaving(false);
    }
  }

  async function handleDeleteTeam() {
    if (!deletingTeam) return;
    setSaving(true);
    try {
      const res = await fetch(`/api/teams/${deletingTeam.slug}`, { method: 'DELETE' });
      if (res.ok) {
        setDeletingTeam(null);
        await fetchData();
      }
    } finally {
      setSaving(false);
    }
  }

  // Get recent projects (top 4 across all teams, sorted by updatedAt)
  const recentProjects = teams
    .flatMap((team) => team.projects.map((p) => ({ ...p, teamSlug: team.slug, teamName: team.name })))
    .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
    .slice(0, 4);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-thask-primary" />
      </div>
    );
  }

  const totalProjects = teams.reduce((sum, t) => sum + t.projects.length, 0);

  return (
    <div className="mx-auto max-w-5xl">
      {/* Header */}
      <div className="mb-8 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <button
          onClick={() => setShowCreateTeam(true)}
          className="flex items-center gap-2 rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-600"
        >
          <Plus className="h-4 w-4" />
          New Team
        </button>
      </div>

      {/* Create Team Form */}
      {showCreateTeam && (
        <div className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Create New Team</h2>
          <form onSubmit={handleCreateTeam} className="space-y-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Team Name</label>
              <input
                type="text"
                value={teamName}
                onChange={(e) => {
                  setTeamName(e.target.value);
                  setTeamSlug(
                    e.target.value
                      .toLowerCase()
                      .replace(/[^a-z0-9]+/g, '-')
                      .replace(/^-|-$/g, ''),
                  );
                }}
                required
                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
                placeholder="My Team"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Slug (URL)</label>
              <input
                type="text"
                value={teamSlug}
                onChange={(e) => setTeamSlug(e.target.value)}
                required
                pattern="^[a-z0-9-]+$"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-thask-primary focus:outline-none focus:ring-2 focus:ring-thask-primary/20"
                placeholder="my-team"
              />
            </div>
            <div className="flex gap-3">
              <button
                type="submit"
                disabled={creating}
                className="rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white hover:bg-blue-600 disabled:opacity-50"
              >
                {creating ? 'Creating...' : 'Create Team'}
              </button>
              <button
                type="button"
                onClick={() => setShowCreateTeam(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {teams.length === 0 && !showCreateTeam ? (
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-12 text-center">
          <FolderOpen className="mx-auto mb-4 h-12 w-12 text-gray-400" />
          <h2 className="mb-2 text-lg font-semibold text-gray-900">No teams yet</h2>
          <p className="mb-4 text-gray-500">Create a team to start organizing your projects.</p>
          <button
            onClick={() => setShowCreateTeam(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-thask-primary px-4 py-2 text-sm font-medium text-white hover:bg-blue-600"
          >
            <Plus className="h-4 w-4" />
            Create Your First Team
          </button>
        </div>
      ) : (
        <>
          {/* Summary Stats */}
          {totalProjects > 0 && (
            <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div className="rounded-lg border border-gray-200 bg-white p-4">
                <div className="flex items-center gap-2 text-xs font-medium text-gray-500">
                  <Box className="h-3.5 w-3.5" />
                  Total Nodes
                </div>
                <div className="mt-1 text-2xl font-bold text-gray-900">{stats.totalNodes}</div>
              </div>
              <div className="rounded-lg border border-gray-200 bg-white p-4">
                <div className="flex items-center gap-2 text-xs font-medium text-gray-500">
                  <Bug className="h-3.5 w-3.5" />
                  Bugs
                </div>
                <div className="mt-1 text-2xl font-bold text-red-600">{stats.totalBugs}</div>
              </div>
              <div className="rounded-lg border border-gray-200 bg-white p-4">
                <div className="flex items-center gap-2 text-xs font-medium text-gray-500">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  Failing
                </div>
                <div className="mt-1 text-2xl font-bold text-orange-600">{stats.failCount}</div>
              </div>
              <div className="rounded-lg border border-gray-200 bg-white p-4">
                <div className="flex items-center gap-2 text-xs font-medium text-gray-500">
                  <Clock className="h-3.5 w-3.5" />
                  Blocked
                </div>
                <div className="mt-1 text-2xl font-bold text-gray-600">{stats.blockedCount}</div>
              </div>
            </div>
          )}

          {/* Recent Projects */}
          {recentProjects.length > 0 && (
            <div className="mb-8">
              <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-gray-500">Recent Projects</h2>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                {recentProjects.map((project) => (
                  <Link
                    key={project.id}
                    href={`/dashboard/${project.teamSlug}/${project.id}`}
                    className="rounded-lg border border-gray-200 bg-white p-4 transition-all hover:border-thask-primary hover:shadow-md"
                  >
                    <div className="text-sm font-medium text-gray-900">{project.name}</div>
                    <div className="mt-1 text-xs text-gray-400">{project.teamName}</div>
                    <div className="mt-2 flex items-center gap-3 text-xs text-gray-500">
                      <span>{project.nodeCount} nodes</span>
                      {project.bugCount > 0 && (
                        <span className="text-red-500">{project.bugCount} bugs</span>
                      )}
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          )}

          {/* Teams & Projects */}
          <div className="space-y-6">
            {teams.map((team) => (
              <div key={team.id} className="rounded-lg border border-gray-200 bg-white p-6">
                <div className="mb-4 flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <h2 className="text-lg font-semibold text-gray-900">{team.name}</h2>
                    <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
                      {team.projects.length} projects
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Link
                      href={`/dashboard/${team.slug}`}
                      className="text-sm font-medium text-thask-primary hover:underline"
                    >
                      View All
                    </Link>
                    <DropdownMenu
                      items={[
                        {
                          label: 'Delete Team',
                          icon: <Trash2 className="h-4 w-4" />,
                          variant: 'danger',
                          separator: true,
                          onClick: () => setDeletingTeam(team),
                        },
                      ]}
                    />
                  </div>
                </div>
                {team.projects.length > 0 ? (
                  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    {team.projects.map((project) => (
                      <div
                        key={project.id}
                        className="group relative rounded-lg border border-gray-100 p-4 transition-colors hover:border-thask-primary hover:bg-blue-50/50"
                      >
                        <Link href={`/dashboard/${team.slug}/${project.id}`} className="block">
                          <div className="pr-8 font-medium text-gray-900">{project.name}</div>
                          {project.description && (
                            <div className="mt-1 truncate text-sm text-gray-500">
                              {project.description}
                            </div>
                          )}
                          <div className="mt-2 flex items-center gap-3 text-xs text-gray-400">
                            <span>{project.nodeCount} nodes</span>
                            {project.bugCount > 0 && (
                              <span className="text-red-500">{project.bugCount} bugs</span>
                            )}
                          </div>
                        </Link>
                        <div className="absolute right-3 top-3">
                          <DropdownMenu
                            items={[
                              {
                                label: 'Edit',
                                icon: <Pencil className="h-4 w-4" />,
                                onClick: () => setEditingProject({ project, teamSlug: team.slug }),
                              },
                              {
                                label: 'Delete',
                                icon: <Trash2 className="h-4 w-4" />,
                                variant: 'danger',
                                separator: true,
                                onClick: () => setDeletingProject({ project, teamSlug: team.slug }),
                              },
                            ]}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-gray-500">No projects yet.</p>
                )}
              </div>
            ))}
          </div>
        </>
      )}

      {/* Edit Project Modal */}
      {editingProject && (
        <EditProjectModal
          name={editingProject.project.name}
          description={editingProject.project.description}
          onSave={handleEditProject}
          onCancel={() => setEditingProject(null)}
          saving={saving}
        />
      )}

      {/* Delete Project Dialog */}
      {deletingProject && (
        <DangerConfirmDialog
          title="Delete Project"
          message={`This will permanently delete "${deletingProject.project.name}" and all its nodes, edges, and history.`}
          confirmText={deletingProject.project.name}
          onConfirm={handleDeleteProject}
          onCancel={() => setDeletingProject(null)}
        />
      )}

      {/* Delete Team Dialog */}
      {deletingTeam && (
        <DangerConfirmDialog
          title="Delete Team"
          message={`This will permanently delete "${deletingTeam.name}" and all its projects, nodes, and data.`}
          confirmText={deletingTeam.name}
          onConfirm={handleDeleteTeam}
          onCancel={() => setDeletingTeam(null)}
        />
      )}
    </div>
  );
}
