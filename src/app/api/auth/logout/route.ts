import { NextResponse } from 'next/server';
import { getSessionToken, invalidateSession, clearSessionCookie } from '@/lib/auth/session';

export async function POST() {
  try {
    const token = await getSessionToken();

    if (token) {
      await invalidateSession(token);
    }

    await clearSessionCookie();

    return NextResponse.json({ data: { success: true } });
  } catch {
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 });
  }
}
