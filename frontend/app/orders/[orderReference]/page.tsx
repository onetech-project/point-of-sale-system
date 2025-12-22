'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { useTranslation } from 'react-i18next';
import PublicLayout from '../../../src/components/layout/PublicLayout';
import OrderConfirmation from '../../../src/components/guest/OrderConfirmation';
import { order as orderService } from '../../../src/services/order';
import { OrderData } from '../../../src/types/cart';
import { formatCurrency } from '../../../src/utils/text';

export default function OrderStatusPage() {
  const router = useRouter();
  const params = useParams();
  const orderReference = params?.orderReference as string;
  const { t } = useTranslation();

  const [orderData, setOrderData] = useState<OrderData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

  useEffect(() => {
    if (orderReference) {
      fetchOrder(orderReference);
    }
  }, [orderReference]);

  // Auto-refresh order status every 10 seconds for pending/paid orders
  useEffect(() => {
    if (!autoRefresh || !orderData || !orderReference) return;

    // Only auto-refresh for pending or paid orders
    if (orderData.order.status !== 'PENDING' && orderData.order.status !== 'PAID') {
      setAutoRefresh(false);
      return;
    }

    const interval = setInterval(() => {
      fetchOrder(orderReference, true);
    }, 10000); // 10 seconds

    return () => clearInterval(interval);
  }, [autoRefresh, orderData, orderReference]);

  const fetchOrder = async (reference: string, silent = false) => {
    try {
      if (!silent) {
        setLoading(true);
        setError(null);
      }

      const fetchedOrder = await orderService.getOrderByReference(reference);
      setOrderData(fetchedOrder);

      // Stop auto-refresh if order is completed or cancelled
      if (fetchedOrder.order.status === 'COMPLETE' || fetchedOrder.order.status === 'CANCELLED') {
        setAutoRefresh(false);
      }
    } catch (err: any) {
      console.error('Failed to fetch order:', err);
      if (!silent) {
        if (err.response?.status === 404) {
          setError(
            t(
              'orderStatus.notFound',
              'Order not found. Please check your order reference.'
            )
          );
        } else {
          setError(
            t(
              'orderStatus.error',
              'Failed to load order details. Please try again.'
            )
          );
        }
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  };

  const handleRefresh = () => {
    if (orderReference) {
      fetchOrder(orderReference);
    }
  };

  if (loading) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
          <div className="text-center">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
            <p className="text-gray-600">
              {t('orderStatus.loading', 'Loading order details...')}
            </p>
          </div>
        </div>
      </PublicLayout>
    );
  }

  if (error) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
          <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-6">
            <div className="text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-red-100 rounded-full mb-4">
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
              <h2 className="text-xl font-bold text-gray-900 mb-2">
                {t('orderStatus.errorTitle', 'Order Not Found')}
              </h2>
              <p className="text-gray-600 mb-6">{error}</p>
              <button
                onClick={handleRefresh}
                className="px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors"
              >
                {t('orderStatus.retry', 'Try Again')}
              </button>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  if (!orderData) {
    return null;
  }

  return (
    <PublicLayout>
      <div className="min-h-screen bg-gray-50">
        <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 pb-8">
          {/* Page Header with Back and Refresh buttons */}
          <div className="py-4">
            <div className="flex items-center justify-between">
              <button
                onClick={() => router.push(`/menu/${orderData.order.tenant_id}`)}
                className="flex items-center gap-2 text-gray-600 hover:text-gray-900 transition-colors"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 19l-7-7 7-7"
                  />
                </svg>
                <span className="font-medium">{t('common.checkout.backToMenu', 'Back to Menu')}</span>
              </button>
              <button
                onClick={handleRefresh}
                disabled={loading}
                className="flex items-center gap-2 px-4 py-2 text-sm text-gray-600 hover:text-gray-900 transition-colors"
              >
                <svg
                  className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                  />
                </svg>
                {t('orderStatus.refresh', 'Refresh')}
              </button>
            </div>
          </div>

          {/* Order Confirmation Component */}
          <OrderConfirmation
            orderReference={orderData.order.order_reference}
            deliveryType={orderData.order.delivery_type}
            customerName={orderData.order.customer_name}
            customerPhone={orderData.order.customer_phone}
            deliveryAddress={orderData.order.delivery_address}
            tableNumber={orderData.order.table_number}
            subtotal={orderData.order.subtotal_amount}
            deliveryFee={orderData.order.delivery_fee}
            total={orderData.order.total_amount}
            orderStatus={orderData.order.status}
            createdAt={orderData.order.created_at}
            paymentQrUrl={orderData.order.payment_url}
            paymentInfo={orderData.payment}
            notes={orderData.notes && orderData.notes.length > 0 ? orderData.notes[0].note : undefined}
            items={orderData.items}
            customerNotes={orderData.order.notes}
          />

          {/* Auto-refresh Indicator */}
          {autoRefresh && (orderData.order.status === 'PENDING' || orderData.order.status === 'PAID') && (
            <div className="mt-4 text-center">
              <p className="text-sm text-gray-500">
                {t(
                  'orderStatus.autoRefresh',
                  'Auto-refreshing every 10 seconds...'
                )}
              </p>
            </div>
          )}
        </main>
      </div>
    </PublicLayout>
  );
}
