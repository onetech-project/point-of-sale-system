import type { MetadataRoute } from 'next';
import { absoluteUrl } from './seo';

const publicRoutes = ['/', '/privacy-policy', '/terms-of-service'];

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();

  return publicRoutes.map(route => ({
    url: absoluteUrl(route),
    lastModified: now,
    changeFrequency: route === '/' ? 'weekly' : 'monthly',
    priority: route === '/' ? 1 : 0.5,
  }));
}
