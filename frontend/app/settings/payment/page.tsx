'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import paymentService, { MidtransConfig } from '@/services/payment';

/**
 * Payment Settings Page
 * Configure Midtrans payment gateway credentials
 */
export default function PaymentSettingsPage() {
  const router = useRouter();
  const { user } = useAuth();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<MidtransConfig>({
    tenant_id: '',
    server_key: '',
    client_key: '',
    merchant_id: '',
    environment: 'sandbox',
    is_configured: false,
  });
  const [showServerKey, setShowServerKey] = useState(false);
  const [showClientKey, setShowClientKey] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (user?.tenantId) {
      fetchMidtransConfig();
    }
  }, [user?.tenantId]);

  const fetchMidtransConfig = async () => {
    if (!user?.tenantId) return;

    try {
      setLoading(true);
      const data = await paymentService.getMidtransConfig(user.tenantId);
      setConfig(data);
    } catch (error: any) {
      console.error('Failed to fetch Midtrans config:', error);
      setErrorMessage('Failed to load payment configuration');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!user?.tenantId) {
      setErrorMessage('Tenant ID is missing');
      return;
    }

    setSaving(true);
    setSuccessMessage('');
    setErrorMessage('');

    try {
      await paymentService.updateMidtransConfig(user.tenantId, {
        server_key: config.server_key,
        client_key: config.client_key,
        merchant_id: config.merchant_id,
        environment: config.environment,
      });

      setSuccessMessage('Payment configuration saved successfully');
      setTimeout(() => setSuccessMessage(''), 5000);
      await fetchMidtransConfig(); // Refresh config
    } catch (error: any) {
      console.error('Failed to save Midtrans config:', error);
      setErrorMessage(
        error.response?.data?.error || 'Failed to save payment configuration'
      );
      setTimeout(() => setErrorMessage(''), 5000);
    } finally {
      setSaving(false);
    }
  };

  const handleInputChange = (field: keyof MidtransConfig, value: string) => {
    setConfig((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER]}>
      <DashboardLayout>
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <button
              onClick={() => router.push('/settings')}
              className="flex items-center text-gray-600 hover:text-gray-900 mb-4 transition-colors"
            >
              <svg
                className="w-5 h-5 mr-2"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
              Back to Settings
            </button>
            <h1 className="text-3xl font-bold text-gray-900">Payment Settings</h1>
            <p className="mt-2 text-gray-600">
              Configure your Midtrans payment gateway credentials for QRIS and other
              payment methods.
            </p>
          </div>

          {/* Status Messages */}
          {successMessage && (
            <div className="mb-6 bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg flex items-center">
              <svg
                className="w-5 h-5 mr-2"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                  clipRule="evenodd"
                />
              </svg>
              {successMessage}
            </div>
          )}

          {errorMessage && (
            <div className="mb-6 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg flex items-center">
              <svg
                className="w-5 h-5 mr-2"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
              {errorMessage}
            </div>
          )}

          {/* Loading State */}
          {loading ? (
            <div className="bg-white rounded-lg shadow-sm p-8 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              <p className="mt-4 text-gray-600">Loading configuration...</p>
            </div>
          ) : (
            <>
              {/* Configuration Status */}
              <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">
                      Configuration Status
                    </h3>
                    <p className="text-sm text-gray-600 mt-1">
                      {config.is_configured
                        ? 'Midtrans is configured and ready to process payments'
                        : 'Midtrans is not configured yet'}
                    </p>
                  </div>
                  <div>
                    {config.is_configured ? (
                      <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
                        <span className="w-2 h-2 mr-2 bg-green-500 rounded-full"></span>
                        Configured
                      </span>
                    ) : (
                      <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800">
                        <span className="w-2 h-2 mr-2 bg-yellow-500 rounded-full"></span>
                        Not Configured
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Configuration Form */}
              <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm p-6">
                <div className="space-y-6">
                  {/* Environment Selection */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Environment
                    </label>
                    <select
                      value={config.environment}
                      onChange={(e) =>
                        handleInputChange(
                          'environment',
                          e.target.value as 'sandbox' | 'production'
                        )
                      }
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    >
                      <option value="sandbox">Sandbox (Testing)</option>
                      <option value="production">Production (Live)</option>
                    </select>
                    <p className="mt-1 text-sm text-gray-500">
                      Use Sandbox for testing, Production for live transactions
                    </p>
                  </div>

                  {/* Server Key */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Server Key <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <input
                        type={showServerKey ? 'text' : 'password'}
                        value={config.server_key}
                        onChange={(e) =>
                          handleInputChange('server_key', e.target.value)
                        }
                        placeholder="SB-Mid-server-xxxxxxxxxxxxx"
                        required
                        className="w-full px-4 py-2 pr-12 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      />
                      <button
                        type="button"
                        onClick={() => setShowServerKey(!showServerKey)}
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-500 hover:text-gray-700"
                      >
                        {showServerKey ? (
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                          </svg>
                        ) : (
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                          </svg>
                        )}
                      </button>
                    </div>
                    <p className="mt-1 text-sm text-gray-500">
                      Your Midtrans server key (kept secure, not visible to clients)
                    </p>
                  </div>

                  {/* Client Key */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Client Key <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <input
                        type={showClientKey ? 'text' : 'password'}
                        value={config.client_key}
                        onChange={(e) =>
                          handleInputChange('client_key', e.target.value)
                        }
                        placeholder="SB-Mid-client-xxxxxxxxxxxxx"
                        required
                        className="w-full px-4 py-2 pr-12 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      />
                      <button
                        type="button"
                        onClick={() => setShowClientKey(!showClientKey)}
                        className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-500 hover:text-gray-700"
                      >
                        {showClientKey ? (
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                          </svg>
                        ) : (
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                          </svg>
                        )}
                      </button>
                    </div>
                    <p className="mt-1 text-sm text-gray-500">
                      Your Midtrans client key (used in frontend applications)
                    </p>
                  </div>

                  {/* Merchant ID */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Merchant ID
                    </label>
                    <input
                      type="text"
                      value={config.merchant_id}
                      onChange={(e) =>
                        handleInputChange('merchant_id', e.target.value)
                      }
                      placeholder="M999999"
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                    <p className="mt-1 text-sm text-gray-500">
                      Optional: Your Midtrans merchant ID
                    </p>
                  </div>

                  {/* Help Text */}
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                    <div className="flex">
                      <svg
                        className="w-5 h-5 text-blue-600 mt-0.5"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                      >
                        <path
                          fillRule="evenodd"
                          d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                          clipRule="evenodd"
                        />
                      </svg>
                      <div className="ml-3">
                        <h4 className="text-sm font-medium text-blue-900">
                          How to get your Midtrans credentials:
                        </h4>
                        <ol className="mt-2 text-sm text-blue-800 list-decimal list-inside space-y-1">
                          <li>Sign up at <a href="https://dashboard.midtrans.com" target="_blank" rel="noopener noreferrer" className="underline">dashboard.midtrans.com</a></li>
                          <li>Go to Settings → Access Keys</li>
                          <li>Copy your Server Key and Client Key</li>
                          <li>Start with Sandbox for testing, switch to Production when ready</li>
                        </ol>
                      </div>
                    </div>
                  </div>

                  {/* Action Buttons */}
                  <div className="flex justify-end space-x-4 pt-4 border-t">
                    <button
                      type="button"
                      onClick={() => router.push('/settings')}
                      disabled={saving}
                      className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={saving}
                      className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center"
                    >
                      {saving ? (
                        <>
                          <svg
                            className="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
                            fill="none"
                            viewBox="0 0 24 24"
                          >
                            <circle
                              className="opacity-25"
                              cx="12"
                              cy="12"
                              r="10"
                              stroke="currentColor"
                              strokeWidth="4"
                            ></circle>
                            <path
                              className="opacity-75"
                              fill="currentColor"
                              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                            ></path>
                          </svg>
                          Saving...
                        </>
                      ) : (
                        'Save Configuration'
                      )}
                    </button>
                  </div>
                </div>
              </form>
            </>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
