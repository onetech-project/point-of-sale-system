'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import { useTranslation } from '@/i18n/provider';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import ProductList from '@/components/products/ProductList';
import productService from '@/services/product';
import { Category } from '@/types/product';
import InventoryDashboard from '@/components/products/InventoryDashboard';

export default function ProductsPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading } = useAuth();
  const { t } = useTranslation(['products', 'common']);
  const [categories, setCategories] = useState<Category[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [showArchived, setShowArchived] = useState(false);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, isLoading, router]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchCategories();
    }
  }, [isAuthenticated]);

  const fetchCategories = async () => {
    try {
      const data = await productService.getCategories();
      setCategories(data);
    } catch (err) {
      console.error('Failed to fetch categories:', err);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">{t('common.loading', { ns: 'common' })}</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">{t('products.title')}</h1>
            <p className="mt-1 text-sm text-gray-500">
              {t('products.subtitle')}
            </p>
          </div>
          <button
            onClick={() => router.push('/products/new')}
            className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-lg hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
          >
            + {t('products.addProduct')}
          </button>
        </div>

        {/* Inventory Dashboard */}
        <div className="mb-6">
          <InventoryDashboard />
        </div>

        {/* Filters */}
        <div className="mb-6 flex flex-wrap gap-4 items-center">
          <div className="flex items-center space-x-2">
            <label htmlFor="category-filter" className="text-sm font-medium text-gray-700">
              {t('products.list.category')}:
            </label>
            <select
              id="category-filter"
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="">{t('products.list.allCategories')}</option>
              {categories.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex items-center">
            <label className="flex items-center space-x-2 text-sm">
              <input
                type="checkbox"
                checked={showArchived}
                onChange={(e) => setShowArchived(e.target.checked)}
                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <span>{t('products.list.showArchived')}</span>
            </label>
          </div>

          <button
            onClick={() => router.push('/products/categories')}
            className="ml-auto px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
          >
            {t('products.actions.manageCategories')}
          </button>
        </div>

        {/* Product List */}
        <ProductList categoryFilter={selectedCategory} showArchived={showArchived} />
      </DashboardLayout>
    </ProtectedRoute>
  );
}
