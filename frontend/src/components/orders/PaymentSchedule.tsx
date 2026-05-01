'use client';

import React from 'react';
import { formatCurrency } from '@/utils/format';
import type { PaymentSchedule as PaymentScheduleType, PaymentRecord } from '@/types/offlineOrder';

interface PaymentScheduleProps {
  schedule: PaymentScheduleType[];
  paymentRecords?: PaymentRecord[];
  totalAmount: number;
  totalPaid: number;
  remainingBalance: number;
}

/**
 * PaymentSchedule Component (T066)
 * Displays installment payment schedule with status tracking
 */
const PaymentSchedule: React.FC<PaymentScheduleProps> = ({
  schedule,
  paymentRecords = [],
  totalAmount,
  totalPaid,
  remainingBalance,
}) => {
  // Calculate which installments have been paid
  const paidInstallments = new Set<number>();
  paymentRecords.forEach(record => {
    if (record.payment_number > 0) {
      // payment_number 0 is down payment, 1+ are installments
      paidInstallments.add(record.payment_number);
    }
  });

  // Format date
  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleDateString('id-ID', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    });
  };

  // Determine installment status
  const getInstallmentStatus = (
    installment: PaymentScheduleType
  ): 'paid' | 'pending' | 'overdue' => {
    if (paidInstallments.has(installment.installment_number)) {
      return 'paid';
    }
    const dueDate = new Date(installment.due_date);
    const now = new Date();
    if (dueDate < now) {
      return 'overdue';
    }
    return 'pending';
  };

  // Status badge
  const StatusBadge: React.FC<{ status: 'paid' | 'pending' | 'overdue' }> = ({ status }) => {
    const colors = {
      paid: 'bg-green-100 text-green-800',
      pending: 'bg-yellow-100 text-yellow-800',
      overdue: 'bg-red-100 text-red-800',
    };

    const labels = {
      paid: 'Paid',
      pending: 'Pending',
      overdue: 'Overdue',
    };

    return (
      <span className={`px-2 py-1 rounded-full text-xs font-semibold ${colors[status]}`}>
        {labels[status]}
      </span>
    );
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Payment Schedule</h3>

      {/* Payment Summary */}
      <div className="grid grid-cols-3 gap-4 mb-6 p-4 bg-gray-50 rounded-lg">
        <div>
          <p className="text-sm text-gray-600">Total Amount</p>
          <p className="text-lg font-bold text-gray-900">{formatCurrency(totalAmount)}</p>
        </div>
        <div>
          <p className="text-sm text-gray-600">Total Paid</p>
          <p className="text-lg font-bold text-green-600">{formatCurrency(totalPaid)}</p>
        </div>
        <div>
          <p className="text-sm text-gray-600">Remaining Balance</p>
          <p className="text-lg font-bold text-red-600">{formatCurrency(remainingBalance)}</p>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="mb-6">
        <div className="flex justify-between text-sm text-gray-600 mb-1">
          <span>Payment Progress</span>
          <span>{Math.round((totalPaid / totalAmount) * 100)}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-2">
          <div
            className="bg-green-600 h-2 rounded-full transition-all duration-300"
            style={{ width: `${Math.min((totalPaid / totalAmount) * 100, 100)}%` }}
          />
        </div>
      </div>

      {/* Installment List */}
      <div className="space-y-3">
        <h4 className="text-md font-semibold text-gray-800 mb-3">Installments</h4>
        {schedule.map(installment => {
          const status = getInstallmentStatus(installment);
          return (
            <div
              key={installment.installment_number}
              className="flex items-center justify-between p-3 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <div className="flex items-center space-x-4">
                <div className="flex-shrink-0">
                  <div className="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center">
                    <span className="text-blue-800 font-semibold">
                      #{installment.installment_number}
                    </span>
                  </div>
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">
                    Installment {installment.installment_number}
                  </p>
                  <p className="text-xs text-gray-500">Due: {formatDate(installment.due_date)}</p>
                </div>
              </div>
              <div className="flex items-center space-x-4">
                <p className="text-md font-semibold text-gray-900">
                  {formatCurrency(installment.amount)}
                </p>
                <StatusBadge status={status} />
              </div>
            </div>
          );
        })}
      </div>

      {/* Summary Footer */}
      {remainingBalance === 0 ? (
        <div className="mt-6 p-4 bg-green-50 border border-green-200 rounded-lg">
          <div className="flex items-center">
            <svg
              className="h-5 w-5 text-green-600 mr-2"
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
            <p className="text-sm font-semibold text-green-800">
              Payment Complete - All installments paid
            </p>
          </div>
        </div>
      ) : (
        <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <p className="text-sm text-blue-800">
            <span className="font-semibold">Next Action:</span> Record payment when customer pays
            the next installment
          </p>
        </div>
      )}
    </div>
  );
};

export default PaymentSchedule;
