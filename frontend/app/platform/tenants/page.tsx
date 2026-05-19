'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, PlatformTenant } from '@/services/platform';

function statusClass(status: string): string {
  if (status === 'active') return 'bg-green-100 text-green-700';
  if (status === 'suspended') return 'bg-yellow-100 text-yellow-700';
  if (status === 'inactive' || status === 'expired') return 'bg-red-100 text-red-700';
  return 'bg-gray-100 text-gray-700';
}

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

export default function PlatformTenantsPage() {
  const [tenants, setTenants] = useState<PlatformTenant[]>([]);
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    platformService
      .listTenants({ q, status })
      .then(response => setTenants(response.tenants))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Tenant Management</h1>
          <p className="mt-1 text-sm text-gray-500">Suspend, deactivate, schedule deletion, and inspect tenant activity.</p>
        </div>
        <div className="flex flex-col gap-3 rounded-xl border border-gray-200 bg-white p-4 md:flex-row">
          <input
            value={q}
            onChange={event => setQ(event.target.value)}
            placeholder="Search business or slug"
            className="min-w-0 flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
          <select
            value={status}
            onChange={event => setStatus(event.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
            <option value="inactive">Inactive</option>
            <option value="deleted">Deleted</option>
          </select>
          <button onClick={load} className="rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white">
            Filter
          </button>
        </div>

        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-left text-gray-600">
              <tr>
                <th className="px-4 py-3 font-semibold">Tenant</th>
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="hidden px-4 py-3 font-semibold md:table-cell">Subscription</th>
                <th className="hidden px-4 py-3 font-semibold lg:table-cell">Users</th>
                <th className="hidden px-4 py-3 font-semibold lg:table-cell">Billing paid</th>
                <th className="px-4 py-3 text-right font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-gray-500">Loading tenants...</td>
                </tr>
              ) : tenants.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-gray-500">No tenants found.</td>
                </tr>
              ) : (
                tenants.map(tenant => (
                  <tr key={tenant.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <p className="font-medium text-gray-900">{tenant.business_name}</p>
                      <p className="text-xs text-gray-500">{tenant.slug}</p>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(tenant.status)}`}>
                        {tenant.status}
                      </span>
                    </td>
                    <td className="hidden px-4 py-3 md:table-cell">
                      <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(tenant.subscription_status)}`}>
                        {tenant.subscription_status}
                      </span>
                    </td>
                    <td className="hidden px-4 py-3 lg:table-cell">{tenant.user_count}</td>
                    <td className="hidden px-4 py-3 lg:table-cell">{formatCurrency(tenant.paid_invoice_total_idr)}</td>
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
      </div>
    </PlatformShell>
  );
}
