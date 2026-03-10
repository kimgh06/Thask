import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { teams, teamMembers, projects } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { updateProjectSchema } from '@/lib/validators';
import { eq, and } from 'drizzle-orm';

type Params = { params: Promise<{ teamSlug: string; projectId: string }> };

async function getTeamMembership(slug: string, userId: string) {
  const result = await db
    .select({ teamId: teams.id, role: teamMembers.role })
    .from(teams)
    .innerJoin(teamMembers, and(eq(teamMembers.teamId, teams.id), eq(teamMembers.userId, userId)))
    .where(eq(teams.slug, slug))
    .limit(1);
  return result[0] ?? null;
}

export async function PATCH(request: NextRequest, { params }: Params) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { teamSlug, projectId } = await params;
  const membership = await getTeamMembership(teamSlug, auth.userId);
  if (!membership) {
    return NextResponse.json({ error: 'Team not found' }, { status: 404 });
  }

  if (membership.role === 'viewer') {
    return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 });
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
      .where(and(eq(projects.id, projectId), eq(projects.teamId, membership.teamId)))
      .returning();

    if (!updated) {
      return NextResponse.json({ error: 'Project not found' }, { status: 404 });
    }

    return NextResponse.json({ data: updated });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}

export async function DELETE(_request: NextRequest, { params }: Params) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { teamSlug, projectId } = await params;
  const membership = await getTeamMembership(teamSlug, auth.userId);
  if (!membership) {
    return NextResponse.json({ error: 'Team not found' }, { status: 404 });
  }

  if (membership.role !== 'owner' && membership.role !== 'admin') {
    return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 });
  }

  try {
    const [deleted] = await db
      .delete(projects)
      .where(and(eq(projects.id, projectId), eq(projects.teamId, membership.teamId)))
      .returning({ id: projects.id });

    if (!deleted) {
      return NextResponse.json({ error: 'Project not found' }, { status: 404 });
    }

    return NextResponse.json({ data: { id: deleted.id } });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
