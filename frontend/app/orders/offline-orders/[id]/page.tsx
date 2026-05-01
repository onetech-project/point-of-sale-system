'use client';

import React, { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { OfflineOrderDetail } from '@/components/orders/OfflineOrderDetail';
import offlineOrderService from '@/services/offlineOrders';
import type { OfflineOrder } from '@/types/offlineOrder';

/**
 * Offline Order Detail Page
 * Displays complete information about a specific offline order
 * Accessible by clicking on an order in the offline orders list
 */
export default function OfflineOrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const orderId = params.id as string;

  const [order, setOrder] = useState<OfflineOrder | null>(null);
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchOrder = async () => {
      if (!orderId) {
        setError('Order ID is required');
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(null);
        const data = await offlineOrderService.getOfflineOrderWithDetails(orderId);
        setOrder(data.order);
        setItems(data.items || []);
      } catch (err: any) {
        console.error('Failed to fetch offline order:', err);
        setError(err.response?.data?.message || 'Failed to load order details');

        // Redirect to list if order not found (404)
        if (err.response?.status === 404) {
          setTimeout(() => {
            router.push('/orders/offline-orders');
          }, 2000);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchOrder();
  }, [orderId, router]);

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="space-y-6">
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
                onClick={() => router.push('/orders/offline-orders')}
                className="mt-4 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors"
              >
                Back to Orders
              </button>
            </div>
          )}

          {/* Order Detail Component */}
          {order && !loading && !error && <OfflineOrderDetail order={order} items={items} />}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
