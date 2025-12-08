'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import ProductForm from '@/components/products/ProductForm';
import { product } from '@/services/product';
import { CreateProductRequest } from '@/types/product';

export default function NewProductPage() {
  const router = useRouter();
  const { t } = useTranslation(['products', 'common']);

  const handleSubmit = async (data: CreateProductRequest) => {
    try {
      const productData = await product.createProduct(data);
      router.push(`/products/${productData.id}`);
    } catch (error) {
      throw error;
    }
  };

  const handleCancel = () => {
    router.push('/products');
  };

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900">{t('products.addProduct')}</h1>
          <p className="mt-1 text-sm text-gray-500">
            {t('products.form.subtitle')}
          </p>
        </div>

        {/* Form */}
        <div className="bg-white shadow rounded-lg p-6">
          <ProductForm onSubmit={handleSubmit} onCancel={handleCancel} />
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
