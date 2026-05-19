'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { PlatformMetric, PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, PlatformOverview } from '@/services/platform';

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

export default function PlatformDashboardPage() {
  const [overview, setOverview] = useState<PlatformOverview | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    platformService
      .getOverview()
      .then(setOverview)
      .catch(err => {
        console.error('Failed to load platform overview:', err);
        setError('Failed to load platform analytics.');
      });
  }, []);

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Platform Dashboard</h1>
          <p className="mt-1 text-sm text-gray-500">
            Subscription revenue and tenant GMV are tracked separately.
          </p>
        </div>
        {error && <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>}
        {!overview ? (
          <div className="h-48 animate-pulse rounded-xl bg-gray-200" />
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <PlatformMetric label="Active tenants" value={overview.active_tenants} tone="success" />
              <PlatformMetric label="Active users" value={overview.active_users} />
              <PlatformMetric label="Recently active users" value={overview.recently_active_users} />
              <PlatformMetric label="Open tickets" value={overview.open_tickets} tone="warning" />
              <PlatformMetric label="Grace period tenants" value={overview.grace_period_tenants} tone="warning" />
              <PlatformMetric label="Expired tenants" value={overview.expired_tenants} tone="danger" />
              <PlatformMetric label="Billing income" value={formatCurrency(overview.billing_income_idr)} />
              <PlatformMetric label="Tenant sales GMV" value={formatCurrency(overview.tenant_sales_gmv_idr)} />
            </div>

            <div className="rounded-xl border border-gray-200 bg-white p-6">
              <div className="flex items-center justify-between">
                <h2 className="font-semibold text-gray-900">Most Sales Tenants</h2>
                <Link href="/platform/tenants" className="text-sm font-medium text-primary-600 hover:underline">
                  Manage tenants
                </Link>
              </div>
              <div className="mt-4 divide-y divide-gray-100">
                {overview.top_sales_tenants.length === 0 ? (
                  <p className="py-6 text-sm text-gray-500">No tenant sales in this period.</p>
                ) : (
                  overview.top_sales_tenants.map(tenant => (
                    <Link
                      key={tenant.tenant_id}
                      href={`/platform/tenants/${tenant.tenant_id}`}
                      className="flex items-center justify-between py-3 hover:bg-gray-50"
                    >
                      <div>
                        <p className="font-medium text-gray-900">{tenant.business_name}</p>
                        <p className="text-sm text-gray-500">{tenant.orders} complete orders</p>
                      </div>
                      <p className="font-semibold text-gray-900">{formatCurrency(tenant.tenant_sales_gmv_idr)}</p>
                    </Link>
                  ))
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </PlatformShell>
  );
}
