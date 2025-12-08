'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams, useSearchParams } from 'next/navigation';
import { useTranslation } from 'react-i18next';

/**
 * Payment Return Page (T070)
 * Handles redirect from Midtrans payment page after payment completion
 * 
 * Midtrans redirects to this page with query parameters:
 * - order_id: The order reference
 * - status_code: Payment status code
 * - transaction_status: Transaction status (settlement, pending, cancel, etc.)
 */
export default function PaymentReturnPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useTranslation();
  const [processing, setProcessing] = useState(true);
  const [message, setMessage] = useState('Processing payment result...');

  useEffect(() => {
    const order_id = searchParams?.get('order_id');
    const status_code = searchParams?.get('status_code');
    const transaction_status = searchParams?.get('transaction_status');

    // Handle payment result
    handlePaymentReturn(
      order_id || '',
      status_code || '',
      transaction_status || ''
    );
  }, [searchParams]);

  const handlePaymentReturn = (
    orderReference: string,
    statusCode: string,
    transactionStatus: string
  ) => {
    console.log('Payment return:', {
      orderReference,
      statusCode,
      transactionStatus,
    });

    // Determine payment result based on transaction status
    if (!orderReference) {
      // No order reference - invalid return
      setMessage('Invalid payment return. Missing order reference.');
      setTimeout(() => {
        router.push('/');
      }, 3000);
      return;
    }

    // Show appropriate message based on transaction status
    switch (transactionStatus) {
      case 'capture':
      case 'settlement':
        // Payment successful
        setMessage(
          t(
            'payment.success',
            'Payment successful! Redirecting to order confirmation...'
          )
        );
        setTimeout(() => {
          router.push(`/orders/${orderReference}`);
        }, 2000);
        break;

      case 'pending':
        // Payment pending (e.g., waiting for bank transfer)
        setMessage(
          t(
            'payment.pending',
            'Payment is pending. Redirecting to order status...'
          )
        );
        setTimeout(() => {
          router.push(`/orders/${orderReference}`);
        }, 2000);
        break;

      case 'deny':
      case 'cancel':
      case 'expire':
        // Payment failed or cancelled
        setMessage(
          t(
            'payment.failed',
            'Payment was not completed. Redirecting to order status...'
          )
        );
        setTimeout(() => {
          router.push(`/orders/${orderReference}`);
        }, 2000);
        break;

      default:
        // Unknown status - redirect to order page anyway
        setMessage(
          t('payment.unknown', 'Redirecting to order confirmation...')
        );
        setTimeout(() => {
          router.push(`/orders/${orderReference}`);
        }, 2000);
    }

    setProcessing(false);
  };

  const transactionStatus = searchParams?.get('transaction_status');
  const orderId = searchParams?.get('order_id');

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
        <div className="text-center">
          {/* Loading Spinner */}
          {processing && (
            <div className="inline-block animate-spin rounded-full h-16 w-16 border-b-4 border-blue-600 mb-6"></div>
          )}

          {/* Success Icon */}
          {!processing &&
            (transactionStatus === 'capture' ||
              transactionStatus === 'settlement') && (
              <div className="inline-flex items-center justify-center w-16 h-16 bg-green-100 rounded-full mb-6">
                <svg
                  className="w-8 h-8 text-green-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </div>
            )}

          {/* Pending Icon */}
          {!processing && transactionStatus === 'pending' && (
            <div className="inline-flex items-center justify-center w-16 h-16 bg-yellow-100 rounded-full mb-6">
              <svg
                className="w-8 h-8 text-yellow-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
          )}

          {/* Error Icon */}
          {!processing &&
            (transactionStatus === 'deny' ||
              transactionStatus === 'cancel' ||
              transactionStatus === 'expire') && (
              <div className="inline-flex items-center justify-center w-16 h-16 bg-red-100 rounded-full mb-6">
                <svg
                  className="w-8 h-8 text-red-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </div>
            )}

          {/* Message */}
          <h2 className="text-xl font-bold text-gray-900 mb-2">
            {t('payment.title', 'Payment Processing')}
          </h2>
          <p className="text-gray-600">{message}</p>

          {/* Payment Details */}
          {orderId && (
            <div className="mt-6 p-4 bg-gray-50 rounded-lg">
              <p className="text-sm text-gray-500 mb-1">
                {t('payment.orderReference', 'Order Reference')}
              </p>
              <p className="text-lg font-mono font-bold text-gray-900">
                {orderId}
              </p>
            </div>
          )}

          {/* Loading Dots */}
          {processing && (
            <div className="mt-6 flex justify-center gap-2">
              <div className="w-2 h-2 bg-blue-600 rounded-full animate-bounce"></div>
              <div
                className="w-2 h-2 bg-blue-600 rounded-full animate-bounce"
                style={{ animationDelay: '0.1s' }}
              ></div>
              <div
                className="w-2 h-2 bg-blue-600 rounded-full animate-bounce"
                style={{ animationDelay: '0.2s' }}
              ></div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
