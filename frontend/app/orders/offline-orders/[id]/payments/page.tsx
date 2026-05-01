'use client';

import React, { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import RecordPayment from '@/components/orders/RecordPayment';
import PaymentSchedule from '@/components/orders/PaymentSchedule';
import offlineOrderService from '@/services/offlineOrders';
import { formatCurrency } from '@/utils/format';
import type {
  OfflineOrder,
  PaymentTerms,
  PaymentRecord,
  RecordPaymentRequest,
} from '@/types/offlineOrder';

/**
 * Record Payment Page (T070)
 * Page for recording a new payment for an offline order with installments
 */
export default function RecordPaymentPage() {
  const params = useParams();
  const router = useRouter();
  const orderId = params.id as string;

  const [order, setOrder] = useState<OfflineOrder | null>(null);
  const [paymentTerms, setPaymentTerms] = useState<PaymentTerms | null>(null);
  const [paymentRecords, setPaymentRecords] = useState<PaymentRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      if (!orderId) {
        setError('Order ID is required');
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(null);

        // Fetch order details
        const orderData = await offlineOrderService.getOfflineOrderById(orderId);
        setOrder(orderData);

        // Fetch payment history (includes payment terms if exists)
        const paymentHistory = await offlineOrderService.getPaymentHistory(orderId);
        setPaymentRecords(paymentHistory.payments || []);
        setPaymentTerms(paymentHistory.payment_terms || null);
      } catch (err: any) {
        console.error('Failed to fetch order data:', err);
        setError(err.response?.data?.message || 'Failed to load order details');

        if (err.response?.status === 404) {
          setTimeout(() => {
            router.push('/orders/offline-orders');
          }, 2000);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [orderId, router]);

  const handlePaymentRecorded = async (paymentRequest: RecordPaymentRequest) => {
    try {
      await offlineOrderService.recordPayment(orderId, paymentRequest);

      // Show success message and redirect to order detail
      setTimeout(() => {
        router.push(`/orders/offline-orders/${orderId}`);
      }, 1000);
    } catch (err: any) {
      console.error('Failed to record payment:', err);
      throw new Error(err.response?.data?.error || 'Failed to record payment');
    }
  };

  const handleCancel = () => {
    router.push(`/orders/offline-orders/${orderId}`);
  };

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="space-y-6 max-w-6xl mx-auto">
          {/* Header */}
          <div className="bg-white rounded-lg shadow p-6">
            <button
              onClick={handleCancel}
              className="text-blue-600 hover:text-blue-800 mb-2 flex items-center gap-1"
            >
              ← Back to Order
            </button>
            <h1 className="text-3xl font-bold text-gray-900">Record Payment</h1>
            {order && (
              <p className="text-gray-600 mt-2">
                Order {order.order_reference} - {formatCurrency(order.total_amount)}
              </p>
            )}
          </div>

          {/* Loading State */}
          {loading && (
            <div className="bg-white rounded-lg shadow p-12 text-center">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
              <p className="mt-4 text-gray-600">Loading order details...</p>
            </div>
          )}

          {/* Error State */}
          {error && !loading && (
            <div className="bg-red-50 border border-red-200 rounded-lg shadow p-6">
              <div className="flex items-center">
                <svg
                  className="h-6 w-6 text-red-600 mr-3"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <div>
                  <h3 className="text-lg font-semibold text-red-900">Error Loading Order</h3>
                  <p className="text-red-700 mt-1">{error}</p>
                  {error.includes('not found') && (
                    <p className="text-red-600 text-sm mt-2">Redirecting to orders list...</p>
                  )}
                </div>
              </div>
              <button
                onClick={handleCancel}
                className="mt-4 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors"
              >
                Back to Orders
              </button>
            </div>
          )}

          {/* Main Content */}
          {order && !loading && !error && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Left Column: Record Payment Form */}
              <div>
                <RecordPayment
                  orderId={orderId}
                  paymentTermsId={paymentTerms?.id}
                  remainingBalance={paymentTerms?.remaining_balance || order.total_amount}
                  onPaymentRecorded={handlePaymentRecorded}
                  onCancel={handleCancel}
                />
              </div>

              {/* Right Column: Payment Schedule & History */}
              <div className="space-y-6">
                {/* Payment Schedule (if installment order) */}
                {paymentTerms && (
                  <PaymentSchedule
                    schedule={paymentTerms.payment_schedule || []}
                    paymentRecords={paymentRecords}
                    totalAmount={paymentTerms.total_amount}
                    totalPaid={paymentTerms.total_paid}
                    remainingBalance={paymentTerms.remaining_balance}
                  />
                )}

                {/* Order Summary */}
                <div className="bg-white rounded-lg shadow p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Order Summary</h3>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-600">Order Reference:</span>
                      <span className="font-medium">{order.order_reference}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Customer:</span>
                      <span className="font-medium">{order.customer_name}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Phone:</span>
                      <span className="font-medium">{order.customer_phone}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-600">Delivery Type:</span>
                      <span className="font-medium capitalize">
                        {order.delivery_type.replace('_', ' ')}
                      </span>
                    </div>
                    <div className="flex justify-between pt-2 border-t">
                      <span className="text-gray-600">Subtotal:</span>
                      <span className="font-medium">{formatCurrency(order.subtotal_amount)}</span>
                    </div>
                    {order.delivery_fee > 0 && (
                      <div className="flex justify-between">
                        <span className="text-gray-600">Delivery Fee:</span>
                        <span className="font-medium">{formatCurrency(order.delivery_fee)}</span>
                      </div>
                    )}
                    <div className="flex justify-between pt-2 border-t font-bold">
                      <span>Total Amount:</span>
                      <span>{formatCurrency(order.total_amount)}</span>
                    </div>
                  </div>
                </div>

                {/* Payment History */}
                {paymentRecords.length > 0 && (
                  <div className="bg-white rounded-lg shadow p-6">
                    <h3 className="text-lg font-semibold text-gray-900 mb-4">Payment History</h3>
                    <div className="space-y-3">
                      {paymentRecords.map(payment => (
                        <div
                          key={payment.id}
                          className="p-3 bg-gray-50 rounded-md border border-gray-200"
                        >
                          <div className="flex justify-between items-start">
                            <div>
                              <p className="text-sm font-medium text-gray-900">
                                Payment #{payment.payment_number}
                              </p>
                              <p className="text-xs text-gray-500">
                                {new Date(payment.payment_date).toLocaleDateString('id-ID')}
                              </p>
                              <p className="text-xs text-gray-600 mt-1 capitalize">
                                {payment.payment_method.replace('_', ' ')}
                              </p>
                            </div>
                            <div className="text-right">
                              <p className="text-sm font-semibold text-green-600">
                                +{formatCurrency(payment.amount_paid)}
                              </p>
                              <p className="text-xs text-gray-500">
                                Balance: {formatCurrency(payment.remaining_balance_after)}
                              </p>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
