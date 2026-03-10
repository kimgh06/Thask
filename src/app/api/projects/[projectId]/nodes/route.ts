import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';
import { nodes, projects, teamMembers, nodeHistory } from '@/lib/db/schema';
import { requireAuth, isAuthError } from '@/lib/auth/guard';
import { createNodeSchema } from '@/lib/validators';
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
  request: NextRequest,
  { params }: { params: Promise<{ projectId: string }> },
) {
  const auth = await requireAuth();
  if (isAuthError(auth)) return auth;

  const { projectId } = await params;
  if (!(await verifyProjectAccess(projectId, auth.userId))) {
    return NextResponse.json({ error: 'Project not found' }, { status: 404 });
  }

  const allNodes = await db.select().from(nodes).where(eq(nodes.projectId, projectId));

  const url = new URL(request.url);
  const typeFilter = url.searchParams.get('type');
  const statusFilter = url.searchParams.get('status');

  let filtered = allNodes;
  if (typeFilter) {
    filtered = filtered.filter((n) => n.type === typeFilter);
  }
  if (statusFilter) {
    filtered = filtered.filter((n) => n.status === statusFilter);
  }

  return NextResponse.json({ data: filtered });
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
    const parsed = createNodeSchema.safeParse(body);

    if (!parsed.success) {
      return NextResponse.json(
        { error: parsed.error.issues[0]?.message ?? 'Invalid input' },
        { status: 400 },
      );
    }

    const [node] = await db
      .insert(nodes)
      .values({
        projectId,
        ...parsed.data,
      })
      .returning();

    if (node) {
      await db.insert(nodeHistory).values({
        nodeId: node.id,
        projectId,
        userId: auth.userId,
        action: 'created',
      });
    }

    return NextResponse.json({ data: node });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
