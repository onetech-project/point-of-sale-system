'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { platformService, PlatformAdmin } from '@/services/platform';

const NAV_ITEMS = [
  { href: '/platform/dashboard', label: 'Dashboard' },
  { href: '/platform/tenants', label: 'Tenants' },
  { href: '/platform/tickets', label: 'Tickets' },
];

export function PlatformShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [admin, setAdmin] = useState<PlatformAdmin | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    platformService
      .session()
      .then(response => setAdmin(response.admin))
      .catch(() => router.replace('/platform/login'))
      .finally(() => setLoading(false));
  }, [router]);

  const logout = async () => {
    await platformService.logout();
    router.replace('/platform/login');
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50">
        <div className="h-10 w-10 animate-spin rounded-full border-4 border-gray-900 border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 lg:px-8">
          <div>
            <Link href="/platform/dashboard" className="text-lg font-bold text-gray-900">
              Posku Platform
            </Link>
            {admin && <p className="text-sm text-gray-500">{admin.name}</p>}
          </div>
          <button
            type="button"
            onClick={logout}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Logout
          </button>
        </div>
      </header>
      <div className="mx-auto flex max-w-7xl gap-6 px-4 py-6 lg:px-8">
        <aside className="hidden w-52 shrink-0 lg:block">
          <nav className="space-y-1">
            {NAV_ITEMS.map(item => {
              const active = pathname === item.href || pathname?.startsWith(`${item.href}/`);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`block rounded-lg px-3 py-2 text-sm font-medium ${
                    active ? 'bg-gray-900 text-white' : 'text-gray-700 hover:bg-gray-100'
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </aside>
        <main className="min-w-0 flex-1">{children}</main>
      </div>
    </div>
  );
}

export function PlatformMetric({
  label,
  value,
  tone = 'default',
}: {
  label: string;
  value: string | number;
  tone?: 'default' | 'warning' | 'danger' | 'success';
}) {
  const toneClass =
    tone === 'warning'
      ? 'border-yellow-200 bg-yellow-50'
      : tone === 'danger'
        ? 'border-red-200 bg-red-50'
        : tone === 'success'
          ? 'border-green-200 bg-green-50'
          : 'border-gray-200 bg-white';

  return (
    <div className={`rounded-lg border p-4 ${toneClass}`}>
      <p className="text-sm font-medium text-gray-500">{label}</p>
      <p className="mt-2 text-2xl font-bold text-gray-900">{value}</p>
    </div>
  );
}
