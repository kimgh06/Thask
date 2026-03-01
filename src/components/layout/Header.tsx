'use client';

import { useRouter } from 'next/navigation';
import { LogOut } from 'lucide-react';
import { useAuthStore } from '@/stores/useAuthStore';

export function Header() {
  const router = useRouter();
  const { user, logout } = useAuthStore();

  async function handleLogout() {
    await fetch('/api/auth/logout', { method: 'POST' });
    logout();
    router.push('/login');
  }

  return (
    <header className="flex h-14 items-center justify-between border-b border-gray-200 bg-white px-6">
      <div />
      <div className="flex items-center gap-3">
        {user && (
          <>
            <span className="text-sm text-gray-500">{user.displayName}</span>
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-thask-primary text-sm font-medium text-white">
              {user.displayName.charAt(0).toUpperCase()}
            </div>
          </>
        )}
        <button
          onClick={handleLogout}
          className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
          title="Sign Out"
        >
          <LogOut className="h-4 w-4" />
        </button>
      </div>
    </header>
  );
}
