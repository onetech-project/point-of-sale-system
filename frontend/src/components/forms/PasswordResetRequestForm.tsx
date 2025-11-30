'use client';

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useRouter } from 'next/navigation';
import Button from '../ui/Button';
import Input from '../ui/Input';

export default function PasswordResetRequestForm() {
  const { t } = useTranslation('auth');
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
      const response = await fetch('/api/auth/password-reset/request', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email }),
      });

      const data = await response.json();

      if (response.ok) {
        setSuccess(true);
      } else {
        setError(data.error || t('forgotPassword.error'));
      }
    } catch (err) {
      setError(t('forgotPassword.error'));
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="text-center">
        <h3 className="text-lg font-semibold text-green-600 mb-4">
          {t('forgotPassword.emailSent')}
        </h3>
        <p className="text-gray-600 mb-6">
          {t('forgotPassword.checkEmail')}
        </p>
        <Button
          variant="secondary"
          onClick={() => router.push('/auth/login')}
        >
          {t('forgotPassword.backToLogin')}
        </Button>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <p className="text-gray-600 mb-4">
          {t('forgotPassword.instructions')}
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded">
          {error}
        </div>
      )}

      <Input
        type="email"
        label={t('email')}
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        required
        placeholder={t('emailPlaceholder')}
        disabled={loading}
      />

      <Button
        type="submit"
        disabled={loading}
        isLoading={loading}
        className="w-full"
      >
        {loading ? t('forgotPassword.sending') : t('forgotPassword.sendResetLink')}
      </Button>

      <div className="text-center">
        <button
          type="button"
          onClick={() => router.push('/auth/login')}
          className="text-sm text-blue-600 hover:text-blue-800"
        >
          {t('forgotPassword.backToLogin')}
        </button>
      </div>
    </form>
  );
}
