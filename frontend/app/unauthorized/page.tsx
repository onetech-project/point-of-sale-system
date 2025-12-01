'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import DashboardLayout from '@/components/layout/DashboardLayout';

export default function UnauthorizedPage() {
  const router = useRouter();
  const { user } = useAuth();

  return (
    <DashboardLayout>
      <div className="min-h-[60vh] flex items-center justify-center">
        <div className="text-center max-w-md px-4">
          {/* Icon */}
          <div className="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-red-100 mb-4">
            <svg
              className="h-8 w-8 text-red-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
          </div>

          {/* Title */}
          <h1 className="text-3xl font-bold text-gray-900 mb-2">
            Access Denied
          </h1>

          {/* Message */}
          <p className="text-gray-600 mb-2">
            You don't have permission to access this page.
          </p>
          
          {user?.role && (
            <p className="text-sm text-gray-500 mb-6">
              Your current role: <span className="font-semibold text-primary-600">{user.role}</span>
            </p>
          )}

          <p className="text-sm text-gray-500 mb-8">
            Contact your administrator if you believe this is an error.
          </p>

          {/* Actions */}
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <button
              onClick={() => router.back()}
              className="px-6 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
            >
              Go Back
            </button>
            <button
              onClick={() => router.push('/dashboard')}
              className="px-6 py-2 text-sm font-medium text-white bg-primary-600 rounded-lg hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
            >
              Go to Dashboard
            </button>
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
