'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { AlertTriangle, CalendarPlus, CreditCard, NotebookPen, RefreshCw, ShieldAlert } from 'lucide-react';
import { PlatformMetric, PlatformShell } from '@/components/platform/PlatformShell';
import Modal from '@/components/ui/Modal';
import {
  platformService,
  PlatformTenantDetail,
  TenantAction,
  TenantActionPayload,
  TenantInvoice,
} from '@/services/platform';

function formatCurrency(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`;
}

function formatDate(value?: string): string {
  if (!value) return '-';
  return new Date(value).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
}

function statusClass(status: string): string {
  if (status === 'active' || status === 'paid') return 'bg-green-100 text-green-700';
  if (status === 'trial') return 'bg-blue-100 text-blue-700';
  if (status === 'grace_period' || status === 'pending' || status === 'suspended') return 'bg-yellow-100 text-yellow-700';
  if (status === 'inactive' || status === 'expired' || status === 'cancelled') return 'bg-red-100 text-red-700';
  return 'bg-gray-100 text-gray-700';
}

interface PendingAction {
  action: TenantAction;
  title: string;
  warning: string;
  tone: 'default' | 'danger';
}

export default function PlatformTenantDetailPage() {
  const params = useParams();
  const tenantId = params?.tenantId as string;
  const [detail, setDetail] = useState<PlatformTenantDetail | null>(null);
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [reason, setReason] = useState('');
  const [plan, setPlan] = useState('starter');
  const [billingCycle, setBillingCycle] = useState('monthly');
  const [extendDays, setExtendDays] = useState(7);
  const [note, setNote] = useState('');
  const [ticketId, setTicketId] = useState('');
  const [loadingAction, setLoadingAction] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const tenant = detail?.tenant;
  const invoices = detail?.billing?.invoices ?? [];
  const notes = detail?.notes ?? [];
  const tickets = detail?.tickets ?? [];
  const activity = detail?.activity ?? [];

  const load = () => {
    if (!tenantId) return;
    platformService.getTenant(tenantId).then(response => {
      setDetail(response);
      setPlan(response.tenant.subscription_plan === 'trial' ? 'starter' : response.tenant.subscription_plan);
      setBillingCycle(response.tenant.billing_cycle);
    });
  };

  useEffect(() => {
    load();
  }, [tenantId]);

  const primaryPendingInvoice = useMemo(() => {
    return invoices.find(invoice => invoice.status === 'pending');
  }, [invoices]);

  const openAction = (action: PendingAction) => {
    setPendingAction(action);
    setReason('');
    setNote('');
    setTicketId(tickets[0]?.id ?? '');
    setError(null);
  };

  const runAction = async () => {
    if (!tenant || !pendingAction) return;
    const payload: TenantActionPayload = { reason };
    if (pendingAction.action === 'change-plan') payload.subscription_plan = plan;
    if (pendingAction.action === 'change-billing-cycle') payload.billing_cycle = billingCycle;
    if (pendingAction.action === 'extend-trial' || pendingAction.action === 'extend-grace') payload.extend_days = extendDays;
    if (pendingAction.action === 'add-note') payload.note = note || reason;
    if (pendingAction.action === 'assign-support-owner') payload.ticket_id = ticketId;

    try {
      setLoadingAction(true);
      setError(null);
      await platformService.tenantAction(tenant.id, pendingAction.action, payload);
      setPendingAction(null);
      load();
    } catch (err: any) {
      setError(err.response?.data?.error ?? 'Failed to update tenant.');
    } finally {
      setLoadingAction(false);
    }
  };

  return (
    <PlatformShell>
      {!tenant || !detail ? (
        <div className="h-64 animate-pulse rounded-lg bg-gray-200" />
      ) : (
        <div className="space-y-6">
          <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
            <div className="min-w-0">
              <h1 className="text-2xl font-bold text-gray-900">{tenant.business_name}</h1>
              <p className="mt-1 text-sm text-gray-500">{tenant.slug} - {tenant.owner_email || 'owner unavailable'}</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <ActionButton label="Suspend" tone="warning" onClick={() => openAction({
                action: 'suspend',
                title: 'Suspend tenant',
                warning: 'Suspension immediately blocks tenant account access and terminates active sessions.',
                tone: 'danger',
              })} />
              <ActionButton label="Deactivate" tone="danger" onClick={() => openAction({
                action: 'deactivate',
                title: 'Deactivate tenant',
                warning: 'Deactivation blocks operational access while preserving history for offboarding.',
                tone: 'danger',
              })} />
              <ActionButton label="Schedule delete" tone="danger" onClick={() => openAction({
                action: 'delete',
                title: 'Schedule tenant deletion',
                warning: 'Deletion is scheduled after the configured grace window. This is a high-impact account action.',
                tone: 'danger',
              })} />
              <ActionButton label="Reactivate" onClick={() => openAction({
                action: 'reactivate',
                title: 'Reactivate tenant',
                warning: 'Reactivation restores normal account status for this tenant.',
                tone: 'default',
              })} />
              {tenant.scheduled_delete_at && (
                <ActionButton label="Cancel delete" onClick={() => openAction({
                  action: 'cancel-delete',
                  title: 'Cancel scheduled deletion',
                  warning: 'This cancels the pending tenant deletion request.',
                  tone: 'default',
                })} />
              )}
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <PlatformMetric label="Tenant status" value={tenant.status} tone={tenant.status === 'active' ? 'success' : 'warning'} />
            <PlatformMetric label="Subscription" value={tenant.subscription_status} tone={tenant.subscription_status === 'expired' ? 'danger' : 'default'} />
            <PlatformMetric label="Users active/total" value={`${tenant.active_user_count}/${tenant.user_count}`} />
            <PlatformMetric label="Billing paid" value={formatCurrency(tenant.paid_invoice_total_idr)} />
          </div>

          <div className="grid gap-4 xl:grid-cols-[1fr_1fr]">
            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <h2 className="font-semibold text-gray-900">Subscription Controls</h2>
              <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
                <MetricLine label="Plan" value={tenant.subscription_plan} />
                <MetricLine label="Billing cycle" value={tenant.billing_cycle} />
                <MetricLine label="Trial ends" value={formatDate(tenant.trial_ends_at)} />
                <MetricLine label="Subscription ends" value={formatDate(tenant.subscription_ends_at)} />
                <MetricLine label="Storage" value={`${formatBytes(tenant.storage_used_bytes)} / ${formatBytes(tenant.storage_quota_bytes)}`} />
                <MetricLine label="Last activity" value={formatDate(tenant.last_active_at)} />
              </dl>
              <div className="mt-5 flex flex-wrap gap-2">
                <ActionButton icon={<CreditCard className="h-4 w-4" />} label="Change plan" onClick={() => openAction({
                  action: 'change-plan',
                  title: 'Change subscription plan',
                  warning: 'This changes platform billing metadata. It does not silently charge the tenant.',
                  tone: 'default',
                })} />
                <ActionButton icon={<RefreshCw className="h-4 w-4" />} label="Change cycle" onClick={() => openAction({
                  action: 'change-billing-cycle',
                  title: 'Change billing cycle',
                  warning: 'This changes the preferred billing interval for future subscription billing.',
                  tone: 'default',
                })} />
                <ActionButton icon={<CalendarPlus className="h-4 w-4" />} label="Extend trial" onClick={() => openAction({
                  action: 'extend-trial',
                  title: 'Extend trial',
                  warning: 'This extends the trial expiry date and keeps the tenant in trial status.',
                  tone: 'default',
                })} />
                <ActionButton icon={<CalendarPlus className="h-4 w-4" />} label="Extend grace" onClick={() => openAction({
                  action: 'extend-grace',
                  title: 'Extend grace period',
                  warning: 'This moves the subscription expiry anchor and returns the tenant to grace period.',
                  tone: 'default',
                })} />
                <ActionButton
                  icon={<RefreshCw className="h-4 w-4" />}
                  label="Refresh payment link"
                  disabled={!primaryPendingInvoice}
                  onClick={() => openAction({
                    action: 'regenerate-payment-link',
                    title: 'Refresh payment link window',
                    warning: 'This extends the latest pending invoice payment-link window. It does not create a tenant login session.',
                    tone: 'default',
                  })}
                />
              </div>
            </section>

            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <h2 className="font-semibold text-gray-900">Internal Notes</h2>
              <div className="mt-4 space-y-3">
                {notes.length === 0 ? (
                  <p className="rounded-lg bg-gray-50 p-4 text-sm text-gray-500">No platform notes yet.</p>
                ) : (
                  notes.map(item => (
                    <div key={item.id} className="rounded-lg border border-gray-100 p-3">
                      <p className="text-sm text-gray-800">{item.body}</p>
                      <p className="mt-2 text-xs text-gray-500">{formatDate(item.created_at)}</p>
                    </div>
                  ))
                )}
              </div>
              <button
                type="button"
                onClick={() => openAction({
                  action: 'add-note',
                  title: 'Add internal note',
                  warning: 'This note is visible to platform admins and written to the tenant activity trail.',
                  tone: 'default',
                })}
                className="mt-4 inline-flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm font-semibold text-gray-700"
              >
                <NotebookPen className="h-4 w-4" aria-hidden="true" />
                Add note
              </button>
            </section>
          </div>

          <section className="overflow-hidden rounded-lg border border-gray-200 bg-white">
            <div className="border-b border-gray-200 px-5 py-4">
              <h2 className="font-semibold text-gray-900">Invoices</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px] text-sm">
                <thead className="bg-gray-50 text-left text-gray-600">
                  <tr>
                    <th className="px-4 py-3 font-semibold">Invoice</th>
                    <th className="px-4 py-3 font-semibold">Status</th>
                    <th className="px-4 py-3 font-semibold">Cycle</th>
                    <th className="px-4 py-3 font-semibold">Due</th>
                    <th className="px-4 py-3 font-semibold">Paid</th>
                    <th className="px-4 py-3 text-right font-semibold">Amount</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {invoices.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-gray-500">No invoices.</td>
                    </tr>
                  ) : (
                    invoices.map(invoice => <InvoiceRow key={invoice.id} invoice={invoice} />)
                  )}
                </tbody>
              </table>
            </div>
          </section>

          <div className="grid gap-4 xl:grid-cols-2">
            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <div className="flex items-center justify-between">
                <h2 className="font-semibold text-gray-900">Support Tickets</h2>
                {tickets.length > 0 && (
                  <button
                    type="button"
                    onClick={() => openAction({
                      action: 'assign-support-owner',
                      title: 'Assign support owner',
                      warning: 'This assigns the selected ticket to the current platform admin unless another assignee is provided by API.',
                      tone: 'default',
                    })}
                    className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-semibold text-gray-700"
                  >
                    Assign
                  </button>
                )}
              </div>
              <div className="mt-4 space-y-3">
                {tickets.length === 0 ? (
                  <p className="rounded-lg bg-gray-50 p-4 text-sm text-gray-500">No support tickets.</p>
                ) : (
                  tickets.map(ticket => (
                    <div key={ticket.id} className="rounded-lg border border-gray-100 p-3">
                      <div className="flex items-start justify-between gap-3">
                        <p className="font-medium text-gray-900">{ticket.subject}</p>
                        <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(ticket.status)}`}>
                          {ticket.status}
                        </span>
                      </div>
                      <p className="mt-1 text-sm text-gray-600">{ticket.description}</p>
                      <p className="mt-2 text-xs text-gray-500">Last activity {formatDate(ticket.last_activity_at)}</p>
                    </div>
                  ))
                )}
              </div>
            </section>

            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <h2 className="font-semibold text-gray-900">Activity</h2>
              <div className="mt-4 divide-y divide-gray-100">
                {activity.length === 0 ? (
                  <p className="py-6 text-sm text-gray-500">No lifecycle activity yet.</p>
                ) : (
                  activity.map((item, index) => (
                    <div key={`${item.action}-${item.occurred_at}-${index}`} className="py-4">
                      <p className="font-medium text-gray-900">{item.action.replaceAll('_', ' ')}</p>
                      <p className="text-sm text-gray-500">
                        {formatDate(item.occurred_at)}
                        {item.before_status && item.after_status ? ` - ${item.before_status} to ${item.after_status}` : ''}
                      </p>
                      {item.reason && <p className="mt-1 text-sm text-gray-700">{item.reason}</p>}
                    </div>
                  ))
                )}
              </div>
            </section>
          </div>
        </div>
      )}

      <Modal
        isOpen={!!pendingAction}
        onClose={() => setPendingAction(null)}
        title={pendingAction?.title ?? 'Tenant action'}
        size="lg"
      >
        {pendingAction && (
          <div className="space-y-4">
            <div className={`flex gap-3 rounded-lg border p-4 ${pendingAction.tone === 'danger' ? 'border-red-200 bg-red-50 text-red-800' : 'border-yellow-200 bg-yellow-50 text-yellow-800'}`}>
              {pendingAction.tone === 'danger' ? <ShieldAlert className="h-5 w-5 shrink-0" /> : <AlertTriangle className="h-5 w-5 shrink-0" />}
              <p className="text-sm">{pendingAction.warning}</p>
            </div>

            {pendingAction.action === 'change-plan' && (
              <label className="block">
                <span className="text-sm font-medium text-gray-700">Plan</span>
                <select value={plan} onChange={event => setPlan(event.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
                  <option value="trial">Trial</option>
                  <option value="starter">Starter</option>
                  <option value="professional">Professional</option>
                  <option value="enterprise">Enterprise</option>
                </select>
              </label>
            )}

            {pendingAction.action === 'change-billing-cycle' && (
              <label className="block">
                <span className="text-sm font-medium text-gray-700">Billing cycle</span>
                <select value={billingCycle} onChange={event => setBillingCycle(event.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
                  <option value="monthly">Monthly</option>
                  <option value="annual">Annual</option>
                </select>
              </label>
            )}

            {(pendingAction.action === 'extend-trial' || pendingAction.action === 'extend-grace') && (
              <label className="block">
                <span className="text-sm font-medium text-gray-700">Days to extend</span>
                <input
                  type="number"
                  min={1}
                  max={365}
                  value={extendDays}
                  onChange={event => setExtendDays(Number(event.target.value))}
                  className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
              </label>
            )}

            {pendingAction.action === 'add-note' && (
              <label className="block">
                <span className="text-sm font-medium text-gray-700">Note</span>
                <textarea
                  value={note}
                  onChange={event => setNote(event.target.value)}
                  className="mt-1 min-h-24 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
              </label>
            )}

            {pendingAction.action === 'assign-support-owner' && (
              <label className="block">
                <span className="text-sm font-medium text-gray-700">Ticket</span>
                <select value={ticketId} onChange={event => setTicketId(event.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
                  {tickets.map(ticket => (
                    <option key={ticket.id} value={ticket.id}>{ticket.subject}</option>
                  ))}
                </select>
              </label>
            )}

            <label className="block">
              <span className="text-sm font-medium text-gray-700">Audit reason</span>
              <textarea
                value={reason}
                onChange={event => setReason(event.target.value)}
                placeholder="Required reason for the platform audit trail"
                className="mt-1 min-h-24 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              />
            </label>

            {error && <p className="text-sm text-red-700">{error}</p>}
            <div className="flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setPendingAction(null)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-700"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={runAction}
                disabled={loadingAction || reason.trim() === ''}
                className={`rounded-lg px-4 py-2 text-sm font-semibold text-white disabled:opacity-50 ${pendingAction.tone === 'danger' ? 'bg-red-600' : 'bg-gray-900'}`}
              >
                {loadingAction ? 'Saving...' : 'Confirm'}
              </button>
            </div>
          </div>
        )}
      </Modal>
    </PlatformShell>
  );
}

function ActionButton({
  label,
  onClick,
  tone = 'default',
  icon,
  disabled = false,
}: {
  label: string;
  onClick: () => void;
  tone?: 'default' | 'warning' | 'danger';
  icon?: React.ReactNode;
  disabled?: boolean;
}) {
  const className =
    tone === 'danger'
      ? 'border-red-300 bg-red-50 text-red-800'
      : tone === 'warning'
        ? 'border-yellow-300 bg-yellow-50 text-yellow-800'
        : 'border-gray-300 bg-white text-gray-700';
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-semibold disabled:opacity-50 ${className}`}
    >
      {icon}
      {label}
    </button>
  );
}

function MetricLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-gray-100 p-3">
      <dt className="text-xs font-medium uppercase text-gray-500">{label}</dt>
      <dd className="mt-1 break-words font-semibold text-gray-900">{value}</dd>
    </div>
  );
}

function InvoiceRow({ invoice }: { invoice: TenantInvoice }) {
  return (
    <tr className="hover:bg-gray-50">
      <td className="max-w-xs px-4 py-3 font-mono text-xs text-gray-900 break-all">{invoice.invoice_number}</td>
      <td className="px-4 py-3">
        <span className={`rounded-full px-2 py-1 text-xs font-semibold uppercase ${statusClass(invoice.status)}`}>
          {invoice.status}
        </span>
      </td>
      <td className="px-4 py-3">{invoice.billing_interval}</td>
      <td className="px-4 py-3">{formatDate(invoice.due_at)}</td>
      <td className="px-4 py-3">{formatDate(invoice.paid_at)}</td>
      <td className="px-4 py-3 text-right font-semibold">{formatCurrency(invoice.amount_idr)}</td>
    </tr>
  );
}

function formatBytes(value: number): string {
  if (value >= 1024 ** 3) return `${(value / 1024 ** 3).toFixed(1)} GB`;
  if (value >= 1024 ** 2) return `${(value / 1024 ** 2).toFixed(1)} MB`;
  return `${value.toLocaleString('id-ID')} B`;
}
