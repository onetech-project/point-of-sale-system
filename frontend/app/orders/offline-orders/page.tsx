'use client';

import React from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { OfflineOrderList } from '@/components/orders/OfflineOrderList';

/**
 * Offline Orders List Page
 * Displays all offline orders with filtering and search capabilities
 * Accessible via Orders > Offline Orders in the navigation
 */
export default function OfflineOrdersPage() {
  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="space-y-6">
          {/* Page Header */}
          <div className="bg-white rounded-lg shadow p-6">
            <h1 className="text-3xl font-bold text-gray-900">Offline Orders</h1>
            <p className="text-gray-600 mt-2">
              Manage orders placed through phone, walk-in, or other offline channels
            </p>
          </div>

          {/* Offline Order List Component */}
          <OfflineOrderList />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
