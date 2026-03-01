import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { teams, teamMembers, projects } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { createTeamSchema } from '@/lib/validators';
import { eq, inArray } from 'drizzle-orm';

export async function GET() {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const memberships = await db
    .select({ teamId: teamMembers.teamId })
    .from(teamMembers)
    .where(eq(teamMembers.userId, auth.userId));

  if (memberships.length === 0) {
    return NextResponse.json({ data: [] });
  }

  const teamIds = memberships.map((m) => m.teamId);

  const userTeams = await db
    .select()
    .from(teams)
    .where(inArray(teams.id, teamIds));

  const teamProjects = await db
    .select()
    .from(projects)
    .where(inArray(projects.teamId, teamIds));

  const teamsWithProjects = userTeams.map((team) => ({
    ...team,
    projects: teamProjects.filter((p) => p.teamId === team.id),
  }));

  return NextResponse.json({ data: teamsWithProjects });
}

export async function POST(request: NextRequest) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  try {
    const body = await request.json();
    const parsed = createTeamSchema.safeParse(body);

    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const { name, slug } = parsed.data;

    const existing = await db
      .select({ id: teams.id })
      .from(teams)
      .where(eq(teams.slug, slug))
      .limit(1);

    if (existing.length > 0) {
      return NextResponse.json({ error: 'Team slug already taken' }, { status: 409 });
    }

    const [team] = await db
      .insert(teams)
      .values({ name, slug, createdBy: auth.userId })
      .returning();

    if (!team) {
      return NextResponse.json({ error: 'Failed to create team' }, { status: 500 });
    }

    await db.insert(teamMembers).values({
      teamId: team.id,
      userId: auth.userId,
      role: 'owner',
    });

    return NextResponse.json({ data: team });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
