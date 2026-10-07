'use client';

import { useLinks } from '@/hooks/useLinks';
import { LinkTable } from '@/components/LinkTable';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Search, Plus } from 'lucide-react';
import Link from 'next/link';
import { useState } from 'react';

export default function LinksPage() {
  const { data: links, isLoading } = useLinks();
  const [searchQuery, setSearchQuery] = useState('');

  const filteredLinks = links?.filter(link => 
    link.shortUrl.toLowerCase().includes(searchQuery.toLowerCase()) || 
    link.originalUrl.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (link.customAlias && link.customAlias.toLowerCase().includes(searchQuery.toLowerCase()))
  ) || [];

  return (
    <div className="flex flex-col space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Your Links</h2>
          <p className="text-muted-foreground mt-1">Manage and monitor all your shortened URLs.</p>
        </div>
        <Link href="/dashboard/create">
          <Button className="gap-2">
            <Plus className="h-4 w-4" />
            Create Link
          </Button>
        </Link>
      </div>

      <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div className="relative w-full sm:w-[350px]">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search by URL or alias..."
            className="pl-8 bg-background"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
      </div>

      <LinkTable data={filteredLinks} isLoading={isLoading} />
    </div>
  );
}
