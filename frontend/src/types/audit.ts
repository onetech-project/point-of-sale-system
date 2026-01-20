// Audit event types for UU PDP compliance audit trail

export interface AuditEvent {
  event_id: string;
  tenant_id: string;
  timestamp: string; // ISO 8601 format
  actor_type: 'user' | 'system' | 'guest' | 'admin';
  actor_id?: string;
  actor_email?: string; // Encrypted
  session_id?: string;
  action: 'CREATE' | 'READ' | 'UPDATE' | 'DELETE' | 'LOGIN' | 'LOGOUT' | 'EXPORT' | 'GRANT' | 'REVOKE';
  resource_type: string; // e.g., 'user', 'order', 'config', 'session'
  resource_id: string;
  ip_address?: string;
  user_agent?: string;
  request_id?: string;
  purpose?: string;
  before_value?: Record<string, any>; // Encrypted PII
  after_value?: Record<string, any>; // Encrypted PII
  metadata?: Record<string, any>;
  created_at: string;
}

export interface AuditQueryFilters {
  action?: string;
  resource_type?: string;
  actor_id?: string;
  start_time?: string; // RFC3339 format
  end_time?: string; // RFC3339 format
  limit?: number;
  offset?: number;
}

export interface AuditEventsResponse {
  events: AuditEvent[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}
