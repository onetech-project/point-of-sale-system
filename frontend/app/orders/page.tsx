'use client';

import React, { Suspense } from 'react';
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
            <h1 className="text-3xl font-bold text-gray-900">Order Management</h1>
            <p className="text-gray-600 mt-2">
              Manage and track all customer orders
            </p>
          </div>

          {isNewOfflineMode ? (
            <OfflineOrderForm
              onSuccess={(orderId) => {
                router.push(`/orders?order_id=${encodeURIComponent(orderId)}&order_type=offline`);
              }}
              onCancel={() => router.push('/orders')}
            />
          ) : (
            <OrderManagement
              initialOrderId={initialOrderId}
              initialOrderType={initialOrderType}
            />
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
