'use client';

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useRouter } from 'next/navigation';
import Button from '../ui/Button';
import Input from '../ui/Input';

interface PasswordResetFormProps {
  token: string;
}

export default function PasswordResetForm({ token }: PasswordResetFormProps) {
  const { t } = useTranslation('auth');
  const router = useRouter();
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  const getPasswordStrength = (pass: string) => {
    if (pass.length < 8) return { strength: 'weak', text: t('password.weak'), color: 'text-red-600' };
    if (pass.length < 12) return { strength: 'medium', text: t('password.medium'), color: 'text-yellow-600' };
    return { strength: 'strong', text: t('password.strong'), color: 'text-green-600' };
  };

  const passwordStrength = getPasswordStrength(password);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError(t('resetPassword.passwordMismatch'));
      return;
    }

    if (password.length < 8) {
      setError(t('resetPassword.passwordTooShort'));
      return;
    }

    setLoading(true);

    try {
      const response = await fetch('/api/auth/password-reset/reset', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          token,
          new_password: password,
        }),
      });

      const data = await response.json();

      if (response.ok) {
        setSuccess(true);
        setTimeout(() => {
          router.push('/auth/login');
        }, 2000);
      } else {
        setError(data.error || t('resetPassword.error'));
      }
    } catch (err) {
      setError(t('resetPassword.error'));
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="text-center">
        <h3 className="text-lg font-semibold text-green-600 mb-4">
          {t('resetPassword.success')}
        </h3>
        <p className="text-gray-600 mb-6">
          {t('resetPassword.redirecting')}
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded">
          {error}
        </div>
      )}

      <div>
        <Input
          type="password"
          label={t('newPassword')}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          placeholder={t('newPasswordPlaceholder')}
          disabled={loading}
        />
        {password && (
          <p className={`text-sm mt-1 ${passwordStrength.color}`}>
            {passwordStrength.text}
          </p>
        )}
      </div>

      <Input
        type="password"
        label={t('confirmPassword')}
        value={confirmPassword}
        onChange={(e) => setConfirmPassword(e.target.value)}
        required
        placeholder={t('confirmPasswordPlaceholder')}
        disabled={loading}
      />

      <Button
        type="submit"
        disabled={loading || password !== confirmPassword}
        isLoading={loading}
        className="w-full"
      >
        {loading ? t('resetPassword.resetting') : t('resetPassword.resetPassword')}
      </Button>
    </form>
  );
}
