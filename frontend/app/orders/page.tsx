'use client';

import React from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { OrderManagement } from '@/components/admin/OrderManagement';

/**
 * Orders Page
 * Main order management interface for tracking and managing customer orders
 * Accessible via Orders menu in the sidebar navigation
 */
export default function OrdersPage() {
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

          {/* Order Management Component */}
          {/* Tenant ID is extracted from session by API Gateway */}
          <OrderManagement />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
