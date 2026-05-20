'use client';

import React, { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { AlertTriangle, ArrowUpRight, Building2, CircleDollarSign, Clock, HardDrive, Ticket } from 'lucide-react';
import { PlatformMetric, PlatformShell } from '@/components/platform/PlatformShell';
import { platformService, PlatformOverview, UrgentItem } from '@/services/platform';

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

function formatNumber(value: number): string {
  return value.toLocaleString('id-ID');
}

function formatDate(value?: string): string {
  if (!value) return '-';
  return new Date(value).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
}

function formatBytes(value: number): string {
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(1)} GB`;
  if (value >= 1024 ** 2) return `${(value / 1024 ** 2).toFixed(1)} MB`;
  return `${value.toLocaleString('id-ID')} B`;
}

function severityClass(severity: string): string {
  if (severity === 'danger') return 'border-red-200 bg-red-50 text-red-800';
  if (severity === 'warning') return 'border-yellow-200 bg-yellow-50 text-yellow-800';
  return 'border-gray-200 bg-white text-gray-800';
}

export default function PlatformDashboardPage() {
  const [overview, setOverview] = useState<PlatformOverview | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    platformService
      .getOverview({ range: 'month' })
      .then(setOverview)
      .catch(err => {
        console.error('Failed to load platform overview:', err);
        setError('Failed to load platform analytics.');
      });
  }, []);

  const maxRevenue = useMemo(() => {
    return Math.max(...(overview?.revenue_timeseries ?? []).map(point => point.paid_income_idr), 1);
  }, [overview]);

  return (
    <PlatformShell>
      <div className="space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Platform Command Center</h1>
            <p className="mt-1 text-sm text-gray-500">
              SaaS income, tenant health, support pressure, and account controls.
            </p>
          </div>
          <Link
            href="/platform/revenue"
            className="inline-flex w-fit items-center gap-2 rounded-lg bg-gray-900 px-4 py-2 text-sm font-semibold text-white"
          >
            <CircleDollarSign className="h-4 w-4" aria-hidden="true" />
            Revenue
          </Link>
        </div>

        {error && <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>}
        {!overview ? (
          <div className="h-64 animate-pulse rounded-lg bg-gray-200" />
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <PlatformMetric label="MTD subscription income" value={formatCurrency(overview.paid_income_idr)} tone="success" />
              <PlatformMetric label="Active tenants" value={formatNumber(overview.active_tenants)} />
              <PlatformMetric label="Grace / expired" value={`${overview.grace_period_tenants} / ${overview.expired_tenants}`} tone="warning" />
              <PlatformMetric label="Will expire in 14d" value={formatNumber(overview.will_expire_14)} tone="danger" />
            </div>

            <div className="grid gap-4 xl:grid-cols-[1.4fr_1fr]">
              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="font-semibold text-gray-900">Subscription Income Trend</h2>
                    <p className="text-sm text-gray-500">
                      {formatDate(overview.start_date)} - {formatDate(overview.end_date)}
                    </p>
                  </div>
                  <p className="text-sm font-semibold text-gray-700">
                    {overview.period_delta_percent === undefined
                      ? 'No previous period'
                      : `${overview.period_delta_percent >= 0 ? '+' : ''}${overview.period_delta_percent.toFixed(1)}%`}
                  </p>
                </div>
                <div className="mt-5 flex h-48 items-end gap-2">
                  {overview.revenue_timeseries.length === 0 ? (
                    <div className="flex h-full w-full items-center justify-center rounded-lg bg-gray-50 text-sm text-gray-500">
                      No paid invoices in this period.
                    </div>
                  ) : (
                    overview.revenue_timeseries.map(point => (
                      <div key={point.date} className="flex min-w-8 flex-1 flex-col items-center justify-end gap-2">
                        <div
                          className="w-full rounded-t bg-gray-900"
                          title={`${point.date}: ${formatCurrency(point.paid_income_idr)}`}
                          style={{ height: `${Math.max(8, (point.paid_income_idr / maxRevenue) * 160)}px` }}
                        />
                        <span className="text-[11px] text-gray-500">{new Date(point.date).getDate()}</span>
                      </div>
                    ))
                  )}
                </div>
              </section>

              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <h2 className="font-semibold text-gray-900">Action Queue</h2>
                <div className="mt-4 space-y-3">
                  {overview.urgent_items.length === 0 ? (
                    <p className="rounded-lg bg-gray-50 p-4 text-sm text-gray-500">No urgent platform work right now.</p>
                  ) : (
                    overview.urgent_items.slice(0, 8).map(item => <UrgentRow key={`${item.type}-${item.tenant_id}-${item.label}`} item={item} />)
                  )}
                </div>
              </section>
            </div>

            <div className="grid gap-4 lg:grid-cols-3">
              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <div className="mb-4 flex items-center gap-2">
                  <Building2 className="h-4 w-4 text-gray-500" aria-hidden="true" />
                  <h2 className="font-semibold text-gray-900">Tenant Health Funnel</h2>
                </div>
                <HealthRow label="Registered" value={overview.tenant_health.registered_tenants} max={overview.tenant_health.registered_tenants} />
                <HealthRow label="Active" value={overview.tenant_health.active_tenants} max={overview.tenant_health.registered_tenants} />
                <HealthRow label="Trial" value={overview.tenant_health.trial_tenants} max={overview.tenant_health.registered_tenants} />
                <HealthRow label="Grace" value={overview.tenant_health.grace_period_tenants} max={overview.tenant_health.registered_tenants} />
                <HealthRow label="Expired / cancelled" value={overview.tenant_health.expired_tenants + overview.tenant_health.cancelled_tenants} max={overview.tenant_health.registered_tenants} />
              </section>

              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <div className="mb-4 flex items-center gap-2">
                  <HardDrive className="h-4 w-4 text-gray-500" aria-hidden="true" />
                  <h2 className="font-semibold text-gray-900">System Usage</h2>
                </div>
                <dl className="space-y-3 text-sm">
                  <MetricLine label="Active users" value={formatNumber(overview.active_users)} />
                  <MetricLine label="Recently active" value={formatNumber(overview.recently_active_users)} />
                  <MetricLine label="Storage used" value={formatBytes(overview.storage_used_bytes)} />
                  <MetricLine label="Storage quota" value={formatBytes(overview.storage_quota_bytes)} />
                  <MetricLine label="Tenant GMV" value={formatCurrency(overview.tenant_sales_gmv_idr)} />
                </dl>
              </section>

              <section className="rounded-lg border border-gray-200 bg-white p-5">
                <div className="mb-4 flex items-center gap-2">
                  <Ticket className="h-4 w-4 text-gray-500" aria-hidden="true" />
                  <h2 className="font-semibold text-gray-900">Support Pressure</h2>
                </div>
                <dl className="space-y-3 text-sm">
                  <MetricLine label="Open tickets" value={formatNumber(overview.open_tickets)} />
                  <MetricLine label="High priority" value={formatNumber(overview.high_priority_tickets)} />
                  {Object.entries(overview.tickets_by_status).map(([status, count]) => (
                    <MetricLine key={status} label={status.replaceAll('_', ' ')} value={formatNumber(count)} />
                  ))}
                </dl>
              </section>
            </div>

            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <div className="flex items-center justify-between">
                <h2 className="font-semibold text-gray-900">Top Paying Tenants</h2>
                <Link href="/platform/tenants" className="inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:underline">
                  Manage tenants
                  <ArrowUpRight className="h-3.5 w-3.5" aria-hidden="true" />
                </Link>
              </div>
              <div className="mt-4 divide-y divide-gray-100">
                {overview.top_paying_tenants.length === 0 ? (
                  <p className="py-6 text-sm text-gray-500">No subscription income in this period.</p>
                ) : (
                  overview.top_paying_tenants.map(tenant => (
                    <Link
                      key={tenant.tenant_id}
                      href={`/platform/tenants/${tenant.tenant_id}`}
                      className="flex items-center justify-between gap-4 py-3 hover:bg-gray-50"
                    >
                      <div className="min-w-0">
                        <p className="font-medium text-gray-900">{tenant.business_name}</p>
                        <p className="text-sm text-gray-500">{tenant.invoice_count} paid invoices</p>
                      </div>
                      <p className="shrink-0 font-semibold text-gray-900">{formatCurrency(tenant.paid_income_idr)}</p>
                    </Link>
                  ))
                )}
              </div>
            </section>
          </>
        )}
      </div>
    </PlatformShell>
  );
}

function UrgentRow({ item }: { item: UrgentItem }) {
  return (
    <Link
      href={`/platform/tenants/${item.tenant_id}`}
      className={`flex items-start gap-3 rounded-lg border p-3 ${severityClass(item.severity)}`}
    >
      {item.severity === 'danger' ? <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" /> : <Clock className="mt-0.5 h-4 w-4 shrink-0" />}
      <span className="min-w-0 flex-1">
        <span className="block font-medium">{item.business_name}</span>
        <span className="block text-sm opacity-80">
          {item.label}
          {item.due_at ? ` - ${formatDate(item.due_at)}` : ''}
          {item.amount_idr ? ` - ${formatCurrency(item.amount_idr)}` : ''}
        </span>
      </span>
    </Link>
  );
}

function HealthRow({ label, value, max }: { label: string; value: number; max: number }) {
  const width = max > 0 ? Math.max(3, (value / max) * 100) : 0;
  return (
    <div className="mb-3">
      <div className="mb-1 flex justify-between text-sm">
        <span className="text-gray-600">{label}</span>
        <span className="font-semibold text-gray-900">{formatNumber(value)}</span>
      </div>
      <div className="h-2 rounded bg-gray-100">
        <div className="h-2 rounded bg-gray-900" style={{ width: `${width}%` }} />
      </div>
    </div>
  );
}

function MetricLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="capitalize text-gray-500">{label}</dt>
      <dd className="font-semibold text-gray-900">{value}</dd>
    </div>
  );
}
