# Order Email Notifications Feature

**Feature Branch:** `004-order-email-notifications`  
**Status:** ✅ Complete  
**Last Updated:** 2024-01-15

---

## 📋 Overview

The Order Email Notifications feature provides comprehensive email notification capabilities for the POS system, including:

1. **Staff Notifications**: Real-time order alerts sent to kitchen/counter staff when orders are paid
2. **Customer Receipts**: Professional invoice emails sent to customers after payment
3. **Notification Preferences**: Admin dashboard to configure staff notification settings and test emails
4. **Notification History**: Audit log of all sent notifications with filtering, pagination, and resend capability

### Business Value

- **Operational Efficiency**: Staff receive instant order notifications without checking dashboard
- **Customer Experience**: Professional receipts improve brand perception and reduce support inquiries
- **Flexibility**: Admins can customize notification settings per staff member
- **Reliability**: Comprehensive error handling with automatic retries ensures delivery
- **Observability**: Full audit trail with metrics for monitoring and debugging

---

## 🏗️ Architecture

### System Components

```
┌─────────────────┐
│  Order Service  │
│  (Publisher)    │
└────────┬────────┘
         │
         │ order.paid event
         ▼
    ┌────────┐
    │ Kafka  │
    └────┬───┘
         │
         │ consume events
         ▼
┌────────────────────┐
│ Notification       │
│ Service            │ ◄────── API requests ────── Frontend
│                    │                              Dashboard
│ - Event Consumer   │
│ - Email Provider   │
│ - SMTP Client      │
│ - Template Engine  │
└─────────┬──────────┘
          │
          │ query/update
          ▼
     ┌─────────┐
     │ PostgreSQL │
     │            │
     │ - notifications
     │ - notification_configs
     └────────────┘
```

### Event Flow

1. Customer completes payment in Order Service
2. Order Service publishes `order.paid` event to Kafka
3. Notification Service consumes event from Kafka
4. Service checks for duplicate notifications (by transaction_id)
5. Service queries notification preferences from database
6. Service renders email templates with order data
7. Service attempts email delivery via SMTP with retry logic
8. Service updates notification status in database
9. Frontend can query notification history via REST API

---

## 🎯 User Stories

### User Story 1: Staff Notifications (P1)

**As a** restaurant owner  
**I want** staff to receive email notifications when orders are paid  
**So that** they can start preparing orders immediately without checking the dashboard

**Acceptance Criteria:**
- ✅ Email sent to enabled staff when order is paid
- ✅ Email includes order details (items, quantity, customer, delivery info)
- ✅ Duplicate notifications prevented (same transaction_id)
- ✅ Failed emails retry automatically with exponential backoff
- ✅ Notification status tracked in database

**Implementation:**
- Event: `order.paid` → Kafka
- Consumer: `handleOrderPaid()` in notification-service
- Template: `order_staff_notification.html`
- Repository: `HasSentOrderNotification()`, `Create()`

---

### User Story 2: Customer Receipts (P1)

**As a** customer  
**I want** to receive a receipt email after paying for my order  
**So that** I have a record of my purchase

**Acceptance Criteria:**
- ✅ Email sent to customer when order is paid (if email provided)
- ✅ Email includes itemized invoice with totals
- ✅ Professional HTML template with branding
- ✅ Failed emails retry automatically
- ✅ Duplicate receipts prevented

**Implementation:**
- Event: `order.paid` → Kafka  
- Consumer: `sendCustomerReceipt()` in notification-service
- Template: `order_invoice.html`
- Repository: `HasSentOrderNotification()`, `Create()`

---

### User Story 3: Notification Preferences (P2)

**As a** tenant admin  
**I want** to configure which staff members receive order notifications  
**So that** I can control notification routing and send test emails

**Acceptance Criteria:**
- ✅ Admin dashboard to view all staff notification settings
- ✅ Toggle switches to enable/disable notifications per staff
- ✅ "Send Test Email" button to verify SMTP configuration
- ✅ Test emails use sample order data
- ✅ Rate limiting (5 test emails per minute)

**Implementation:**
- Frontend: `NotificationSettings.tsx` component
- API: GET/PATCH `/users/:id/notification-preferences` (user-service)
- API: POST `/notifications/test` (notification-service)
- Middleware: Rate limiting on test endpoint

---

### User Story 4: Notification History (P3)

**As a** tenant admin  
**I want** to view a log of all sent email notifications  
**So that** I can audit delivery and resend failed notifications

**Acceptance Criteria:**
- ✅ Paginated list of notifications (20 per page)
- ✅ Filter by order reference, status, type, date range
- ✅ Display status badges (sent/pending/failed)
- ✅ View error messages for failed notifications
- ✅ Resend button for failed notifications (max 3 retries)
- ✅ Auto-refresh after successful resend

**Implementation:**
- Frontend: `NotificationHistory.tsx` component
- API: GET `/notifications/history` with pagination/filters
- API: POST `/notifications/:id/resend` with retry enforcement
- Repository: Dynamic SQL query builder with filters

---

## 🗄️ Database Schema

### `notifications` Table

```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('staff', 'customer', 'email', 'push')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
    subject VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    sent_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_msg TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_tenant ON notifications(tenant_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created ON notifications(created_at DESC);
CREATE INDEX idx_notifications_metadata_gin ON notifications USING gin(metadata);
```

**Key Fields:**
- `type`: `staff` (order alerts) or `customer` (receipts)
- `status`: `pending` → `sent` or `failed`
- `metadata`: JSONB containing `order_reference`, `transaction_id`, `event_type`
- `retry_count`: Tracks number of send attempts (max 3)
- `error_msg`: Detailed error for debugging failed notifications

### `notification_configs` Table

```sql
CREATE TABLE notification_configs (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    staff_notification_enabled BOOLEAN DEFAULT true,
    customer_receipt_enabled BOOLEAN DEFAULT true,
    admin_email VARCHAR(255),
    staff_emails TEXT[], -- PostgreSQL array
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Purpose:** Tenant-level notification settings (global enable/disable, admin email, staff list)

### `users` Table Extension

```sql
ALTER TABLE users ADD COLUMN staff_notifications_enabled BOOLEAN DEFAULT true;
```

**Purpose:** Per-user notification preference (allows individual staff to opt-out)

---

## 📧 Email Templates

All templates located in `backend/notification-service/templates/`

### Staff Notification Template

**File:** `order_staff_notification.html`

**Template Data:**
```go
{
    "OrderReference": "ORD-2024-001",
    "CustomerName": "John Doe",
    "CustomerPhone": "+62 812-3456-7890",
    "DeliveryType": "delivery", // or "pickup", "dine_in"
    "DeliveryAddress": "Jl. Example No. 123",
    "TableNumber": "5",
    "Items": [
        {
            "ProductName": "Nasi Goreng",
            "Quantity": 2,
            "UnitPrice": 25000,
            "TotalPrice": 50000
        }
    ],
    "SubtotalAmount": 50000,
    "DeliveryFee": 10000,
    "TotalAmount": 60000,
    "PaidAt": "2024-01-15T10:30:00Z"
}
```

**Features:**
- Conditional sections (delivery address, table number)
- Itemized order list
- Currency formatting helper (`formatCurrency`)
- Date/time formatting

### Customer Invoice Template

**File:** `order_invoice.html`

**Template Data:** Same as staff notification

**Features:**
- Professional invoice layout
- Company branding section
- Itemized billing
- Payment confirmation
- Footer with contact information

---

## 🔧 Configuration

### Environment Variables

#### Notification Service

```bash
# SMTP Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=noreply@yourcompany.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=POS System <noreply@yourcompany.com>
SMTP_RETRY_ATTEMPTS=3

# Template Configuration
TEMPLATE_DIR=./templates

# Frontend URL (for links in emails)
FRONTEND_DOMAIN=https://pos.yourcompany.com

# Kafka Configuration
KAFKA_BROKER=localhost:9092
KAFKA_GROUP_ID=notification-service
KAFKA_TOPIC=order-events

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/pos_db?sslmode=disable
```

#### User Service

```bash
# Database (for notification preferences)
DATABASE_URL=postgresql://user:pass@localhost:5432/pos_db?sslmode=disable
```

### SMTP Provider Setup

**Gmail Example:**

1. Enable 2-Factor Authentication on Google Account
2. Generate App Password: https://myaccount.google.com/apppasswords
3. Use App Password as `SMTP_PASSWORD`
4. Set `SMTP_HOST=smtp.gmail.com` and `SMTP_PORT=587`

**SendGrid Example:**

1. Create SendGrid account and API key
2. Set `SMTP_HOST=smtp.sendgrid.net`
3. Set `SMTP_PORT=587`
4. Set `SMTP_USERNAME=apikey`
5. Set `SMTP_PASSWORD=<your-sendgrid-api-key>`

**AWS SES Example:**

1. Verify sender email in AWS SES console
2. Create SMTP credentials
3. Set `SMTP_HOST=email-smtp.<region>.amazonaws.com`
4. Set `SMTP_PORT=587`
5. Use SMTP credentials from SES

---

## 🔄 Error Handling & Resilience

### Error Classification

The notification service classifies SMTP errors into categories:

| Error Type | Retryable | Description |
|------------|-----------|-------------|
| Connection | ✅ Yes | Network issues, server unavailable |
| Timeout | ✅ Yes | Request timeout, deadline exceeded |
| Rate Limited | ✅ Yes | SMTP provider rate limit hit |
| Authentication | ❌ No | Invalid credentials, auth failed |
| Invalid Recipient | ❌ No | Bad email address, mailbox unavailable |

### Retry Strategy

**Exponential Backoff:**
- Attempt 1: Immediate
- Attempt 2: After 2 seconds
- Attempt 3: After 4 seconds  
- Attempt 4: After 8 seconds

**Maximum Retries:** 3 (configurable via `SMTP_RETRY_ATTEMPTS`)

**Example Log:**
```
[EMAIL] Retry attempt 1/3 after 2s
[EMAIL] Retryable error (attempt 2/4): SMTP connection failed
[EMAIL] Successfully sent after 2 retries
```

### Duplicate Prevention

**Mechanism:** Check `transaction_id` before sending

```go
alreadySent, err := s.repo.HasSentOrderNotification(ctx, tenantID, transactionID)
if alreadySent {
    log.Printf("[DUPLICATE_NOTIFICATION] transaction_id=%s - Skipping duplicate", transactionID)
    return nil
}
```

**Why It Matters:**
- Kafka may deliver messages multiple times (at-least-once delivery)
- Network retries can cause duplicate events
- Prevents customers from receiving multiple receipts

---

## 📊 Monitoring & Metrics

### Structured Logging

All metrics logged with `[METRIC]` prefix for parsing:

```
[METRIC] notification.email.sent=1 [retry_count=0]
[METRIC] notification.email.failed=1 [error_type=connection, retryable=true]
[METRIC] notification.email.duration_ms=1234
[METRIC] notification.duplicate.prevented=1 [tenant_id=..., payment_method=qris]
```

### Key Metrics

| Metric | Type | Tags | Purpose |
|--------|------|------|---------|
| `notification.email.sent` | Counter | retry_count | Track successful deliveries |
| `notification.email.failed` | Counter | error_type, retryable | Track failures by cause |
| `notification.email.duration_ms` | Gauge | - | Monitor delivery performance |
| `notification.duplicate.prevented` | Counter | tenant_id, payment_method | Track duplicate prevention |

### Alerting Recommendations

1. **High Failure Rate**: Alert if `failed / (sent + failed) > 10%` over 5 minutes
2. **SMTP Auth Failures**: Alert on any `error_type=auth` (indicates config issue)
3. **High Latency**: Alert if `duration_ms > 5000ms` p95 over 5 minutes
4. **Duplicate Spike**: Alert if `duplicate.prevented > 100` per hour (may indicate event replay)

---

## 🧪 Testing

### Test Notification Endpoint

**Purpose:** Verify SMTP configuration without waiting for real orders

**Endpoint:** `POST /api/v1/notifications/test`

**Request:**
```json
{
  "recipient_email": "test@example.com",
  "notification_type": "staff"
}
```

**Behavior:**
- Generates sample order data
- Renders template with test data
- Sends email via configured SMTP
- Returns notification ID

**Rate Limit:** 5 requests per minute per user

### Development Mode

When `SMTP_USERNAME` is empty, emails are logged instead of sent:

```
[EMAIL] To: test@example.com, Subject: New Order Received - ORD-TEST-001
<rendered HTML content>
```

This allows testing template rendering without SMTP setup.

### Integration Tests

Located in `frontend/tests/integration/`:

- `notification-settings.test.tsx`: Test NotificationSettings component
- `notification-history.test.tsx`: Test NotificationHistory component

Run with: `npm test`

### E2E Tests

Located in `frontend/tests/e2e/`:

- `notification-config.spec.ts`: Test full notification preferences workflow
- `notification-history.spec.ts`: Test history viewing and resend workflow

Run with: `npx playwright test`

### Contract Tests

Located in `backend/notification-service/tests/contract/`:

- `notification_history_test.go`: Test GET /notifications/history API contract
- `resend_notification_test.go`: Test POST /notifications/:id/resend API contract

Run with: `go test ./tests/contract/...`

---

## 🚀 Deployment

### Prerequisites

- [x] PostgreSQL with migrations applied
- [x] Kafka running and accessible
- [x] SMTP credentials configured
- [x] Frontend domain configured in env vars

### Deployment Checklist

1. **Database Migrations**
   ```bash
   # Run migrations
   cd backend/migrations
   ./run-migrations.sh
   ```

2. **Environment Variables**
   - Set all SMTP_* variables
   - Set FRONTEND_DOMAIN to production URL
   - Set KAFKA_BROKER to production broker
   - Set DATABASE_URL to production database

3. **Service Deployment**
   ```bash
   # Build notification-service
   cd backend/notification-service
   go build -o notification-service.bin
   
   # Deploy binary
   ./notification-service.bin
   ```

4. **Verify SMTP Connection**
   ```bash
   # Send test email via API
   curl -X POST https://api.yourcompany.com/api/v1/notifications/test \
     -H "Authorization: Bearer $JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"recipient_email": "admin@yourcompany.com", "notification_type": "staff"}'
   ```

5. **Monitor Logs**
   ```bash
   # Watch for metrics and errors
   tail -f notification-service.log | grep -E '\[METRIC\]|\[ERROR\]|\[EMAIL_SEND'
   ```

6. **Test End-to-End**
   - Create test order in dashboard
   - Complete payment
   - Verify staff notification received
   - Verify customer receipt received (if email provided)
   - Check notification history in dashboard

---

## 🔍 Troubleshooting

### Emails Not Sending

**Symptoms:** Notifications created but status remains `pending`

**Diagnosis:**
1. Check SMTP environment variables are set
2. Check SMTP credentials are valid
3. Check network connectivity to SMTP server
4. Check service logs for `[EMAIL_SEND_FAILED]` entries

**Solutions:**
```bash
# Test SMTP connection manually
telnet smtp.gmail.com 587

# Check environment variables
env | grep SMTP

# Check service logs
grep "EMAIL_SEND_FAILED" notification-service.log

# Verify credentials with test notification
curl -X POST http://localhost:8080/api/v1/notifications/test \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{"recipient_email": "test@example.com", "notification_type": "staff"}'
```

### Duplicate Notifications

**Symptoms:** Same order notification sent multiple times

**Diagnosis:**
1. Check `retry_count` in notifications table
2. Check logs for `[DUPLICATE_NOTIFICATION]` entries
3. Verify transaction_id is unique per order

**Solutions:**
```sql
-- Check for duplicate notifications
SELECT transaction_id, COUNT(*) 
FROM notifications 
WHERE metadata->>'transaction_id' IS NOT NULL 
GROUP BY metadata->>'transaction_id' 
HAVING COUNT(*) > 1;

-- Check duplicate prevention is working
SELECT COUNT(*) FROM notifications 
WHERE metadata @> '{"event_type": "order.paid"}'::jsonb;
```

### High Email Failure Rate

**Symptoms:** Many notifications with status `failed`

**Diagnosis:**
1. Check error_msg field for patterns
2. Check SMTP provider status page
3. Check rate limiting errors (429)

**Solutions:**
```sql
-- Analyze failure reasons
SELECT error_msg, COUNT(*) as count
FROM notifications
WHERE status = 'failed'
GROUP BY error_msg
ORDER BY count DESC
LIMIT 10;

-- Check retryable vs permanent failures
SELECT 
  CASE 
    WHEN error_msg LIKE '%connection%' OR error_msg LIKE '%timeout%' THEN 'retryable'
    WHEN error_msg LIKE '%auth%' OR error_msg LIKE '%invalid%' THEN 'permanent'
    ELSE 'unknown'
  END as error_category,
  COUNT(*) as count
FROM notifications
WHERE status = 'failed'
GROUP BY error_category;
```

### Template Rendering Errors

**Symptoms:** Emails sent with "Template not found" or "Error rendering template"

**Diagnosis:**
1. Check TEMPLATE_DIR environment variable
2. Verify template files exist
3. Check file permissions
4. Check template syntax

**Solutions:**
```bash
# Verify templates exist
ls -la backend/notification-service/templates/

# Check service can read templates
cd backend/notification-service
cat templates/order_staff_notification.html

# Check logs for template loading
grep "Loaded template" notification-service.log
```

---

## 📖 API Reference

See [docs/API.md](./API.md) for complete API documentation.

**Key Endpoints:**
- `GET /api/v1/notifications/history` - Get notification history
- `POST /api/v1/notifications/:id/resend` - Resend failed notification
- `POST /api/v1/notifications/test` - Send test notification
- `GET /api/v1/notifications/config` - Get notification config
- `PATCH /api/v1/notifications/config` - Update notification config
- `GET /api/v1/users/notification-preferences` - Get user preferences
- `PATCH /api/v1/users/:id/notification-preferences` - Update user preference

---

## 🎨 Frontend Components

### NotificationSettings Component

**Location:** `frontend/src/components/admin/NotificationSettings.tsx`

**Features:**
- View all staff notification preferences
- Toggle switches for enable/disable per staff
- "Send Test Email" button with confirmation modal
- Real-time feedback (success/error messages)
- i18n support for all text

**Route:** `/settings/notifications`

### NotificationHistory Component

**Location:** `frontend/src/components/admin/NotificationHistory.tsx`

**Features:**
- Paginated notification list (20 per page)
- Filter controls (order ref, status, type, dates)
- Status badges (color-coded)
- Expandable error details for failed notifications
- Resend button with loading states
- Auto-refresh after successful resend

**Route:** `/settings/notifications/history`

---

## 📚 Related Documentation

- [API.md](./API.md) - Complete API documentation
- [BACKEND_CONVENTIONS.md](./BACKEND_CONVENTIONS.md) - Backend coding standards
- [FRONTEND_CONVENTIONS.md](./FRONTEND_CONVENTIONS.md) - Frontend coding standards
- [ENVIRONMENT.md](./ENVIRONMENT.md) - Environment variable reference
- [QUICK_START.md](./QUICK_START.md) - Development setup guide

---

## 📝 Changelog

### 2024-01-15 - Initial Release

**Added:**
- Staff order notifications via email
- Customer receipt emails
- Notification preferences dashboard
- Notification history with audit log
- SMTP error handling with retry logic
- Duplicate notification prevention
- Monitoring metrics
- Comprehensive documentation

**Technical Highlights:**
- TDD approach (all tests written before implementation)
- Comprehensive error classification (5 error types)
- Exponential backoff retry (configurable)
- Structured logging for metrics aggregation
- Full i18n support (English + Indonesian)
- Production-ready error handling

---

## 🙏 Credits

**Implementation Team:** AI-Assisted Development  
**Feature Spec:** `specs/004-order-email-notifications/`  
**Task Breakdown:** 96 tasks across 7 phases  
**Development Approach:** Test-Driven Development (TDD)

---

**For questions or issues, see the troubleshooting section above or check service logs.** 🚀
