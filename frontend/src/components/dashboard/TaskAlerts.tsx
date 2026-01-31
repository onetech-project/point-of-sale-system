'use client';

import React from 'react';
import { DelayedOrder, RestockAlert } from '@/types/analytics';
import { formatCurrency, formatNumber } from '@/utils/format';

export interface TaskAlertsProps {
  delayedOrders: DelayedOrder[];
  restockAlerts: RestockAlert[];
  loading?: boolean;
  onNavigateToOrder?: (orderId: number) => void;
  onNavigateToProduct?: (productId: number) => void;
}

export const TaskAlerts: React.FC<TaskAlertsProps> = ({
  delayedOrders,
  restockAlerts,
  loading = false,
  onNavigateToOrder,
  onNavigateToProduct,
}) => {
  if (loading) {
    return (
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Delayed Orders Skeleton */}
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <div className="p-6 border-b border-gray-200 animate-pulse">
            <div className="h-6 bg-gray-200 rounded w-1/3"></div>
          </div>
          <div className="divide-y divide-gray-200">
            {[1, 2, 3].map((i) => (
              <div key={i} className="p-4 animate-pulse flex items-center gap-4">
                <div className="w-10 h-10 bg-gray-200 rounded-full"></div>
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 rounded w-3/4"></div>
                  <div className="h-3 bg-gray-200 rounded w-1/2"></div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Restock Alerts Skeleton */}
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <div className="p-6 border-b border-gray-200 animate-pulse">
            <div className="h-6 bg-gray-200 rounded w-1/3"></div>
          </div>
          <div className="divide-y divide-gray-200">
            {[1, 2, 3].map((i) => (
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
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {/* Delayed Orders */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900">
              Delayed Orders
            </h3>
            {delayedOrders.length > 0 && (
              <span className="px-3 py-1 bg-orange-100 text-orange-700 text-sm font-medium rounded-full">
                {delayedOrders.length}
              </span>
            )}
          </div>
        </div>

        <div className="divide-y divide-gray-200 max-h-96 overflow-y-auto">
          {delayedOrders.length === 0 ? (
            <div className="p-6 text-center text-gray-500">
              <svg
                className="mx-auto w-12 h-12 text-gray-400 mb-2"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <p className="text-sm">No delayed orders</p>
            </div>
          ) : (
            delayedOrders.map((order) => {
              const isUrgent = order.elapsed_minutes > 30;
              const badgeColor = isUrgent ? 'bg-red-100 text-red-700' : 'bg-orange-100 text-orange-700';

              return (
                <div
                  key={order.order_id}
                  className="p-4 hover:bg-gray-50 transition-colors cursor-pointer"
                  onClick={() => onNavigateToOrder?.(order.order_id)}
                >
                  <div className="flex items-start gap-4">
                    {/* Urgency Badge */}
                    <div className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center text-sm font-bold ${badgeColor}`}>
                      {order.elapsed_minutes}m
                    </div>

                    {/* Order Details */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <p className="text-sm font-semibold text-gray-900">
                            Order #{order.order_number}
                          </p>
                          <p className="text-xs text-gray-500 mt-1">
                            Customer: {order.masked_name} • {order.masked_phone}
                          </p>
                          <p className="text-xs text-gray-500 mt-1">
                            {order.item_count} items • {formatCurrency(order.total_amount)}
                          </p>
                        </div>
                        <span className={`px-2 py-1 text-xs font-medium rounded ${badgeColor}`}>
                          {order.status}
                        </span>
                      </div>

                      {isUrgent && (
                        <div className="mt-2 flex items-center gap-1 text-xs text-red-600">
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                            />
                          </svg>
                          <span>Urgent: Delayed &gt;30 minutes</span>
                        </div>
                      )}
                    </div>

                    {/* Navigation Arrow */}
                    <svg
                      className="w-5 h-5 text-gray-400 flex-shrink-0"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5l7 7-7 7"
                      />
                    </svg>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Restock Alerts */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900">
              Low Stock Alerts
            </h3>
            {restockAlerts.length > 0 && (
              <span className="px-3 py-1 bg-red-100 text-red-700 text-sm font-medium rounded-full">
                {restockAlerts.length}
              </span>
            )}
          </div>
        </div>

        <div className="divide-y divide-gray-200 max-h-96 overflow-y-auto">
          {restockAlerts.length === 0 ? (
            <div className="p-6 text-center text-gray-500">
              <svg
                className="mx-auto w-12 h-12 text-gray-400 mb-2"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <p className="text-sm">All products are well stocked</p>
            </div>
          ) : (
            restockAlerts.map((alert) => {
              const isCritical = alert.status === 'critical';
              const badgeColor = isCritical ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700';

              return (
                <div
                  key={alert.product_id}
                  className="p-4 hover:bg-gray-50 transition-colors cursor-pointer"
                  onClick={() => onNavigateToProduct?.(alert.product_id)}
                >
                  <div className="flex items-center gap-4">
                    {/* Product Image */}
                    {alert.image_url ? (
                      <img
                        src={alert.image_url}
                        alt={alert.product_name}
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
                            d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
                          />
                        </svg>
                      </div>
                    )}

                    {/* Product Details */}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-gray-900 truncate">
                        {alert.product_name}
                      </p>
                      <p className="text-xs text-gray-500 mt-1">
                        {alert.category_name} • SKU: {alert.sku}
                      </p>
                      <div className="flex items-center gap-3 mt-2">
                        <span className={`px-2 py-1 text-xs font-medium rounded ${badgeColor}`}>
                          {isCritical ? 'OUT OF STOCK' : 'LOW STOCK'}
                        </span>
                        <span className="text-xs text-gray-600">
                          {alert.current_stock} / {alert.low_stock_threshold}
                        </span>
                      </div>
                    </div>

                    {/* Reorder Info */}
                    <div className="text-right flex-shrink-0">
                      <p className="text-xs text-gray-500">Reorder</p>
                      <p className="text-sm font-semibold text-gray-900">
                        {formatNumber(alert.recommended_reorder, 0)} units
                      </p>
                      <p className="text-xs text-gray-500 mt-1">
                        ~{formatCurrency(alert.recommended_reorder * alert.cost_price)}
                      </p>
                    </div>

                    {/* Navigation Arrow */}
                    <svg
                      className="w-5 h-5 text-gray-400 flex-shrink-0"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5l7 7-7 7"
                      />
                    </svg>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};
