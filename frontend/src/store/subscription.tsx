'use client';

import { createContext, useContext, useState, ReactNode } from 'react';

interface SubscriptionContextType {
  isExpired: boolean;
  setExpired: (expired: boolean) => void;
}

const SubscriptionContext = createContext<SubscriptionContextType | null>(null);

export function SubscriptionProvider({ children }: { children: ReactNode }) {
  const [isExpired, setExpired] = useState(false);

  return (
    <SubscriptionContext.Provider value={{ isExpired, setExpired }}>
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

