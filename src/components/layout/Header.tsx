'use client';

import { useAuthStore } from '@/stores/useAuthStore';

export function Header() {
  const user = useAuthStore((s) => s.user);

  return (
    <header className="flex h-14 items-center justify-between border-b border-gray-200 bg-white px-6">
      <div />
      <div className="flex items-center gap-3">
        {user && (
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-thask-primary text-sm font-medium text-white">
            {user.displayName.charAt(0).toUpperCase()}
          </div>
        )}
      </div>
    </header>
  );
}
