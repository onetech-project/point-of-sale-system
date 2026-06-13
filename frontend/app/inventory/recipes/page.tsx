'use client';

import React, { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  AlertTriangle,
  ArrowRight,
  Calculator,
  CheckCircle2,
  Loader2,
  RefreshCw,
} from 'lucide-react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import { ROLES } from '@/constants/roles';
import { product as productService } from '@/services/product';
import { recipes as recipeService } from '@/services/recipes';
import { Product } from '@/types/product';
import { Recipe } from '@/types/recipes';
import { formatCurrency, formatNumber } from '@/utils/format';
import {
  getRecipeCogsRatio,
  getRecipeErrorMessage,
  isRecipeNotFound,
  toRecipeNumber,
} from '@/components/recipes/recipeUtils';

type RecipeOverviewStatus = 'missing' | 'high_cogs' | 'ok' | 'error';

interface RecipeOverviewRow {
  product: Product;
  recipe: Recipe | null;
  status: RecipeOverviewStatus;
  cogsRatio: number;
  error?: string;
}

const HIGH_COGS_THRESHOLD = 70;
const PRODUCT_SCAN_LIMIT = 100;

export default function RecipeOverviewPage() {
  const [rows, setRows] = useState<RecipeOverviewRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadOverview();
  }, []);

  const loadOverview = async () => {
    try {
      setLoading(true);
      setError(null);

      const productResponse = await productService.getProducts({
        limit: PRODUCT_SCAN_LIMIT,
        archived: false,
      });

      const nextRows = await Promise.all(
        productResponse.data.map(async product => {
          try {
            const recipe = await recipeService.getActiveRecipe(product.id);
            const cogsRatio = getRecipeCogsRatio(recipe.cost, product.selling_price);

            return {
              product,
              recipe,
              cogsRatio,
              status: cogsRatio >= HIGH_COGS_THRESHOLD ? 'high_cogs' : 'ok',
            } satisfies RecipeOverviewRow;
          } catch (err) {
            if (isRecipeNotFound(err)) {
              return {
                product,
                recipe: null,
                cogsRatio: 0,
                status: 'missing',
              } satisfies RecipeOverviewRow;
            }

            return {
              product,
              recipe: null,
              cogsRatio: 0,
              status: 'error',
              error: getRecipeErrorMessage(err, 'Unable to load recipe.'),
            } satisfies RecipeOverviewRow;
          }
        })
      );

      setRows(nextRows);
    } catch (err) {
      console.error('Failed to load recipe overview:', err);
      setError(getRecipeErrorMessage(err, 'Failed to load recipe overview.'));
    } finally {
      setLoading(false);
    }
  };

  const missingRows = useMemo(() => rows.filter(row => row.status === 'missing'), [rows]);
  const highCogsRows = useMemo(() => rows.filter(row => row.status === 'high_cogs'), [rows]);
  const errorRows = useMemo(() => rows.filter(row => row.status === 'error'), [rows]);
  const healthyRows = useMemo(() => rows.filter(row => row.status === 'ok'), [rows]);

  return (
    <ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
      <DashboardLayout>
        <div className="mx-auto max-w-7xl">
          <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div className="flex items-center gap-2">
                <Calculator className="h-6 w-6 text-primary-600" aria-hidden="true" />
                <h1 className="text-3xl font-bold text-gray-900">Recipe costing</h1>
              </div>
              <p className="mt-1 text-sm text-gray-500">
                Scan active products for missing recipes and recipes with COGS at or above{' '}
                {HIGH_COGS_THRESHOLD}% of selling price.
              </p>
            </div>
            <button
              type="button"
              onClick={loadOverview}
              disabled={loading}
              className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Refresh
            </button>
          </div>

          {error && (
            <div className="mb-6 rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {loading ? (
            <div className="flex items-center justify-center rounded-lg bg-white py-16 text-sm text-gray-500 shadow">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" aria-hidden="true" />
              Loading recipe overview...
            </div>
          ) : (
            <div className="space-y-6">
              <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
                <SummaryBox label="Products scanned" value={rows.length} />
                <SummaryBox label="Missing recipes" value={missingRows.length} tone="danger" />
                <SummaryBox label="High COGS" value={highCogsRows.length} tone="warning" />
                <SummaryBox label="Within target" value={healthyRows.length} tone="success" />
              </div>

              {errorRows.length > 0 && (
                <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-900">
                  {errorRows.length} product{errorRows.length === 1 ? '' : 's'} could not be
                  checked. Use the product detail page to retry individual recipe costing.
                </div>
              )}

              <RecipeTable
                title="Missing recipes"
                emptyMessage="Every scanned active product has an active recipe."
                rows={missingRows}
                mode="missing"
              />

              <RecipeTable
                title="High COGS"
                emptyMessage="No scanned recipes are above the COGS threshold."
                rows={highCogsRows}
                mode="cost"
              />
            </div>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}

function SummaryBox({
  label,
  value,
  tone = 'neutral',
}: {
  label: string;
  value: number;
  tone?: 'neutral' | 'warning' | 'danger' | 'success';
}) {
  const toneClass =
    tone === 'danger'
      ? 'border-red-200 bg-red-50 text-red-900'
      : tone === 'warning'
        ? 'border-yellow-200 bg-yellow-50 text-yellow-900'
        : tone === 'success'
          ? 'border-green-200 bg-green-50 text-green-900'
          : 'border-gray-200 bg-white text-gray-900';

  return (
    <div className={`rounded-lg border p-4 shadow-sm ${toneClass}`}>
      <div className="text-sm text-gray-600">{label}</div>
      <div className="mt-2 text-2xl font-semibold">{formatNumber(value, 0)}</div>
    </div>
  );
}

function RecipeTable({
  title,
  emptyMessage,
  rows,
  mode,
}: {
  title: string;
  emptyMessage: string;
  rows: RecipeOverviewRow[];
  mode: 'missing' | 'cost';
}) {
  return (
    <section className="rounded-lg bg-white p-6 shadow">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
        <span className="text-sm text-gray-500">{rows.length} products</span>
      </div>

      {rows.length === 0 ? (
        <div className="mt-4 flex items-center rounded-md border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600">
          <CheckCircle2 className="mr-2 h-4 w-4 text-green-600" aria-hidden="true" />
          {emptyMessage}
        </div>
      ) : (
        <div className="mt-4 overflow-x-auto rounded-md border border-gray-200">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50">
              <tr>
                <TableHead>Product</TableHead>
                <TableHead>Price</TableHead>
                {mode === 'cost' && (
                  <>
                    <TableHead>COGS</TableHead>
                    <TableHead>Gross margin</TableHead>
                  </>
                )}
                <TableHead>Status</TableHead>
                <TableHead>Action</TableHead>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 bg-white">
              {rows.map(row => (
                <tr key={row.product.id}>
                  <TableCell>
                    <div className="font-medium text-gray-900">{row.product.name}</div>
                    <div className="text-xs text-gray-500">
                      {row.product.sku}
                      {row.product.category_name ? ` · ${row.product.category_name}` : ''}
                    </div>
                  </TableCell>
                  <TableCell>{formatCurrency(row.product.selling_price, false, 0)}</TableCell>
                  {mode === 'cost' && row.recipe && (
                    <>
                      <TableCell>
                        <div className="font-medium text-gray-900">
                          {formatCurrency(toRecipeNumber(row.recipe.cost.total_cogs), false, 0)}
                        </div>
                        <div className="text-xs text-gray-500">
                          {formatNumber(row.cogsRatio, 1)}% of price
                        </div>
                      </TableCell>
                      <TableCell>
                        {formatNumber(toRecipeNumber(row.recipe.cost.gross_margin_percentage), 1)}%
                      </TableCell>
                    </>
                  )}
                  <TableCell>
                    {mode === 'missing' ? (
                      <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-1 text-xs font-medium text-red-800">
                        <AlertTriangle className="mr-1 h-3 w-3" aria-hidden="true" />
                        Missing recipe
                      </span>
                    ) : (
                      <span className="inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-1 text-xs font-medium text-yellow-800">
                        <AlertTriangle className="mr-1 h-3 w-3" aria-hidden="true" />
                        High COGS
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Link
                      href={`/products/${row.product.id}`}
                      className="inline-flex items-center gap-1 text-sm font-medium text-primary-700 hover:text-primary-800"
                    >
                      Open product
                      <ArrowRight className="h-4 w-4" aria-hidden="true" />
                    </Link>
                  </TableCell>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
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
