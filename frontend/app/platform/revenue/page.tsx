'use client';

import React, { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { CalendarDays, CircleDollarSign, ReceiptText } from 'lucide-react';
import { PlatformMetric, PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, RevenueSummary } from '@/services/platform';

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

function formatDate(value?: string): string {
  if (!value) return '-';
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
}

function statusClass(status: string): string {
  if (status === 'paid') return 'bg-green-100 text-green-700';
  if (status === 'pending') return 'bg-yellow-100 text-yellow-700';
  if (status === 'expired') return 'bg-red-100 text-red-700';
  return 'bg-gray-100 text-gray-700';
}

export default function PlatformRevenuePage() {
  const [range, setRange] = useState<'month' | 'quarter' | 'year'>('month');
  const [revenue, setRevenue] = useState<RevenueSummary | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    platformService
      .getRevenue({ range })
      .then(setRevenue)
      .finally(() => setLoading(false));
  }, [range]);

  const maxRevenue = useMemo(() => {
    return Math.max(...(revenue?.revenue_timeseries ?? []).map(point => point.paid_income_idr), 1);
  }, [revenue]);

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Subscription Revenue</h1>
            <p className="mt-1 text-sm text-gray-500">Income, overdue invoices, billing-cycle mix, and top paying tenants.</p>
          </div>
          <div className="inline-flex w-fit rounded-lg border border-gray-300 bg-white p-1">
            <button
              type="button"
              onClick={() => setRange('month')}
              className={`rounded-md px-3 py-1.5 text-sm font-medium ${range === 'month' ? 'bg-gray-900 text-white' : 'text-gray-700'}`}
            >
              Month
            </button>
            <button
              type="button"
              onClick={() => setRange('quarter')}
              className={`rounded-md px-3 py-1.5 text-sm font-medium ${range === 'quarter' ? 'bg-gray-900 text-white' : 'text-gray-700'}`}
            >
              Quarter
            </button>
            <button
              type="button"
              onClick={() => setRange('year')}
              className={`rounded-md px-3 py-1.5 text-sm font-medium ${range === 'year' ? 'bg-gray-900 text-white' : 'text-gray-700'}`}
            >
              YTD
            </button>
          </div>
        </div>

        {loading || !revenue ? (
          <div className="h-64 animate-pulse rounded-lg bg-gray-200" />
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <PlatformMetric label="Paid income" value={formatCurrency(revenue.paid_income_idr)} tone="success" />
              <PlatformMetric label="Pending invoices" value={formatCurrency(revenue.pending_income_idr)} tone="warning" />
              <PlatformMetric label="Overdue invoices" value={formatCurrency(revenue.overdue_income_idr)} tone="danger" />
              <PlatformMetric
                label="Period delta"
                value={
                  revenue.period_delta_percent === undefined
                    ? 'n/a'
                    : `${revenue.period_delta_percent >= 0 ? '+' : ''}${revenue.period_delta_percent.toFixed(1)}%`
                }
              />
            </div>

            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <CircleDollarSign className="h-4 w-4 text-gray-500" aria-hidden="true" />
                  <h2 className="font-semibold text-gray-900">Paid Income Timeline</h2>
                </div>
                <p className="text-sm text-gray-500">
                  {formatDate(revenue.start_date)} - {formatDate(revenue.end_date)}
                </p>
              </div>
              <div className="mt-5 flex h-56 items-end gap-2">
                {revenue.revenue_timeseries.length === 0 ? (
                  <div className="flex h-full w-full items-center justify-center rounded-lg bg-gray-50 text-sm text-gray-500">
                    No paid invoices in this range.
                  </div>
                ) : (
                  revenue.revenue_timeseries.map(point => (
                    <div key={point.date} className="flex min-w-8 flex-1 flex-col items-center justify-end gap-2">
                      <div
                        className="w-full rounded-t bg-gray-900"
                        title={`${point.date}: ${formatCurrency(point.paid_income_idr)}`}
                        style={{ height: `${Math.max(8, (point.paid_income_idr / maxRevenue) * 190)}px` }}
                      />
                      <span className="text-[11px] text-gray-500">{new Date(point.date).getDate()}</span>
                    </div>
                  ))
                )}
              </div>
            </section>

            <div className="grid gap-4 xl:grid-cols-2">
              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <div className="mb-4 flex items-center gap-2">
                  <ReceiptText className="h-4 w-4 text-gray-500" aria-hidden="true" />
                  <h2 className="font-semibold text-gray-900">Invoice Mix</h2>
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <p className="mb-2 text-sm font-medium text-gray-500">By status</p>
                    <MetricMap values={revenue.invoice_count_by_status} />
                  </div>
                  <div>
                    <p className="mb-2 text-sm font-medium text-gray-500">By billing cycle</p>
                    <MetricMap values={revenue.billing_cycle_mix} />
                  </div>
                </div>
              </section>

              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <div className="mb-4 flex items-center gap-2">
                  <CalendarDays className="h-4 w-4 text-gray-500" aria-hidden="true" />
                  <h2 className="font-semibold text-gray-900">Top Paying Tenants</h2>
                </div>
                <div className="divide-y divide-gray-100">
                  {revenue.top_paying_tenants.length === 0 ? (
                    <p className="py-6 text-sm text-gray-500">No paid tenants in this range.</p>
                  ) : (
                    revenue.top_paying_tenants.map(tenant => (
                      <Link
                        key={tenant.tenant_id}
                        href={`/platform/tenants/${tenant.tenant_id}`}
                        className="flex items-center justify-between gap-4 py-3 hover:bg-gray-50"
                      >
                        <div className="min-w-0">
                          <p className="font-medium text-gray-900">{tenant.business_name}</p>
                          <p className="text-sm text-gray-500">{tenant.invoice_count} invoices</p>
                        </div>
                        <p className="shrink-0 font-semibold text-gray-900">{formatCurrency(tenant.paid_income_idr)}</p>
                      </Link>
                    ))
                  )}
                </div>
              </section>
            </div>

            <section className="overflow-hidden rounded-lg border border-gray-200 bg-white">
              <div className="border-b border-gray-200 px-5 py-4">
                <h2 className="font-semibold text-gray-900">Overdue Pending Invoices</h2>
              </div>
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-left text-gray-600">
                  <tr>
                    <th className="px-4 py-3 font-semibold">Invoice</th>
                    <th className="px-4 py-3 font-semibold">Tenant</th>
                    <th className="px-4 py-3 font-semibold">Due</th>
                    <th className="px-4 py-3 text-right font-semibold">Amount</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {revenue.overdue_invoices.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="px-4 py-8 text-center text-gray-500">No overdue invoices.</td>
                    </tr>
                  ) : (
                    revenue.overdue_invoices.map(invoice => (
                      <tr key={invoice.id} className="hover:bg-gray-50">
                        <td className="max-w-xs px-4 py-3 font-mono text-xs text-gray-900 break-all">{invoice.invoice_number}</td>
                        <td className="px-4 py-3">
                          <Link href={`/platform/tenants/${invoice.tenant_id}`} className="font-medium text-primary-600 hover:underline">
                            {invoice.tenant_name}
                          </Link>
                        </td>
                        <td className="px-4 py-3">{formatDate(invoice.due_at)}</td>
                        <td className="px-4 py-3 text-right font-semibold">{formatCurrency(invoice.amount_idr)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </section>
          </>
        )}
      </div>
    </PlatformShell>
  );
}

function MetricMap({ values }: { values: Record<string, number> }) {
  const entries = Object.entries(values);
  if (entries.length === 0) return <p className="rounded-lg bg-gray-50 p-4 text-sm text-gray-500">No invoices.</p>;
  return (
    <div className="space-y-2">
      {entries.map(([key, value]) => (
        <div key={key} className="flex items-center justify-between gap-4 rounded-lg border border-gray-100 px-3 py-2">
          <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(key)}`}>{key}</span>
          <span className="font-semibold text-gray-900">{value.toLocaleString('id-ID')}</span>
        </div>
      ))}
    </div>
  );
}
