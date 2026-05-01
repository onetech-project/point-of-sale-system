'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { formatCurrency } from '../../utils/format';
import {
  OfflineOrder,
  OfflineOrderItem,
  PaymentTerms,
  PaymentRecord,
  OrderStatus,
} from '../../types/offlineOrder';
import { AuditTrail } from './AuditTrail';
import { DeleteOrderModal } from './DeleteOrderModal';
import offlineOrderService from '../../services/offlineOrders';

interface OfflineOrderDetailProps {
  order: OfflineOrder;
  items?: OfflineOrderItem[];
  paymentTerms?: PaymentTerms;
  paymentRecords?: PaymentRecord[];
  onRefresh?: () => void;
}

export const OfflineOrderDetail: React.FC<OfflineOrderDetailProps> = ({
  order,
  items = [],
  paymentTerms,
  paymentRecords = [],
  onRefresh,
}) => {
  const router = useRouter();
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const getStatusColor = (status: OrderStatus): string => {
    switch (status) {
      case 'PENDING':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      case 'PAID':
        return 'bg-green-100 text-green-800 border-green-300';
      case 'COMPLETE':
        return 'bg-blue-100 text-blue-800 border-blue-300';
      case 'CANCELLED':
        return 'bg-red-100 text-red-800 border-red-300';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-300';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('id-ID', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const formatDateShort = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('id-ID', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const handleBack = () => {
    router.push('/orders/offline-orders');
  };

  const handleRecordPayment = () => {
    router.push(`/orders/offline-orders/${order.id}/payments`);
  };

  const handleEdit = () => {
    router.push(`/orders/offline-orders/${order.id}/edit`);
  };

  const handleDelete = () => {
    setShowDeleteModal(true);
    setDeleteError(null);
  };

  const handleConfirmDelete = async (reason: string) => {
    setIsDeleting(true);
    setDeleteError(null);

    try {
      await offlineOrderService.deleteOfflineOrder(order.id, reason);

      // Success - redirect to orders list
      setShowDeleteModal(false);
      router.push('/orders/offline-orders?deleted=true');
    } catch (error: any) {
      console.error('Failed to delete order:', error);

      // Extract error message
      const errorMessage =
        error?.response?.data?.error || error?.message || 'Failed to delete order';
      setDeleteError(errorMessage);
      setIsDeleting(false);
    }
  };

  const handleCancelDelete = () => {
    setShowDeleteModal(false);
    setDeleteError(null);
  };

  // Check if user can delete (owner/manager only)
  // Note: In a real app, get this from user context/auth
  const canDelete = true; // TODO: Replace with actual role check from auth context

  return (
    <div className="space-y-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex justify-between items-start">
        <div>
          <button
            onClick={handleBack}
            className="text-blue-600 hover:text-blue-800 mb-2 flex items-center gap-1"
          >
            ← Back to Orders
          </button>
          <h1 className="text-3xl font-bold text-gray-900">Order {order.order_reference}</h1>
          <p className="text-gray-500 mt-1">Created {formatDate(order.created_at)}</p>
        </div>
        <div className="flex items-center gap-3">
          <span
            className={`px-4 py-2 text-sm font-semibold rounded-lg border ${getStatusColor(
              order.status
            )}`}
          >
            {order.status}
          </span>
          {order.status === 'PENDING' && (
            <>
              <button
                onClick={handleEdit}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium"
              >
                Edit Order
              </button>
              <button
                onClick={handleRecordPayment}
                className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 font-medium"
              >
                Record Payment
              </button>
              {canDelete && (
                <button
                  onClick={handleDelete}
                  className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 font-medium flex items-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                  Delete
                </button>
              )}
            </>
          )}
          {order.status === 'CANCELLED' && canDelete && (
            <button
              onClick={handleDelete}
              className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 font-medium flex items-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
              Delete
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Order Items */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Order Items</h2>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead>
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Product
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                      Qty
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                      Unit Price
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                      Total
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {items.map(item => (
                    <tr key={item.id}>
                      <td className="px-4 py-3 text-sm text-gray-900">{item.product_name}</td>
                      <td className="px-4 py-3 text-sm text-gray-900 text-right">
                        {item.quantity}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900 text-right">
                        {formatCurrency(item.unit_price)}
                      </td>
                      <td className="px-4 py-3 text-sm font-medium text-gray-900 text-right">
                        {formatCurrency(item.total_price)}
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot className="bg-gray-50">
                  <tr>
                    <td
                      colSpan={3}
                      className="px-4 py-3 text-sm font-medium text-gray-900 text-right"
                    >
                      Subtotal:
                    </td>
                    <td className="px-4 py-3 text-sm font-medium text-gray-900 text-right">
                      {formatCurrency(order.subtotal_amount)}
                    </td>
                  </tr>
                  {order.delivery_fee > 0 && (
                    <tr>
                      <td
                        colSpan={3}
                        className="px-4 py-3 text-sm font-medium text-gray-900 text-right"
                      >
                        Delivery Fee:
                      </td>
                      <td className="px-4 py-3 text-sm font-medium text-gray-900 text-right">
                        {formatCurrency(order.delivery_fee)}
                      </td>
                    </tr>
                  )}
                  <tr>
                    <td
                      colSpan={3}
                      className="px-4 py-3 text-lg font-bold text-gray-900 text-right"
                    >
                      Total:
                    </td>
                    <td className="px-4 py-3 text-lg font-bold text-gray-900 text-right">
                      {formatCurrency(order.total_amount)}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>

          {/* Payment Information */}
          {paymentTerms && (
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Payment Terms</h2>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-gray-500">Total Amount</p>
                    <p className="text-lg font-semibold">
                      {formatCurrency(paymentTerms.total_amount)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Down Payment</p>
                    <p className="text-lg font-semibold">
                      {paymentTerms.down_payment_amount
                        ? formatCurrency(paymentTerms.down_payment_amount)
                        : 'N/A'}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Total Paid</p>
                    <p className="text-lg font-semibold text-green-600">
                      {formatCurrency(paymentTerms.total_paid)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-500">Remaining Balance</p>
                    <p className="text-lg font-semibold text-orange-600">
                      {formatCurrency(paymentTerms.remaining_balance)}
                    </p>
                  </div>
                </div>

                {paymentTerms.payment_schedule.length > 0 && (
                  <div className="mt-4">
                    <h3 className="text-sm font-medium text-gray-700 mb-2">Payment Schedule</h3>
                    <div className="space-y-2">
                      {paymentTerms.payment_schedule.map(schedule => (
                        <div
                          key={schedule.installment_number}
                          className="flex justify-between items-center p-3 bg-gray-50 rounded-md"
                        >
                          <div>
                            <p className="text-sm font-medium text-gray-900">
                              Installment #{schedule.installment_number}
                            </p>
                            <p className="text-xs text-gray-500">
                              Due: {formatDateShort(schedule.due_date)}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="text-sm font-semibold text-gray-900">
                              {formatCurrency(schedule.amount)}
                            </p>
                            <span
                              className={`text-xs px-2 py-1 rounded-full ${
                                schedule.status === 'paid'
                                  ? 'bg-green-100 text-green-800'
                                  : schedule.status === 'overdue'
                                    ? 'bg-red-100 text-red-800'
                                    : 'bg-yellow-100 text-yellow-800'
                              }`}
                            >
                              {schedule.status}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Payment History */}
          {paymentRecords.length > 0 && (
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Payment History</h2>
              <div className="space-y-3">
                {paymentRecords.map(payment => (
                  <div key={payment.id} className="border border-gray-200 rounded-lg p-4">
                    <div className="flex justify-between items-start">
                      <div>
                        <p className="font-medium text-gray-900">
                          Payment #{payment.payment_number}
                        </p>
                        <p className="text-sm text-gray-500">{formatDate(payment.payment_date)}</p>
                        <p className="text-xs text-gray-500 mt-1 capitalize">
                          Method: {payment.payment_method.replace('_', ' ')}
                        </p>
                        {payment.notes && (
                          <p className="text-sm text-gray-600 mt-2">{payment.notes}</p>
                        )}
                        {payment.receipt_number && (
                          <p className="text-xs text-gray-500 mt-1">
                            Receipt: {payment.receipt_number}
                          </p>
                        )}
                      </div>
                      <div className="text-right">
                        <p className="text-lg font-semibold text-green-600">
                          {formatCurrency(payment.amount_paid)}
                        </p>
                        <p className="text-xs text-gray-500 mt-1">
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

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Customer Information */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Customer Information</h2>
            <div className="space-y-3">
              <div>
                <p className="text-sm text-gray-500">Name</p>
                <p className="text-sm font-medium text-gray-900">{order.customer_name}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Phone</p>
                <p className="text-sm font-medium text-gray-900">{order.customer_phone}</p>
              </div>
              {order.customer_email && (
                <div>
                  <p className="text-sm text-gray-500">Email</p>
                  <p className="text-sm font-medium text-gray-900">{order.customer_email}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-gray-500">Data Consent</p>
                <p className="text-sm font-medium text-gray-900">
                  {order.data_consent_given ? '✓ Given' : '✗ Not Given'}
                </p>
                {order.consent_method && (
                  <p className="text-xs text-gray-500 capitalize">Method: {order.consent_method}</p>
                )}
              </div>
            </div>
          </div>

          {/* Delivery Information */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Delivery Information</h2>
            <div className="space-y-3">
              <div>
                <p className="text-sm text-gray-500">Type</p>
                <p className="text-sm font-medium text-gray-900 capitalize">
                  {order.delivery_type.replace('_', ' ')}
                </p>
              </div>
              {order.table_number && (
                <div>
                  <p className="text-sm text-gray-500">Table Number</p>
                  <p className="text-sm font-medium text-gray-900">{order.table_number}</p>
                </div>
              )}
              {order.notes && (
                <div>
                  <p className="text-sm text-gray-500">Notes</p>
                  <p className="text-sm text-gray-900">{order.notes}</p>
                </div>
              )}
            </div>
          </div>

          {/* Order Metadata */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Order Details</h2>
            <div className="space-y-3 text-sm">
              <div>
                <p className="text-gray-500">Order ID</p>
                <p className="font-mono text-xs text-gray-700">{order.id}</p>
              </div>
              <div>
                <p className="text-gray-500">Created At</p>
                <p className="text-gray-900">{formatDate(order.created_at)}</p>
              </div>
              {order.paid_at && (
                <div>
                  <p className="text-gray-500">Paid At</p>
                  <p className="text-gray-900">{formatDate(order.paid_at)}</p>
                </div>
              )}
              {order.completed_at && (
                <div>
                  <p className="text-gray-500">Completed At</p>
                  <p className="text-gray-900">{formatDate(order.completed_at)}</p>
                </div>
              )}
              {order.cancelled_at && (
                <div>
                  <p className="text-gray-500">Cancelled At</p>
                  <p className="text-gray-900">{formatDate(order.cancelled_at)}</p>
                </div>
              )}
              {order.last_modified_at && (
                <div>
                  <p className="text-gray-500">Last Modified</p>
                  <p className="text-gray-900">{formatDate(order.last_modified_at)}</p>
                </div>
              )}
            </div>
          </div>

          {/* Audit Trail (T084-T085) */}
          <AuditTrail order={order} />
        </div>
      </div>

      {/* Delete Order Modal (T096-T097) */}
      <DeleteOrderModal
        isOpen={showDeleteModal}
        orderReference={order.order_reference}
        onConfirm={handleConfirmDelete}
        onCancel={handleCancelDelete}
        isDeleting={isDeleting}
      />

      {/* Delete Error Display */}
      {deleteError && (
        <div className="fixed bottom-4 right-4 bg-red-50 border border-red-200 rounded-lg p-4 shadow-lg max-w-md">
          <div className="flex items-start gap-3">
            <svg
              className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5"
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
            <div className="flex-1">
              <p className="text-sm font-medium text-red-800">Failed to delete order</p>
              <p className="text-sm text-red-700 mt-1">{deleteError}</p>
            </div>
            <button
              onClick={() => setDeleteError(null)}
              className="text-red-600 hover:text-red-800"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
