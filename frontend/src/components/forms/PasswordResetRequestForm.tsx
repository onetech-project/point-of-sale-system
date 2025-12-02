'use client';

import { useState } from 'react';
import { useTranslation } from '@/i18n/provider';
import { useRouter } from 'next/navigation';
import Button from '../ui/Button';
import Input from '../ui/Input';
import authService from '@/services/auth';

export default function PasswordResetRequestForm() {
  const { t } = useTranslation(['auth', 'common']);
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await authService.requestPasswordReset(email);
      setSuccess(true);
    } catch (err: any) {
      setError(err.message || t('auth.forgotPassword.error'));
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="text-center">
        <h3 className="text-lg font-semibold text-green-600 mb-4">
          {t('auth.forgotPassword.emailSent')}
        </h3>
        <p className="text-gray-600 mb-6">{t('auth.forgotPassword.checkEmail')}</p>
        <Button variant="secondary" onClick={() => router.push('/login')}>
          {t('auth.forgotPassword.backToLogin')}
        </Button>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <p className="text-gray-600 mb-4">{t('auth.forgotPassword.instructions')}</p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded">
          {error}
        </div>
      )}

      <Input
        type="email"
        label={t('auth.login.email')}
        value={email}
        onChange={e => setEmail(e.target.value)}
        required
        placeholder={t('auth.signup.emailPlaceholder')}
        disabled={loading}
      />

      <Button type="submit" disabled={loading} isLoading={loading} className="w-full">
        {loading ? t('auth.forgotPassword.sending') : t('auth.forgotPassword.sendResetLink')}
      </Button>

      <div className="text-center">
        <button
          type="button"
          onClick={() => router.push('/login')}
          className="text-sm text-blue-600 hover:text-blue-800"
        >
          {t('auth.forgotPassword.backToLogin')}
        </button>
      </div>
    </form>
  );
}
