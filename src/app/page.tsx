import { redirect } from 'next/navigation';
import { getSessionToken, validateSession } from '@/lib/auth/session';

export default async function Home() {
  const token = await getSessionToken();

  if (!token) {
    redirect('/login');
  }

  const session = await validateSession(token);

  if (!session) {
    redirect('/login');
  }

  redirect('/dashboard');
}
