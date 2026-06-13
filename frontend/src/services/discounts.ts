import apiClient from './api';
import type {
  CreateDiscountRuleRequest,
  DiscountRule,
  DiscountRuleListParams,
  PricingPreviewRequest,
  PricingResult,
  UpdateDiscountRuleRequest,
} from '@/types/discounts';

const DISCOUNT_RULES_BASE = '/api/v1/admin/discount-rules';

class DiscountService {
  async listDiscountRules(params: DiscountRuleListParams = {}): Promise<DiscountRule[]> {
    const queryParams = new URLSearchParams();

    if (params.targetType) {
      queryParams.append('target_type', params.targetType);
    }
    if (params.targetId) {
      queryParams.append('target_id', params.targetId);
    }
    if (params.active !== undefined) {
      queryParams.append('active', String(params.active));
    }
    if (params.limit !== undefined) {
      queryParams.append('limit', String(params.limit));
    }
    if (params.offset !== undefined) {
      queryParams.append('offset', String(params.offset));
    }

    const url = queryParams.toString()
      ? `${DISCOUNT_RULES_BASE}?${queryParams}`
      : DISCOUNT_RULES_BASE;
    const response = await apiClient.get<{ discount_rules: DiscountRule[] }>(url);
    return response.discount_rules || [];
  }

  async createDiscountRule(data: CreateDiscountRuleRequest): Promise<DiscountRule> {
    return apiClient.post<DiscountRule>(DISCOUNT_RULES_BASE, data);
  }

  async updateDiscountRule(id: string, data: UpdateDiscountRuleRequest): Promise<DiscountRule> {
    return apiClient.patch<DiscountRule>(`${DISCOUNT_RULES_BASE}/${id}`, data);
  }

  async deleteDiscountRule(id: string): Promise<void> {
    return apiClient.delete(`${DISCOUNT_RULES_BASE}/${id}`);
  }

  async previewPricing(data: PricingPreviewRequest): Promise<PricingResult> {
    return apiClient.post<PricingResult>(`${DISCOUNT_RULES_BASE}/preview`, data);
  }
}

export const discounts = new DiscountService();
