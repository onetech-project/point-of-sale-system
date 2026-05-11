'use client';

import { useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import apiClient from '@/services/api';
import { useSubscription } from '@/store/subscription';

export function useSubscriptionErrorHandler() {
  const { invalidateSubscription, setExpired } = useSubscription();
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    apiClient.setSubscriptionExpiredHandler(() => {
      // Keep subscription recovery usable; every other protected screen redirects there.
      if (pathname?.startsWith('/subscription')) return;
      invalidateSubscription();
      setExpired(true);
      router.replace('/subscription?reason=expired');
    });

    return () => {
      apiClient.setSubscriptionExpiredHandler(() => {});
    };
  }, [invalidateSubscription, setExpired, pathname, router]);
}
