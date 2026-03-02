import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { nodes, edges, nodeHistory, projects, teamMembers, users } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { updateNodeSchema } from '@/lib/validators';
import { eq, and, or, desc, inArray } from 'drizzle-orm';
import { computeWaterfall } from '@/lib/waterfall';
import type { WaterfallNode, WaterfallEdge } from '@/lib/waterfall';

async function verifyProjectAccess(projectId: string, userId: string) {
  const result = await db
    .select({ id: projects.id })
    .from(projects)
    .innerJoin(
      teamMembers,
      and(eq(teamMembers.teamId, projects.teamId), eq(teamMembers.userId, userId)),
    )
    .where(eq(projects.id, projectId))
    .limit(1);
  return result.length > 0;
}

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ projectId: string; nodeId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId, nodeId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  const [node] = await db
    .select()
    .from(nodes)
    .where(and(eq(nodes.id, nodeId), eq(nodes.projectId, projectId)))
    .limit(1);

  if (!node) {
    return NextResponse.json({ error: 'Node not found' }, { status: 404 });
  }

  const connectedEdges = await db
    .select()
    .from(edges)
    .where(or(eq(edges.sourceId, nodeId), eq(edges.targetId, nodeId)));

  const connectedNodeIds = new Set<string>();
  connectedEdges.forEach((e) => {
    if (e.sourceId !== nodeId) connectedNodeIds.add(e.sourceId);
    if (e.targetId !== nodeId) connectedNodeIds.add(e.targetId);
  });

  const history = await db
    .select({
      id: nodeHistory.id,
      action: nodeHistory.action,
      fieldName: nodeHistory.fieldName,
      oldValue: nodeHistory.oldValue,
      newValue: nodeHistory.newValue,
      createdAt: nodeHistory.createdAt,
      userName: users.displayName,
    })
    .from(nodeHistory)
    .innerJoin(users, eq(nodeHistory.userId, users.id))
    .where(eq(nodeHistory.nodeId, nodeId))
    .orderBy(desc(nodeHistory.createdAt))
    .limit(20);

  return NextResponse.json({
    data: {
      ...node,
      connectedEdges,
      connectedNodeIds: Array.from(connectedNodeIds),
      history,
    },
  });
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string; nodeId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId, nodeId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  try {
    const body = await request.json();
    const parsed = updateNodeSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const [existing] = await db
      .select()
      .from(nodes)
      .where(and(eq(nodes.id, nodeId), eq(nodes.projectId, projectId)))
      .limit(1);

    if (!existing) {
      return NextResponse.json({ error: 'Node not found' }, { status: 404 });
    }

    const [updated] = await db
      .update(nodes)
      .set({ ...parsed.data, updatedAt: new Date() })
      .where(eq(nodes.id, nodeId))
      .returning();

    // Record history for each changed field
    const changes = parsed.data;
    for (const [key, value] of Object.entries(changes)) {
      if (value !== undefined) {
        const oldVal = existing[key as keyof typeof existing];
        const action = key === 'status' ? 'status_changed' as const : 'updated' as const;
        await db.insert(nodeHistory).values({
          nodeId,
          projectId,
          userId: auth.userId,
          action,
          fieldName: key,
          oldValue: oldVal != null ? String(oldVal) : null,
          newValue: value != null ? String(value) : null,
        });
      }
    }

    // Waterfall: propagate status changes through connected nodes
    let propagated: { nodeId: string; oldStatus: string; newStatus: string }[] = [];

    if (
      changes.status &&
      changes.status !== existing.status &&
      (changes.status === 'PASS' || changes.status === 'FAIL')
    ) {
      const allNodes = await db
        .select({ id: nodes.id, status: nodes.status, parentId: nodes.parentId })
        .from(nodes)
        .where(eq(nodes.projectId, projectId));

      const allEdges = await db
        .select({ sourceId: edges.sourceId, targetId: edges.targetId, edgeType: edges.edgeType })
        .from(edges)
        .where(eq(edges.projectId, projectId));

      const waterfallChanges = computeWaterfall(
        nodeId,
        changes.status,
        allNodes as WaterfallNode[],
        allEdges as WaterfallEdge[],
      );

      if (waterfallChanges.length > 0) {
        // Apply each propagated status change
        for (const wc of waterfallChanges) {
          await db
            .update(nodes)
            .set({ status: wc.newStatus, updatedAt: new Date() })
            .where(eq(nodes.id, wc.nodeId));

          await db.insert(nodeHistory).values({
            nodeId: wc.nodeId,
            projectId,
            userId: auth.userId,
            action: 'status_changed',
            fieldName: 'status',
            oldValue: wc.oldStatus,
            newValue: wc.newStatus,
          });
        }

        propagated = waterfallChanges;
      }
    }

    return NextResponse.json({ data: updated, propagated });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}

export async function DELETE(
  _request: NextRequest,
  { params }: { params: Promise<{ projectId: string; nodeId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId, nodeId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  // Unparent children before deleting a GROUP node
  await db
    .update(nodes)
    .set({ parentId: null, updatedAt: new Date() })
    .where(and(eq(nodes.parentId, nodeId), eq(nodes.projectId, projectId)));

  // Explicitly delete connected edges (DB CASCADE would handle this, but explicit for clarity)
  await db
    .delete(edges)
    .where(and(eq(edges.projectId, projectId), or(eq(edges.sourceId, nodeId), eq(edges.targetId, nodeId))));

  await db
    .delete(nodes)
    .where(and(eq(nodes.id, nodeId), eq(nodes.projectId, projectId)));

  return NextResponse.json({ data: { success: true } });
}
