'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES, type Role } from '@/constants/roles';

/**
 * Settings Page
 * Central hub for all administrative and configuration settings
 */
export default function SettingsPage() {
  const router = useRouter();
  const { user } = useAuth();

  const settingsCategories: Array<{
    title: string;
    description: string;
    icon: React.ReactNode;
    href: string;
    roles: Role[];
    color: string;
    comingSoon?: boolean;
  }> = [
      {
        title: 'Order Settings',
        description: 'Configure delivery types, fees, and order policies',
        icon: (
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        ),
        href: '/settings/orders',
        roles: [ROLES.OWNER, ROLES.MANAGER],
        color: 'blue',
      },
      {
        title: 'Tenant Settings',
        description: 'Configure tenant information and preferences',
        icon: (
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
            />
          </svg>
        ),
        href: '/settings/tenant',
        roles: [ROLES.OWNER],
        color: 'purple',
        comingSoon: true,
      },
      {
        title: 'User Management',
        description: 'Manage users, roles, and permissions',
        icon: (
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
            />
          </svg>
        ),
        href: '/users/invite',
        roles: [ROLES.OWNER, ROLES.MANAGER],
        color: 'green',
      },
      {
        title: 'Payment Settings',
        description: 'Configure payment gateways and methods',
        icon: (
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"
            />
          </svg>
        ),
        href: '/settings/payment',
        roles: [ROLES.OWNER],
        color: 'yellow',
      },
      {
        title: 'Notifications',
        description: 'Configure email notifications and view notification history',
        icon: (
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
            />
          </svg>
        ),
        href: '/settings/notifications',
        roles: [ROLES.OWNER, ROLES.MANAGER],
        color: 'red',
      },
      {
        title: 'System Settings',
        description: 'Configure system-wide settings',
        icon: (
          <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
            />
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
            />
          </svg>
        ),
        href: '/settings/system',
        roles: [ROLES.OWNER],
        color: 'gray',
        comingSoon: true,
      },
    ];

  const getColorClasses = (color: string) => {
    const colors: Record<string, { bg: string; text: string; hover: string }> = {
      blue: { bg: 'bg-blue-100', text: 'text-blue-600', hover: 'hover:bg-blue-50' },
      purple: { bg: 'bg-purple-100', text: 'text-purple-600', hover: 'hover:bg-purple-50' },
      green: { bg: 'bg-green-100', text: 'text-green-600', hover: 'hover:bg-green-50' },
      yellow: { bg: 'bg-yellow-100', text: 'text-yellow-600', hover: 'hover:bg-yellow-50' },
      red: { bg: 'bg-red-100', text: 'text-red-600', hover: 'hover:bg-red-50' },
      gray: { bg: 'bg-gray-100', text: 'text-gray-600', hover: 'hover:bg-gray-50' },
    };
    return colors[color] || colors.gray;
  };

  const userRole = user?.role;
  const availableSettings = settingsCategories.filter(
    (category) => userRole && category.roles.includes(userRole)
  );

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900">Settings</h1>
          <p className="mt-2 text-sm text-gray-600">
            Manage your system configuration and preferences
          </p>
        </div>

        {/* Settings Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {availableSettings.map((category) => {
            const colors = getColorClasses(category.color);
            return (
              <button
                key={category.href}
                onClick={() => !category.comingSoon && router.push(category.href)}
                disabled={category.comingSoon}
                className={`
                  relative bg-white rounded-lg shadow-sm border-2 border-gray-200 p-6 text-left
                  transition-all duration-200
                  ${category.comingSoon
                    ? 'opacity-60 cursor-not-allowed'
                    : 'hover:border-primary-300 hover:shadow-md cursor-pointer'
                  }
                `}
              >
                {/* Coming Soon Badge */}
                {category.comingSoon && (
                  <div className="absolute top-3 right-3 bg-gray-200 text-gray-600 text-xs font-medium px-2 py-1 rounded">
                    Coming Soon
                  </div>
                )}

                {/* Icon */}
                <div className={`inline-flex p-3 rounded-lg ${colors.bg} ${colors.text} mb-4`}>
                  {category.icon}
                </div>

                {/* Content */}
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  {category.title}
                </h3>
                <p className="text-sm text-gray-600">{category.description}</p>

                {/* Arrow Icon */}
                {!category.comingSoon && (
                  <div className="mt-4 flex items-center text-primary-600">
                    <span className="text-sm font-medium">Configure</span>
                    <svg
                      className="w-4 h-4 ml-1"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5l7 7-7 7"
                      />
                    </svg>
                  </div>
                )}
              </button>
            );
          })}
        </div>

        {/* Help Section */}
        <div className="mt-12 bg-blue-50 border border-blue-200 rounded-lg p-6">
          <div className="flex items-start">
            <div className="flex-shrink-0">
              <svg
                className="w-6 h-6 text-blue-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div className="ml-4">
              <h3 className="text-lg font-medium text-blue-900">Need Help?</h3>
              <p className="mt-1 text-sm text-blue-700">
                Contact support or refer to the documentation for assistance with system
                configuration.
              </p>
            </div>
          </div>
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
