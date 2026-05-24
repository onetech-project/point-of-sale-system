'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '../../../../src/components/layout/DashboardLayout';
import { billingService, BillingInvoice } from '@/services/billing';
import { redirectToPayment } from '@/utils/paymentRedirect';
import { ROLES } from '@/constants/roles';
import { useAuth } from '@/store/auth';

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

export default function PayInvoicePage() {
  const params = useParams();
  const invoiceId = params?.invoiceId as string;
  const { user, isLoading: authLoading } = useAuth();
  const canManageSubscription = user?.role === ROLES.OWNER || user?.role === ROLES.MANAGER;

  const [invoice, setInvoice] = useState<BillingInvoice | null>(null);
  const [paymentUrl, setPaymentUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authLoading || !user?.role) return;
    if (!invoiceId) return;
    if (!canManageSubscription) {
      setError('Subscription payments are managed by owners and managers.');
      setLoading(false);
      return;
    }

    const init = async () => {
      try {
        // Fetch invoice details and initiate payment in parallel
        const [inv, payResult] = await Promise.all([
          billingService.getInvoice(invoiceId),
          billingService.initiatePayment(invoiceId),
        ]);
        setInvoice(inv);
        setPaymentUrl(payResult.payment_url);

        // Auto-redirect to payment page
        if (payResult.payment_url) {
          redirectToPayment(payResult.payment_url);
        }
      } catch (err: any) {
        console.error('Failed to initiate payment:', err);
        setError(err.response?.data?.message ?? 'Failed to initiate payment. Please try again.');
      } finally {
        setLoading(false);
      }
    };

    init();
  }, [authLoading, canManageSubscription, invoiceId, user?.role]);

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="max-w-lg mx-auto py-8 space-y-6">
          <div className="flex items-center gap-4">
            <Link
              href="/subscription/invoices"
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
            <h1 className="text-2xl font-bold text-gray-900">Complete Payment</h1>
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {loading ? (
            <div className="bg-white border border-gray-200 rounded-xl p-8 flex flex-col items-center">
              <div className="w-12 h-12 border-4 border-primary-600 border-t-transparent rounded-full animate-spin mb-4" />
              <p className="text-gray-600">Preparing your payment page&hellip;</p>
            </div>
          ) : (
            <>
              {invoice && (
                <div className="bg-white border border-gray-200 rounded-xl p-6 space-y-3">
                  <h2 className="font-semibold text-gray-900">Invoice Summary</h2>
                  <div className="divide-y divide-gray-100 text-sm">
                    <div className="flex justify-between py-2">
                      <span className="text-gray-500">Invoice Number</span>
                      <span className="font-mono text-gray-800">{invoice.invoice_number}</span>
                    </div>
                    <div className="flex justify-between py-2">
                      <span className="text-gray-500">Amount</span>
                      <span className="font-semibold text-gray-900">
                        {formatCurrencyIDR(invoice.amount_idr)}
                      </span>
                    </div>
                    <div className="flex justify-between py-2">
                      <span className="text-gray-500">Period</span>
                      <span className="text-gray-800">
                        {formatDateIDR(invoice.period_start)} &ndash;{' '}
                        {formatDateIDR(invoice.period_end)}
                      </span>
                    </div>
                  </div>
                </div>
              )}

              {paymentUrl && (
                <div className="bg-primary-50 border border-primary-200 rounded-xl p-6 text-center space-y-4">
                  <p className="text-sm text-primary-700">
                    Redirecting to the payment page. If it didn&apos;t open automatically, use the
                    button below.
                  </p>
                  <a
                    href={paymentUrl}
                    className="inline-block bg-primary-600 hover:bg-primary-700 text-white font-semibold py-3 px-6 rounded-lg transition-colors"
                  >
                    Open Payment Page
                  </a>
                </div>
              )}

              <div className="text-center">
                <Link
                  href="/subscription/invoices"
                  className="text-sm text-gray-500 hover:text-gray-700 hover:underline"
                >
                  Back to Invoices
                </Link>
              </div>
            </>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
