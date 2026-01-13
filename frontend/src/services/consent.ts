import apiClient from './api';

export interface ConsentPurpose {
  purpose_code: string;
  display_name_id: string;
  description_id: string;
  is_required: boolean;
  display_order: number;
}

export interface ConsentRecord {
  id: string;
  subject_type: 'tenant' | 'guest';
  subject_id: string;
  purpose_code: string;
  granted: boolean;
  granted_at: string;
  revoked_at?: string;
  metadata?: {
    ip_address?: string;
    user_agent?: string;
  };
}

export interface GrantConsentRequest {
  tenant_id?: string; // Required for guest checkouts
  subject_type: 'tenant' | 'guest';
  subject_id: string;
  consents: {
    purpose_code: string;
    granted: boolean;
  }[];
  metadata?: {
    ip_address?: string;
    user_agent?: string;
  };
}

export interface ConsentStatus {
  subject_type: string;
  subject_id: string;
  active_consents: string[];
  revoked_consents: string[];
}

class ConsentService {
  /**
   * Get all available consent purposes
   * @param language - Language code (id or en)
   * @param context - Context filter: 'tenant' for registration, 'guest' for checkout
   */
  async getConsentPurposes(language: string = 'id', context?: 'tenant' | 'guest'): Promise<ConsentPurpose[]> {
    const params = context ? { context } : {};
    const response = await apiClient.get<{ data: ConsentPurpose[] }>(
      '/api/v1/consent/purposes',
      {
        params,
        headers: {
          'Accept-Language': language,
        },
      }
    );
    return response.data;
  }

  /**
   * Grant consents for a user or guest
   */
  async grantConsents(data: GrantConsentRequest): Promise<void> {
    // Transform consents array to purpose_codes array (only granted consents)
    const purpose_codes = data.consents
      .filter(c => c.granted)
      .map(c => c.purpose_code);
    
    const requestBody = {
      tenant_id: data.tenant_id,
      purpose_codes,
      subject_type: data.subject_type,
      subject_id: data.subject_type === 'tenant' ? data.subject_id : undefined,
      guest_order_id: data.subject_type === 'guest' ? data.subject_id : undefined,
      consent_method: data.subject_type === 'tenant' ? 'registration' : 'checkout',
      ...(data.metadata && {
        // Metadata is handled by backend from request headers
      }),
    };
    
    await apiClient.post('/api/v1/consent/grant', requestBody);
  }

  /**
   * Get consent status for a subject
   */
  async getConsentStatus(
    subjectType: 'tenant' | 'guest',
    subjectId: string
  ): Promise<ConsentStatus> {
    const response = await apiClient.get<{ data: ConsentStatus }>(
      `/api/v1/consent/status?subject_type=${subjectType}&subject_id=${subjectId}`
    );
    return response.data;
  }

  /**
   * Revoke an optional consent
   */
  async revokeConsent(
    subjectType: 'tenant' | 'guest',
    subjectId: string,
    purposeCode: string
  ): Promise<void> {
    await apiClient.post('/api/v1/consent/revoke', {
      subject_type: subjectType,
      subject_id: subjectId,
      purpose_code: purposeCode,
    });
  }

  /**
   * Get consent history for a subject
   */
  async getConsentHistory(
    subjectType: 'tenant' | 'guest',
    subjectId: string
  ): Promise<ConsentRecord[]> {
    const response = await apiClient.get<{ data: ConsentRecord[] }>(
      `/api/v1/consent/history?subject_type=${subjectType}&subject_id=${subjectId}`
    );
    return response.data;
  }

  /**
   * Get current privacy policy
   */
  async getPrivacyPolicy(): Promise<{
    version: string;
    policy_text_id: string;
    effective_date: string;
  }> {
    const response = await apiClient.get<{
      data: {
        version: string;
        policy_text_id: string;
        effective_date: string;
      };
    }>('/api/v1/privacy-policy');
    return response.data;
  }

  /**
   * Helper to get browser IP and user agent
   */
  getConsentMetadata(): { ip_address?: string; user_agent?: string } {
    return {
      user_agent: typeof window !== 'undefined' ? window.navigator.userAgent : undefined,
      // IP address will be set by backend from request headers
    };
  }

  /**
   * Validate that all required consents are granted
   */
  validateRequiredConsents(
    purposes: ConsentPurpose[],
    consents: { [key: string]: boolean }
  ): { valid: boolean; missing: string[] } {
    const requiredPurposes = purposes.filter((p) => p.is_required);
    const missing = requiredPurposes
      .filter((p) => !consents[p.purpose_code])
      .map((p) => p.purpose_code);

    return {
      valid: missing.length === 0,
      missing,
    };
  }
}

export default new ConsentService();
