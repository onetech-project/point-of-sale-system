'use client';

import { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import PublicLayout from '@/components/layout/PublicLayout';
import PasswordResetForm from '@/components/forms/PasswordResetForm';

function ResetPasswordContent() {
  const { t } = useTranslation(['auth', 'common']);
  const searchParams = useSearchParams();
  const token = searchParams.get('token');

  if (!token) {
    return (
      <PublicLayout>
        <div className="min-h-[calc(100vh-128px)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
          <div className="max-w-md w-full">
            <div className="bg-white rounded-2xl shadow-xl p-8 text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-red-100 rounded-2xl mb-4">
                <svg className="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <h2 className="text-2xl font-bold text-gray-900 mb-4">
                {t('auth.resetPassword.invalidLink')}
              </h2>
              <p className="text-gray-600">
                {t('auth.resetPassword.invalidLinkMessage')}
              </p>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  return (
    <PublicLayout>
      <div className="min-h-[calc(100vh-128px)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-2xl shadow-xl p-8 space-y-6">
            <div className="text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-primary-600 to-primary-700 rounded-2xl mb-4">
                <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                </svg>
              </div>
              <h2 className="text-3xl font-bold text-gray-900">
                {t('auth.resetPassword.title')}
              </h2>
            </div>
            <PasswordResetForm token={token} />
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    }>
      <ResetPasswordContent />
    </Suspense>
  );
}
