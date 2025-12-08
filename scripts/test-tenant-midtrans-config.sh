#!/bin/bash

# Tenant-Specific Midtrans Configuration Test Script
# This script verifies that the tenant Midtrans configuration feature is working correctly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_URL="http://localhost:8080"
TENANT_SERVICE_URL="http://localhost:8084"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "Tenant Midtrans Config Test Suite"
echo "======================================"
echo ""

# Test 1: Check if tenant-service is running
echo -n "Test 1: Tenant Service Health Check... "
HEALTH=$(curl -s ${TENANT_SERVICE_URL}/health || echo "FAILED")
if echo "$HEALTH" | grep -q "tenant-service"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "  Tenant service is not running or not healthy"
    exit 1
fi

# Test 2: Check if order-service is running
echo -n "Test 2: Order Service Health Check... "
ORDER_HEALTH=$(curl -s http://localhost:8087/health || echo "FAILED")
if echo "$ORDER_HEALTH" | grep -q "order-service"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "  Order service is not running or not healthy"
    exit 1
fi

# Test 3: Check if API Gateway is running
echo -n "Test 3: API Gateway Health Check... "
GATEWAY_HEALTH=$(curl -s ${BASE_URL}/health || echo "FAILED")
if echo "$GATEWAY_HEALTH" | grep -q "api-gateway"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "  API Gateway is not running or not healthy"
    exit 1
fi

# Test 4: Check if migration was applied
echo -n "Test 4: Database Migration Check... "
MIGRATION_CHECK=$(PGPASSWORD=pos_password psql -h localhost -U pos_user -d pos_db -t -c "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'tenant_configs' AND column_name = 'midtrans_server_key');" 2>/dev/null || echo "FAILED")
if echo "$MIGRATION_CHECK" | grep -q "t"; then
    echo -e "${GREEN}PASS${NC}"
else
    if command -v psql &> /dev/null; then
        echo -e "${RED}FAIL${NC}"
        echo "  Migration 000021 not applied - run migrations first"
        exit 1
    else
        echo -e "${YELLOW}SKIP${NC} (psql not installed - manual verification required)"
    fi
fi

# Test 5: Check tenant-service routes (direct)
echo -n "Test 5: Tenant Service Direct Route Check... "
# Note: This will return 401 without auth, but should not be 404
TENANT_ROUTE=$(curl -s -o /dev/null -w "%{http_code}" ${TENANT_SERVICE_URL}/api/v1/admin/tenants/test-tenant/midtrans-config)
if [ "$TENANT_ROUTE" = "401" ] || [ "$TENANT_ROUTE" = "403" ] || [ "$TENANT_ROUTE" = "200" ]; then
    echo -e "${GREEN}PASS${NC} (HTTP $TENANT_ROUTE - route exists)"
else
    echo -e "${RED}FAIL${NC} (HTTP $TENANT_ROUTE - route not found)"
    echo "  Expected 401/403/200, got $TENANT_ROUTE"
fi

# Test 6: Check API Gateway proxy route
echo -n "Test 6: API Gateway Proxy Route Check... "
# Note: This will return 401 without auth, but should not be 404
GATEWAY_ROUTE=$(curl -s -o /dev/null -w "%{http_code}" ${BASE_URL}/api/v1/admin/tenants/test-tenant/midtrans-config)
if [ "$GATEWAY_ROUTE" = "401" ] || [ "$GATEWAY_ROUTE" = "403" ] || [ "$GATEWAY_ROUTE" = "200" ]; then
    echo -e "${GREEN}PASS${NC} (HTTP $GATEWAY_ROUTE - route exists)"
else
    echo -e "${RED}FAIL${NC} (HTTP $GATEWAY_ROUTE - route not found)"
    echo "  Expected 401/403/200, got $GATEWAY_ROUTE"
fi

# Test 7: Verify tenant-service has /api/v1 prefix in routes
echo -n "Test 7: Tenant Service Route Prefix Check... "
if grep -q "/api/v1/admin/tenants" "${SCRIPT_DIR}/../backend/tenant-service/main.go"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "  tenant-service routes don't include /api/v1 prefix"
fi

# Test 8: Verify order-service has tenant-specific config function
echo -n "Test 8: Order Service Tenant Config Function... "
if grep -q "GetSnapClientForTenant" "${SCRIPT_DIR}/../backend/order-service/src/config/midtrans.go"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "  Order service missing tenant-specific config functions"
fi

# Test 9: Check frontend payment settings page exists
echo -n "Test 9: Frontend Payment Settings Page... "
if [ -f "${SCRIPT_DIR}/../frontend/app/settings/payment/page.tsx" ]; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    echo "  Frontend payment settings page not found"
fi

# Test 10: Verify documentation was updated
echo -n "Test 10: Documentation Update Check... "
if grep -q "Tenant-Specific Payment Configuration" "${SCRIPT_DIR}/../docs/ENVIRONMENT.md"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${YELLOW}WARN${NC}"
    echo "  Documentation may need updating"
fi

echo ""
echo "======================================"
echo -e "${GREEN}All tests completed!${NC}"
echo "======================================"
echo ""
echo "Next Steps:"
echo "1. Login as a tenant owner"
echo "2. Navigate to Settings → Payment Settings"
echo "3. Configure Midtrans credentials"
echo "4. Test guest ordering with QRIS payment"
echo ""
echo "Troubleshooting:"
echo "- If routes return 404, check service logs in /tmp/*.log"
echo "- If authentication fails, verify JWT token and user role"
echo "- If migration failed, run: cd backend/migrations && migrate -path . -database \"postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable\" up"
echo ""
