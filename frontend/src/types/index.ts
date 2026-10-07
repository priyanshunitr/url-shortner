export type User = {
  id: string;
  email: string;
  createdAt: string;
};

export type Link = {
  id: string;
  shortUrl: string;
  originalUrl: string;
  customAlias?: string;
  clicks: number;
  status: 'active' | 'expired';
  expiresAt?: string | null;
  createdAt: string;
};

export type Analytics = {
  totalClicks: number;
  clicksToday: number;
  clicksThisWeek: number;
  clicksThisMonth: number;
  clicksByDay: { date: string; clicks: number }[];
  recentVisits: { id: string; ip?: string; referrer: string; device: string; browser: string; timestamp: string }[];
  referrers: { source: string; count: number }[];
  devices: { device: string; count: number }[];
  browsers: { browser: string; count: number }[];
};
