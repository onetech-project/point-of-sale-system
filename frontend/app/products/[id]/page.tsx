'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import ProductForm from '@/components/products/ProductForm';
import StockAdjustmentModal from '@/components/products/StockAdjustmentModal';
import { product as productService } from '@/services/product';
import { Product, UpdateProductRequest, StockAdjustmentRequest } from '@/types/product';
import { formatNumber } from '@/utils/format';

export default function ProductDetailPage() {
  const router = useRouter();
  const params = useParams();
  const { t } = useTranslation(['products', 'common']);
  const productId = params?.id as string;

  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [uploadingPhoto, setUploadingPhoto] = useState(false);
  const [showStockModal, setShowStockModal] = useState(false);

  useEffect(() => {
    if (productId) {
      fetchProduct();
    }
  }, [productId]);

  const fetchProduct = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await productService.getProduct(productId);
      setProduct(data);
    } catch (err: any) {
      console.error('Failed to fetch product:', err);
      setError(err.response?.data?.message || 'Failed to load product');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateProduct = async (data: UpdateProductRequest) => {
    try {
      const updated = await productService.updateProduct(productId, data);
      setProduct(updated);
      setIsEditing(false);
    } catch (error) {
      throw error;
    }
  };

  const handlePhotoUpload = async (file: File) => {
    try {
      setUploadingPhoto(true);
      const updated = await productService.uploadPhoto(productId, file);
      setProduct(updated);
      setPhotoFile(null);
    } catch (err: any) {
      console.error('Failed to upload photo:', err);
      alert(err.response?.data?.message || 'Failed to upload photo');
    } finally {
      setUploadingPhoto(false);
    }
  };

  const handleDeletePhoto = async () => {
    if (!confirm('Are you sure you want to delete this photo?')) {
      return;
    }

    try {
      await productService.deletePhoto(productId);
      await fetchProduct();
    } catch (err: any) {
      console.error('Failed to delete photo:', err);
      alert(err.response?.data?.message || 'Failed to delete photo');
    }
  };

  const handleStockAdjustment = async (data: StockAdjustmentRequest) => {
    try {
      const updated = await productService.adjustStock(productId, data);
      setProduct(updated);
    } catch (error) {
      throw error;
    }
  };

  const handleArchive = async () => {
    if (!confirm(t('products.confirm.archive'))) {
      return;
    }

    try {
      await productService.archiveProduct(productId);
      await fetchProduct(); // Refetch to get updated data
    } catch (err: any) {
      console.error('Failed to archive product:', err);
      alert(err.response?.data?.message || t('products.messages.updateError'));
    }
  };

  const handleRestore = async () => {
    try {
      await productService.restoreProduct(productId);
      await fetchProduct(); // Refetch to get updated data
    } catch (err: any) {
      console.error('Failed to restore product:', err);
      alert(err.response?.data?.message || t('products.messages.updateError'));
    }
  };

  const handleDelete = async () => {
    if (!confirm(t('products.confirm.delete') + ' ' + t('products.confirm.deleteWarning'))) {
      return;
    }

    try {
      await productService.deleteProduct(productId);
      router.push('/products');
    } catch (err: any) {
      console.error('Failed to delete product:', err);
      alert(err.response?.data?.message || t('products.messages.deleteProtected'));
    }
  };

  if (loading) {
    return (
      <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
        <div className="min-h-screen flex items-center justify-center bg-gray-50">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">{t('common.loading', { ns: 'common' })}</p>
          </div>
        </div>
      </ProtectedRoute>
    );
  }

  if (error || !product) {
    return (
      <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
        <DashboardLayout>
          <div className="p-4 bg-red-50 border border-red-200 rounded-md">
            <p className="text-sm text-red-600">{error || t('products.messages.loadError')}</p>
            <button
              onClick={() => router.push('/products')}
              className="mt-2 text-sm text-primary-600 underline hover:text-primary-800"
            >
              {t('common.back', { ns: 'common' })}
            </button>
          </div>
        </DashboardLayout>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="max-w-6xl mx-auto">
          {/* Header */}
          <div className="mb-6 flex items-start justify-between">
            <div>
              <div className="flex items-center space-x-3">
                <h1 className="text-3xl font-bold text-gray-900">{product.name}</h1>
                {product.archived_at && (
                  <span className="px-3 py-1 text-sm font-medium bg-gray-100 text-gray-600 rounded-full">
                    Archived
                  </span>
                )}
              </div>
              <p className="mt-1 text-sm text-gray-500">SKU: {product.sku}</p>
            </div>

            <div className="flex space-x-2">
              <button
                onClick={() => router.push('/products')}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
              >
                {t('common.back', { ns: 'common' })}
              </button>
              {!isEditing && (
                <button
                  onClick={() => setIsEditing(true)}
                  className="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-lg hover:bg-primary-700"
                >
                  {t('common.edit', { ns: 'common' })}
                </button>
              )}
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Left Column - Product Photo */}
            <div className="lg:col-span-1">
              <div className="bg-white shadow rounded-lg p-6">
                <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('products.form.photo')}</h2>

                <div className="mb-4">
                  {product.photo_path ? (
                    <div className="relative">
                      <img
                        src={productService.getPhotoUrl(productId)}
                        alt={product.name}
                        className="w-full h-64 object-cover rounded-lg"
                        onError={(e) => {
                          const img = e.target as HTMLImageElement;
                          img.style.display = 'none';
                        }}
                      />
                      <button
                        onClick={handleDeletePhoto}
                        className="absolute top-2 right-2 p-2 bg-red-600 text-white rounded-full hover:bg-red-700"
                        title={t('products.form.deletePhoto')}
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                  ) : (
                    <div className="w-full h-64 bg-gray-100 rounded-lg flex items-center justify-center">
                      <div className="text-gray-400 text-center">
                        <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        </svg>
                        <p className="mt-2 text-sm">{t('products.noPhoto', { defaultValue: 'No photo' })}</p>
                      </div>
                    </div>
                  )}
                </div>

                <div>
                  <input
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) {
                        handlePhotoUpload(file);
                      }
                    }}
                    disabled={uploadingPhoto}
                    className="block w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-md file:border-0 file:text-sm file:font-medium file:bg-primary-50 file:text-primary-700 hover:file:bg-primary-100"
                  />
                  <p className="mt-1 text-xs text-gray-500">
                    {t('products.form.maxFileSize')}
                  </p>
                </div>
              </div>

              {/* Stock Info */}
              <div className="mt-6 bg-white shadow rounded-lg p-6">
                <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('products.details.inventory')}</h2>
                <div className="space-y-3">
                  <div className="flex justify-between">
                    <span className="text-sm text-gray-600">{t('products.details.currentStock')}:</span>
                    <span className="text-sm font-medium">{product.stock_quantity} {t('common.units', { ns: 'common' })}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-gray-600">{t('products.details.stockStatus')}:</span>
                    <span className={`text-sm font-medium ${product.stock_quantity <= 0
                      ? 'text-red-600'
                      : product.stock_quantity <= 10
                        ? 'text-yellow-600'
                        : 'text-green-600'
                      }`}>
                      {product.stock_quantity <= 0 ? t('products.list.outOfStock') : product.stock_quantity <= 10 ? t('products.list.lowStock') : t('products.list.inStock')}
                    </span>
                  </div>
                </div>
                <button
                  onClick={() => setShowStockModal(true)}
                  className="mt-4 w-full px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
                >
                  {t('products.actions.adjustStock')}
                </button>
              </div>
            </div>

            {/* Right Column - Product Details */}
            <div className="lg:col-span-2">
              {isEditing ? (
                <div className="bg-white shadow rounded-lg p-6">
                  <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('products.editProduct')}</h2>
                  <ProductForm
                    initialData={product}
                    onSubmit={handleUpdateProduct}
                    onCancel={() => setIsEditing(false)}
                    isEdit
                  />
                </div>
              ) : (
                <div className="space-y-6">
                  {/* Product Details Card */}
                  <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('products.details.information')}</h2>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-gray-500">{t('products.form.sku')}</label>
                        <p className="mt-1 text-gray-900">{product.sku}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-500">{t('products.form.category')}</label>
                        <p className="mt-1 text-gray-900">{product.category_name || t('common.uncategorized', { ns: 'common', defaultValue: 'Uncategorized' })}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-500">{t('products.form.sellingPrice')}</label>
                        <p className="mt-1 text-gray-900 text-lg font-bold">{formatNumber(product.selling_price)}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-500">{t('products.form.costPrice')}</label>
                        <p className="mt-1 text-gray-900">{formatNumber(product.cost_price)}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-500">{t('products.form.taxRate')}</label>
                        <p className="mt-1 text-gray-900">{product.tax_rate}%</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-500">{t('products.details.margin')}</label>
                        <p className="mt-1 text-gray-900">
                          {((product.selling_price - product.cost_price) / product.selling_price * 100).toFixed(1)}%
                        </p>
                      </div>
                    </div>

                    {product.description && (
                      <div className="mt-4">
                        <label className="text-sm font-medium text-gray-500">{t('products.form.description')}</label>
                        <p className="mt-1 text-gray-900">{product.description}</p>
                      </div>
                    )}

                    <div className="mt-4 pt-4 border-t border-gray-200">
                      <div className="text-sm text-gray-500">
                        {t('products.details.createdAt')}: {new Date(product.created_at).toLocaleDateString()}
                        {product.updated_at !== product.created_at && (
                          <> · {t('products.details.lastUpdated')}: {new Date(product.updated_at).toLocaleDateString()}</>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Actions Card */}
                  <div className="bg-white shadow rounded-lg p-6">
                    <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('common.actions', { ns: 'common' })}</h2>
                    <div className="space-y-3">
                      {product.archived_at ? (
                        <button
                          onClick={handleRestore}
                          className="w-full px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700"
                        >
                          {t('products.restoreProduct')}
                        </button>
                      ) : (
                        <button
                          onClick={handleArchive}
                          className="w-full px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                        >
                          {t('products.archiveProduct')}
                        </button>
                      )}
                      <button
                        onClick={handleDelete}
                        className="w-full px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700"
                      >
                        {t('products.deleteProduct')}
                      </button>
                    </div>
                    <p className="mt-2 text-xs text-gray-500">
                      {t('products.messages.deleteProtected')}
                    </p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Stock Adjustment Modal */}
        {showStockModal && product && (
          <StockAdjustmentModal
            productId={productId}
            productName={product.name}
            currentStock={product.stock_quantity}
            onClose={() => setShowStockModal(false)}
            onSubmit={handleStockAdjustment}
          />
        )}
      </DashboardLayout>
    </ProtectedRoute>
  );
}
