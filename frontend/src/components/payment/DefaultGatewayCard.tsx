'use client';

import React, { useEffect, useState } from 'react';
import paymentService, { PaymentGatewayName, PaymentGatewaySettings } from '@/services/payment';

interface DefaultGatewayCardProps {
  tenantId: string;
  /** Bump this to reload the gateway status, e.g. after credentials are saved */
  refreshKey?: number;
}

const GATEWAYS: { value: PaymentGatewayName; label: string; description: string }[] = [
  { value: 'midtrans', label: 'Midtrans', description: 'QRIS via Midtrans Core API' },
  { value: 'manjo', label: 'Manjo', description: 'QRIS MPM via Manjo' },
];

/**
 * Lets the owner pick which configured gateway generates guest QRIS payments.
 */
export default function DefaultGatewayCard({ tenantId, refreshKey = 0 }: DefaultGatewayCardProps) {
  const [settings, setSettings] = useState<PaymentGatewaySettings | null>(null);
  const [selected, setSelected] = useState<PaymentGatewayName>('midtrans');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    let mounted = true;
    setLoading(true);
    paymentService
      .getPaymentGateway(tenantId)
      .then(data => {
        if (!mounted) return;
        setSettings(data);
        setSelected(data.default_gateway);
      })
      .catch(error => {
        console.error('Failed to fetch payment gateway settings:', error);
        if (mounted) setMessage({ type: 'error', text: 'Failed to load payment gateway settings' });
      })
      .finally(() => {
        if (mounted) setLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, [tenantId, refreshKey]);

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);
    try {
      await paymentService.updateDefaultGateway(tenantId, selected);
      setSettings(prev => (prev ? { ...prev, default_gateway: selected } : prev));
      setMessage({ type: 'success', text: 'Default payment gateway updated' });
    } catch (error: any) {
      console.error('Failed to update default gateway:', error);
      setMessage({
        type: 'error',
        text: error.response?.data?.error || 'Failed to update default payment gateway',
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
      <h3 className="text-lg font-semibold text-gray-900">Default Payment Gateway</h3>
      <p className="text-sm text-gray-600 mt-1">
        New guest orders generate their QRIS with this gateway. Only configured gateways can be
        selected.
      </p>

      {message && (
        <div
          className={`mt-4 px-4 py-3 rounded-lg border text-sm ${
            message.type === 'success'
              ? 'bg-green-50 border-green-200 text-green-800'
              : 'bg-red-50 border-red-200 text-red-800'
          }`}
        >
          {message.text}
        </div>
      )}

      {loading ? (
        <p className="mt-4 text-sm text-gray-500">Loading...</p>
      ) : (
        <>
          <div className="mt-4 grid gap-3 sm:grid-cols-2" role="radiogroup">
            {GATEWAYS.map(gateway => {
              const status = settings?.gateways?.[gateway.value];
              const configured = !!status?.configured;
              const isSelected = selected === gateway.value;
              return (
                <label
                  key={gateway.value}
                  className={`flex items-start gap-3 rounded-lg border p-4 ${
                    configured ? 'cursor-pointer' : 'cursor-not-allowed opacity-60'
                  } ${isSelected ? 'border-blue-500 ring-1 ring-blue-500' : 'border-gray-200'}`}
                >
                  <input
                    type="radio"
                    name="default-gateway"
                    value={gateway.value}
                    checked={isSelected}
                    disabled={!configured}
                    onChange={() => setSelected(gateway.value)}
                    className="mt-1"
                  />
                  <span className="min-w-0 flex-1">
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-gray-900">{gateway.label}</span>
                      {configured ? (
                        <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800">
                          Configured ({status?.environment})
                        </span>
                      ) : (
                        <span className="rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-800">
                          Not configured
                        </span>
                      )}
                      {settings?.default_gateway === gateway.value && (
                        <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800">
                          Current default
                        </span>
                      )}
                    </span>
                    <span className="mt-1 block text-sm text-gray-500">{gateway.description}</span>
                  </span>
                </label>
              );
            })}
          </div>
          <div className="mt-4 flex justify-end">
            <button
              type="button"
              onClick={handleSave}
              disabled={saving || !settings || selected === settings.default_gateway}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {saving ? 'Saving...' : 'Set as Default'}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
