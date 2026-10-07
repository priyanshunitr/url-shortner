'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useCreateLink } from '@/hooks/useLinks';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { toast } from 'sonner';
import { ArrowLeft, Link as LinkIcon, Calendar } from 'lucide-react';
import Link from 'next/link';
import { CopyButton } from '@/components/CopyButton';

export default function CreateLinkPage() {
  const router = useRouter();
  const createLink = useCreateLink();
  const [url, setUrl] = useState('');
  const [alias, setAlias] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  
  const [createdLink, setCreatedLink] = useState<{shortUrl: string} | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!url) {
      toast.error('Please enter a valid URL');
      return;
    }

    try {
      const result = await createLink.mutateAsync({
        originalUrl: url,
        customAlias: alias || undefined,
        expiresAt: expiresAt ? new Date(expiresAt).toISOString() : undefined,
      });
      
      setCreatedLink({ shortUrl: `https://${result.shortUrl}` });
      toast.success('Link created successfully!');
    } catch (err) {
      toast.error('Failed to create link. Alias might be taken.');
    }
  };

  if (createdLink) {
    return (
      <div className="max-w-2xl mx-auto mt-8">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-emerald-500/10 text-emerald-500 rounded-full flex items-center justify-center mb-4">
              <LinkIcon className="w-6 h-6" />
            </div>
            <CardTitle className="text-2xl">Your short link is ready!</CardTitle>
            <CardDescription>Share this link with your audience.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center gap-2 p-4 border rounded-lg bg-muted/50">
              <Input readOnly value={createdLink.shortUrl} className="font-medium text-lg bg-transparent border-0 focus-visible:ring-0" />
              <CopyButton text={createdLink.shortUrl} variant="default" size="default" className="shrink-0 gap-2" />
            </div>
            
            <div className="flex justify-center p-6 border rounded-lg border-dashed">
              <div className="flex flex-col items-center">
                <div className="w-48 h-48 bg-white p-2 rounded-lg mb-4 flex items-center justify-center">
                  {/* Real QR code would go here */}
                  <div className="grid grid-cols-5 gap-1 w-full h-full p-2 border-2 border-black">
                    {Array.from({ length: 25 }).map((_, i) => (
                      <div key={i} className={`bg-black ${i % 3 === 0 ? 'opacity-100' : 'opacity-0'}`} />
                    ))}
                  </div>
                </div>
                <p className="text-sm text-muted-foreground">Scan QR Code</p>
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex justify-between">
            <Button variant="outline" onClick={() => { setCreatedLink(null); setUrl(''); setAlias(''); setExpiresAt(''); }}>
              Create Another
            </Button>
            <Link href="/dashboard/links">
              <Button>Go to Links</Button>
            </Link>
          </CardFooter>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/dashboard/links">
          <Button variant="ghost" size="icon" className="rounded-full">
            <ArrowLeft className="h-5 w-5" />
          </Button>
        </Link>
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Create new link</h2>
          <p className="text-muted-foreground text-sm">Shorten a long URL and customize its behavior.</p>
        </div>
      </div>

      <Card>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-6 pt-6">
            <div className="space-y-2">
              <Label htmlFor="url">Destination URL <span className="text-rose-500">*</span></Label>
              <Input 
                id="url" 
                placeholder="https://example.com/very/long/url/that/needs/shortening" 
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                required
                type="url"
              />
              <p className="text-[0.8rem] text-muted-foreground">This is where your short link will redirect to.</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="alias">Custom Alias (Optional)</Label>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground text-sm font-medium">snap.link/</span>
                  <Input 
                    id="alias" 
                    placeholder="my-campaign" 
                    value={alias}
                    onChange={(e) => setAlias(e.target.value)}
                  />
                </div>
                <p className="text-[0.8rem] text-muted-foreground">Leave empty for a random string.</p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="expires">Expiration Date (Optional)</Label>
                <div className="relative">
                  <Input 
                    id="expires" 
                    type="datetime-local"
                    value={expiresAt}
                    onChange={(e) => setExpiresAt(e.target.value)}
                  />
                </div>
                <p className="text-[0.8rem] text-muted-foreground">Link will expire after this time.</p>
              </div>
            </div>
          </CardContent>
          <CardFooter className="flex justify-end gap-4 border-t px-6 py-4 bg-muted/20">
            <Link href="/dashboard">
              <Button variant="ghost" type="button">Cancel</Button>
            </Link>
            <Button type="submit" disabled={createLink.isPending}>
              {createLink.isPending ? 'Creating...' : 'Create Short Link'}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
