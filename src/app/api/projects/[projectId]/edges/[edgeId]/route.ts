import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { edges, projects, teamMembers } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { updateEdgeSchema } from '@/lib/validators';
import { eq, and } from 'drizzle-orm';

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

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string; edgeId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId, edgeId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  try {
    const body = await request.json();
    const parsed = updateEdgeSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const [updated] = await db
      .update(edges)
      .set(parsed.data)
      .where(and(eq(edges.id, edgeId), eq(edges.projectId, projectId)))
      .returning();

    if (!updated) {
      return NextResponse.json({ error: 'Edge not found' }, { status: 404 });
    }

    return NextResponse.json({ data: updated });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}

export async function DELETE(
  _request: NextRequest,
  { params }: { params: Promise<{ projectId: string; edgeId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId, edgeId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  await db
    .delete(edges)
    .where(and(eq(edges.id, edgeId), eq(edges.projectId, projectId)));

  return NextResponse.json({ data: { success: true } });
}
