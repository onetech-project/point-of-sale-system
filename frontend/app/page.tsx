import type { Metadata } from 'next';
import LandingPage from '@/components/landing/LandingPage';
import LandingAuthRedirect from '@/components/landing/LandingAuthRedirect';
import { DEFAULT_DESCRIPTION, DEFAULT_TITLE, SITE_NAME, absoluteUrl } from './seo';

export const metadata: Metadata = {
  title: {
    absolute: DEFAULT_TITLE,
  },
  description: DEFAULT_DESCRIPTION,
  alternates: {
    canonical: '/',
  },
  openGraph: {
    title: DEFAULT_TITLE,
    description: DEFAULT_DESCRIPTION,
    url: '/',
    siteName: SITE_NAME,
    locale: 'id_ID',
    type: 'website',
  },
  twitter: {
    card: 'summary',
    title: DEFAULT_TITLE,
    description: DEFAULT_DESCRIPTION,
  },
};

const homeJsonLd = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'Organization',
      '@id': absoluteUrl('/#organization'),
      name: SITE_NAME,
      url: absoluteUrl('/'),
    },
    {
      '@type': 'WebSite',
      '@id': absoluteUrl('/#website'),
      name: SITE_NAME,
      url: absoluteUrl('/'),
      inLanguage: 'id-ID',
      publisher: {
        '@id': absoluteUrl('/#organization'),
      },
    },
    {
      '@type': 'SoftwareApplication',
      '@id': absoluteUrl('/#software'),
      name: SITE_NAME,
      applicationCategory: 'BusinessApplication',
      operatingSystem: 'Web',
      url: absoluteUrl('/'),
      description: DEFAULT_DESCRIPTION,
      inLanguage: 'id-ID',
      offers: {
        '@type': 'Offer',
        price: '299000',
        priceCurrency: 'IDR',
        availability: 'https://schema.org/InStock',
      },
      publisher: {
        '@id': absoluteUrl('/#organization'),
      },
    },
  ],
};

function jsonLdMarkup(value: unknown) {
  return { __html: JSON.stringify(value).replace(/</g, '\\u003c') };
}

export default function Home() {
  return (
    <>
      <LandingAuthRedirect />
      <LandingPage />
      <script type="application/ld+json" dangerouslySetInnerHTML={jsonLdMarkup(homeJsonLd)} />
    </>
  );
}
