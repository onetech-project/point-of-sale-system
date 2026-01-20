'use client';

import React, { useEffect, useState } from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import ConsentSettingsSection from '@/components/consent/ConsentSettingsSection';
import { useTranslation } from 'react-i18next';

export default function PrivacySettingsPage() {
  const { t } = useTranslation('privacy_settings');

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="max-w-4xl mx-auto p-6">
          <h1 className="text-3xl font-bold mb-2">{t('page.title')}</h1>
          <p className="text-gray-600 mb-6">{t('page.subtitle')}</p>

          <div className="bg-white rounded-lg shadow p-6 mb-6">
            <div className="flex items-start mb-4">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-gray-900">{t('info.title')}</h3>
                <p className="mt-1 text-sm text-gray-600">{t('info.description')}</p>
              </div>
            </div>
          </div>

          <ConsentSettingsSection />

          <div className="bg-gray-50 rounded-lg border border-gray-200 p-4 mt-6">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">{t('compliance.title')}</h3>
            <p className="text-sm text-gray-600">{t('compliance.description')}</p>
          </div>
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
