'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import photoService from '@/services/photo';
import type { StorageQuota } from '@/types/photo';
import { HardDrive, AlertTriangle, CheckCircle } from 'lucide-react';

export default function TenantSettingsPage() {
  const router = useRouter();
  const { t } = useTranslation(['common', 'products']);
  const [storageQuota, setStorageQuota] = useState<StorageQuota | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchStorageQuota();
  }, []);

  const fetchStorageQuota = async () => {
    try {
      setLoading(true);
      const quota = await photoService.getStorageQuota();
      setStorageQuota(quota);
    } catch (err: any) {
      console.error('Failed to fetch storage quota:', err);
      setError(err.response?.data?.message || 'Failed to load storage quota');
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  const getQuotaStatusColor = (percentage: number): string => {
    if (percentage >= 95) return 'text-red-600 bg-red-100';
    if (percentage >= 90) return 'text-orange-600 bg-orange-100';
    if (percentage >= 80) return 'text-yellow-600 bg-yellow-100';
    return 'text-green-600 bg-green-100';
  };

  const getProgressBarColor = (percentage: number): string => {
    if (percentage >= 95) return 'bg-red-600';
    if (percentage >= 90) return 'bg-orange-600';
    if (percentage >= 80) return 'bg-yellow-600';
    return 'bg-green-600';
  };

  const getQuotaIcon = (percentage: number) => {
    if (percentage >= 80) {
      return <AlertTriangle className="w-5 h-5" />;
    }
    return <CheckCircle className="w-5 h-5" />;
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER]}>
      <DashboardLayout>
        <div className="max-w-4xl mx-auto">
          {/* Header */}
          <div className="mb-6">
            <button
              onClick={() => router.push('/settings')}
              className="mb-4 text-sm text-gray-600 hover:text-gray-900 flex items-center gap-1"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
              {t('common.back')}
            </button>
            <h1 className="text-3xl font-bold text-gray-900">Tenant Settings</h1>
            <p className="mt-1 text-sm text-gray-500">
              Manage your tenant configuration and storage
            </p>
          </div>

          {/* Storage Quota Card */}
          <div className="bg-white shadow rounded-lg p-6">
            <div className="flex items-center gap-3 mb-6">
              <div className="p-3 bg-purple-100 rounded-lg">
                <HardDrive className="w-6 h-6 text-purple-600" />
              </div>
              <div>
                <h2 className="text-xl font-semibold text-gray-900">Storage Quota</h2>
                <p className="text-sm text-gray-500">Product photo storage usage</p>
              </div>
            </div>

            {loading ? (
              <div className="flex items-center justify-center py-12">
                <div className="w-8 h-8 border-4 border-purple-500 border-t-transparent rounded-full animate-spin"></div>
              </div>
            ) : error ? (
              <div className="p-4 bg-red-50 border border-red-200 rounded-md">
                <p className="text-sm text-red-600">{error}</p>
              </div>
            ) : storageQuota ? (
              <div className="space-y-6">
                {/* Quota Warning Banner */}
                {storageQuota.usage_percentage >= 80 && (
                  <div className={`p-4 rounded-lg flex items-start gap-3 ${storageQuota.quota_exceeded
                      ? 'bg-red-50 border border-red-200'
                      : storageQuota.usage_percentage >= 95
                        ? 'bg-red-50 border border-red-200'
                        : storageQuota.usage_percentage >= 90
                          ? 'bg-orange-50 border border-orange-200'
                          : 'bg-yellow-50 border border-yellow-200'
                    }`}>
                    <AlertTriangle className={`w-5 h-5 flex-shrink-0 ${storageQuota.quota_exceeded || storageQuota.usage_percentage >= 95
                        ? 'text-red-600'
                        : storageQuota.usage_percentage >= 90
                          ? 'text-orange-600'
                          : 'text-yellow-600'
                      }`} />
                    <div className="flex-1">
                      <p className={`text-sm font-medium ${storageQuota.quota_exceeded || storageQuota.usage_percentage >= 95
                          ? 'text-red-800'
                          : storageQuota.usage_percentage >= 90
                            ? 'text-orange-800'
                            : 'text-yellow-800'
                        }`}>
                        {storageQuota.quota_exceeded
                          ? 'Storage quota exceeded!'
                          : storageQuota.usage_percentage >= 95
                            ? 'Storage nearly full (95%+)'
                            : storageQuota.usage_percentage >= 90
                              ? 'Storage approaching limit (90%+)'
                              : 'Storage warning (80%+)'}
                      </p>
                      <p className={`text-sm mt-1 ${storageQuota.quota_exceeded || storageQuota.usage_percentage >= 95
                          ? 'text-red-700'
                          : storageQuota.usage_percentage >= 90
                            ? 'text-orange-700'
                            : 'text-yellow-700'
                        }`}>
                        {storageQuota.quota_exceeded
                          ? 'You cannot upload more photos until you delete some existing photos or contact support to increase your quota.'
                          : 'Please consider deleting unused photos or contact support to increase your storage quota.'}
                      </p>
                    </div>
                  </div>
                )}

                {/* Usage Stats */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="bg-gray-50 p-4 rounded-lg">
                    <p className="text-sm text-gray-600 mb-1">Storage Used</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatBytes(storageQuota.storage_used_bytes)}
                    </p>
                  </div>
                  <div className="bg-gray-50 p-4 rounded-lg">
                    <p className="text-sm text-gray-600 mb-1">Total Quota</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatBytes(storageQuota.storage_quota_bytes)}
                    </p>
                  </div>
                  <div className="bg-gray-50 p-4 rounded-lg">
                    <p className="text-sm text-gray-600 mb-1">Available</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatBytes(storageQuota.available_bytes)}
                    </p>
                  </div>
                </div>

                {/* Progress Bar */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-gray-700">
                      Usage: {storageQuota.usage_percentage.toFixed(1)}%
                    </span>
                    <span className={`text-sm font-medium px-2 py-1 rounded-full flex items-center gap-1 ${getQuotaStatusColor(storageQuota.usage_percentage)}`}>
                      {getQuotaIcon(storageQuota.usage_percentage)}
                      {storageQuota.quota_exceeded
                        ? 'Exceeded'
                        : storageQuota.usage_percentage >= 95
                          ? 'Critical'
                          : storageQuota.usage_percentage >= 90
                            ? 'High'
                            : storageQuota.usage_percentage >= 80
                              ? 'Warning'
                              : 'Healthy'}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-4 overflow-hidden">
                    <div
                      className={`h-full transition-all duration-500 ${getProgressBarColor(storageQuota.usage_percentage)}`}
                      style={{ width: `${Math.min(storageQuota.usage_percentage, 100)}%` }}
                    />
                  </div>
                </div>

                {/* Photo Count */}
                <div className="pt-4 border-t border-gray-200">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-600">Total Photos Stored</span>
                    <span className="text-lg font-semibold text-gray-900">
                      {storageQuota.photo_count}
                    </span>
                  </div>
                </div>

                {/* Help Text */}
                <div className="pt-4 border-t border-gray-200">
                  <p className="text-xs text-gray-500">
                    💡 <strong>Tip:</strong> Delete unused product photos to free up storage space.
                    Each product can have up to 5 photos, with a maximum size of 5MB per photo.
                  </p>
                </div>
              </div>
            ) : null}
          </div>

          {/* Additional Settings Placeholder */}
          <div className="mt-6 bg-white shadow rounded-lg p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Tenant Information</h2>
            <p className="text-sm text-gray-500">
              Additional tenant settings will be available here in future updates.
            </p>
          </div>
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
