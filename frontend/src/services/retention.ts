import api from './api';

export interface RetentionPolicy {
  id: string;
  table_name: string;
  record_type?: string | null;
  retention_period_days: number;
  retention_field: string;
  grace_period_days?: number | null;
  legal_minimum_days: number;
  cleanup_method: 'soft_delete' | 'hard_delete' | 'anonymize';
  notification_days_before?: number | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateRetentionPolicyRequest {
  retention_period_days?: number;
  grace_period_days?: number | null;
  cleanup_method?: 'soft_delete' | 'hard_delete' | 'anonymize';
  notification_days_before?: number | null;
  is_active?: boolean;
}

/**
 * Get all retention policies
 */
export const getRetentionPolicies = async (): Promise<RetentionPolicy[]> => {
  const response = await api.get('/user-service/retention-policies');
  return response.data;
};

/**
 * Get a specific retention policy by ID
 */
export const getRetentionPolicy = async (id: string): Promise<RetentionPolicy> => {
  const response = await api.get(`/user-service/retention-policies/${id}`);
  return response.data;
};

/**
 * Update a retention policy
 */
export const updateRetentionPolicy = async (
  id: string,
  data: UpdateRetentionPolicyRequest
): Promise<RetentionPolicy> => {
  const response = await api.put(`/user-service/retention-policies/${id}`, data);
  return response.data;
};

/**
 * Get expired record count for a policy
 */
export const getExpiredRecordCount = async (policyId: string): Promise<number> => {
  const response = await api.get(`/user-service/retention-policies/${policyId}/expired-count`);
  return response.data.count;
};
