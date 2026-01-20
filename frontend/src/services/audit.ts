import apiClient from './api';
import type { AuditEventsResponse, AuditQueryFilters } from '../types/audit';

class AuditService {
  /**
   * Get audit trail for authenticated tenant (T111)
   * GET /api/v1/audit/tenant
   */
  async getTenantAuditEvents(filters?: AuditQueryFilters): Promise<AuditEventsResponse> {
    const params = new URLSearchParams();

    if (filters?.action) params.append('action', filters.action);
    if (filters?.resource_type) params.append('resource_type', filters.resource_type);
    if (filters?.actor_id) params.append('actor_id', filters.actor_id);
    if (filters?.start_time) params.append('start_time', filters.start_time);
    if (filters?.end_time) params.append('end_time', filters.end_time);
    if (filters?.limit) params.append('limit', filters.limit.toString());
    if (filters?.offset) params.append('offset', filters.offset.toString());

    const queryString = params.toString();
    const url = queryString ? `/api/v1/audit/tenant?${queryString}` : '/api/v1/audit/tenant';

    const response = await apiClient.get<AuditEventsResponse>(url);
    return response;
  }
}

const auditService = new AuditService();
export default auditService;
