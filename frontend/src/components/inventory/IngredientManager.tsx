'use client';

import { useEffect, useMemo, useState } from 'react';
import { Edit2, Link2, Plus, RefreshCw, Search, Trash2 } from 'lucide-react';
import Modal from '@/components/ui/Modal';
import { inventory } from '@/services/inventory';
import type {
  Ingredient,
  IngredientUOMConversion,
  UOM,
  UpdateIngredientRequest,
} from '@/types/inventory';
import { formatCurrency } from '@/utils/format';
import {
  decimalInputValue,
  formatQuantity,
  getApiErrorMessage,
  getStockStatus,
  getUOMCode,
  getUOMLabel,
  toNumber,
} from './inventoryUtils';

interface IngredientFormState {
  sku: string;
  name: string;
  base_uom_id: string;
  minimum_stock_base: string;
  average_cost_per_base_unit: string;
  track_stock: boolean;
  is_active: boolean;
}

interface ConversionFormState {
  from_uom_id: string;
  to_uom_id: string;
  multiplier: string;
}

const buildEmptyIngredientForm = (uoms: UOM[]): IngredientFormState => ({
  sku: '',
  name: '',
  base_uom_id: uoms.find(uom => uom.is_active)?.id || '',
  minimum_stock_base: '0',
  average_cost_per_base_unit: '0',
  track_stock: true,
  is_active: true,
});

const buildEmptyConversionForm = (uoms: UOM[], baseUOMId?: string): ConversionFormState => ({
  from_uom_id: uoms.find(uom => uom.is_active && uom.id !== baseUOMId)?.id || '',
  to_uom_id: baseUOMId || uoms.find(uom => uom.is_active)?.id || '',
  multiplier: '1',
});

export default function IngredientManager() {
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [uoms, setUOMs] = useState<UOM[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [appliedSearch, setAppliedSearch] = useState('');
  const [includeInactive, setIncludeInactive] = useState(false);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);
  const [uomsLoading, setUOMsLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [isIngredientModalOpen, setIsIngredientModalOpen] = useState(false);
  const [editingIngredient, setEditingIngredient] = useState<Ingredient | null>(null);
  const [ingredientForm, setIngredientForm] = useState<IngredientFormState>(
    buildEmptyIngredientForm([])
  );

  const [selectedIngredient, setSelectedIngredient] = useState<Ingredient | null>(null);
  const [conversions, setConversions] = useState<IngredientUOMConversion[]>([]);
  const [conversionsLoading, setConversionsLoading] = useState(false);
  const [conversionSaving, setConversionSaving] = useState(false);
  const [conversionDeletingId, setConversionDeletingId] = useState<string | null>(null);
  const [conversionError, setConversionError] = useState<string | null>(null);
  const [editingConversion, setEditingConversion] = useState<IngredientUOMConversion | null>(null);
  const [conversionForm, setConversionForm] = useState<ConversionFormState>(
    buildEmptyConversionForm([])
  );

  const activeUOMs = useMemo(() => uoms.filter(uom => uom.is_active), [uoms]);

  useEffect(() => {
    void fetchUOMs();
  }, []);

  useEffect(() => {
    void fetchIngredients();
  }, [page, includeInactive, appliedSearch]);

  const fetchUOMs = async () => {
    try {
      setUOMsLoading(true);
      const data = await inventory.getUOMs(true);
      setUOMs(data);
      setIngredientForm(current =>
        current.base_uom_id ? current : buildEmptyIngredientForm(data)
      );
    } catch (err) {
      console.error('Failed to fetch inventory UoMs:', err);
      setError(getApiErrorMessage(err, 'Failed to load units.'));
    } finally {
      setUOMsLoading(false);
    }
  };

  const fetchIngredients = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await inventory.getIngredients({
        page,
        limit: 20,
        search: appliedSearch,
        includeInactive,
      });
      setIngredients(response.data);
      setTotalPages(response.total_pages);
    } catch (err) {
      console.error('Failed to fetch inventory ingredients:', err);
      setError(getApiErrorMessage(err, 'Failed to load ingredients.'));
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPage(1);
    setAppliedSearch(searchQuery.trim());
  };

  const openCreateModal = () => {
    setEditingIngredient(null);
    setIngredientForm(buildEmptyIngredientForm(uoms));
    setFormError(null);
    setIsIngredientModalOpen(true);
  };

  const openEditModal = (ingredient: Ingredient) => {
    setEditingIngredient(ingredient);
    setIngredientForm({
      sku: ingredient.sku,
      name: ingredient.name,
      base_uom_id: ingredient.base_uom_id,
      minimum_stock_base: decimalInputValue(ingredient.minimum_stock_base),
      average_cost_per_base_unit: decimalInputValue(ingredient.average_cost_per_base_unit),
      track_stock: ingredient.track_stock,
      is_active: ingredient.is_active,
    });
    setFormError(null);
    setIsIngredientModalOpen(true);
  };

  const closeIngredientModal = () => {
    if (saving) {
      return;
    }
    setIsIngredientModalOpen(false);
  };

  const handleIngredientSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const sku = ingredientForm.sku.trim();
    const name = ingredientForm.name.trim();
    const minimumStock = Number(ingredientForm.minimum_stock_base);
    const averageCost = Number(ingredientForm.average_cost_per_base_unit);

    if (!sku || !name || !ingredientForm.base_uom_id) {
      setFormError('SKU, name, and base unit are required.');
      return;
    }
    if (!Number.isFinite(minimumStock) || minimumStock < 0) {
      setFormError('Minimum stock must be zero or greater.');
      return;
    }
    if (!Number.isFinite(averageCost) || averageCost < 0) {
      setFormError('Average cost must be zero or greater.');
      return;
    }

    try {
      setSaving(true);
      setFormError(null);

      const payload: UpdateIngredientRequest = {
        sku,
        name,
        base_uom_id: ingredientForm.base_uom_id,
        minimum_stock_base: minimumStock,
        average_cost_per_base_unit: averageCost,
        track_stock: ingredientForm.track_stock,
        is_active: ingredientForm.is_active,
      };

      if (editingIngredient) {
        await inventory.updateIngredient(editingIngredient.id, payload);
      } else {
        await inventory.createIngredient({
          sku,
          name,
          base_uom_id: ingredientForm.base_uom_id,
          minimum_stock_base: minimumStock,
          average_cost_per_base_unit: averageCost,
          track_stock: ingredientForm.track_stock,
          is_active: ingredientForm.is_active,
        });
      }

      setIsIngredientModalOpen(false);
      await fetchIngredients();
    } catch (err) {
      console.error('Failed to save inventory ingredient:', err);
      setFormError(getApiErrorMessage(err, 'Failed to save ingredient.'));
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteIngredient = async (ingredient: Ingredient) => {
    if (!window.confirm(`Delete ${ingredient.name}?`)) {
      return;
    }

    try {
      setDeletingId(ingredient.id);
      setError(null);
      await inventory.deleteIngredient(ingredient.id);
      await fetchIngredients();
    } catch (err) {
      console.error('Failed to delete inventory ingredient:', err);
      setError(getApiErrorMessage(err, 'Failed to delete ingredient.'));
    } finally {
      setDeletingId(null);
    }
  };

  const openConversionsModal = async (ingredient: Ingredient) => {
    setSelectedIngredient(ingredient);
    setEditingConversion(null);
    setConversionForm(buildEmptyConversionForm(uoms, ingredient.base_uom_id));
    setConversionError(null);
    setConversions([]);
    await fetchConversions(ingredient.id);
  };

  const closeConversionsModal = () => {
    if (conversionSaving) {
      return;
    }
    setSelectedIngredient(null);
  };

  const fetchConversions = async (ingredientId = selectedIngredient?.id) => {
    if (!ingredientId) {
      return;
    }

    try {
      setConversionsLoading(true);
      setConversionError(null);
      const data = await inventory.getIngredientConversions(ingredientId);
      setConversions(data);
    } catch (err) {
      console.error('Failed to fetch ingredient conversions:', err);
      setConversionError(getApiErrorMessage(err, 'Failed to load conversions.'));
    } finally {
      setConversionsLoading(false);
    }
  };

  const handleConversionSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!selectedIngredient) {
      return;
    }

    const multiplier = Number(conversionForm.multiplier);
    if (!conversionForm.from_uom_id || !conversionForm.to_uom_id) {
      setConversionError('From and to units are required.');
      return;
    }
    if (conversionForm.from_uom_id === conversionForm.to_uom_id) {
      setConversionError('Conversion units must be different.');
      return;
    }
    if (!Number.isFinite(multiplier) || multiplier <= 0) {
      setConversionError('Multiplier must be greater than zero.');
      return;
    }

    try {
      setConversionSaving(true);
      setConversionError(null);

      if (editingConversion) {
        await inventory.updateIngredientConversion(selectedIngredient.id, editingConversion.id, {
          from_uom_id: conversionForm.from_uom_id,
          to_uom_id: conversionForm.to_uom_id,
          multiplier,
        });
      } else {
        await inventory.createIngredientConversion(selectedIngredient.id, {
          from_uom_id: conversionForm.from_uom_id,
          to_uom_id: conversionForm.to_uom_id,
          multiplier,
        });
      }

      setEditingConversion(null);
      setConversionForm(buildEmptyConversionForm(uoms, selectedIngredient.base_uom_id));
      await fetchConversions(selectedIngredient.id);
    } catch (err) {
      console.error('Failed to save ingredient conversion:', err);
      setConversionError(getApiErrorMessage(err, 'Failed to save conversion.'));
    } finally {
      setConversionSaving(false);
    }
  };

  const editConversion = (conversion: IngredientUOMConversion) => {
    setEditingConversion(conversion);
    setConversionForm({
      from_uom_id: conversion.from_uom_id,
      to_uom_id: conversion.to_uom_id,
      multiplier: decimalInputValue(conversion.multiplier),
    });
    setConversionError(null);
  };

  const handleDeleteConversion = async (conversion: IngredientUOMConversion) => {
    if (!selectedIngredient || !window.confirm('Delete this conversion?')) {
      return;
    }

    try {
      setConversionDeletingId(conversion.id);
      setConversionError(null);
      await inventory.deleteIngredientConversion(selectedIngredient.id, conversion.id);
      await fetchConversions(selectedIngredient.id);
    } catch (err) {
      console.error('Failed to delete ingredient conversion:', err);
      setConversionError(getApiErrorMessage(err, 'Failed to delete conversion.'));
    } finally {
      setConversionDeletingId(null);
    }
  };

  if ((loading && ingredients.length === 0) || uomsLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-sm text-gray-500">Loading ingredients...</div>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <form onSubmit={handleSearch} className="flex flex-1 gap-2">
            <input
              type="text"
              value={searchQuery}
              onChange={event => setSearchQuery(event.target.value)}
              placeholder="Search ingredients"
              className="min-w-0 flex-1 rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
            <button
              type="submit"
              className="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-700"
            >
              <Search className="h-4 w-4" aria-hidden="true" />
              Search
            </button>
          </form>

          <div className="flex flex-wrap items-center gap-3">
            <label className="flex items-center gap-2 text-sm text-gray-700">
              <input
                type="checkbox"
                checked={includeInactive}
                onChange={event => {
                  setIncludeInactive(event.target.checked);
                  setPage(1);
                }}
                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              Show inactive
            </label>
            <button
              type="button"
              onClick={fetchIngredients}
              disabled={loading}
              className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <RefreshCw
                className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`}
                aria-hidden="true"
              />
              Refresh
            </button>
            <button
              type="button"
              onClick={openCreateModal}
              disabled={activeUOMs.length === 0}
              className="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Plus className="h-4 w-4" aria-hidden="true" />
              Add ingredient
            </button>
          </div>
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {activeUOMs.length === 0 && (
        <div className="rounded-md border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-800">
          Add an active unit before creating ingredients.
        </div>
      )}

      <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
        {ingredients.length === 0 ? (
          <div className="px-5 py-12 text-center text-sm text-gray-500">No ingredients found.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Ingredient
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Stock
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Minimum
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Average cost
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
                {ingredients.map(ingredient => {
                  const status = getStockStatus(
                    ingredient.current_stock_base,
                    ingredient.minimum_stock_base,
                    ingredient.is_active
                  );
                  const uomCode =
                    ingredient.base_uom_code || getUOMCode(uoms, ingredient.base_uom_id);

                  return (
                    <tr key={ingredient.id}>
                      <td className="whitespace-nowrap px-5 py-4">
                        <div className="font-medium text-gray-900">{ingredient.name}</div>
                        <div className="text-sm text-gray-500">{ingredient.sku}</div>
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatQuantity(ingredient.current_stock_base)} {uomCode}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatQuantity(ingredient.minimum_stock_base)} {uomCode}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                        {formatCurrency(toNumber(ingredient.average_cost_per_base_unit), false, 0)}
                      </td>
                      <td className="whitespace-nowrap px-5 py-4">
                        <div className="flex flex-wrap gap-2">
                          <span
                            className={`rounded-full px-2.5 py-1 text-xs font-medium ${status.className}`}
                          >
                            {status.label}
                          </span>
                          {!ingredient.track_stock && (
                            <span className="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700">
                              Untracked
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="whitespace-nowrap px-5 py-4 text-right">
                        <div className="flex justify-end gap-2">
                          <button
                            type="button"
                            onClick={() => openConversionsModal(ingredient)}
                            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50"
                            aria-label={`Manage conversions for ${ingredient.name}`}
                            title="Conversions"
                          >
                            <Link2 className="h-4 w-4" aria-hidden="true" />
                          </button>
                          <button
                            type="button"
                            onClick={() => openEditModal(ingredient)}
                            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50"
                            aria-label={`Edit ${ingredient.name}`}
                            title="Edit"
                          >
                            <Edit2 className="h-4 w-4" aria-hidden="true" />
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDeleteIngredient(ingredient)}
                            disabled={deletingId === ingredient.id}
                            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-200 text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-40"
                            aria-label={`Delete ${ingredient.name}`}
                            title="Delete"
                          >
                            <Trash2 className="h-4 w-4" aria-hidden="true" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3">
          <button
            type="button"
            onClick={() => setPage(current => Math.max(1, current - 1))}
            disabled={page === 1}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm text-gray-600">
            Page {page} of {totalPages}
          </span>
          <button
            type="button"
            onClick={() => setPage(current => Math.min(totalPages, current + 1))}
            disabled={page === totalPages}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}

      <Modal
        isOpen={isIngredientModalOpen}
        onClose={closeIngredientModal}
        title={editingIngredient ? 'Edit ingredient' : 'Add ingredient'}
        size="lg"
      >
        <form onSubmit={handleIngredientSubmit} className="space-y-4">
          {formError && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
              {formError}
            </div>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="ingredient-sku" className="block text-sm font-medium text-gray-700">
                SKU
              </label>
              <input
                id="ingredient-sku"
                type="text"
                value={ingredientForm.sku}
                onChange={event =>
                  setIngredientForm(current => ({ ...current, sku: event.target.value }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="ING-001"
              />
            </div>

            <div>
              <label htmlFor="ingredient-name" className="block text-sm font-medium text-gray-700">
                Name
              </label>
              <input
                id="ingredient-name"
                type="text"
                value={ingredientForm.name}
                onChange={event =>
                  setIngredientForm(current => ({ ...current, name: event.target.value }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                placeholder="Sugar"
              />
            </div>
          </div>

          <div>
            <label
              htmlFor="ingredient-base-uom"
              className="block text-sm font-medium text-gray-700"
            >
              Base unit
            </label>
            <select
              id="ingredient-base-uom"
              value={ingredientForm.base_uom_id}
              onChange={event =>
                setIngredientForm(current => ({ ...current, base_uom_id: event.target.value }))
              }
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="" disabled>
                Select a unit
              </option>
              {activeUOMs.map(uom => (
                <option key={uom.id} value={uom.id}>
                  {uom.name} ({uom.code})
                </option>
              ))}
            </select>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label
                htmlFor="ingredient-minimum"
                className="block text-sm font-medium text-gray-700"
              >
                Minimum stock
              </label>
              <input
                id="ingredient-minimum"
                type="number"
                min="0"
                step="0.0001"
                value={ingredientForm.minimum_stock_base}
                onChange={event =>
                  setIngredientForm(current => ({
                    ...current,
                    minimum_stock_base: event.target.value,
                  }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>

            <div>
              <label htmlFor="ingredient-cost" className="block text-sm font-medium text-gray-700">
                Average cost
              </label>
              <input
                id="ingredient-cost"
                type="number"
                min="0"
                step="0.01"
                value={ingredientForm.average_cost_per_base_unit}
                onChange={event =>
                  setIngredientForm(current => ({
                    ...current,
                    average_cost_per_base_unit: event.target.value,
                  }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700">
              <input
                type="checkbox"
                checked={ingredientForm.track_stock}
                onChange={event =>
                  setIngredientForm(current => ({ ...current, track_stock: event.target.checked }))
                }
                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              Track stock
            </label>
            <label className="flex items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700">
              <input
                type="checkbox"
                checked={ingredientForm.is_active}
                onChange={event =>
                  setIngredientForm(current => ({ ...current, is_active: event.target.checked }))
                }
                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              Active
            </label>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={closeIngredientModal}
              disabled={saving}
              className="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving ? 'Saving...' : 'Save ingredient'}
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={Boolean(selectedIngredient)}
        onClose={closeConversionsModal}
        title={selectedIngredient ? `${selectedIngredient.name} conversions` : 'Conversions'}
        size="xl"
      >
        {selectedIngredient && (
          <div className="grid gap-5 lg:grid-cols-[1fr_1.1fr]">
            <form onSubmit={handleConversionSubmit} className="space-y-4">
              {conversionError && (
                <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                  {conversionError}
                </div>
              )}

              <div>
                <label
                  htmlFor="conversion-from"
                  className="block text-sm font-medium text-gray-700"
                >
                  From unit
                </label>
                <select
                  id="conversion-from"
                  value={conversionForm.from_uom_id}
                  onChange={event =>
                    setConversionForm(current => ({ ...current, from_uom_id: event.target.value }))
                  }
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  <option value="" disabled>
                    Select a unit
                  </option>
                  {activeUOMs.map(uom => (
                    <option key={uom.id} value={uom.id}>
                      {uom.name} ({uom.code})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label htmlFor="conversion-to" className="block text-sm font-medium text-gray-700">
                  To unit
                </label>
                <select
                  id="conversion-to"
                  value={conversionForm.to_uom_id}
                  onChange={event =>
                    setConversionForm(current => ({ ...current, to_uom_id: event.target.value }))
                  }
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  <option value="" disabled>
                    Select a unit
                  </option>
                  {activeUOMs.map(uom => (
                    <option key={uom.id} value={uom.id}>
                      {uom.name} ({uom.code})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label
                  htmlFor="conversion-multiplier"
                  className="block text-sm font-medium text-gray-700"
                >
                  Multiplier
                </label>
                <input
                  id="conversion-multiplier"
                  type="number"
                  min="0.0001"
                  step="0.0001"
                  value={conversionForm.multiplier}
                  onChange={event =>
                    setConversionForm(current => ({ ...current, multiplier: event.target.value }))
                  }
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>

              <div className="flex justify-end gap-2">
                {editingConversion && (
                  <button
                    type="button"
                    onClick={() => {
                      setEditingConversion(null);
                      setConversionForm(
                        buildEmptyConversionForm(uoms, selectedIngredient.base_uom_id)
                      );
                    }}
                    disabled={conversionSaving}
                    className="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    New conversion
                  </button>
                )}
                <button
                  type="submit"
                  disabled={conversionSaving}
                  className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {conversionSaving
                    ? 'Saving...'
                    : editingConversion
                      ? 'Update conversion'
                      : 'Add conversion'}
                </button>
              </div>
            </form>

            <div className="rounded-lg border border-gray-200">
              <div className="border-b border-gray-200 px-4 py-3">
                <div className="text-sm font-medium text-gray-900">
                  Base: {getUOMLabel(uoms, selectedIngredient.base_uom_id)}
                </div>
              </div>
              {conversionsLoading ? (
                <div className="px-4 py-8 text-center text-sm text-gray-500">
                  Loading conversions...
                </div>
              ) : conversions.length === 0 ? (
                <div className="px-4 py-8 text-center text-sm text-gray-500">
                  No conversions configured.
                </div>
              ) : (
                <div className="divide-y divide-gray-200">
                  {conversions.map(conversion => (
                    <div
                      key={conversion.id}
                      className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div>
                        <div className="font-medium text-gray-900">
                          {getUOMLabel(uoms, conversion.from_uom_id)} to{' '}
                          {getUOMLabel(uoms, conversion.to_uom_id)}
                        </div>
                        <div className="text-sm text-gray-500">
                          Multiplier: {formatQuantity(conversion.multiplier, 4)}
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          onClick={() => editConversion(conversion)}
                          className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50"
                          aria-label="Edit conversion"
                          title="Edit"
                        >
                          <Edit2 className="h-4 w-4" aria-hidden="true" />
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDeleteConversion(conversion)}
                          disabled={conversionDeletingId === conversion.id}
                          className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-200 text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-40"
                          aria-label="Delete conversion"
                          title="Delete"
                        >
                          <Trash2 className="h-4 w-4" aria-hidden="true" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
