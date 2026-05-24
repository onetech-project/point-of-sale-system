'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/store/auth';
import { ROLES } from '@/constants/roles';
import { tenantService } from '@/services/tenant';

const MidtransConfigBanner: React.FC = () => {
  const { user, isAuthenticated, isLoading } = useAuth();
  const [midtransConfigured, setMidtransConfigured] = useState<boolean | null>(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    if (isLoading || !isAuthenticated) return;

    let mounted = true;
    tenantService
      .getTenantInfo()
      .then(tenant => {
        if (mounted) {
          setMidtransConfigured(tenant.midtrans_configured ?? true);
        }
      })
      .catch(error => {
        console.error('Failed to load Midtrans configuration status:', error);
        if (mounted) {
          setMidtransConfigured(true);
        }
      });

    return () => {
      mounted = false;
    };
  }, [isAuthenticated, isLoading]);

  if (dismissed || midtransConfigured !== false) return null;

  const canConfigure = user?.role === ROLES.OWNER;

  return (
    <div className="flex items-center justify-between gap-4 bg-amber-500 px-4 py-2 text-sm font-medium text-white">
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span>Midtrans is not configured. Guest ordering is currently unavailable.</span>
        {canConfigure ? (
          <Link href="/settings/payment" className="underline font-semibold hover:opacity-80">
            Configure Payment Settings
          </Link>
        ) : (
          <span className="font-semibold">Contact an owner.</span>
        )}
      </div>
      <button
        onClick={() => setDismissed(true)}
        aria-label="Dismiss Midtrans configuration banner"
        className="ml-2 rounded p-1 transition-colors hover:bg-white/20"
      >
        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>
  );
};

export default MidtransConfigBanner;
