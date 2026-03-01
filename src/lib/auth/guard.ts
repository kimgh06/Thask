import { NextResponse } from 'next/server';
import { getSessionToken, validateSession } from './session';

export interface AuthContext {
  userId: string;
  email: string;
  displayName: string;
}

export async function requireAuth(): Promise<AuthContext | NextResponse> {
  const token = await getSessionToken();

  if (!token) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }

  const session = await validateSession(token);

  if (!session) {
    return NextResponse.json({ error: 'Session expired' }, { status: 401 });
  }

  return {
    userId: session.userId,
    email: session.email,
    displayName: session.displayName,
  };
}

export function isAuthError(result: AuthContext | NextResponse): result is NextResponse {
  return result instanceof NextResponse;
}
