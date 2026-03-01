import { NextResponse } from 'next/server';
import { getSessionToken, validateSession } from '@/lib/auth/session';

export async function GET() {
  try {
    const token = await getSessionToken();

    if (!token) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
    }

    const session = await validateSession(token);

    if (!session) {
      return NextResponse.json({ error: 'Session expired' }, { status: 401 });
    }

    return NextResponse.json({
      data: {
        id: session.userId,
        email: session.email,
        displayName: session.displayName,
      },
    });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
