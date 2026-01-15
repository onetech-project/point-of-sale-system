import React, { useState, useEffect } from 'react';
import { tenant } from '../../services/tenant';
import DeliveryTypeSelector from './DeliveryTypeSelector';
import AddressInput from './AddressInput';
import ConsentPurposeList from '../consent/ConsentPurposeList';
import { useTranslation } from 'react-i18next';
import { CheckoutData, TenantConfig } from '../../types/checkout';
import { formatPrice } from '../../utils/format';

interface CheckoutFormProps {
  tenantId: string;
  cartTotal: number;
  onSubmit: (data: CheckoutData) => void;
  loading?: boolean;
  estimatedDeliveryFee?: number; // T084: Add delivery fee prop
}

export const CheckoutForm: React.FC<CheckoutFormProps> = ({
  tenantId,
  cartTotal,
  onSubmit,
  loading = false,
  estimatedDeliveryFee = 0, // T084: Default to 0
}) => {
  const { t } = useTranslation(['common', 'consent']);
  const [tenantConfig, setTenantConfig] = useState<TenantConfig | null>(null);
  const [configLoading, setConfigLoading] = useState(true);
  const [formData, setFormData] = useState<CheckoutData>({
    delivery_type: '',
    customer_name: '',
    customer_phone: '',
    customer_email: '',
    delivery_address: '',
    table_number: '',
    notes: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [consents, setConsents] = useState<{ [key: string]: boolean }>({});
  const [consentError, setConsentError] = useState<string>('');

  useEffect(() => {
    fetchTenantConfig();
  }, [tenantId]);

  const fetchTenantConfig = async () => {
    try {
      setConfigLoading(true);
      const response = await tenant.getTenantConfig(tenantId);
      setTenantConfig(response);

      // Set default delivery type if only one is available
      if (response.enabled_delivery_types.length === 1) {
        setFormData(prev => ({
          ...prev,
          delivery_type: response.enabled_delivery_types[0],
        }));
      }
    } catch (error) {
      console.error('Failed to fetch tenant config:', error);
      // Set default config
      setTenantConfig({
        tenant_id: tenantId,
        enabled_delivery_types: ['pickup', 'delivery', 'dine_in'],
        auto_calculate_fees: false,
      });
    } finally {
      setConfigLoading(false);
    }
  };

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    // Validate delivery type
    if (!formData.delivery_type) {
      newErrors.delivery_type = t('common.checkout.form.selectDeliveryType');
    }

    // Validate customer name
    if (!formData.customer_name.trim()) {
      newErrors.customer_name = t('common.checkout.form.nameRequired');
    } else if (formData.customer_name.trim().length < 2) {
      newErrors.customer_name = t('common.checkout.form.nameMinLength');
    }

    // Validate phone number
    const phoneRegex = /^(\+62|62|0)[0-9]{9,12}$/;
    if (!formData.customer_phone.trim()) {
      newErrors.customer_phone = t('common.checkout.form.phoneRequired');
    } else if (!phoneRegex.test(formData.customer_phone.replace(/[\s-]/g, ''))) {
      newErrors.customer_phone = t('common.checkout.form.phoneInvalid');
    }

    // Validate email (optional but must be valid if provided)
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (formData.customer_email && formData.customer_email.trim() && !emailRegex.test(formData.customer_email.trim())) {
      newErrors.customer_email = t('common.checkout.form.emailInvalid');
    }

    // Validate delivery address for delivery orders
    if (formData.delivery_type === 'delivery') {
      if (!formData.delivery_address?.trim()) {
        newErrors.delivery_address = t('common.checkout.form.addressRequired');
      } else if (formData.delivery_address.trim().length < 10) {
        newErrors.delivery_address = t('common.checkout.form.addressMinLength');
      }
    }

    // Validate required consents (order_processing and payment_processing_midtrans)
    const requiredConsentCodes = ['order_processing', 'payment_processing_midtrans'];
    const hasAllRequiredConsents = requiredConsentCodes.every(code => consents[code] === true);
    if (!hasAllRequiredConsents) {
      setConsentError(t('consent_error_required', { ns: 'consent' }));
      return false;
    }

    setErrors(newErrors);
    setConsentError('');
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    // Get granted optional consents only (required consents are implicit on backend)
    const grantedOptionalConsents = Object.entries(consents)
      .filter(([key, granted]) => {
        // Only include optional consents that were granted
        // Required consents (order_processing, payment_processing_midtrans) are NOT sent (backend enforces)
        const isOptional = !['order_processing', 'payment_processing_midtrans'].includes(key);
        return isOptional && granted;
      })
      .map(([purpose_code]) => purpose_code); // Array of consent codes only

    // Clean up data based on delivery type
    const submitData: CheckoutData = {
      delivery_type: formData.delivery_type,
      customer_name: formData.customer_name.trim(),
      customer_phone: formData.customer_phone.replace(/[\s-]/g, ''),
      customer_email: formData.customer_email?.trim() || undefined,
      notes: formData.notes?.trim() || undefined,
      consents: grantedOptionalConsents, // Simplified payload - only optional consent codes
    };

    if (formData.delivery_type === 'delivery') {
      submitData.delivery_address = formData.delivery_address?.trim();
    } else if (formData.delivery_type === 'dine_in') {
      submitData.table_number = formData.table_number?.trim();
    }

    onSubmit(submitData);
  };

  if (configLoading) {
    return (
      <div className="flex justify-center items-center p-8">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Delivery Type Selection */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <DeliveryTypeSelector
          enabledTypes={tenantConfig?.enabled_delivery_types || []}
          selectedType={formData.delivery_type}
          onChange={(type) => {
            setFormData({ ...formData, delivery_type: type });
            setErrors({ ...errors, delivery_type: '' });
          }}
        />
        {errors.delivery_type && (
          <p className="mt-2 text-sm text-red-600">{errors.delivery_type}</p>
        )}
      </div>

      {/* Contact Information */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-lg font-semibold mb-4">{t('common.checkout.form.contactInfo')}</h2>

        <div className="space-y-4">
          {/* Customer Name */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('common.checkout.form.yourName')} <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formData.customer_name}
              onChange={(e) => {
                setFormData({ ...formData, customer_name: e.target.value });
                setErrors({ ...errors, customer_name: '' });
              }}
              className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${errors.customer_name ? 'border-red-500' : 'border-gray-300'
                }`}
              placeholder={t('common.checkout.form.namePlaceholder')}
            />
            {errors.customer_name && (
              <p className="mt-1 text-sm text-red-600">{errors.customer_name}</p>
            )}
          </div>

          {/* Customer Phone */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('common.checkout.form.phoneNumber')} <span className="text-red-500">*</span>
            </label>
            <input
              type="tel"
              value={formData.customer_phone}
              onChange={(e) => {
                setFormData({ ...formData, customer_phone: e.target.value });
                setErrors({ ...errors, customer_phone: '' });
              }}
              className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${errors.customer_phone ? 'border-red-500' : 'border-gray-300'
                }`}
              placeholder="081234567890"
            />
            {errors.customer_phone && (
              <p className="mt-1 text-sm text-red-600">{errors.customer_phone}</p>
            )}
          </div>

          {/* Customer Email (Optional) */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('common.checkout.form.email')} <span className="text-gray-500 text-xs">(optional for invoice)</span>
            </label>
            <input
              type="email"
              value={formData.customer_email || ''}
              onChange={(e) => {
                setFormData({ ...formData, customer_email: e.target.value });
                setErrors({ ...errors, customer_email: '' });
              }}
              className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${errors.customer_email ? 'border-red-500' : 'border-gray-300'
                }`}
              placeholder="customer@example.com"
            />
            {errors.customer_email && (
              <p className="mt-1 text-sm text-red-600">{errors.customer_email}</p>
            )}
            <p className="mt-1 text-xs text-gray-500">
              {t('common.checkout.form.emailHint', 'We\'ll send your invoice to this email')}
            </p>
          </div>
        </div>
      </div>

      {/* Conditional Fields Based on Delivery Type */}
      {formData.delivery_type === 'delivery' && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-lg font-semibold mb-4">{t('common.checkout.form.deliveryAddress')}</h2>
          <AddressInput
            value={formData.delivery_address || ''}
            onChange={(value) => {
              setFormData({ ...formData, delivery_address: value });
              setErrors({ ...errors, delivery_address: '' });
            }}
            error={errors.delivery_address}
          />
        </div>
      )}

      {formData.delivery_type === 'dine_in' && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-lg font-semibold mb-4">{t('common.checkout.form.tableInfo')}</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {t('common.checkout.form.tableNumber')}
            </label>
            <input
              type="text"
              value={formData.table_number || ''}
              onChange={(e) => setFormData({ ...formData, table_number: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder={t('common.checkout.form.tablePlaceholder')}
            />
            <p className="mt-1 text-xs text-gray-500">
              {t('common.checkout.form.tableHint')}
            </p>
          </div>
        </div>
      )}

      {/* Additional Notes */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-lg font-semibold mb-4">{t('common.checkout.form.additionalNotes')}</h2>
        <textarea
          value={formData.notes || ''}
          onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
          rows={3}
          className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
          placeholder={t('common.checkout.form.notesPlaceholder')}
        />
      </div>

      {/* Order Summary - T084: Enhanced with delivery fee display */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-lg font-semibold mb-4">{t('common.checkout.form.orderSummary')}</h2>
        <div className="space-y-3">
          {/* Subtotal */}
          <div className="flex justify-between text-gray-700">
            <span>{t('common.checkout.form.subtotal')}</span>
            <span>{formatPrice(cartTotal)}</span>
          </div>

          {/* Delivery Fee */}
          {formData.delivery_type === 'delivery' && (
            <div className="flex justify-between text-gray-700">
              <div className="flex items-center gap-1">
                <span>{t('common.checkout.form.deliveryFee')}</span>
                {tenantConfig?.auto_calculate_fees && (
                  <span className="text-xs text-gray-500">
                    (calculated)
                  </span>
                )}
              </div>
              {estimatedDeliveryFee > 0 ? (
                <span className="font-medium">
                  {formatPrice(estimatedDeliveryFee)}
                </span>
              ) : (
                <span className="text-gray-500 text-sm">
                  {tenantConfig?.auto_calculate_fees
                    ? 'Calculated at checkout'
                    : 'Free'}
                </span>
              )}
            </div>
          )}

          {/* Total */}
          <div className="border-t pt-3 flex justify-between items-center">
            <span className="text-lg font-semibold">{t('common.cart.total')}</span>
            <div className="text-right">
              <div className="text-2xl font-bold text-blue-600">
                {formatPrice(
                  cartTotal +
                  (formData.delivery_type === 'delivery'
                    ? estimatedDeliveryFee
                    : 0)
                )}
              </div>
              {formData.delivery_type === 'delivery' &&
                estimatedDeliveryFee === 0 &&
                tenantConfig?.auto_calculate_fees && (
                  <p className="text-xs text-gray-500 mt-1">
                    Final total will be calculated after address validation
                  </p>
                )}
            </div>
          </div>
        </div>
      </div>

      {/* Consent Collection Section */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-lg font-semibold mb-4">
          {t('common.checkout.form.dataConsent', 'Data Privacy Consent')}
        </h2>
        <ConsentPurposeList
          context="guest"
          onConsentChange={(newConsents) => {
            setConsents(newConsents);
            if (consentError) setConsentError('');
          }}
          initialConsents={consents}
          showError={!!consentError}
          errorMessage={consentError}
        />
      </div>

      {/* Submit Button */}
      <button
        type="submit"
        disabled={loading}
        className={`w-full py-4 rounded-lg font-semibold text-white transition-colors ${loading
          ? 'bg-gray-400 cursor-not-allowed'
          : 'bg-blue-600 hover:bg-blue-700'
          }`}
      >
        {loading ? t('common.checkout.form.processing') : t('common.checkout.form.proceedToPayment')}
      </button>
    </form>
  );
};

export default CheckoutForm;
