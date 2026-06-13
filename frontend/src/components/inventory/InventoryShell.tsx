'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import type { ReactNode } from 'react';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { ROLES } from '@/constants/roles';

interface InventoryShellProps {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}

const navItems = [
  { href: '/inventory', label: 'Overview' },
  { href: '/inventory/ingredients', label: 'Ingredients' },
  { href: '/inventory/uoms', label: 'Units' },
  { href: '/inventory/stock', label: 'Stock' },
  { href: '/inventory/recipes', label: 'Recipes' },
  { href: '/inventory/bundles', label: 'Bundles' },
];

export default function InventoryShell({
  title,
  subtitle,
  actions,
  children,
}: InventoryShellProps) {
  const pathname = usePathname();

  const isActive = (href: string) => {
    if (href === '/inventory') {
      return pathname === href;
    }
    return pathname === href || pathname.startsWith(`${href}/`);
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="mb-6 space-y-4">
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">{title}</h1>
              {subtitle && <p className="mt-1 text-sm text-gray-500">{subtitle}</p>}
            </div>
            {actions && <div className="flex flex-wrap gap-2 md:justify-end">{actions}</div>}
          </div>

          <div className="border-b border-gray-200">
            <nav className="-mb-px flex gap-6 overflow-x-auto" aria-label="Inventory navigation">
              {navItems.map(item => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`whitespace-nowrap border-b-2 px-1 pb-3 text-sm font-medium transition-colors ${
                    isActive(item.href)
                      ? 'border-primary-600 text-primary-700'
                      : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                  }`}
                >
                  {item.label}
                </Link>
              ))}
            </nav>
          </div>
        </div>

        {children}
      </DashboardLayout>
    </ProtectedRoute>
  );
}
