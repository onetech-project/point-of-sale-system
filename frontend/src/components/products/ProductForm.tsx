'use client';

import React, { useState, useEffect } from 'react';
import { useTranslation } from '@/i18n/provider';
import CategorySelect from './CategorySelect';
import { CreateProductRequest, UpdateProductRequest, Product } from '@/types/product';
import { formatNumber, parseFormattedNumber } from '@/utils/format';

interface ProductFormProps {
  initialData?: Product;
  onSubmit: (data: any) => Promise<void>;
  onCancel?: () => void;
  isEdit?: boolean;
}

const ProductForm: React.FC<ProductFormProps> = ({
  initialData,
  onSubmit,
  onCancel,
  isEdit = false,
}) => {
  const { t } = useTranslation(['products', 'common']);
  const [formData, setFormData] = useState({
    sku: initialData?.sku || '',
    name: initialData?.name || '',
    description: initialData?.description || '',
    category_id: initialData?.category_id || '',
    selling_price: initialData?.selling_price?.toString() || '',
    cost_price: initialData?.cost_price?.toString() || '',
    tax_rate: initialData?.tax_rate?.toString() || '0',
    stock_quantity: initialData?.stock_quantity?.toString() || '0',
  });

  const [displayPrices, setDisplayPrices] = useState({
    selling_price: initialData?.selling_price ? formatNumber(initialData.selling_price, 0) : '',
    cost_price: initialData?.cost_price ? formatNumber(initialData.cost_price, 0) : '',
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.sku.trim()) {
      newErrors.sku = t('products.validation.skuRequired');
    } else if (formData.sku.length > 50) {
      newErrors.sku = t('products.validation.skuMaxLength');
    }

    if (!formData.name.trim()) {
      newErrors.name = t('products.validation.nameRequired');
    } else if (formData.name.length > 255) {
      newErrors.name = t('products.validation.nameMaxLength');
    }

    const sellingPrice = parseFloat(formData.selling_price);
    if (isNaN(sellingPrice) || sellingPrice < 0) {
      newErrors.selling_price = t('products.validation.sellingPricePositive');
    }

    const costPrice = parseFloat(formData.cost_price);
    if (isNaN(costPrice) || costPrice < 0) {
      newErrors.cost_price = t('products.validation.costPricePositive');
    }

    const taxRate = parseFloat(formData.tax_rate);
    if (isNaN(taxRate) || taxRate < 0 || taxRate > 100) {
      newErrors.tax_rate = t('products.validation.taxRateRange');
    }

    if (!isEdit) {
      const stockQuantity = parseInt(formData.stock_quantity);
      if (isNaN(stockQuantity)) {
        newErrors.stock_quantity = t('products.validation.stockQuantityNumber');
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      const submitData: any = {
        sku: formData.sku.trim(),
        name: formData.name.trim(),
        description: formData.description.trim() || undefined,
        category_id: formData.category_id || undefined,
        selling_price: parseFloat(formData.selling_price),
        cost_price: parseFloat(formData.cost_price),
        tax_rate: parseFloat(formData.tax_rate),
      };

      if (!isEdit) {
        submitData.stock_quantity = parseInt(formData.stock_quantity);
      }

      await onSubmit(submitData);
    } catch (error: any) {
      console.error('Form submission error:', error);
      if (error.response?.data?.message) {
        setErrors({ submit: error.response.data.message });
      } else {
        setErrors({ submit: t('products.messages.saveError') });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const handlePriceChange = (field: 'selling_price' | 'cost_price', displayValue: string) => {
    // Remove all non-digit characters except decimal point
    const cleanValue = displayValue.replace(/[^\d.]/g, '');

    // Only allow one decimal point
    const parts = cleanValue.split('.');
    const sanitized = parts.length > 2 ? parts[0] + '.' + parts.slice(1).join('') : cleanValue;

    // Store clean value
    setFormData(prev => ({ ...prev, [field]: sanitized }));

    // Format display value with thousand separator
    if (sanitized) {
      const [intPart, decPart] = sanitized.split('.');
      const formattedInt = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
      const formatted = decPart !== undefined ? `${formattedInt}.${decPart}` : formattedInt;
      setDisplayPrices(prev => ({ ...prev, [field]: formatted }));
    } else {
      setDisplayPrices(prev => ({ ...prev, [field]: '' }));
    }

    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const handlePriceBlur = (field: 'selling_price' | 'cost_price') => {
    const value = parseFloat(formData[field]);
    if (!isNaN(value)) {
      const formatted = formatNumber(value, 0);
      setDisplayPrices(prev => ({ ...prev, [field]: formatted }));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {errors.submit && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-md">
          <p className="text-sm text-red-600">{errors.submit}</p>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* SKU */}
        <div>
          <label htmlFor="sku" className="block text-sm font-medium text-gray-700 mb-1">
            {t('products.form.sku')} <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            id="sku"
            value={formData.sku}
            onChange={e => handleChange('sku', e.target.value)}
            disabled={isSubmitting}
            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${errors.sku ? 'border-red-500' : 'border-gray-300'
              }`}
            placeholder={t('products.form.skuPlaceholder')}
          />
          {errors.sku && <p className="mt-1 text-sm text-red-600">{errors.sku}</p>}
        </div>

        {/* Name */}
        <div>
          <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-1">
            {t('products.form.name')} <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            id="name"
            value={formData.name}
            onChange={e => handleChange('name', e.target.value)}
            disabled={isSubmitting}
            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${errors.name ? 'border-red-500' : 'border-gray-300'
              }`}
            placeholder={t('products.form.namePlaceholder')}
          />
          {errors.name && <p className="mt-1 text-sm text-red-600">{errors.name}</p>}
        </div>

        {/* Category */}
        <div>
          <label htmlFor="category" className="block text-sm font-medium text-gray-700 mb-1">
            {t('products.form.category')}
          </label>
          <CategorySelect
            value={formData.category_id}
            onChange={value => handleChange('category_id', value)}
            error={errors.category_id}
            disabled={isSubmitting}
          />
        </div>

        {/* Selling Price */}
        <div>
          <label htmlFor="selling_price" className="block text-sm font-medium text-gray-700 mb-1">
            {t('products.form.sellingPrice')} <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            id="selling_price"
            value={displayPrices.selling_price}
            onChange={e => handlePriceChange('selling_price', e.target.value)}
            onBlur={() => handlePriceBlur('selling_price')}
            disabled={isSubmitting}
            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${errors.selling_price ? 'border-red-500' : 'border-gray-300'
              }`}
            placeholder="0.00"
          />
          {errors.selling_price && (
            <p className="mt-1 text-sm text-red-600">{errors.selling_price}</p>
          )}
        </div>

        {/* Cost Price */}
        <div>
          <label htmlFor="cost_price" className="block text-sm font-medium text-gray-700 mb-1">
            {t('products.form.costPrice')} <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            id="cost_price"
            value={displayPrices.cost_price}
            onChange={e => handlePriceChange('cost_price', e.target.value)}
            onBlur={() => handlePriceBlur('cost_price')}
            disabled={isSubmitting}
            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${errors.cost_price ? 'border-red-500' : 'border-gray-300'
              }`}
            placeholder="0.00"
          />
          {errors.cost_price && <p className="mt-1 text-sm text-red-600">{errors.cost_price}</p>}
        </div>

        {/* Tax Rate */}
        <div>
          <label htmlFor="tax_rate" className="block text-sm font-medium text-gray-700 mb-1">
            {t('products.form.taxRate')}
          </label>
          <input
            type="number"
            id="tax_rate"
            step="0.01"
            min="0"
            max="100"
            value={formData.tax_rate}
            onChange={e => handleChange('tax_rate', e.target.value)}
            disabled={isSubmitting}
            className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${errors.tax_rate ? 'border-red-500' : 'border-gray-300'
              }`}
            placeholder="0"
          />
          {errors.tax_rate && <p className="mt-1 text-sm text-red-600">{errors.tax_rate}</p>}
        </div>

        {/* Stock Quantity (only for create) */}
        {!isEdit && (
          <div>
            <label
              htmlFor="stock_quantity"
              className="block text-sm font-medium text-gray-700 mb-1"
            >
              {t('products.form.stockQuantity')}
            </label>
            <input
              type="number"
              id="stock_quantity"
              min="0"
              value={formData.stock_quantity}
              onChange={e => handleChange('stock_quantity', e.target.value)}
              disabled={isSubmitting}
              className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${errors.stock_quantity ? 'border-red-500' : 'border-gray-300'
                }`}
              placeholder="0"
            />
            {errors.stock_quantity && (
              <p className="mt-1 text-sm text-red-600">{errors.stock_quantity}</p>
            )}
          </div>
        )}
      </div>

      {/* Description */}
      <div>
        <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
          {t('products.form.description')}
        </label>
        <textarea
          id="description"
          rows={4}
          value={formData.description}
          onChange={e => handleChange('description', e.target.value)}
          disabled={isSubmitting}
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          placeholder={t('products.form.descriptionPlaceholder')}
        />
      </div>

      {/* Form Actions */}
      <div className="flex justify-end space-x-3 pt-4">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
          >
            {t('common.cancel', { ns: 'common' })}
          </button>
        )}
        <button
          type="submit"
          disabled={isSubmitting}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {isSubmitting ? t('common.saving', { ns: 'common' }) : isEdit ? t('products.form.updateButton') : t('products.form.createButton')}
        </button>
      </div>
    </form>
  );
};

export default ProductForm;
