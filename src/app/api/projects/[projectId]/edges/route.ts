import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { edges, projects, teamMembers } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { createEdgeSchema } from '@/lib/validators';
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

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  const projectEdges = await db
    .select()
    .from(edges)
    .where(eq(edges.projectId, projectId));

  return NextResponse.json({ data: projectEdges });
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  try {
    const body = await request.json();
    const parsed = createEdgeSchema.safeParse(body);

    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const [edge] = await db
      .insert(edges)
      .values({
        projectId,
        ...parsed.data,
      })
      .returning();

    return NextResponse.json({ data: edge });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
