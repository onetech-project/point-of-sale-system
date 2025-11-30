'use client';

import { useTranslation } from '@/i18n/provider';
import PublicLayout from '@/components/layout/PublicLayout';
import PasswordResetRequestForm from '@/components/forms/PasswordResetRequestForm';

export default function ForgotPasswordPage() {
  const { t } = useTranslation(['auth', 'common']);

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
                {t('auth.forgotPassword.title')}
              </h2>
            </div>
            <PasswordResetRequestForm />
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}
