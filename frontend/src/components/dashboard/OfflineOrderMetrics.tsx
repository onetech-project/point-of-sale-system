'use client';

import React from 'react';
import { formatCurrency, formatNumber } from '@/utils/format';

interface OfflineOrderMetricsProps {
  offlineOrderCount: number;
  offlineRevenue: number;
  offlinePercentage: number;
  onlineOrderCount: number;
  onlineRevenue: number;
  installmentCount: number;
  installmentRevenue: number;
  pendingInstallments: number;
  loading?: boolean;
}

/**
 * OfflineOrderMetrics Component (T104-T106)
 * US5: Display offline order metrics in dashboard
 * Shows offline vs online order breakdown, installment stats
 */
export const OfflineOrderMetrics: React.FC<OfflineOrderMetricsProps> = ({
  offlineOrderCount,
  offlineRevenue,
  offlinePercentage,
  onlineOrderCount,
  onlineRevenue,
  installmentCount,
  installmentRevenue,
  pendingInstallments,
  loading = false,
}) => {
  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow p-6 animate-pulse">
        <div className="h-6 bg-gray-200 rounded w-1/3 mb-4" />
        <div className="space-y-3">
          <div className="h-20 bg-gray-200 rounded" />
          <div className="h-20 bg-gray-200 rounded" />
          <div className="h-20 bg-gray-200 rounded" />
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Offline Order Insights</h3>
        <div className="flex items-center gap-2 text-sm text-gray-600">
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
          <span>Offline Orders</span>
        </div>
      </div>

      <div className="space-y-4">
        {/* Offline vs Online Comparison */}
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-blue-50 rounded-lg p-4 border border-blue-100">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-blue-900">Offline</span>
              <span className="text-xs font-semibold px-2 py-1 bg-blue-200 text-blue-900 rounded">
                {offlinePercentage.toFixed(1)}%
              </span>
            </div>
            <p className="text-2xl font-bold text-blue-900">{formatNumber(offlineOrderCount, 0)}</p>
            <p className="text-sm text-blue-700 mt-1">
              {formatCurrency(offlineRevenue, true)} revenue
            </p>
          </div>

          <div className="bg-green-50 rounded-lg p-4 border border-green-100">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-green-900">Online</span>
              <span className="text-xs font-semibold px-2 py-1 bg-green-200 text-green-900 rounded">
                {(100 - offlinePercentage).toFixed(1)}%
              </span>
            </div>
            <p className="text-2xl font-bold text-green-900">{formatNumber(onlineOrderCount, 0)}</p>
            <p className="text-sm text-green-700 mt-1">
              {formatCurrency(onlineRevenue, true)} revenue
            </p>
          </div>
        </div>

        {/* Installment Orders */}
        <div className="bg-purple-50 rounded-lg p-4 border border-purple-100">
          <div className="flex items-center gap-2 mb-2">
            <svg
              className="w-5 h-5 text-purple-600"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"
              />
            </svg>
            <span className="text-sm font-medium text-purple-900">Installment Orders</span>
          </div>
          <div className="grid grid-cols-2 gap-4 mt-3">
            <div>
              <p className="text-xs text-purple-700">Active Installments</p>
              <p className="text-xl font-bold text-purple-900">
                {formatNumber(installmentCount, 0)}
              </p>
            </div>
            <div>
              <p className="text-xs text-purple-700">Total Value</p>
              <p className="text-xl font-bold text-purple-900">
                {formatCurrency(installmentRevenue, true)}
              </p>
            </div>
          </div>
        </div>

        {/* Pending Installments Alert */}
        {pendingInstallments > 0 && (
          <div className="bg-orange-50 rounded-lg p-4 border border-orange-200">
            <div className="flex items-start gap-3">
              <svg
                className="w-5 h-5 text-orange-600 flex-shrink-0 mt-0.5"
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
                <p className="text-sm font-medium text-orange-900">Pending Payments</p>
                <p className="text-lg font-bold text-orange-900 mt-1">
                  {formatCurrency(pendingInstallments, true)}
                </p>
                <p className="text-xs text-orange-700 mt-1">
                  Outstanding installment payments requiring follow-up
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Quick Stats Summary */}
        <div className="pt-4 border-t border-gray-200">
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <p className="text-xs text-gray-600">Offline AOV</p>
              <p className="text-lg font-semibold text-gray-900">
                {offlineOrderCount > 0
                  ? formatCurrency(offlineRevenue / offlineOrderCount, true)
                  : '—'}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-600">Online AOV</p>
              <p className="text-lg font-semibold text-gray-900">
                {onlineOrderCount > 0
                  ? formatCurrency(onlineRevenue / onlineOrderCount, true)
                  : '—'}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-600">Installment Rate</p>
              <p className="text-lg font-semibold text-gray-900">
                {offlineOrderCount > 0
                  ? `${((installmentCount / offlineOrderCount) * 100).toFixed(1)}%`
                  : '—'}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default OfflineOrderMetrics;
