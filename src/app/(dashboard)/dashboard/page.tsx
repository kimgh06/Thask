'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Plus, FolderOpen } from 'lucide-react';
import type { Team, Project } from '@/types/auth';

interface TeamWithProjects extends Team {
  projects: Project[];
}

export default function DashboardPage() {
  const [teams, setTeams] = useState<TeamWithProjects[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateTeam, setShowCreateTeam] = useState(false);
  const [teamName, setTeamName] = useState('');
  const [teamSlug, setTeamSlug] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    fetchTeams();
  }, []);

  async function fetchTeams() {
    try {
      const res = await fetch('/api/teams');
      const json = await res.json();
      if (res.ok) {
        setTeams(json.data || []);
      }
    } finally {
      setLoading(false);
    }
  }

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
        await fetchTeams();
      }
    } finally {
      setCreating(false);
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
        <div className="space-y-6">
          {teams.map((team) => (
            <div key={team.id} className="rounded-lg border border-gray-200 bg-white p-6">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold text-gray-900">{team.name}</h2>
                <Link
                  href={`/dashboard/${team.slug}`}
                  className="text-sm font-medium text-thask-primary hover:underline"
                >
                  View Projects
                </Link>
              </div>
              {team.projects && team.projects.length > 0 ? (
                <div className="grid gap-3 sm:grid-cols-2">
                  {team.projects.map((project) => (
                    <Link
                      key={project.id}
                      href={`/dashboard/${team.slug}/${project.id}`}
                      className="rounded-lg border border-gray-100 p-4 transition-colors hover:border-thask-primary hover:bg-blue-50/50"
                    >
                      <div className="font-medium text-gray-900">{project.name}</div>
                      {project.description && (
                        <div className="mt-1 truncate text-sm text-gray-500">
                          {project.description}
                        </div>
                      )}
                    </Link>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">No projects yet.</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
