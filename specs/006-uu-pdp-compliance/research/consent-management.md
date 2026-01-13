# Consent Management Research: UU PDP No.27 Tahun 2022 Compliance

**Feature**: Indonesian Data Protection Compliance (UU PDP)  
**Research Focus**: Consent collection, versioning, lifecycle, and legal compliance  
**Date**: January 2, 2026  
**Researcher**: GitHub Copilot

## Decision: Versioned Consent with Granular Purpose Tracking

**Consent Model**: Implement a consent record system with:

- **Versioning**: Each consent record links to a consent policy version (e.g., "1.0", "2.0")
- **Granularity**: Separate consent records per purpose type (operational, analytics, advertising, third-party)
- **Lifecycle Tracking**: Capture grant → active → revocation states with full audit metadata
- **Dual User Model**: Support both tenant (persistent account) and guest (order-linked) consent

### Core Schema Design

```sql
-- Consent policy versions (defines what users agree to)
CREATE TABLE consent_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version VARCHAR(20) UNIQUE NOT NULL,  -- "1.0", "1.1", "2.0"
    effective_date TIMESTAMP NOT NULL,
    policy_text_id TEXT NOT NULL,         -- i18n key: "consent.policy.v1_0"
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deprecated_at TIMESTAMP,              -- When policy is no longer offered

    CONSTRAINT valid_version CHECK (version ~ '^\d+\.\d+$')
);

-- Purpose types (what we can get consent for)
CREATE TABLE consent_purposes (
    id VARCHAR(50) PRIMARY KEY,           -- 'operational_data', 'analytics', 'advertising', 'third_party_midtrans'
    purpose_name_en TEXT NOT NULL,        -- i18n key: "consent.purpose.operational_data"
    purpose_name_id TEXT NOT NULL,        -- i18n key: "consent.purpose.operational_data"
    description_en TEXT NOT NULL,         -- i18n key: "consent.purpose.operational_data.desc"
    description_id TEXT NOT NULL,         -- i18n key: "consent.purpose.operational_data.desc"
    is_required BOOLEAN NOT NULL,         -- true for operational/order_processing
    scope VARCHAR(20) NOT NULL,           -- 'tenant' | 'guest' | 'both'
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Individual consent records
CREATE TABLE consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Subject identification (one must be set)
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    guest_order_id UUID REFERENCES guest_orders(id) ON DELETE CASCADE,

    -- Consent details
    purpose_id VARCHAR(50) NOT NULL REFERENCES consent_purposes(id),
    policy_version VARCHAR(20) NOT NULL REFERENCES consent_policies(version),

    -- State
    granted BOOLEAN NOT NULL DEFAULT false,
    granted_at TIMESTAMP,
    revoked_at TIMESTAMP,

    -- Proof metadata (for legal compliance)
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    consent_method VARCHAR(50),           -- 'registration_form', 'checkout_form', 'settings_page'

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT one_subject_only CHECK (
        (tenant_id IS NOT NULL AND user_id IS NULL AND guest_order_id IS NULL) OR
        (tenant_id IS NULL AND user_id IS NOT NULL AND guest_order_id IS NULL) OR
        (tenant_id IS NULL AND user_id IS NULL AND guest_order_id IS NOT NULL)
    ),
    CONSTRAINT valid_grant_state CHECK (
        (granted = true AND granted_at IS NOT NULL) OR
        (granted = false)
    ),
    CONSTRAINT valid_revoke_state CHECK (
        (revoked_at IS NULL) OR
        (revoked_at IS NOT NULL AND granted = true AND revoked_at >= granted_at)
    ),
    CONSTRAINT no_future_timestamps CHECK (
        granted_at <= NOW() AND (revoked_at IS NULL OR revoked_at <= NOW())
    )
);

-- Indexes for consent queries
CREATE INDEX idx_consent_records_tenant ON consent_records(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_consent_records_user ON consent_records(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_consent_records_guest_order ON consent_records(guest_order_id) WHERE guest_order_id IS NOT NULL;
CREATE INDEX idx_consent_records_purpose ON consent_records(purpose_id, granted) WHERE revoked_at IS NULL;
CREATE INDEX idx_consent_records_policy_version ON consent_records(policy_version);

-- Unique constraint: one active consent per subject+purpose combination
CREATE UNIQUE INDEX idx_consent_active_unique ON consent_records(
    COALESCE(tenant_id::text, user_id::text, guest_order_id::text),
    purpose_id
) WHERE revoked_at IS NULL;
```

### Versioning Strategy

**Problem**: When privacy policy changes materially (e.g., new third-party processor added), existing consents under v1.0 may not cover new processing. How to handle re-consent?

**Solution**: Consent invalidation with graceful re-prompt

1. **Version Increment**: Create new `consent_policies` record with incremented version

   ```sql
   INSERT INTO consent_policies (version, effective_date, policy_text_id)
   VALUES ('2.0', '2026-06-01', 'consent.policy.v2_0');
   ```

2. **Mark Old Version Deprecated**: Set `deprecated_at` on old policy version

   ```sql
   UPDATE consent_policies
   SET deprecated_at = '2026-06-01'
   WHERE version = '1.0';
   ```

3. **Identify Stale Consents**: Query users with outdated consent versions

   ```sql
   SELECT DISTINCT u.id, u.email
   FROM users u
   JOIN consent_records cr ON cr.user_id = u.id
   WHERE cr.policy_version = '1.0'
     AND cr.granted = true
     AND cr.revoked_at IS NULL
     AND u.status = 'active';
   ```

4. **Re-prompt Strategy**:

   - **Non-blocking**: Show banner on login: "Our privacy policy has been updated. Please review and accept."
   - **Blocking (if material change)**: Require re-consent before sensitive operations (e.g., processing payments)
   - **Grace Period**: Allow 30 days for re-consent before blocking account
   - **Preserve History**: Don't revoke old consents; create NEW consent records with v2.0

5. **Implementation Pattern**:

   ```go
   // Check if user needs re-consent
   func (s *ConsentService) NeedsReConsent(ctx context.Context, userID string) (bool, error) {
       latestPolicy, err := s.repo.GetLatestConsentPolicy(ctx)
       if err != nil {
           return false, err
       }

       userConsents, err := s.repo.GetActiveConsents(ctx, userID)
       if err != nil {
           return false, err
       }

       // Check if any consent is on outdated version
       for _, consent := range userConsents {
           if consent.PolicyVersion != latestPolicy.Version {
               return true, nil
           }
       }

       return false, nil
   }
   ```

**Key Decisions**:

- ✅ **Version in consent record**: Each consent record stores `policy_version` to track which policy user agreed to
- ✅ **Consent invalidation**: Old consents remain valid (not revoked) but system flags users needing re-consent
- ✅ **Re-prompt strategies**:
  - Banner notification (non-blocking) for minor updates
  - Grace period + blocking (30 days) for material changes affecting legal basis
  - Force re-consent at next login for critical changes (e.g., new required third-party processor)

---

## Rationale: Legal Compliance + Flexibility

### Legal Compliance (UU PDP No.27 Tahun 2022)

**Article 20 (Explicit Consent)**:

- Consent must be "specific, informed, and unambiguous indication" of data subject's wishes
- Our approach:
  - ✅ **Specific**: Granular purposes (operational, analytics, advertising, third-party) instead of blanket consent
  - ✅ **Informed**: Each purpose has detailed description (`consent_purposes.description_id` → i18n text)
  - ✅ **Unambiguous**: Positive opt-in (checkbox) required, no pre-checked boxes

**Article 21 (Withdrawal Right)**:

- Data subjects have right to withdraw consent at any time
- Our approach:
  - ✅ **Easy withdrawal**: Settings page with simple toggle/button to revoke optional consents
  - ✅ **Immediate effect**: `revoked_at` timestamp captures withdrawal, system stops processing immediately
  - ✅ **No penalty**: Cannot revoke required consents (operational, order processing) without account deletion, but optional consents (analytics, advertising) revocable anytime

**Article 6 (Transparency)**:

- Controller must inform data subject of purposes, legal basis, retention, rights
- Our approach:
  - ✅ **Clear language**: Purpose descriptions in plain Indonesian language (Bahasa Indonesia)
  - ✅ **Privacy policy link**: Prominent link on registration/checkout forms
  - ✅ **Consent proof**: Record exactly what user saw (`policy_version` → `policy_text_id` → localized text)

### User Experience

**Registration Flow** (Tenant):

```
[Registration Form]
┌─────────────────────────────────────────────────────────────┐
│ Business Name: [________________]                            │
│ Email:         [________________]                            │
│ Password:      [________________]                            │
│                                                               │
│ ☑ I agree to operational data processing (Required)          │
│   → Manage orders, inventory, users for your business        │
│                                                               │
│ ☑ I agree to service analysis and improvement (Required)     │
│   → Analyze usage to improve platform features               │
│                                                               │
│ ☐ I agree to advertising and promotions (Optional)           │
│   → Receive marketing communications about new features      │
│                                                               │
│ ☑ I agree to third-party payment processing (Required)       │
│   → Midtrans for secure payment processing                   │
│   → [View Midtrans Privacy Policy]                           │
│                                                               │
│ [Privacy Policy] [Terms of Service]                          │
│                                                               │
│ [Register] ← disabled until required consents checked        │
└─────────────────────────────────────────────────────────────┘
```

**Checkout Flow** (Guest Customer):

```
[Checkout Page]
┌─────────────────────────────────────────────────────────────┐
│ Name:  [________________]                                     │
│ Phone: [________________]                                     │
│ Email: [________________]                                     │
│                                                               │
│ ☑ I agree to order processing and delivery (Required)        │
│   → Process your order and deliver to your address           │
│                                                               │
│ ☑ I agree to order communications (Required)                 │
│   → Send order status updates via email                      │
│                                                               │
│ ☐ I agree to promotional communications (Optional)           │
│   → Receive special offers from [Tenant Name]                │
│                                                               │
│ ☑ I agree to payment processing via Midtrans (Required)      │
│   → Secure payment processing by our payment partner         │
│                                                               │
│ [Privacy Policy]                                              │
│                                                               │
│ [Place Order] ← disabled until required consents checked     │
└─────────────────────────────────────────────────────────────┘
```

**Settings Page** (Tenant Consent Management):

```
[Privacy Settings]
┌─────────────────────────────────────────────────────────────┐
│ Your Privacy Preferences                                      │
│                                                               │
│ ✓ Operational Data Processing                                │
│   Status: Active (Required for service)                      │
│   Granted: 2025-12-15 10:23:45 WIB                           │
│   Policy Version: 1.0                                         │
│   [Cannot be revoked while account is active]                │
│                                                               │
│ ✓ Service Analysis and Improvement                           │
│   Status: Active (Required for service)                      │
│   Granted: 2025-12-15 10:23:45 WIB                           │
│   Policy Version: 1.0                                         │
│   [Cannot be revoked while account is active]                │
│                                                               │
│ ✓ Advertising and Promotions                                 │
│   Status: Active (Optional)                                   │
│   Granted: 2025-12-15 10:23:45 WIB                           │
│   Policy Version: 1.0                                         │
│   [Revoke Consent] ← Immediate effect                        │
│                                                               │
│ ✓ Third-Party Payment Processing                             │
│   Status: Active (Required for payments)                     │
│   Granted: 2025-12-15 10:23:45 WIB                           │
│   Policy Version: 1.0                                         │
│   Partner: Midtrans                                           │
│   [Cannot be revoked while using payment features]           │
│                                                               │
│ [View Full Privacy Policy] [Download My Data]                │
└─────────────────────────────────────────────────────────────┘
```

### Audit Trail

Every consent action creates audit log entry:

```go
type ConsentAuditEntry struct {
    Timestamp    time.Time
    TenantID     *string
    UserID       *string
    OrderID      *string
    Action       string  // "consent_granted", "consent_revoked", "consent_checked"
    Purpose      string  // "operational_data", "analytics", etc.
    PolicyVersion string
    IPAddress    string
    UserAgent    string
    Result       string  // "success", "validation_error"
}
```

### Flexibility for Future Changes

**Extensibility Points**:

1. **New purposes**: Add row to `consent_purposes` without schema migration
2. **Purpose scope changes**: Modify `consent_purposes.is_required` or `scope`
3. **New policy versions**: Insert new `consent_policies` record
4. **Third-party processors**: Add new purpose (e.g., `third_party_google_analytics`)
5. **Regional variations**: Add `consent_purposes.region` column if expanding beyond Indonesia

**Code Structure**:

```
backend/shared/consent/
├── models.go           # ConsentRecord, ConsentPolicy, ConsentPurpose structs
├── repository.go       # Database operations
├── service.go          # Business logic: Grant, Revoke, Check, NeedsReConsent
├── middleware.go       # HTTP middleware to check consent before operations
└── validator.go        # Validate consent requirements for operation types

frontend/src/components/consent/
├── ConsentCheckboxGroup.tsx    # Reusable consent form component
├── ConsentSettingsPanel.tsx    # Privacy settings page component
├── ConsentBanner.tsx           # Re-consent notification banner
└── useConsent.ts               # React hook for consent state management
```

---

## Alternatives Considered

### Alternative 1: Single Boolean "Accepted Terms" Flag

**Approach**: Store single `terms_accepted` boolean on user/order record  
**Why Rejected**:

- ❌ **Not granular**: Cannot track individual purposes (operational vs analytics vs advertising)
- ❌ **No versioning**: Cannot detect when users need to re-consent to updated policies
- ❌ **Cannot revoke**: No way to revoke optional consents while keeping account active
- ❌ **Weak legal proof**: No record of WHAT user consented to, WHEN, or under which policy version
- ❌ **UU PDP non-compliant**: Article 21 requires ability to withdraw consent; single flag doesn't support partial withdrawal

### Alternative 2: JSON Blob of Consent Data

**Approach**: Store consent as JSON in single column: `{"operational": true, "analytics": false, "version": "1.0"}`  
**Why Rejected**:

- ❌ **No referential integrity**: Cannot foreign key to consent purposes or policies
- ❌ **Hard to query**: Cannot efficiently query "all users who haven't consented to analytics"
- ❌ **Schema evolution pain**: Adding new purpose requires updating JSON structure across all records
- ❌ **No audit history**: Cannot track consent changes over time (grant → revoke → re-grant)
- ❌ **Complex validation**: Must validate JSON structure in application code instead of database constraints

### Alternative 3: Consent per Purpose per User (No Versioning)

**Approach**: `consent_records(user_id, purpose, granted)` without `policy_version`  
**Why Rejected**:

- ❌ **Cannot detect stale consents**: If policy text changes, no way to know which users consented under old terms
- ❌ **Legal risk**: Regulator asks "What did user consent to?" → Cannot provide snapshot of exact policy text
- ❌ **Cannot force re-consent**: No mechanism to invalidate old consents when policy materially changes
- ✅ **Simpler schema**: Less complex than versioned approach
- **Verdict**: Simplicity not worth legal compliance risk

### Alternative 4: Event Sourcing for Consent

**Approach**: Store all consent events (granted, revoked, updated) as immutable event stream  
**Why Rejected (for MVP)**:

- ✅ **Perfect audit trail**: Complete history of all consent changes
- ✅ **Time-travel queries**: Can reconstruct consent state at any point in time
- ❌ **Over-engineering**: Too complex for initial implementation
- ❌ **Query complexity**: Need to replay events to get current consent state (can cache, but adds complexity)
- ❌ **Infrastructure overhead**: Requires event store, snapshot mechanism, replay logic
- **Verdict**: Great for mature system, but violates YAGNI principle for MVP. Current design with `created_at`/`updated_at`/`revoked_at` provides sufficient audit trail. Consider for v2 if audit requirements grow.

---

## Implementation Notes

### Database Schema

See "Core Schema Design" section above for full schema. Key tables:

- `consent_policies`: Version definitions
- `consent_purposes`: Purpose types and metadata
- `consent_records`: Individual consent grants/revokes

**Migration Strategy**:

1. **Migration 000028**: Create tables in order: `consent_policies` → `consent_purposes` → `consent_records`
2. **Seed data**: Insert initial policy v1.0 and purpose types
3. **No existing data migration**: Feature is new, no retroactive consent required (assumption: new registrations only)

**If retroactive consent needed** (for existing users):

```sql
-- Create backdated consent records for existing users
-- WARNING: Legal review required before using this approach
INSERT INTO consent_records (user_id, purpose_id, policy_version, granted, granted_at, ip_address, consent_method)
SELECT
    u.id,
    'operational_data',
    '1.0',
    true,
    u.created_at,  -- Backdated to account creation
    '0.0.0.0',     -- No IP available for historical consents
    'implicit_from_migration'
FROM users u
WHERE u.status != 'deleted'
  AND NOT EXISTS (
      SELECT 1 FROM consent_records cr
      WHERE cr.user_id = u.id AND cr.purpose_id = 'operational_data'
  );
```

⚠️ **Legal Risk**: Backdating consents may not meet "explicit consent" requirement. Prefer re-consent flow for existing users.

### UI Patterns

**Checkbox Requirements** (per GDPR/UU PDP best practices):

- ✅ **NOT pre-checked**: User must actively check box (no default opt-in)
- ✅ **Clear label**: Purpose name visible without clicking
- ✅ **Detailed description**: Expandable section or tooltip with full explanation
- ✅ **Required indicator**: Visual cue (asterisk, "Required" badge) for mandatory consents
- ✅ **Separate from T&C**: Consent checkboxes separate from generic "I agree to Terms" checkbox
- ✅ **Links to policies**: Privacy Policy and third-party policies linked inline
- ❌ **No "Accept All"** button: Each consent must be individually checked (for required items) or explicitly optional

**Example React Component**:

```tsx
interface ConsentCheckboxProps {
  purpose: ConsentPurpose
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
}

function ConsentCheckbox({ purpose, checked, onChange, disabled }: ConsentCheckboxProps) {
  const { t } = useTranslation()

  return (
    <div className="consent-checkbox">
      <label className={purpose.isRequired ? 'required' : 'optional'}>
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          disabled={disabled || purpose.isRequired}
          required={purpose.isRequired}
        />
        <span className="consent-label">
          {t(purpose.displayNameId)}
          {purpose.isRequired && <span className="badge-required">Required</span>}
        </span>
      </label>
      <p className="consent-description">{t(purpose.descriptionId)}</p>
      {purpose.id === 'third_party_midtrans' && (
        <a href="https://midtrans.com/privacy" target="_blank" rel="noopener">
          View Midtrans Privacy Policy →
        </a>
      )}
    </div>
  )
}
```

### Legal Considerations

**What Constitutes "Explicit" Consent in UI?**

Based on GDPR precedent (applicable to UU PDP interpretation):

✅ **Valid explicit consent**:

- Checkbox that user must actively check (NOT pre-checked)
- Clear, specific statement of what is being consented to
- Separate checkbox per purpose (granular consent)
- Button label like "I agree" or "Accept" (not just "Continue" or "Next")
- Form validation prevents submission without required consents

❌ **Invalid/weak consent**:

- Pre-checked checkbox (implied consent)
- Bundled consent: "I agree to Terms, Privacy, and receiving marketing emails" (not granular)
- Scrolling or continuing = consent (too ambiguous)
- Inactivity = consent (must be active opt-in)
- Consent buried in lengthy T&C without separate acknowledgment

**UU PDP-Specific Notes**:

- **Language**: Consent forms must be in Bahasa Indonesia (primary), English optional
- **Age verification**: UU PDP applies to children 18+ (different from GDPR 16+). If collecting data from minors, parental consent required. Out of scope for business POS system (B2B).
- **Sensitive data**: Payment info, location data are sensitive per UU PDP Article 4. Require explicit consent with extra clarity ("Your payment info will be shared with Midtrans for processing").
- **Record retention**: Consent records must be retained as long as processing continues + statute of limitations (UU PDP doesn't specify, assume 7 years per Indonesian general practice).

### Consent Proof Metadata

**What to store as legal proof?**

Minimum required metadata per consent record:

1. ✅ **Timestamp** (`granted_at`): ISO 8601 with timezone (e.g., "2025-12-15T10:23:45+07:00")
2. ✅ **Policy version** (`policy_version`): Links to exact policy text user saw
3. ✅ **IP address** (`ip_address`): Evidence of where consent originated (IPv4/IPv6)
4. ✅ **User agent** (`user_agent`): Browser/device info (proves it was user, not automated)
5. ✅ **Session ID** (`session_id`): Links to broader session context if audit needed
6. ✅ **Consent method** (`consent_method`): How consent was captured (registration_form, checkout_form, settings_page)
7. ✅ **Policy text snapshot**: Via `policy_text_id` → i18n content versioning system

**Not storing** (nice-to-have but not MVP):

- ❌ UI screenshot: Complex to implement, large storage overhead, not legally required
- ❌ Clickstream: Every mouse movement/click leading to consent (over-engineering)
- ❌ Geolocation: IP address sufficient, precise GPS not needed and raises privacy concerns
- ❌ Device fingerprint: User agent sufficient, fingerprinting may itself require consent

**Example consent proof query**:

```sql
-- Reconstruct exactly what user consented to
SELECT
    cr.granted_at,
    cr.ip_address,
    cr.user_agent,
    cr.consent_method,
    cp.purpose_id,
    cp.purpose_name_id,
    cp.description_id,
    cpol.version AS policy_version,
    cpol.policy_text_id,
    cpol.effective_date
FROM consent_records cr
JOIN consent_purposes cp ON cp.id = cr.purpose_id
JOIN consent_policies cpol ON cpol.version = cr.policy_version
WHERE cr.user_id = 'abc-123'
  AND cr.granted = true
ORDER BY cr.granted_at DESC;
```

### Integration Points

**Where to check consent?**

1. **Registration** (Tenant Signup):

   - ❌ **Fail-fast**: Cannot create user without required consents
   - ✅ **Validation**: `POST /api/auth/register` validates consent payload
   - ✅ **Atomic**: User creation + consent records in single transaction

2. **Checkout** (Guest Order):

   - ❌ **Fail-fast**: Cannot place order without required consents
   - ✅ **Validation**: `POST /api/orders` validates consent payload
   - ✅ **Link to order**: Consent records reference `guest_order_id`

3. **Data Processing Operations**:

   - **Analytics events**: Check `analytics` consent before logging telemetry
   - **Marketing emails**: Check `advertising` consent before sending promotions
   - **Payment processing**: Check `third_party_midtrans` consent before creating Midtrans transaction
   - **Implementation**: Middleware/decorator pattern

4. **API Calls**:
   - **Per-request check** (if needed): Middleware checks consent for sensitive endpoints
   - **Cached consent state**: Avoid DB query on every request; cache active consents in Redis/session
   - **Example**: `GET /api/analytics` → middleware checks if user has `analytics` consent → 403 if revoked

**Middleware Pattern** (Go):

```go
// CheckConsent middleware ensures user has active consent for operation
func CheckConsent(purpose string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            userID := c.Get("user_id").(string)

            hasConsent, err := consentService.HasActiveConsent(c.Request().Context(), userID, purpose)
            if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check consent")
            }

            if !hasConsent {
                return echo.NewHTTPError(http.StatusForbidden,
                    fmt.Sprintf("Operation requires '%s' consent. Please review your privacy settings.", purpose))
            }

            return next(c)
        }
    }
}

// Usage
e.POST("/api/analytics/events", trackEvent, CheckConsent("analytics"))
```

**Fail-Fast Strategy**:

- ✅ **Registration/Checkout**: Validate consent BEFORE creating user/order (fail fast at form submission)
- ✅ **Payment processing**: Check `third_party_midtrans` consent before calling Midtrans API
- ✅ **Data export**: Check consents before generating export (don't generate file then reject)
- ❌ **Don't process then ask**: Never process data then retroactively check consent (too late)

### Guest vs Tenant Consent

**Key Differences**:

| Aspect               | Tenant (Account-Based)                                         | Guest (Order-Based)                                                                             |
| -------------------- | -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| **Subject ID**       | `tenant_id` + `user_id` in `consent_records`                   | `guest_order_id` in `consent_records`                                                           |
| **Persistence**      | Consent persists across sessions (account lifecycle)           | Consent tied to single order (transient)                                                        |
| **Access Method**    | Login → Settings page → Consent management                     | Order reference + email/phone verification → Consent view                                       |
| **Revocation Scope** | Can revoke optional consents (analytics, advertising) anytime  | Can revoke promotional emails; cannot revoke order processing (already processed)               |
| **Re-consent Flow**  | Banner on login + grace period + forced re-consent             | Email notification if policy changes (best effort, guest may not see)                           |
| **Purposes**         | operational_data, analytics, advertising, third_party_midtrans | order_processing, order_communications, promotional_communications, payment_processing_midtrans |

**Data Model Flexibility**:

```sql
-- Same table supports both via CHECK constraint
CONSTRAINT one_subject_only CHECK (
    (tenant_id IS NOT NULL AND user_id IS NULL AND guest_order_id IS NULL) OR  -- Tenant-level consent
    (tenant_id IS NULL AND user_id IS NOT NULL AND guest_order_id IS NULL) OR  -- User-level consent
    (tenant_id IS NULL AND user_id IS NULL AND guest_order_id IS NOT NULL)    -- Guest-level consent
)
```

**Query Patterns**:

```go
// Get tenant consents
func (r *ConsentRepository) GetTenantConsents(ctx context.Context, tenantID string) ([]ConsentRecord, error) {
    var consents []ConsentRecord
    err := r.db.SelectContext(ctx, &consents,
        `SELECT * FROM consent_records
         WHERE tenant_id = $1 AND revoked_at IS NULL`,
        tenantID)
    return consents, err
}

// Get guest consents
func (r *ConsentRepository) GetGuestOrderConsents(ctx context.Context, orderID string) ([]ConsentRecord, error) {
    var consents []ConsentRecord
    err := r.db.SelectContext(ctx, &consents,
        `SELECT * FROM consent_records
         WHERE guest_order_id = $1 AND revoked_at IS NULL`,
        orderID)
    return consents, err
}
```

**Guest Consent Challenges**:

- **No account**: Guest has no persistent identity → consent tied to order ID
- **Email as identifier**: Guest can access consent via order reference + email verification
- **Limited re-consent**: If policy changes post-order, cannot force re-consent (order already processed)
- **Data deletion**: Guest can request deletion of PII, but order record retained for merchant (see spec FR-032)

**Tenant Consent Advantages**:

- **Persistent identity**: Consent follows user across sessions
- **Centralized management**: Settings page for all privacy preferences
- **Re-consent enforcement**: Can block account until re-consent if policy materially changes
- **Granular control**: Separate consents for operational, analytics, advertising, third-party

### Testing Approach

**Unit Tests** (Go):

```go
func TestConsentService_GrantConsent(t *testing.T) {
    tests := []struct{
        name          string
        userID        string
        purpose       string
        policyVersion string
        wantErr       bool
    }{
        {"valid grant", "user-123", "analytics", "1.0", false},
        {"invalid purpose", "user-123", "invalid", "1.0", true},
        {"invalid version", "user-123", "analytics", "99.0", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.GrantConsent(ctx, tt.userID, tt.purpose, tt.policyVersion, metadata)
            if (err != nil) != tt.wantErr {
                t.Errorf("GrantConsent() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestConsentService_RevokeConsent(t *testing.T) {
    // Setup: Grant consent first
    service.GrantConsent(ctx, "user-123", "advertising", "1.0", metadata)

    // Test revocation
    err := service.RevokeConsent(ctx, "user-123", "advertising")
    assert.NoError(t, err)

    // Verify revoked
    hasConsent, err := service.HasActiveConsent(ctx, "user-123", "advertising")
    assert.NoError(t, err)
    assert.False(t, hasConsent)
}

func TestConsentService_CannotRevokeRequired(t *testing.T) {
    // Setup: Grant required consent
    service.GrantConsent(ctx, "user-123", "operational_data", "1.0", metadata)

    // Attempt to revoke required consent
    err := service.RevokeConsent(ctx, "user-123", "operational_data")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "cannot revoke required consent")
}
```

**Integration Tests** (BDD-style):

```go
func TestConsentFlow_Registration(t *testing.T) {
    // Given: New tenant registration request with consents
    req := RegisterRequest{
        Email:    "test@example.com",
        Password: "SecurePass123",
        Consents: []ConsentGrant{
            {Purpose: "operational_data", Granted: true},
            {Purpose: "analytics", Granted: true},
            {Purpose: "advertising", Granted: false},
            {Purpose: "third_party_midtrans", Granted: true},
        },
    }

    // When: Submitting registration
    resp, err := client.Post("/api/auth/register", req)

    // Then: Registration succeeds
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    // And: Consent records are created
    consents := getConsentsForUser(resp.UserID)
    assert.Len(t, consents, 4)
    assert.True(t, consents["operational_data"].Granted)
    assert.False(t, consents["advertising"].Granted)

    // And: Audit log entry exists
    auditLogs := getAuditLogs("consent_granted", resp.UserID)
    assert.NotEmpty(t, auditLogs)
}
```

**E2E Tests** (Frontend + Backend):

```typescript
describe('Consent Management', () => {
  it('should prevent registration without required consents', async () => {
    await page.goto('/register')
    await page.fill('[name="email"]', 'test@example.com')
    await page.fill('[name="password"]', 'SecurePass123')

    // Don't check required consents
    await page.click('button[type="submit"]')

    // Should show validation errors
    const error = await page.textContent('.consent-error')
    expect(error).toContain('Required consent: operational data')
  })

  it('should allow revoking optional consent', async () => {
    await login('tenant@example.com', 'password')
    await page.goto('/account/privacy-settings')

    // Revoke advertising consent
    await page.click('[data-purpose="advertising"] .revoke-button')
    await page.click('.confirm-revoke')

    // Should update UI
    const status = await page.textContent('[data-purpose="advertising"] .status')
    expect(status).toContain('Revoked')
  })
})
```

**Data Validation Tests**:

```sql
-- Test constraint: cannot revoke before granting
INSERT INTO consent_records (user_id, purpose_id, policy_version, granted, granted_at, revoked_at)
VALUES ('user-123', 'analytics', '1.0', true, NOW(), NOW() - INTERVAL '1 day');
-- Should fail: revoked_at < granted_at

-- Test constraint: no future timestamps
INSERT INTO consent_records (user_id, purpose_id, policy_version, granted, granted_at)
VALUES ('user-123', 'analytics', '1.0', true, NOW() + INTERVAL '1 day');
-- Should fail: granted_at > NOW()

-- Test unique constraint: one active consent per subject+purpose
INSERT INTO consent_records (user_id, purpose_id, policy_version, granted, granted_at)
VALUES ('user-123', 'analytics', '1.0', true, NOW());
INSERT INTO consent_records (user_id, purpose_id, policy_version, granted, granted_at)
VALUES ('user-123', 'analytics', '2.0', true, NOW());
-- Should fail: duplicate active consent for user-123 + analytics
```

---

## Summary

**Decision**: Implement versioned consent system with granular purposes, separate records for grant/revoke lifecycle, and dual support for tenant (persistent) and guest (order-linked) consent.

**Key Characteristics**:

- ✅ **Versioned policies**: Track which policy version user consented to
- ✅ **Granular purposes**: Separate consent per data processing purpose
- ✅ **Auditable**: Full proof metadata (timestamp, IP, user agent, policy version)
- ✅ **Flexible revocation**: Support withdrawing optional consents without deleting account
- ✅ **UU PDP compliant**: Meets Article 20 (explicit consent), Article 21 (withdrawal), Article 6 (transparency)
- ✅ **User-friendly**: Clear UI with required/optional indicators, easy revocation process
- ✅ **Future-proof**: Can add new purposes, update policies, support re-consent flows

**Next Steps** (Phase 1):

1. Create detailed database schema in `data-model.md`
2. Define API contracts for consent endpoints in `contracts/`
3. Design service interfaces (`ConsentService`, `ConsentRepository`)
4. Plan frontend components (`ConsentCheckboxGroup`, `ConsentSettingsPanel`)
5. Write test scenarios based on spec acceptance criteria

**Risks & Mitigations**:

- **Risk**: Retroactive consent for existing users may not meet "explicit" standard
  - **Mitigation**: Force re-consent flow on next login, don't backdate consent records
- **Risk**: Policy changes require re-consent, but guests have no persistent identity
  - **Mitigation**: Email notification to guest (best effort), document limitation in privacy policy
- **Risk**: Complex consent state management (versioning, revocation, re-prompts)
  - **Mitigation**: Comprehensive testing, clear service layer abstractions, good documentation
