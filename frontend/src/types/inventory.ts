export type InventoryNumber = number | string;

export type UOMCategory = 'weight' | 'volume' | 'count' | 'packaging' | (string & {});

export interface UOM {
  id: string;
  tenant_id?: string | null;
  code: string;
  name: string;
  category: UOMCategory;
  is_base_unit: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateUOMRequest {
  code: string;
  name: string;
  category: UOMCategory;
  is_base_unit: boolean;
  is_active?: boolean;
}

export interface UpdateUOMRequest {
  name?: string;
  category?: UOMCategory;
  is_base_unit?: boolean;
  is_active?: boolean;
}

export interface Ingredient {
  id: string;
  tenant_id: string;
  outlet_id?: string | null;
  sku: string;
  name: string;
  base_uom_id: string;
  base_uom_code?: string | null;
  current_stock_base: InventoryNumber;
  average_cost_per_base_unit: InventoryNumber;
  minimum_stock_base: InventoryNumber;
  track_stock: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateIngredientRequest {
  outlet_id?: string | null;
  sku: string;
  name: string;
  base_uom_id: string;
  minimum_stock_base: number;
  average_cost_per_base_unit: number;
  track_stock?: boolean;
  is_active?: boolean;
}

export interface UpdateIngredientRequest {
  outlet_id?: string | null;
  sku?: string;
  name?: string;
  base_uom_id?: string;
  minimum_stock_base?: number;
  average_cost_per_base_unit?: number;
  track_stock?: boolean;
  is_active?: boolean;
}

export interface IngredientListParams {
  page?: number;
  limit?: number;
  search?: string;
  includeInactive?: boolean;
}

export interface InventoryPaginationParams {
  page?: number;
  limit?: number;
}

export interface PaginatedInventoryResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  offset: number;
}

export interface IngredientUOMConversion {
  id: string;
  tenant_id?: string | null;
  ingredient_id?: string | null;
  from_uom_id: string;
  to_uom_id: string;
  multiplier: InventoryNumber;
  created_at: string;
  updated_at: string;
}

export interface CreateConversionRequest {
  from_uom_id: string;
  to_uom_id: string;
  multiplier: number;
}

export interface UpdateConversionRequest {
  from_uom_id?: string;
  to_uom_id?: string;
  multiplier?: number;
}

export type StockMovementType =
  | 'INITIAL_STOCK'
  | 'PURCHASE_IN'
  | 'MANUAL_ADJUSTMENT'
  | 'WASTE'
  | (string & {});

export interface StockMovement {
  id: string;
  tenant_id: string;
  outlet_id?: string | null;
  ingredient_id: string;
  movement_type: StockMovementType;
  quantity_base: InventoryNumber;
  unit_cost: InventoryNumber;
  total_cost: InventoryNumber;
  reference_type?: string | null;
  reference_id?: string | null;
  idempotency_key?: string | null;
  reason?: string | null;
  occurred_at: string;
  created_by?: string | null;
  created_at: string;
}

export interface StockMutationRequest {
  ingredient_id: string;
  outlet_id?: string | null;
  quantity: number;
  uom_id: string;
  unit_cost: number;
  idempotency_key?: string;
  reference_type?: string;
  reference_id?: string;
  reason?: string;
  occurred_at?: string;
}

export interface ManualAdjustmentRequest {
  ingredient_id: string;
  outlet_id?: string | null;
  quantity_delta: number;
  uom_id: string;
  unit_cost: number;
  idempotency_key?: string;
  reason: string;
  occurred_at?: string;
}

export interface InventoryValuation {
  valuation: InventoryNumber;
}

export interface InventoryOverview {
  total_ingredients: number;
  active_uoms: number;
  low_stock_count: number;
  valuation: InventoryNumber;
}

export interface BundleItem {
  id: string;
  product_id: string;
  product_name?: string;
  product_sku?: string;
  quantity: InventoryNumber;
}

export interface Bundle {
  id: string;
  tenant_id: string;
  sku: string;
  name: string;
  description?: string | null;
  selling_price: InventoryNumber;
  is_active: boolean;
  items: BundleItem[];
  created_at: string;
  updated_at: string;
}

export interface BundleListParams extends InventoryPaginationParams {
  search?: string;
  includeInactive?: boolean;
}

export interface BundleItemInput {
  product_id: string;
  quantity: number;
}

export interface CreateBundleRequest {
  sku: string;
  name: string;
  description?: string | null;
  selling_price: number;
  is_active?: boolean;
  items: BundleItemInput[];
}

export interface UpdateBundleRequest {
  sku?: string;
  name?: string;
  description?: string | null;
  selling_price?: number;
  is_active?: boolean;
  items?: BundleItemInput[];
}

export interface BundleItemCost {
  bundle_item_id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: InventoryNumber;
  recipe_id: string;
  recipe_version: number;
  unit_material_cost: InventoryNumber;
  unit_fixed_overhead_cost: InventoryNumber;
  unit_percentage_overhead_cost: InventoryNumber;
  unit_total_cogs: InventoryNumber;
  line_material_cost: InventoryNumber;
  line_fixed_overhead_cost: InventoryNumber;
  line_percentage_overhead_cost: InventoryNumber;
  line_total_cogs: InventoryNumber;
}

export interface BundleCost {
  bundle_id: string;
  bundle_name: string;
  bundle_sku: string;
  items: BundleItemCost[];
  material_cost: InventoryNumber;
  fixed_overhead_cost: InventoryNumber;
  percentage_overhead_cost: InventoryNumber;
  total_cogs: InventoryNumber;
  selling_price: InventoryNumber;
  gross_profit: InventoryNumber;
  gross_margin_percentage: InventoryNumber;
}
