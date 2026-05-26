import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import { AuthProvider } from '@/store/auth';
import { SubscriptionProvider } from '@/store/subscription';
import { I18nProvider } from '@/i18n/provider';
import { VersionUpdateBanner } from '@/components/common/VersionUpdateBanner';
import { OnboardingProvider } from '@/components/onboarding/OnboardingProvider';
import '@/styles/globals.css';
import { DEFAULT_DESCRIPTION, DEFAULT_TITLE, SITE_NAME, defaultRobots, getSiteUrl } from './seo';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  metadataBase: new URL(getSiteUrl()),
  applicationName: SITE_NAME,
  title: {
    default: DEFAULT_TITLE,
    template: `%s | ${SITE_NAME}`,
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
  robots: defaultRobots,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id">
      <body className={inter.className}>
        <I18nProvider>
          <AuthProvider>
            <SubscriptionProvider>
              <OnboardingProvider>
                {children}
                <VersionUpdateBanner />
              </OnboardingProvider>
            </SubscriptionProvider>
          </AuthProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
