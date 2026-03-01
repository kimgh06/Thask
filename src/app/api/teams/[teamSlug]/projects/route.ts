import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { teams, teamMembers, projects } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { createProjectSchema } from '@/lib/validators';
import { eq, and } from 'drizzle-orm';

async function getTeamBySlug(slug: string, userId: string) {
  const result = await db
    .select({ teamId: teams.id, role: teamMembers.role })
    .from(teams)
    .innerJoin(teamMembers, and(eq(teamMembers.teamId, teams.id), eq(teamMembers.userId, userId)))
    .where(eq(teams.slug, slug))
    .limit(1);
  return result[0] ?? null;
}

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ teamSlug: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { teamSlug } = await params;
  const membership = await getTeamBySlug(teamSlug, auth.userId);
  if (!membership) {
    return NextResponse.json({ error: 'Team not found' }, { status: 404 });
  }

  const teamProjects = await db
    .select()
    .from(projects)
    .where(eq(projects.teamId, membership.teamId));

  return NextResponse.json({ data: teamProjects });
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ teamSlug: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { teamSlug } = await params;
  const membership = await getTeamBySlug(teamSlug, auth.userId);
  if (!membership) {
    return NextResponse.json({ error: 'Team not found' }, { status: 404 });
  }

  try {
    const body = await request.json();
    const parsed = createProjectSchema.safeParse(body);

    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const [project] = await db
      .insert(projects)
      .values({
        teamId: membership.teamId,
        name: parsed.data.name,
        description: parsed.data.description,
        createdBy: auth.userId,
      })
      .returning();

    return NextResponse.json({ data: project });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
