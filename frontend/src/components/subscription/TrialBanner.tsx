'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useSubscription } from '@/store/subscription';

const TrialBanner: React.FC = () => {
  const { subscription, refreshSubscription } = useSubscription();
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    void refreshSubscription();
  }, [refreshSubscription]);

  if (!subscription) return null;
  if (dismissed) return null;
  const status = subscription.status || subscription.subscription_status;
  if (status !== 'trial' && status !== 'grace_period') return null;

  const isGracePeriod = status === 'grace_period';
  const cleanupDate = subscription.retention_cleanup_at
    ? new Date(subscription.retention_cleanup_at).toLocaleDateString('id-ID', {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
      })
    : null;

  return (
    <div
      className={`flex items-center justify-between px-4 py-2 text-sm font-medium text-white ${
        isGracePeriod ? 'bg-red-600' : 'bg-primary-600'
      }`}
    >
      <div className="flex items-center gap-2">
        {isGracePeriod ? (
          <span>
            Subscription grace period. Subscribe now
            {cleanupDate ? ` before data cleanup on ${cleanupDate}.` : ' to avoid data cleanup.'}
          </span>
        ) : (
          <span>
            {subscription.trial_days_remaining ?? subscription.days_remaining ?? 0} days left in
            your free trial.
          </span>
        )}
        <Link
          href="/subscription"
          className="underline font-semibold hover:opacity-80 transition-opacity"
        >
          {isGracePeriod ? 'Subscribe Now' : 'Upgrade Now'}
        </Link>
      </div>
      {!isGracePeriod && status === 'trial' && (
        <button
          onClick={() => setDismissed(true)}
          aria-label="Dismiss banner"
          className="ml-4 p-1 rounded hover:bg-white/20 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      )}
    </div>
  );
};

export default TrialBanner;
