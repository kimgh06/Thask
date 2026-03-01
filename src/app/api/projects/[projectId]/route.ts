import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { projects, teamMembers } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { updateProjectSchema } from '@/lib/validators';
import { eq, and } from 'drizzle-orm';

async function getProjectWithAuth(projectId: string, userId: string) {
  const result = await db
    .select({ project: projects, role: teamMembers.role })
    .from(projects)
    .innerJoin(
      teamMembers,
      and(eq(teamMembers.teamId, projects.teamId), eq(teamMembers.userId, userId)),
    )
    .where(eq(projects.id, projectId))
    .limit(1);
  return result[0] ?? null;
}

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;
  const result = await getProjectWithAuth(projectId, auth.userId);
  if (!result) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  return NextResponse.json({ data: result.project });
}

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;
  const result = await getProjectWithAuth(projectId, auth.userId);
  if (!result) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  try {
    const body = await request.json();
    const parsed = updateProjectSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const [updated] = await db
      .update(projects)
      .set({ ...parsed.data, updatedAt: new Date() })
      .where(eq(projects.id, projectId))
      .returning();

    return NextResponse.json({ data: updated });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}

export async function DELETE(
  _request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;
  const result = await getProjectWithAuth(projectId, auth.userId);
  if (!result) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  if (result.role !== 'owner' && result.role !== 'admin') {
    return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 });
  }

  await db.delete(projects).where(eq(projects.id, projectId));
  return NextResponse.json({ data: { success: true } });
}
