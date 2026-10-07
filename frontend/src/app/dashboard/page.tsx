'use client';

import { useLinks } from '@/hooks/useLinks';
import { useAnalytics as useAnalyticsHook } from '@/hooks/useAnalytics';
import { MetricCard } from '@/components/MetricCard';
import { AnalyticsChart } from '@/components/AnalyticsChart';
import { LinkTable } from '@/components/LinkTable';
import { BarChart3, Link as LinkIcon, Activity, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import Link from 'next/link';

export default function DashboardPage() {
  const { data: links, isLoading: isLinksLoading } = useLinks();
  const { data: analytics, isLoading: isAnalyticsLoading } = useAnalyticsHook();

  const activeLinks = links?.filter((l) => l.status === 'active').length || 0;
  const expiredLinks = links?.filter((l) => l.status === 'expired').length || 0;

  return (
    <div className="flex-1 space-y-8">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Overview</h2>
        <div className="flex items-center space-x-2">
          <Link href="/dashboard/create">
            <Button>Create Short Link</Button>
          </Link>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Links"
          value={links?.length || 0}
          icon={LinkIcon}
          description="Total links created"
        />
        <MetricCard
          title="Total Clicks"
          value={analytics?.totalClicks?.toLocaleString() || 0}
          icon={BarChart3}
          trend={{ value: 12, isUpward: true }}
          description="Compared to last month"
        />
        <MetricCard
          title="Active Links"
          value={activeLinks}
          icon={Activity}
          description="Links currently working"
        />
        <MetricCard
          title="Expired Links"
          value={expiredLinks}
          icon={AlertCircle}
          description="Links no longer active"
        />
      </div>

      <div className="grid gap-4 grid-cols-1 md:grid-cols-7">
        <div className="col-span-1 md:col-span-4 rounded-xl border bg-card text-card-foreground shadow-sm">
          <div className="flex flex-col space-y-1.5 p-6">
            <h3 className="font-semibold leading-none tracking-tight">Click Traffic (Last 7 Days)</h3>
            <p className="text-sm text-muted-foreground">Your link performance over time.</p>
          </div>
          <div className="p-6 pt-0">
            {isAnalyticsLoading ? (
              <div className="h-[300px] flex items-center justify-center text-muted-foreground">Loading chart...</div>
            ) : (
              <AnalyticsChart data={analytics?.clicksByDay || []} />
            )}
          </div>
        </div>
        
        <div className="col-span-1 md:col-span-3 rounded-xl border bg-card text-card-foreground shadow-sm flex flex-col">
          <div className="flex flex-col space-y-1.5 p-6">
            <h3 className="font-semibold leading-none tracking-tight">Top Sources</h3>
            <p className="text-sm text-muted-foreground">Where your traffic is coming from.</p>
          </div>
          <div className="p-6 pt-0 flex-1">
            <div className="space-y-4">
              {analytics?.referrers?.slice(0, 5).map((ref) => (
                <div key={ref.source} className="flex items-center justify-between">
                  <span className="font-medium">{ref.source}</span>
                  <span className="text-muted-foreground">{ref.count.toLocaleString()}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        <h3 className="text-xl font-semibold tracking-tight">Recent Links</h3>
        <LinkTable data={links?.slice(0, 5) || []} isLoading={isLinksLoading} />
        {links && links.length > 5 && (
          <div className="flex justify-center mt-4">
            <Link href="/dashboard/links">
              <Button variant="outline">View All Links</Button>
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
