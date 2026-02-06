'use client';

import React from 'react';
import Link from 'next/link';

interface QuickAction {
  label: string;
  description: string;
  href: string;
  icon: React.ReactNode;
  color: string;
}

interface QuickActionsProps {
  /**
   * Additional CSS classes for the container
   */
  className?: string;
}

/**
 * QuickActions Component
 * 
 * Provides one-click navigation to frequently accessed pages from the dashboard.
 * Currently includes:
 * - Team invitation page
 * - Settings page
 * 
 * @component
 * @example
 * ```tsx
 * <QuickActions />
 * ```
 */
export const QuickActions: React.FC<QuickActionsProps> = ({ className = '' }) => {
  const actions: QuickAction[] = [
    {
      label: 'Invite Team Members',
      description: 'Add new users to your organization',
      href: '/users/invite',
      color: 'bg-blue-500 hover:bg-blue-600',
      icon: (
        <svg
          className="w-6 h-6 text-white"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z"
          />
        </svg>
      ),
    },
    {
      label: 'Settings',
      description: 'Configure your tenant and preferences',
      href: '/settings',
      color: 'bg-gray-600 hover:bg-gray-700',
      icon: (
        <svg
          className="w-6 h-6 text-white"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
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
    },
  ];

  return (
    <div className={`bg-white rounded-lg shadow-sm border border-gray-200 p-6 ${className}`}>
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-gray-900">Quick Actions</h2>
        <p className="text-sm text-gray-600">Common tasks and shortcuts</p>
      </div>
      
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {actions.map((action) => (
          <Link
            key={action.href}
            href={action.href}
            className="group flex items-start gap-4 p-4 rounded-lg border border-gray-200 hover:border-gray-300 hover:shadow-md transition-all duration-200"
          >
            <div className={`flex-shrink-0 w-12 h-12 rounded-lg ${action.color} flex items-center justify-center transition-colors duration-200`}>
              {action.icon}
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-sm font-medium text-gray-900 group-hover:text-primary-600 transition-colors">
                {action.label}
              </h3>
              <p className="mt-1 text-xs text-gray-500 line-clamp-2">
                {action.description}
              </p>
            </div>
            <svg
              className="flex-shrink-0 w-5 h-5 text-gray-400 group-hover:text-primary-600 transition-colors"
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
          </Link>
        ))}
      </div>
    </div>
  );
};
