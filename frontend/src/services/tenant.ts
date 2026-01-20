import apiClient from './api';
import { 
  TenantData, 
  TenantInfo, 
  TenantConfig, 
  DeleteUserResponse 
} from '../types/tenant';

/**
 * Tenant Service
 * Handles all tenant-related API operations including data management for UU PDP compliance
 */
export const tenantService = {
  /**
   * Get tenant configuration by ID (public endpoint)
   * Includes delivery types, branding, and operational settings
   */
  async getTenantConfig(tenantId: string): Promise<TenantConfig> {
    const response = await apiClient.get<TenantConfig>(
      `/api/public/tenants/${tenantId}/config`
    );
    return response;
  },

  /**
   * Check if tenant is active and accessible
   */
  async validateTenant(tenantId: string): Promise<boolean> {
    try {
      await this.getTenantConfig(tenantId);
      return true;
    } catch (error: any) {
      if (error.response?.status === 404 || error.response?.status === 403) {
        return false;
      }
      throw error;
    }
  },

  /**
   * Get tenant info (authenticated endpoint)
   * Returns basic tenant information for the authenticated user's tenant
   */
  async getTenantInfo(): Promise<TenantInfo> {
    const response = await apiClient.get<TenantInfo>('/api/tenant');
    return response;
  },

  /**
   * Get all tenant data (UU PDP Article 3 - Right to Access)
   * Returns aggregated tenant data including business profile, team members, and configuration
   * Requires: Owner role
   */
  async getAllTenantData(): Promise<TenantData> {
    const response = await apiClient.get<TenantData>('/api/v1/tenant/data');
    return response;
  },

  /**
   * Export tenant data as JSON (UU PDP Article 4 - Data Portability)
   * Downloads a JSON file with all tenant data
   * Requires: Owner role
   */
  async exportTenantData(): Promise<Blob> {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/tenant/data/export`, {
      method: 'POST',
      credentials: 'include', // Send cookies for authentication
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error('Failed to export tenant data');
    }

    return await response.blob();
  },

  /**
   * Delete a team member (UU PDP Article 5 - Right to Deletion)
   * @param userId - User ID to delete
   * @param force - If true, performs hard delete (permanent). If false, performs soft delete (90-day retention)
   * Requires: Owner role
   */
  async deleteUser(userId: string, force: boolean = false): Promise<DeleteUserResponse> {
    const url = `/api/v1/tenant/users/${userId}${force ? '?force=true' : ''}`;
    const response = await apiClient.delete<DeleteUserResponse>(url);
    return response;
  },
};

// Export legacy alias for backward compatibility
export const tenant = tenantService;
