'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { LayoutDashboard } from 'lucide-react';
import { useGraphStore } from '@/stores/useGraphStore';
import { cn } from '@/lib/utils';

const navItems = [
  { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
];

export function Sidebar() {
  const pathname = usePathname();
  const { isDetailPanelOpen } = useGraphStore();

  return (
    <aside className="flex h-full w-full flex-col border-r border-gray-200 bg-white">
      {/* Header — always visible */}
      <div className="flex h-14 items-center border-b border-gray-200 px-4">
        <Link href="/dashboard" className="flex items-center gap-2 text-xl font-bold text-gray-900">
          <Image src="/icon.svg" alt="Thask" width={28} height={28} />
          Thask
        </Link>
      </div>

      {/* Middle — portal slot or default nav */}
      {isDetailPanelOpen ? (
        <div id="sidebar-detail-slot" className="flex flex-1 flex-col overflow-hidden" />
      ) : (
        <nav className="flex-1 space-y-1 p-3">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-blue-50 text-thask-primary'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900',
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
      )}
    </aside>
  );
}