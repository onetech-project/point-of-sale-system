'use client';

import React, { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import ProductForm from '@/components/products/ProductForm';
import productService from '@/services/product';
import { CreateProductRequest } from '@/types/product';

export default function NewProductPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading } = useAuth();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, isLoading, router]);

  const handleSubmit = async (data: CreateProductRequest) => {
    try {
      const product = await productService.createProduct(data);
      router.push(`/products/${product.id}`);
    } catch (error) {
      throw error;
    }
  };

  const handleCancel = () => {
    router.push('/products');
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading...</p>
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
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900">Add New Product</h1>
          <p className="mt-1 text-sm text-gray-500">
            Create a new product in your inventory
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
