'use client';

import { Sidebar } from '@/components/layout/Sidebar';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] flex-1">
      <Sidebar />
      <main className="flex-1 md:pl-64">
        <div className="container p-6 md:p-8">
          {children}
        </div>
      </main>
    </div>
  );
}
