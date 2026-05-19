'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  ReactNode,
} from 'react';
import { billingService, BillingSubscriptionUI } from '@/services/billing';
import { useAuth } from '@/store/auth';

const SUBSCRIPTION_CACHE_TTL_MS = 60 * 1000;

interface SubscriptionContextType {
  subscription: BillingSubscriptionUI | null;
  isLoading: boolean;
  error: string | null;
  isExpired: boolean;
  setExpired: (expired: boolean) => void;
  refreshSubscription: (options?: {
    force?: boolean;
    allowUnauthenticated?: boolean;
  }) => Promise<BillingSubscriptionUI | null>;
  invalidateSubscription: () => void;
  clearSubscription: () => void;
}

const SubscriptionContext = createContext<SubscriptionContextType | null>(null);

export function SubscriptionProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [subscription, setSubscriptionState] = useState<BillingSubscriptionUI | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isExpired, setExpired] = useState(false);
  const lastFetchedAtRef = useRef(0);
  const subscriptionRef = useRef<BillingSubscriptionUI | null>(null);
  const pendingRequestRef = useRef<Promise<BillingSubscriptionUI | null> | null>(null);

  const setSubscription = useCallback((next: BillingSubscriptionUI | null) => {
    subscriptionRef.current = next;
    setSubscriptionState(next);
    setExpired(
      next?.subscription_status === 'expired' ||
        next?.status === 'expired' ||
        next?.subscription_status === 'cancelled' ||
        next?.status === 'cancelled'
    );
  }, []);

  const clearSubscription = useCallback(() => {
    lastFetchedAtRef.current = 0;
    pendingRequestRef.current = null;
    setError(null);
    setIsLoading(false);
    setSubscription(null);
  }, [setSubscription]);

  const invalidateSubscription = useCallback(() => {
    lastFetchedAtRef.current = 0;
  }, []);

  const refreshSubscription = useCallback(
    async (options?: { force?: boolean; allowUnauthenticated?: boolean }) => {
      if (!isAuthenticated && !options?.allowUnauthenticated) {
        clearSubscription();
        return null;
      }

      const force = options?.force ?? false;
      const now = Date.now();
      const cached = subscriptionRef.current;
      const cacheIsFresh = cached && now - lastFetchedAtRef.current < SUBSCRIPTION_CACHE_TTL_MS;

      if (!force && cacheIsFresh) {
        return cached;
      }

      if (pendingRequestRef.current) {
        return pendingRequestRef.current;
      }

      setIsLoading(true);
      const request = billingService
        .getSubscription()
        .then((next) => {
          lastFetchedAtRef.current = Date.now();
          setError(null);
          setSubscription(next);
          return next;
        })
        .catch((err) => {
          console.error('Failed to refresh subscription:', err);
          setError('Failed to load subscription status.');
          return null;
        })
        .finally(() => {
          pendingRequestRef.current = null;
          setIsLoading(false);
        });

      pendingRequestRef.current = request;
      return request;
    },
    [clearSubscription, isAuthenticated, setSubscription]
  );

  useEffect(() => {
    if (authLoading) return;
    if (!isAuthenticated) {
      clearSubscription();
      return;
    }
    void refreshSubscription();
  }, [authLoading, clearSubscription, isAuthenticated, refreshSubscription]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const refreshIfStale = () => {
      if (document.visibilityState !== 'visible') return;
      void refreshSubscription();
    };

    document.addEventListener('visibilitychange', refreshIfStale);
    window.addEventListener('focus', refreshIfStale);
    return () => {
      document.removeEventListener('visibilitychange', refreshIfStale);
      window.removeEventListener('focus', refreshIfStale);
    };
  }, [isAuthenticated, refreshSubscription]);

  return (
    <SubscriptionContext.Provider
      value={{
        subscription,
        isLoading,
        error,
        isExpired,
        setExpired,
        refreshSubscription,
        invalidateSubscription,
        clearSubscription,
      }}
    >
      {children}
    </SubscriptionContext.Provider>
  );
}

export function useSubscription(): SubscriptionContextType {
  const context = useContext(SubscriptionContext);
  if (!context) {
    throw new Error('useSubscription must be used within SubscriptionProvider');
  }
  return context;
}
