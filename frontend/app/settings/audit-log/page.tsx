'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import { AuditLog } from '@/components/audit/AuditLog';

/**
 * Audit Log Page - T112
 * Displays audit trail for tenant owners to view all data access and modifications
 * per UU PDP compliance requirements (User Story 4)
 */
export default function AuditLogPage() {
  const router = useRouter();

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER]}>
      <DashboardLayout>
        <div className="container mx-auto py-8">
          <div className="mb-6">
            <button
              onClick={() => router.push('/settings')}
              className="flex items-center space-x-2 text-gray-600 hover:text-gray-900 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
              <span>Back to Settings</span>
            </button>
          </div>
          <AuditLog />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
