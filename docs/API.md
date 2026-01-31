# API Documentation

## Overview

This document provides comprehensive API documentation for the POS System microservices.

---

## Notification Service API

Base URL: `http://api-gateway:8080/api/v1`

### Notification History

#### Get Notification History

Get paginated notification history with optional filters.

**Endpoint**: `GET /notifications/history`

**Authentication**: Required (JWT)

**Authorization**: Tenant-scoped (only returns notifications for authenticated user's tenant)

**Query Parameters**:

| Parameter       | Type    | Required | Default | Description                                      |
| --------------- | ------- | -------- | ------- | ------------------------------------------------ |
| page            | integer | No       | 1       | Page number (1-indexed)                          |
| page_size       | integer | No       | 20      | Items per page (max 100)                         |
| order_reference | string  | No       | -       | Filter by order reference                        |
| status          | string  | No       | -       | Filter by status (sent/pending/failed/cancelled) |
| type            | string  | No       | -       | Filter by type (staff/customer)                  |
| start_date      | string  | No       | -       | Filter by start date (ISO 8601)                  |
| end_date        | string  | No       | -       | Filter by end date (ISO 8601)                    |

**Response**: `200 OK`

```json
{
  "notifications": [
    {
      "id": 123,
      "tenant_id": "tenant-uuid",
      "user_id": "user-uuid",
      "type": "staff",
      "status": "sent",
      "subject": "New Order Received - ORD-2024-001",
      "recipient": "staff@example.com",
      "sent_at": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-15T10:29:55Z",
      "retry_count": 0,
      "metadata": {
        "order_reference": "ORD-2024-001",
        "transaction_id": "txn-123",
        "event_type": "order.paid"
      }
    }
  ],
  "pagination": {
    "current_page": 1,
    "page_size": 20,
    "total_items": 156,
    "total_pages": 8
  }
}
```

**Error Responses**:

- `400 Bad Request`: Invalid query parameters
- `401 Unauthorized`: Missing or invalid JWT token
- `500 Internal Server Error`: Server error

---

#### Resend Notification

Resend a failed notification email.

**Endpoint**: `POST /notifications/:notification_id/resend`

**Authentication**: Required (JWT)

**Authorization**: Tenant-scoped (can only resend notifications belonging to authenticated user's tenant)

**Path Parameters**:

| Parameter       | Type    | Required | Description               |
| --------------- | ------- | -------- | ------------------------- |
| notification_id | integer | Yes      | Notification ID to resend |

**Response**: `200 OK`

```json
{
  "message": "Notification resent successfully"
}
```

**Error Responses**:

- `400 Bad Request`: Invalid notification ID
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: Notification belongs to different tenant
- `404 Not Found`: Notification not found
- `409 Conflict`: Notification already sent successfully
- `429 Too Many Requests`: Maximum retry attempts (3) exceeded
- `500 Internal Server Error`: Server error

**Error Response Example**:

```json
{
  "error": "max retries exceeded",
  "message": "This notification has already been retried 3 times"
}
```

---

### Notification Configuration

#### Get Notification Config

Get notification configuration for tenant.

**Endpoint**: `GET /notifications/config`

**Authentication**: Required (JWT)

**Authorization**: Admin role required

**Response**: `200 OK`

```json
{
  "tenant_id": "tenant-uuid",
  "staff_notification_enabled": true,
  "customer_receipt_enabled": true,
  "admin_email": "admin@example.com",
  "staff_emails": ["staff1@example.com", "staff2@example.com"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses**:

- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not admin
- `404 Not Found`: Configuration not found
- `500 Internal Server Error`: Server error

---

#### Update Notification Config

Update notification configuration for tenant.

**Endpoint**: `PATCH /notifications/config`

**Authentication**: Required (JWT)

**Authorization**: Admin role required

**Request Body**:

```json
{
  "staff_notification_enabled": true,
  "customer_receipt_enabled": true,
  "admin_email": "admin@example.com",
  "staff_emails": ["staff1@example.com", "staff2@example.com"]
}
```

**Response**: `200 OK`

```json
{
  "tenant_id": "tenant-uuid",
  "staff_notification_enabled": true,
  "customer_receipt_enabled": true,
  "admin_email": "admin@example.com",
  "staff_emails": ["staff1@example.com", "staff2@example.com"],
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses**:

- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not admin
- `500 Internal Server Error`: Server error

---

#### Send Test Notification

Send a test notification email to verify configuration.

**Endpoint**: `POST /notifications/test`

**Authentication**: Required (JWT)

**Authorization**: Admin role required

**Rate Limiting**: 5 requests per minute per user

**Request Body**:

```json
{
  "recipient_email": "test@example.com",
  "notification_type": "staff"
}
```

**Response**: `200 OK`

```json
{
  "message": "Test notification sent successfully",
  "notification_id": 123
}
```

**Error Responses**:

- `400 Bad Request`: Invalid request body or missing fields
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not admin
- `429 Too Many Requests`: Rate limit exceeded (5/min)
- `500 Internal Server Error`: Server error

---

## User Service API

Base URL: `http://api-gateway:8080/api/v1`

### User Notification Preferences

#### Get User Notification Preferences

Get notification preferences for all users in tenant.

**Endpoint**: `GET /users/notification-preferences`

**Authentication**: Required (JWT)

**Authorization**: Admin role required, tenant-scoped

**Response**: `200 OK`

```json
{
  "preferences": [
    {
      "user_id": "user-uuid-1",
      "email": "staff1@example.com",
      "name": "John Doe",
      "staff_notifications_enabled": true
    },
    {
      "user_id": "user-uuid-2",
      "email": "staff2@example.com",
      "name": "Jane Smith",
      "staff_notifications_enabled": false
    }
  ]
}
```

**Error Responses**:

- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not admin
- `500 Internal Server Error`: Server error

---

#### Update User Notification Preference

Update notification preference for a specific user.

**Endpoint**: `PATCH /users/:user_id/notification-preferences`

**Authentication**: Required (JWT)

**Authorization**: Admin role required, can only update users in same tenant

**Path Parameters**:

| Parameter | Type   | Required | Description         |
| --------- | ------ | -------- | ------------------- |
| user_id   | string | Yes      | User UUID to update |

**Request Body**:

```json
{
  "staff_notifications_enabled": false
}
```

**Response**: `200 OK`

```json
{
  "user_id": "user-uuid",
  "staff_notifications_enabled": false,
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses**:

- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not admin or trying to update user from different tenant
- `404 Not Found`: User not found
- `500 Internal Server Error`: Server error

---

## Rate Limiting

All API endpoints implement rate limiting to prevent abuse:

| Endpoint                                  | Rate Limit                   |
| ----------------------------------------- | ---------------------------- |
| GET /notifications/history                | 100 requests/minute per user |
| POST /notifications/:id/resend            | 10 requests/minute per user  |
| GET /notifications/config                 | 60 requests/minute per user  |
| PATCH /notifications/config               | 30 requests/minute per user  |
| POST /notifications/test                  | 5 requests/minute per user   |
| GET /users/notification-preferences       | 60 requests/minute per user  |
| PATCH /users/:id/notification-preferences | 30 requests/minute per user  |

Rate limit responses return `429 Too Many Requests` with a `Retry-After` header.

---

## Error Handling

All API errors follow a consistent format:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

### Common Error Codes

| Code              | HTTP Status | Description                                            |
| ----------------- | ----------- | ------------------------------------------------------ |
| `invalid_request` | 400         | Request body or parameters are invalid                 |
| `unauthorized`    | 401         | Missing or invalid authentication token                |
| `forbidden`       | 403         | User lacks required permissions                        |
| `not_found`       | 404         | Resource not found                                     |
| `conflict`        | 409         | Resource conflict (e.g., duplicate, already processed) |
| `rate_limited`    | 429         | Rate limit exceeded                                    |
| `internal_error`  | 500         | Internal server error                                  |

### Notification-Specific Error Codes

| Code                     | HTTP Status | Description                                                   |
| ------------------------ | ----------- | ------------------------------------------------------------- |
| `max_retries_exceeded`   | 429         | Notification has been retried maximum number of times (3)     |
| `already_sent`           | 409         | Cannot resend notification that was already sent successfully |
| `invalid_recipient`      | 400         | Email address is invalid or unavailable                       |
| `smtp_auth_failed`       | 500         | SMTP authentication failed (check configuration)              |
| `smtp_connection_failed` | 500         | Cannot connect to SMTP server                                 |

---

## Authentication

All API endpoints require JWT authentication via Bearer token:

```
Authorization: Bearer <jwt_token>
```

The JWT token must include:

- `tenant_id`: Tenant UUID for multi-tenancy isolation
- `user_id`: User UUID
- `role`: User role (admin/staff/customer)

Tokens are obtained from the auth-service login endpoint.

---

## Multi-Tenancy

All endpoints are tenant-scoped. Resources are automatically filtered by the tenant_id from the JWT token. Users cannot access resources belonging to other tenants.

---

## Monitoring & Metrics

The notification service tracks the following metrics:

| Metric                             | Type    | Tags                      | Description                         |
| ---------------------------------- | ------- | ------------------------- | ----------------------------------- |
| `notification.email.sent`          | Counter | retry_count               | Successful email deliveries         |
| `notification.email.failed`        | Counter | error_type, retryable     | Failed email deliveries             |
| `notification.email.duration_ms`   | Gauge   | -                         | Email delivery time in milliseconds |
| `notification.duplicate.prevented` | Counter | tenant_id, payment_method | Duplicate notifications prevented   |

Metrics are logged in structured format for aggregation by monitoring systems (Prometheus, Datadog, etc.).

---

## SMTP Configuration

Email delivery requires the following environment variables:

| Variable            | Required | Default                | Description                              |
| ------------------- | -------- | ---------------------- | ---------------------------------------- |
| SMTP_HOST           | Yes      | localhost              | SMTP server hostname                     |
| SMTP_PORT           | Yes      | 587                    | SMTP server port                         |
| SMTP_USERNAME       | Yes      | -                      | SMTP authentication username             |
| SMTP_PASSWORD       | Yes      | -                      | SMTP authentication password             |
| SMTP_FROM           | Yes      | noreply@pos-system.com | From email address                       |
| SMTP_RETRY_ATTEMPTS | No       | 3                      | Maximum retry attempts for failed emails |

### Error Handling & Retry Logic

The notification service implements comprehensive error handling:

1. **Error Classification**: Errors are categorized as:
   - Connection errors (network issues)
   - Authentication errors (invalid credentials)
   - Timeout errors (request timeouts)
   - Invalid recipient errors (bad email addresses)
   - Rate limiting errors (SMTP provider limits)

2. **Retry Logic**:
   - Retryable errors (connection, timeout, rate limit) are automatically retried
   - Exponential backoff: 2s, 4s, 8s delays between retries
   - Non-retryable errors (auth, invalid recipient) fail immediately
   - Maximum 3 retry attempts (configurable via SMTP_RETRY_ATTEMPTS)

3. **Status Tracking**:
   - Each notification tracks `retry_count` in database
   - Failed notifications include `error_msg` for debugging
   - Timestamps: `sent_at` for success, `failed_at` for failures

---

## Webhook Events (Future)

Future versions may support webhooks for real-time notification delivery status:

```json
{
  "event": "notification.sent",
  "notification_id": 123,
  "status": "sent",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Testing

### Development Mode

When `SMTP_USERNAME` is empty, emails are logged to stdout instead of sent:

```
[EMAIL] To: test@example.com, Subject: Test Email
<email body content>
```

This allows testing email rendering without SMTP configuration.

### Test Notification Endpoint

Use `POST /notifications/test` to send test emails and verify:

- SMTP configuration is correct
- Email templates render properly
- Rate limiting is working
- Authentication and authorization are enforced

---

## Best Practices

1. **Rate Limiting**: Respect rate limits to avoid 429 errors
2. **Error Handling**: Implement exponential backoff for retries on 5xx errors
3. **Pagination**: Use appropriate page_size for notification history (max 100)
4. **Filters**: Combine filters to reduce result set and improve performance
5. **Monitoring**: Track notification delivery metrics for observability
6. **Duplicate Prevention**: The system automatically prevents duplicate notifications for the same transaction_id
7. **Retry Strategy**: Let the service handle retries automatically; don't retry 4xx errors

---

---

## UU PDP Compliance API

Base URL: `http://api-gateway:8080/api/v1`

---

### Consent Management

#### Grant Consent

Grant consent for one or more purposes.

**Endpoint**: `POST /consent/grant`

**Authentication**: Required (JWT) OR Guest (order reference)

**Request Body**:

```json
{
  "subject_type": "tenant",
  "subject_id": "uuid",
  "consents": [
    {
      "purpose_code": "operational",
      "granted": true
    },
    {
      "purpose_code": "analytics",
      "granted": true
    }
  ]
}
```

**Parameters**:

| Field                   | Type    | Required | Description                                                                     |
| ----------------------- | ------- | -------- | ------------------------------------------------------------------------------- |
| subject_type            | string  | Yes      | Either "tenant" or "guest"                                                      |
| subject_id              | string  | Yes      | Tenant ID or Guest Order ID                                                     |
| consents                | array   | Yes      | Array of consent grants                                                         |
| consents[].purpose_code | string  | Yes      | Purpose code (operational, analytics, advertising, payment_processing_midtrans) |
| consents[].granted      | boolean | Yes      | Whether consent is granted                                                      |

**Response**: `201 Created`

```json
{
  "message": "Consents recorded successfully",
  "consents": [
    {
      "id": "consent-uuid",
      "purpose_code": "operational",
      "granted_at": "2026-01-16T10:00:00Z"
    }
  ]
}
```

**Error Responses**:

- `400 Bad Request`: Invalid request body, missing required consent
- `401 Unauthorized`: Missing or invalid JWT token
- `409 Conflict`: Consent already exists for this purpose
- `500 Internal Server Error`: Server error

---

#### Revoke Consent

Revoke consent for an optional purpose.

**Endpoint**: `POST /consent/revoke`

**Authentication**: Required (JWT)

**Authorization**: Tenant-scoped (can only revoke own consents)

**Request Body**:

```json
{
  "purpose_code": "analytics"
}
```

**Parameters**:

| Field        | Type   | Required | Description                               |
| ------------ | ------ | -------- | ----------------------------------------- |
| purpose_code | string | Yes      | Purpose code to revoke (must be optional) |

**Response**: `200 OK`

```json
{
  "message": "Consent revoked successfully",
  "revoked_at": "2026-01-16T10:00:00Z"
}
```

**Error Responses**:

- `400 Bad Request`: Cannot revoke required consent
- `401 Unauthorized`: Missing or invalid JWT token
- `404 Not Found`: No active consent found for this purpose
- `500 Internal Server Error`: Server error

---

#### Get Consent Status

Get all consent records for authenticated user/tenant.

**Endpoint**: `GET /consent/status`

**Authentication**: Required (JWT) OR Guest (order reference)

**Response**: `200 OK`

```json
{
  "consents": [
    {
      "purpose_code": "operational",
      "purpose_description": "Essential service operations",
      "is_required": true,
      "granted": true,
      "granted_at": "2026-01-01T10:00:00Z",
      "revoked_at": null,
      "version": 1
    },
    {
      "purpose_code": "analytics",
      "purpose_description": "Usage analytics for service improvement",
      "is_required": false,
      "granted": false,
      "granted_at": null,
      "revoked_at": "2026-01-10T15:30:00Z",
      "version": 1
    }
  ]
}
```

---

### Tenant Data Rights

#### View Tenant Data

Get all data associated with tenant account.

**Endpoint**: `GET /tenant/data`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Response**: `200 OK`

```json
{
  "tenant": {
    "id": "uuid",
    "business_name": "My Business",
    "email": "owner@business.com",
    "phone": "+628123456789",
    "created_at": "2025-01-01T00:00:00Z"
  },
  "users": [
    {
      "id": "uuid",
      "email": "staff@business.com",
      "full_name": "John Doe",
      "role": "STAFF",
      "created_at": "2025-01-15T10:00:00Z"
    }
  ],
  "configurations": {
    "midtrans_config": {
      "server_key": "[ENCRYPTED]",
      "client_key": "[ENCRYPTED]"
    }
  },
  "consent_records": [
    {
      "purpose_code": "operational",
      "granted_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

**Error Responses**:

- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not OWNER role
- `500 Internal Server Error`: Server error

---

#### Export Tenant Data

Export all tenant data in JSON format for portability.

**Endpoint**: `POST /tenant/data/export`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Response**: `200 OK`

Content-Type: `application/json`  
Content-Disposition: `attachment; filename="tenant_data_2026-01-16.json"`

```json
{
  "export_date": "2026-01-16T10:00:00Z",
  "tenant": { ... },
  "users": [ ... ],
  "orders": [ ... ],
  "configurations": { ... },
  "consent_records": [ ... ],
  "audit_trail": [ ... ]
}
```

**Error Responses**:

- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not OWNER role
- `500 Internal Server Error`: Server error

---

#### Delete Team Member

Delete a team member from tenant account.

**Endpoint**: `DELETE /tenant/users/:user_id`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Path Parameters**:

| Parameter | Type   | Required | Description       |
| --------- | ------ | -------- | ----------------- |
| user_id   | string | Yes      | User ID to delete |

**Query Parameters**:

| Parameter | Type    | Required | Default | Description                                                 |
| --------- | ------- | -------- | ------- | ----------------------------------------------------------- |
| force     | boolean | No       | false   | If true, hard delete immediately (skip 90-day grace period) |

**Response**: `200 OK`

```json
{
  "message": "User deleted successfully",
  "deletion_type": "soft_delete",
  "grace_period_days": 90,
  "permanent_deletion_date": "2026-04-16T10:00:00Z"
}
```

**Error Responses**:

- `400 Bad Request`: Cannot delete last OWNER
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not OWNER role
- `404 Not Found`: User not found
- `500 Internal Server Error`: Server error

---

### Guest Data Rights

#### View Guest Order Data

View all data associated with a guest order.

**Endpoint**: `GET /guest/order/:order_reference/data`

**Authentication**: None (public endpoint with verification)

**Path Parameters**:

| Parameter       | Type   | Required | Description          |
| --------------- | ------ | -------- | -------------------- |
| order_reference | string | Yes      | Order reference code |

**Query Parameters** (verification required):

| Parameter | Type   | Required | Description            |
| --------- | ------ | -------- | ---------------------- |
| email     | string | No\*     | Customer email address |
| phone     | string | No\*     | Customer phone number  |

\*At least one of email or phone is required

**Response**: `200 OK`

```json
{
  "order": {
    "order_reference": "ORD-2026-001",
    "customer_name": "Jane Smith",
    "customer_email": "jane@example.com",
    "customer_phone": "+628123456789",
    "delivery_address": "Jl. Example No. 123",
    "total_amount": 150000,
    "created_at": "2026-01-15T14:30:00Z"
  },
  "consent_records": [
    {
      "purpose_code": "operational",
      "granted_at": "2026-01-15T14:30:00Z"
    }
  ],
  "is_anonymized": false
}
```

**Error Responses**:

- `400 Bad Request`: Missing email or phone parameter
- `404 Not Found`: Order not found or verification failed
- `410 Gone`: Order data has been deleted/anonymized
- `500 Internal Server Error`: Server error

---

#### Delete Guest Order Data

Request deletion/anonymization of guest order data.

**Endpoint**: `POST /guest/order/:order_reference/delete`

**Authentication**: None (public endpoint with verification)

**Path Parameters**:

| Parameter       | Type   | Required | Description          |
| --------------- | ------ | -------- | -------------------- |
| order_reference | string | Yes      | Order reference code |

**Request Body**:

```json
{
  "email": "jane@example.com",
  "phone": "+628123456789"
}
```

**Parameters**:

| Field | Type   | Required | Description            |
| ----- | ------ | -------- | ---------------------- |
| email | string | No\*     | Customer email address |
| phone | string | No\*     | Customer phone number  |

\*At least one of email or phone is required for verification

**Response**: `200 OK`

```json
{
  "message": "Guest order data anonymized successfully",
  "anonymized_at": "2026-01-16T10:00:00Z",
  "order_reference": "ORD-2026-001"
}
```

**Effects**:

- Customer name replaced with "Deleted User"
- Customer email set to null
- Customer phone set to null
- Delivery address set to null
- Order record preserved for merchant (total_amount, items, timestamps)
- `is_anonymized` flag set to true
- Audit event created

**Error Responses**:

- `400 Bad Request`: Missing email or phone parameter
- `404 Not Found`: Order not found or verification failed
- `409 Conflict`: Order already anonymized
- `500 Internal Server Error`: Server error

---

### Retention Policy Management

#### Get Retention Policies

Get all retention policies (admin only).

**Endpoint**: `GET /admin/retention-policies`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Response**: `200 OK`

```json
{
  "policies": [
    {
      "id": "uuid",
      "table_name": "users",
      "record_type": "deleted",
      "retention_period_days": 90,
      "retention_field": "deleted_at",
      "grace_period_days": 30,
      "legal_minimum_days": 0,
      "cleanup_method": "anonymize",
      "notification_days_before": 30,
      "is_active": true
    },
    {
      "id": "uuid",
      "table_name": "audit_events",
      "record_type": "all",
      "retention_period_days": 2555,
      "retention_field": "timestamp",
      "grace_period_days": 0,
      "legal_minimum_days": 2555,
      "cleanup_method": "hard_delete",
      "notification_days_before": 0,
      "is_active": true
    }
  ]
}
```

---

#### Update Retention Policy

Update retention period for a policy (admin only).

**Endpoint**: `PUT /admin/retention-policies/:policy_id`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Path Parameters**:

| Parameter | Type   | Required | Description         |
| --------- | ------ | -------- | ------------------- |
| policy_id | string | Yes      | Retention policy ID |

**Request Body**:

```json
{
  "retention_period_days": 120
}
```

**Validation**:

- retention_period_days must be ≥ legal_minimum_days
- Frontend displays warning if attempting to set below legal minimum

**Response**: `200 OK`

```json
{
  "message": "Retention policy updated successfully",
  "policy": {
    "id": "uuid",
    "retention_period_days": 120,
    "updated_at": "2026-01-16T10:00:00Z"
  }
}
```

**Error Responses**:

- `400 Bad Request`: Retention period below legal minimum
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not OWNER role
- `404 Not Found`: Policy not found
- `500 Internal Server Error`: Server error

---

#### Get Expired Record Count

Preview how many records would be cleaned up by a policy.

**Endpoint**: `GET /admin/retention-policies/:policy_id/expired-count`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Response**: `200 OK`

```json
{
  "policy_id": "uuid",
  "table_name": "users",
  "expired_count": 45,
  "oldest_record_date": "2025-10-10T12:00:00Z"
}
```

---

### Audit Trail

#### Get Audit Events

Query audit trail with filters.

**Endpoint**: `GET /admin/audit`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Query Parameters**:

| Parameter     | Type    | Required | Description                                         |
| ------------- | ------- | -------- | --------------------------------------------------- |
| date_range    | string  | No       | Date range in format "YYYY-MM-DD,YYYY-MM-DD"        |
| action_type   | string  | No       | Filter by action (create, update, delete)           |
| resource_type | string  | No       | Filter by resource type (user, order, tenant, etc.) |
| actor_id      | string  | No       | Filter by actor (user who performed action)         |
| limit         | integer | No       | Max results (default 100, max 1000)                 |

**Response**: `200 OK`

```json
{
  "events": [
    {
      "event_id": "uuid",
      "event_type": "USER_DELETED",
      "tenant_id": "uuid",
      "actor_id": "uuid",
      "actor_email": "admin@business.com",
      "action": "delete",
      "resource_type": "user",
      "resource_id": "uuid",
      "before_value": "{...encrypted...}",
      "after_value": null,
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "compliance_tag": "UU_PDP_Article_5",
      "timestamp": "2026-01-16T10:00:00Z"
    }
  ],
  "total": 1,
  "limit": 100
}
```

**Note**: `before_value` and `after_value` are encrypted to protect PII in audit logs.

---

### Compliance Reporting

#### Get Compliance Report

Generate compliance status report (admin only).

**Endpoint**: `GET /admin/compliance/report`

**Authentication**: Required (JWT)

**Authorization**: OWNER role only

**Response**: `200 OK`

```json
{
  "report_date": "2026-01-16T10:00:00Z",
  "encrypted_records": {
    "users": 1250,
    "guest_orders": 8940,
    "tenant_configs": 150
  },
  "active_consents": {
    "operational": 1400,
    "analytics": 890,
    "advertising": 450
  },
  "audit_events": {
    "total": 145680,
    "last_30_days": 12450,
    "oldest_event_date": "2019-01-16T00:00:00Z"
  },
  "retention_coverage": {
    "users": "100%",
    "guest_orders": "100%",
    "audit_events": "100%"
  },
  "compliance_status": "COMPLIANT",
  "issues": []
}
```

**Compliance Status Values**:

- `COMPLIANT`: All checks passed
- `WARNING`: Minor issues detected (e.g., missing optional consents)
- `NON_COMPLIANT`: Critical issues detected (e.g., unencrypted PII, missing required consents)

**Possible Issues**:

- `unencrypted_pii_detected`: PII found without encryption
- `missing_required_consents`: Tenant/guest without operational consent
- `audit_trail_gaps`: Missing audit events for critical operations
- `retention_violation`: Data retained beyond policy period

---

## Changelog

### 2026-01-16

- Added UU PDP Compliance API section
- Added consent management endpoints (grant, revoke, status)
- Added tenant data rights endpoints (view, export, delete team member)
- Added guest data rights endpoints (view order data, delete order data)
- Added retention policy management endpoints (list, update, expired count)
- Added audit trail query endpoint
- Added compliance reporting endpoint

### 2024-01-15

- Added notification history endpoint with pagination and filters
- Added resend notification endpoint with retry limit enforcement
- Added notification configuration management endpoints
- Added test notification endpoint with rate limiting
- Added user notification preferences endpoints
- Implemented comprehensive error handling and retry logic
- Added monitoring metrics for email delivery

### 2026-01-31

- Added analytics service with business insights dashboard endpoints
- Added sales overview endpoint with period comparison
- Added top products endpoint with revenue and quantity rankings
- Added top customers endpoint with PII masking
- Added operational tasks endpoint (delayed orders, low stock alerts)
- Added time series sales trend endpoint with configurable granularity
- Implemented Redis caching for analytics queries with dynamic TTL
- Added comprehensive query performance logging

---

## Analytics Service API

Base URL: `http://api-gateway:8080/api/v1`

**Authentication**: All endpoints require JWT authentication  
**Authorization**: Tenant Owner role required for all analytics endpoints  
**Tenant Isolation**: All queries automatically filtered by authenticated user's tenant_id

---

### Get Sales Overview

Get comprehensive sales analytics including metrics, trends, and category breakdown.

**Endpoint**: `GET /analytics/overview`

**Query Parameters**:

| Parameter  | Type   | Required    | Default    | Description                                                   |
| ---------- | ------ | ----------- | ---------- | ------------------------------------------------------------- |
| time_range | string | No          | this_month | Predefined time range (see options below)                     |
| start_date | string | Conditional | -          | Custom start date (YYYY-MM-DD), required if time_range=custom |
| end_date   | string | Conditional | -          | Custom end date (YYYY-MM-DD), required if time_range=custom   |

**Time Range Options**:

- `today` - Current day
- `yesterday` - Previous day
- `this_week` - Current week (Monday-Sunday)
- `last_week` - Previous week
- `this_month` - Current month (default)
- `last_month` - Previous month
- `last_30_days` - Last 30 days
- `last_90_days` - Last 90 days
- `this_year` - Current year
- `custom` - Custom date range (requires start_date and end_date)

**Response**: `200 OK`

```json
{
  "metrics": {
    "total_revenue": 125450.75,
    "total_orders": 342,
    "average_order_value": 366.81,
    "inventory_value": 45230.0,
    "revenue_change": 12.5,
    "orders_change": 8.3,
    "aov_change": 3.9,
    "previous_revenue": 111645.0,
    "previous_orders": 316,
    "previous_aov": 353.31,
    "start_date": "2026-01-01T00:00:00Z",
    "end_date": "2026-01-31T23:59:59Z"
  }
}
```

**Field Descriptions**:

- `total_revenue`: Sum of all completed order amounts in the period
- `total_orders`: Count of completed orders
- `average_order_value`: Total revenue / total orders
- `inventory_value`: Sum of (product cost × quantity) for all products
- `revenue_change`: Percentage change vs previous period
- `orders_change`: Percentage change in order count vs previous period
- `aov_change`: Percentage change in average order value vs previous period

**Error Responses**:

- `400 Bad Request`: Invalid time_range or date format
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not a tenant owner
- `500 Internal Server Error`: Server error

**Example Requests**:

```bash
# Get current month overview
curl -X GET "http://localhost:8080/api/v1/analytics/overview?time_range=this_month" \
  -H "Authorization: Bearer $TOKEN"

# Get custom date range
curl -X GET "http://localhost:8080/api/v1/analytics/overview?time_range=custom&start_date=2026-01-01&end_date=2026-01-31" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Top Products

Get rankings of top and bottom performing products by revenue and quantity sold.

**Endpoint**: `GET /analytics/top-products`

**Query Parameters**:

| Parameter  | Type    | Required    | Default    | Description                              |
| ---------- | ------- | ----------- | ---------- | ---------------------------------------- |
| time_range | string  | No          | this_month | Time range (see overview options)        |
| limit      | integer | No          | 5          | Number of products per ranking (1-20)    |
| start_date | string  | Conditional | -          | Custom start date (if time_range=custom) |
| end_date   | string  | Conditional | -          | Custom end date (if time_range=custom)   |

**Response**: `200 OK`

```json
{
  "top_by_revenue": [
    {
      "product_id": 42,
      "name": "Premium Coffee Beans",
      "quantity_sold": 156,
      "revenue": 4680.0,
      "rank": 1
    }
  ],
  "top_by_quantity": [
    {
      "product_id": 23,
      "name": "Bottled Water",
      "quantity_sold": 890,
      "revenue": 1780.0,
      "rank": 1
    }
  ],
  "bottom_by_revenue": [
    {
      "product_id": 78,
      "name": "Specialty Tea",
      "quantity_sold": 3,
      "revenue": 45.0,
      "rank": 1
    }
  ],
  "bottom_by_quantity": [
    {
      "product_id": 91,
      "name": "Limited Edition Mug",
      "quantity_sold": 1,
      "revenue": 25.0,
      "rank": 1
    }
  ]
}
```

**Error Responses**:

- `400 Bad Request`: Invalid parameters (limit out of range 1-20)
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not a tenant owner

**Example Request**:

```bash
curl -X GET "http://localhost:8080/api/v1/analytics/top-products?limit=10&time_range=last_30_days" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Top Customers

Get rankings of top customers by total spending and order count with masked PII.

**Endpoint**: `GET /analytics/top-customers`

**Query Parameters**:

| Parameter  | Type    | Required    | Default    | Description                              |
| ---------- | ------- | ----------- | ---------- | ---------------------------------------- |
| time_range | string  | No          | this_month | Time range (see overview options)        |
| limit      | integer | No          | 5          | Number of customers per ranking (1-20)   |
| start_date | string  | Conditional | -          | Custom start date (if time_range=custom) |
| end_date   | string  | Conditional | -          | Custom end date (if time_range=custom)   |

**Response**: `200 OK`

```json
{
  "top_by_spending": [
    {
      "customer_id": "cust-uuid-123",
      "name": "J***",
      "phone": "****1234",
      "email": "j***@example.com",
      "total_spent": 2450.0,
      "order_count": 8,
      "rank": 1
    }
  ],
  "top_by_orders": [
    {
      "customer_id": "cust-uuid-456",
      "name": "M***",
      "phone": "****5678",
      "email": "m***@company.com",
      "total_spent": 1890.0,
      "order_count": 15,
      "rank": 1
    }
  ]
}
```

**PII Masking**:

- **Name**: Shows only first character + asterisks (e.g., "John" → "J\*\*\*")
- **Phone**: Shows only last 4 digits (e.g., "+62812345678" → "\*\*\*\*5678")
- **Email**: Shows first character + domain (e.g., "john@example.com" → "j\*\*\*@example.com")

**Error Responses**:

- `400 Bad Request`: Invalid parameters
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not a tenant owner

**Example Request**:

```bash
curl -X GET "http://localhost:8080/api/v1/analytics/top-customers?limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Operational Tasks

Get actionable alerts for delayed orders and low stock products.

**Endpoint**: `GET /analytics/tasks`

**Query Parameters**: None

**Response**: `200 OK`

```json
{
  "delayed_orders": {
    "count": 3,
    "delayed_orders": [
      {
        "order_id": 1234,
        "order_reference": "ORD-2026-001234",
        "customer_phone": "****1234",
        "elapsed_minutes": 45,
        "created_at": "2026-01-31T10:15:00Z"
      }
    ]
  },
  "restock_alerts": {
    "count": 5,
    "restock_alerts": [
      {
        "product_id": 42,
        "product_name": "Premium Coffee Beans",
        "current_quantity": 8,
        "restock_threshold": 20,
        "units_needed": 12
      }
    ]
  }
}
```

**Alert Criteria**:

- **Delayed Orders**: Orders in 'pending' status for more than 30 minutes
- **Restock Alerts**: Products where current quantity ≤ restock threshold

**Error Responses**:

- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not a tenant owner
- `500 Internal Server Error`: Server error

**Example Request**:

```bash
curl -X GET "http://localhost:8080/api/v1/analytics/tasks" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Get Sales Trend

Get time series data for sales revenue and order count with configurable granularity.

**Endpoint**: `GET /analytics/sales-trend`

**Query Parameters**:

| Parameter   | Type   | Required | Description                                              |
| ----------- | ------ | -------- | -------------------------------------------------------- |
| granularity | string | Yes      | Time bucket size (daily/weekly/monthly/quarterly/yearly) |
| start_date  | string | Yes      | Start date (YYYY-MM-DD)                                  |
| end_date    | string | Yes      | End date (YYYY-MM-DD)                                    |

**Granularity Options**:

- `daily` - One data point per day (max 90 days range)
- `weekly` - One data point per week (max 52 weeks range)
- `monthly` - One data point per month (max 12 months range)
- `quarterly` - One data point per quarter (max 8 quarters range)
- `yearly` - One data point per year (max 5 years range)

**Response**: `200 OK`

```json
{
  "period": "2026-01-01 to 2026-01-31",
  "granularity": "daily",
  "start_date": "2026-01-01",
  "end_date": "2026-01-31",
  "revenue_data": [
    {
      "date": "2026-01-01",
      "label": "Jan 01",
      "value": 4250.5
    },
    {
      "date": "2026-01-02",
      "label": "Jan 02",
      "value": 3890.25
    }
  ],
  "orders_data": [
    {
      "date": "2026-01-01",
      "label": "Jan 01",
      "value": 12
    },
    {
      "date": "2026-01-02",
      "label": "Jan 02",
      "value": 11
    }
  ]
}
```

**Label Formats by Granularity**:

- **Daily**: "Jan 02" (short month + day)
- **Weekly**: "Week of Jan 02" (week start date)
- **Monthly**: "Jan 2026" (month + year)
- **Quarterly**: "2026 Q1" (year + quarter number)
- **Yearly**: "2026" (year only)

**Gap Filling**:
The API uses PostgreSQL `generate_series` to ensure complete date ranges. Dates with no sales will have `value: 0` to ensure chart continuity.

**Validation Rules**:

- `end_date` must be ≥ `start_date`
- Both dates must not be in the future
- Granularity must be one of the valid options

**Error Responses**:

- `400 Bad Request`: Invalid granularity, date format, or date range
- `401 Unauthorized`: Missing or invalid JWT token
- `403 Forbidden`: User is not a tenant owner
- `500 Internal Server Error`: Server error

**Example Requests**:

```bash
# Daily trend for last 30 days
curl -X GET "http://localhost:8080/api/v1/analytics/sales-trend?granularity=daily&start_date=2026-01-01&end_date=2026-01-31" \
  -H "Authorization: Bearer $TOKEN"

# Monthly trend for last year
curl -X GET "http://localhost:8080/api/v1/analytics/sales-trend?granularity=monthly&start_date=2025-01-01&end_date=2025-12-31" \
  -H "Authorization: Bearer $TOKEN"

# Quarterly trend
curl -X GET "http://localhost:8080/api/v1/analytics/sales-trend?granularity=quarterly&start_date=2024-01-01&end_date=2025-12-31" \
  -H "Authorization: Bearer $TOKEN"
```

---

### Performance Characteristics

**Caching Strategy**:

- All analytics queries are cached in Redis
- Cache TTL varies by data freshness:
  - Current period (today, this week, this month): 5 minutes
  - Historical data: 1 hour
- Cache keys include tenant_id and query parameters

**Expected Response Times**:

- Cached queries: < 50ms
- Uncached queries: < 300ms (p95)
- Time series queries with 365 data points: < 500ms (p95)

**Query Performance Logging**:
All endpoints log `query_time_ms` for monitoring:

```json
{
  "level": "info",
  "tenant_id": 123,
  "time_range": "this_month",
  "query_time_ms": 87,
  "message": "Sales overview retrieved successfully"
}
```

**Rate Limiting**: Follow standard API gateway rate limits (100 requests/minute per tenant)

---

### Common Error Codes

| Status | Error Code                | Description                                               |
| ------ | ------------------------- | --------------------------------------------------------- |
| 400    | `invalid_time_range`      | Invalid time range parameter                              |
| 400    | `invalid_date_format`     | Date must be YYYY-MM-DD format                            |
| 400    | `invalid_date_range`      | End date must be after start date                         |
| 400    | `future_date_not_allowed` | Dates in the future are not allowed                       |
| 400    | `invalid_granularity`     | Granularity must be daily/weekly/monthly/quarterly/yearly |
| 400    | `invalid_limit`           | Limit must be between 1 and 20                            |
| 401    | `unauthorized`            | Missing or invalid JWT token                              |
| 403    | `forbidden`               | User is not a tenant owner                                |
| 500    | `internal_error`          | Internal server error                                     |

---

### Security & Privacy

**Authentication**:

- All endpoints require valid JWT token in Authorization header
- Token must contain valid tenant_id claim

**Authorization**:

- Only users with "tenant_owner" role can access analytics
- Middleware validates role before query execution

**Tenant Isolation**:

- All database queries automatically filtered by tenant_id from JWT
- Row-Level Security (RLS) policies enforce tenant boundaries
- Zero cross-tenant data leakage possible

**PII Protection**:

- Customer names masked to first character + asterisks
- Phone numbers masked to show only last 4 digits
- Email addresses masked to show first character + domain
- All customer PII encrypted at rest using Vault transit encryption
- Decryption and masking happen in repository layer before API response

**Logging**:

- No PII logged (phone, email, name masked in logs)
- Query parameters logged for debugging
- Performance metrics (query_time_ms) logged for monitoring
- Structured logging with zerolog

---
