'use client';

import { useState } from 'react';
import { useTranslation } from '@/i18n/provider';
import { useRouter } from 'next/navigation';
import Button from '../ui/Button';
import Input from '../ui/Input';
import authService from '@/services/auth';

interface PasswordResetFormProps {
  token: string;
}

export default function PasswordResetForm({ token }: PasswordResetFormProps) {
  const { t } = useTranslation(['auth', 'common']);
  const router = useRouter();
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [passwordStrength, setPasswordStrength] = useState({
    minLength: false,
    hasLetterAndNumber: false,
    hasUpperCase: false,
    hasSpecialChar: false,
  });

  const handlePasswordChange = (value: string) => {
    setPassword(value);
    setPasswordStrength({
      minLength: value.length >= 8,
      hasLetterAndNumber: /(?=.*[A-Za-z])(?=.*\d)/.test(value),
      hasUpperCase: /[A-Z]/.test(value),
      hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(value),
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError(t('auth.resetPassword.passwordMismatch'));
      return;
    }

    if (!passwordStrength.minLength || !passwordStrength.hasLetterAndNumber) {
      setError(t('auth.resetPassword.passwordTooWeak'));
      return;
    }

    setLoading(true);

    try {
      await authService.resetPassword(token, password);
      setSuccess(true);
      setTimeout(() => {
        router.push('/login');
      }, 2000);
    } catch (err: any) {
      setError(err.message || t('auth.resetPassword.error'));
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="text-center">
        <h3 className="text-lg font-semibold text-green-600 mb-4">
          {t('auth.resetPassword.success')}
        </h3>
        <p className="text-gray-600 mb-6">{t('auth.resetPassword.redirecting')}</p>
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
          label={t('auth.resetPassword.newPassword')}
          value={password}
          onChange={e => handlePasswordChange(e.target.value)}
          required
          placeholder={t('auth.resetPassword.newPasswordPlaceholder')}
          disabled={loading}
        />

        {/* Password Strength Indicators */}
        {password && (
          <div className="mt-3 space-y-2">
            <p className="text-xs font-medium text-gray-700">
              {t('auth.signup.passwordRequirements')}
            </p>
            <div className="space-y-1.5">
              <PasswordRequirement
                met={passwordStrength.minLength}
                text={t('auth.signup.passwordMinLength')}
              />
              <PasswordRequirement
                met={passwordStrength.hasLetterAndNumber}
                text={t('auth.signup.passwordLetterNumber')}
              />
              <PasswordRequirement
                met={passwordStrength.hasUpperCase}
                text={t('auth.signup.passwordUpperCase')}
              />
              <PasswordRequirement
                met={passwordStrength.hasSpecialChar}
                text={t('auth.signup.passwordSpecialChar')}
              />
            </div>
          </div>
        )}
      </div>

      <Input
        type="password"
        label={t('auth.resetPassword.confirmPassword')}
        value={confirmPassword}
        onChange={e => setConfirmPassword(e.target.value)}
        required
        placeholder={t('auth.resetPassword.confirmPasswordPlaceholder')}
        disabled={loading}
      />

      <Button
        type="submit"
        disabled={loading || password !== confirmPassword}
        isLoading={loading}
        className="w-full"
      >
        {loading ? t('auth.resetPassword.resetting') : t('auth.resetPassword.resetPassword')}
      </Button>
    </form>
  );
}

function PasswordRequirement({ met, text }: { met: boolean; text: string }) {
  return (
    <div className="flex items-center space-x-2">
      {met ? (
        <svg
          className="w-4 h-4 text-green-500 flex-shrink-0"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
      ) : (
        <svg
          className="w-4 h-4 text-gray-300 flex-shrink-0"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
            clipRule="evenodd"
          />
        </svg>
      )}
      <span className={`text-xs ${met ? 'text-green-700 font-medium' : 'text-gray-500'}`}>
        {text}
      </span>
    </div>
  );
}
