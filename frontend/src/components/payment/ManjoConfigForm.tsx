'use client';

import React, { useEffect, useState } from 'react';
import paymentService, { ManjoConfig, PaymentEnvironment } from '@/services/payment';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
// Manjo stores this base URL and appends /v1.0/access-token/b2b and
// /v1.0/qr/qr-mpm-notify itself.
const CALLBACK_BASE_URL = `${API_BASE_URL}/api/v1/webhooks/payments/manjo`;

interface ManjoConfigFormProps {
  tenantId: string;
  onSaved?: () => void;
  onCancel?: () => void;
}

const inputClassName =
  'w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent';

/**
 * Manjo QRIS credentials form.
 * The client secret is write-only: the API reports whether one is stored, and
 * leaving the field blank keeps the stored value. The RSA key pair is the
 * platform's, so tenants never enter a private key.
 */
export default function ManjoConfigForm({ tenantId, onSaved, onCancel }: ManjoConfigFormProps) {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<ManjoConfig | null>(null);
  const [clientId, setClientId] = useState('');
  const [merchantId, setMerchantId] = useState('');
  const [partnerId, setPartnerId] = useState('');
  const [environment, setEnvironment] = useState<PaymentEnvironment>('sandbox');
  const [clientSecret, setClientSecret] = useState('');
  const [showClientSecret, setShowClientSecret] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');

  const fetchConfig = async () => {
    try {
      setLoading(true);
      const data = await paymentService.getManjoConfig(tenantId);
      setConfig(data);
      setClientId(data.client_id || '');
      setMerchantId(data.merchant_id || '');
      setPartnerId(data.partner_id || '');
      setEnvironment(data.environment || 'sandbox');
      setClientSecret('');
    } catch (error) {
      console.error('Failed to fetch Manjo config:', error);
      setErrorMessage('Failed to load Manjo configuration');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setSuccessMessage('');
    setErrorMessage('');

    try {
      await paymentService.updateManjoConfig(tenantId, {
        client_id: clientId.trim(),
        merchant_id: merchantId.trim(),
        partner_id: partnerId.trim(),
        environment,
        ...(clientSecret.trim() ? { client_secret: clientSecret.trim() } : {}),
      });
      setSuccessMessage('Manjo configuration saved successfully');
      setTimeout(() => setSuccessMessage(''), 5000);
      await fetchConfig();
      onSaved?.();
    } catch (error: any) {
      console.error('Failed to save Manjo config:', error);
      setErrorMessage(error.response?.data?.error || 'Failed to save Manjo configuration');
      setTimeout(() => setErrorMessage(''), 5000);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-sm p-8 text-center">
        <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="mt-4 text-gray-600">Loading configuration...</p>
      </div>
    );
  }

  const hasClientSecret = !!config?.has_client_secret;

  return (
    <>
      {successMessage && (
        <div className="mb-6 bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg">
          {successMessage}
        </div>
      )}
      {errorMessage && (
        <div className="mb-6 bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
          {errorMessage}
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">Configuration Status</h3>
            <p className="text-sm text-gray-600 mt-1">
              {config?.is_configured
                ? 'Manjo is configured and ready to process payments'
                : 'Manjo is not configured yet'}
            </p>
          </div>
          {config?.is_configured ? (
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

      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm p-6">
        <div className="space-y-6">
          <div>
            <label
              htmlFor="manjo-environment"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Environment
            </label>
            <select
              id="manjo-environment"
              value={environment}
              onChange={e => setEnvironment(e.target.value as PaymentEnvironment)}
              className={inputClassName}
            >
              <option value="sandbox">Sandbox (Testing)</option>
              <option value="production">Production (Live)</option>
            </select>
            <p className="mt-1 text-sm text-gray-500">
              Use Sandbox for testing, Production for live transactions
            </p>
          </div>

          <div>
            <label
              htmlFor="manjo-client-id"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Client ID <span className="text-red-500">*</span>
            </label>
            <input
              id="manjo-client-id"
              type="text"
              value={clientId}
              onChange={e => setClientId(e.target.value)}
              placeholder="EP9613058999"
              required
              className={inputClassName}
            />
            <p className="mt-1 text-sm text-gray-500">
              The Client ID (X-CLIENT-KEY) issued by Manjo
            </p>
          </div>

          <div>
            <label
              htmlFor="manjo-merchant-id"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Merchant ID <span className="text-red-500">*</span>
            </label>
            <input
              id="manjo-merchant-id"
              type="text"
              value={merchantId}
              onChange={e => setMerchantId(e.target.value)}
              placeholder="MT60169117"
              required
              className={inputClassName}
            />
          </div>

          <div>
            <label
              htmlFor="manjo-partner-id"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Partner ID
            </label>
            <input
              id="manjo-partner-id"
              type="text"
              value={partnerId}
              onChange={e => setPartnerId(e.target.value)}
              placeholder="Same as Client ID"
              className={inputClassName}
            />
            <p className="mt-1 text-sm text-gray-500">
              X-PARTNER-ID provided by Manjo; leave blank to use Client ID
            </p>
          </div>

          <div>
            <label
              htmlFor="manjo-client-secret"
              className="block text-sm font-medium text-gray-700 mb-2"
            >
              Client Secret {!hasClientSecret && <span className="text-red-500">*</span>}
            </label>
            <div className="relative">
              <input
                id="manjo-client-secret"
                type={showClientSecret ? 'text' : 'password'}
                value={clientSecret}
                onChange={e => setClientSecret(e.target.value)}
                placeholder={
                  hasClientSecret ? 'Leave blank to keep current' : 'Client secret from Manjo'
                }
                required={!hasClientSecret}
                autoComplete="new-password"
                className={`${inputClassName} pr-16`}
              />
              <button
                type="button"
                onClick={() => setShowClientSecret(!showClientSecret)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-500 hover:text-gray-700"
              >
                {showClientSecret ? 'Hide' : 'Show'}
              </button>
            </div>
            <p className="mt-1 text-sm text-gray-500">
              Used to sign QRIS requests and verify Manjo notifications (HMAC-SHA512)
            </p>
          </div>

          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <h4 className="text-sm font-medium text-blue-900">Setting up Manjo:</h4>
            <ol className="mt-2 text-sm text-blue-800 list-decimal list-inside space-y-1">
              <li>Get your Client ID, Client Secret and Merchant ID from Manjo</li>
              <li>
                Give Manjo this callback base URL (Manjo adds the access token and
                payment notification paths itself):
                <code className="block mt-1 break-all rounded bg-white px-2 py-1 text-xs text-blue-900">
                  {CALLBACK_BASE_URL}
                </code>
              </li>
              <li>Start with Sandbox for testing, switch to Production when ready</li>
            </ol>
          </div>

          <div className="flex justify-end space-x-4 pt-4 border-t">
            {onCancel && (
              <button
                type="button"
                onClick={onCancel}
                disabled={saving}
                className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
            )}
            <button
              type="submit"
              disabled={saving}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {saving ? 'Saving...' : 'Save Configuration'}
            </button>
          </div>
        </div>
      </form>
    </>
  );
}
