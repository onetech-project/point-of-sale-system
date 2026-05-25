'use client';

import { Suspense, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, MailCheck } from 'lucide-react';
import PublicLayout from '@/components/layout/PublicLayout';
import { useTranslation } from '@/i18n/provider';
import authService from '@/services/auth';

function ResendVerificationContent() {
  const { t } = useTranslation(['auth']);
  const searchParams = useSearchParams();
  const [email, setEmail] = useState(searchParams.get('email') || '');
  const [emailError, setEmailError] = useState('');
  const [serverError, setServerError] = useState('');
  const [status, setStatus] = useState<'idle' | 'success'>('idle');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validateEmail = () => {
    const normalizedEmail = email.trim().toLowerCase();
    if (!normalizedEmail) {
      setEmailError(t('auth.resendVerification.emailRequired'));
      return null;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
      setEmailError(t('auth.resendVerification.emailInvalid'));
      return null;
    }
    return normalizedEmail;
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setEmailError('');
    setServerError('');

    const normalizedEmail = validateEmail();
    if (!normalizedEmail) {
      return;
    }

    setIsSubmitting(true);
    try {
      await authService.resendVerificationEmail(normalizedEmail);
      setStatus('success');
    } catch (error) {
      setServerError(error instanceof Error ? error.message : t('auth.resendVerification.error'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <PublicLayout>
      <div className="min-h-[calc(100vh-128px)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-2xl shadow-xl p-8 space-y-6">
            <div className="text-center">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-primary-100 rounded-2xl mb-4">
                <MailCheck className="w-8 h-8 text-primary-600" aria-hidden="true" />
              </div>
              <h1 className="text-3xl font-bold text-gray-900">
                {t('auth.resendVerification.title')}
              </h1>
              <p className="mt-2 text-sm text-gray-600">
                {t('auth.resendVerification.instructions')}
              </p>
            </div>

            {status === 'success' && (
              <div className="rounded-lg bg-green-50 border border-green-200 p-4 text-sm text-green-800">
                {t('auth.resendVerification.successMessage')}
              </div>
            )}

            {serverError && (
              <div className="rounded-lg bg-red-50 border border-red-200 p-4 text-sm text-red-800">
                {serverError}
              </div>
            )}

            <form className="space-y-5" onSubmit={handleSubmit}>
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
                  {t('auth.resendVerification.email')}
                </label>
                <input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  required
                  className={`input-field ${emailError ? 'border-red-300 focus:ring-red-500 focus:border-red-500' : ''}`}
                  placeholder={t('auth.resendVerification.emailPlaceholder')}
                  value={email}
                  onChange={(event) => {
                    setEmail(event.target.value);
                    if (emailError) setEmailError('');
                    if (serverError) setServerError('');
                  }}
                  disabled={isSubmitting}
                />
                {emailError && (
                  <p className="mt-1 text-sm text-red-600">{emailError}</p>
                )}
              </div>

              <button
                type="submit"
                disabled={isSubmitting}
                className="btn-primary w-full flex items-center justify-center gap-2"
              >
                <MailCheck className="h-5 w-5" aria-hidden="true" />
                {isSubmitting
                  ? t('auth.resendVerification.sending')
                  : t('auth.resendVerification.submit')}
              </button>
            </form>

            <div className="flex flex-col gap-3 border-t border-gray-200 pt-4 text-center sm:flex-row sm:items-center sm:justify-between sm:text-left">
              <Link
                href="/login"
                className="inline-flex items-center justify-center gap-2 text-sm font-medium text-primary-600 transition-colors hover:text-primary-500"
              >
                <ArrowLeft className="h-4 w-4" aria-hidden="true" />
                {t('auth.resendVerification.backToLogin')}
              </Link>
              <Link
                href="/signup"
                className="text-sm font-medium text-primary-600 transition-colors hover:text-primary-500"
              >
                {t('auth.resendVerification.createAccount')}
              </Link>
            </div>
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}

export default function ResendVerificationPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center">Loading...</div>}>
      <ResendVerificationContent />
    </Suspense>
  );
}
