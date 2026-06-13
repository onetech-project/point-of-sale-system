'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  Calculator,
  ClipboardList,
  PackagePlus,
  RefreshCw,
  ShoppingCart,
  SlidersHorizontal,
  Trash2,
} from 'lucide-react';
import { inventory } from '@/services/inventory';
import type { Ingredient, StockMovement, StockMutationRequest, UOM } from '@/types/inventory';
import { formatCurrency } from '@/utils/format';
import {
  decimalInputValue,
  formatDateTime,
  formatQuantity,
  getApiErrorMessage,
  getStockStatus,
  getUOMCode,
  movementBadgeClass,
  movementTypeLabel,
  toNumber,
} from './inventoryUtils';

type StockOperation = 'initial' | 'purchase' | 'adjustment' | 'waste';

interface StockFormState {
  ingredient_id: string;
  uom_id: string;
  quantity: string;
  unit_cost: string;
  reason: string;
  reference_type: string;
  reference_id: string;
  occurred_at: string;
}

const operationItems: Array<{
  value: StockOperation;
  label: string;
  icon: React.ReactNode;
}> = [
  { value: 'initial', label: 'Initial', icon: <PackagePlus className="h-4 w-4" /> },
  { value: 'purchase', label: 'Purchase', icon: <ShoppingCart className="h-4 w-4" /> },
  { value: 'adjustment', label: 'Adjustment', icon: <SlidersHorizontal className="h-4 w-4" /> },
  { value: 'waste', label: 'Waste', icon: <Trash2 className="h-4 w-4" /> },
];

const emptyForm: StockFormState = {
  ingredient_id: '',
  uom_id: '',
  quantity: '',
  unit_cost: '0',
  reason: '',
  reference_type: '',
  reference_id: '',
  occurred_at: '',
};

const toDateTimeLocal = (date: Date) => {
  const offsetMs = date.getTimezoneOffset() * 60 * 1000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
};

const buildOptionalStockFields = (form: StockFormState) => {
  const optionalFields: Pick<
    StockMutationRequest,
    'reason' | 'reference_type' | 'reference_id' | 'occurred_at'
  > = {};

  if (form.reason.trim()) {
    optionalFields.reason = form.reason.trim();
  }
  if (form.reference_type.trim()) {
    optionalFields.reference_type = form.reference_type.trim();
  }
  if (form.reference_id.trim()) {
    optionalFields.reference_id = form.reference_id.trim();
  }
  if (form.occurred_at) {
    optionalFields.occurred_at = new Date(form.occurred_at).toISOString();
  }

  return optionalFields;
};

export default function StockManager() {
  const [operation, setOperation] = useState<StockOperation>('purchase');
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [uoms, setUOMs] = useState<UOM[]>([]);
  const [lowStock, setLowStock] = useState<Ingredient[]>([]);
  const [valuation, setValuation] = useState(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [form, setForm] = useState<StockFormState>({
    ...emptyForm,
    occurred_at: toDateTimeLocal(new Date()),
  });

  const [historyIngredientId, setHistoryIngredientId] = useState('');
  const [movements, setMovements] = useState<StockMovement[]>([]);
  const [movementsLoading, setMovementsLoading] = useState(false);
  const [movementsError, setMovementsError] = useState<string | null>(null);
  const [movementPage, setMovementPage] = useState(1);
  const [movementTotalPages, setMovementTotalPages] = useState(1);

  const activeIngredients = useMemo(
    () => ingredients.filter(ingredient => ingredient.is_active),
    [ingredients]
  );

  const activeUOMs = useMemo(() => uoms.filter(uom => uom.is_active), [uoms]);

  const selectedIngredient = useMemo(
    () => ingredients.find(ingredient => ingredient.id === form.ingredient_id),
    [ingredients, form.ingredient_id]
  );

  useEffect(() => {
    void fetchStockWorkspace();
  }, []);

  useEffect(() => {
    if (historyIngredientId) {
      void fetchMovements(historyIngredientId, movementPage);
    }
  }, [historyIngredientId, movementPage]);

  const fetchStockWorkspace = async () => {
    try {
      setLoading(true);
      setError(null);

      const [ingredientResponse, uomResponse, lowStockResponse, valuationResponse] =
        await Promise.all([
          inventory.getIngredients({ page: 1, limit: 100 }),
          inventory.getUOMs(false),
          inventory.getLowStockIngredients({ page: 1, limit: 8 }),
          inventory.getValuation(),
        ]);

      const nextIngredients = ingredientResponse.data;
      const nextUOMs = uomResponse;
      setIngredients(nextIngredients);
      setUOMs(nextUOMs);
      setLowStock(lowStockResponse.data);
      setValuation(toNumber(valuationResponse.valuation));

      const firstIngredient = nextIngredients.find(ingredient => ingredient.is_active);
      if (firstIngredient) {
        setForm(current => {
          const ingredient =
            nextIngredients.find(item => item.id === current.ingredient_id) || firstIngredient;
          return {
            ...current,
            ingredient_id: ingredient.id,
            uom_id: current.uom_id || ingredient.base_uom_id,
            unit_cost:
              current.unit_cost && current.unit_cost !== '0'
                ? current.unit_cost
                : decimalInputValue(ingredient.average_cost_per_base_unit),
          };
        });

        setHistoryIngredientId(current => current || firstIngredient.id);
      }
    } catch (err) {
      console.error('Failed to fetch stock workspace:', err);
      setError(getApiErrorMessage(err, 'Failed to load stock workspace.'));
    } finally {
      setLoading(false);
    }
  };

  const fetchMovements = async (ingredientId = historyIngredientId, page = movementPage) => {
    if (!ingredientId) {
      setMovements([]);
      return;
    }

    try {
      setMovementsLoading(true);
      setMovementsError(null);
      const response = await inventory.getIngredientMovements(ingredientId, {
        page,
        limit: 12,
      });
      setMovements(response.data);
      setMovementTotalPages(response.total_pages);
    } catch (err) {
      console.error('Failed to fetch ingredient movement history:', err);
      setMovementsError(getApiErrorMessage(err, 'Failed to load movement history.'));
    } finally {
      setMovementsLoading(false);
    }
  };

  const setIngredient = (ingredientId: string) => {
    const ingredient = ingredients.find(item => item.id === ingredientId);
    setForm(current => ({
      ...current,
      ingredient_id: ingredientId,
      uom_id: ingredient?.base_uom_id || current.uom_id,
      unit_cost: ingredient
        ? decimalInputValue(ingredient.average_cost_per_base_unit)
        : current.unit_cost,
    }));
  };

  const handleOperationChange = (nextOperation: StockOperation) => {
    setOperation(nextOperation);
    setFormError(null);
    setSuccessMessage(null);
  };

  const validateForm = () => {
    const quantity = Number(form.quantity);
    const unitCost = Number(form.unit_cost || '0');

    if (!form.ingredient_id || !form.uom_id) {
      return 'Ingredient and unit are required.';
    }
    if (!Number.isFinite(quantity) || quantity === 0) {
      return 'Quantity must not be zero.';
    }
    if (operation !== 'adjustment' && quantity < 0) {
      return 'Quantity must be greater than zero.';
    }
    if (!Number.isFinite(unitCost) || unitCost < 0) {
      return 'Unit cost must be zero or greater.';
    }
    if ((operation === 'adjustment' || operation === 'waste') && !form.reason.trim()) {
      return 'Reason is required.';
    }

    return null;
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const validationError = validateForm();
    if (validationError) {
      setFormError(validationError);
      return;
    }

    const quantity = Number(form.quantity);
    const unitCost = Number(form.unit_cost || '0');

    try {
      setSaving(true);
      setFormError(null);
      setSuccessMessage(null);

      if (operation === 'adjustment') {
        await inventory.recordAdjustment({
          ingredient_id: form.ingredient_id,
          quantity_delta: quantity,
          uom_id: form.uom_id,
          unit_cost: unitCost,
          reason: form.reason.trim(),
          occurred_at: form.occurred_at ? new Date(form.occurred_at).toISOString() : undefined,
        });
      } else {
        const payload: StockMutationRequest = {
          ingredient_id: form.ingredient_id,
          quantity,
          uom_id: form.uom_id,
          unit_cost: unitCost,
          ...buildOptionalStockFields(form),
        };

        if (operation === 'initial') {
          await inventory.recordInitialStock(payload);
        } else if (operation === 'purchase') {
          await inventory.recordPurchase(payload);
        } else {
          await inventory.recordWaste(payload);
        }
      }

      setSuccessMessage('Stock movement recorded.');
      setForm(current => ({
        ...current,
        quantity: '',
        reason: '',
        reference_type: '',
        reference_id: '',
        occurred_at: toDateTimeLocal(new Date()),
      }));
      await fetchStockWorkspace();
      if (historyIngredientId === form.ingredient_id) {
        await fetchMovements(form.ingredient_id, movementPage);
      } else {
        setHistoryIngredientId(form.ingredient_id);
        setMovementPage(1);
      }
    } catch (err) {
      console.error('Failed to record stock movement:', err);
      setFormError(getApiErrorMessage(err, 'Failed to record stock movement.'));
    } finally {
      setSaving(false);
    }
  };

  if (loading && ingredients.length === 0) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-sm text-gray-500">Loading stock...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {error && (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-gray-500">Inventory value</p>
              <p className="mt-2 text-2xl font-bold text-gray-900">
                {formatCurrency(valuation, false, 0)}
              </p>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-green-50 text-green-700">
              <Calculator className="h-5 w-5" aria-hidden="true" />
            </div>
          </div>
        </div>
        <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-gray-500">Low stock</p>
              <p className="mt-2 text-2xl font-bold text-gray-900">{lowStock.length}</p>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-yellow-50 text-yellow-800">
              <AlertTriangle className="h-5 w-5" aria-hidden="true" />
            </div>
          </div>
        </div>
        <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium text-gray-500">Tracked ingredients</p>
              <p className="mt-2 text-2xl font-bold text-gray-900">{activeIngredients.length}</p>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-blue-50 text-blue-700">
              <ClipboardList className="h-5 w-5" aria-hidden="true" />
            </div>
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[1fr_0.85fr]">
        <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <div className="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Record stock</h2>
              <p className="text-sm text-gray-500">
                Initial stock, purchases, adjustments, and waste.
              </p>
            </div>
            <button
              type="button"
              onClick={fetchStockWorkspace}
              disabled={loading}
              className="inline-flex items-center gap-2 self-start rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 md:self-auto"
            >
              <RefreshCw
                className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`}
                aria-hidden="true"
              />
              Refresh
            </button>
          </div>

          <div className="p-5">
            <div className="mb-5 grid grid-cols-2 gap-2 sm:grid-cols-4">
              {operationItems.map(item => (
                <button
                  key={item.value}
                  type="button"
                  onClick={() => handleOperationChange(item.value)}
                  className={`inline-flex items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-colors ${
                    operation === item.value
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'
                  }`}
                >
                  {item.icon}
                  {item.label}
                </button>
              ))}
            </div>

            {formError && (
              <div className="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                {formError}
              </div>
            )}
            {successMessage && (
              <div className="mb-4 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-700">
                {successMessage}
              </div>
            )}

            {activeIngredients.length === 0 || activeUOMs.length === 0 ? (
              <div className="rounded-md border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-800">
                Add active ingredients and units before recording stock.
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <div>
                    <label
                      htmlFor="stock-ingredient"
                      className="block text-sm font-medium text-gray-700"
                    >
                      Ingredient
                    </label>
                    <select
                      id="stock-ingredient"
                      value={form.ingredient_id}
                      onChange={event => setIngredient(event.target.value)}
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      {activeIngredients.map(ingredient => (
                        <option key={ingredient.id} value={ingredient.id}>
                          {ingredient.name} ({ingredient.sku})
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label htmlFor="stock-uom" className="block text-sm font-medium text-gray-700">
                      Unit
                    </label>
                    <select
                      id="stock-uom"
                      value={form.uom_id}
                      onChange={event =>
                        setForm(current => ({ ...current, uom_id: event.target.value }))
                      }
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      {activeUOMs.map(uom => (
                        <option key={uom.id} value={uom.id}>
                          {uom.name} ({uom.code})
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {selectedIngredient && (
                  <div className="rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700">
                    Current stock: {formatQuantity(selectedIngredient.current_stock_base)}{' '}
                    {selectedIngredient.base_uom_code ||
                      getUOMCode(uoms, selectedIngredient.base_uom_id)}
                  </div>
                )}

                <div className="grid gap-4 md:grid-cols-3">
                  <div>
                    <label
                      htmlFor="stock-quantity"
                      className="block text-sm font-medium text-gray-700"
                    >
                      {operation === 'adjustment' ? 'Quantity delta' : 'Quantity'}
                    </label>
                    <input
                      id="stock-quantity"
                      type="number"
                      step="0.0001"
                      min={operation === 'adjustment' ? undefined : '0.0001'}
                      value={form.quantity}
                      onChange={event =>
                        setForm(current => ({ ...current, quantity: event.target.value }))
                      }
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                  </div>

                  <div>
                    <label htmlFor="stock-cost" className="block text-sm font-medium text-gray-700">
                      Unit cost
                    </label>
                    <input
                      id="stock-cost"
                      type="number"
                      min="0"
                      step="0.01"
                      value={form.unit_cost}
                      onChange={event =>
                        setForm(current => ({ ...current, unit_cost: event.target.value }))
                      }
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                  </div>

                  <div>
                    <label
                      htmlFor="stock-occurred-at"
                      className="block text-sm font-medium text-gray-700"
                    >
                      Occurred at
                    </label>
                    <input
                      id="stock-occurred-at"
                      type="datetime-local"
                      value={form.occurred_at}
                      onChange={event =>
                        setForm(current => ({ ...current, occurred_at: event.target.value }))
                      }
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                  </div>
                </div>

                <div>
                  <label htmlFor="stock-reason" className="block text-sm font-medium text-gray-700">
                    Reason
                  </label>
                  <input
                    id="stock-reason"
                    type="text"
                    value={form.reason}
                    onChange={event =>
                      setForm(current => ({ ...current, reason: event.target.value }))
                    }
                    className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    placeholder={
                      operation === 'waste'
                        ? 'Expired, damaged, spill'
                        : operation === 'adjustment'
                          ? 'Cycle count, correction'
                          : 'Optional'
                    }
                  />
                </div>

                {operation !== 'adjustment' && (
                  <div className="grid gap-4 md:grid-cols-2">
                    <div>
                      <label
                        htmlFor="stock-reference-type"
                        className="block text-sm font-medium text-gray-700"
                      >
                        Reference type
                      </label>
                      <input
                        id="stock-reference-type"
                        type="text"
                        value={form.reference_type}
                        onChange={event =>
                          setForm(current => ({
                            ...current,
                            reference_type: event.target.value,
                          }))
                        }
                        className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                        placeholder="invoice"
                      />
                    </div>
                    <div>
                      <label
                        htmlFor="stock-reference-id"
                        className="block text-sm font-medium text-gray-700"
                      >
                        Reference ID
                      </label>
                      <input
                        id="stock-reference-id"
                        type="text"
                        value={form.reference_id}
                        onChange={event =>
                          setForm(current => ({ ...current, reference_id: event.target.value }))
                        }
                        className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                        placeholder="PO-001"
                      />
                    </div>
                  </div>
                )}

                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={saving}
                    className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {saving ? 'Recording...' : 'Record movement'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </section>

        <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 px-5 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Low-stock report</h2>
            <p className="text-sm text-gray-500">Current stock compared with minimum stock.</p>
          </div>
          {lowStock.length === 0 ? (
            <div className="px-5 py-10 text-center text-sm text-gray-500">
              No low-stock ingredients.
            </div>
          ) : (
            <div className="divide-y divide-gray-200">
              {lowStock.map(ingredient => {
                const status = getStockStatus(
                  ingredient.current_stock_base,
                  ingredient.minimum_stock_base,
                  ingredient.is_active
                );
                return (
                  <div
                    key={ingredient.id}
                    className="flex items-center justify-between gap-4 px-5 py-4"
                  >
                    <div>
                      <div className="font-medium text-gray-900">{ingredient.name}</div>
                      <div className="text-sm text-gray-500">
                        {formatQuantity(ingredient.current_stock_base)} /{' '}
                        {formatQuantity(ingredient.minimum_stock_base)}{' '}
                        {ingredient.base_uom_code || getUOMCode(uoms, ingredient.base_uom_id)}
                      </div>
                    </div>
                    <span
                      className={`rounded-full px-2.5 py-1 text-xs font-medium ${status.className}`}
                    >
                      {status.label}
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </section>
      </div>

      <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
        <div className="flex flex-col gap-4 border-b border-gray-200 px-5 py-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Movement history</h2>
            <p className="text-sm text-gray-500">Ingredient movement ledger.</p>
          </div>
          <select
            value={historyIngredientId}
            onChange={event => {
              setHistoryIngredientId(event.target.value);
              setMovementPage(1);
            }}
            className="w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500 md:w-80"
          >
            {ingredients.map(ingredient => (
              <option key={ingredient.id} value={ingredient.id}>
                {ingredient.name} ({ingredient.sku})
              </option>
            ))}
          </select>
        </div>

        {movementsError && (
          <div className="m-5 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {movementsError}
          </div>
        )}

        {movementsLoading ? (
          <div className="px-5 py-10 text-center text-sm text-gray-500">Loading movements...</div>
        ) : movements.length === 0 ? (
          <div className="px-5 py-10 text-center text-sm text-gray-500">No movement history.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Type
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Quantity
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Unit cost
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Total
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Reason
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Occurred
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {movements.map(movement => {
                  const ingredient = ingredients.find(item => item.id === movement.ingredient_id);
                  return (
                    <tr key={movement.id}>
                      <td className="whitespace-nowrap px-5 py-4">
                        <span
                          className={`rounded-full px-2.5 py-1 text-xs font-medium ${movementBadgeClass(
                            movement.movement_type
                          )}`}
                        >
                          {movementTypeLabel(movement.movement_type)}
                        </span>
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatQuantity(movement.quantity_base)}{' '}
                        {ingredient?.base_uom_code || getUOMCode(uoms, ingredient?.base_uom_id)}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatCurrency(toNumber(movement.unit_cost), false, 0)}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatCurrency(toNumber(movement.total_cost), false, 0)}
                      </td>
                      <td className="px-5 py-4 text-sm text-gray-700">
                        {movement.reason || movement.reference_id || '-'}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatDateTime(movement.occurred_at)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {movementTotalPages > 1 && (
          <div className="flex items-center justify-center gap-3 border-t border-gray-200 px-5 py-4">
            <button
              type="button"
              onClick={() => setMovementPage(current => Math.max(1, current - 1))}
              disabled={movementPage === 1}
              className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-sm text-gray-600">
              Page {movementPage} of {movementTotalPages}
            </span>
            <button
              type="button"
              onClick={() => setMovementPage(current => Math.min(movementTotalPages, current + 1))}
              disabled={movementPage === movementTotalPages}
              className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Next
            </button>
          </div>
        )}
      </section>
    </div>
  );
}
