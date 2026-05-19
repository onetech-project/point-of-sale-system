'use client';

import React, { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { PlatformMetric, PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, PlatformTenant, TenantActivity } from '@/services/platform';

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

function formatDate(value?: string): string {
  if (!value) return '-';
  return new Date(value).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
}

export default function PlatformTenantDetailPage() {
  const params = useParams();
  const tenantId = params?.tenantId as string;
  const [tenant, setTenant] = useState<PlatformTenant | null>(null);
  const [activity, setActivity] = useState<TenantActivity[]>([]);
  const [reason, setReason] = useState('');
  const [loadingAction, setLoadingAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = () => {
    if (!tenantId) return;
    platformService.getTenant(tenantId).then(response => {
      setTenant(response.tenant);
      setActivity(response.activity);
    });
  };

  useEffect(() => {
    load();
  }, [tenantId]);

  const runAction = async (action: 'suspend' | 'deactivate' | 'delete' | 'reactivate' | 'cancel-delete') => {
    if (!tenant) return;
    try {
      setLoadingAction(action);
      setError(null);
      await platformService.tenantAction(tenant.id, action, reason);
      setReason('');
      load();
    } catch (err: any) {
      setError(err.response?.data?.error ?? 'Failed to update tenant.');
    } finally {
      setLoadingAction(null);
    }
  };

  return (
    <PlatformShell>
      {!tenant ? (
        <div className="h-48 animate-pulse rounded-xl bg-gray-200" />
      ) : (
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{tenant.business_name}</h1>
            <p className="mt-1 text-sm text-gray-500">{tenant.slug}</p>
          </div>

          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <PlatformMetric label="Tenant status" value={tenant.status} />
            <PlatformMetric label="Subscription" value={tenant.subscription_status} />
            <PlatformMetric label="Users" value={tenant.user_count} />
            <PlatformMetric label="Billing paid" value={formatCurrency(tenant.paid_invoice_total_idr)} />
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-6">
            <h2 className="font-semibold text-gray-900">Lifecycle Actions</h2>
            <textarea
              value={reason}
              onChange={event => setReason(event.target.value)}
              placeholder="Reason for audit trail"
              className="mt-4 min-h-24 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
            />
            {error && <p className="mt-3 text-sm text-red-700">{error}</p>}
            <div className="mt-4 flex flex-wrap gap-3">
              <button
                onClick={() => runAction('suspend')}
                disabled={!!loadingAction}
                className="rounded-lg border border-yellow-300 bg-yellow-50 px-4 py-2 text-sm font-semibold text-yellow-800 disabled:opacity-60"
              >
                {loadingAction === 'suspend' ? 'Suspending...' : 'Suspend'}
              </button>
              <button
                onClick={() => runAction('deactivate')}
                disabled={!!loadingAction}
                className="rounded-lg border border-red-300 bg-red-50 px-4 py-2 text-sm font-semibold text-red-800 disabled:opacity-60"
              >
                {loadingAction === 'deactivate' ? 'Deactivating...' : 'Deactivate'}
              </button>
              <button
                onClick={() => runAction('delete')}
                disabled={!!loadingAction}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60"
              >
                {loadingAction === 'delete' ? 'Scheduling...' : 'Schedule Delete'}
              </button>
              <button
                onClick={() => runAction('reactivate')}
                disabled={!!loadingAction}
                className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60"
              >
                {loadingAction === 'reactivate' ? 'Reactivating...' : 'Reactivate'}
              </button>
              {tenant.scheduled_delete_at && (
                <button
                  onClick={() => runAction('cancel-delete')}
                  disabled={!!loadingAction}
                  className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-700 disabled:opacity-60"
                >
                  Cancel Delete
                </button>
              )}
            </div>
            {tenant.scheduled_delete_at && (
              <p className="mt-4 text-sm text-red-700">
                Scheduled delete after {formatDate(tenant.scheduled_delete_at)}.
              </p>
            )}
          </div>

          <div className="rounded-xl border border-gray-200 bg-white p-6">
            <h2 className="font-semibold text-gray-900">Activity</h2>
            <div className="mt-4 divide-y divide-gray-100">
              {activity.length === 0 ? (
                <p className="py-6 text-sm text-gray-500">No lifecycle activity yet.</p>
              ) : (
                activity.map((item, index) => (
                  <div key={`${item.action}-${item.occurred_at}-${index}`} className="py-4">
                    <p className="font-medium text-gray-900">{item.action}</p>
                    <p className="text-sm text-gray-500">
                      {formatDate(item.occurred_at)}
                      {item.before_status && item.after_status
                        ? ` - ${item.before_status} to ${item.after_status}`
                        : ''}
                    </p>
                    {item.reason && <p className="mt-1 text-sm text-gray-700">{item.reason}</p>}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </PlatformShell>
  );
}
