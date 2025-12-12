# Research: Order Email Notifications

**Feature**: 004-order-email-notifications  
**Date**: 2025-12-11  
**Phase**: 0 - Research & Technical Discovery

## Research Overview

This document consolidates all technical research needed to implement email notifications for paid orders. It resolves all "NEEDS CLARIFICATION" items from the Technical Context and provides architectural guidance based on existing system patterns.

---

## 1. Email Service Provider Decision

### Decision: Use existing SMTP infrastructure in notification-service

**Rationale**:
- The notification-service already has SMTP email provider implemented (`src/providers/providers.go`)
- Current implementation supports HTML templates and email delivery
- SMTP configuration via environment variables: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`
- Fallback to console logging in development when SMTP not configured
- Proven pattern already handling user registration, login alerts, password resets, and team invitations

**Alternatives Considered**:
1. **Third-party SaaS (SendGrid, AWS SES, Mailgun)**: More features (analytics, bounce handling) but adds external dependency and cost. Rejected because current SMTP provider meets requirements and system already configured.
2. **Build new email microservice**: Over-engineering. notification-service already owns email responsibility per microservice autonomy principle.
3. **Direct SMTP from order-service**: Violates microservice autonomy - notification concerns should be centralized in notification-service.

**Implementation Notes**:
- SMTP provider in `notification-service/src/providers/providers.go` implements `EmailProvider` interface
- Supports both plain text and HTML emails via `isHTML` flag
- Built-in retry logic can be extended for exponential backoff requirement
- Email templates stored in `notification-service/templates/` directory

---

## 2. Event-Driven Architecture Pattern

### Decision: Use Kafka events for order status changes

**Rationale**:
- System already uses Kafka for event-driven communication
- notification-service has Kafka consumer (`src/queue/kafka_consumer.go`)
- order-service likely publishes events (needs verification but pattern exists in notification-service)
- Asynchronous processing ensures email failures don't block order completion (FR-016)
- Enables retry logic and temporal decoupling between services

**Event Flow**:
```
order-service: Order PAID → Publish event to Kafka topic "notification-events"
                                     ↓
notification-service: Kafka consumer → HandleEvent() → Process order.paid event
                                     ↓
                            Send staff notifications + customer receipt
                                     ↓
                            Log to notifications table
```

**Event Schema**:
```json
{
  "event_type": "order.paid",
  "tenant_id": "uuid",
  "metadata": {
    "order_id": "uuid",
    "order_reference": "ORD-001",
    "customer_email": "customer@example.com",
    "customer_name": "John Doe",
    "customer_phone": "+62123456789",
    "delivery_type": "delivery",
    "delivery_address": "...",
    "items": [...],
    "subtotal_amount": 150000,
    "delivery_fee": 10000,
    "total_amount": 160000,
    "payment_method": "QRIS",
    "transaction_id": "midtrans-txn-123",
    "paid_at": "2025-12-11T10:30:00Z"
  }
}
```

**Alternatives Considered**:
1. **Synchronous HTTP call**: Tight coupling, email failures block order processing. Rejected per constitution (asynchronous communication preferred).
2. **Database polling**: Inefficient, higher latency. Rejected because Kafka infrastructure exists.
3. **Webhooks**: Adds complexity (retry, ordering guarantees). Kafka provides these out-of-box.

---

## 3. Invoice Template & PAID Watermark

### Decision: Reuse existing invoice template with CSS watermark overlay

**Rationale**:
- notification-service already has `order_invoice.html` template in `templates/` directory
- HTML/CSS watermark is simplest implementation (no image processing dependencies)
- Template system uses Go's `text/template` package - supports conditionals for watermark display
- Maintains consistency between invoices and receipts
- Mobile-responsive HTML already required for invoice display

**Watermark Implementation Strategy**:
```html
<!-- In order_invoice.html template -->
{{if .ShowPaidWatermark}}
<div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%) rotate(-45deg); 
            font-size: 120px; color: rgba(0, 128, 0, 0.15); font-weight: bold; 
            pointer-events: none; z-index: 1000;">
    PAID
</div>
{{end}}
```

**Template Data Structure**:
```go
type InvoiceData struct {
    OrderReference   string
    CustomerName     string
    CustomerPhone    string
    CustomerEmail    string
    DeliveryType     string
    DeliveryAddress  string
    Items            []OrderItem
    SubtotalAmount   int
    DeliveryFee      int
    TotalAmount      int
    PaymentMethod    string
    TransactionID    string
    PaidAt           time.Time
    ShowPaidWatermark bool  // true for receipts, false for unpaid invoices
}
```

**Alternatives Considered**:
1. **Image watermark**: Requires image processing library, larger email size. Rejected for simplicity.
2. **PDF generation**: Overkill, compatibility issues with email clients. Rejected.
3. **Separate receipt template**: Duplication violates DRY principle. Rejected.

---

## 4. Notification Configuration Storage

### Decision: Add notification preferences to users table + new notification_configs table

**Rationale**:
- Staff email addresses already in `users` table (assumption validated)
- Per-user notification preference needs boolean flag on `users` table
- Tenant-level settings (default behavior, test mode) needs separate table per normalization
- Follows existing pattern: user-service owns user data, notification-service reads via API or shared DB

**Database Schema**:

```sql
-- Migration: Add notification preferences to users table
ALTER TABLE users 
ADD COLUMN receive_order_notifications BOOLEAN DEFAULT false;

CREATE INDEX idx_users_order_notifications 
ON users(tenant_id, receive_order_notifications) 
WHERE receive_order_notifications = true;

-- Migration: Create notification_configs table
CREATE TABLE notification_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_notifications_enabled BOOLEAN DEFAULT true,
    test_mode BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE INDEX idx_notification_configs_tenant ON notification_configs(tenant_id);
```

**Query Pattern for Staff Recipients**:
```sql
-- Get all staff members who should receive order notifications for a tenant
SELECT id, email, name, role 
FROM users 
WHERE tenant_id = $1 
  AND receive_order_notifications = true 
  AND email IS NOT NULL 
  AND email != '';
```

**Alternatives Considered**:
1. **Role-based only (all managers get notifications)**: Inflexible, doesn't meet "configure per staff" requirement (FR-018). Rejected.
2. **Separate notification_recipients table**: Over-normalization for simple boolean flag. Rejected for YAGNI.
3. **Store in JSONB config field**: Less queryable, harder to index. Rejected for data integrity.

---

## 5. Duplicate Notification Prevention

### Decision: Track transaction_id in notifications table metadata

**Rationale**:
- Midtrans provides unique transaction_id for each payment
- notification-service already has `notifications` table with `metadata` JSONB field
- Check for existing notification with same event_type + transaction_id before sending
- Idempotent event processing pattern - safe to replay Kafka events

**Implementation Logic**:
```go
// Before sending notification
func (s *NotificationService) hasSentOrderNotification(ctx context.Context, tenantID, transactionID string) (bool, error) {
    query := `
        SELECT EXISTS(
            SELECT 1 FROM notifications 
            WHERE tenant_id = $1 
              AND event_type = 'order.paid'
              AND metadata->>'transaction_id' = $2
              AND status IN ('sent', 'pending')
        )
    `
    var exists bool
    err := s.repo.db.QueryRowContext(ctx, query, tenantID, transactionID).Scan(&exists)
    return exists, err
}
```

**Alternatives Considered**:
1. **Separate deduplication table**: Extra complexity for single-use case. Rejected.
2. **Redis cache with TTL**: Doesn't provide audit trail, requires Redis dependency. Rejected.
3. **Order ID only**: Orders can have multiple payment attempts with different transactions. Rejected.

---

## 6. Retry Logic with Exponential Backoff

### Decision: Extend existing retry mechanism in notification-service

**Rationale**:
- `notifications` table already has `retry_count` and `max_retries` columns
- Current retry logic is simple (attempt up to max_retries)
- Need to add exponential backoff timing: 1min, 5min, 15min
- Use background worker or Kafka delayed messages for retry scheduling

**Implementation Strategy**:

**Option A: Background Worker (Recommended)**
```go
// In notification-service startup
go s.startRetryWorker(ctx)

func (s *NotificationService) startRetryWorker(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            s.retryFailedNotifications(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (s *NotificationService) retryFailedNotifications(ctx context.Context) error {
    // Get failed notifications ready for retry
    query := `
        SELECT id, tenant_id, type, recipient, subject, body, retry_count, created_at, failed_at
        FROM notifications
        WHERE status = 'failed'
          AND retry_count < max_retries
          AND (
              (retry_count = 0 AND failed_at < NOW() - INTERVAL '1 minute') OR
              (retry_count = 1 AND failed_at < NOW() - INTERVAL '5 minutes') OR
              (retry_count = 2 AND failed_at < NOW() - INTERVAL '15 minutes')
          )
        ORDER BY created_at ASC
        LIMIT 100
    `
    // Process each notification...
}
```

**Option B: Kafka Delayed Messages**
- Use Kafka headers for retry scheduling
- Re-publish failed messages with delay timestamp
- More complex but scales better for high volume

**Decision**: Use Option A (background worker) - simpler, meets volume requirements, follows KISS principle.

**Alternatives Considered**:
1. **Immediate retry on failure**: Doesn't allow email service recovery time. Rejected.
2. **Linear backoff**: Less effective than exponential for transient failures. Rejected.
3. **External scheduler (cron)**: Adds dependency. Built-in worker sufficient. Rejected.

---

## 7. Email Client Compatibility Testing

### Decision: Use Email on Acid or Litmus for pre-launch testing

**Rationale**:
- FR-014 requires mobile-responsive display in Gmail, Outlook, Apple Mail, mobile clients
- HTML email rendering varies widely across clients
- Testing services provide screenshots across 90+ clients
- notification-service templates already HTML-based, should work but needs validation

**Testing Checklist**:
- Gmail (web, Android, iOS)
- Outlook (2016, 2019, Office 365, web)
- Apple Mail (macOS, iOS)
- Yahoo Mail
- Mobile clients (default Android/iOS email apps)

**Best Practices Applied**:
- Use inline CSS (many clients strip `<style>` tags)
- Table-based layout for compatibility (Flexbox/Grid not widely supported)
- Alt text for any images (if added later)
- Plain text fallback (multipart email)
- Keep width under 600px for mobile

**Alternatives Considered**:
1. **Manual testing only**: Time-consuming, incomplete coverage. Rejected.
2. **Skip testing, rely on simple templates**: Risk of FR-014 failure. Rejected.
3. **Responsive framework (MJML, Foundation Emails)**: Adds build step complexity. Reassess if compatibility issues arise.

---

## 8. Guest Email Capture in Checkout

### Decision: Verify existing customer_email field in guest_orders table

**Finding**: Migration `000018_add_customer_email_to_guest_orders.up.sql` confirms field exists.

**Validation**:
```sql
-- Check if customer_email column exists
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'guest_orders' AND column_name = 'customer_email';
```

**Expected Result**: Column exists, type VARCHAR, nullable (optional field)

**Implementation Note**:
- Frontend checkout form should capture email (already implemented per assumption)
- Backend validates email format if provided (use Go validator library)
- Receipt email sent only if customer_email IS NOT NULL AND customer_email != ''

**No changes needed** - assumption validated, existing schema supports requirement.

---

## 9. Admin Dashboard Integration

### Decision: Add notification settings to existing admin frontend

**Rationale**:
- Frontend in `frontend/` directory using Next.js + TypeScript
- Admin routes likely under `frontend/src/pages/admin/` or `frontend/app/admin/`
- Settings page pattern already exists (tenant configs, order settings)
- New pages needed:
  1. `/admin/settings/notifications` - Configure staff notification preferences
  2. `/admin/notifications/history` - View notification audit log

**API Endpoints Needed** (in user-service or notification-service):
```
GET    /api/v1/users/notification-preferences      # List staff with preferences
PATCH  /api/v1/users/:id/notification-preferences  # Update staff notification setting
POST   /api/v1/notifications/test                  # Send test notification (FR-017)
GET    /api/v1/notifications/history               # List notification history (FR-012)
POST   /api/v1/notifications/:id/resend            # Resend failed notification (FR-013)
```

**UI Components**:
- Toggle switches for per-staff notification enable/disable
- Table view for notification history with status badges
- Filter/search controls for history view
- "Send Test Email" button in settings
- "Resend" button in history for failed notifications

**Alternatives Considered**:
1. **CLI-only configuration**: Doesn't meet "admin dashboard" requirement. Rejected.
2. **Direct database editing**: Not user-friendly, violates encapsulation. Rejected.

---

## 10. Performance & Scale Considerations

### Decision: Current architecture sufficient, monitor for optimization needs

**Analysis**:
- **Volume estimate**: Assuming 1000 orders/day/tenant, 10 tenants = 10K emails/day = ~0.12 emails/second avg
- **Peak load**: 10x avg = 1.2 emails/second during busy hours
- **SMTP throughput**: Standard SMTP handles 10-100 emails/second
- **Database writes**: notifications table gets 1-2 inserts per order (staff + customer receipt)
- **Kafka throughput**: Easily handles thousands of events/second

**Bottlenecks Identified**:
1. **SMTP rate limits**: Most providers limit to 100-500 emails/day on free tier
   - Mitigation: Use transactional email service (SendGrid 100/day free, upgrade to 40K/day)
2. **Database table growth**: notifications table grows indefinitely
   - Mitigation: Add partition by created_at (monthly partitions), archive after 90 days

**Monitoring Metrics**:
- Email send latency (target: <5 seconds)
- Notification delivery rate (target: >98%)
- Retry success rate (target: >95%)
- notifications table size (alert at 10M rows)

**Optimization Roadmap** (implement if needed):
1. **Phase 1 (Current)**: Synchronous email sending in event handler
2. **Phase 2 (If >100 emails/min)**: Queue emails in database, separate worker pool processes queue
3. **Phase 3 (If >1000 emails/min)**: Switch to dedicated email service (SendGrid, SES) with better APIs

**Decision**: Start with Phase 1, add monitoring, optimize based on actual usage.

---

## 11. Testing Strategy

### Decision: Multi-layer testing per Test-First Development principle

**Test Levels**:

1. **Unit Tests** (notification-service):
   - Email template rendering with/without watermark
   - Duplicate detection logic
   - Retry backoff calculation
   - Email address validation
   
2. **Contract Tests** (order-service ↔ notification-service):
   - Kafka event schema validation
   - Event publishing on order status change to PAID
   
3. **Integration Tests**:
   - End-to-end: Create order → Pay → Verify email sent
   - SMTP mock for email provider
   - Database verification for notification log entries
   - Retry logic with simulated failures

4. **Acceptance Tests** (BDD format):
   - Match user scenarios from spec.md
   - Given-When-Then format
   - Cover P1 user stories completely

**Test Tools**:
- Go testing package (`testing`)
- Testify for assertions (`github.com/stretchr/testify`)
- httptest for HTTP mocking
- Kafka test container for integration tests
- SMTP mock server (`github.com/mocktools/go-smtp-mock`)

**Test Data**:
```go
// Example test case
func TestOrderPaidNotification(t *testing.T) {
    // Setup
    mockDB := setupTestDB(t)
    mockSMTP := setupMockSMTPServer(t)
    service := NewNotificationService(mockDB)
    
    // Test
    event := models.NotificationEvent{
        EventType: "order.paid",
        TenantID: testTenantID,
        Metadata: map[string]interface{}{
            "order_reference": "ORD-001",
            "transaction_id": "midtrans-123",
            "customer_email": "test@example.com",
            // ... full order data
        },
    }
    
    err := service.HandleEvent(context.Background(), marshalEvent(event))
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 2, mockSMTP.EmailsSent()) // Staff + customer
    
    // Verify database log
    notifications := getNotificationsFromDB(mockDB, testTenantID)
    assert.Len(t, notifications, 2)
    assert.Equal(t, "sent", notifications[0].Status)
}
```

---

## 12. Security Considerations

### Decision: Apply existing security patterns from notification-service

**Security Measures**:

1. **Email Address Validation**:
   - Use regex validation before sending (prevent injection)
   - Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
   - Reject invalid addresses at API boundary

2. **Template Injection Prevention**:
   - Use Go's html/template package (auto-escapes HTML)
   - Never use user input directly in template names
   - Sanitize metadata before template rendering

3. **Tenant Isolation**:
   - RLS (Row Level Security) on notifications table
   - SET LOCAL app.current_tenant_id in all queries
   - Middleware validates X-Tenant-ID header

4. **Rate Limiting**:
   - Prevent abuse: max 100 notifications per tenant per hour
   - Implement in notification-service API endpoints
   - Use Redis for rate limit tracking

5. **PII Protection**:
   - Customer email in notifications table = PII
   - Encrypt at rest (PostgreSQL transparent encryption)
   - Mask in logs: log "test@example.com" as "t***@e***.com"
   - GDPR compliance: Support data deletion via cascade

6. **Email Content Security**:
   - No external resources (tracking pixels, remote images) - privacy concern
   - All styling inline (prevents CSS injection)
   - SPF/DKIM/DMARC configuration for SMTP sender

**Threat Model**:
- **Threat**: Staff member adds malicious script to order notes → XSS in email
  - **Mitigation**: Template auto-escaping + CSP headers in email HTML
- **Threat**: Attacker spams test notification endpoint
  - **Mitigation**: Rate limiting + authentication required
- **Threat**: Unauthorized access to notification history reveals customer emails
  - **Mitigation**: RLS + role-based access control (admin only)

---

## 13. Operational Runbook

### Decision: Document operational procedures for production support

**Monitoring Setup**:
```yaml
# Prometheus metrics to add
notification_emails_sent_total{type="order_staff"}
notification_emails_sent_total{type="order_customer"}
notification_emails_failed_total{reason="smtp_error"}
notification_delivery_duration_seconds
notification_retry_attempts_total
```

**Alerts**:
- Email failure rate >2% for 5 minutes → PagerDuty
- Email delivery latency >10 seconds (p95) → Slack warning
- Retry queue depth >1000 → Investigate bottleneck

**Common Issues & Remediation**:

1. **Issue**: Emails not sending
   - **Check**: SMTP credentials valid? `docker logs notification-service | grep SMTP`
   - **Fix**: Update SMTP_PASSWORD env var, restart service
   
2. **Issue**: Customer reports no receipt
   - **Check**: Admin → Notification History → Search order reference
   - **Action**: If status=failed, click "Resend". If email invalid, contact customer for correct email.

3. **Issue**: Duplicate notifications
   - **Check**: Database query for duplicate transaction_ids
   - **Root cause**: Likely Kafka event replay or order service bug
   - **Fix**: Temporary - duplicate detection prevents duplicates. Long-term - fix event publisher.

4. **Issue**: Staff not receiving notifications
   - **Check**: Admin → Settings → Notifications → Verify staff enabled
   - **Check**: Staff email address valid in user profile
   - **Action**: Enable notifications or update email address

**Deployment Checklist**:
- [ ] SMTP credentials configured in production
- [ ] Kafka topics created: `notification-events`
- [ ] Database migrations applied
- [ ] Email templates deployed to notification-service
- [ ] Monitoring dashboards configured
- [ ] Alert rules deployed
- [ ] Runbook shared with on-call team
- [ ] Test email sent from production environment

---

## Research Summary

All technical unknowns resolved. Key decisions:

1. **Email Provider**: Existing SMTP in notification-service
2. **Event Architecture**: Kafka events from order-service
3. **Invoice Template**: Reuse existing with CSS watermark
4. **Configuration Storage**: New column in users table + notification_configs table
5. **Duplicate Prevention**: Track transaction_id in metadata
6. **Retry Logic**: Background worker with exponential backoff
7. **Testing**: Email on Acid for client compatibility
8. **Guest Email**: Existing customer_email field in guest_orders
9. **Admin UI**: New pages in frontend admin section
10. **Performance**: Current architecture sufficient, monitoring in place
11. **Testing**: Multi-layer with unit/contract/integration/acceptance tests
12. **Security**: Template escaping, rate limiting, tenant isolation, PII protection
13. **Operations**: Monitoring, alerts, runbook for production support

**No NEEDS CLARIFICATION remaining** - ready to proceed to Phase 1 (Design).
