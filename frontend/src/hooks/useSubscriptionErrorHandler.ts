'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import apiClient from '@/services/api';
import { useSubscription } from '@/store/subscription';

export function useSubscriptionErrorHandler() {
  const { setExpired } = useSubscription();
  const pathname = usePathname();

  useEffect(() => {
    apiClient.setSubscriptionExpiredHandler(() => {
      // Don't show the wall when the user is already on the subscription page
      if (pathname?.startsWith('/subscription')) return;
      setExpired(true);
    });

    return () => {
      apiClient.setSubscriptionExpiredHandler(() => {});
    };
  }, [setExpired, pathname]);
}
