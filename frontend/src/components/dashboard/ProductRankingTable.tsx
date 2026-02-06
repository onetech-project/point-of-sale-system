'use client';

import React from 'react';
import { ProductRanking } from '@/types/analytics';
import { formatNumber, formatCurrency } from '@/utils/format';

export interface ProductRankingTableProps {
  products: ProductRanking[];
  loading?: boolean;
  type?: 'top' | 'bottom'; // Display style
  metric?: 'revenue' | 'quantity'; // Which metric is being ranked
}

export const ProductRankingTable: React.FC<ProductRankingTableProps> = ({
  products,
  loading = false,
  type = 'top',
  metric = 'revenue',
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
              <div className="w-12 h-12 bg-gray-200 rounded"></div>
              <div className="flex-1 space-y-2">
                <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (!products || products.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow p-6 text-center text-gray-500">
        No products found for this period
      </div>
    );
  }

  const title = type === 'top' 
    ? `Top ${products.length} Products by ${metric === 'revenue' ? 'Revenue' : 'Quantity'}`
    : `Bottom ${products.length} Products by ${metric === 'revenue' ? 'Revenue' : 'Quantity'}`;

  return (
    <div className="bg-white rounded-lg shadow overflow-hidden">
      <div className="p-6 border-b border-gray-200">
        <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
      </div>

      <div className="divide-y divide-gray-200">
        {products.map((product, index) => (
          <div
            key={product.product_id}
            className="p-4 hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-4">
              {/* Rank Badge */}
              <div
                className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center font-bold text-sm ${
                  index === 0 && type === 'top'
                    ? 'bg-yellow-100 text-yellow-700'
                    : index === 1 && type === 'top'
                    ? 'bg-gray-200 text-gray-700'
                    : index === 2 && type === 'top'
                    ? 'bg-orange-100 text-orange-700'
                    : 'bg-gray-100 text-gray-600'
                }`}
              >
                #{index + 1}
              </div>

              {/* Product Image */}
              {product.image_url ? (
                <img
                  src={product.image_url}
                  alt={product.name}
                  className="w-12 h-12 rounded-lg object-cover flex-shrink-0"
                />
              ) : (
                <div className="w-12 h-12 rounded-lg bg-gray-200 flex items-center justify-center flex-shrink-0">
                  <svg
                    className="w-6 h-6 text-gray-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                    />
                  </svg>
                </div>
              )}

              {/* Product Details */}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">
                  {product.name}
                </p>
                <p className="text-xs text-gray-500">{product.category_name}</p>
              </div>

              {/* Metrics */}
              <div className="text-right">
                <p className="text-sm font-semibold text-gray-900">
                  {formatCurrency(product.revenue)}
                </p>
                <p className="text-xs text-gray-500">
                  {formatNumber(product.quantity_sold)} sold
                </p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
