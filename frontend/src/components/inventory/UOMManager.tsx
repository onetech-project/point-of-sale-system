'use client';

import { useEffect, useMemo, useState } from 'react';
import { Edit2, Plus, RefreshCw, Trash2 } from 'lucide-react';
import Modal from '@/components/ui/Modal';
import { inventory } from '@/services/inventory';
import type { CreateUOMRequest, UOM, UOMCategory } from '@/types/inventory';
import { getApiErrorMessage, UOM_CATEGORIES } from './inventoryUtils';

interface UOMFormState {
  code: string;
  name: string;
  category: UOMCategory;
  is_base_unit: boolean;
  is_active: boolean;
}

const emptyForm: UOMFormState = {
  code: '',
  name: '',
  category: 'weight',
  is_base_unit: false,
  is_active: true,
};

export default function UOMManager() {
  const [uoms, setUOMs] = useState<UOM[]>([]);
  const [includeInactive, setIncludeInactive] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingUOM, setEditingUOM] = useState<UOM | null>(null);
  const [form, setForm] = useState<UOMFormState>(emptyForm);

  useEffect(() => {
    void fetchUOMs();
  }, [includeInactive]);

  const groupedUOMs = useMemo(() => {
    return UOM_CATEGORIES.map(category => ({
      category,
      uoms: uoms.filter(uom => uom.category === category),
    })).filter(group => group.uoms.length > 0);
  }, [uoms]);

  const fetchUOMs = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await inventory.getUOMs(includeInactive);
      setUOMs(data);
    } catch (err) {
      console.error('Failed to fetch inventory UoMs:', err);
      setError(getApiErrorMessage(err, 'Failed to load units.'));
    } finally {
      setLoading(false);
    }
  };

  const openCreateModal = () => {
    setEditingUOM(null);
    setForm(emptyForm);
    setFormError(null);
    setIsModalOpen(true);
  };

  const openEditModal = (uom: UOM) => {
    setEditingUOM(uom);
    setForm({
      code: uom.code,
      name: uom.name,
      category: uom.category,
      is_base_unit: uom.is_base_unit,
      is_active: uom.is_active,
    });
    setFormError(null);
    setIsModalOpen(true);
  };

  const closeModal = () => {
    if (saving) {
      return;
    }
    setIsModalOpen(false);
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmedName = form.name.trim();
    const trimmedCode = form.code.trim();
    if (!trimmedName || (!editingUOM && !trimmedCode)) {
      setFormError('Code and name are required.');
      return;
    }

    try {
      setSaving(true);
      setFormError(null);

      if (editingUOM) {
        await inventory.updateUOM(editingUOM.id, {
          name: trimmedName,
          category: form.category,
          is_base_unit: form.is_base_unit,
          is_active: form.is_active,
        });
      } else {
        const payload: CreateUOMRequest = {
          code: trimmedCode,
          name: trimmedName,
          category: form.category,
          is_base_unit: form.is_base_unit,
          is_active: form.is_active,
        };
        await inventory.createUOM(payload);
      }

      setIsModalOpen(false);
      await fetchUOMs();
    } catch (err) {
      console.error('Failed to save inventory UoM:', err);
      setFormError(getApiErrorMessage(err, 'Failed to save unit.'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (uom: UOM) => {
    if (!window.confirm(`Delete ${uom.name}?`)) {
      return;
    }

    try {
      setDeletingId(uom.id);
      setError(null);
      await inventory.deleteUOM(uom.id);
      await fetchUOMs();
    } catch (err) {
      console.error('Failed to delete inventory UoM:', err);
      setError(getApiErrorMessage(err, 'Failed to delete unit.'));
    } finally {
      setDeletingId(null);
    }
  };

  if (loading && uoms.length === 0) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-sm text-gray-500">Loading units...</div>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm md:flex-row md:items-center md:justify-between">
        <label className="flex items-center gap-2 text-sm text-gray-700">
          <input
            type="checkbox"
            checked={includeInactive}
            onChange={event => setIncludeInactive(event.target.checked)}
            className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          Show inactive units
        </label>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={fetchUOMs}
            disabled={loading}
            className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} aria-hidden="true" />
            Refresh
          </button>
          <button
            type="button"
            onClick={openCreateModal}
            className="inline-flex items-center gap-2 rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-700"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            Add unit
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
        {uoms.length === 0 ? (
          <div className="px-5 py-12 text-center text-sm text-gray-500">No units found.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Unit
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Category
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Source
                  </th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                    Flags
                  </th>
                  <th className="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {groupedUOMs.flatMap(group =>
                  group.uoms.map(uom => {
                    const isCustom = Boolean(uom.tenant_id);
                    return (
                      <tr key={uom.id}>
                        <td className="whitespace-nowrap px-5 py-4">
                          <div className="font-medium text-gray-900">{uom.name}</div>
                          <div className="text-sm text-gray-500">{uom.code}</div>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm capitalize text-gray-700">
                          {uom.category}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4">
                          <span
                            className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${
                              isCustom
                                ? 'bg-primary-50 text-primary-700'
                                : 'bg-gray-100 text-gray-700'
                            }`}
                          >
                            {isCustom ? 'Custom' : 'Global'}
                          </span>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4">
                          <div className="flex flex-wrap gap-2">
                            {uom.is_base_unit && (
                              <span className="rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-700">
                                Base
                              </span>
                            )}
                            <span
                              className={`rounded-full px-2.5 py-1 text-xs font-medium ${
                                uom.is_active
                                  ? 'bg-green-50 text-green-700'
                                  : 'bg-gray-100 text-gray-700'
                              }`}
                            >
                              {uom.is_active ? 'Active' : 'Inactive'}
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-right">
                          <div className="flex justify-end gap-2">
                            <button
                              type="button"
                              onClick={() => openEditModal(uom)}
                              disabled={!isCustom}
                              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-40"
                              aria-label={`Edit ${uom.name}`}
                              title={isCustom ? 'Edit' : 'Global units are read-only'}
                            >
                              <Edit2 className="h-4 w-4" aria-hidden="true" />
                            </button>
                            <button
                              type="button"
                              onClick={() => handleDelete(uom)}
                              disabled={!isCustom || deletingId === uom.id}
                              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-200 text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-40"
                              aria-label={`Delete ${uom.name}`}
                              title={isCustom ? 'Delete' : 'Global units are read-only'}
                            >
                              <Trash2 className="h-4 w-4" aria-hidden="true" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <Modal
        isOpen={isModalOpen}
        onClose={closeModal}
        title={editingUOM ? 'Edit unit' : 'Add unit'}
        size="md"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          {formError && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
              {formError}
            </div>
          )}

          <div>
            <label htmlFor="uom-code" className="block text-sm font-medium text-gray-700">
              Code
            </label>
            <input
              id="uom-code"
              type="text"
              value={form.code}
              onChange={event => setForm(current => ({ ...current, code: event.target.value }))}
              disabled={Boolean(editingUOM)}
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:bg-gray-100"
              placeholder="kg"
            />
          </div>

          <div>
            <label htmlFor="uom-name" className="block text-sm font-medium text-gray-700">
              Name
            </label>
            <input
              id="uom-name"
              type="text"
              value={form.name}
              onChange={event => setForm(current => ({ ...current, name: event.target.value }))}
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              placeholder="Kilogram"
            />
          </div>

          <div>
            <label htmlFor="uom-category" className="block text-sm font-medium text-gray-700">
              Category
            </label>
            <select
              id="uom-category"
              value={form.category}
              onChange={event =>
                setForm(current => ({ ...current, category: event.target.value as UOMCategory }))
              }
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {UOM_CATEGORIES.map(category => (
                <option key={category} value={category}>
                  {category.charAt(0).toUpperCase() + category.slice(1)}
                </option>
              ))}
            </select>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700">
              <input
                type="checkbox"
                checked={form.is_base_unit}
                onChange={event =>
                  setForm(current => ({ ...current, is_base_unit: event.target.checked }))
                }
                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              Base unit
            </label>
            <label className="flex items-center gap-2 rounded-md border border-gray-200 px-3 py-2 text-sm text-gray-700">
              <input
                type="checkbox"
                checked={form.is_active}
                onChange={event =>
                  setForm(current => ({ ...current, is_active: event.target.checked }))
                }
                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              Active
            </label>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={closeModal}
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
              {saving ? 'Saving...' : 'Save unit'}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
