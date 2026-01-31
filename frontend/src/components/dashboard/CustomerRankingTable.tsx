'use client';

import React from 'react';
import { CustomerRanking } from '@/types/analytics';
import { formatNumber, formatCurrency } from '@/utils/format';

export interface CustomerRankingTableProps {
  customers: CustomerRanking[];
  loading?: boolean;
}

export const CustomerRankingTable: React.FC<CustomerRankingTableProps> = ({
  customers,
  loading = false,
}) => {
  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <div className="h-6 bg-gray-200 rounded w-1/3 animate-pulse"></div>
        </div>
        <div className="divide-y divide-gray-200">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="p-4 animate-pulse flex items-center gap-4">
              <div className="w-10 h-10 bg-gray-200 rounded-full"></div>
              <div className="flex-1 space-y-2">
                <div className="h-4 bg-gray-200 rounded w-1/2"></div>
                <div className="h-3 bg-gray-200 rounded w-1/3"></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!customers || customers.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow p-6 text-center text-gray-500">
        No customers found for this period
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow overflow-hidden">
      <div className="p-6 border-b border-gray-200">
        <h3 className="text-lg font-semibold text-gray-900">
          Top {customers.length} Customers
        </h3>
      </div>

      <div className="divide-y divide-gray-200">
        {customers.map((customer, index) => (
          <div
            key={index}
            className="p-4 hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-4">
              {/* Rank Badge */}
              <div
                className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center font-bold text-sm ${
                  index === 0
                    ? 'bg-yellow-100 text-yellow-700'
                    : index === 1
                    ? 'bg-gray-200 text-gray-700'
                    : index === 2
                    ? 'bg-orange-100 text-orange-700'
                    : 'bg-gray-100 text-gray-600'
                }`}
              >
                #{index + 1}
              </div>

              {/* Customer Avatar */}
              <div className="flex-shrink-0 w-10 h-10 rounded-full bg-primary-100 flex items-center justify-center">
                <svg
                  className="w-6 h-6 text-primary-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                  />
                </svg>
              </div>

              {/* Customer Details (Masked PII) */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">
                  {customer.masked_name || 'Anonymous'}
                </p>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  {customer.masked_phone && (
                    <span className="font-mono">{customer.masked_phone}</span>
                  )}
                  {customer.masked_email && (
                    <span className="truncate">{customer.masked_email}</span>
                  )}
                </div>
              </div>

              {/* Metrics */}
              <div className="text-right">
                <p className="text-sm font-semibold text-gray-900">
                  {formatCurrency(customer.total_spent)}
                </p>
                <p className="text-xs text-gray-500">
                  {formatNumber(customer.order_count)} orders
                </p>
                <p className="text-xs text-gray-500">
                  Avg: {formatCurrency(customer.avg_order_value)}
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Privacy Notice */}
      <div className="p-4 bg-gray-50 border-t border-gray-200">
        <div className="flex items-start gap-2">
          <svg
            className="w-4 h-4 text-gray-400 mt-0.5 flex-shrink-0"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
            />
          </svg>
          <p className="text-xs text-gray-500">
            Customer personal information is masked for privacy. Names show first character only, 
            phone numbers show last 4 digits, and email shows first character + domain.
          </p>
        </div>
      </div>
    </div>
  );
};
