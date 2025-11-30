'use client';

import React, { useEffect, useState } from 'react';
import { useAuth } from '@/store/auth';
import { useRouter } from 'next/navigation';
import apiClient from '@/services/api';
import DashboardLayout from '@/components/layout/DashboardLayout';

interface TenantInfo {
  id: string;
  businessName: string;
  slug: string;
  status: string;
  createdAt: string;
}

export default function ProfilePage() {
  const { user, isAuthenticated, isLoading } = useAuth();
  const router = useRouter();
  const [tenantInfo, setTenantInfo] = useState<TenantInfo | null>(null);
  const [loadingTenant, setLoadingTenant] = useState(true);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, isLoading, router]);

  useEffect(() => {
    const fetchTenantInfo = async () => {
      if (!isAuthenticated) return;

      try {
        setLoadingTenant(true);
        const data = await apiClient.get<TenantInfo>('/api/tenant');
        setTenantInfo(data);
      } catch (error) {
        console.error('Failed to fetch tenant info:', error);
      } finally {
        setLoadingTenant(false);
      }
    };

    fetchTenantInfo();
  }, [isAuthenticated]);

  if (isLoading || loadingTenant) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading profile...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  const formatDate = (dateString: string) => {
    if (!dateString) return 'N/A';
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  return (
    <DashboardLayout>
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Profile</h1>
        <p className="mt-2 text-sm text-gray-600">
          View your account and organization information
        </p>
      </div>

        {/* User Information Card */}
        <div className="bg-white rounded-lg shadow mb-6">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-xl font-semibold text-gray-900 flex items-center">
              <svg className="w-5 h-5 mr-2 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              User Information
            </h2>
          </div>
          <div className="p-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Full Name
                </label>
                <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-900">
                  {user?.firstName && user?.lastName
                    ? `${user.firstName} ${user.lastName}`
                    : user?.firstName || user?.lastName || 'Not set'}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Email
                </label>
                <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-900">
                  {user?.email || 'N/A'}
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Role
                </label>
                <div className="px-4 py-3 bg-gray-50 rounded-lg">
                  <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                    user?.role === 'owner'
                      ? 'bg-purple-100 text-purple-800'
                      : user?.role === 'admin'
                      ? 'bg-blue-100 text-blue-800'
                      : 'bg-gray-100 text-gray-800'
                  }`}>
                    {user?.role?.toUpperCase() || 'N/A'}
                  </span>
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  User ID
                </label>
                <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-600 font-mono text-sm">
                  {user?.id || 'N/A'}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Tenant Information Card */}
        {tenantInfo && (
          <div className="bg-white rounded-lg shadow">
            <div className="px-6 py-4 border-b border-gray-200">
              <h2 className="text-xl font-semibold text-gray-900 flex items-center">
                <svg className="w-5 h-5 mr-2 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                </svg>
                Organization Information
              </h2>
              <p className="mt-1 text-sm text-gray-500">
                This information is managed by the organization owner
              </p>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Business Name
                  </label>
                  <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-900 font-medium">
                    {tenantInfo.businessName}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Organization Slug
                  </label>
                  <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-600 font-mono text-sm">
                    {tenantInfo.slug}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Status
                  </label>
                  <div className="px-4 py-3 bg-gray-50 rounded-lg">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      tenantInfo.status === 'active'
                        ? 'bg-green-100 text-green-800'
                        : tenantInfo.status === 'suspended'
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-red-100 text-red-800'
                    }`}>
                      <svg className="w-2 h-2 mr-1.5" fill="currentColor" viewBox="0 0 8 8">
                        <circle cx={4} cy={4} r={3} />
                      </svg>
                      {tenantInfo.status.toUpperCase()}
                    </span>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Created On
                  </label>
                  <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-600">
                    {formatDate(tenantInfo.createdAt)}
                  </div>
                </div>

                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Organization ID
                  </label>
                  <div className="px-4 py-3 bg-gray-50 rounded-lg text-gray-600 font-mono text-sm">
                    {tenantInfo.id}
                  </div>
                </div>
              </div>

              {/* Info Message */}
              <div className="mt-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                <div className="flex">
                  <svg className="w-5 h-5 text-blue-600 mr-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <div>
                    <h3 className="text-sm font-medium text-blue-900">
                      Read-Only Information
                    </h3>
                    <p className="mt-1 text-sm text-blue-700">
                      Organization information is managed by the account owner. Contact your administrator if you need to make changes.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Back Button */}
        <div className="mt-8">
          <button
            onClick={() => router.push('/dashboard')}
            className="inline-flex items-center px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors"
          >
            <svg className="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back to Dashboard
          </button>
        </div>
    </DashboardLayout>
  );
}
