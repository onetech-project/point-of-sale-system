'use client';

import React, { useState } from 'react';
import { formatCurrency } from '@/utils/format';
import type { PaymentMethod, RecordPaymentRequest } from '@/types/offlineOrder';

interface RecordPaymentProps {
  orderId: string;
  paymentTermsId?: string;
  remainingBalance: number;
  onPaymentRecorded: (payment: RecordPaymentRequest) => Promise<void>;
  onCancel?: () => void;
}

/**
 * RecordPayment Component (T067)
 * Form for recording a new payment for an offline order with installments
 */
const RecordPayment: React.FC<RecordPaymentProps> = ({
  orderId,
  paymentTermsId,
  remainingBalance,
  onPaymentRecorded,
  onCancel,
}) => {
  const [amountPaid, setAmountPaid] = useState<string>('');
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('cash');
  const [notes, setNotes] = useState<string>('');
  const [receiptNumber, setReceiptNumber] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    const amount = parseInt(amountPaid);
    if (isNaN(amount) || amount <= 0) {
      setError('Please enter a valid payment amount');
      return;
    }

    if (amount > remainingBalance) {
      setError(
        `Payment amount cannot exceed remaining balance (${formatCurrency(remainingBalance)})`
      );
      return;
    }

    if (!paymentMethod) {
      setError('Please select a payment method');
      return;
    }

    try {
      setLoading(true);

      const paymentRequest: RecordPaymentRequest = {
        order_id: orderId,
        payment_terms_id: paymentTermsId,
        payment_number: 0, // Will be calculated by backend
        amount_paid: amount,
        payment_method: paymentMethod,
        notes: notes.trim() || undefined,
        receipt_number: receiptNumber.trim() || undefined,
      };

      await onPaymentRecorded(paymentRequest);

      // Reset form on success
      setAmountPaid('');
      setPaymentMethod('cash');
      setNotes('');
      setReceiptNumber('');
    } catch (err: any) {
      console.error('Failed to record payment:', err);
      setError(err.message || 'Failed to record payment. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  // Handle quick amount buttons
  const handleQuickAmount = (amount: number) => {
    setAmountPaid(amount.toString());
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Record Payment</h3>

      {/* Remaining Balance Display */}
      <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
        <p className="text-sm text-blue-800 mb-1">Remaining Balance</p>
        <p className="text-2xl font-bold text-blue-900">{formatCurrency(remainingBalance)}</p>
      </div>

      {/* Error Message */}
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-sm text-red-800">{error}</p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Amount Paid */}
        <div>
          <label htmlFor="amountPaid" className="block text-sm font-medium text-gray-700 mb-1">
            Payment Amount <span className="text-red-500">*</span>
          </label>
          <input
            type="number"
            id="amountPaid"
            value={amountPaid}
            onChange={e => setAmountPaid(e.target.value)}
            min="1"
            max={remainingBalance}
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Enter amount"
          />
          <div className="mt-2 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => handleQuickAmount(remainingBalance)}
              className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
            >
              Full Amount
            </button>
            {remainingBalance >= 100000 && (
              <button
                type="button"
                onClick={() => handleQuickAmount(100000)}
                className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
              >
                Rp 100,000
              </button>
            )}
            {remainingBalance >= 50000 && (
              <button
                type="button"
                onClick={() => handleQuickAmount(50000)}
                className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
              >
                Rp 50,000
              </button>
            )}
          </div>
        </div>

        {/* Payment Method */}
        <div>
          <label htmlFor="paymentMethod" className="block text-sm font-medium text-gray-700 mb-1">
            Payment Method <span className="text-red-500">*</span>
          </label>
          <select
            id="paymentMethod"
            value={paymentMethod}
            onChange={e => setPaymentMethod(e.target.value as PaymentMethod)}
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="cash">Cash</option>
            <option value="card">Card (Debit/Credit)</option>
            <option value="bank_transfer">Bank Transfer</option>
            <option value="check">Check</option>
            <option value="other">Other</option>
          </select>
        </div>

        {/* Receipt Number */}
        <div>
          <label htmlFor="receiptNumber" className="block text-sm font-medium text-gray-700 mb-1">
            Receipt Number (Optional)
          </label>
          <input
            type="text"
            id="receiptNumber"
            value={receiptNumber}
            onChange={e => setReceiptNumber(e.target.value)}
            maxLength={100}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="e.g., RCP-2026-001"
          />
        </div>

        {/* Notes */}
        <div>
          <label htmlFor="notes" className="block text-sm font-medium text-gray-700 mb-1">
            Notes (Optional)
          </label>
          <textarea
            id="notes"
            value={notes}
            onChange={e => setNotes(e.target.value)}
            rows={3}
            maxLength={1000}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Any additional notes about this payment..."
          />
          <p className="text-xs text-gray-500 mt-1">{notes.length}/1000 characters</p>
        </div>

        {/* Action Buttons */}
        <div className="flex space-x-3 pt-4">
          <button
            type="submit"
            disabled={loading}
            className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? (
              <span className="flex items-center justify-center">
                <svg
                  className="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
                Recording...
              </span>
            ) : (
              'Record Payment'
            )}
          </button>
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              disabled={loading}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-500 disabled:bg-gray-100 disabled:cursor-not-allowed transition-colors"
            >
              Cancel
            </button>
          )}
        </div>
      </form>
    </div>
  );
};

export default RecordPayment;
