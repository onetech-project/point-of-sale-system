export type DiscountTargetType = 'product' | 'bundle';
export type DiscountType = 'percentage' | 'fixed_amount';

export interface DiscountRule {
  id: string;
  tenant_id: string;
  name: string;
  description?: string | null;
  target_type: DiscountTargetType;
  target_id: string;
  discount_type: DiscountType;
  discount_value: number;
  starts_at?: string | null;
  ends_at?: string | null;
  is_active: boolean;
  exclusive: boolean;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface CreateDiscountRuleRequest {
  name: string;
  description?: string | null;
  target_type: DiscountTargetType;
  target_id: string;
  discount_type: DiscountType;
  discount_value: number;
  starts_at?: string | null;
  ends_at?: string | null;
  is_active?: boolean;
  exclusive?: boolean;
  priority?: number;
}

export interface UpdateDiscountRuleRequest {
  name?: string;
  description?: string | null;
  target_type?: DiscountTargetType;
  target_id?: string;
  discount_type?: DiscountType;
  discount_value?: number;
  starts_at?: string | null;
  ends_at?: string | null;
  is_active?: boolean;
  exclusive?: boolean;
  priority?: number;
}

export interface DiscountRuleListParams {
  targetType?: DiscountTargetType;
  targetId?: string;
  active?: boolean;
  limit?: number;
  offset?: number;
}

export interface PricingPreviewRequest {
  item_type: DiscountTargetType;
  product_id?: string;
  bundle_id?: string;
  quantity: number;
  unit_price: number;
  discount_rule_id?: string;
  apply_discount?: boolean;
  at?: string;
}

export interface PricingResult {
  item_type: DiscountTargetType;
  product_id?: string | null;
  bundle_id?: string | null;
  quantity: number;
  list_unit_price: number;
  list_total_price: number;
  unit_price: number;
  total_price: number;
  discount_amount: number;
  discount_rule_id?: string | null;
  discount_type?: DiscountType | null;
  discount_value?: number | null;
  discount_name?: string | null;
  pricing_snapshot?: unknown;
}
