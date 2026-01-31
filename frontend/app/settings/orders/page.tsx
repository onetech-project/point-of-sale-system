'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import axios from 'axios';
import { formatCurrency } from '../../../src/utils/format';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface OrderSettings {
  delivery_enabled: boolean;
  pickup_enabled: boolean;
  dine_in_enabled: boolean;
  default_delivery_fee: number;
  min_order_amount: number;
  max_delivery_distance: number;
  estimated_prep_time: number;
  auto_accept_orders: boolean;
  require_phone_verification: boolean;
  charge_delivery_fee: boolean;
}

/**
 * Order Settings Page
 * Configure order-related settings like delivery types, fees, and order policies
 */
export default function OrderSettingsPage() {
  const router = useRouter();
  const { user } = useAuth();
  const tenantId = user?.tenantId;

  const [settings, setSettings] = useState<OrderSettings>({
    delivery_enabled: true,
    pickup_enabled: true,
    dine_in_enabled: false,
    default_delivery_fee: 10000,
    min_order_amount: 20000,
    max_delivery_distance: 10,
    estimated_prep_time: 30,
    auto_accept_orders: false,
    require_phone_verification: false,
    charge_delivery_fee: true,
  });

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    if (tenantId) {
      fetchSettings();
    }
  }, [tenantId]);

  const fetchSettings = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await axios.get(
        `${API_BASE_URL}/api/v1/admin/settings/orders?tenant_id=${tenantId}`,
        { withCredentials: true }
      );

      setSettings(response.data);
    } catch (err) {
      console.error('Failed to fetch order settings:', err);
      setError('Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      setError(null);
      setSuccess(false);

      await axios.put(
        `${API_BASE_URL}/api/v1/admin/settings/orders?tenant_id=${tenantId}`,
        settings,
        { withCredentials: true }
      );

      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      console.error('Failed to save order settings:', err);
      setError('Failed to save settings. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="space-y-6 max-w-4xl">
          {/* Page Header */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-3xl font-bold text-gray-900">Order Settings</h1>
                <p className="text-gray-600 mt-2">
                  Configure delivery types, fees, and order policies
                </p>
              </div>
              <button
                onClick={() => router.push('/settings')}
                className="flex items-center gap-2 px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
                </svg>
                Back to Settings
              </button>
            </div>
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4">
              <p className="text-red-800">{error}</p>
            </div>
          )}

          {success && (
            <div className="bg-green-50 border border-green-200 rounded-lg p-4">
              <p className="text-green-800">Settings saved successfully!</p>
            </div>
          )}

          {/* Delivery Type Settings */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Delivery Types</h2>
            <p className="text-sm text-gray-600 mb-6">
              Enable or disable different order fulfillment methods
            </p>

            <div className="space-y-4">
              <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
                <div className="flex-1">
                  <div className="font-medium text-gray-900">Delivery</div>
                  <div className="text-sm text-gray-600">Allow customers to order for delivery</div>
                </div>
                <input
                  type="checkbox"
                  checked={settings.delivery_enabled}
                  onChange={(e) => setSettings({ ...settings, delivery_enabled: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              </label>

              <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
                <div className="flex-1">
                  <div className="font-medium text-gray-900">Pickup / Takeaway</div>
                  <div className="text-sm text-gray-600">Allow customers to pick up orders</div>
                </div>
                <input
                  type="checkbox"
                  checked={settings.pickup_enabled}
                  onChange={(e) => setSettings({ ...settings, pickup_enabled: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              </label>

              <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
                <div className="flex-1">
                  <div className="font-medium text-gray-900">Dine In</div>
                  <div className="text-sm text-gray-600">Allow customers to order for dine-in</div>
                </div>
                <input
                  type="checkbox"
                  checked={settings.dine_in_enabled}
                  onChange={(e) => setSettings({ ...settings, dine_in_enabled: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              </label>
            </div>
          </div>

          {/* Pricing Settings */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Pricing & Fees</h2>
            <p className="text-sm text-gray-600 mb-6">
              Configure delivery fees and minimum order amounts
            </p>

            <div className="space-y-6">
              <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
                <div className="flex-1">
                  <div className="font-medium text-gray-900">Charge Delivery Fee</div>
                  <div className="text-sm text-gray-600">
                    Collect delivery fees through the system. Disable if you handle delivery fees externally (e.g., third-party delivery services)
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={settings.charge_delivery_fee}
                  onChange={(e) => setSettings({ ...settings, charge_delivery_fee: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              </label>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Default Delivery Fee
                </label>
                <div className="flex items-center gap-4">
                  <input
                    type="number"
                    value={settings.default_delivery_fee}
                    onChange={(e) => setSettings({ ...settings, default_delivery_fee: parseInt(e.target.value) || 0 })}
                    className="flex-1 px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                    min="0"
                    step="1000"
                    disabled={!settings.charge_delivery_fee}
                  />
                  <span className="text-sm text-gray-600 min-w-[120px]">
                    {formatCurrency(settings.default_delivery_fee)}
                  </span>
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  {settings.charge_delivery_fee
                    ? "Base delivery fee charged to customers"
                    : "Disabled - delivery fees handled externally"}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Minimum Order Amount
                </label>
                <div className="flex items-center gap-4">
                  <input
                    type="number"
                    value={settings.min_order_amount}
                    onChange={(e) => setSettings({ ...settings, min_order_amount: parseInt(e.target.value) || 0 })}
                    className="flex-1 px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
                    min="0"
                    step="5000"
                  />
                  <span className="text-sm text-gray-600 min-w-[120px]">
                    {formatCurrency(settings.min_order_amount)}
                  </span>
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  Minimum subtotal required to place an order
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Maximum Delivery Distance (km)
                </label>
                <input
                  type="number"
                  value={settings.max_delivery_distance}
                  onChange={(e) => setSettings({ ...settings, max_delivery_distance: parseInt(e.target.value) || 0 })}
                  className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
                  min="0"
                  step="1"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Maximum distance for delivery orders
                </p>
              </div>
            </div>
          </div>

          {/* Order Processing Settings */}
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Order Processing</h2>
            <p className="text-sm text-gray-600 mb-6">
              Configure how orders are processed and handled
            </p>

            <div className="space-y-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Estimated Preparation Time (minutes)
                </label>
                <input
                  type="number"
                  value={settings.estimated_prep_time}
                  onChange={(e) => setSettings({ ...settings, estimated_prep_time: parseInt(e.target.value) || 0 })}
                  className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
                  min="0"
                  step="5"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Average time to prepare an order
                </p>
              </div>

              <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
                <div className="flex-1">
                  <div className="font-medium text-gray-900">Auto-Accept Orders</div>
                  <div className="text-sm text-gray-600">
                    Automatically accept new orders without manual confirmation
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={settings.auto_accept_orders}
                  onChange={(e) => setSettings({ ...settings, auto_accept_orders: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              </label>

              <label className="flex items-center justify-between p-4 border rounded-lg hover:bg-gray-50">
                <div className="flex-1">
                  <div className="font-medium text-gray-900">Require Phone Verification</div>
                  <div className="text-sm text-gray-600">
                    Send SMS verification code to customer phone numbers
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={settings.require_phone_verification}
                  onChange={(e) => setSettings({ ...settings, require_phone_verification: e.target.checked })}
                  className="w-5 h-5 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                />
              </label>
            </div>
          </div>

          {/* Save Button */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-600">
                Changes will apply to new orders immediately
              </p>
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors"
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </div>
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
