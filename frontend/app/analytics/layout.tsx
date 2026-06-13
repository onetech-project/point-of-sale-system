'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import type { ReactNode } from 'react';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { ROLES } from '@/constants/roles';

interface AnalyticsLayoutProps {
  children: ReactNode;
}

const navItems = [
  { href: '/analytics', label: 'Overview' },
  { href: '/analytics/inventory', label: 'Inventory Analytics' },
];

export default function AnalyticsLayout({ children }: AnalyticsLayoutProps) {
  const pathname = usePathname();

  const isActive = (href: string) => {
    if (href === '/analytics') {
      return pathname === href;
    }
    return pathname === href || pathname.startsWith(`${href}/`);
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="mb-6 space-y-4">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Analytics</h1>
            <p className="mt-1 text-sm text-gray-500">
              Business insights, profitability analysis, and strategic planning.
            </p>
          </div>

          <nav className="flex space-x-1 rounded-lg bg-gray-100 p-1">
            {navItems.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                  isActive(item.href)
                    ? 'bg-white text-gray-900 shadow-sm'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </div>

        {children}
      </DashboardLayout>
    </ProtectedRoute>
  );
}
