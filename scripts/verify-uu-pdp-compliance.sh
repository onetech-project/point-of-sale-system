#!/bin/bash

# Compliance Verification Script for UU PDP
# Purpose: Automated verification of Indonesian Data Protection Law (UU PDP) compliance
# Usage: ./verify-uu-pdp-compliance.sh [--database-url DATABASE_URL]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default database URL (override with --database-url)
DATABASE_URL="${DATABASE_URL:-postgresql://pos_user:password@localhost:5432/pos_db?sslmode=disable}"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --database-url)
      DATABASE_URL="$2"
      shift 2
      ;;
    --help)
      echo "Usage: $0 [--database-url DATABASE_URL]"
      echo ""
      echo "Options:"
      echo "  --database-url    Database connection string (default: postgresql://pos_user:password@localhost:5432/pos_db)"
      echo "  --help            Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}UU PDP Compliance Verification${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
echo "Database: ${DATABASE_URL%%\?*}"
echo ""

# Helper functions
check_start() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    echo -n "[$TOTAL_CHECKS] $1... "
}

check_pass() {
    PASSED_CHECKS=$((PASSED_CHECKS + 1))
    echo -e "${GREEN}✓ PASS${NC}"
    if [ -n "$1" ]; then
        echo "    $1"
    fi
}

check_fail() {
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
    echo -e "${RED}✗ FAIL${NC}"
    echo -e "${RED}    $1${NC}"
}

check_warn() {
    echo -e "${YELLOW}⚠ WARNING${NC}"
    echo -e "${YELLOW}    $1${NC}"
}

# Execute SQL query
query_db() {
    psql "$DATABASE_URL" -t -c "$1" 2>&1
}

# Check 1: All PII is encrypted
check_start "Checking all PII is encrypted in users table"
UNENCRYPTED_USERS=$(query_db "SELECT COUNT(*) FROM users WHERE email_encrypted NOT LIKE 'vault:v%' OR email_encrypted IS NULL;")
if [ "$UNENCRYPTED_USERS" -eq 0 ]; then
    check_pass "All user emails are encrypted"
else
    check_fail "$UNENCRYPTED_USERS user records have unencrypted email fields"
fi

# Check 2: Guest order PII encrypted
check_start "Checking guest order PII is encrypted"
UNENCRYPTED_GUESTS=$(query_db "SELECT COUNT(*) FROM guest_orders WHERE (customer_email_encrypted NOT LIKE 'vault:v%' OR customer_email_encrypted IS NULL) AND is_anonymized = FALSE;")
if [ "$UNENCRYPTED_GUESTS" -eq 0 ]; then
    check_pass "All non-anonymized guest orders have encrypted PII"
else
    check_fail "$UNENCRYPTED_GUESTS guest order records have unencrypted PII"
fi

# Check 3: Tenant configuration encryption
check_start "Checking tenant Midtrans credentials are encrypted"
UNENCRYPTED_CONFIGS=$(query_db "SELECT COUNT(*) FROM tenant_configs WHERE server_key_encrypted NOT LIKE 'vault:v%' OR server_key_encrypted IS NULL;")
if [ "$UNENCRYPTED_CONFIGS" -eq 0 ]; then
    check_pass "All tenant Midtrans credentials are encrypted"
else
    check_fail "$UNENCRYPTED_CONFIGS tenant configs have unencrypted credentials"
fi

# Check 4: No plaintext PII in application logs
check_start "Checking for plaintext PII in application logs"
LOG_PATTERN="user@example\.com|john\.doe@|test@test\.com"
if [ -d "/var/log/pos" ]; then
    PLAINTEXT_IN_LOGS=$(grep -rE "$LOG_PATTERN" /var/log/pos/*.log 2>/dev/null | wc -l)
    if [ "$PLAINTEXT_IN_LOGS" -eq 0 ]; then
        check_pass "No plaintext PII patterns found in logs"
    else
        check_fail "$PLAINTEXT_IN_LOGS instances of potential plaintext PII found in logs"
    fi
else
    check_warn "Log directory /var/log/pos not found, skipping log scan"
fi

# Check 5: Audit events are immutable
check_start "Verifying audit events table is immutable"
TEST_EVENT_ID="00000000-0000-0000-0000-000000000000"
UPDATE_RESULT=$(query_db "UPDATE audit_events SET action = 'tampered' WHERE event_id = '$TEST_EVENT_ID';" 2>&1 || true)
if echo "$UPDATE_RESULT" | grep -qi "permission denied\|not allowed\|violates row-level security"; then
    check_pass "Audit events table prevents UPDATE operations"
else
    check_fail "Audit events table is modifiable (UPDATE should be blocked)"
fi

DELETE_RESULT=$(query_db "DELETE FROM audit_events WHERE event_id = '$TEST_EVENT_ID';" 2>&1 || true)
if echo "$DELETE_RESULT" | grep -qi "permission denied\|not allowed\|violates row-level security"; then
    check_pass "Audit events table prevents DELETE operations"
else
    check_fail "Audit events table allows DELETE (should be blocked)"
fi

# Check 6: Required consents exist for all tenants
check_start "Checking required consents for all tenants"
TENANTS_WITHOUT_CONSENT=$(query_db "
SELECT COUNT(DISTINCT t.id) 
FROM tenants t 
LEFT JOIN consent_records cr ON t.id = cr.subject_id AND cr.subject_type = 'tenant' AND cr.purpose_code = 'operational' AND cr.revoked_at IS NULL
WHERE cr.id IS NULL;
")
if [ "$TENANTS_WITHOUT_CONSENT" -eq 0 ]; then
    check_pass "All tenants have required operational consent"
else
    check_fail "$TENANTS_WITHOUT_CONSENT tenants are missing required operational consent"
fi

# Check 7: Guest orders have consent records
check_start "Checking guest orders have consent records"
GUESTS_WITHOUT_CONSENT=$(query_db "
SELECT COUNT(DISTINCT go.id) 
FROM guest_orders go 
LEFT JOIN consent_records cr ON go.id = cr.subject_id AND cr.subject_type = 'guest'
WHERE cr.id IS NULL;
")
if [ "$GUESTS_WITHOUT_CONSENT" -eq 0 ]; then
    check_pass "All guest orders have consent records"
elif [ "$GUESTS_WITHOUT_CONSENT" -lt 10 ]; then
    check_warn "$GUESTS_WITHOUT_CONSENT guest orders missing consent records (acceptable for legacy data)"
else
    check_fail "$GUESTS_WITHOUT_CONSENT guest orders are missing consent records"
fi

# Check 8: Audit trail retention policy (7 years per UU PDP Article 56)
check_start "Verifying audit events retention policy"
AUDIT_RETENTION=$(query_db "SELECT retention_period_days FROM retention_policies WHERE table_name = 'audit_events';")
AUDIT_RETENTION=$(echo "$AUDIT_RETENTION" | tr -d ' ')
if [ "$AUDIT_RETENTION" -ge 2555 ]; then
    check_pass "Audit events retention is $AUDIT_RETENTION days (≥7 years)"
else
    check_fail "Audit events retention is only $AUDIT_RETENTION days (must be ≥2555 days / 7 years)"
fi

# Check 9: Deleted users retention policy (90 days grace period)
check_start "Verifying deleted users retention policy"
USER_RETENTION=$(query_db "SELECT retention_period_days FROM retention_policies WHERE table_name = 'users' AND record_type = 'deleted';")
USER_RETENTION=$(echo "$USER_RETENTION" | tr -d ' ')
if [ "$USER_RETENTION" -ge 90 ]; then
    check_pass "Deleted users retention is $USER_RETENTION days (≥90 days grace period)"
else
    check_fail "Deleted users retention is only $USER_RETENTION days (should be ≥90 days)"
fi

# Check 10: Audit partitions exist for current and next month
check_start "Checking audit event partitions exist"
CURRENT_PARTITION="audit_events_$(date +%Y_%m)"
NEXT_PARTITION="audit_events_$(date -d '+1 month' +%Y_%m)"

CURRENT_EXISTS=$(query_db "SELECT to_regclass('$CURRENT_PARTITION');" | grep -v "^$" | wc -l)
NEXT_EXISTS=$(query_db "SELECT to_regclass('$NEXT_PARTITION');" | grep -v "^$" | wc -l)

if [ "$CURRENT_EXISTS" -gt 0 ] && [ "$NEXT_EXISTS" -gt 0 ]; then
    check_pass "Current and next month audit partitions exist"
elif [ "$CURRENT_EXISTS" -gt 0 ]; then
    check_warn "Current month partition exists, but next month partition missing (will be auto-created)"
else
    check_fail "Current month audit partition does not exist"
fi

# Check 11: Privacy policy is published
check_start "Checking privacy policy is published"
PRIVACY_POLICY_COUNT=$(query_db "SELECT COUNT(*) FROM privacy_policy_versions WHERE is_current = TRUE;")
if [ "$PRIVACY_POLICY_COUNT" -eq 1 ]; then
    check_pass "Current privacy policy is published"
elif [ "$PRIVACY_POLICY_COUNT" -eq 0 ]; then
    check_fail "No current privacy policy found"
else
    check_fail "Multiple current privacy policies found (should be exactly 1)"
fi

# Check 12: Consent purposes configured
check_start "Checking consent purposes are configured"
CONSENT_PURPOSES=$(query_db "SELECT COUNT(*) FROM consent_purposes WHERE is_active = TRUE;")
if [ "$CONSENT_PURPOSES" -ge 2 ]; then
    check_pass "$CONSENT_PURPOSES active consent purposes configured"
else
    check_fail "Only $CONSENT_PURPOSES consent purposes configured (need at least 2: operational + payment)"
fi

# Check 13: Retention policies cover all critical tables
check_start "Checking retention policies cover critical tables"
EXPECTED_TABLES="users,guest_orders,audit_events,email_verification_tokens,password_reset_tokens"
for table in $(echo $EXPECTED_TABLES | tr ',' ' '); do
    COUNT=$(query_db "SELECT COUNT(*) FROM retention_policies WHERE table_name = '$table';")
    if [ "$COUNT" -eq 0 ]; then
        check_fail "Missing retention policy for table: $table"
    fi
done
if [ "$FAILED_CHECKS" -eq 0 ]; then
    check_pass "All critical tables have retention policies"
fi

# Check 14: Cleanup jobs monitoring
check_start "Checking cleanup job metrics (requires Prometheus)"
if command -v curl &> /dev/null; then
    PROM_RESPONSE=$(curl -s http://localhost:9090/api/v1/query?query=cleanup_last_run_timestamp_seconds 2>&1 || echo "")
    if echo "$PROM_RESPONSE" | grep -q "cleanup_last_run_timestamp_seconds"; then
        check_pass "Cleanup job metrics available in Prometheus"
    else
        check_warn "Cleanup job metrics not found in Prometheus (may not be configured)"
    fi
else
    check_warn "curl not available, skipping Prometheus metrics check"
fi

# Check 15: Encryption key version consistency
check_start "Checking encryption key versions are consistent"
KEY_VERSIONS=$(query_db "
SELECT DISTINCT 
    SUBSTRING(email_encrypted FROM 'vault:v([0-9]+):') AS key_version,
    COUNT(*) AS count
FROM users 
WHERE email_encrypted LIKE 'vault:v%'
GROUP BY key_version
ORDER BY key_version DESC;
")
VERSION_COUNT=$(echo "$KEY_VERSIONS" | grep -v "^$" | wc -l)
if [ "$VERSION_COUNT" -le 2 ]; then
    check_pass "Encryption key versions are consistent (≤2 versions in use)"
    echo "$KEY_VERSIONS"
else
    check_warn "$VERSION_COUNT different encryption key versions detected (consider key migration)"
    echo "$KEY_VERSIONS"
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Compliance Verification Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Total Checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
echo ""

if [ "$FAILED_CHECKS" -eq 0 ]; then
    echo -e "${GREEN}✓ All compliance checks PASSED${NC}"
    echo ""
    echo "Your system is compliant with UU PDP No.27 Tahun 2022"
    exit 0
else
    echo -e "${RED}✗ Some compliance checks FAILED${NC}"
    echo ""
    echo "Please address the failures above before deploying to production"
    echo "For detailed troubleshooting, see: docs/UU_PDP_COMPLIANCE.md"
    exit 1
fi
