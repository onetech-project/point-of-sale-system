'use client';

import React, { Suspense } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { OrderManagement } from '@/components/admin/OrderManagement';
import { OfflineOrderForm } from '@/components/orders/OfflineOrderForm';

/**
 * Orders Page
 * Main order management interface for tracking and managing customer orders
 * Accessible via Orders menu in the sidebar navigation
 */
function OrdersPageContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const initialOrderId = searchParams.get('order_id') || undefined;
  const requestedOrderType = searchParams.get('order_type');
  const initialOrderType = requestedOrderType === 'offline' ? 'offline' : 'online';
  const mode = searchParams.get('mode');
  const isNewOfflineMode = mode === 'new-offline';

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="space-y-6">
          {/* Page Header */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h1 className="text-3xl font-bold text-gray-900">Order Management</h1>
                <p className="text-gray-600 mt-2">Manage and track all customer orders</p>
              </div>
              <Link
                href="/orders/discounts"
                className="inline-flex items-center justify-center rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Discount rules
              </Link>
            </div>
          </div>

          {isNewOfflineMode ? (
            <OfflineOrderForm
              onSuccess={orderId => {
                router.push(`/orders?order_id=${encodeURIComponent(orderId)}&order_type=offline`);
              }}
              onCancel={() => router.push('/orders')}
            />
          ) : (
            <OrderManagement initialOrderId={initialOrderId} initialOrderType={initialOrderType} />
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}

export default function OrdersPage() {
  return (
    <Suspense fallback={null}>
      <OrdersPageContent />
    </Suspense>
  );
}
