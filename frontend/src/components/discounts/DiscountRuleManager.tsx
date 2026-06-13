'use client';

import { useEffect, useMemo, useState } from 'react';
import { BadgePercent, Edit2, Loader2, RefreshCw, Save, Tag, Trash2, X } from 'lucide-react';
import { discounts } from '@/services/discounts';
import { inventory } from '@/services/inventory';
import { product as productService } from '@/services/product';
import type {
  DiscountRule,
  DiscountTargetType,
  DiscountType,
  PricingResult,
} from '@/types/discounts';
import type { Bundle } from '@/types/inventory';
import type { Product } from '@/types/product';
import { formatCurrency } from '@/utils/format';
import { getApiErrorMessage, toNumber } from '@/components/inventory/inventoryUtils';

interface DiscountFormState {
  name: string;
  description: string;
  target_type: DiscountTargetType;
  target_id: string;
  discount_type: DiscountType;
  discount_value: string;
  starts_at: string;
  ends_at: string;
  is_active: boolean;
  exclusive: boolean;
  priority: string;
}

const emptyForm = (): DiscountFormState => ({
  name: '',
  description: '',
  target_type: 'product',
  target_id: '',
  discount_type: 'percentage',
  discount_value: '0',
  starts_at: '',
  ends_at: '',
  is_active: true,
  exclusive: true,
  priority: '100',
});

const toLocalInputValue = (value?: string | null) => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString().slice(0, 16);
};

const toIsoOrNull = (value: string) => {
  if (!value.trim()) {
    return null;
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date.toISOString();
};

const formatRuleValue = (rule: Pick<DiscountRule, 'discount_type' | 'discount_value'>) =>
  rule.discount_type === 'percentage'
    ? `${rule.discount_value}%`
    : formatCurrency(rule.discount_value, false, 0);

export default function DiscountRuleManager() {
  const [rules, setRules] = useState<DiscountRule[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [bundles, setBundles] = useState<Bundle[]>([]);
  const [selectedRule, setSelectedRule] = useState<DiscountRule | null>(null);
  const [form, setForm] = useState<DiscountFormState>(emptyForm);
  const [previewQuantity, setPreviewQuantity] = useState('1');
  const [previewUnitPrice, setPreviewUnitPrice] = useState('0');
  const [preview, setPreview] = useState<PricingResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [previewing, setPreviewing] = useState(false);
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

  const bundleById = useMemo(() => {
    const map = new Map<string, Bundle>();
    bundles.forEach(bundle => map.set(bundle.id, bundle));
    return map;
  }, [bundles]);

  const targetOptions = form.target_type === 'product' ? products : bundles;

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [ruleResponse, productResponse, bundleResponse] = await Promise.all([
        discounts.listDiscountRules({ limit: 100, offset: 0 }),
        productService.getProducts({ page: 1, limit: 200, archived: false }),
        inventory.getBundles({ page: 1, limit: 200, includeInactive: true }),
      ]);

      setRules(ruleResponse);
      setProducts(productResponse.data || []);
      setBundles(bundleResponse.data);
    } catch (err) {
      console.error('Failed to load discount rules:', err);
      setError(getApiErrorMessage(err, 'Failed to load discount rules.'));
    } finally {
      setLoading(false);
    }
  };

  const targetName = (type: DiscountTargetType, id: string) => {
    if (type === 'product') {
      const product = productById.get(id);
      return product ? `${product.name} (${product.sku})` : id;
    }

    const bundle = bundleById.get(id);
    return bundle ? `${bundle.name} (${bundle.sku})` : id;
  };

  const targetPrice = (type: DiscountTargetType, id: string) => {
    if (type === 'product') {
      return productById.get(id)?.selling_price || 0;
    }
    return toNumber(bundleById.get(id)?.selling_price);
  };

  const resetForm = () => {
    setSelectedRule(null);
    setForm(emptyForm());
    setPreview(null);
    setPreviewQuantity('1');
    setPreviewUnitPrice('0');
    setError(null);
    setSuccess(null);
  };

  const editRule = (rule: DiscountRule) => {
    setSelectedRule(rule);
    setForm({
      name: rule.name,
      description: rule.description || '',
      target_type: rule.target_type,
      target_id: rule.target_id,
      discount_type: rule.discount_type,
      discount_value: String(rule.discount_value),
      starts_at: toLocalInputValue(rule.starts_at),
      ends_at: toLocalInputValue(rule.ends_at),
      is_active: rule.is_active,
      exclusive: rule.exclusive,
      priority: String(rule.priority),
    });
    setPreviewUnitPrice(String(targetPrice(rule.target_type, rule.target_id)));
    setPreviewQuantity('1');
    setPreview(null);
    setSuccess(null);
    setError(null);
  };

  const handleTargetTypeChange = (targetType: DiscountTargetType) => {
    setForm(current => ({ ...current, target_type: targetType, target_id: '' }));
    setPreviewUnitPrice('0');
    setPreview(null);
  };

  const handleTargetChange = (targetId: string) => {
    setForm(current => ({ ...current, target_id: targetId }));
    setPreviewUnitPrice(String(targetPrice(form.target_type, targetId)));
    setPreview(null);
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);
    setSuccess(null);

    if (!form.name.trim() || !form.target_id) {
      setError('Name and target are required.');
      return;
    }

    const discountValue = Number(form.discount_value);
    if (!Number.isFinite(discountValue) || discountValue <= 0) {
      setError('Discount value must be greater than zero.');
      return;
    }
    if (form.discount_type === 'percentage' && discountValue > 100) {
      setError('Percentage discounts cannot exceed 100%.');
      return;
    }

    const priority = Number.parseInt(form.priority, 10);
    if (!Number.isFinite(priority)) {
      setError('Priority must be a number.');
      return;
    }

    try {
      setSaving(true);
      const payload = {
        name: form.name.trim(),
        description: form.description.trim() || null,
        target_type: form.target_type,
        target_id: form.target_id,
        discount_type: form.discount_type,
        discount_value: discountValue,
        starts_at: toIsoOrNull(form.starts_at),
        ends_at: toIsoOrNull(form.ends_at),
        is_active: form.is_active,
        exclusive: form.exclusive,
        priority,
      };

      const saved = selectedRule
        ? await discounts.updateDiscountRule(selectedRule.id, payload)
        : await discounts.createDiscountRule(payload);

      setSelectedRule(saved);
      setForm({
        name: saved.name,
        description: saved.description || '',
        target_type: saved.target_type,
        target_id: saved.target_id,
        discount_type: saved.discount_type,
        discount_value: String(saved.discount_value),
        starts_at: toLocalInputValue(saved.starts_at),
        ends_at: toLocalInputValue(saved.ends_at),
        is_active: saved.is_active,
        exclusive: saved.exclusive,
        priority: String(saved.priority),
      });
      setPreviewUnitPrice(String(targetPrice(saved.target_type, saved.target_id)));
      setSuccess(selectedRule ? 'Discount rule updated.' : 'Discount rule created.');
      await loadData();
    } catch (err) {
      console.error('Failed to save discount rule:', err);
      setError(getApiErrorMessage(err, 'Failed to save discount rule.'));
    } finally {
      setSaving(false);
    }
  };

  const deleteRule = async (rule: DiscountRule) => {
    if (!window.confirm(`Delete discount rule "${rule.name}"?`)) {
      return;
    }

    try {
      setError(null);
      await discounts.deleteDiscountRule(rule.id);
      setSuccess('Discount rule deleted.');
      if (selectedRule?.id === rule.id) {
        resetForm();
      }
      await loadData();
    } catch (err) {
      console.error('Failed to delete discount rule:', err);
      setError(getApiErrorMessage(err, 'Failed to delete discount rule.'));
    }
  };

  const previewPricing = async (rule?: DiscountRule) => {
    const itemType = rule?.target_type || form.target_type;
    const targetId = rule?.target_id || form.target_id;
    const unitPrice = toNumber(previewUnitPrice) || targetPrice(itemType, targetId);
    const quantity = Number.parseInt(previewQuantity, 10) || 1;
    const discountRuleId = rule?.id || selectedRule?.id;

    if (!targetId || unitPrice < 0) {
      setError('Select a target and unit price before previewing.');
      return;
    }

    try {
      setPreviewing(true);
      setError(null);
      const result = await discounts.previewPricing({
        item_type: itemType,
        product_id: itemType === 'product' ? targetId : undefined,
        bundle_id: itemType === 'bundle' ? targetId : undefined,
        quantity,
        unit_price: Math.round(unitPrice),
        discount_rule_id: discountRuleId || undefined,
        apply_discount: !discountRuleId,
      });
      setPreview(result);
    } catch (err) {
      console.error('Failed to preview pricing:', err);
      setError(getApiErrorMessage(err, 'Failed to preview discount pricing.'));
    } finally {
      setPreviewing(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16 text-sm text-gray-500">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" aria-hidden="true" />
        Loading discount rules...
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
          <div className="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Discount rules</h2>
              <p className="text-sm text-gray-500">
                Product and bundle discounts used by order pricing snapshots.
              </p>
            </div>
            <button
              type="button"
              onClick={loadData}
              className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Refresh
            </button>
          </div>

          {rules.length === 0 ? (
            <div className="px-5 py-12 text-center text-sm text-gray-500">
              No discount rules found.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Rule
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Target
                    </th>
                    <th className="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500">
                      Value
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
                  {rules.map(rule => (
                    <tr key={rule.id}>
                      <td className="whitespace-nowrap px-5 py-4">
                        <div className="font-medium text-gray-900">{rule.name}</div>
                        <div className="text-sm text-gray-500">
                          Priority {rule.priority} {rule.exclusive ? 'exclusive' : 'stackable'}
                        </div>
                      </td>
                      <td className="px-5 py-4 text-sm text-gray-700">
                        <div className="font-medium capitalize">{rule.target_type}</div>
                        <div className="text-gray-500">
                          {targetName(rule.target_type, rule.target_id)}
                        </div>
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-right text-sm font-medium text-gray-900">
                        {formatRuleValue(rule)}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4">
                        <span
                          className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${
                            rule.is_active
                              ? 'bg-green-100 text-green-700'
                              : 'bg-gray-100 text-gray-700'
                          }`}
                        >
                          {rule.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-right">
                        <div className="inline-flex gap-2">
                          <button
                            type="button"
                            onClick={() => previewPricing(rule)}
                            className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                          >
                            <BadgePercent className="h-4 w-4" aria-hidden="true" />
                            Preview
                          </button>
                          <button
                            type="button"
                            onClick={() => editRule(rule)}
                            className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                          >
                            <Edit2 className="h-4 w-4" aria-hidden="true" />
                            Edit
                          </button>
                          <button
                            type="button"
                            onClick={() => deleteRule(rule)}
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

        <section className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Pricing preview</h2>
              <p className="text-sm text-gray-500">
                Preview the selected saved rule, or the active rule for the current target.
              </p>
            </div>
            <button
              type="button"
              onClick={() => previewPricing()}
              disabled={previewing}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:bg-gray-400"
            >
              {previewing ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <BadgePercent className="h-4 w-4" aria-hidden="true" />
              )}
              Preview current target
            </button>
          </div>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <label className="block">
              <span className="text-sm font-medium text-gray-700">Quantity</span>
              <input
                type="number"
                min="1"
                step="1"
                value={previewQuantity}
                onChange={event => setPreviewQuantity(event.target.value)}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              />
            </label>
            <label className="block">
              <span className="text-sm font-medium text-gray-700">Unit price</span>
              <input
                type="number"
                min="0"
                step="1"
                value={previewUnitPrice}
                onChange={event => setPreviewUnitPrice(event.target.value)}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              />
            </label>
          </div>

          {preview && (
            <div className="mt-5 grid gap-4 sm:grid-cols-4">
              <PreviewMetric
                label="List total"
                value={formatCurrency(preview.list_total_price, false, 0)}
              />
              <PreviewMetric
                label="Discount"
                value={formatCurrency(preview.discount_amount, false, 0)}
              />
              <PreviewMetric
                label="Final total"
                value={formatCurrency(preview.total_price, false, 0)}
              />
              <PreviewMetric label="Rule" value={preview.discount_name || 'No discount'} />
            </div>
          )}
        </section>
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-5 rounded-lg border border-gray-200 bg-white p-5 shadow-sm"
      >
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">
              {selectedRule ? 'Edit rule' : 'New rule'}
            </h2>
            <p className="text-sm text-gray-500">Set one discount target per rule.</p>
          </div>
          {selectedRule && (
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
            <span className="text-sm font-medium text-gray-700">Target type</span>
            <select
              value={form.target_type}
              onChange={event => handleTargetTypeChange(event.target.value as DiscountTargetType)}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            >
              <option value="product">Product</option>
              <option value="bundle">Bundle</option>
            </select>
          </label>
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Target</span>
            <select
              value={form.target_id}
              onChange={event => handleTargetChange(event.target.value)}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              required
            >
              <option value="">Select target</option>
              {targetOptions.map(target => (
                <option key={target.id} value={target.id}>
                  {target.name} ({target.sku})
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Discount type</span>
            <select
              value={form.discount_type}
              onChange={event =>
                setForm(current => ({
                  ...current,
                  discount_type: event.target.value as DiscountType,
                }))
              }
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            >
              <option value="percentage">Percentage</option>
              <option value="fixed_amount">Fixed amount</option>
            </select>
          </label>
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Discount value</span>
            <input
              type="number"
              min="0"
              step={form.discount_type === 'percentage' ? '0.01' : '1'}
              value={form.discount_value}
              onChange={event =>
                setForm(current => ({ ...current, discount_value: event.target.value }))
              }
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              required
            />
          </label>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Starts at</span>
            <input
              type="datetime-local"
              value={form.starts_at}
              onChange={event =>
                setForm(current => ({ ...current, starts_at: event.target.value }))
              }
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </label>
          <label className="block">
            <span className="text-sm font-medium text-gray-700">Ends at</span>
            <input
              type="datetime-local"
              value={form.ends_at}
              onChange={event => setForm(current => ({ ...current, ends_at: event.target.value }))}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </label>
        </div>

        <label className="block">
          <span className="text-sm font-medium text-gray-700">Priority</span>
          <input
            type="number"
            step="1"
            value={form.priority}
            onChange={event => setForm(current => ({ ...current, priority: event.target.value }))}
            className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </label>

        <div className="space-y-3">
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700">
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
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700">
            <input
              type="checkbox"
              checked={form.exclusive}
              onChange={event =>
                setForm(current => ({ ...current, exclusive: event.target.checked }))
              }
              className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            Exclusive
          </label>
        </div>

        <button
          type="submit"
          disabled={saving}
          className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:bg-gray-400"
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : selectedRule ? (
            <Save className="h-4 w-4" aria-hidden="true" />
          ) : (
            <Tag className="h-4 w-4" aria-hidden="true" />
          )}
          {selectedRule ? 'Save rule' : 'Create rule'}
        </button>
      </form>
    </div>
  );
}

function PreviewMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
      <div className="text-xs font-medium uppercase text-gray-500">{label}</div>
      <div className="mt-1 truncate text-lg font-bold text-gray-900">{value}</div>
    </div>
  );
}
