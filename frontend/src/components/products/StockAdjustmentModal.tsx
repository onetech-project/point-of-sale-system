'use client';

import React, { useState } from 'react';
import { useTranslation } from '@/i18n/provider';
import { StockAdjustmentReason, StockAdjustmentRequest } from '@/types/product';

interface StockAdjustmentModalProps {
  productId: string;
  productName: string;
  currentStock: number;
  onClose: () => void;
  onSubmit: (data: StockAdjustmentRequest) => Promise<void>;
}

const StockAdjustmentModal: React.FC<StockAdjustmentModalProps> = ({
  productId,
  productName,
  currentStock,
  onClose,
  onSubmit,
}) => {
  const { t } = useTranslation(['products', 'common']);
  const [formData, setFormData] = useState({
    new_quantity: currentStock.toString(),
    reason: 'physical_count' as StockAdjustmentReason,
    notes: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const newQuantity = parseInt(formData.new_quantity);
    if (isNaN(newQuantity) || newQuantity < 0) {
      setError(t('products.validation.stockInvalid'));
      return;
    }

    setIsSubmitting(true);
    try {
      await onSubmit({
        new_quantity: newQuantity,
        reason: formData.reason,
        notes: formData.notes || undefined,
      });
      onClose();
    } catch (err: any) {
      setError(err.response?.data?.message || t('products.messages.stockAdjustError'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const quantityDelta = parseInt(formData.new_quantity) - currentStock;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex min-h-full items-center justify-center p-4">
        <div
          className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
          onClick={onClose}
        ></div>

        <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full">
          <div className="p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">
                {t('products.actions.adjustStock')}
              </h3>
              <button onClick={onClose} className="text-gray-400 hover:text-gray-500">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            <div className="mb-4">
              <p className="text-sm text-gray-600">{productName}</p>
              <p className="text-sm text-gray-500">
                {t('products.details.currentStock')}:{' '}
                <span className="font-medium">{currentStock}</span>{' '}
                {t('common.units', { ns: 'common' })}
              </p>
            </div>

            {error && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
                <p className="text-sm text-red-600">{error}</p>
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label
                  htmlFor="new_quantity"
                  className="block text-sm font-medium text-gray-700 mb-1"
                >
                  {t('products.inventory.newQuantity')} *
                </label>
                <input
                  type="number"
                  id="new_quantity"
                  value={formData.new_quantity}
                  onChange={e => setFormData({ ...formData, new_quantity: e.target.value })}
                  min="0"
                  required
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
                {!isNaN(quantityDelta) && (
                  <p
                    className={`mt-1 text-sm ${quantityDelta >= 0 ? 'text-green-600' : 'text-red-600'}`}
                  >
                    {t('products.inventory.adjustment')}: {quantityDelta >= 0 ? '+' : ''}
                    {quantityDelta} {t('common.units', { ns: 'common' })}
                  </p>
                )}
              </div>

              <div>
                <label htmlFor="reason" className="block text-sm font-medium text-gray-700 mb-1">
                  {t('products.inventory.reason')} *
                </label>
                <select
                  id="reason"
                  value={formData.reason}
                  onChange={e =>
                    setFormData({ ...formData, reason: e.target.value as StockAdjustmentReason })
                  }
                  required
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="supplier_delivery">
                    {t('products.inventory.adjustmentReasons.supplier_delivery')}
                  </option>
                  <option value="physical_count">
                    {t('products.inventory.adjustmentReasons.physical_count')}
                  </option>
                  <option value="shrinkage">
                    {t('products.inventory.adjustmentReasons.shrinkage')}
                  </option>
                  <option value="damage">{t('products.inventory.adjustmentReasons.damage')}</option>
                  <option value="return">{t('products.inventory.adjustmentReasons.return')}</option>
                  <option value="correction">
                    {t('products.inventory.adjustmentReasons.correction')}
                  </option>
                </select>
              </div>

              <div>
                <label htmlFor="notes" className="block text-sm font-medium text-gray-700 mb-1">
                  {t('common.notes', { ns: 'common' })}
                </label>
                <textarea
                  id="notes"
                  value={formData.notes}
                  onChange={e => setFormData({ ...formData, notes: e.target.value })}
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  placeholder={t('common.notesPlaceholder', { ns: 'common' })}
                />
              </div>

              <div className="flex space-x-3 pt-4">
                <button
                  type="button"
                  onClick={onClose}
                  disabled={isSubmitting}
                  className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
                >
                  {t('common.cancel', { ns: 'common' })}
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="flex-1 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:opacity-50"
                >
                  {isSubmitting
                    ? t('common.saving', { ns: 'common' })
                    : t('common.save', { ns: 'common' })}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StockAdjustmentModal;
