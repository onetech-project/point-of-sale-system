'use client';

import { useMemo } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { ShieldAlert } from 'lucide-react';

export default function AccountUnavailablePage() {
  const searchParams = useSearchParams();
  const status = searchParams?.get('status');
  const message = useMemo(() => {
    if (status === 'inactive') {
      return 'This tenant account is inactive. Please contact platform support or your administrator.';
    }
    return 'This tenant account is suspended. Please contact platform support or your administrator.';
  }, [status]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-12">
      <section className="w-full max-w-md rounded-lg border border-gray-200 bg-white p-8 text-center shadow-sm">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-red-50 text-red-600">
          <ShieldAlert className="h-7 w-7" aria-hidden="true" />
        </div>
        <h1 className="mt-5 text-2xl font-bold text-gray-900">Tenant Account Unavailable</h1>
        <p className="mt-3 text-sm leading-6 text-gray-600">{message}</p>
        <Link
          href="/login"
          className="mt-6 inline-flex w-full items-center justify-center rounded-lg bg-gray-900 px-4 py-3 text-sm font-semibold text-white hover:bg-gray-800"
        >
          Back to login
        </Link>
      </section>
    </main>
  );
}
