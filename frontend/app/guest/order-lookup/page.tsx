'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import PublicLayout from '@/components/layout/PublicLayout';

interface FormData {
  order_reference: string;
  email: string;
  phone: string;
}

interface FormErrors {
  order_reference?: string;
  email?: string;
  phone?: string;
}

export default function GuestOrderLookupPage() {
  const { t } = useTranslation(['guest_data', 'common']);
  const router = useRouter();

  const [formData, setFormData] = useState<FormData>({
    order_reference: '',
    email: '',
    phone: ''
  });

  const [errors, setErrors] = useState<FormErrors>({});
  const [isLoading, setIsLoading] = useState(false);
  const [serverError, setServerError] = useState('');

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
    if (errors[name as keyof FormErrors]) {
      setErrors(prev => ({
        ...prev,
        [name]: undefined
      }));
    }
    setServerError('');
  };

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    if (!formData.order_reference) {
      newErrors.order_reference = t('guest_data:errors.order_reference_required');
    }

    // At least one verification method required
    if (!formData.email && !formData.phone) {
      newErrors.email = t('guest_data:errors.verification_required');
      newErrors.phone = t('guest_data:errors.verification_required');
    }

    // Validate email format if provided
    if (formData.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = t('guest_data:errors.invalid_email');
    }

    // Validate phone format if provided (basic validation)
    if (formData.phone && !/^[+\d\s()-]{8,}$/.test(formData.phone)) {
      newErrors.phone = t('guest_data:errors.invalid_phone');
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setServerError('');

    if (!validateForm()) {
      return;
    }

    setIsLoading(true);

    try {
      // Navigate to data page with verification credentials
      const params = new URLSearchParams();
      if (formData.email) params.set('email', formData.email);
      if (formData.phone) params.set('phone', formData.phone);
      
      router.push(`/guest/data/${formData.order_reference}?${params.toString()}`);
    } catch (error: any) {
      setServerError(error.message || t('guest_data:errors.lookup_failed'));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <PublicLayout>
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-2xl shadow-xl p-8">
            {/* Header */}
            <div className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-100 rounded-full mb-4">
                <svg className="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">
                {t('guest_data:lookup.title')}
              </h1>
              <p className="text-gray-600">
                {t('guest_data:lookup.subtitle')}
              </p>
            </div>

            {/* Info Box */}
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm text-blue-700">
                    {t('guest_data:lookup.info_message')}
                  </p>
                </div>
              </div>
            </div>

            {/* Server Error */}
            {serverError && (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
                <div className="flex">
                  <div className="flex-shrink-0">
                    <svg className="h-5 w-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                    </svg>
                  </div>
                  <div className="ml-3">
                    <p className="text-sm text-red-700">{serverError}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Form */}
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Order Reference */}
              <div>
                <label htmlFor="order_reference" className="block text-sm font-medium text-gray-700 mb-2">
                  {t('guest_data:lookup.order_reference')}
                </label>
                <input
                  id="order_reference"
                  name="order_reference"
                  type="text"
                  value={formData.order_reference}
                  onChange={handleChange}
                  className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                    errors.order_reference ? 'border-red-300 bg-red-50' : 'border-gray-300'
                  }`}
                  placeholder="ORD-XXXX-XXXX"
                />
                {errors.order_reference && (
                  <p className="mt-2 text-sm text-red-600">{errors.order_reference}</p>
                )}
              </div>

              {/* Divider */}
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-gray-300"></div>
                </div>
                <div className="relative flex justify-center text-sm">
                  <span className="px-2 bg-white text-gray-500">
                    {t('guest_data:lookup.verification_label')}
                  </span>
                </div>
              </div>

              {/* Email */}
              <div>
                <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-2">
                  {t('guest_data:lookup.email')}
                </label>
                <input
                  id="email"
                  name="email"
                  type="email"
                  value={formData.email}
                  onChange={handleChange}
                  className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                    errors.email ? 'border-red-300 bg-red-50' : 'border-gray-300'
                  }`}
                  placeholder="your.email@example.com"
                />
                {errors.email && (
                  <p className="mt-2 text-sm text-red-600">{errors.email}</p>
                )}
              </div>

              {/* Phone */}
              <div>
                <label htmlFor="phone" className="block text-sm font-medium text-gray-700 mb-2">
                  {t('guest_data:lookup.phone')}
                </label>
                <input
                  id="phone"
                  name="phone"
                  type="tel"
                  value={formData.phone}
                  onChange={handleChange}
                  className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                    errors.phone ? 'border-red-300 bg-red-50' : 'border-gray-300'
                  }`}
                  placeholder="+62 812 3456 7890"
                />
                {errors.phone && (
                  <p className="mt-2 text-sm text-red-600">{errors.phone}</p>
                )}
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={isLoading}
                className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-medium hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? t('common:loading') : t('guest_data:lookup.submit')}
              </button>
            </form>

            {/* Privacy Note */}
            <div className="mt-6 text-center">
              <p className="text-xs text-gray-500">
                {t('guest_data:lookup.privacy_note')}
              </p>
            </div>
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}
