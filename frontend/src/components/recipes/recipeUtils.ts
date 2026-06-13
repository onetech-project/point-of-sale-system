import type { RecipeCost, RecipeDecimal } from '@/types/recipes';

export const toRecipeNumber = (value: RecipeDecimal | null | undefined): number => {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? value : 0;
  }

  if (typeof value === 'string') {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  return 0;
};

export const getRecipeCogsRatio = (
  cost: RecipeCost | null | undefined,
  fallbackSellingPrice = 0
): number => {
  if (!cost) {
    return 0;
  }

  const sellingPrice = toRecipeNumber(cost.selling_price) || fallbackSellingPrice;
  if (sellingPrice <= 0) {
    return 0;
  }

  return (toRecipeNumber(cost.total_cogs) / sellingPrice) * 100;
};

export const isRecipeNotFound = (error: unknown): boolean => {
  const maybeError = error as { response?: { status?: number } };
  return maybeError?.response?.status === 404;
};

export const getRecipeErrorMessage = (error: unknown, fallback: string): string => {
  const maybeError = error as { response?: { data?: { message?: string } }; message?: string };
  return maybeError?.response?.data?.message || maybeError?.message || fallback;
};
