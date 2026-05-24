'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { billingService, BillingInvoice, BillingPaymentAttempt } from '@/services/billing';
import { redirectToPayment } from '@/utils/paymentRedirect';
import { useAuth } from '@/store/auth';
import { ROLES } from '@/constants/roles';

function formatCurrencyIDR(amount: number): string {
  return `Rp\u00a0${amount.toLocaleString('id-ID')}`;
}

function formatDateTimeIDR(dateStr?: string): string {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function statusClass(status: string): string {
  if (status === 'paid' || status === 'completed') return 'bg-green-100 text-green-700';
  if (status === 'pending') return 'bg-yellow-100 text-yellow-700';
  if (status === 'failed' || status === 'expired') return 'bg-red-100 text-red-700';
  return 'bg-gray-100 text-gray-700';
}

export default function InvoiceDetailPage() {
  const params = useParams();
  const invoiceId = params?.invoiceId as string;
  const { user } = useAuth();
  const [invoice, setInvoice] = useState<BillingInvoice | null>(null);
  const [attempts, setAttempts] = useState<BillingPaymentAttempt[]>([]);
  const [loading, setLoading] = useState(true);
  const [paying, setPaying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const canManageSubscription = user?.role === ROLES.OWNER || user?.role === ROLES.MANAGER;

  useEffect(() => {
    if (!invoiceId) return;
    Promise.all([
      billingService.getInvoice(invoiceId),
      billingService.getInvoicePaymentAttempts(invoiceId),
    ])
      .then(([inv, paymentAttempts]) => {
        setInvoice(inv);
        setAttempts(paymentAttempts);
      })
      .catch(err => {
        console.error('Failed to load invoice:', err);
        setError('Failed to load invoice details.');
      })
      .finally(() => setLoading(false));
  }, [invoiceId]);

  const handlePay = async () => {
    if (!invoice) return;
    if (!canManageSubscription) {
      setError('Subscription payments are managed by owners and managers.');
      return;
    }
    try {
      setPaying(true);
      const result = await billingService.initiatePayment(invoice.id);
      if (result.payment_url) {
        redirectToPayment(result.payment_url);
      }
    } catch (err: any) {
      console.error('Failed to initiate payment:', err);
      setError(err.response?.data?.message ?? 'Failed to initiate payment.');
    } finally {
      setPaying(false);
    }
  };

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="mx-auto max-w-4xl py-8">
          <div className="mb-6 flex items-center justify-between gap-4 print:hidden">
            <div className="flex items-center gap-4">
              <Link
                href="/subscription/invoices"
                className="text-sm text-gray-500 hover:text-gray-700"
              >
                Back
              </Link>
              <h1 className="text-2xl font-bold text-gray-900">Invoice Detail</h1>
            </div>
            <button
              type="button"
              onClick={() => window.print()}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
            >
              Print
            </button>
          </div>

          {loading && <div className="h-48 animate-pulse rounded-xl bg-gray-200" />}
          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {invoice && (
            <div className="space-y-6">
              <div className="rounded-xl border border-gray-200 bg-white p-8 print:border-0 print:p-0">
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <p className="text-sm font-semibold uppercase tracking-wide text-gray-500">
                      Subscription Invoice
                    </p>
                    <h2 className="mt-2 font-mono text-2xl font-bold text-gray-900">
                      {invoice.invoice_number}
                    </h2>
                    {invoice.status === 'paid' && (
                      <span className="mt-3 inline-flex rounded-full bg-green-100 px-3 py-1 text-sm font-bold uppercase text-green-700">
                        Paid Receipt
                      </span>
                    )}
                  </div>
                  <div className="text-left sm:text-right">
                    <span
                      className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase ${statusClass(
                        invoice.status
                      )}`}
                    >
                      {invoice.status}
                    </span>
                    <p className="mt-3 text-3xl font-bold text-gray-900">
                      {formatCurrencyIDR(invoice.amount_idr)}
                    </p>
                  </div>
                </div>

                <div className="mt-8 grid gap-4 border-t border-gray-200 pt-6 sm:grid-cols-2">
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">
                      Billing interval
                    </p>
                    <p className="mt-1 font-medium capitalize text-gray-900">
                      {invoice.billing_interval}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">Due date</p>
                    <p className="mt-1 font-medium text-gray-900">
                      {formatDateTimeIDR(invoice.due_at)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">Service period</p>
                    <p className="mt-1 font-medium text-gray-900">
                      {formatDateTimeIDR(invoice.period_start)} -{' '}
                      {formatDateTimeIDR(invoice.period_end)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">Paid at</p>
                    <p className="mt-1 font-medium text-gray-900">
                      {formatDateTimeIDR(invoice.paid_at)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">
                      Payment link expires
                    </p>
                    <p className="mt-1 font-medium text-gray-900">
                      {formatDateTimeIDR(invoice.payment_link_expires_at)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-gray-500">Midtrans order</p>
                    <p className="mt-1 break-all font-medium text-gray-900">
                      {invoice.midtrans_order_id ?? '-'}
                    </p>
                  </div>
                </div>

                {invoice.status === 'pending' && canManageSubscription && (
                  <button
                    type="button"
                    onClick={handlePay}
                    disabled={paying}
                    className="mt-8 rounded-lg bg-primary-600 px-5 py-3 text-sm font-semibold text-white hover:bg-primary-700 disabled:opacity-60 print:hidden"
                  >
                    {paying ? 'Opening payment...' : 'Pay Invoice'}
                  </button>
                )}
              </div>

              <div className="rounded-xl border border-gray-200 bg-white p-6 print:hidden">
                <h3 className="font-semibold text-gray-900">Payment History</h3>
                <div className="mt-4 divide-y divide-gray-100">
                  {attempts.length === 0 ? (
                    <p className="py-6 text-sm text-gray-500">No payment attempts recorded yet.</p>
                  ) : (
                    attempts.map(attempt => (
                      <div
                        key={attempt.id}
                        className="flex flex-col gap-2 py-4 sm:flex-row sm:items-center sm:justify-between"
                      >
                        <div>
                          <p className="font-medium text-gray-900">
                            {attempt.payment_method || 'Payment link'} -{' '}
                            {formatCurrencyIDR(attempt.amount_idr)}
                          </p>
                          <p className="text-sm text-gray-500">
                            {formatDateTimeIDR(attempt.created_at)}
                          </p>
                          {attempt.error_msg && (
                            <p className="text-sm text-red-600">{attempt.error_msg}</p>
                          )}
                        </div>
                        <span
                          className={`inline-flex w-fit rounded-full px-2.5 py-1 text-xs font-semibold uppercase ${statusClass(
                            attempt.status
                          )}`}
                        >
                          {attempt.status}
                        </span>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
