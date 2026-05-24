'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '../../../src/components/layout/DashboardLayout';
import { billingService, BillingInvoice } from '@/services/billing';
import { redirectToPayment } from '@/utils/paymentRedirect';
import { useAuth } from '@/store/auth';
import { ROLES } from '@/constants/roles';

function formatCurrencyIDR(amount: number): string {
  return `Rp\u00a0${amount.toLocaleString('id-ID')}`;
}

function formatDateIDR(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}

const STATUS_CONFIG: Record<BillingInvoice['status'], { label: string; className: string }> = {
  paid: { label: 'Paid', className: 'bg-green-100 text-green-700' },
  pending: { label: 'Pending', className: 'bg-yellow-100 text-yellow-700' },
  expired: { label: 'Expired', className: 'bg-red-100 text-red-700' },
  cancelled: { label: 'Cancelled', className: 'bg-gray-100 text-gray-600' },
};

export default function InvoicesPage() {
  const { user } = useAuth();
  const [invoices, setInvoices] = useState<BillingInvoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [payingId, setPayingId] = useState<string | null>(null);
  const canManageSubscription = user?.role === ROLES.OWNER || user?.role === ROLES.MANAGER;

  useEffect(() => {
    billingService
      .getInvoices()
      .then(setInvoices)
      .catch(err => {
        console.error('Failed to load invoices:', err);
        setError('Failed to load invoices. Please try again.');
      })
      .finally(() => setLoading(false));
  }, []);

  const handlePay = async (invoice: BillingInvoice) => {
    if (!canManageSubscription) {
      setError('Subscription payments are managed by owners and managers.');
      return;
    }
    try {
      setPayingId(invoice.id);
      const result = await billingService.initiatePayment(invoice.id);
      if (result.payment_url) {
        redirectToPayment(result.payment_url);
      }
    } catch (err: any) {
      console.error('Payment initiation failed:', err);
      setError(err.response?.data?.message ?? 'Failed to initiate payment. Please try again.');
    } finally {
      setPayingId(null);
    }
  };

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="max-w-4xl mx-auto py-8 space-y-6">
          <div className="flex items-center gap-4">
            <Link
              href="/subscription"
              className="text-gray-400 hover:text-gray-600 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            </Link>
            <h1 className="text-2xl font-bold text-gray-900">Invoices</h1>
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {loading ? (
            <div className="animate-pulse space-y-3">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-16 bg-gray-200 rounded-lg" />
              ))}
            </div>
          ) : invoices.length === 0 ? (
            <div className="bg-white border border-gray-200 rounded-xl p-12 text-center">
              <svg
                className="w-12 h-12 text-gray-300 mx-auto mb-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                />
              </svg>
              <p className="text-gray-600 font-medium">No invoices yet</p>
              <p className="text-sm text-gray-400 mt-1">
                Your billing invoices will appear here after you subscribe.
              </p>
            </div>
          ) : (
            <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    <th className="text-left px-4 py-3 font-semibold text-gray-700">Invoice #</th>
                    <th className="text-left px-4 py-3 font-semibold text-gray-700">Amount</th>
                    <th className="text-left px-4 py-3 font-semibold text-gray-700 hidden sm:table-cell">
                      Period
                    </th>
                    <th className="text-left px-4 py-3 font-semibold text-gray-700">Status</th>
                    <th className="text-left px-4 py-3 font-semibold text-gray-700 hidden md:table-cell">
                      Due
                    </th>
                    <th className="text-right px-4 py-3 font-semibold text-gray-700">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {invoices.map(invoice => {
                    const statusCfg = STATUS_CONFIG[invoice.status] ?? STATUS_CONFIG.cancelled;
                    return (
                      <tr key={invoice.id} className="hover:bg-gray-50 transition-colors">
                        <td className="px-4 py-3 font-mono text-gray-800">
                          {invoice.invoice_number}
                        </td>
                        <td className="px-4 py-3 text-gray-800">
                          {formatCurrencyIDR(invoice.amount_idr)}
                        </td>
                        <td className="px-4 py-3 text-gray-600 hidden sm:table-cell">
                          {formatDateIDR(invoice.period_start)} &ndash;{' '}
                          {formatDateIDR(invoice.period_end)}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`px-2 py-0.5 rounded-full text-xs font-semibold ${statusCfg.className}`}
                          >
                            {statusCfg.label}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-600 hidden md:table-cell">
                          {invoice.due_at
                            ? formatDateIDR(invoice.due_at)
                            : formatDateIDR(invoice.created_at)}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-3">
                            <Link
                              href={`/subscription/invoices/${invoice.id}`}
                              className="text-xs text-primary-600 hover:underline"
                            >
                              View
                            </Link>
                            {invoice.status === 'pending' && canManageSubscription && (
                              <button
                                onClick={() => handlePay(invoice)}
                                disabled={payingId === invoice.id}
                                className="text-xs font-semibold bg-primary-600 hover:bg-primary-700 disabled:opacity-60 text-white px-3 py-1.5 rounded-lg transition-colors"
                              >
                                {payingId === invoice.id ? 'Loading...' : 'Pay Now'}
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
