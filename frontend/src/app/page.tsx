'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ArrowRight, Link as LinkIcon, BarChart3, Clock, Zap, Shield, Globe } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { CopyButton } from '@/components/CopyButton';
import { api } from '@/services/api';

const features = [
  {
    title: 'Custom Aliases',
    description: 'Create memorable, branded links instead of random characters.',
    icon: LinkIcon,
  },
  {
    title: 'Analytics Dashboard',
    description: 'Track clicks, geographic data, and referrer sources in real-time.',
    icon: BarChart3,
  },
  {
    title: 'Link Expiration',
    description: 'Set links to automatically expire after a specific date and time.',
    icon: Clock,
  },
  {
    title: 'Fast Redirects',
    description: 'Lightning-fast redirection powered by global edge networks.',
    icon: Zap,
  },
  {
    title: 'Rate Limiting',
    description: 'Protect your links from abuse with built-in rate limiting.',
    icon: Shield,
  },
  {
    title: 'Secure Infrastructure',
    description: 'Enterprise-grade security to keep your data and links safe.',
    icon: Globe,
  },
];

export default function LandingPage() {
  const [url, setUrl] = useState('');
  const [shortUrl, setShortUrl] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleShorten = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url) return;
    
    setIsLoading(true);
    try {
      const result = await api.createLink({ originalUrl: url });
      setShortUrl(`https://${result.shortUrl}`);
    } catch (err) {
      // Handle error
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex flex-col min-h-screen">
      <main className="flex-1">
        {/* Hero Section */}
        <section className="py-20 md:py-32 px-4 text-center max-w-5xl mx-auto">
          <h1 className="text-4xl md:text-6xl font-extrabold tracking-tight mb-6 text-balance">
            Shorten, share, and track <br className="hidden md:block" /> your links with ease.
          </h1>
          <p className="text-xl text-muted-foreground mb-10 max-w-2xl mx-auto text-balance">
            The modern URL shortener built for teams. Get detailed analytics, custom aliases, and enterprise-grade reliability.
          </p>
          
          <div className="max-w-xl mx-auto">
            <Card className="border-2 shadow-sm">
              <CardContent className="p-2">
                <form onSubmit={handleShorten} className="flex gap-2">
                  <Input 
                    type="url" 
                    placeholder="https://your-very-long-url.com/goes/here" 
                    className="border-0 focus-visible:ring-0 focus-visible:ring-offset-0 text-base h-12"
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    required
                  />
                  <Button type="submit" size="lg" className="h-12 shrink-0" disabled={isLoading}>
                    {isLoading ? 'Shortening...' : 'Shorten'}
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </Button>
                </form>
              </CardContent>
            </Card>
            
            {shortUrl && (
              <div className="mt-4 p-4 bg-muted/50 rounded-lg border flex items-center justify-between animate-in fade-in slide-in-from-bottom-2">
                <div className="flex flex-col items-start overflow-hidden mr-4">
                  <span className="text-sm text-muted-foreground mb-1">Your short link is ready:</span>
                  <a href={shortUrl} target="_blank" rel="noreferrer" className="text-lg font-medium text-primary hover:underline truncate max-w-full">
                    {shortUrl}
                  </a>
                </div>
                <CopyButton text={shortUrl} variant="default" />
              </div>
            )}
          </div>
        </section>

        {/* Features Section */}
        <section className="py-20 bg-muted/30 px-4">
          <div className="max-w-5xl mx-auto">
            <div className="text-center mb-16">
              <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-4">Everything you need</h2>
              <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
                SnapLink provides all the tools you need to manage your links effectively.
              </p>
            </div>
            
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {features.map((feature) => (
                <Card key={feature.title} className="border-none shadow-none bg-transparent">
                  <CardHeader>
                    <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4">
                      <feature.icon className="h-5 w-5 text-primary" />
                    </div>
                    <CardTitle className="text-xl">{feature.title}</CardTitle>
                    <CardDescription className="text-base">{feature.description}</CardDescription>
                  </CardHeader>
                </Card>
              ))}
            </div>
          </div>
        </section>
        
        {/* CTA Section */}
        <section className="py-20 px-4 text-center max-w-3xl mx-auto">
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-6">Ready to get started?</h2>
          <p className="text-lg text-muted-foreground mb-8">
            Join thousands of users who trust SnapLink for their URL shortening needs.
          </p>
          <div className="flex justify-center gap-4">
            <Link href="/register">
              <Button size="lg" className="h-12 px-8">Create an account</Button>
            </Link>
            <Link href="/login">
              <Button size="lg" variant="outline" className="h-12 px-8">Log in</Button>
            </Link>
          </div>
        </section>
      </main>

      <footer className="border-t py-8 px-4 mt-auto">
        <div className="max-w-5xl mx-auto flex flex-col md:flex-row justify-between items-center gap-4">
          <div className="flex items-center gap-2">
            <LinkIcon className="h-5 w-5" />
            <span className="font-bold">SnapLink</span>
          </div>
          <p className="text-sm text-muted-foreground">
            &copy; {new Date().getFullYear()} SnapLink. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
