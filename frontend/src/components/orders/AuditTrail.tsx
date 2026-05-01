'use client';

import React from 'react';
import type { OfflineOrder } from '@/types/offlineOrder';

interface AuditTrailProps {
  order: OfflineOrder;
}

/**
 * AuditTrail Component (T084)
 * US3: Display audit trail for offline order changes
 * Shows creation and modification information
 */
export const AuditTrail: React.FC<AuditTrailProps> = ({ order }) => {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('id-ID', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Audit Trail</h3>

      <div className="space-y-4">
        {/* Creation Record */}
        <div className="flex items-start gap-3 pb-4 border-b border-gray-200">
          <div className="flex-shrink-0 w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
            <svg
              className="w-5 h-5 text-blue-600"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-900">Order Created</p>
            <p className="text-sm text-gray-600 mt-1">{formatDate(order.created_at)}</p>
            {order.recorded_by_user_id && (
              <p className="text-xs text-gray-500 mt-1">
                Created by: {order.recorded_by_user_id.substring(0, 8)}...
              </p>
            )}
            <div className="mt-2 space-y-1">
              <p className="text-xs text-gray-600">
                <span className="font-medium">Order Reference:</span> {order.order_reference}
              </p>
              <p className="text-xs text-gray-600">
                <span className="font-medium">Status:</span> {order.status}
              </p>
              <p className="text-xs text-gray-600">
                <span className="font-medium">Customer:</span> {order.customer_name}
              </p>
            </div>
          </div>
        </div>

        {/* Modification Record */}
        {order.last_modified_at && (
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 w-10 h-10 bg-yellow-100 rounded-full flex items-center justify-center">
              <svg
                className="w-5 h-5 text-yellow-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900">Order Modified</p>
              <p className="text-sm text-gray-600 mt-1">{formatDate(order.last_modified_at)}</p>
              {order.last_modified_by_user_id && (
                <p className="text-xs text-gray-500 mt-1">
                  Modified by: {order.last_modified_by_user_id.substring(0, 8)}...
                </p>
              )}
            </div>
          </div>
        )}

        {/* Status Change Records */}
        {order.paid_at && (
          <div className="flex items-start gap-3 pt-4 border-t border-gray-200">
            <div className="flex-shrink-0 w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
              <svg
                className="w-5 h-5 text-green-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900">Payment Completed</p>
              <p className="text-sm text-gray-600 mt-1">{formatDate(order.paid_at)}</p>
              <p className="text-xs text-gray-600 mt-1">Order status changed to PAID</p>
            </div>
          </div>
        )}

        {order.completed_at && (
          <div className="flex items-start gap-3 pt-4 border-t border-gray-200">
            <div className="flex-shrink-0 w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
              <svg
                className="w-5 h-5 text-blue-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900">Order Completed</p>
              <p className="text-sm text-gray-600 mt-1">{formatDate(order.completed_at)}</p>
              <p className="text-xs text-gray-600 mt-1">Order status changed to COMPLETE</p>
            </div>
          </div>
        )}

        {order.cancelled_at && (
          <div className="flex items-start gap-3 pt-4 border-t border-gray-200">
            <div className="flex-shrink-0 w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
              <svg
                className="w-5 h-5 text-red-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900">Order Cancelled</p>
              <p className="text-sm text-gray-600 mt-1">{formatDate(order.cancelled_at)}</p>
              <p className="text-xs text-gray-600 mt-1">Order status changed to CANCELLED</p>
            </div>
          </div>
        )}
      </div>

      {/* Future Enhancement Note */}
      <div className="mt-6 pt-4 border-t border-gray-200">
        <p className="text-xs text-gray-500 italic">
          Detailed change history is recorded in the audit trail system. Complete audit logs
          including field-level changes can be viewed in the admin panel.
        </p>
      </div>
    </div>
  );
};

export default AuditTrail;
