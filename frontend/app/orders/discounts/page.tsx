'use client';

import DiscountRuleManager from '@/components/discounts/DiscountRuleManager';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { ROLES } from '@/constants/roles';

export default function DiscountRulesPage() {
  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="space-y-6">
          <div className="rounded-lg bg-white p-6 shadow">
            <h1 className="text-3xl font-bold text-gray-900">Discount Rules</h1>
            <p className="mt-2 text-gray-600">
              Manage product and bundle discounts for order pricing.
            </p>
          </div>
          <DiscountRuleManager />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
