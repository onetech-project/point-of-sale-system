'use client';

import React, { useEffect, useState } from 'react';
import productService from '@/services/product';
import { Category } from '@/types/product';

interface CategorySelectProps {
  value?: string;
  onChange: (categoryId: string) => void;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
}

const CategorySelect: React.FC<CategorySelectProps> = ({
  value,
  onChange,
  error,
  required = false,
  disabled = false,
  placeholder = 'Select a category',
}) => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  useEffect(() => {
    const fetchCategories = async () => {
      try {
        setLoading(true);
        setFetchError(null);
        const data = await productService.getCategories();
        setCategories(data);
      } catch (err: any) {
        console.error('Failed to fetch categories:', err);
        setFetchError(err.response?.data?.message || 'Failed to load categories');
      } finally {
        setLoading(false);
      }
    };

    fetchCategories();
  }, []);

  return (
    <div className="w-full">
      <select
        value={value || ''}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled || loading}
        required={required}
        className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 ${
          error ? 'border-red-500' : 'border-gray-300'
        } ${disabled || loading ? 'bg-gray-100 cursor-not-allowed' : 'bg-white'}`}
      >
        <option value="">{loading ? 'Loading...' : placeholder}</option>
        {categories.map((category) => (
          <option key={category.id} value={category.id}>
            {category.name}
          </option>
        ))}
      </select>
      {fetchError && (
        <p className="mt-1 text-sm text-red-600">{fetchError}</p>
      )}
      {error && !fetchError && (
        <p className="mt-1 text-sm text-red-600">{error}</p>
      )}
    </div>
  );
};

export default CategorySelect;
