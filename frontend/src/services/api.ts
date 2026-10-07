import { Link, Analytics, User } from '../types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  // In a real app, you would get the token from cookies or localStorage
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE_URL}${url}`, { ...options, headers });
  
  if (!response.ok) {
    throw new ApiError('API request failed', response.status);
  }
  
  return response.json();
}

// Mocks for development
const mockLinks: Link[] = [
  { id: '1', shortUrl: 'snap.link/abc', originalUrl: 'https://example.com/very-long-url', clicks: 142, status: 'active', createdAt: new Date().toISOString() },
  { id: '2', shortUrl: 'snap.link/promo24', originalUrl: 'https://github.com', customAlias: 'promo24', clicks: 89, status: 'active', createdAt: new Date().toISOString() },
  { id: '3', shortUrl: 'snap.link/xyz', originalUrl: 'https://youtube.com', clicks: 12, status: 'expired', expiresAt: new Date().toISOString(), createdAt: new Date().toISOString() },
];

const USE_MOCK = process.env.NEXT_PUBLIC_USE_MOCKS === 'true';

export const api = {
  // Auth
  login: async (credentials: Record<string, string>) => {
    if (USE_MOCK) return { token: 'mock-token', user: { id: '1', email: credentials.email } };
    return fetchWithAuth('/auth/login', { method: 'POST', body: JSON.stringify(credentials) });
  },
  register: async (credentials: Record<string, string>) => {
    if (USE_MOCK) return { token: 'mock-token', user: { id: '1', email: credentials.email } };
    return fetchWithAuth('/auth/register', { method: 'POST', body: JSON.stringify(credentials) });
  },
  
  // Links
  getLinks: async (): Promise<Link[]> => {
    if (USE_MOCK) return new Promise(resolve => setTimeout(() => resolve(mockLinks), 500));
    return fetchWithAuth('/links');
  },
  createLink: async (data: Partial<Link>): Promise<Link> => {
    if (USE_MOCK) {
      const newLink: Link = {
        id: Math.random().toString(36).substring(2, 11),
        shortUrl: `snap.link/${data.customAlias || Math.random().toString(36).substring(2, 8)}`,
        originalUrl: data.originalUrl || '',
        customAlias: data.customAlias,
        clicks: 0,
        status: 'active',
        createdAt: new Date().toISOString(),
        expiresAt: data.expiresAt,
      };
      return new Promise(resolve => setTimeout(() => resolve(newLink), 500));
    }
    return fetchWithAuth('/links', { method: 'POST', body: JSON.stringify(data) });
  },
  deleteLink: async (id: string) => {
    if (USE_MOCK) return new Promise(resolve => setTimeout(() => resolve({ success: true }), 500));
    return fetchWithAuth(`/links/${id}`, { method: 'DELETE' });
  },

  // Analytics
  getAnalytics: async (linkId?: string): Promise<Analytics> => {
    if (USE_MOCK) {
      return new Promise(resolve => setTimeout(() => resolve({
        totalClicks: 1243,
        clicksToday: 42,
        clicksThisWeek: 312,
        clicksThisMonth: 1243,
        clicksByDay: [
          { date: 'Mon', clicks: 120 },
          { date: 'Tue', clicks: 150 },
          { date: 'Wed', clicks: 180 },
          { date: 'Thu', clicks: 140 },
          { date: 'Fri', clicks: 210 },
          { date: 'Sat', clicks: 280 },
          { date: 'Sun', clicks: 163 },
        ],
        recentVisits: [
          { id: '1', referrer: 'Twitter', device: 'Mobile', browser: 'Safari', timestamp: new Date().toISOString() },
          { id: '2', referrer: 'Direct', device: 'Desktop', browser: 'Chrome', timestamp: new Date().toISOString() },
          { id: '3', referrer: 'Google', device: 'Mobile', browser: 'Chrome', timestamp: new Date().toISOString() },
        ],
        referrers: [
          { source: 'Twitter', count: 450 },
          { source: 'Direct', count: 320 },
          { source: 'Google', count: 200 },
          { source: 'GitHub', count: 150 },
        ],
        devices: [
          { device: 'Mobile', count: 850 },
          { device: 'Desktop', count: 350 },
          { device: 'Tablet', count: 43 },
        ],
        browsers: [
          { browser: 'Chrome', count: 620 },
          { browser: 'Safari', count: 410 },
          { browser: 'Firefox', count: 120 },
          { browser: 'Edge', count: 93 },
        ]
      }), 500));
    }
    return fetchWithAuth(linkId ? `/analytics/${linkId}` : '/analytics');
  }
};
