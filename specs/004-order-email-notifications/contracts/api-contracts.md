# API Contracts: Order Email Notifications

**Feature**: 004-order-email-notifications  
**Date**: 2025-12-11  
**Phase**: 1 - Design

This document defines all API contracts for the order email notifications feature, including HTTP REST endpoints and asynchronous event schemas.

---

## 1. Event Contracts (Kafka)

### 1.1 OrderPaidEvent

**Topic**: `notification-events`  
**Producer**: order-service  
**Consumer**: notification-service  
**Key**: `tenant_id` (for partition ordering)

#### Event Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["event_id", "event_type", "tenant_id", "timestamp", "metadata"],
  "properties": {
    "event_id": {
      "type": "string",
      "format": "uuid",
      "description": "Unique identifier for this event"
    },
    "event_type": {
      "type": "string",
      "const": "order.paid",
      "description": "Event type identifier"
    },
    "tenant_id": {
      "type": "string",
      "format": "uuid",
      "description": "Tenant that owns this order"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "When this event was created (ISO 8601)"
    },
    "metadata": {
      "type": "object",
      "required": [
        "order_id",
        "order_reference",
        "transaction_id",
        "customer_name",
        "customer_phone",
        "delivery_type",
        "items",
        "subtotal_amount",
        "delivery_fee",
        "total_amount",
        "payment_method",
        "paid_at"
      ],
      "properties": {
        "order_id": {
          "type": "string",
          "format": "uuid"
        },
        "order_reference": {
          "type": "string",
          "pattern": "^[A-Z0-9-]+$",
          "example": "ORD-001"
        },
        "transaction_id": {
          "type": "string",
          "minLength": 1,
          "description": "Payment provider transaction ID (for duplicate detection)"
        },
        "customer_name": {
          "type": "string",
          "minLength": 1,
          "maxLength": 255
        },
        "customer_phone": {
          "type": "string",
          "pattern": "^\\+?[0-9]{8,20}$"
        },
        "customer_email": {
          "type": "string",
          "format": "email",
          "nullable": true,
          "description": "Optional customer email for receipt"
        },
        "delivery_type": {
          "type": "string",
          "enum": ["pickup", "delivery", "dine_in"]
        },
        "delivery_address": {
          "type": "string",
          "nullable": true,
          "description": "Required if delivery_type is 'delivery'"
        },
        "table_number": {
          "type": "string",
          "nullable": true,
          "description": "Optional for dine_in orders"
        },
        "items": {
          "type": "array",
          "minItems": 1,
          "items": {
            "type": "object",
            "required": ["product_id", "product_name", "quantity", "unit_price", "total_price"],
            "properties": {
              "product_id": {
                "type": "string",
                "format": "uuid"
              },
              "product_name": {
                "type": "string",
                "minLength": 1
              },
              "quantity": {
                "type": "integer",
                "minimum": 1
              },
              "unit_price": {
                "type": "integer",
                "minimum": 0,
                "description": "Price in smallest currency unit (e.g., cents)"
              },
              "total_price": {
                "type": "integer",
                "minimum": 0,
                "description": "quantity * unit_price"
              }
            }
          }
        },
        "subtotal_amount": {
          "type": "integer",
          "minimum": 0,
          "description": "Sum of all item totals"
        },
        "delivery_fee": {
          "type": "integer",
          "minimum": 0
        },
        "total_amount": {
          "type": "integer",
          "minimum": 0,
          "description": "subtotal_amount + delivery_fee"
        },
        "payment_method": {
          "type": "string",
          "example": "QRIS"
        },
        "paid_at": {
          "type": "string",
          "format": "date-time",
          "description": "When payment was confirmed"
        }
      }
    }
  }
}
```

#### Example Event

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "order.paid",
  "tenant_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "timestamp": "2025-12-11T10:30:00Z",
  "metadata": {
    "order_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "order_reference": "ORD-001",
    "transaction_id": "midtrans-1234567890",
    "customer_name": "John Doe",
    "customer_phone": "+6281234567890",
    "customer_email": "john.doe@example.com",
    "delivery_type": "delivery",
    "delivery_address": "Jl. Sudirman No. 123, Jakarta 12345",
    "table_number": null,
    "items": [
      {
        "product_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
        "product_name": "Nasi Goreng",
        "quantity": 2,
        "unit_price": 25000,
        "total_price": 50000
      },
      {
        "product_id": "a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6",
        "product_name": "Es Teh Manis",
        "quantity": 2,
        "unit_price": 5000,
        "total_price": 10000
      }
    ],
    "subtotal_amount": 60000,
    "delivery_fee": 10000,
    "total_amount": 70000,
    "payment_method": "QRIS",
    "paid_at": "2025-12-11T10:30:00Z"
  }
}
```

#### Event Processing Guarantees

- **Delivery**: At-least-once (Kafka default)
- **Ordering**: Per partition (keyed by tenant_id)
- **Idempotency**: Consumer checks transaction_id for duplicates
- **Retry**: Consumer retries on transient failures (connection errors)
- **Dead Letter**: Events that fail processing >3 times go to DLQ topic

---

## 2. HTTP REST API Contracts

### 2.1 List Staff Notification Preferences

**Endpoint**: `GET /api/v1/users/notification-preferences`  
**Service**: user-service (or notification-service if centralized)  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
```

**Query Parameters**:
- None (returns all staff for authenticated tenant)

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "name": "Jane Manager",
        "email": "jane@restaurant.com",
        "role": "manager",
        "receive_order_notifications": true
      },
      {
        "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
        "name": "Bob Cashier",
        "email": "bob@restaurant.com",
        "role": "cashier",
        "receive_order_notifications": false
      }
    ]
  }
}
```

#### Error Responses

**401 Unauthorized**:
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or missing authentication token"
  }
}
```

**403 Forbidden**:
```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "User does not have permission to view notification preferences"
  }
}
```

---

### 2.2 Update Staff Notification Preference

**Endpoint**: `PATCH /api/v1/users/:user_id/notification-preferences`  
**Service**: user-service  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
Content-Type: application/json
```

**Path Parameters**:
- `user_id` (UUID): The user to update

**Body**:
```json
{
  "receive_order_notifications": true
}
```

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "receive_order_notifications": true,
    "updated_at": "2025-12-11T10:30:00Z"
  }
}
```

#### Error Responses

**400 Bad Request**:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_INPUT",
    "message": "receive_order_notifications must be a boolean",
    "details": {
      "field": "receive_order_notifications",
      "received": "string"
    }
  }
}
```

**404 Not Found**:
```json
{
  "success": false,
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User with ID 3fa85f64-5717-4562-b3fc-2c963f66afa6 not found"
  }
}
```

---

### 2.3 Send Test Notification

**Endpoint**: `POST /api/v1/notifications/test`  
**Service**: notification-service  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
Content-Type: application/json
```

**Body**:
```json
{
  "notification_type": "order_staff",
  "recipient_email": "test@example.com"
}
```

**Fields**:
- `notification_type` (enum): One of `order_staff`, `order_customer`
- `recipient_email` (string, email): Email address to send test to (overrides stored test_email in notification_configs if provided; if omitted, uses stored test_email)

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "notification_id": "a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6",
    "status": "sent",
    "sent_at": "2025-12-11T10:30:00Z",
    "recipient": "test@example.com",
    "message": "Test notification sent successfully"
  }
}
```

#### Error Responses

**400 Bad Request**:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_EMAIL",
    "message": "recipient_email must be a valid email address",
    "details": {
      "field": "recipient_email",
      "value": "invalid-email"
    }
  }
}
```

**429 Too Many Requests**:
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many test notifications sent. Please try again in 60 seconds.",
    "retry_after": 60
  }
}
```

---

### 2.4 List Notification History

**Endpoint**: `GET /api/v1/notifications/history`  
**Service**: notification-service  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
```

**Query Parameters**:
- `page` (integer, default: 1): Page number
- `page_size` (integer, default: 20, max: 100): Items per page
- `order_reference` (string, optional): Filter by order reference
- `status` (enum, optional): Filter by status (pending, sent, failed, cancelled)
- `type` (enum, optional): Filter by type (order_staff, order_customer)
- `start_date` (ISO 8601, optional): Filter by created_at >= start_date
- `end_date` (ISO 8601, optional): Filter by created_at <= end_date

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "notifications": [
      {
        "id": "a1b2c3d4-5e6f-7g8h-9i0j-k1l2m3n4o5p6",
        "event_type": "order.paid.staff",
        "type": "email",
        "recipient": "manager@restaurant.com",
        "subject": "New Order Paid: ORD-001",
        "status": "sent",
        "sent_at": "2025-12-11T10:30:15Z",
        "retry_count": 0,
        "created_at": "2025-12-11T10:30:10Z",
        "order_reference": "ORD-001"
      },
      {
        "id": "b2c3d4e5-6f7g-8h9i-0j1k-l2m3n4o5p6q7",
        "event_type": "order.paid.customer",
        "type": "email",
        "recipient": "customer@example.com",
        "subject": "Your Order Receipt: ORD-001",
        "status": "failed",
        "failed_at": "2025-12-11T10:30:20Z",
        "error_msg": "SMTP connection timeout",
        "retry_count": 1,
        "created_at": "2025-12-11T10:30:10Z",
        "order_reference": "ORD-001"
      }
    ],
    "pagination": {
      "current_page": 1,
      "page_size": 20,
      "total_items": 156,
      "total_pages": 8
    }
  }
}
```

#### Error Responses

**400 Bad Request**:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "page_size must be between 1 and 100",
    "details": {
      "field": "page_size",
      "value": 150
    }
  }
}
```

---

### 2.5 Resend Failed Notification

**Endpoint**: `POST /api/v1/notifications/:notification_id/resend`  
**Service**: notification-service  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
```

**Path Parameters**:
- `notification_id` (UUID): The notification to resend

**Body**: None

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "notification_id": "b2c3d4e5-6f7g-8h9i-0j1k-l2m3n4o5p6q7",
    "status": "sent",
    "sent_at": "2025-12-11T11:00:00Z",
    "retry_count": 2,
    "message": "Notification resent successfully"
  }
}
```

#### Error Responses

**404 Not Found**:
```json
{
  "success": false,
  "error": {
    "code": "NOTIFICATION_NOT_FOUND",
    "message": "Notification with ID b2c3d4e5-6f7g-8h9i-0j1k-l2m3n4o5p6q7 not found"
  }
}
```

**409 Conflict**:
```json
{
  "success": false,
  "error": {
    "code": "ALREADY_SENT",
    "message": "Notification already successfully sent and cannot be resent"
  }
}
```

**429 Too Many Requests**:
```json
{
  "success": false,
  "error": {
    "code": "MAX_RETRIES_EXCEEDED",
    "message": "Maximum retry attempts (3) exceeded. Cannot resend.",
    "details": {
      "retry_count": 3,
      "max_retries": 3
    }
  }
}
```

---

### 2.6 Get Notification Config

**Endpoint**: `GET /api/v1/notifications/config`  
**Service**: notification-service  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
```

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "tenant_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "order_notifications_enabled": true,
    "test_mode": false,
    "test_email": null,
    "created_at": "2025-11-01T00:00:00Z",
    "updated_at": "2025-12-01T10:00:00Z"
  }
}
```

---

### 2.7 Update Notification Config

**Endpoint**: `PATCH /api/v1/notifications/config`  
**Service**: notification-service  
**Auth**: Required (tenant admin or owner)

#### Request

**Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
Content-Type: application/json
```

**Body**:
```json
{
  "order_notifications_enabled": true,
  "test_mode": false,
  "test_email": "test@restaurant.com"
}
```

**Fields** (all optional):
- `order_notifications_enabled` (boolean): Global enable/disable
- `test_mode` (boolean): Send emails only to test_email
- `test_email` (string, email, nullable): Email for test mode

#### Response

**Status**: 200 OK

**Body**:
```json
{
  "success": true,
  "data": {
    "tenant_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "order_notifications_enabled": true,
    "test_mode": false,
    "test_email": "test@restaurant.com",
    "updated_at": "2025-12-11T10:30:00Z"
  }
}
```

#### Error Responses

**400 Bad Request**:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CONFIG",
    "message": "test_email is required when test_mode is true",
    "details": {
      "test_mode": true,
      "test_email": null
    }
  }
}
```

---

## 3. Contract Testing Requirements

### 3.1 Event Schema Validation

**Tool**: JSON Schema validator  
**Test**: Kafka producer must publish events matching OrderPaidEvent schema  
**Validation**: Consumer rejects events that don't match schema

**Test Case**:
```go
func TestOrderPaidEventSchema(t *testing.T) {
    // Valid event should pass
    validEvent := createValidOrderPaidEvent()
    assert.NoError(t, validateEventSchema(validEvent))
    
    // Missing required field should fail
    invalidEvent := createOrderPaidEventWithoutTransactionID()
    assert.Error(t, validateEventSchema(invalidEvent))
}
```

### 3.2 API Contract Tests

**Tool**: Pact or Spring Cloud Contract  
**Test**: Consumer-driven contracts between services

**Example Pact**:
```go
// notification-service expects order-service to publish order.paid events
func TestOrderServicePublishesOrderPaidEvent(t *testing.T) {
    pact := dsl.Pact{
        Consumer: "notification-service",
        Provider: "order-service",
    }
    
    pact.AddInteraction().
        Given("An order transitions to PAID status").
        UponReceiving("An order.paid event").
        WithRequest(dsl.Request{
            Method: "MESSAGE",
            Headers: dsl.MapMatcher{
                "Content-Type": "application/json",
            },
        }).
        WillRespondWith(dsl.Response{
            Status: 200,
            Body: dsl.Match(OrderPaidEvent{}),
        })
        
    assert.NoError(t, pact.Verify())
}
```

### 3.3 Backward Compatibility

**Rule**: Event schema changes must be backward compatible  
**Allowed**:
- Add optional fields to metadata
- Add new event types

**Forbidden**:
- Remove required fields
- Change field types
- Rename fields

**Validation**: Schema versioning + compatibility checker in CI/CD

---

## 4. Error Code Reference

| Code | HTTP Status | Description |
|------|-------------|-------------|
| UNAUTHORIZED | 401 | Missing or invalid authentication token |
| FORBIDDEN | 403 | User lacks permission for this operation |
| USER_NOT_FOUND | 404 | User ID does not exist |
| NOTIFICATION_NOT_FOUND | 404 | Notification ID does not exist |
| INVALID_INPUT | 400 | Request body validation failed |
| INVALID_EMAIL | 400 | Email format invalid |
| INVALID_PARAMETER | 400 | Query parameter validation failed |
| INVALID_CONFIG | 400 | Configuration validation failed |
| ALREADY_SENT | 409 | Notification already successfully sent |
| MAX_RETRIES_EXCEEDED | 429 | Retry limit reached |
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| INTERNAL_ERROR | 500 | Unexpected server error |

---

## 5. Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| POST /notifications/test | 10 requests | per minute per tenant |
| POST /notifications/:id/resend | 20 requests | per minute per tenant |
| GET /notifications/history | 100 requests | per minute per tenant |
| PATCH /notifications/config | 10 requests | per minute per tenant |
| PATCH /users/:id/notification-preferences | 50 requests | per minute per tenant |

**Implementation**: Redis-based rate limiter in API Gateway or service middleware.

---

## Contract Summary

**Event Contracts**: 1 (OrderPaidEvent on Kafka)  
**HTTP Endpoints**: 7 REST APIs  
**Authentication**: JWT bearer token + tenant header  
**Versioning**: API v1, event schema v1  
**Testing**: JSON Schema + Pact contract tests  
**Error Handling**: Structured error responses with codes  
**Rate Limiting**: Per-tenant, per-endpoint limits

**Ready for**: Implementation and testing phases.
