'use client';

import React from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import { NotificationSettings } from '../../../src/components/admin/NotificationSettings';

export default function NotificationsPage() {
  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="container mx-auto py-8">
          <NotificationSettings />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
