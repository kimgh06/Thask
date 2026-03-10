import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { teams, teamMembers } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { eq, and } from 'drizzle-orm';
import { z } from 'zod';

type Params = { params: Promise<{ teamSlug: string }> };

const updateTeamSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  slug: z
    .string()
    .min(1)
    .max(100)
    .regex(/^[a-z0-9-]+$/, 'Slug must be lowercase alphanumeric with dashes')
    .optional(),
});

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

  const { teamSlug } = await params;
  const membership = await getTeamMembership(teamSlug, auth.userId);
  if (!membership) {
    return NextResponse.json({ error: 'Team not found' }, { status: 404 });
  }

  if (membership.role !== 'owner' && membership.role !== 'admin') {
    return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 });
  }

  try {
    const body = await request.json();
    const parsed = updateTeamSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    if (parsed.data.slug) {
      const existing = await db
        .select({ id: teams.id })
        .from(teams)
        .where(eq(teams.slug, parsed.data.slug))
        .limit(1);
      if (existing.length > 0 && existing[0]!.id !== membership.teamId) {
        return NextResponse.json({ error: 'Team slug already taken' }, { status: 409 });
      }
    }

    const [updated] = await db
      .update(teams)
      .set({ ...parsed.data, updatedAt: new Date() })
      .where(eq(teams.id, membership.teamId))
      .returning();

    if (!updated) {
      return NextResponse.json({ error: 'Team not found' }, { status: 404 });
    }

    return NextResponse.json({ data: updated });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}

export async function DELETE(_request: NextRequest, { params }: Params) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { teamSlug } = await params;
  const membership = await getTeamMembership(teamSlug, auth.userId);
  if (!membership) {
    return NextResponse.json({ error: 'Team not found' }, { status: 404 });
  }

  if (membership.role !== 'owner') {
    return NextResponse.json({ error: 'Only the team owner can delete the team' }, { status: 403 });
  }

  try {
    const [deleted] = await db
      .delete(teams)
      .where(eq(teams.id, membership.teamId))
      .returning({ id: teams.id });

    if (!deleted) {
      return NextResponse.json({ error: 'Team not found' }, { status: 404 });
    }

    return NextResponse.json({ data: { id: deleted.id } });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
