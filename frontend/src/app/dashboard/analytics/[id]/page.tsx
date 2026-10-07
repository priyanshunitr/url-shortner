'use client';

import { useAnalytics } from '@/hooks/useAnalytics';
import { MetricCard } from '@/components/MetricCard';
import { AnalyticsChart } from '@/components/AnalyticsChart';
import { ArrowLeft, MousePointerClick, Calendar, Globe, MonitorSmartphone } from 'lucide-react';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import { use } from 'react';
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from '@/components/ui/table';

export default function AnalyticsPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const { data: analytics, isLoading } = useAnalytics(resolvedParams.id);

  if (isLoading) {
    return <div className="flex justify-center py-20 text-muted-foreground">Loading analytics...</div>;
  }

  if (!analytics) {
    return <div className="flex justify-center py-20 text-rose-500">Failed to load analytics</div>;
  }

  return (
    <div className="flex-1 space-y-8">
      <div className="flex items-center gap-4">
        <Link href="/dashboard/links">
          <Button variant="ghost" size="icon" className="rounded-full">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Analytics</h2>
          <p className="text-muted-foreground mt-1">Detailed performance metrics for your link.</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Clicks"
          value={analytics.totalClicks.toLocaleString()}
          icon={MousePointerClick}
        />
        <MetricCard
          title="Clicks Today"
          value={analytics.clicksToday.toLocaleString()}
          icon={Calendar}
          trend={{ value: 5, isUpward: true }}
        />
        <MetricCard
          title="This Week"
          value={analytics.clicksThisWeek.toLocaleString()}
          icon={Calendar}
        />
        <MetricCard
          title="This Month"
          value={analytics.clicksThisMonth.toLocaleString()}
          icon={Calendar}
        />
      </div>

      <div className="rounded-xl border bg-card text-card-foreground shadow-sm">
        <div className="flex flex-col space-y-1.5 p-6 border-b">
          <h3 className="font-semibold leading-none tracking-tight">Clicks Over Time</h3>
        </div>
        <div className="p-6">
          <AnalyticsChart data={analytics.clicksByDay} />
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="rounded-xl border bg-card text-card-foreground shadow-sm">
          <div className="flex flex-col space-y-1.5 p-6 border-b">
            <h3 className="font-semibold flex items-center gap-2">
              <Globe className="h-4 w-4 text-muted-foreground" />
              Referrers
            </h3>
          </div>
          <div className="p-6">
            <div className="space-y-4">
              {analytics.referrers.map((ref) => (
                <div key={ref.source} className="flex items-center justify-between">
                  <span className="font-medium">{ref.source}</span>
                  <div className="flex items-center gap-4">
                    <div className="w-32 h-2 bg-muted rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-primary" 
                        style={{ width: `${(ref.count / analytics.totalClicks) * 100}%` }}
                      />
                    </div>
                    <span className="text-muted-foreground w-12 text-right">{ref.count}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="rounded-xl border bg-card text-card-foreground shadow-sm">
          <div className="flex flex-col space-y-1.5 p-6 border-b">
            <h3 className="font-semibold flex items-center gap-2">
              <MonitorSmartphone className="h-4 w-4 text-muted-foreground" />
              Devices & Browsers
            </h3>
          </div>
          <div className="p-6 flex flex-col sm:flex-row gap-8">
            <div className="flex-1 space-y-4">
              <h4 className="text-sm font-medium text-muted-foreground mb-2">Devices</h4>
              {analytics.devices.map((device) => (
                <div key={device.device} className="flex items-center justify-between">
                  <span className="font-medium text-sm">{device.device}</span>
                  <span className="text-muted-foreground text-sm">{device.count}</span>
                </div>
              ))}
            </div>
            <div className="flex-1 space-y-4">
              <h4 className="text-sm font-medium text-muted-foreground mb-2">Browsers</h4>
              {analytics.browsers.map((browser) => (
                <div key={browser.browser} className="flex items-center justify-between">
                  <span className="font-medium text-sm">{browser.browser}</span>
                  <span className="text-muted-foreground text-sm">{browser.count}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        <h3 className="text-xl font-semibold tracking-tight">Recent Visits</h3>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>IP Address</TableHead>
                <TableHead>Referrer</TableHead>
                <TableHead>Device</TableHead>
                <TableHead>Browser</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {analytics.recentVisits.map((visit) => (
                <TableRow key={visit.id}>
                  <TableCell className="text-muted-foreground whitespace-nowrap">
                    {new Date(visit.timestamp).toLocaleString()}
                  </TableCell>
                  <TableCell className="font-mono text-sm">{visit.ip || 'Unknown'}</TableCell>
                  <TableCell>{visit.referrer}</TableCell>
                  <TableCell>{visit.device}</TableCell>
                  <TableCell>{visit.browser}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
