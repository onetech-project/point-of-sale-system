'use client';

import React, { useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import Sidebar from './Sidebar';
import TrialBanner from '@/components/subscription/TrialBanner';
import { useSubscriptionErrorHandler } from '@/hooks/useSubscriptionErrorHandler';
import { useSubscription } from '@/store/subscription';

interface DashboardLayoutProps {
  children: React.ReactNode;
}

const DashboardLayout: React.FC<DashboardLayoutProps> = ({ children }) => {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const { subscription, refreshSubscription } = useSubscription();
  useSubscriptionErrorHandler();
  const appName = process.env.NEXT_PUBLIC_APP_NAME || 'POS';
  const appInitial = appName.charAt(0).toUpperCase();

  React.useEffect(() => {
    if (pathname?.startsWith('/subscription')) return;
    void refreshSubscription().then((latest) => {
      if ((latest ?? subscription)?.subscription_status === 'expired') {
        router.replace('/subscription?reason=expired');
      }
    });
  }, [pathname, refreshSubscription, router, subscription]);

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50">
      {/* Sidebar */}
      <Sidebar isOpen={isSidebarOpen} onClose={() => setIsSidebarOpen(false)} />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top Bar (Mobile) */}
        <header className="lg:hidden bg-white border-b border-gray-200 h-16 flex items-center px-4">
          <button
            onClick={() => setIsSidebarOpen(true)}
            className="p-2 rounded-lg hover:bg-gray-100 transition-colors"
            aria-label="Open menu"
          >
            <svg
              className="w-6 h-6 text-gray-700"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </button>

          <div className="flex items-center space-x-2 ml-4">
            <div className="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-xl">{appInitial}</span>
            </div>
            <span className="text-xl font-bold text-gray-900">{appName}</span>
          </div>
        </header>

        {/* Trial / Grace Period Banner */}
        <TrialBanner />

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto p-4 lg:p-8">
          <div className="max-w-7xl mx-auto">{children}</div>
        </main>
      </div>
    </div>
  );
};

export default DashboardLayout;
