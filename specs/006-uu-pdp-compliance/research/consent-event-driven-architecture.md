# Consent Recording Event-Driven Architecture

**Decision Date**: January 14, 2026  
**Context**: T079 redesign for reliability and atomicity  
**Status**: Proposed

## Problem Statement

### Current Implementation (T079 - At Risk)

**Flow**:

```
1. Frontend submits registration/checkout form
2. Backend creates user/order → returns user_id/order_id
3. Frontend receives response with IDs
4. Frontend calls POST /api/v1/consent/grant separately
5. Backend records consent in consent_records table
```

**Critical Risks**:

1. **Split Transaction Risk**: User/order exists but consent not recorded if step 4 fails
2. **Frontend Retry Complexity**: Frontend must implement retry logic for consent API
3. **Race Conditions**: User might start using system before consent is recorded
4. **Wrong Subject IDs**: Frontend uses temporary IDs before backend finalization
5. **Network Failures**: Any network issue between steps 2-4 causes inconsistency
6. **Browser Crashes**: User closes browser after registration but before consent call
7. **API Timeout**: Slow consent API causes poor UX, but registration already succeeded

### Real-World Failure Scenarios

**Scenario 1: Registration Success, Consent Failure**

```
Time    Action                              State
T0      User submits registration form
T1      Backend creates tenant & user       ✅ tenant_id=abc, user_id=xyz
T2      Frontend receives response
T3      Frontend calls POST /consent/grant
T4      Network timeout (30s)               ❌ No consent record
T5      User redirected to dashboard        🚨 User active WITHOUT consent!
```

**Scenario 2: Checkout Success, Consent Lost**

```
Time    Action                              State
T0      Guest submits checkout form
T1      Backend creates guest_order         ✅ order_id=123, status=pending_payment
T2      Frontend receives payment URL
T3      User clicks payment URL (new tab)
T4      Original tab closed                 ❌ Consent API never called
T5      User pays via Midtrans              🚨 Order active WITHOUT consent!
```

**Scenario 3: Browser Crash**

```
Time    Action                              State
T0      User submits registration
T1      Backend creates user                ✅ user_id=xyz
T2      Frontend starts consent API call
T3      Browser crashes mid-request         ❌ No consent record
T4      User opens browser, logs in         🚨 User active WITHOUT consent!
```

## Legal Implications (UU PDP Compliance)

### Article 20 (Explicit Consent Requirement)

**Violation**: Processing personal data WITHOUT valid consent record

- **UU PDP Article 20**: "Personal data may be processed **if and only if** the data subject has given **explicit consent**"
- **Penalty**: IDR 6 billion fine + potential criminal liability
- **Audit Risk**: Regulator asks "Show me consent record for user X" → No record exists
- **Burden of Proof**: **Controller (you) must prove consent was given**, not data subject

### Data Subject Rights (Articles 3-6)

**Right to Know** (Article 3):

- User asks: "What did I consent to?" → System has no record
- Cannot prove when/how/where consent was collected

**Right to Access** (Article 4):

- User requests consent history → Empty/incomplete results
- Cannot demonstrate compliance with transparency obligation

**Right to Deletion** (Article 5):

- User requests data deletion → But we have no proof they ever consented to storage
- Legally ambiguous: Did we collect data lawfully?

## Proposed Solution: Event-Driven Consent Recording

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend                                    │
│  1. User submits form WITH consent choices                          │
│  2. Single API call: POST /register or POST /checkout               │
└─────────────────────────────────────┬───────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Backend Service (Tenant/Order)                    │
│  3. START TRANSACTION                                               │
│  4. Create user/tenant OR guest_order (get subject_id)              │
│  5. COMMIT TRANSACTION                                              │
│  6. Publish ConsentGrantedEvent to Kafka                            │
│  7. Return response to frontend                                     │
└─────────────────────────────────────┬───────────────────────────────┘
                                      │
                                      ▼
                              ┌────────────┐
                              │   Kafka    │
                              │   Topic:   │
                              │  consents  │
                              └──────┬─────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Audit Service Consumer                         │
│  8. Consume ConsentGrantedEvent (with retry + DLQ)                  │
│  9. Decrypt IP address via Vault                                    │
│ 10. Insert into consent_records (idempotent via event_id)           │
│ 11. Acknowledge message to Kafka                                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

#### 1. Consent Data in Registration/Checkout Request

**Registration Request** (POST /api/auth/register):

```json
{
  "businessName": "Test Restaurant",
  "email": "owner@test.com",
  "password": "SecurePass123",
  "firstName": "John",
  "lastName": "Doe",
  "consents": [
    {
      "purpose_code": "operational",
      "granted": true
    },
    {
      "purpose_code": "analytics",
      "granted": false
    },
    {
      "purpose_code": "advertising",
      "granted": false
    },
    {
      "purpose_code": "third_party_midtrans",
      "granted": true
    }
  ]
}
```

**Checkout Request** (POST /api/v1/orders/checkout):

```json
{
  "tenant_id": "tenant-uuid",
  "items": [...],
  "delivery_type": "delivery",
  "customer_name": "Jane Customer",
  "customer_phone": "081234567890",
  "customer_email": "jane@example.com",
  "delivery_address": "Jl. Sudirman No. 123",
  "consents": [
    {
      "purpose_code": "order_processing",
      "granted": true
    },
    {
      "purpose_code": "order_communications",
      "granted": true
    },
    {
      "purpose_code": "promotional_communications",
      "granted": false
    },
    {
      "purpose_code": "payment_processing_midtrans",
      "granted": true
    }
  ]
}
```

#### 2. ConsentGrantedEvent Schema

**Event Structure**:

```go
type ConsentGrantedEvent struct {
    EventID        string    `json:"event_id"`         // Idempotency key (UUID)
    EventType      string    `json:"event_type"`       // "consent.granted"
    TenantID       string    `json:"tenant_id"`        // Tenant UUID
    SubjectType    string    `json:"subject_type"`     // "tenant" or "guest"
    SubjectID      string    `json:"subject_id"`       // user_id OR order_id (proper UUID)
    ConsentMethod  string    `json:"consent_method"`   // "registration" or "checkout"
    PolicyVersion  string    `json:"policy_version"`   // "1.0.0"
    Consents       []Consent `json:"consents"`         // Array of consent decisions
    Metadata       Metadata  `json:"metadata"`         // IP, user agent, timestamp
    Timestamp      time.Time `json:"timestamp"`        // Event creation time
}

type Consent struct {
    PurposeCode string `json:"purpose_code"` // "operational", "analytics", etc.
    Granted     bool   `json:"granted"`      // true/false
}

type Metadata struct {
    IPAddress  string `json:"ip_address"`   // From request
    UserAgent  string `json:"user_agent"`   // From request headers
    SessionID  string `json:"session_id"`   // Optional
    RequestID  string `json:"request_id"`   // Distributed tracing
}
```

**Example Event**:

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "consent.granted",
  "tenant_id": "tenant-abc-123",
  "subject_type": "tenant",
  "subject_id": "user-xyz-789",
  "consent_method": "registration",
  "policy_version": "1.0.0",
  "consents": [
    { "purpose_code": "operational", "granted": true },
    { "purpose_code": "analytics", "granted": false },
    { "purpose_code": "advertising", "granted": false },
    { "purpose_code": "third_party_midtrans", "granted": true }
  ],
  "metadata": {
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ...",
    "session_id": "session-uuid",
    "request_id": "req-12345"
  },
  "timestamp": "2026-01-14T10:30:00Z"
}
```

#### 3. Idempotency and Retry Safety

**Event ID Generation**:

```go
// In registration handler AFTER user creation
eventID := uuid.New().String()

// Publish event with eventID
event := ConsentGrantedEvent{
    EventID:     eventID,
    SubjectID:   user.ID.String(), // NOW we have the real user_id
    // ...
}
```

**Consumer Idempotency Check**:

```go
// In audit service consumer
func (c *ConsentConsumer) HandleConsentGranted(ctx context.Context, event ConsentGrantedEvent) error {
    // Check if already processed
    exists, err := c.repo.EventProcessed(ctx, event.EventID)
    if err != nil {
        return fmt.Errorf("failed to check event: %w", err)
    }
    if exists {
        log.Info().Str("event_id", event.EventID).Msg("Event already processed, skipping")
        return nil // ACK the message, don't reprocess
    }

    // Process consents
    for _, consent := range event.Consents {
        err := c.insertConsentRecord(ctx, event, consent)
        if err != nil {
            return fmt.Errorf("failed to insert consent: %w", err)
        }
    }

    // Mark event as processed
    return c.repo.MarkEventProcessed(ctx, event.EventID)
}
```

**Idempotency Table**:

```sql
CREATE TABLE processed_consent_events (
    event_id VARCHAR(100) PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id UUID NOT NULL,
    subject_type VARCHAR(10) NOT NULL,
    subject_id UUID NOT NULL
);

CREATE INDEX idx_processed_events_tenant ON processed_consent_events(tenant_id, processed_at DESC);
```

#### 4. Error Handling and Dead Letter Queue

**Kafka Consumer Configuration**:

```yaml
consumer:
  group_id: audit-service-consent-consumer
  topics:
    - consents
  auto_offset_reset: earliest
  enable_auto_commit: false # Manual commit after processing
  max_retry_attempts: 5
  retry_backoff_ms: 1000 # Exponential backoff: 1s, 2s, 4s, 8s, 16s
  dead_letter_topic: consents-dlq
```

**Retry Logic**:

```go
func (c *ConsentConsumer) ProcessMessage(msg *kafka.Message) error {
    var event ConsentGrantedEvent
    if err := json.Unmarshal(msg.Value, &event); err != nil {
        // Malformed event → DLQ
        c.sendToDLQ(msg, "unmarshal_error", err)
        return nil // ACK original message
    }

    // Process with retries
    err := c.HandleConsentGrantedWithRetry(context.Background(), event, 5)
    if err != nil {
        // Failed after 5 retries → DLQ
        c.sendToDLQ(msg, "processing_error", err)
        return nil // ACK original message
    }

    return nil // Success, commit offset
}

func (c *ConsentConsumer) HandleConsentGrantedWithRetry(ctx context.Context, event ConsentGrantedEvent, maxRetries int) error {
    var lastErr error
    backoff := time.Second

    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := c.HandleConsentGranted(ctx, event)
        if err == nil {
            return nil // Success
        }

        lastErr = err
        log.Warn().
            Err(err).
            Int("attempt", attempt).
            Str("event_id", event.EventID).
            Msg("Consent processing failed, retrying")

        time.Sleep(backoff)
        backoff *= 2 // Exponential backoff
    }

    return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
```

**Dead Letter Queue Monitoring**:

```go
// Prometheus metrics
consentDLQMessages := prometheus.NewCounter(prometheus.CounterOpts{
    Name: "consent_dlq_messages_total",
    Help: "Total number of consent events sent to DLQ",
})

// Alert if DLQ has messages
alert ConsentDLQNotEmpty {
    condition: consent_dlq_messages_total > 0
    for: 5m
    severity: critical
    message: "Consent events are failing and going to DLQ. Investigate immediately."
}
```

### Implementation Steps

#### Step 1: Update Request DTOs

**File**: `backend/auth-service/src/dto/register_request.go`

```go
package dto

type RegisterTenantRequest struct {
    BusinessName string    `json:"businessName" validate:"required,min=2,max=100"`
    Email        string    `json:"email" validate:"required,email"`
    Password     string    `json:"password" validate:"required,min=8"`
    FirstName    string    `json:"firstName" validate:"required,min=2,max=50"`
    LastName     string    `json:"lastName" validate:"required,min=2,max=50"`
    Consents     []Consent `json:"consents" validate:"required,min=1,dive"` // NEW
}

type Consent struct {
    PurposeCode string `json:"purpose_code" validate:"required"`
    Granted     bool   `json:"granted"`
}
```

**File**: `backend/order-service/src/dto/checkout_request.go`

```go
package dto

type CheckoutRequest struct {
    TenantID        string    `json:"tenant_id" validate:"required,uuid"`
    Items           []Item    `json:"items" validate:"required,min=1,dive"`
    DeliveryType    string    `json:"delivery_type" validate:"required,oneof=dine_in takeaway delivery"`
    CustomerName    string    `json:"customer_name" validate:"required"`
    CustomerPhone   string    `json:"customer_phone" validate:"required,e164"`
    CustomerEmail   string    `json:"customer_email" validate:"omitempty,email"`
    DeliveryAddress string    `json:"delivery_address"`
    Consents        []Consent `json:"consents" validate:"required,min=1,dive"` // NEW
}
```

#### Step 2: Validate Required Consents

**File**: `backend/auth-service/src/validators/consent_validator.go`

```go
package validators

import (
    "fmt"
    "context"
    "backend/auth-service/src/dto"
)

var requiredTenantConsents = []string{"operational", "third_party_midtrans"}

func ValidateTenantConsents(ctx context.Context, consents []dto.Consent) error {
    grantedMap := make(map[string]bool)
    for _, consent := range consents {
        if consent.Granted {
            grantedMap[consent.PurposeCode] = true
        }
    }

    // Check all required consents are granted
    var missing []string
    for _, required := range requiredTenantConsents {
        if !grantedMap[required] {
            missing = append(missing, required)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("missing required consents: %v", missing)
    }

    return nil
}
```

**File**: `backend/order-service/src/validators/consent_validator.go`

```go
package validators

import (
    "fmt"
    "context"
    "backend/order-service/src/dto"
)

var requiredGuestConsents = []string{"order_processing", "payment_processing_midtrans"}

func ValidateGuestConsents(ctx context.Context, consents []dto.Consent) error {
    grantedMap := make(map[string]bool)
    for _, consent := range consents {
        if consent.Granted {
            grantedMap[consent.PurposeCode] = true
        }
    }

    // Check all required consents are granted
    var missing []string
    for _, required := range requiredGuestConsents {
        if !grantedMap[required] {
            missing = append(missing, required)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("missing required consents: %v", missing)
    }

    return nil
}
```

#### Step 3: Update Registration Handler

**File**: `backend/auth-service/src/handlers/auth/register.go`

```go
func (h *Handler) RegisterTenant(c echo.Context) error {
    ctx := c.Request().Context()

    var req dto.RegisterTenantRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "Invalid request"})
    }

    // Validate request including consents
    if err := c.Validate(req); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    // Validate required consents
    if err := validators.ValidateTenantConsents(ctx, req.Consents); err != nil {
        return c.JSON(400, map[string]string{
            "error": "Missing required consents",
            "code": "CONSENT_REQUIRED",
            "details": err.Error(),
        })
    }

    // Create tenant and user (existing logic)
    tenant, user, err := h.authService.RegisterTenant(ctx, req)
    if err != nil {
        return c.JSON(500, map[string]string{"error": err.Error()})
    }

    // NEW: Publish ConsentGrantedEvent to Kafka
    event := events.ConsentGrantedEvent{
        EventID:       uuid.New().String(),
        EventType:     "consent.granted",
        TenantID:      tenant.ID.String(),
        SubjectType:   "tenant",
        SubjectID:     user.ID.String(), // Real user_id from database
        ConsentMethod: "registration",
        PolicyVersion: "1.0.0", // TODO: Get from database
        Consents:      req.Consents,
        Metadata: events.ConsentMetadata{
            IPAddress: c.RealIP(),
            UserAgent: c.Request().UserAgent(),
            RequestID: c.Response().Header().Get("X-Request-ID"),
        },
        Timestamp: time.Now(),
    }

    // Publish to Kafka (fire-and-forget, Kafka handles retry)
    if err := h.eventPublisher.PublishConsentGranted(ctx, event); err != nil {
        // Log error but don't fail registration
        log.Error().Err(err).Str("user_id", user.ID.String()).Msg("Failed to publish consent event")
        // TODO: Add to retry queue or alert
    }

    return c.JSON(201, map[string]interface{}{
        "tenant": tenant,
        "user": user,
        "message": "Registration successful",
    })
}
```

#### Step 4: Update Checkout Handler

**File**: `backend/order-service/src/handlers/order/checkout.go`

```go
func (h *Handler) Checkout(c echo.Context) error {
    ctx := c.Request().Context()

    var req dto.CheckoutRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, map[string]string{"error": "Invalid request"})
    }

    // Validate required consents
    if err := validators.ValidateGuestConsents(ctx, req.Consents); err != nil {
        return c.JSON(400, map[string]string{
            "error": "Missing required consents",
            "code": "CONSENT_REQUIRED",
            "details": err.Error(),
        })
    }

    // Create guest order (existing logic)
    order, err := h.orderService.CreateGuestOrder(ctx, req)
    if err != nil {
        return c.JSON(500, map[string]string{"error": err.Error()})
    }

    // NEW: Publish ConsentGrantedEvent to Kafka
    event := events.ConsentGrantedEvent{
        EventID:       uuid.New().String(),
        EventType:     "consent.granted",
        TenantID:      req.TenantID,
        SubjectType:   "guest",
        SubjectID:     order.ID.String(), // Real order_id from database
        ConsentMethod: "checkout",
        PolicyVersion: "1.0.0",
        Consents:      req.Consents,
        Metadata: events.ConsentMetadata{
            IPAddress: c.RealIP(),
            UserAgent: c.Request().UserAgent(),
            RequestID: c.Response().Header().Get("X-Request-ID"),
        },
        Timestamp: time.Now(),
    }

    // Publish to Kafka
    if err := h.eventPublisher.PublishConsentGranted(ctx, event); err != nil {
        log.Error().Err(err).Str("order_id", order.ID.String()).Msg("Failed to publish consent event")
    }

    return c.JSON(201, map[string]interface{}{
        "order": order,
        "payment_url": order.PaymentURL,
        "message": "Order created successfully",
    })
}
```

#### Step 5: Audit Service Consumer

**File**: `backend/audit-service/src/consumers/consent_consumer.go`

```go
package consumers

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"

    "github.com/pos/audit-service/src/events"
    "github.com/pos/audit-service/src/repository"
    "github.com/pos/audit-service/src/utils"
)

type ConsentConsumer struct {
    consumer     *kafka.Consumer
    repo         *repository.ConsentRepository
    encryptor    utils.Encryptor
    dlqProducer  *kafka.Producer
}

func NewConsentConsumer(
    broker string,
    groupID string,
    repo *repository.ConsentRepository,
    encryptor utils.Encryptor,
) (*ConsentConsumer, error) {
    consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
        "bootstrap.servers": broker,
        "group.id":          groupID,
        "auto.offset.reset": "earliest",
        "enable.auto.commit": false,
    })
    if err != nil {
        return nil, err
    }

    dlqProducer, err := kafka.NewProducer(&kafka.ConfigMap{
        "bootstrap.servers": broker,
    })
    if err != nil {
        return nil, err
    }

    return &ConsentConsumer{
        consumer:    consumer,
        repo:        repo,
        encryptor:   encryptor,
        dlqProducer: dlqProducer,
    }, nil
}

func (c *ConsentConsumer) Start(ctx context.Context) error {
    if err := c.consumer.Subscribe("consents", nil); err != nil {
        return err
    }

    log.Info().Msg("Consent consumer started")

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            msg, err := c.consumer.ReadMessage(1 * time.Second)
            if err != nil {
                if err.(kafka.Error).Code() == kafka.ErrTimedOut {
                    continue
                }
                log.Error().Err(err).Msg("Failed to read message")
                continue
            }

            if err := c.processMessage(ctx, msg); err != nil {
                log.Error().Err(err).Msg("Failed to process message")
                continue
            }

            // Commit offset after successful processing
            if _, err := c.consumer.CommitMessage(msg); err != nil {
                log.Error().Err(err).Msg("Failed to commit offset")
            }
        }
    }
}

func (c *ConsentConsumer) processMessage(ctx context.Context, msg *kafka.Message) error {
    var event events.ConsentGrantedEvent
    if err := json.Unmarshal(msg.Value, &event); err != nil {
        c.sendToDLQ(msg, "unmarshal_error", err)
        return nil // ACK message
    }

    // Check idempotency
    processed, err := c.repo.IsEventProcessed(ctx, event.EventID)
    if err != nil {
        return fmt.Errorf("failed to check event: %w", err)
    }
    if processed {
        log.Info().Str("event_id", event.EventID).Msg("Event already processed")
        return nil
    }

    // Encrypt IP address
    encryptedIP, err := c.encryptor.Encrypt(ctx, event.Metadata.IPAddress, "ip")
    if err != nil {
        return fmt.Errorf("failed to encrypt IP: %w", err)
    }

    // Insert consent records
    subjectUUID, err := uuid.Parse(event.SubjectID)
    if err != nil {
        c.sendToDLQ(msg, "invalid_subject_id", err)
        return nil
    }

    for _, consent := range event.Consents {
        if !consent.Granted {
            continue // Only record granted consents
        }

        record := &models.ConsentRecord{
            RecordID:      uuid.New(),
            TenantID:      event.TenantID,
            SubjectType:   event.SubjectType,
            SubjectID:     &event.SubjectID,
            PurposeCode:   consent.PurposeCode,
            Granted:       consent.Granted,
            PolicyVersion: event.PolicyVersion,
            ConsentMethod: event.ConsentMethod,
            IPAddress:     &encryptedIP,
            UserAgent:     &event.Metadata.UserAgent,
        }

        // Determine subject_id vs guest_order_id
        if event.SubjectType == "guest" {
            record.SubjectID = nil
            record.GuestOrderID = &subjectUUID
        } else {
            record.SubjectID = &subjectUUID
            record.GuestOrderID = nil
        }

        if err := c.repo.CreateConsentRecord(ctx, record); err != nil {
            return fmt.Errorf("failed to create consent record: %w", err)
        }
    }

    // Mark event as processed
    if err := c.repo.MarkEventProcessed(ctx, event.EventID); err != nil {
        return fmt.Errorf("failed to mark event processed: %w", err)
    }

    log.Info().
        Str("event_id", event.EventID).
        Str("subject_type", event.SubjectType).
        Str("subject_id", event.SubjectID).
        Int("consents_count", len(event.Consents)).
        Msg("Consent event processed successfully")

    return nil
}

func (c *ConsentConsumer) sendToDLQ(msg *kafka.Message, reason string, err error) {
    dlqMsg := &kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     stringPtr("consents-dlq"),
            Partition: kafka.PartitionAny,
        },
        Value: msg.Value,
        Headers: []kafka.Header{
            {Key: "error_reason", Value: []byte(reason)},
            {Key: "error_message", Value: []byte(err.Error())},
            {Key: "original_topic", Value: []byte(*msg.TopicPartition.Topic)},
            {Key: "original_partition", Value: []byte(fmt.Sprintf("%d", msg.TopicPartition.Partition))},
            {Key: "original_offset", Value: []byte(fmt.Sprintf("%d", msg.TopicPartition.Offset))},
        },
    }

    if err := c.dlqProducer.Produce(dlqMsg, nil); err != nil {
        log.Error().Err(err).Msg("Failed to send message to DLQ")
    }
}
```

### Benefits of Event-Driven Approach

#### 1. Atomicity from User Perspective

- User submits ONE request
- Backend either succeeds (user + event published) or fails (rollback everything)
- No partial success states

#### 2. Reliability and Durability

- Kafka guarantees message delivery (with replication)
- Consumer retries on failure (exponential backoff)
- DLQ for permanently failed messages
- No lost consent records

#### 3. Proper Subject IDs

- Backend has actual database IDs (user_id, order_id) at event creation time
- No temporary/placeholder IDs
- No frontend guessing what ID to use

#### 4. Decoupling

- Registration/checkout services don't depend on audit service being online
- Audit service can be offline, events queue up in Kafka
- Independent scaling of producers and consumers

#### 5. Audit Trail

- Event itself is auditable (Kafka retention)
- Can replay events for compliance verification
- Full history of consent processing attempts

#### 6. Performance

- Registration/checkout returns immediately after publishing event (async)
- User doesn't wait for consent recording to complete
- Consumer processes at its own pace

### Migration Path

#### Phase 1: Implement Event-Driven (New Registrations)

1. Deploy updated auth-service and order-service with consent validation
2. Deploy audit-service consumer
3. New registrations/checkouts use event-driven flow
4. Keep old POST /consent/grant API for backward compatibility

#### Phase 2: Migrate Existing Consents (If Any)

If any consent records were created using old API:

1. Query consent_records table for records without event_id
2. Generate synthetic ConsentGrantedEvent for each
3. Backfill processed_consent_events table
4. Verify data integrity

#### Phase 3: Remove Old API

1. Deprecate POST /consent/grant endpoint
2. Remove frontend calls to consent API
3. Remove consent grant handler from audit-service
4. Clean up unused code

### Testing Strategy

#### Unit Tests

```go
func TestConsentConsumer_ProcessMessage_Idempotent(t *testing.T) {
    // Given: Event already processed
    repo.EXPECT().IsEventProcessed(ctx, eventID).Return(true, nil)

    // When: Processing same event again
    err := consumer.processMessage(ctx, msg)

    // Then: No error, no duplicate insert
    assert.NoError(t, err)
    repo.AssertNotCalled(t, "CreateConsentRecord")
}

func TestConsentConsumer_ProcessMessage_EncryptsIP(t *testing.T) {
    // Given: Valid consent event with IP
    event := ConsentGrantedEvent{
        Metadata: ConsentMetadata{IPAddress: "192.168.1.1"},
    }

    // When: Processing event
    encryptor.EXPECT().Encrypt(ctx, "192.168.1.1", "ip").Return("vault:v1:encrypted", nil)

    // Then: IP is encrypted before storage
    err := consumer.processMessage(ctx, msg)
    assert.NoError(t, err)
}
```

#### Integration Tests

```go
func TestRegistration_WithConsents_EventPublished(t *testing.T) {
    // Given: Registration request with consents
    req := RegisterTenantRequest{
        // ... standard fields
        Consents: []Consent{
            {PurposeCode: "operational", Granted: true},
            {PurposeCode: "third_party_midtrans", Granted: true},
        },
    }

    // When: Submitting registration
    resp := httptest.Post(t, "/api/auth/register", req)

    // Then: Registration succeeds
    assert.Equal(t, 201, resp.StatusCode)

    // And: Event published to Kafka
    event := kafka.WaitForMessage(t, "consents", 5*time.Second)
    assert.Equal(t, "consent.granted", event.EventType)
    assert.Len(t, event.Consents, 2)
}
```

#### End-to-End Tests

```go
func TestE2E_Registration_ConsentRecorded(t *testing.T) {
    // Given: Clean database
    db.TruncateTables(t, "users", "tenants", "consent_records")

    // When: Registering new tenant
    tenant, user := registerTenant(t, "Test Business", "test@example.com")

    // Then: Wait for async consent processing
    time.Sleep(2 * time.Second)

    // And: Consent records exist
    consents := db.QueryConsents(t, user.ID)
    assert.Len(t, consents, 2) // operational + third_party_midtrans
    assert.True(t, consents[0].Granted)
}
```

### Monitoring and Alerts

#### Metrics

```go
var (
    consentEventsPublished = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "consent_events_published_total",
            Help: "Total consent events published to Kafka",
        },
        []string{"subject_type", "consent_method"},
    )

    consentEventsProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "consent_events_processed_total",
            Help: "Total consent events successfully processed",
        },
        []string{"subject_type"},
    )

    consentEventsFailed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "consent_events_failed_total",
            Help: "Total consent events failed and sent to DLQ",
        },
        []string{"subject_type", "error_reason"},
    )

    consentProcessingDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "consent_processing_duration_seconds",
            Help:    "Duration of consent event processing",
            Buckets: prometheus.DefBuckets,
        },
    )
)
```

#### Alerts

```yaml
groups:
  - name: consent_alerts
    rules:
      - alert: ConsentDLQNotEmpty
        expr: consent_events_failed_total > 0
        for: 5m
        severity: critical
        annotations:
          summary: 'Consent events failing (DLQ not empty)'
          description: '{{ $value }} consent events have failed and are in DLQ'

      - alert: ConsentConsumerLag
        expr: kafka_consumer_lag{topic="consents"} > 1000
        for: 10m
        severity: warning
        annotations:
          summary: 'Consent consumer lagging behind'
          description: 'Consumer lag is {{ $value }} messages'

      - alert: ConsentPublishFailures
        expr: rate(consent_publish_errors_total[5m]) > 0.1
        for: 5m
        severity: critical
        annotations:
          summary: 'High rate of consent publish failures'
          description: '{{ $value }} consent events/sec failing to publish'
```

## Decision

✅ **Adopt Event-Driven Consent Recording**

### Rationale

1. **Legal Compliance**: Eliminates risk of users/orders without consent records
2. **Reliability**: Kafka durability + retries ensure no data loss
3. **Atomicity**: Single transaction from user perspective
4. **Proper IDs**: Backend has real database IDs at event time
5. **Scalability**: Decoupled services, independent scaling
6. **Audit Trail**: Full event history for compliance verification

### Implementation Timeline

- **Week 1**: Implement event schema, publishers, validators
- **Week 2**: Implement audit service consumer with idempotency
- **Week 3**: Update frontend to send consents in requests
- **Week 4**: Testing (unit, integration, E2E)
- **Week 5**: Deploy to production, monitor metrics

### Next Steps

1. Update T079 task description in tasks.md
2. Create new tasks for event publisher implementation
3. Create new tasks for consumer implementation
4. Update frontend components to remove separate consent API calls
5. Add monitoring and alerting configuration
