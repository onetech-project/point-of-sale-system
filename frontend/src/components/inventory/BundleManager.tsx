'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  Calculator,
  Edit2,
  Loader2,
  PackagePlus,
  Plus,
  RefreshCw,
  Save,
  Search,
  Trash2,
  X,
} from 'lucide-react';
import { inventory } from '@/services/inventory';
import { product as productService } from '@/services/product';
import type { Bundle, BundleCost } from '@/types/inventory';
import type { Product } from '@/types/product';
import { formatCurrency } from '@/utils/format';
import { decimalInputValue, formatQuantity, getApiErrorMessage, toNumber } from './inventoryUtils';

interface BundleFormItem {
  client_id: string;
  product_id: string;
  quantity: string;
}

interface BundleFormState {
  sku: string;
  name: string;
  description: string;
  selling_price: string;
  is_active: boolean;
  items: BundleFormItem[];
}

const emptyForm = (): BundleFormState => ({
  sku: '',
  name: '',
  description: '',
  selling_price: '0',
  is_active: true,
  items: [],
});

const makeFormItem = (productId: string, quantity: string): BundleFormItem => ({
  client_id: `${productId}-${Date.now()}-${Math.random().toString(16).slice(2)}`,
  product_id: productId,
  quantity,
});

const formatMoney = (value: unknown) => formatCurrency(toNumber(value as never), false, 0);

export default function BundleManager() {
  const [bundles, setBundles] = useState<Bundle[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [selectedBundle, setSelectedBundle] = useState<Bundle | null>(null);
  const [bundleCost, setBundleCost] = useState<BundleCost | null>(null);
  const [form, setForm] = useState<BundleFormState>(emptyForm);
  const [itemProductId, setItemProductId] = useState('');
  const [itemQuantity, setItemQuantity] = useState('1');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [costLoading, setCostLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    void loadData();
  }, []);

  const productById = useMemo(() => {
    const map = new Map<string, Product>();
    products.forEach(product => map.set(product.id, product));
    return map;
  }, [products]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [bundleResponse, productResponse] = await Promise.all([
        inventory.getBundles({ page: 1, limit: 100, includeInactive: true, search }),
        productService.getProducts({ page: 1, limit: 200, archived: false }),
      ]);

      setBundles(bundleResponse.data);
      setProducts(productResponse.data || []);
    } catch (err) {
      console.error('Failed to load bundles:', err);
      setError(getApiErrorMessage(err, 'Failed to load bundle data.'));
    } finally {
      setLoading(false);
    }
  };

  const loadCost = async (bundleId: string) => {
    try {
      setCostLoading(true);
      setBundleCost(null);
      const cost = await inventory.getBundleCost(bundleId);
      setBundleCost(cost);
    } catch (err) {
      console.error('Failed to load bundle cost:', err);
      setError(getApiErrorMessage(err, 'Failed to calculate bundle cost.'));
    } finally {
      setCostLoading(false);
    }
  };

  const resetForm = () => {
    setSelectedBundle(null);
    setBundleCost(null);
    setForm(emptyForm());
    setItemProductId('');
    setItemQuantity('1');
    setError(null);
    setSuccess(null);
  };

  const editBundle = (bundle: Bundle) => {
    setSelectedBundle(bundle);
    setForm({
      sku: bundle.sku,
      name: bundle.name,
      description: bundle.description || '',
      selling_price: decimalInputValue(bundle.selling_price),
      is_active: bundle.is_active,
      items: bundle.items.map(item =>
        makeFormItem(item.product_id, decimalInputValue(item.quantity))
      ),
    });
    setSuccess(null);
    setError(null);
    void loadCost(bundle.id);
  };

  const addItem = () => {
    if (!itemProductId || toNumber(itemQuantity) <= 0) {
      setError('Select a product and enter a positive quantity.');
      return;
    }

    setForm(current => {
      const existingIndex = current.items.findIndex(item => item.product_id === itemProductId);
      if (existingIndex >= 0) {
        const nextItems = [...current.items];
        const existing = nextItems[existingIndex];
        nextItems[existingIndex] = {
          ...existing,
          quantity: String(toNumber(existing.quantity) + toNumber(itemQuantity)),
        };
        return { ...current, items: nextItems };
      }

      return {
        ...current,
        items: [...current.items, makeFormItem(itemProductId, itemQuantity)],
      };
    });

    setItemProductId('');
    setItemQuantity('1');
    setError(null);
  };

  const updateItemQuantity = (clientId: string, quantity: string) => {
    setForm(current => ({
      ...current,
      items: current.items.map(item =>
        item.client_id === clientId ? { ...item, quantity } : item
      ),
    }));
  };

  const removeItem = (clientId: string) => {
    setForm(current => ({
      ...current,
      items: current.items.filter(item => item.client_id !== clientId),
    }));
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);
    setSuccess(null);

    const sellingPrice = toNumber(form.selling_price);
    const items = form.items.map(item => ({
      product_id: item.product_id,
      quantity: toNumber(item.quantity),
    }));

    if (!form.sku.trim() || !form.name.trim()) {
      setError('SKU and name are required.');
      return;
    }
    if (sellingPrice < 0) {
      setError('Selling price cannot be negative.');
      return;
    }
    if (items.length === 0 || items.some(item => !item.product_id || item.quantity <= 0)) {
      setError('Add at least one product with a positive quantity.');
      return;
    }

    try {
      setSaving(true);
      const payload = {
        sku: form.sku.trim(),
        name: form.name.trim(),
        description: form.description.trim() || null,
        selling_price: sellingPrice,
        is_active: form.is_active,
        items,
      };

      const saved = selectedBundle
        ? await inventory.updateBundle(selectedBundle.id, payload)
        : await inventory.createBundle(payload);

      setSuccess(selectedBundle ? 'Bundle updated.' : 'Bundle created.');
      setSelectedBundle(saved);
      setForm({
        sku: saved.sku,
        name: saved.name,
        description: saved.description || '',
        selling_price: decimalInputValue(saved.selling_price),
        is_active: saved.is_active,
        items: saved.items.map(item =>
          makeFormItem(item.product_id, decimalInputValue(item.quantity))
        ),
      });
      await loadData();
      await loadCost(saved.id);
    } catch (err) {
      console.error('Failed to save bundle:', err);
      setError(getApiErrorMessage(err, 'Failed to save bundle.'));
    } finally {
      setSaving(false);
    }
  };

  const deleteBundle = async (bundle: Bundle) => {
    if (!window.confirm(`Delete bundle "${bundle.name}"?`)) {
      return;
    }

    try {
      setError(null);
      await inventory.deleteBundle(bundle.id);
      setSuccess('Bundle deleted.');
      if (selectedBundle?.id === bundle.id) {
        resetForm();
      }
      await loadData();
    } catch (err) {
      console.error('Failed to delete bundle:', err);
      setError(getApiErrorMessage(err, 'Failed to delete bundle.'));
    }
  };

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void loadData();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16 text-sm text-gray-500">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" aria-hidden="true" />
        Loading bundles...
      </div>
    );
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-6">
        {(error || success) && (
          <div
            className={`rounded-md border p-4 text-sm ${
              error
                ? 'border-red-200 bg-red-50 text-red-700'
                : 'border-green-200 bg-green-50 text-green-700'
            }`}
          >
            {error || success}
          </div>
        )}

        <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <div className="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Bundles</h2>
              <p className="text-sm text-gray-500">
                Product groups sold as one item with expanded recipe COGS.
              </p>
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <form onSubmit={submitSearch} className="flex gap-2">
                <input
                  type="search"
                  value={search}
                  onChange={event => setSearch(event.target.value)}
                  placeholder="Search bundles"
                  className="min-w-0 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
                />
                <button
                  type="submit"
                  className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                  <Search className="h-4 w-4" aria-hidden="true" />
                  Search
                </button>
              </form>
              <button
                type="button"
                onClick={loadData}
                className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                <RefreshCw className="h-4 w-4" aria-hidden="true" />
                Refresh
              </button>
            </div>
          </div>

          {bundles.length === 0 ? (
            <div className="px-5 py-12 text-center text-sm text-gray-500">No bundles found.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Bundle
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Items
                    </th>
                    <th className="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500">
                      Price
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Status
                    </th>
                    <th className="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 bg-white">
                  {bundles.map(bundle => (
                    <tr key={bundle.id}>
                      <td className="whitespace-nowrap px-5 py-4">
                        <div className="font-medium text-gray-900">{bundle.name}</div>
                        <div className="text-sm text-gray-500">{bundle.sku}</div>
                      </td>
                      <td className="px-5 py-4 text-sm text-gray-700">
                        {bundle.items.length.toLocaleString('en-US')}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-right text-sm font-medium text-gray-900">
                        {formatMoney(bundle.selling_price)}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${
                            bundle.is_active
                              ? 'bg-green-100 text-green-700'
                              : 'bg-gray-100 text-gray-700'
                          }`}
                        >
                          {bundle.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-right">
                        <div className="inline-flex gap-2">
                          <button
                            type="button"
                            onClick={() => editBundle(bundle)}
                            className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                          >
                            <Edit2 className="h-4 w-4" aria-hidden="true" />
                            Edit
                          </button>
                          <button
                            type="button"
                            onClick={() => deleteBundle(bundle)}
                            className="inline-flex items-center gap-1 rounded-lg border border-red-200 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50"
                          >
                            <Trash2 className="h-4 w-4" aria-hidden="true" />
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {selectedBundle && (
          <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
            <div className="flex items-center justify-between gap-3 border-b border-gray-200 px-5 py-4">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">Cost preview</h2>
                <p className="text-sm text-gray-500">{selectedBundle.name}</p>
              </div>
              <button
                type="button"
                onClick={() => loadCost(selectedBundle.id)}
                disabled={costLoading}
                className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
              >
                {costLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                ) : (
                  <Calculator className="h-4 w-4" aria-hidden="true" />
                )}
                Calculate
              </button>
            </div>

            {costLoading ? (
              <div className="px-5 py-8 text-sm text-gray-500">Calculating bundle COGS...</div>
            ) : bundleCost ? (
              <div className="space-y-5 p-5">
                <div className="grid gap-4 sm:grid-cols-4">
                  <CostMetric label="Material" value={formatMoney(bundleCost.material_cost)} />
                  <CostMetric
                    label="Overhead"
                    value={formatMoney(
                      toNumber(bundleCost.fixed_overhead_cost) +
                        toNumber(bundleCost.percentage_overhead_cost)
                    )}
                  />
                  <CostMetric label="COGS" value={formatMoney(bundleCost.total_cogs)} />
                  <CostMetric
                    label="Margin"
                    value={`${formatQuantity(bundleCost.gross_margin_percentage)}%`}
                  />
                </div>

                <div className="overflow-x-auto rounded-lg border border-gray-200">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                          Product
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">
                          Qty
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">
                          Unit COGS
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">
                          Line COGS
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 bg-white">
                      {bundleCost.items.map(item => (
                        <tr key={item.bundle_item_id}>
                          <td className="px-4 py-3 text-sm text-gray-900">
                            <div className="font-medium">{item.product_name}</div>
                            <div className="text-gray-500">v{item.recipe_version}</div>
                          </td>
                          <td className="px-4 py-3 text-right text-sm text-gray-700">
                            {formatQuantity(item.quantity)}
                          </td>
                          <td className="px-4 py-3 text-right text-sm text-gray-700">
                            {formatMoney(item.unit_total_cogs)}
                          </td>
                          <td className="px-4 py-3 text-right text-sm font-medium text-gray-900">
                            {formatMoney(item.line_total_cogs)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : (
              <div className="px-5 py-8 text-sm text-gray-500">
                Select calculate after saving a bundle.
              </div>
            )}
          </section>
        )}
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-5 rounded-lg border border-gray-200 bg-white p-5 shadow-sm"
      >
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">
              {selectedBundle ? 'Edit bundle' : 'New bundle'}
            </h2>
            <p className="text-sm text-gray-500">Set the selling price and included products.</p>
          </div>
          {selectedBundle && (
            <button
              type="button"
              onClick={resetForm}
              className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              <X className="h-4 w-4" aria-hidden="true" />
              New
            </button>
          )}
        </div>

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          <label className="block">
            <span className="text-sm font-medium text-gray-700">SKU</span>
            <input
              type="text"
              value={form.sku}
              onChange={event => setForm(current => ({ ...current, sku: event.target.value }))}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              required
            />
          </label>
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Name</span>
            <input
              type="text"
              value={form.name}
              onChange={event => setForm(current => ({ ...current, name: event.target.value }))}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              required
            />
          </label>
        </div>

        <label className="block">
          <span className="text-sm font-medium text-gray-700">Description</span>
          <textarea
            value={form.description}
            onChange={event =>
              setForm(current => ({ ...current, description: event.target.value }))
            }
            rows={3}
            className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </label>

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Selling price</span>
            <input
              type="number"
              min="0"
              step="1"
              value={form.selling_price}
              onChange={event =>
                setForm(current => ({ ...current, selling_price: event.target.value }))
              }
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              required
            />
          </label>
          <label className="flex items-center gap-2 pt-6 text-sm font-medium text-gray-700 xl:pt-0">
            <input
              type="checkbox"
              checked={form.is_active}
              onChange={event =>
                setForm(current => ({ ...current, is_active: event.target.checked }))
              }
              className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            Active
          </label>
        </div>

        <div className="space-y-3">
          <div>
            <h3 className="text-sm font-semibold text-gray-900">Bundle items</h3>
            <p className="text-sm text-gray-500">
              Products are expanded through their active recipes.
            </p>
          </div>

          <div className="grid gap-2 sm:grid-cols-[1fr_110px_auto]">
            <select
              value={itemProductId}
              onChange={event => setItemProductId(event.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            >
              <option value="">Select product</option>
              {products.map(product => (
                <option key={product.id} value={product.id}>
                  {product.name} ({product.sku})
                </option>
              ))}
            </select>
            <input
              type="number"
              min="0.01"
              step="0.01"
              value={itemQuantity}
              onChange={event => setItemQuantity(event.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
            <button
              type="button"
              onClick={addItem}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-700"
            >
              <Plus className="h-4 w-4" aria-hidden="true" />
              Add
            </button>
          </div>

          {form.items.length === 0 ? (
            <div className="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500">
              No products added.
            </div>
          ) : (
            <div className="overflow-hidden rounded-lg border border-gray-200">
              {form.items.map(item => {
                const product = productById.get(item.product_id);
                return (
                  <div
                    key={item.client_id}
                    className="grid gap-3 border-b border-gray-200 p-3 last:border-b-0 sm:grid-cols-[1fr_100px_auto] sm:items-center"
                  >
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium text-gray-900">
                        {product?.name || item.product_id}
                      </div>
                      <div className="text-xs text-gray-500">{product?.sku || 'Product'}</div>
                    </div>
                    <input
                      type="number"
                      min="0.01"
                      step="0.01"
                      value={item.quantity}
                      onChange={event => updateItemQuantity(item.client_id, event.target.value)}
                      className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
                    />
                    <button
                      type="button"
                      onClick={() => removeItem(item.client_id)}
                      className="inline-flex items-center justify-center rounded-lg border border-red-200 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
                      aria-label="Remove bundle item"
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <button
          type="submit"
          disabled={saving}
          className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:bg-gray-400"
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : selectedBundle ? (
            <Save className="h-4 w-4" aria-hidden="true" />
          ) : (
            <PackagePlus className="h-4 w-4" aria-hidden="true" />
          )}
          {selectedBundle ? 'Save bundle' : 'Create bundle'}
        </button>
      </form>
    </div>
  );
}

function CostMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
      <div className="text-xs font-medium uppercase text-gray-500">{label}</div>
      <div className="mt-1 text-lg font-bold text-gray-900">{value}</div>
    </div>
  );
}
