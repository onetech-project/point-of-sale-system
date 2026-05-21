import { Inter } from 'next/font/google';
import { AuthProvider } from '@/store/auth';
import { SubscriptionProvider } from '@/store/subscription';
import { I18nProvider } from '@/i18n/provider';
import { VersionUpdateBanner } from '@/components/common/VersionUpdateBanner';
import '@/styles/globals.css';

const inter = Inter({ subsets: ['latin'] });

export const metadata = {
  title: 'Posku',
  description: 'Modern Point of Sale System',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <I18nProvider>
          <AuthProvider>
            <SubscriptionProvider>
              {children}
              <VersionUpdateBanner />
            </SubscriptionProvider>
          </AuthProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
