'use client';

import React, { useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  Calculator,
  CheckCircle2,
  ClipboardList,
  Loader2,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react';
import { Product } from '@/types/product';
import {
  CreateRecipeRequest,
  Ingredient,
  Recipe,
  RecipeOverheadCalculationType,
  RecipeVersionSummary,
  UOM,
} from '@/types/recipes';
import { recipes as recipeService } from '@/services/recipes';
import { formatCurrency, formatNumber } from '@/utils/format';
import {
  getRecipeCogsRatio,
  getRecipeErrorMessage,
  isRecipeNotFound,
  toRecipeNumber,
} from './recipeUtils';

interface RecipeCostingPanelProps {
  product: Product;
}

type RecipeItemForm = {
  ingredient_id: string;
  quantity: string;
  uom_id: string;
  waste_percentage: string;
};

type RecipeOverheadForm = {
  name: string;
  calculation_type: RecipeOverheadCalculationType;
  amount: string;
};

type RecipeFormState = {
  yield_quantity: string;
  effective_from: string;
  items: RecipeItemForm[];
  overheads: RecipeOverheadForm[];
};

const todayInputValue = () => new Date().toISOString().slice(0, 10);

const emptyItem = (): RecipeItemForm => ({
  ingredient_id: '',
  quantity: '1',
  uom_id: '',
  waste_percentage: '0',
});

const emptyOverhead = (): RecipeOverheadForm => ({
  name: '',
  calculation_type: 'FIXED',
  amount: '0',
});

const emptyForm = (): RecipeFormState => ({
  yield_quantity: '1',
  effective_from: todayInputValue(),
  items: [emptyItem()],
  overheads: [],
});

const formFromRecipe = (recipe: Recipe | null): RecipeFormState => {
  if (!recipe) {
    return emptyForm();
  }

  return {
    yield_quantity: String(toRecipeNumber(recipe.yield_quantity) || 1),
    effective_from: todayInputValue(),
    items:
      recipe.items.length > 0
        ? recipe.items.map(item => ({
            ingredient_id: item.ingredient_id,
            quantity: String(toRecipeNumber(item.quantity) || 1),
            uom_id: item.uom_id,
            waste_percentage: String(toRecipeNumber(item.waste_percentage)),
          }))
        : [emptyItem()],
    overheads: recipe.overheads.map(overhead => ({
      name: overhead.name,
      calculation_type: overhead.calculation_type,
      amount: String(toRecipeNumber(overhead.amount)),
    })),
  };
};

const formatPercent = (value: number) => `${formatNumber(value, 1)}%`;

const decimalInputValue = (value: string) => {
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : NaN;
};

export default function RecipeCostingPanel({ product }: RecipeCostingPanelProps) {
  const [activeRecipe, setActiveRecipe] = useState<Recipe | null>(null);
  const [selectedRecipe, setSelectedRecipe] = useState<Recipe | null>(null);
  const [versions, setVersions] = useState<RecipeVersionSummary[]>([]);
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [uoms, setUoms] = useState<UOM[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingVersion, setLoadingVersion] = useState(false);
  const [loadingReferences, setLoadingReferences] = useState(true);
  const [missingRecipe, setMissingRecipe] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [referenceError, setReferenceError] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [form, setForm] = useState<RecipeFormState>(() => emptyForm());

  const ingredientById = useMemo(
    () => new Map(ingredients.map(ingredient => [ingredient.id, ingredient])),
    [ingredients]
  );

  const selectedCost = selectedRecipe?.cost || null;
  const cogsRatio = getRecipeCogsRatio(selectedCost, product.selling_price);

  useEffect(() => {
    loadRecipeData();
    loadReferences();
  }, [product.id]);

  const loadRecipeData = async () => {
    try {
      setLoading(true);
      setError(null);
      setMissingRecipe(false);

      const [activeResult, versionsResult] = await Promise.allSettled([
        recipeService.getActiveRecipe(product.id),
        recipeService.listRecipeVersions(product.id),
      ]);

      if (versionsResult.status === 'fulfilled') {
        setVersions(versionsResult.value);
      } else {
        setVersions([]);
      }

      if (activeResult.status === 'fulfilled') {
        setActiveRecipe(activeResult.value);
        setSelectedRecipe(activeResult.value);
        return;
      }

      if (isRecipeNotFound(activeResult.reason)) {
        setActiveRecipe(null);
        setSelectedRecipe(null);
        setMissingRecipe(true);
        return;
      }

      throw activeResult.reason;
    } catch (err) {
      console.error('Failed to load recipe costing:', err);
      setError(getRecipeErrorMessage(err, 'Failed to load recipe costing.'));
    } finally {
      setLoading(false);
    }
  };

  const loadReferences = async () => {
    try {
      setLoadingReferences(true);
      setReferenceError(null);
      const [ingredientResponse, uomResponse] = await Promise.all([
        recipeService.listIngredients({ limit: 100 }),
        recipeService.listUOMs(),
      ]);
      setIngredients(ingredientResponse.ingredients);
      setUoms(uomResponse);
    } catch (err) {
      console.error('Failed to load recipe references:', err);
      setReferenceError(getRecipeErrorMessage(err, 'Failed to load ingredients or units.'));
    } finally {
      setLoadingReferences(false);
    }
  };

  const handleRefresh = () => {
    loadRecipeData();
    loadReferences();
  };

  const handleStartCreate = () => {
    setForm(formFromRecipe(activeRecipe));
    setFormError(null);
    setIsCreating(true);
  };

  const handleVersionChange = async (event: React.ChangeEvent<HTMLSelectElement>) => {
    const version = Number.parseInt(event.target.value, 10);
    if (!Number.isFinite(version)) {
      return;
    }

    if (activeRecipe?.version === version) {
      setSelectedRecipe(activeRecipe);
      return;
    }

    try {
      setLoadingVersion(true);
      setError(null);
      const recipe = await recipeService.getRecipeVersion(product.id, version);
      setSelectedRecipe(recipe);
    } catch (err) {
      console.error('Failed to load recipe version:', err);
      setError(getRecipeErrorMessage(err, 'Failed to load recipe version.'));
    } finally {
      setLoadingVersion(false);
    }
  };

  const updateItem = (index: number, nextItem: Partial<RecipeItemForm>) => {
    setForm(current => ({
      ...current,
      items: current.items.map((item, itemIndex) =>
        itemIndex === index ? { ...item, ...nextItem } : item
      ),
    }));
  };

  const updateOverhead = (index: number, nextOverhead: Partial<RecipeOverheadForm>) => {
    setForm(current => ({
      ...current,
      overheads: current.overheads.map((overhead, overheadIndex) =>
        overheadIndex === index ? { ...overhead, ...nextOverhead } : overhead
      ),
    }));
  };

  const validateForm = () => {
    const yieldQuantity = decimalInputValue(form.yield_quantity);
    if (!Number.isFinite(yieldQuantity) || yieldQuantity <= 0) {
      return 'Yield quantity must be greater than zero.';
    }

    if (form.items.length === 0) {
      return 'Add at least one recipe item.';
    }

    for (const [index, item] of form.items.entries()) {
      const quantity = decimalInputValue(item.quantity);
      const waste = decimalInputValue(item.waste_percentage);
      if (!item.ingredient_id) {
        return `Select an ingredient for item ${index + 1}.`;
      }
      if (!item.uom_id) {
        return `Select a unit for item ${index + 1}.`;
      }
      if (!Number.isFinite(quantity) || quantity <= 0) {
        return `Quantity must be greater than zero for item ${index + 1}.`;
      }
      if (!Number.isFinite(waste) || waste < 0) {
        return `Waste percentage cannot be negative for item ${index + 1}.`;
      }
    }

    for (const [index, overhead] of form.overheads.entries()) {
      const amount = decimalInputValue(overhead.amount);
      if (!overhead.name.trim()) {
        return `Enter a name for overhead ${index + 1}.`;
      }
      if (!Number.isFinite(amount) || amount < 0) {
        return `Overhead amount cannot be negative for overhead ${index + 1}.`;
      }
    }

    return null;
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();

    const validationError = validateForm();
    if (validationError) {
      setFormError(validationError);
      return;
    }

    const payload: CreateRecipeRequest = {
      yield_quantity: decimalInputValue(form.yield_quantity),
      effective_from: form.effective_from
        ? new Date(`${form.effective_from}T00:00:00`).toISOString()
        : undefined,
      items: form.items.map(item => ({
        ingredient_id: item.ingredient_id,
        quantity: decimalInputValue(item.quantity),
        uom_id: item.uom_id,
        waste_percentage: decimalInputValue(item.waste_percentage),
      })),
      overheads: form.overheads.map(overhead => ({
        name: overhead.name.trim(),
        calculation_type: overhead.calculation_type,
        amount: decimalInputValue(overhead.amount),
      })),
    };

    try {
      setSaving(true);
      setFormError(null);
      const created = await recipeService.createRecipeVersion(product.id, payload);
      setActiveRecipe(created);
      setSelectedRecipe(created);
      setMissingRecipe(false);
      setIsCreating(false);
      setVersions(await recipeService.listRecipeVersions(product.id));
    } catch (err) {
      console.error('Failed to create recipe version:', err);
      setFormError(getRecipeErrorMessage(err, 'Failed to create recipe version.'));
    } finally {
      setSaving(false);
    }
  };

  const handleIngredientChange = (index: number, ingredientId: string) => {
    const ingredient = ingredientById.get(ingredientId);
    updateItem(index, {
      ingredient_id: ingredientId,
      uom_id: ingredient?.base_uom_id || form.items[index]?.uom_id || '',
    });
  };

  const selectedVersionValue = selectedRecipe?.version ? String(selectedRecipe.version) : '';

  return (
    <section className="bg-white shadow rounded-lg p-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <Calculator className="h-5 w-5 text-primary-600" aria-hidden="true" />
            <h2 className="text-lg font-semibold text-gray-900">Recipe costing</h2>
          </div>
          <p className="mt-1 text-sm text-gray-500">
            Track active recipe cost, COGS, gross profit, and margin for this product.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handleRefresh}
            disabled={loading || loadingReferences}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50"
          >
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Refresh
          </button>
          <button
            type="button"
            onClick={handleStartCreate}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            New version
          </button>
        </div>
      </div>

      {error && (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {referenceError && (
        <div className="mt-4 rounded-md border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800">
          {referenceError}
        </div>
      )}

      {loading ? (
        <div className="mt-6 flex items-center justify-center py-10 text-sm text-gray-500">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" aria-hidden="true" />
          Loading recipe costing...
        </div>
      ) : missingRecipe && !selectedRecipe ? (
        <div className="mt-6 rounded-md border border-dashed border-gray-300 p-6 text-center">
          <ClipboardList className="mx-auto h-10 w-10 text-gray-400" aria-hidden="true" />
          <h3 className="mt-3 text-sm font-semibold text-gray-900">No active recipe</h3>
          <p className="mt-1 text-sm text-gray-500">
            Add ingredients and overheads to calculate COGS for {product.name}.
          </p>
          <button
            type="button"
            onClick={handleStartCreate}
            className="mt-4 inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            Create first version
          </button>
        </div>
      ) : selectedRecipe ? (
        <div className="mt-6 space-y-6">
          <div className="flex flex-col gap-3 rounded-md border border-gray-200 bg-gray-50 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2 text-sm text-gray-700">
              {selectedRecipe.is_active ? (
                <CheckCircle2 className="h-4 w-4 text-green-600" aria-hidden="true" />
              ) : (
                <AlertTriangle className="h-4 w-4 text-yellow-600" aria-hidden="true" />
              )}
              <span className="font-medium">
                Version {selectedRecipe.version}
                {selectedRecipe.is_active ? ' (active)' : ''}
              </span>
              <span className="text-gray-500">
                Effective {new Date(selectedRecipe.effective_from).toLocaleDateString()}
              </span>
            </div>
            {versions.length > 0 && (
              <label className="flex items-center gap-2 text-sm text-gray-600">
                Version
                <select
                  value={selectedVersionValue}
                  onChange={handleVersionChange}
                  disabled={loadingVersion}
                  className="rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  {versions.map(version => (
                    <option key={version.id} value={version.version}>
                      v{version.version}
                      {version.is_active ? ' active' : ''}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </div>

          {loadingVersion ? (
            <div className="flex items-center justify-center py-8 text-sm text-gray-500">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" aria-hidden="true" />
              Loading version...
            </div>
          ) : (
            <>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <MetricBox
                  label="COGS"
                  value={formatCurrency(toRecipeNumber(selectedRecipe.cost.total_cogs), false, 0)}
                  helper={`${formatPercent(cogsRatio)} of selling price`}
                  tone={cogsRatio >= 70 ? 'danger' : cogsRatio >= 55 ? 'warning' : 'neutral'}
                />
                <MetricBox
                  label="Gross profit"
                  value={formatCurrency(toRecipeNumber(selectedRecipe.cost.gross_profit), false, 0)}
                  helper="Per sellable unit"
                  tone={toRecipeNumber(selectedRecipe.cost.gross_profit) < 0 ? 'danger' : 'neutral'}
                />
                <MetricBox
                  label="Gross margin"
                  value={formatPercent(toRecipeNumber(selectedRecipe.cost.gross_margin_percentage))}
                  helper="After recipe cost"
                  tone={
                    toRecipeNumber(selectedRecipe.cost.gross_margin_percentage) < 30
                      ? 'danger'
                      : 'neutral'
                  }
                />
                <MetricBox
                  label="Selling price"
                  value={formatCurrency(
                    toRecipeNumber(selectedRecipe.cost.selling_price),
                    false,
                    0
                  )}
                  helper={`Yield ${formatNumber(toRecipeNumber(selectedRecipe.yield_quantity), 2)}`}
                />
              </div>

              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                <CostLine label="Material" value={selectedRecipe.cost.material_cost} />
                <CostLine label="Fixed overhead" value={selectedRecipe.cost.fixed_overhead_cost} />
                <CostLine
                  label="Percentage overhead"
                  value={selectedRecipe.cost.percentage_overhead_cost}
                />
              </div>

              <div>
                <h3 className="text-sm font-semibold text-gray-900">Recipe items</h3>
                <div className="mt-3 overflow-x-auto rounded-md border border-gray-200">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead className="bg-gray-50">
                      <tr>
                        <TableHead>Ingredient</TableHead>
                        <TableHead>Quantity</TableHead>
                        <TableHead>Waste</TableHead>
                        <TableHead>Ingredient cost</TableHead>
                        <TableHead>Cost with waste</TableHead>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 bg-white">
                      {selectedRecipe.items.map(item => (
                        <tr key={item.id}>
                          <TableCell>
                            <div className="font-medium text-gray-900">
                              {item.ingredient_name || 'Ingredient'}
                            </div>
                            <div className="text-xs text-gray-500">{item.uom_code || 'Unit'}</div>
                          </TableCell>
                          <TableCell>
                            {formatNumber(toRecipeNumber(item.quantity), 3)} {item.uom_code || ''}
                          </TableCell>
                          <TableCell>
                            {formatPercent(toRecipeNumber(item.waste_percentage))}
                          </TableCell>
                          <TableCell>
                            {formatCurrency(toRecipeNumber(item.ingredient_cost), false, 0)}
                          </TableCell>
                          <TableCell>
                            {formatCurrency(toRecipeNumber(item.cost_with_waste), false, 0)}
                          </TableCell>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>

              <div>
                <h3 className="text-sm font-semibold text-gray-900">Overheads</h3>
                {selectedRecipe.overheads.length > 0 ? (
                  <div className="mt-3 overflow-x-auto rounded-md border border-gray-200">
                    <table className="min-w-full divide-y divide-gray-200 text-sm">
                      <thead className="bg-gray-50">
                        <tr>
                          <TableHead>Name</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead>Amount</TableHead>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 bg-white">
                        {selectedRecipe.overheads.map(overhead => (
                          <tr key={overhead.id}>
                            <TableCell>{overhead.name}</TableCell>
                            <TableCell>
                              {overhead.calculation_type === 'FIXED' ? 'Fixed' : 'Percentage'}
                            </TableCell>
                            <TableCell>
                              {overhead.calculation_type === 'FIXED'
                                ? formatCurrency(toRecipeNumber(overhead.amount), false, 0)
                                : formatPercent(toRecipeNumber(overhead.amount))}
                            </TableCell>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <p className="mt-2 text-sm text-gray-500">No overheads on this version.</p>
                )}
              </div>
            </>
          )}
        </div>
      ) : null}

      {isCreating && (
        <form onSubmit={handleSubmit} className="mt-6 border-t border-gray-200 pt-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h3 className="text-base font-semibold text-gray-900">Create recipe version</h3>
              <p className="mt-1 text-sm text-gray-500">
                Saving a new version makes it the active recipe for future costing.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setIsCreating(false)}
              disabled={saving}
              className="text-sm font-medium text-gray-600 hover:text-gray-900 disabled:opacity-50"
            >
              Cancel
            </button>
          </div>

          {formError && (
            <div className="mt-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
              {formError}
            </div>
          )}

          {loadingReferences && (
            <div className="mt-4 flex items-center text-sm text-gray-500">
              <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />
              Loading ingredients and units...
            </div>
          )}

          <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-gray-700">
              Yield quantity
              <input
                type="number"
                min="0.0001"
                step="0.0001"
                value={form.yield_quantity}
                onChange={event =>
                  setForm(current => ({ ...current, yield_quantity: event.target.value }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </label>
            <label className="block text-sm font-medium text-gray-700">
              Effective from
              <input
                type="date"
                value={form.effective_from}
                onChange={event =>
                  setForm(current => ({ ...current, effective_from: event.target.value }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </label>
          </div>

          <div className="mt-6">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-semibold text-gray-900">Items</h4>
              <button
                type="button"
                onClick={() =>
                  setForm(current => ({
                    ...current,
                    items: [...current.items, emptyItem()],
                  }))
                }
                className="inline-flex items-center gap-2 text-sm font-medium text-primary-700 hover:text-primary-800"
              >
                <Plus className="h-4 w-4" aria-hidden="true" />
                Add item
              </button>
            </div>
            <div className="mt-3 space-y-3">
              {form.items.map((item, index) => (
                <div
                  key={index}
                  className="grid grid-cols-1 gap-3 rounded-md border border-gray-200 p-3 md:grid-cols-[2fr_1fr_1fr_1fr_auto]"
                >
                  <label className="block text-sm font-medium text-gray-700">
                    Ingredient
                    <select
                      value={item.ingredient_id}
                      onChange={event => handleIngredientChange(index, event.target.value)}
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      <option value="">Select ingredient</option>
                      {ingredients.map(ingredient => (
                        <option key={ingredient.id} value={ingredient.id}>
                          {ingredient.name} ({ingredient.sku})
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="block text-sm font-medium text-gray-700">
                    Quantity
                    <input
                      type="number"
                      min="0.0001"
                      step="0.0001"
                      value={item.quantity}
                      onChange={event => updateItem(index, { quantity: event.target.value })}
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                  </label>
                  <label className="block text-sm font-medium text-gray-700">
                    Unit
                    <select
                      value={item.uom_id}
                      onChange={event => updateItem(index, { uom_id: event.target.value })}
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    >
                      <option value="">Select unit</option>
                      {uoms.map(uom => (
                        <option key={uom.id} value={uom.id}>
                          {uom.code} - {uom.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="block text-sm font-medium text-gray-700">
                    Waste %
                    <input
                      type="number"
                      min="0"
                      step="0.01"
                      value={item.waste_percentage}
                      onChange={event =>
                        updateItem(index, { waste_percentage: event.target.value })
                      }
                      className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                  </label>
                  <button
                    type="button"
                    onClick={() =>
                      setForm(current => ({
                        ...current,
                        items:
                          current.items.length > 1
                            ? current.items.filter((_, itemIndex) => itemIndex !== index)
                            : current.items,
                      }))
                    }
                    disabled={form.items.length <= 1}
                    className="inline-flex h-10 w-10 items-center justify-center self-end rounded-md border border-gray-300 text-gray-500 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-40"
                    aria-label={`Remove item ${index + 1}`}
                  >
                    <Trash2 className="h-4 w-4" aria-hidden="true" />
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div className="mt-6">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-semibold text-gray-900">Overheads</h4>
              <button
                type="button"
                onClick={() =>
                  setForm(current => ({
                    ...current,
                    overheads: [...current.overheads, emptyOverhead()],
                  }))
                }
                className="inline-flex items-center gap-2 text-sm font-medium text-primary-700 hover:text-primary-800"
              >
                <Plus className="h-4 w-4" aria-hidden="true" />
                Add overhead
              </button>
            </div>
            {form.overheads.length === 0 ? (
              <p className="mt-2 text-sm text-gray-500">No overheads added.</p>
            ) : (
              <div className="mt-3 space-y-3">
                {form.overheads.map((overhead, index) => (
                  <div
                    key={index}
                    className="grid grid-cols-1 gap-3 rounded-md border border-gray-200 p-3 md:grid-cols-[2fr_1fr_1fr_auto]"
                  >
                    <label className="block text-sm font-medium text-gray-700">
                      Name
                      <input
                        type="text"
                        value={overhead.name}
                        onChange={event => updateOverhead(index, { name: event.target.value })}
                        placeholder="Packaging, labor, platform fee"
                        className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                      />
                    </label>
                    <label className="block text-sm font-medium text-gray-700">
                      Type
                      <select
                        value={overhead.calculation_type}
                        onChange={event =>
                          updateOverhead(index, {
                            calculation_type: event.target.value as RecipeOverheadCalculationType,
                          })
                        }
                        className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                      >
                        <option value="FIXED">Fixed</option>
                        <option value="PERCENTAGE">Percentage</option>
                      </select>
                    </label>
                    <label className="block text-sm font-medium text-gray-700">
                      Amount {overhead.calculation_type === 'PERCENTAGE' ? '(%)' : '(IDR)'}
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={overhead.amount}
                        onChange={event => updateOverhead(index, { amount: event.target.value })}
                        className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                      />
                    </label>
                    <button
                      type="button"
                      onClick={() =>
                        setForm(current => ({
                          ...current,
                          overheads: current.overheads.filter(
                            (_, overheadIndex) => overheadIndex !== index
                          ),
                        }))
                      }
                      className="inline-flex h-10 w-10 items-center justify-center self-end rounded-md border border-gray-300 text-gray-500 hover:bg-gray-50"
                      aria-label={`Remove overhead ${index + 1}`}
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="mt-6 flex justify-end">
            <button
              type="submit"
              disabled={saving || loadingReferences}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded-md hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
              Save recipe version
            </button>
          </div>
        </form>
      )}
    </section>
  );
}

function MetricBox({
  label,
  value,
  helper,
  tone = 'neutral',
}: {
  label: string;
  value: string;
  helper: string;
  tone?: 'neutral' | 'warning' | 'danger';
}) {
  const toneClass =
    tone === 'danger'
      ? 'border-red-200 bg-red-50 text-red-900'
      : tone === 'warning'
        ? 'border-yellow-200 bg-yellow-50 text-yellow-900'
        : 'border-gray-200 bg-white text-gray-900';

  return (
    <div className={`rounded-md border p-4 ${toneClass}`}>
      <div className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</div>
      <div className="mt-2 text-xl font-semibold">{value}</div>
      <div className="mt-1 text-xs text-gray-500">{helper}</div>
    </div>
  );
}

function CostLine({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="flex items-center justify-between rounded-md border border-gray-200 px-4 py-3 text-sm">
      <span className="text-gray-600">{label}</span>
      <span className="font-medium text-gray-900">
        {formatCurrency(toRecipeNumber(value), false, 0)}
      </span>
    </div>
  );
}

function TableHead({ children }: { children: React.ReactNode }) {
  return (
    <th
      scope="col"
      className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500"
    >
      {children}
    </th>
  );
}

function TableCell({ children }: { children: React.ReactNode }) {
  return <td className="whitespace-nowrap px-4 py-3 text-gray-700">{children}</td>;
}
