'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import Link from 'next/link';
import { authService } from '@/services/auth';
import PublicLayout from '@/components/layout/PublicLayout';

interface FormData {
  businessName: string;
  email: string;
  password: string;
  confirmPassword: string;
  firstName: string;
  lastName: string;
}

interface FormErrors {
  businessName?: string;
  email?: string;
  password?: string;
  confirmPassword?: string;
  firstName?: string;
  lastName?: string;
}

export default function SignupPage() {
  const { t } = useTranslation(['auth', 'common']);
  const router = useRouter();

  const [formData, setFormData] = useState<FormData>({
    businessName: '',
    email: '',
    password: '',
    confirmPassword: '',
    firstName: '',
    lastName: '',
  });

  const [errors, setErrors] = useState<FormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [status, setStatus] = useState<'success' | 'error' | null>(null);
  const [serverError, setServerError] = useState('');
  const [passwordStrength, setPasswordStrength] = useState({
    minLength: false,
    hasLetterAndNumber: false,
    hasUpperCase: false,
    hasSpecialChar: false,
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));

    // Update password strength indicators in real-time
    if (name === 'password') {
      setPasswordStrength({
        minLength: value.length >= 8,
        hasLetterAndNumber: /(?=.*[A-Za-z])(?=.*\d)/.test(value),
        hasUpperCase: /[A-Z]/.test(value),
        hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(value),
      });
    }

    if (errors[name as keyof FormErrors]) {
      setErrors((prev) => ({
        ...prev,
        [name]: undefined,
      }));
    }
  };

  const validate = (): FormErrors => {
    const newErrors: FormErrors = {};

    if (!formData.businessName.trim()) {
      newErrors.businessName = t('auth.signup.errors.businessNameRequired');
    } else if (formData.businessName.length > 100) {
      newErrors.businessName = t('auth.signup.errors.businessNameTooLong');
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!formData.email.trim()) {
      newErrors.email = t('auth.signup.errors.emailRequired');
    } else if (!emailRegex.test(formData.email)) {
      newErrors.email = t('auth.signup.errors.emailInvalid');
    }

    if (!formData.password) {
      newErrors.password = t('auth.signup.errors.passwordRequired');
    } else if (formData.password.length < 8) {
      newErrors.password = t('auth.signup.errors.passwordTooShort');
    } else if (!/(?=.*[A-Za-z])(?=.*\d)/.test(formData.password)) {
      newErrors.password = t('auth.signup.errors.passwordWeak');
    }

    if (formData.password !== formData.confirmPassword) {
      newErrors.confirmPassword = t('auth.signup.errors.passwordMismatch');
    }

    return newErrors;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setServerError('');
    setStatus(null);

    const newErrors = validate();
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setIsSubmitting(true);

    try {
      await authService.registerTenant({
        businessName: formData.businessName,
        email: formData.email.toLowerCase(),
        password: formData.password,
        ownerProfile: {
          firstName: formData.firstName,
          lastName: formData.lastName
        }
      });

      setStatus('success');

      setTimeout(() => {
        router.push('/login');
      }, 5000);
    } catch (error) {
      setServerError(error instanceof Error ? error.message : t('auth.signup.errors.registrationFailed'));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <PublicLayout>
      <div className="min-h-[calc(100vh-128px)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-2xl shadow-xl p-8 space-y-6">
            {status === 'success' && (
              <div className='text-center'>
                <div className="inline-flex items-center justify-center w-16 h-16 bg-green-100 rounded-full mb-4">
                  <svg className="h-8 w-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <h2 className="text-2xl font-bold text-gray-900 mb-2">{t('auth.signup.successTitle')}</h2>
                <p className="text-gray-600">{t('auth.signup.successMessage')}</p>
              </div>
            )}
            {status === null && (
              <>
                {/* Header */}
                <div className="text-center">
                  <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-primary-600 to-primary-700 rounded-2xl mb-4">
                    <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
                    </svg>
                  </div>
                  <h2 className="text-3xl font-bold text-gray-900">
                    {t('auth.signup.title')}
                  </h2>
                  <p className="mt-2 text-sm text-gray-600">
                    {t('auth.signup.subtitle')}
                  </p>
                </div>

                {/* Form */}
                <form className="space-y-5" onSubmit={handleSubmit}>
                  {serverError && (
                    <div className="rounded-lg bg-red-50 border border-red-200 p-4">
                      <div className="flex">
                        <svg className="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                        </svg>
                        <div className="ml-3">
                          <p className="text-sm text-red-800">{serverError}</p>
                        </div>
                      </div>
                    </div>
                  )}

                  <div>
                    <label htmlFor="businessName" className="block text-sm font-medium text-gray-700 mb-1">
                      {t('auth.signup.businessName')} <span className="text-red-500">*</span>
                    </label>
                    <input
                      id="businessName"
                      name="businessName"
                      type="text"
                      required
                      className={`input-field ${errors.businessName ? 'border-red-300 focus:ring-red-500 focus:border-red-500' : ''}`}
                      placeholder={t('auth.signup.businessNamePlaceholder')}
                      value={formData.businessName}
                      onChange={handleChange}
                      disabled={isSubmitting}
                    />
                    {errors.businessName && (
                      <p className="mt-1 text-sm text-red-600">{errors.businessName}</p>
                    )}
                  </div>

                  <div>
                    <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
                      {t('auth.signup.email')} <span className="text-red-500">*</span>
                    </label>
                    <input
                      id="email"
                      name="email"
                      type="email"
                      autoComplete="email"
                      required
                      className={`input-field ${errors.email ? 'border-red-300 focus:ring-red-500 focus:border-red-500' : ''}`}
                      placeholder={t('auth.signup.emailPlaceholder')}
                      value={formData.email}
                      onChange={handleChange}
                      disabled={isSubmitting}
                    />
                    {errors.email && (
                      <p className="mt-1 text-sm text-red-600">{errors.email}</p>
                    )}
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label htmlFor="firstName" className="block text-sm font-medium text-gray-700 mb-1">
                        {t('auth.signup.firstName')}
                      </label>
                      <input
                        id="firstName"
                        name="firstName"
                        type="text"
                        className="input-field"
                        value={formData.firstName}
                        onChange={handleChange}
                        disabled={isSubmitting}
                      />
                    </div>
                    <div>
                      <label htmlFor="lastName" className="block text-sm font-medium text-gray-700 mb-1">
                        {t('auth.signup.lastName')}
                      </label>
                      <input
                        id="lastName"
                        name="lastName"
                        type="text"
                        className="input-field"
                        value={formData.lastName}
                        onChange={handleChange}
                        disabled={isSubmitting}
                      />
                    </div>
                  </div>

                  <div>
                    <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
                      {t('auth.signup.password')} <span className="text-red-500">*</span>
                    </label>
                    <input
                      id="password"
                      name="password"
                      type="password"
                      autoComplete="new-password"
                      required
                      className={`input-field ${errors.password ? 'border-red-300 focus:ring-red-500 focus:border-red-500' : ''}`}
                      placeholder="••••••••"
                      value={formData.password}
                      onChange={handleChange}
                      disabled={isSubmitting}
                    />
                    {errors.password && (
                      <p className="mt-1 text-sm text-red-600">{errors.password}</p>
                    )}

                    {/* Password Strength Indicators */}
                    {formData.password && (
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

                  <div>
                    <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-1">
                      {t('auth.signup.confirmPassword')} <span className="text-red-500">*</span>
                    </label>
                    <input
                      id="confirmPassword"
                      name="confirmPassword"
                      type="password"
                      autoComplete="new-password"
                      required
                      className={`input-field ${errors.confirmPassword ? 'border-red-300 focus:ring-red-500 focus:border-red-500' : ''}`}
                      placeholder="••••••••"
                      value={formData.confirmPassword}
                      onChange={handleChange}
                      disabled={isSubmitting}
                    />
                    {errors.confirmPassword && (
                      <p className="mt-1 text-sm text-red-600">{errors.confirmPassword}</p>
                    )}
                  </div>

                  <button
                    type="submit"
                    disabled={isSubmitting}
                    className="btn-primary w-full flex items-center justify-center"
                  >
                    {isSubmitting ? (
                      <>
                        <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        {t('auth.signup.submitting')}
                      </>
                    ) : (
                      t('auth.signup.submit')
                    )}
                  </button>
                </form>

                {/* Footer */}
                <div className="text-center pt-4 border-t border-gray-200">
                  <p className="text-sm text-gray-600">
                    {t('auth.signup.haveAccount')}{' '}
                    <Link href="/login" className="font-medium text-primary-600 hover:text-primary-500 transition-colors">
                      {t('auth.signup.signIn')}
                    </Link>
                  </p>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}

// Password Requirement Component
function PasswordRequirement({ met, text }: { met: boolean; text: string }) {
  return (
    <div className="flex items-center space-x-2">
      {met ? (
        <svg className="w-4 h-4 text-green-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
        </svg>
      ) : (
        <svg className="w-4 h-4 text-gray-300 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
        </svg>
      )}
      <span className={`text-xs ${met ? 'text-green-700 font-medium' : 'text-gray-500'}`}>
        {text}
      </span>
    </div>
  );
}
