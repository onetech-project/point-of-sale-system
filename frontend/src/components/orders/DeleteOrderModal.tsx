'use client';

import React, { useState } from 'react';

interface DeleteOrderModalProps {
  isOpen: boolean;
  orderReference: string;
  onConfirm: (reason: string) => void;
  onCancel: () => void;
  isDeleting: boolean;
}

/**
 * DeleteOrderModal Component (T097)
 * US4: Role-Based Deletion - Confirmation modal with reason input
 * Only accessible to owner and manager roles
 */
export const DeleteOrderModal: React.FC<DeleteOrderModalProps> = ({
  isOpen,
  orderReference,
  onConfirm,
  onCancel,
  isDeleting,
}) => {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Validate reason length
    if (reason.trim().length < 5) {
      setError('Deletion reason must be at least 5 characters');
      return;
    }

    if (reason.trim().length > 500) {
      setError('Deletion reason must not exceed 500 characters');
      return;
    }

    setError('');
    onConfirm(reason.trim());
  };

  const handleCancel = () => {
    setReason('');
    setError('');
    onCancel();
  };

  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black bg-opacity-50 transition-opacity"
        onClick={handleCancel}
      />

      {/* Modal */}
      <div className="flex min-h-full items-center justify-center p-4">
        <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full">
          {/* Header */}
          <div className="px-6 py-4 border-b border-gray-200">
            <div className="flex items-center gap-3">
              <div className="flex-shrink-0 w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
                <svg
                  className="w-6 h-6 text-red-600"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold text-gray-900">Delete Order</h3>
                <p className="text-sm text-gray-600">
                  Order: <span className="font-mono">{orderReference}</span>
                </p>
              </div>
            </div>
          </div>

          {/* Body */}
          <form onSubmit={handleSubmit}>
            <div className="px-6 py-4">
              <div className="mb-4">
                <p className="text-sm text-gray-700 mb-4">
                  This action cannot be undone. The order will be permanently marked as deleted.
                </p>

                <label
                  htmlFor="deletion-reason"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  Reason for deletion *
                </label>
                <textarea
                  id="deletion-reason"
                  value={reason}
                  onChange={e => {
                    setReason(e.target.value);
                    setError('');
                  }}
                  placeholder="Please provide a detailed reason for deleting this order..."
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-red-500 min-h-[100px] resize-y"
                  disabled={isDeleting}
                  maxLength={500}
                  required
                />
                <div className="mt-1 flex justify-between items-center">
                  <p className="text-xs text-gray-500">Minimum 5 characters required</p>
                  <p className="text-xs text-gray-500">{reason.length}/500</p>
                </div>

                {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
              </div>

              <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3">
                <div className="flex gap-2">
                  <svg
                    className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                    />
                  </svg>
                  <div className="flex-1">
                    <p className="text-sm font-medium text-yellow-800">Warning</p>
                    <p className="text-xs text-yellow-700 mt-1">
                      Only PENDING or CANCELLED orders can be deleted. This action is logged in the
                      audit trail.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            {/* Footer */}
            <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end gap-3 rounded-b-lg">
              <button
                type="button"
                onClick={handleCancel}
                disabled={isDeleting}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isDeleting || reason.trim().length < 5}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 border border-transparent rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isDeleting ? (
                  <>
                    <svg
                      className="animate-spin h-4 w-4 text-white"
                      xmlns="http://www.w3.org/2000/svg"
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
                    Deleting...
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                      />
                    </svg>
                    Delete Order
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};

export default DeleteOrderModal;
