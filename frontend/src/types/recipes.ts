export type RecipeDecimal = number | string;

export type RecipeOverheadCalculationType = 'FIXED' | 'PERCENTAGE';

export interface RecipeCost {
  material_cost: RecipeDecimal;
  fixed_overhead_cost: RecipeDecimal;
  percentage_overhead_cost: RecipeDecimal;
  total_cogs: RecipeDecimal;
  selling_price: RecipeDecimal;
  gross_profit: RecipeDecimal;
  gross_margin_percentage: RecipeDecimal;
}

export interface RecipeItem {
  id: string;
  ingredient_id: string;
  ingredient_name?: string;
  quantity: RecipeDecimal;
  uom_id: string;
  uom_code?: string;
  normalized_quantity_base: RecipeDecimal;
  waste_percentage: RecipeDecimal;
  ingredient_cost: RecipeDecimal;
  cost_with_waste: RecipeDecimal;
}

export interface RecipeOverhead {
  id: string;
  name: string;
  calculation_type: RecipeOverheadCalculationType;
  amount: RecipeDecimal;
}

export interface Recipe {
  id: string;
  tenant_id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  version: number;
  yield_quantity: RecipeDecimal;
  is_active: boolean;
  effective_from: string;
  items: RecipeItem[];
  overheads: RecipeOverhead[];
  cost: RecipeCost;
  created_at: string;
  updated_at: string;
}

export interface RecipeVersionSummary {
  id: string;
  product_id: string;
  version: number;
  yield_quantity: RecipeDecimal;
  is_active: boolean;
  effective_from: string;
  created_at: string;
}

export interface CreateRecipeItemRequest {
  ingredient_id: string;
  quantity: RecipeDecimal;
  uom_id: string;
  waste_percentage: RecipeDecimal;
}

export interface CreateRecipeOverheadRequest {
  name: string;
  calculation_type: RecipeOverheadCalculationType;
  amount: RecipeDecimal;
}

export interface CreateRecipeRequest {
  yield_quantity: RecipeDecimal;
  effective_from?: string;
  items: CreateRecipeItemRequest[];
  overheads: CreateRecipeOverheadRequest[];
}

export interface Ingredient {
  id: string;
  tenant_id: string;
  outlet_id?: string | null;
  sku: string;
  name: string;
  base_uom_id: string;
  base_uom_code?: string | null;
  current_stock_base: RecipeDecimal;
  average_cost_per_base_unit: RecipeDecimal;
  minimum_stock_base: RecipeDecimal;
  track_stock: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface IngredientListParams {
  search?: string;
  limit?: number;
  offset?: number;
  include_inactive?: boolean;
}

export interface IngredientListResponse {
  ingredients: Ingredient[];
  total: number;
  limit: number;
  offset: number;
}

export interface UOM {
  id: string;
  tenant_id?: string | null;
  code: string;
  name: string;
  category: string;
  is_base_unit: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
