'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { LayoutDashboard, Link as LinkIcon, BarChart2, Settings, PlusCircle } from 'lucide-react';

const sidebarItems = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Links', href: '/dashboard/links', icon: LinkIcon },
  { name: 'Analytics', href: '/dashboard/analytics/overview', icon: BarChart2 },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 z-40 w-64 hidden border-r bg-background md:block pt-14">
      <div className="flex h-full flex-col px-3 py-4">
        <div className="mb-4 px-3">
          <Link href="/dashboard/create" className="w-full">
            <Button className="w-full justify-start gap-2" variant="default">
              <PlusCircle className="h-4 w-4" />
              Create Link
            </Button>
          </Link>
        </div>
        <nav className="flex-1 space-y-1">
          {sidebarItems.map((item) => {
            const isActive = pathname === item.href || (item.href !== '/dashboard' && pathname.startsWith(item.href));
            return (
              <Link key={item.name} href={item.href}>
                <span
                  className={cn(
                    'group flex items-center rounded-md px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground',
                    isActive ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'
                  )}
                >
                  <item.icon className="mr-3 h-4 w-4" />
                  <span>{item.name}</span>
                </span>
              </Link>
            );
          })}
        </nav>
      </div>
    </aside>
  );
}
