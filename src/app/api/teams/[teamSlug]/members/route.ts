import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { teams, teamMembers, users } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { eq, and } from 'drizzle-orm';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ teamSlug: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { teamSlug } = await params;

  const result = await db
    .select({
      userId: users.id,
      displayName: users.displayName,
      email: users.email,
      role: teamMembers.role,
    })
    .from(teams)
    .innerJoin(teamMembers, eq(teamMembers.teamId, teams.id))
    .innerJoin(users, eq(users.id, teamMembers.userId))
    .where(
      and(
        eq(teams.slug, teamSlug),
        eq(teamMembers.teamId, teams.id),
      ),
    );

  if (result.length === 0) {
    // Check if the user has access
    const membership = await db
      .select({ id: teams.id })
      .from(teams)
      .innerJoin(teamMembers, and(eq(teamMembers.teamId, teams.id), eq(teamMembers.userId, auth.userId)))
      .where(eq(teams.slug, teamSlug))
      .limit(1);

    if (membership.length === 0) {
      return NextResponse.json({ error: 'Team not found' }, { status: 404 });
    }
  }

  return NextResponse.json({ data: result });
}
