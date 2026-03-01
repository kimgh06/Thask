import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { nodes, projects, teamMembers } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { batchPositionSchema } from '@/lib/validators';
import { eq, and } from 'drizzle-orm';

export async function PATCH(
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

  try {
    const body = await request.json();
    const parsed = batchPositionSchema.safeParse(body);
    if (!parsed.success) {
      return NextResponse.json({ error: 'Invalid input' }, { status: 400 });
    }

    await Promise.all(
      parsed.data.positions.map((pos) => {
        const setData: Record<string, number> = { positionX: pos.x, positionY: pos.y };
        if (pos.width !== undefined) setData.width = pos.width;
        if (pos.height !== undefined) setData.height = pos.height;
        return db
          .update(nodes)
          .set(setData)
          .where(and(eq(nodes.id, pos.id), eq(nodes.projectId, projectId)));
      }),
    );

    return NextResponse.json({ data: { success: true } });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
