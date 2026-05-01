'use client';

import React from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { OfflineOrderForm } from '@/components/orders/OfflineOrderForm';

/**
 * Create Offline Order Page
 * Form for creating new offline orders with customer and delivery details
 * Accessible via "Create New Order" button on offline orders list
 */
export default function NewOfflineOrderPage() {
  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="space-y-6">
          {/* Page Header */}
          <div className="bg-white rounded-lg shadow p-6">
            <h1 className="text-3xl font-bold text-gray-900">Create Offline Order</h1>
            <p className="text-gray-600 mt-2">
              Record a new order from phone, walk-in, or other offline channels
            </p>
          </div>

          {/* Offline Order Form Component */}
          <OfflineOrderForm />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
