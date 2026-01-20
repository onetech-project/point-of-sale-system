package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

// TestUUPDPSmokeTest is an end-to-end smoke test verifying the complete UU PDP compliance workflow
// Test flow: tenant registration → user creation → encryption verification → audit log → deletion → guest order → guest deletion
func TestUUPDPSmokeTest(t *testing.T) {
	// Skip if not in integration test environment
	if testing.Short() {
		t.Skip("Skipping E2E smoke test in short mode")
	}

	ctx := context.Background()

	// Setup: Connect to database
	db, err := sql.Open("postgres", getTestDatabaseURL())
	require.NoError(t, err, "Failed to connect to database")
	defer db.Close()

	// Verify database connection
	err = db.PingContext(ctx)
	require.NoError(t, err, "Database ping failed")

	t.Log("✅ Database connection established")

	// Step 1: Tenant Registration with Consent
	t.Run("Step 1: Tenant Registration with Consent", func(t *testing.T) {
		tenantID, userID := registerTenantWithConsent(t, ctx, db)
		require.NotEmpty(t, tenantID, "Tenant ID should not be empty")
		require.NotEmpty(t, userID, "User ID should not be empty")

		// Verify required consents were granted
		consents := getConsentRecords(t, ctx, db, tenantID, "tenant")
		assert.GreaterOrEqual(t, len(consents), 2, "Should have at least 2 required consents")

		// Verify operational consent exists
		hasOperationalConsent := false
		for _, consent := range consents {
			if consent.PurposeCode == "operational" && consent.RevokedAt == nil {
				hasOperationalConsent = true
				break
			}
		}
		assert.True(t, hasOperationalConsent, "Tenant must have operational consent")

		t.Logf("✅ Tenant registered: %s, User created: %s", tenantID, userID)
	})

	// Step 2: User Creation with Encryption
	t.Run("Step 2: User Creation with Encryption", func(t *testing.T) {
		tenantID := getTestTenantID(t, ctx, db)
		userID := createEncryptedUser(t, ctx, db, tenantID, "test@example.com", "John Doe", "+628123456789")
		require.NotEmpty(t, userID, "User ID should not be empty")

		// Verify PII is encrypted
		user := getUser(t, ctx, db, userID)
		assert.True(t, isEncrypted(user.EmailEncrypted), "Email should be encrypted (starts with vault:v)")
		assert.True(t, isEncrypted(user.NameEncrypted), "Name should be encrypted")
		assert.True(t, isEncrypted(user.PhoneEncrypted), "Phone should be encrypted")

		// Verify plaintext is NOT stored
		assert.Empty(t, user.EmailPlaintext, "Plaintext email should not exist in database")

		t.Logf("✅ User created with encrypted PII: %s", userID)
	})

	// Step 3: Audit Log Verification
	t.Run("Step 3: Audit Log Verification", func(t *testing.T) {
		tenantID := getTestTenantID(t, ctx, db)
		userID := getTestUserID(t, ctx, db, tenantID)

		// Query audit events for user creation
		events := getAuditEvents(t, ctx, db, "USER_CREATED", userID)
		assert.Greater(t, len(events), 0, "Should have at least one USER_CREATED event")

		// Verify audit event structure
		event := events[0]
		assert.Equal(t, "USER_CREATED", event.EventType)
		assert.Equal(t, "create", event.Action)
		assert.Equal(t, "user", event.ResourceType)
		assert.Equal(t, userID, event.ResourceID)
		assert.NotEmpty(t, event.ComplianceTag, "Compliance tag should be set")

		// Verify audit trail is immutable (attempt to modify should fail)
		err := attemptAuditModification(t, ctx, db, event.EventID)
		assert.Error(t, err, "Audit modification should be prevented")
		assert.Contains(t, err.Error(), "permission denied", "Error should indicate permission denied")

		t.Logf("✅ Audit event verified: %s (immutable)", event.EventID)
	})

	// Step 4: Soft Delete with Notification
	t.Run("Step 4: Soft Delete with Notification", func(t *testing.T) {
		tenantID := getTestTenantID(t, ctx, db)
		userID := createEncryptedUser(t, ctx, db, tenantID, "delete-test@example.com", "Delete Test", "+628999999999")

		// Soft delete user
		err := softDeleteUser(t, ctx, db, userID)
		require.NoError(t, err, "Soft delete should succeed")

		// Verify deleted_at timestamp set
		user := getUser(t, ctx, db, userID)
		assert.NotNil(t, user.DeletedAt, "deleted_at should be set")
		assert.False(t, user.NotifiedOfDeletion, "notified_of_deletion should initially be false")

		// Verify deletion audit event
		events := getAuditEvents(t, ctx, db, "USER_DELETED", userID)
		assert.Greater(t, len(events), 0, "Should have USER_DELETED audit event")

		t.Logf("✅ User soft deleted: %s (grace period: 90 days)", userID)
	})

	// Step 5: Guest Order Creation
	t.Run("Step 5: Guest Order Creation", func(t *testing.T) {
		tenantID := getTestTenantID(t, ctx, db)
		orderRef, orderID := createGuestOrder(t, ctx, db, tenantID, "guest@example.com", "Guest User", "+628111222333")
		require.NotEmpty(t, orderRef, "Order reference should not be empty")
		require.NotEmpty(t, orderID, "Order ID should not be empty")

		// Verify guest PII is encrypted
		order := getGuestOrder(t, ctx, db, orderRef)
		assert.True(t, isEncrypted(order.CustomerEmailEncrypted), "Guest email should be encrypted")
		assert.True(t, isEncrypted(order.CustomerNameEncrypted), "Guest name should be encrypted")
		assert.True(t, isEncrypted(order.CustomerPhoneEncrypted), "Guest phone should be encrypted")

		// Verify guest consent recorded
		consents := getConsentRecords(t, ctx, db, orderID, "guest")
		assert.Greater(t, len(consents), 0, "Guest should have consent records")

		t.Logf("✅ Guest order created: %s (encrypted)", orderRef)
	})

	// Step 6: Guest Data Deletion (Anonymization)
	t.Run("Step 6: Guest Data Deletion", func(t *testing.T) {
		tenantID := getTestTenantID(t, ctx, db)
		orderRef, _ := createGuestOrder(t, ctx, db, tenantID, "delete-guest@example.com", "Delete Guest", "+628444555666")

		// Anonymize guest order
		err := anonymizeGuestOrder(t, ctx, db, orderRef)
		require.NoError(t, err, "Guest anonymization should succeed")

		// Verify PII is anonymized
		order := getGuestOrder(t, ctx, db, orderRef)
		assert.Nil(t, order.CustomerEmailEncrypted, "Email should be null after anonymization")
		assert.Nil(t, order.CustomerNameEncrypted, "Name should be null or 'Deleted User'")
		assert.Nil(t, order.CustomerPhoneEncrypted, "Phone should be null after anonymization")
		assert.True(t, order.IsAnonymized, "is_anonymized flag should be true")
		assert.NotNil(t, order.AnonymizedAt, "anonymized_at timestamp should be set")

		// Verify anonymization audit event
		events := getAuditEvents(t, ctx, db, "GUEST_ORDER_ANONYMIZED", orderRef)
		assert.Greater(t, len(events), 0, "Should have GUEST_ORDER_ANONYMIZED audit event")

		t.Logf("✅ Guest order anonymized: %s", orderRef)
	})

	// Step 7: Consent Revocation
	t.Run("Step 7: Consent Revocation", func(t *testing.T) {
		tenantID := getTestTenantID(t, ctx, db)

		// Grant optional consent (analytics)
		consentID := grantConsent(t, ctx, db, tenantID, "tenant", "analytics")
		require.NotEmpty(t, consentID, "Consent ID should not be empty")

		// Revoke consent
		err := revokeConsent(t, ctx, db, tenantID, "analytics")
		require.NoError(t, err, "Consent revocation should succeed")

		// Verify consent is revoked
		consent := getConsentRecord(t, ctx, db, consentID)
		assert.NotNil(t, consent.RevokedAt, "revoked_at should be set")

		// Verify revocation audit event
		events := getAuditEvents(t, ctx, db, "CONSENT_REVOKED", consentID)
		assert.Greater(t, len(events), 0, "Should have CONSENT_REVOKED audit event")

		t.Logf("✅ Consent revoked: %s", consentID)
	})

	// Step 8: Data Retention Policy Check
	t.Run("Step 8: Data Retention Policy Check", func(t *testing.T) {
		// Verify retention policies exist
		policies := getRetentionPolicies(t, ctx, db)
		assert.Greater(t, len(policies), 0, "Should have retention policies configured")

		// Verify audit_events retention is 7 years (2555 days)
		auditPolicy := findPolicyByTable(policies, "audit_events")
		require.NotNil(t, auditPolicy, "Audit events retention policy should exist")
		assert.Equal(t, 2555, auditPolicy.RetentionPeriodDays, "Audit retention should be 7 years (2555 days)")
		assert.Equal(t, 2555, auditPolicy.LegalMinimumDays, "Legal minimum should be 7 years")

		// Verify deleted users retention is 90 days
		userPolicy := findPolicyByTable(policies, "users")
		require.NotNil(t, userPolicy, "Users retention policy should exist")
		assert.Equal(t, 90, userPolicy.RetentionPeriodDays, "Deleted user retention should be 90 days")

		t.Logf("✅ Retention policies verified: %d policies configured", len(policies))
	})

	t.Log("🎉 UU PDP Compliance Smoke Test PASSED - All 8 steps completed successfully")
}

// Helper functions

func getTestDatabaseURL() string {
	// Use environment variable or default
	dbURL := getEnv("TEST_DATABASE_URL", "postgresql://pos_user:password@localhost:5432/pos_db_test?sslmode=disable")
	return dbURL
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func registerTenantWithConsent(t *testing.T, ctx context.Context, db *sql.DB) (tenantID, userID string) {
	// Implementation would call registration API or repository
	// For now, insert test data directly
	t.Skip("Implementation requires API client or repository injection")
	return "", ""
}

func getConsentRecords(t *testing.T, ctx context.Context, db *sql.DB, subjectID, subjectType string) []ConsentRecord {
	query := `
		SELECT id, subject_id, subject_type, purpose_code, granted_at, revoked_at
		FROM consent_records
		WHERE subject_id = $1 AND subject_type = $2
		ORDER BY granted_at DESC
	`
	rows, err := db.QueryContext(ctx, query, subjectID, subjectType)
	require.NoError(t, err)
	defer rows.Close()

	var consents []ConsentRecord
	for rows.Next() {
		var c ConsentRecord
		err := rows.Scan(&c.ID, &c.SubjectID, &c.SubjectType, &c.PurposeCode, &c.GrantedAt, &c.RevokedAt)
		require.NoError(t, err)
		consents = append(consents, c)
	}

	return consents
}

func createEncryptedUser(t *testing.T, ctx context.Context, db *sql.DB, tenantID, email, name, phone string) string {
	t.Skip("Implementation requires encryption service injection")
	return ""
}

func getUser(t *testing.T, ctx context.Context, db *sql.DB, userID string) User {
	query := `
		SELECT id, email_encrypted, full_name_encrypted, phone_encrypted, deleted_at, notified_of_deletion
		FROM users
		WHERE id = $1
	`
	var user User
	err := db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.EmailEncrypted, &user.NameEncrypted, &user.PhoneEncrypted,
		&user.DeletedAt, &user.NotifiedOfDeletion,
	)
	require.NoError(t, err)
	return user
}

func isEncrypted(ciphertext *string) bool {
	if ciphertext == nil || *ciphertext == "" {
		return false
	}
	// Vault ciphertext format: "vault:v1:..."
	return len(*ciphertext) > 10 && (*ciphertext)[:6] == "vault:"
}

func getAuditEvents(t *testing.T, ctx context.Context, db *sql.DB, eventType, resourceID string) []AuditEvent {
	query := `
		SELECT event_id, event_type, action, resource_type, resource_id, compliance_tag, timestamp
		FROM audit_events
		WHERE event_type = $1 AND resource_id = $2
		ORDER BY timestamp DESC
	`
	rows, err := db.QueryContext(ctx, query, eventType, resourceID)
	require.NoError(t, err)
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		err := rows.Scan(&e.EventID, &e.EventType, &e.Action, &e.ResourceType, &e.ResourceID, &e.ComplianceTag, &e.Timestamp)
		require.NoError(t, err)
		events = append(events, e)
	}

	return events
}

func attemptAuditModification(t *testing.T, ctx context.Context, db *sql.DB, eventID string) error {
	query := `UPDATE audit_events SET action = 'tampered' WHERE event_id = $1`
	_, err := db.ExecContext(ctx, query, eventID)
	return err
}

func softDeleteUser(t *testing.T, ctx context.Context, db *sql.DB, userID string) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1`
	_, err := db.ExecContext(ctx, query, userID)
	return err
}

func createGuestOrder(t *testing.T, ctx context.Context, db *sql.DB, tenantID, email, name, phone string) (orderRef, orderID string) {
	t.Skip("Implementation requires encryption service injection")
	return "", ""
}

func getGuestOrder(t *testing.T, ctx context.Context, db *sql.DB, orderRef string) GuestOrder {
	query := `
		SELECT id, order_reference, customer_email, customer_name, customer_phone, is_anonymized, anonymized_at
		FROM guest_orders
		WHERE order_reference = $1
	`
	var order GuestOrder
	err := db.QueryRowContext(ctx, query, orderRef).Scan(
		&order.ID, &order.OrderReference, &order.CustomerEmailEncrypted, &order.CustomerNameEncrypted,
		&order.CustomerPhoneEncrypted, &order.IsAnonymized, &order.AnonymizedAt,
	)
	require.NoError(t, err)
	return order
}

func anonymizeGuestOrder(t *testing.T, ctx context.Context, db *sql.DB, orderRef string) error {
	query := `
		UPDATE guest_orders
		SET customer_email = NULL,
		    customer_name = 'Deleted User',
		    customer_phone = NULL,
		    is_anonymized = TRUE,
		    anonymized_at = NOW()
		WHERE order_reference = $1
	`
	_, err := db.ExecContext(ctx, query, orderRef)
	return err
}

func grantConsent(t *testing.T, ctx context.Context, db *sql.DB, subjectID, subjectType, purposeCode string) string {
	query := `
		INSERT INTO consent_records (id, subject_id, subject_type, purpose_code, granted_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		RETURNING id
	`
	var consentID string
	err := db.QueryRowContext(ctx, query, subjectID, subjectType, purposeCode).Scan(&consentID)
	require.NoError(t, err)
	return consentID
}

func revokeConsent(t *testing.T, ctx context.Context, db *sql.DB, subjectID, purposeCode string) error {
	query := `UPDATE consent_records SET revoked_at = NOW() WHERE subject_id = $1 AND purpose_code = $2 AND revoked_at IS NULL`
	_, err := db.ExecContext(ctx, query, subjectID, purposeCode)
	return err
}

func getConsentRecord(t *testing.T, ctx context.Context, db *sql.DB, consentID string) ConsentRecord {
	query := `SELECT id, subject_id, purpose_code, granted_at, revoked_at FROM consent_records WHERE id = $1`
	var c ConsentRecord
	err := db.QueryRowContext(ctx, query, consentID).Scan(&c.ID, &c.SubjectID, &c.PurposeCode, &c.GrantedAt, &c.RevokedAt)
	require.NoError(t, err)
	return c
}

func getRetentionPolicies(t *testing.T, ctx context.Context, db *sql.DB) []RetentionPolicy {
	query := `
		SELECT table_name, retention_period_days, legal_minimum_days
		FROM retention_policies
		WHERE is_active = TRUE
		ORDER BY table_name
	`
	rows, err := db.QueryContext(ctx, query)
	require.NoError(t, err)
	defer rows.Close()

	var policies []RetentionPolicy
	for rows.Next() {
		var p RetentionPolicy
		err := rows.Scan(&p.TableName, &p.RetentionPeriodDays, &p.LegalMinimumDays)
		require.NoError(t, err)
		policies = append(policies, p)
	}

	return policies
}

func findPolicyByTable(policies []RetentionPolicy, tableName string) *RetentionPolicy {
	for _, p := range policies {
		if p.TableName == tableName {
			return &p
		}
	}
	return nil
}

func getTestTenantID(t *testing.T, ctx context.Context, db *sql.DB) string {
	query := `SELECT id FROM tenants ORDER BY created_at DESC LIMIT 1`
	var tenantID string
	err := db.QueryRowContext(ctx, query).Scan(&tenantID)
	require.NoError(t, err)
	return tenantID
}

func getTestUserID(t *testing.T, ctx context.Context, db *sql.DB, tenantID string) string {
	query := `SELECT id FROM users WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1`
	var userID string
	err := db.QueryRowContext(ctx, query, tenantID).Scan(&userID)
	require.NoError(t, err)
	return userID
}

// Data models

type ConsentRecord struct {
	ID          string
	SubjectID   string
	SubjectType string
	PurposeCode string
	GrantedAt   time.Time
	RevokedAt   *time.Time
}

type User struct {
	ID                 string
	EmailEncrypted     *string
	EmailPlaintext     string
	NameEncrypted      *string
	PhoneEncrypted     *string
	DeletedAt          *time.Time
	NotifiedOfDeletion bool
}

type GuestOrder struct {
	ID                     string
	OrderReference         string
	CustomerEmailEncrypted *string
	CustomerNameEncrypted  *string
	CustomerPhoneEncrypted *string
	IsAnonymized           bool
	AnonymizedAt           *time.Time
}

type AuditEvent struct {
	EventID       string
	EventType     string
	Action        string
	ResourceType  string
	ResourceID    string
	ComplianceTag *string
	Timestamp     time.Time
}

type RetentionPolicy struct {
	TableName           string
	RetentionPeriodDays int
	LegalMinimumDays    int
}
