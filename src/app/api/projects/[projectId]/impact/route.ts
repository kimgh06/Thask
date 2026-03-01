import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { nodes, edges, projects, teamMembers } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { eq, and, gte, or, inArray } from 'drizzle-orm';

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;

  const access = await db
    .select({ id: projects.id })
    .from(projects)
    .innerJoin(
      teamMembers,
      and(eq(teamMembers.teamId, projects.teamId), eq(teamMembers.userId, auth.userId)),
    )
    .where(eq(projects.id, projectId))
    .limit(1);

  if (access.length === 0) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  const url = new URL(request.url);
  const since = url.searchParams.get('since');
  const depth = parseInt(url.searchParams.get('depth') ?? '2', 10);

  const sinceDate = since ? new Date(since) : new Date(Date.now() - 7 * 24 * 60 * 60 * 1000);

  // Step 1: Find recently changed nodes
  const changedNodes = await db
    .select()
    .from(nodes)
    .where(and(eq(nodes.projectId, projectId), gte(nodes.updatedAt, sinceDate)));

  if (changedNodes.length === 0) {
    // No changes, still return FAIL/BUG nodes
    const failNodes = await db
      .select()
      .from(nodes)
      .where(
        and(
          eq(nodes.projectId, projectId),
          or(eq(nodes.status, 'FAIL'), eq(nodes.type, 'BUG')),
        ),
      );

    return NextResponse.json({
      data: {
        changedNodes: [],
        impactedNodes: [],
        failNodes,
        impactEdges: [],
      },
    });
  }

  // Step 2: BFS to find impacted nodes up to N depth
  const changedIds = new Set(changedNodes.map((n) => n.id));
  const allImpactedIds = new Set(changedIds);
  let frontier = Array.from(changedIds);

  const allEdges = await db
    .select()
    .from(edges)
    .where(eq(edges.projectId, projectId));

  for (let d = 0; d < depth && frontier.length > 0; d++) {
    const nextFrontier: string[] = [];
    for (const edge of allEdges) {
      if (frontier.includes(edge.sourceId) && !allImpactedIds.has(edge.targetId)) {
        allImpactedIds.add(edge.targetId);
        nextFrontier.push(edge.targetId);
      }
      if (frontier.includes(edge.targetId) && !allImpactedIds.has(edge.sourceId)) {
        allImpactedIds.add(edge.sourceId);
        nextFrontier.push(edge.sourceId);
      }
    }
    frontier = nextFrontier;
  }

  // Step 3: Fetch impacted nodes (excluding changed ones, which we already have)
  const impactedOnlyIds = Array.from(allImpactedIds).filter((id) => !changedIds.has(id));
  const impactedNodes =
    impactedOnlyIds.length > 0
      ? await db.select().from(nodes).where(inArray(nodes.id, impactedOnlyIds))
      : [];

  // Step 4: FAIL/BUG nodes in project
  const failNodes = await db
    .select()
    .from(nodes)
    .where(
      and(
        eq(nodes.projectId, projectId),
        or(eq(nodes.status, 'FAIL'), eq(nodes.type, 'BUG')),
      ),
    );

  // Step 5: Edges within the impact subgraph
  const impactEdges = allEdges.filter(
    (e) => allImpactedIds.has(e.sourceId) && allImpactedIds.has(e.targetId),
  );

  return NextResponse.json({
    data: {
      changedNodes,
      impactedNodes,
      failNodes,
      impactEdges,
    },
  });
}
