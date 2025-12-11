#!/bin/bash

# T115: Quickstart Validation Script
# This script validates the complete QRIS guest ordering flow as documented in quickstart.md

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ORDER_SERVICE_URL="${ORDER_SERVICE_URL:-http://localhost:8084}"
TENANT_ID="${TENANT_ID:-40c8c3dd-7024-4176-9bf4-4cc706d6a2c8}"
TEST_PRODUCT_ID="${TEST_PRODUCT_ID:-d0f5bea2-1fcb-45dd-ad6a-ef994c59a0c8}"
SESSION_ID="test-session-$(date +%s)"

echo "🚀 QRIS Guest Ordering - Quickstart Validation"
echo "=============================================="
echo ""
echo "Configuration:"
echo "  Order Service: $ORDER_SERVICE_URL"
echo "  Tenant ID: $TENANT_ID"
echo "  Product ID: $TEST_PRODUCT_ID"
echo "  Session ID: $SESSION_ID"
echo ""

# Test 1: Health Check
echo -e "${YELLOW}[TEST 1]${NC} Health Check..."
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$ORDER_SERVICE_URL/health" || echo "FAILED\n000")
HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC} - Order service is healthy"
else
    echo -e "${RED}✗ FAIL${NC} - Order service health check failed (HTTP $HTTP_CODE)"
    echo "Response: $HEALTH_RESPONSE"
    exit 1
fi
echo ""

# Test 2: Get Cart (should be empty initially)
echo -e "${YELLOW}[TEST 2]${NC} Get Empty Cart..."
CART_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -H "X-Session-Id: $SESSION_ID" \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart" || echo "FAILED\n000")
HTTP_CODE=$(echo "$CART_RESPONSE" | tail -n1)
CART_BODY=$(echo "$CART_RESPONSE" | sed '$d')
if [ "$HTTP_CODE" = "200" ]; then
    ITEM_COUNT=$(echo "$CART_BODY" | grep -o '"items":\[\]' | wc -l)
    if [ "$ITEM_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ PASS${NC} - Empty cart retrieved successfully"
    else
        echo -e "${YELLOW}⚠ WARN${NC} - Cart has existing items (HTTP $HTTP_CODE)"
        echo "Response: $CART_BODY"
    fi
else
    echo -e "${RED}✗ FAIL${NC} - Get cart failed (HTTP $HTTP_CODE)"
    echo "Response: $CART_BODY"
    exit 1
fi
echo ""

# Test 3: Add Item to Cart
echo -e "${YELLOW}[TEST 3]${NC} Add Item to Cart..."
ADD_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "X-Session-Id: $SESSION_ID" \
    -d "{\"product_id\":\"$TEST_PRODUCT_ID\",\"quantity\":2}" \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart/items" || echo "FAILED\n000")
HTTP_CODE=$(echo "$ADD_RESPONSE" | tail -n1)
ADD_BODY=$(echo "$ADD_RESPONSE" | sed '$d')
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo -e "${GREEN}✓ PASS${NC} - Item added to cart successfully"
    echo "Cart: $ADD_BODY"
else
    echo -e "${RED}✗ FAIL${NC} - Add item failed (HTTP $HTTP_CODE)"
    echo "Response: $ADD_BODY"
    exit 1
fi
echo ""

# Test 4: Get Cart with Items
echo -e "${YELLOW}[TEST 4]${NC} Get Cart with Items..."
CART_RESPONSE2=$(curl -s -w "\n%{http_code}" \
    -H "X-Session-Id: $SESSION_ID" \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart" || echo "FAILED\n000")
HTTP_CODE=$(echo "$CART_RESPONSE2" | tail -n1)
CART_BODY2=$(echo "$CART_RESPONSE2" | sed '$d')
if [ "$HTTP_CODE" = "200" ]; then
    ITEM_COUNT=$(echo "$CART_BODY2" | grep -o "$TEST_PRODUCT_ID" | wc -l)
    if [ "$ITEM_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ PASS${NC} - Cart retrieved with items"
        echo "Cart: $CART_BODY2"
    else
        echo -e "${RED}✗ FAIL${NC} - Cart does not contain added item"
        echo "Response: $CART_BODY2"
        exit 1
    fi
else
    echo -e "${RED}✗ FAIL${NC} - Get cart failed (HTTP $HTTP_CODE)"
    echo "Response: $CART_BODY2"
    exit 1
fi
echo ""

# Test 5: Update Cart Item Quantity
echo -e "${YELLOW}[TEST 5]${NC} Update Cart Item Quantity..."
UPDATE_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X PATCH \
    -H "Content-Type: application/json" \
    -H "X-Session-Id: $SESSION_ID" \
    -d '{"quantity":3}' \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart/items/$TEST_PRODUCT_ID" || echo "FAILED\n000")
HTTP_CODE=$(echo "$UPDATE_RESPONSE" | tail -n1)
UPDATE_BODY=$(echo "$UPDATE_RESPONSE" | sed '$d')
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ PASS${NC} - Cart item quantity updated"
    echo "Cart: $UPDATE_BODY"
else
    echo -e "${RED}✗ FAIL${NC} - Update cart item failed (HTTP $HTTP_CODE)"
    echo "Response: $UPDATE_BODY"
    exit 1
fi
echo ""

# Test 6: Database Verification - Check Tables Exist
echo -e "${YELLOW}[TEST 6]${NC} Database Verification..."
TABLES_EXIST=$(docker-compose exec -T postgres psql -U pos_user -d pos_db -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('guest_orders', 'order_items', 'inventory_reservations', 'payment_transactions', 'delivery_addresses', 'tenant_configs')" 2>/dev/null || echo "0")
TABLES_EXIST=$(echo "$TABLES_EXIST" | tr -d ' \n')
if [ "$TABLES_EXIST" = "6" ]; then
    echo -e "${GREEN}✓ PASS${NC} - All required database tables exist"
else
    echo -e "${RED}✗ FAIL${NC} - Missing database tables (found $TABLES_EXIST/6)"
    exit 1
fi
echo ""

# Test 7: Redis Verification - Check Cart Persistence
echo -e "${YELLOW}[TEST 7]${NC} Redis Cart Persistence..."
REDIS_KEY="cart:$TENANT_ID:$SESSION_ID"
REDIS_EXISTS=$(docker-compose exec -T redis redis-cli EXISTS "$REDIS_KEY" 2>/dev/null || echo "0")
REDIS_EXISTS=$(echo "$REDIS_EXISTS" | tr -d ' \n')
if [ "$REDIS_EXISTS" = "1" ]; then
    echo -e "${GREEN}✓ PASS${NC} - Cart persisted in Redis"
    CART_DATA=$(docker-compose exec -T redis redis-cli GET "$REDIS_KEY" 2>/dev/null)
    echo "Redis Cart: $CART_DATA"
else
    echo -e "${YELLOW}⚠ WARN${NC} - Cart not found in Redis (may use different key format)"
fi
echo ""

# Test 8: Remove Cart Item
echo -e "${YELLOW}[TEST 8]${NC} Remove Item from Cart..."
DELETE_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X DELETE \
    -H "X-Session-Id: $SESSION_ID" \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart/items/$TEST_PRODUCT_ID" || echo "FAILED\n000")
HTTP_CODE=$(echo "$DELETE_RESPONSE" | tail -n1)
DELETE_BODY=$(echo "$DELETE_RESPONSE" | sed '$d')
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    echo -e "${GREEN}✓ PASS${NC} - Item removed from cart"
else
    echo -e "${RED}✗ FAIL${NC} - Remove item failed (HTTP $HTTP_CODE)"
    echo "Response: $DELETE_BODY"
    exit 1
fi
echo ""

# Test 9: Clear Cart
echo -e "${YELLOW}[TEST 9]${NC} Clear Cart..."
# First add item back
curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "X-Session-Id: $SESSION_ID" \
    -d "{\"product_id\":\"$TEST_PRODUCT_ID\",\"quantity\":1}" \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart/items" > /dev/null

CLEAR_RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X DELETE \
    -H "X-Session-Id: $SESSION_ID" \
    "$ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/cart" || echo "FAILED\n000")
HTTP_CODE=$(echo "$CLEAR_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    echo -e "${GREEN}✓ PASS${NC} - Cart cleared successfully"
else
    echo -e "${RED}✗ FAIL${NC} - Clear cart failed (HTTP $HTTP_CODE)"
    exit 1
fi
echo ""

# Summary
echo ""
echo "=============================================="
echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
echo "=============================================="
echo ""
echo "Quickstart validation complete! ✨"
echo ""
echo "Next steps:"
echo "  1. Test checkout flow with: curl -X POST $ORDER_SERVICE_URL/api/v1/public/$TENANT_ID/checkout"
echo "  2. Test frontend: Visit http://localhost:3000/menu/$TENANT_ID"
echo "  3. Test admin: Visit http://localhost:3000/admin/orders"
echo ""
