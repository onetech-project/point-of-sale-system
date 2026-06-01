'use client';

import { useEffect, useRef, useState } from 'react';
import { Copy, Download, ExternalLink } from 'lucide-react';
import { QRCodeCanvas } from 'qrcode.react';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { ROLES } from '@/constants/roles';
import { tenantService } from '@/services/tenant';
import type { TenantInfo } from '@/types/tenant';

const DEFAULT_SITE_URL = 'https://posku.web.id';

function getMenuBaseUrl() {
  const configuredUrl = process.env.NEXT_PUBLIC_SITE_URL;

  if (configuredUrl) {
    return configuredUrl;
  }

  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin;
  }

  return DEFAULT_SITE_URL;
}

export function buildMenuUrl(slug: string) {
  return new URL(`/menu/${encodeURIComponent(slug)}`, `${getMenuBaseUrl()}/`).toString();
}

export default function MenuQrPage() {
  const qrRef = useRef<HTMLDivElement>(null);
  const [tenant, setTenant] = useState<TenantInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let mounted = true;

    tenantService
      .getTenantInfo()
      .then(data => {
        if (!mounted) return;
        setTenant(data);
      })
      .catch(() => {
        if (!mounted) return;
        setError('Failed to load tenant information.');
      })
      .finally(() => {
        if (!mounted) return;
        setLoading(false);
      });

    return () => {
      mounted = false;
    };
  }, []);

  const menuUrl = tenant?.slug ? buildMenuUrl(tenant.slug) : '';

  const copyLink = async () => {
    if (!menuUrl) return;

    await navigator.clipboard.writeText(menuUrl);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  const downloadPng = () => {
    const canvas = qrRef.current?.querySelector('canvas');
    if (!canvas || !tenant) return;

    const link = document.createElement('a');
    link.href = canvas.toDataURL('image/png');
    link.download = `${tenant.slug}-menu-qr.png`;
    link.click();
  };

  const openMenu = () => {
    if (!menuUrl) return;
    window.open(menuUrl, '_blank', 'noopener,noreferrer');
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER, ROLES.CASHIER]}>
      <DashboardLayout>
        <main className="max-w-4xl mx-auto p-6">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-gray-900">QR Generator</h1>
            <p className="text-gray-600 mt-2">Share your public menu with customers.</p>
          </div>

          <section className="bg-white border border-gray-200 rounded-lg shadow-sm p-6">
            {loading && (
              <div role="status" className="text-gray-600">
                Loading menu QR...
              </div>
            )}

            {!loading && error && (
              <div role="alert" className="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
                {error}
              </div>
            )}

            {!loading && !error && tenant && (
              <div className="grid gap-8 lg:grid-cols-[280px_1fr]">
                <div className="flex flex-col items-center">
                  <div ref={qrRef} className="rounded-lg border border-gray-200 bg-white p-4">
                    <QRCodeCanvas
                      value={menuUrl}
                      size={240}
                      includeMargin
                      level="M"
                      aria-label="Public menu QR code"
                    />
                  </div>
                </div>

                <div className="min-w-0 space-y-5">
                  <div>
                    <p className="text-sm font-medium text-gray-500">Business</p>
                    <p className="mt-1 text-lg font-semibold text-gray-900">{tenant.businessName}</p>
                  </div>

                  <div>
                    <label htmlFor="menu-url" className="text-sm font-medium text-gray-500">
                      Menu link
                    </label>
                    <input
                      id="menu-url"
                      readOnly
                      value={menuUrl}
                      className="mt-2 w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-900"
                    />
                  </div>

                  <div className="flex flex-wrap gap-3">
                    <button
                      type="button"
                      onClick={copyLink}
                      className="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700"
                    >
                      <Copy className="h-4 w-4" aria-hidden="true" />
                      {copied ? 'Copied' : 'Copy link'}
                    </button>
                    <button
                      type="button"
                      onClick={openMenu}
                      className="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                    >
                      <ExternalLink className="h-4 w-4" aria-hidden="true" />
                      Open menu
                    </button>
                    <button
                      type="button"
                      onClick={downloadPng}
                      className="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                    >
                      <Download className="h-4 w-4" aria-hidden="true" />
                      Download PNG
                    </button>
                  </div>
                </div>
              </div>
            )}

            {!loading && !error && !tenant && (
              <div role="alert" className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 text-yellow-800">
                Tenant information is unavailable.
              </div>
            )}
          </section>
        </main>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
