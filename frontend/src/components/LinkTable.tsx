'use client';

import { useState } from 'react';
import Link from 'next/link';
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from '@/components/ui/table';
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuLabel, 
  DropdownMenuSeparator, 
  DropdownMenuTrigger 
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import { MoreHorizontal, ExternalLink, BarChart2, Edit, Trash, QrCode } from 'lucide-react';
import { CopyButton } from './CopyButton';
import { Link as LinkType } from '@/types';
import { useDeleteLink } from '@/hooks/useLinks';
import { toast } from 'sonner';

interface LinkTableProps {
  data: LinkType[];
  isLoading?: boolean;
}

export function LinkTable({ data, isLoading }: LinkTableProps) {
  const deleteLink = useDeleteLink();

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to delete this link?')) {
      try {
        await deleteLink.mutateAsync(id);
        toast.success('Link deleted successfully');
      } catch (err) {
        toast.error('Failed to delete link');
      }
    }
  };

  if (isLoading) {
    return <div className="text-center py-10 text-muted-foreground">Loading links...</div>;
  }

  if (!data || data.length === 0) {
    return (
      <div className="text-center py-16 border rounded-lg border-dashed">
        <h3 className="text-lg font-medium mb-2">No links found</h3>
        <p className="text-muted-foreground mb-4">You haven&apos;t created any short links yet.</p>
        <Link href="/dashboard/create">
          <Button>Create your first link</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Short URL</TableHead>
            <TableHead className="hidden md:table-cell">Original URL</TableHead>
            <TableHead>Clicks</TableHead>
            <TableHead className="hidden sm:table-cell">Status</TableHead>
            <TableHead className="hidden lg:table-cell">Created</TableHead>
            <TableHead className="w-[80px]"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((link) => (
            <TableRow key={link.id}>
              <TableCell className="font-medium">
                <div className="flex items-center gap-2">
                  <a href={`https://${link.shortUrl}`} target="_blank" rel="noreferrer" className="hover:underline text-primary">
                    {link.shortUrl}
                  </a>
                  <CopyButton text={`https://${link.shortUrl}`} size="sm" className="h-6 w-6" />
                </div>
              </TableCell>
              <TableCell className="hidden md:table-cell max-w-[200px] truncate text-muted-foreground">
                <a href={link.originalUrl} target="_blank" rel="noreferrer" className="hover:underline flex items-center gap-1">
                  <span className="truncate">{link.originalUrl}</span>
                  <ExternalLink className="h-3 w-3 shrink-0" />
                </a>
              </TableCell>
              <TableCell>{link.clicks.toLocaleString()}</TableCell>
              <TableCell className="hidden sm:table-cell">
                <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                  link.status === 'active' 
                    ? 'bg-emerald-500/10 text-emerald-500' 
                    : 'bg-rose-500/10 text-rose-500'
                }`}>
                  {link.status.charAt(0).toUpperCase() + link.status.slice(1)}
                </span>
              </TableCell>
              <TableCell className="hidden lg:table-cell text-muted-foreground">
                {new Date(link.createdAt).toLocaleDateString()}
              </TableCell>
              <TableCell>
                <DropdownMenu>
                  <DropdownMenuTrigger className="flex h-8 w-8 p-0 items-center justify-center rounded-md hover:bg-accent hover:text-accent-foreground">
                    <span className="sr-only">Open menu</span>
                    <MoreHorizontal className="h-4 w-4" />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel>Actions</DropdownMenuLabel>
                    <DropdownMenuItem>
                      <Link href={`/dashboard/analytics/${link.id}`} className="cursor-pointer flex w-full items-center">
                        <BarChart2 className="mr-2 h-4 w-4" />
                        <span>Analytics</span>
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem>
                      <QrCode className="mr-2 h-4 w-4" />
                      <span>QR Code</span>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem className="text-rose-500 focus:text-rose-500 focus:bg-rose-500/10 cursor-pointer" onClick={() => handleDelete(link.id)}>
                      <Trash className="mr-2 h-4 w-4" />
                      <span>Delete</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
