'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/hooks/useAuth';
import { Link as LinkIcon, LogOut, Settings, BarChart2, PlusCircle, LayoutDashboard, Menu } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export function Navbar() {
  const pathname = usePathname();
  const { logout } = useAuth();
  const isAuthPage = pathname === '/login' || pathname === '/register';
  const isDashboard = pathname.startsWith('/dashboard') || pathname.startsWith('/settings');

  if (isAuthPage) return null;

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-16 items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-6">
          <div className="flex items-center md:hidden mr-2">
            {isDashboard && (
              <DropdownMenu>
                <DropdownMenuTrigger className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-accent hover:text-accent-foreground">
                  <Menu className="h-5 w-5" />
                  <span className="sr-only">Toggle menu</span>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                  <DropdownMenuItem>
                    <Link href="/dashboard" className="flex items-center w-full">
                      <LayoutDashboard className="mr-2 h-4 w-4" />
                      <span>Dashboard</span>
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Link href="/dashboard/links" className="flex items-center w-full">
                      <LinkIcon className="mr-2 h-4 w-4" />
                      <span>Links</span>
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Link href="/dashboard/analytics/overview" className="flex items-center w-full">
                      <BarChart2 className="mr-2 h-4 w-4" />
                      <span>Analytics</span>
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <Link href="/settings" className="flex items-center w-full">
                      <Settings className="mr-2 h-4 w-4" />
                      <span>Settings</span>
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => logout()} className="text-rose-500 focus:bg-rose-500/10 focus:text-rose-500 cursor-pointer">
                    <LogOut className="mr-2 h-4 w-4" />
                    <span>Log out</span>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
          
          <Link href={isDashboard ? '/dashboard' : '/'} className="flex items-center space-x-2">
            <LinkIcon className="h-5 w-5" />
            <span className="font-bold inline-block">SnapLink</span>
          </Link>
          
          {isDashboard && (
            <nav className="hidden md:flex items-center space-x-6 text-sm font-medium">
              <Link href="/dashboard" className={`transition-colors hover:text-foreground/80 ${pathname === '/dashboard' ? 'text-foreground' : 'text-foreground/60'}`}>Dashboard</Link>
              <Link href="/dashboard/links" className={`transition-colors hover:text-foreground/80 ${pathname === '/dashboard/links' ? 'text-foreground' : 'text-foreground/60'}`}>Links</Link>
            </nav>
          )}
        </div>

        <div className="flex items-center space-x-4">
          {isDashboard ? (
            <>
              <Link href="/dashboard/create">
                <Button variant="default" size="sm" className="hidden sm:flex h-8 gap-1">
                  <PlusCircle className="h-4 w-4" />
                  <span>Create Link</span>
                </Button>
              </Link>
              <Button variant="ghost" size="icon" className="hidden md:flex h-8 w-8" onClick={() => logout()}>
                <LogOut className="h-4 w-4" />
                <span className="sr-only">Log out</span>
              </Button>
            </>
          ) : (
            <>
              <Link href="/login">
                <Button variant="ghost" size="sm">Log in</Button>
              </Link>
              <Link href="/register">
                <Button size="sm">Get Started</Button>
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
