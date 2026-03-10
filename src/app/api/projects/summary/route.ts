import { NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { teams, teamMembers, projects, nodes } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { eq, inArray, sql } from 'drizzle-orm';

export async function GET() {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const memberships = await db
    .select({ teamId: teamMembers.teamId })
    .from(teamMembers)
    .where(eq(teamMembers.userId, auth.userId));

  if (memberships.length === 0) {
    return NextResponse.json({ data: { teams: [], stats: { totalNodes: 0, totalBugs: 0, failCount: 0, blockedCount: 0 } } });
  }

  const teamIds = memberships.map((m) => m.teamId);

  const userTeams = await db.select().from(teams).where(inArray(teams.id, teamIds));

  const teamProjects = await db
    .select()
    .from(projects)
    .where(inArray(projects.teamId, teamIds));

  const projectIds = teamProjects.map((p) => p.id);

  let stats = { totalNodes: 0, totalBugs: 0, failCount: 0, blockedCount: 0 };
  let projectNodeCounts: Record<string, { total: number; bugs: number }> = {};

  if (projectIds.length > 0) {
    const nodeStats = await db
      .select({
        projectId: nodes.projectId,
        total: sql<number>`count(*)::int`,
        bugs: sql<number>`count(*) filter (where ${nodes.type} = 'BUG')::int`,
        failCount: sql<number>`count(*) filter (where ${nodes.status} = 'FAIL')::int`,
        blockedCount: sql<number>`count(*) filter (where ${nodes.status} = 'BLOCKED')::int`,
      })
      .from(nodes)
      .where(inArray(nodes.projectId, projectIds))
      .groupBy(nodes.projectId);

    for (const row of nodeStats) {
      projectNodeCounts[row.projectId] = { total: row.total, bugs: row.bugs };
      stats.totalNodes += row.total;
      stats.totalBugs += row.bugs;
      stats.failCount += row.failCount;
      stats.blockedCount += row.blockedCount;
    }
  }

  const teamsWithProjects = userTeams.map((team) => ({
    ...team,
    projects: teamProjects
      .filter((p) => p.teamId === team.id)
      .map((p) => ({
        ...p,
        nodeCount: projectNodeCounts[p.id]?.total ?? 0,
        bugCount: projectNodeCounts[p.id]?.bugs ?? 0,
      })),
  }));

  // Sort: most recently updated projects first
  teamsWithProjects.forEach((t) => {
    t.projects.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
  });

  return NextResponse.json({ data: { teams: teamsWithProjects, stats } });
}
