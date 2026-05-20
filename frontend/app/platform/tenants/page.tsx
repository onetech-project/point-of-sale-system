'use client';

import React, { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { Download, Filter, Search } from 'lucide-react';
import { PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, PlatformTenant } from '@/services/platform';

function statusClass(status: string): string {
  if (status === 'active') return 'bg-green-100 text-green-700';
  if (status === 'trial') return 'bg-blue-100 text-blue-700';
  if (status === 'grace_period' || status === 'suspended') return 'bg-yellow-100 text-yellow-700';
  if (status === 'inactive' || status === 'expired' || status === 'cancelled') return 'bg-red-100 text-red-700';
  return 'bg-gray-100 text-gray-700';
}

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

function formatDate(value?: string): string {
  if (!value) return '-';
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
}

const LIMIT = 50;

export default function PlatformTenantsPage() {
  const [tenants, setTenants] = useState<PlatformTenant[]>([]);
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const [subscriptionStatus, setSubscriptionStatus] = useState('');
  const [plan, setPlan] = useState('');
  const [expiresWithinDays, setExpiresWithinDays] = useState('');
  const [sort, setSort] = useState('-created_at');
  const [offset, setOffset] = useState(0);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const params = useMemo(
    () => ({
      q,
      status,
      subscription_status: subscriptionStatus,
      plan,
      expires_within_days: expiresWithinDays,
      sort,
      limit: LIMIT,
      offset,
    }),
    [q, status, subscriptionStatus, plan, expiresWithinDays, sort, offset]
  );

  const load = () => {
    setLoading(true);
    platformService
      .listTenants(params)
      .then(response => {
        setTenants(response.tenants);
        setTotal(response.total);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [offset]);

  const applyFilters = () => {
    setOffset(0);
    setLoading(true);
    platformService
      .listTenants({ ...params, offset: 0 })
      .then(response => {
        setTenants(response.tenants);
        setTotal(response.total);
      })
      .finally(() => setLoading(false));
  };

  const exportCsv = () => {
    const header = ['business', 'slug', 'owner', 'tenant_status', 'subscription_status', 'plan', 'cycle', 'expires_at', 'paid_total'];
    const rows = tenants.map(tenant => [
      tenant.business_name,
      tenant.slug,
      tenant.owner_email ?? '',
      tenant.status,
      tenant.subscription_status,
      tenant.subscription_plan,
      tenant.billing_cycle,
      tenant.expires_at ?? '',
      String(tenant.paid_invoice_total_idr),
    ]);
    const csv = [header, ...rows]
      .map(row => row.map(cell => `"${cell.replaceAll('"', '""')}"`).join(','))
      .join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'platform-tenants.csv';
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Tenant Registry</h1>
            <p className="mt-1 text-sm text-gray-500">All registered tenants with subscription health and account controls.</p>
          </div>
          <button
            type="button"
            onClick={exportCsv}
            className="inline-flex w-fit items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
          >
            <Download className="h-4 w-4" aria-hidden="true" />
            Export loaded
          </button>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <label className="xl:col-span-2">
              <span className="sr-only">Search tenants</span>
              <div className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2">
                <Search className="h-4 w-4 text-gray-400" aria-hidden="true" />
                <input
                  value={q}
                  onChange={event => setQ(event.target.value)}
                  placeholder="Search business or slug"
                  className="min-w-0 flex-1 border-0 text-sm outline-none"
                />
              </div>
            </label>
            <Select value={status} onChange={setStatus} label="Tenant status" options={['active', 'suspended', 'inactive', 'deleted']} />
            <Select value={subscriptionStatus} onChange={setSubscriptionStatus} label="Subscription" options={['trial', 'active', 'grace_period', 'expired', 'cancelled']} />
            <Select value={plan} onChange={setPlan} label="Plan" options={['trial', 'starter', 'professional', 'enterprise']} />
            <Select value={expiresWithinDays} onChange={setExpiresWithinDays} label="Expires" options={['7', '14', '30']} optionLabels={{ '7': 'Within 7d', '14': 'Within 14d', '30': 'Within 30d' }} />
          </div>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <select
              value={sort}
              onChange={event => setSort(event.target.value)}
              className="w-fit rounded-lg border border-gray-300 px-3 py-2 text-sm"
            >
              <option value="-created_at">Newest registered</option>
              <option value="created_at">Oldest registered</option>
              <option value="business_name">Business A-Z</option>
              <option value="-business_name">Business Z-A</option>
              <option value="expires_at">Expiry soonest</option>
              <option value="-paid_total">Highest paid total</option>
              <option value="-last_active">Recently active</option>
            </select>
            <button onClick={applyFilters} className="inline-flex w-fit items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white">
              <Filter className="h-4 w-4" aria-hidden="true" />
              Apply filters
            </button>
          </div>
        </div>

        <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
          <div className="border-b border-gray-200 px-4 py-3 text-sm text-gray-500">
            Showing {tenants.length} of {total.toLocaleString('id-ID')} tenants
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[980px] text-sm">
              <thead className="bg-gray-50 text-left text-gray-600">
                <tr>
                  <th className="px-4 py-3 font-semibold">Tenant</th>
                  <th className="px-4 py-3 font-semibold">Owner</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                  <th className="px-4 py-3 font-semibold">Subscription</th>
                  <th className="px-4 py-3 font-semibold">Expiry</th>
                  <th className="px-4 py-3 font-semibold">Users</th>
                  <th className="px-4 py-3 font-semibold">Billing paid</th>
                  <th className="px-4 py-3 font-semibold">Tickets</th>
                  <th className="px-4 py-3 text-right font-semibold">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {loading ? (
                  <tr>
                    <td colSpan={9} className="px-4 py-8 text-center text-gray-500">Loading tenants...</td>
                  </tr>
                ) : tenants.length === 0 ? (
                  <tr>
                    <td colSpan={9} className="px-4 py-8 text-center text-gray-500">No tenants found.</td>
                  </tr>
                ) : (
                  tenants.map(tenant => (
                    <tr key={tenant.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <p className="font-medium text-gray-900">{tenant.business_name}</p>
                        <p className="text-xs text-gray-500">{tenant.slug}</p>
                      </td>
                      <td className="px-4 py-3 text-gray-600">{tenant.owner_email || '-'}</td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(tenant.status)}`}>
                          {tenant.status}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-col gap-1">
                          <span className={`w-fit rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(tenant.subscription_status)}`}>
                            {tenant.subscription_status}
                          </span>
                          <span className="text-xs text-gray-500">{tenant.subscription_plan} / {tenant.billing_cycle}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">{formatDate(tenant.expires_at)}</td>
                      <td className="px-4 py-3">{tenant.active_user_count}/{tenant.user_count}</td>
                      <td className="px-4 py-3">{formatCurrency(tenant.paid_invoice_total_idr)}</td>
                      <td className="px-4 py-3">{tenant.open_ticket_count}</td>
                      <td className="px-4 py-3 text-right">
                        <Link href={`/platform/tenants/${tenant.id}`} className="font-medium text-primary-600 hover:underline">
                          Open
                        </Link>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          <div className="flex items-center justify-between border-t border-gray-200 px-4 py-3">
            <button
              type="button"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - LIMIT))}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-sm text-gray-500">Page {Math.floor(offset / LIMIT) + 1}</span>
            <button
              type="button"
              disabled={offset + LIMIT >= total}
              onClick={() => setOffset(offset + LIMIT)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </PlatformShell>
  );
}

function Select({
  value,
  onChange,
  label,
  options,
  optionLabels = {},
}: {
  value: string;
  onChange: (value: string) => void;
  label: string;
  options: string[];
  optionLabels?: Record<string, string>;
}) {
  return (
    <select
      value={value}
      onChange={event => onChange(event.target.value)}
      className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
      aria-label={label}
    >
      <option value="">{label}: all</option>
      {options.map(option => (
        <option key={option} value={option}>
          {optionLabels[option] ?? option.replaceAll('_', ' ')}
        </option>
      ))}
    </select>
  );
}
