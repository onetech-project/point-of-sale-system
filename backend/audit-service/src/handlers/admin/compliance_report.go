package admin

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// ComplianceReportHandler handles compliance status reporting
type ComplianceReportHandler struct {
	db *sql.DB
}

// NewComplianceReportHandler creates a new compliance report handler
func NewComplianceReportHandler(db *sql.DB) *ComplianceReportHandler {
	return &ComplianceReportHandler{
		db: db,
	}
}

// ComplianceReport represents aggregated compliance metrics
type ComplianceReport struct {
	ReportDate        time.Time                  `json:"report_date"`
	EncryptedRecords  EncryptedRecordMetrics     `json:"encrypted_records"`
	ActiveConsents    map[string]int             `json:"active_consents"`
	AuditEvents       AuditEventMetrics          `json:"audit_events"`
	RetentionCoverage map[string]string          `json:"retention_coverage"`
	ComplianceStatus  string                     `json:"compliance_status"` // COMPLIANT, WARNING, NON_COMPLIANT
	Issues            []ComplianceIssue          `json:"issues"`
}

// EncryptedRecordMetrics tracks encrypted PII counts
type EncryptedRecordMetrics struct {
	Users         int `json:"users"`
	GuestOrders   int `json:"guest_orders"`
	TenantConfigs int `json:"tenant_configs"`
}

// AuditEventMetrics tracks audit trail statistics
type AuditEventMetrics struct {
	Total           int        `json:"total"`
	Last30Days      int        `json:"last_30_days"`
	OldestEventDate *time.Time `json:"oldest_event_date"`
}

// ComplianceIssue represents a compliance violation
type ComplianceIssue struct {
	Severity    string `json:"severity"`    // CRITICAL, WARNING
	Category    string `json:"category"`    // encryption, consent, audit, retention
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// GetComplianceReport handles GET /admin/compliance/report
func (h *ComplianceReportHandler) GetComplianceReport(c echo.Context) error {
	// Generate compliance report
	report, err := h.generateComplianceReport(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate compliance report",
		})
	}

	// Return report as JSON
	return c.JSON(http.StatusOK, report)
}

// generateComplianceReport aggregates compliance metrics from database
func (h *ComplianceReportHandler) generateComplianceReport(c echo.Context) (*ComplianceReport, error) {
	report := &ComplianceReport{
		ReportDate:        time.Now(),
		ActiveConsents:    make(map[string]int),
		RetentionCoverage: make(map[string]string),
		Issues:            []ComplianceIssue{},
	}

	// Check encrypted records
	if err := h.checkEncryptedRecords(c, report); err != nil {
		return nil, err
	}

	// Check active consents
	if err := h.checkActiveConsents(c, report); err != nil {
		return nil, err
	}

	// Check audit events
	if err := h.checkAuditEvents(c, report); err != nil {
		return nil, err
	}

	// Check retention coverage
	if err := h.checkRetentionCoverage(c, report); err != nil {
		return nil, err
	}

	// Determine overall compliance status
	report.ComplianceStatus = h.determineComplianceStatus(report)

	return report, nil
}

// checkEncryptedRecords verifies all PII is encrypted
func (h *ComplianceReportHandler) checkEncryptedRecords(c echo.Context, report *ComplianceReport) error {
	// Count encrypted user records
	query := `SELECT COUNT(*) FROM users WHERE email_encrypted LIKE 'vault:v%'`
	err := h.db.QueryRowContext(c.Request().Context(), query).Scan(&report.EncryptedRecords.Users)
	if err != nil {
		return err
	}

	// Check for unencrypted user records
	var unencryptedUsers int
	query = `SELECT COUNT(*) FROM users WHERE email_encrypted NOT LIKE 'vault:v%' OR email_encrypted IS NULL`
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&unencryptedUsers)
	if err != nil {
		return err
	}
	if unencryptedUsers > 0 {
		report.Issues = append(report.Issues, ComplianceIssue{
			Severity:    "CRITICAL",
			Category:    "encryption",
			Description: "Unencrypted PII detected in users table",
			Remediation: "Run encryption migration: ./scripts/encrypt-existing-data.sh",
		})
	}

	// Count encrypted guest orders
	query = `SELECT COUNT(*) FROM guest_orders WHERE customer_email_encrypted LIKE 'vault:v%' AND is_anonymized = FALSE`
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&report.EncryptedRecords.GuestOrders)
	if err != nil {
		return err
	}

	// Check for unencrypted guest orders
	var unencryptedGuests int
	query = `SELECT COUNT(*) FROM guest_orders WHERE (customer_email_encrypted NOT LIKE 'vault:v%' OR customer_email_encrypted IS NULL) AND is_anonymized = FALSE`
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&unencryptedGuests)
	if err != nil {
		return err
	}
	if unencryptedGuests > 0 {
		report.Issues = append(report.Issues, ComplianceIssue{
			Severity:    "CRITICAL",
			Category:    "encryption",
			Description: "Unencrypted guest order PII detected",
			Remediation: "Run encryption migration for guest_orders table",
		})
	}

	// Count encrypted tenant configs
	query = `SELECT COUNT(*) FROM tenant_configs WHERE server_key_encrypted LIKE 'vault:v%'`
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&report.EncryptedRecords.TenantConfigs)
	if err != nil {
		return err
	}

	return nil
}

// checkActiveConsents verifies consent coverage
func (h *ComplianceReportHandler) checkActiveConsents(c echo.Context, report *ComplianceReport) error {
	// Count active consents by purpose
	query := `
		SELECT cp.purpose_code, COUNT(DISTINCT cr.subject_id) AS consent_count
		FROM consent_purposes cp
		LEFT JOIN consent_records cr ON cp.purpose_code = cr.purpose_code AND cr.revoked_at IS NULL
		WHERE cp.is_active = TRUE
		GROUP BY cp.purpose_code
	`
	rows, err := h.db.QueryContext(c.Request().Context(), query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var purposeCode string
		var count int
		if err := rows.Scan(&purposeCode, &count); err != nil {
			return err
		}
		report.ActiveConsents[purposeCode] = count
	}

	// Check for tenants without required consents
	query = `
		SELECT COUNT(DISTINCT t.id) 
		FROM tenants t 
		LEFT JOIN consent_records cr ON t.id = cr.subject_id 
			AND cr.subject_type = 'tenant' 
			AND cr.purpose_code = 'operational' 
			AND cr.revoked_at IS NULL
		WHERE cr.id IS NULL
	`
	var tenantsWithoutConsent int
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&tenantsWithoutConsent)
	if err != nil {
		return err
	}
	if tenantsWithoutConsent > 0 {
		report.Issues = append(report.Issues, ComplianceIssue{
			Severity:    "CRITICAL",
			Category:    "consent",
			Description: "Tenants without required operational consent",
			Remediation: "Contact affected tenants to re-grant consent",
		})
	}

	return nil
}

// checkAuditEvents verifies audit trail integrity
func (h *ComplianceReportHandler) checkAuditEvents(c echo.Context, report *ComplianceReport) error {
	// Count total audit events
	query := `SELECT COUNT(*) FROM audit_events`
	err := h.db.QueryRowContext(c.Request().Context(), query).Scan(&report.AuditEvents.Total)
	if err != nil {
		return err
	}

	// Count audit events in last 30 days
	query = `SELECT COUNT(*) FROM audit_events WHERE timestamp > NOW() - INTERVAL '30 days'`
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&report.AuditEvents.Last30Days)
	if err != nil {
		return err
	}

	// Get oldest audit event date
	query = `SELECT MIN(timestamp) FROM audit_events`
	err = h.db.QueryRowContext(c.Request().Context(), query).Scan(&report.AuditEvents.OldestEventDate)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Check for audit event gaps (missing expected events)
	if report.AuditEvents.Last30Days == 0 && report.AuditEvents.Total > 0 {
		report.Issues = append(report.Issues, ComplianceIssue{
			Severity:    "WARNING",
			Category:    "audit",
			Description: "No audit events recorded in last 30 days",
			Remediation: "Check audit service and Kafka consumer status",
		})
	}

	return nil
}

// checkRetentionCoverage verifies retention policies exist
func (h *ComplianceReportHandler) checkRetentionCoverage(c echo.Context, report *ComplianceReport) error {
	// Get retention policies
	query := `
		SELECT table_name, retention_period_days, legal_minimum_days
		FROM retention_policies
		WHERE is_active = TRUE
	`
	rows, err := h.db.QueryContext(c.Request().Context(), query)
	if err != nil {
		return err
	}
	defer rows.Close()

	criticalTables := map[string]bool{
		"users":        false,
		"guest_orders": false,
		"audit_events": false,
	}

	for rows.Next() {
		var tableName string
		var retentionDays, legalMinDays int
		if err := rows.Scan(&tableName, &retentionDays, &legalMinDays); err != nil {
			return err
		}

		// Mark table as covered
		if _, exists := criticalTables[tableName]; exists {
			criticalTables[tableName] = true
		}

		// Check retention meets legal minimum
		if retentionDays < legalMinDays {
			report.Issues = append(report.Issues, ComplianceIssue{
				Severity:    "CRITICAL",
				Category:    "retention",
				Description: "Retention period below legal minimum for " + tableName,
				Remediation: "Update retention policy to meet legal requirements",
			})
			report.RetentionCoverage[tableName] = "NON_COMPLIANT"
		} else {
			report.RetentionCoverage[tableName] = "100%"
		}
	}

	// Check for missing critical table policies
	for table, covered := range criticalTables {
		if !covered {
			report.Issues = append(report.Issues, ComplianceIssue{
				Severity:    "CRITICAL",
				Category:    "retention",
				Description: "Missing retention policy for " + table,
				Remediation: "Create retention policy in retention_policies table",
			})
			report.RetentionCoverage[table] = "0%"
		}
	}

	return nil
}

// determineComplianceStatus calculates overall compliance based on issues
func (h *ComplianceReportHandler) determineComplianceStatus(report *ComplianceReport) string {
	if len(report.Issues) == 0 {
		return "COMPLIANT"
	}

	// Check for critical issues
	for _, issue := range report.Issues {
		if issue.Severity == "CRITICAL" {
			return "NON_COMPLIANT"
		}
	}

	// Only warnings present
	return "WARNING"
}
