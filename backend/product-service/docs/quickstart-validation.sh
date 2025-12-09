#!/bin/bash

# T157: Quickstart Validation Script
# Tests all API examples from specs/004-product-inventory/quickstart.md
# Run this script after product-service is running to validate all endpoints

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8086}"
TEST_TENANT_ID="${TEST_TENANT_ID:-00000000-0000-0000-0000-000000000001}"

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to print test results
print_result() {
    local test_name="$1"
    local result="$2"
    local response="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        echo -e "  Response: $response"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Function to make API call
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ -z "$data" ]; then
        curl -s -X "$method" "$API_BASE_URL$endpoint" \
            -H "X-Tenant-ID: $TEST_TENANT_ID" \
            -H "Content-Type: application/json"
    else
        curl -s -X "$method" "$API_BASE_URL$endpoint" \
            -H "X-Tenant-ID: $TEST_TENANT_ID" \
            -H "Content-Type: application/json" \
            -d "$data"
    fi
}

echo "========================================"
echo "Product Service API Validation"
echo "========================================"
echo "API Base URL: $API_BASE_URL"
echo "Test Tenant ID: $TEST_TENANT_ID"
echo ""

# Test 1: Health Check
echo "Testing Health Check Endpoints..."
response=$(curl -s "$API_BASE_URL/health")
if echo "$response" | grep -q "ok"; then
    print_result "GET /health" "PASS" "$response"
else
    print_result "GET /health" "FAIL" "$response"
fi

response=$(curl -s "$API_BASE_URL/ready")
if echo "$response" | grep -q "database"; then
    print_result "GET /ready" "PASS" "$response"
else
    print_result "GET /ready" "FAIL" "$response"
fi

echo ""

# Test 2: Create Category
echo "Testing Category Management..."
CATEGORY_DATA='{
    "name": "Test Category",
    "description": "Category for testing"
}'

response=$(api_call "POST" "/api/v1/categories" "$CATEGORY_DATA")
if echo "$response" | grep -q "id"; then
    CATEGORY_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    print_result "POST /categories (Create)" "PASS" "Category ID: $CATEGORY_ID"
else
    print_result "POST /categories (Create)" "FAIL" "$response"
    CATEGORY_ID="00000000-0000-0000-0000-000000000002"
fi

echo ""

# Test 3: List Categories
echo "Testing Category List..."
response=$(api_call "GET" "/api/v1/categories")
if echo "$response" | grep -q "Test Category"; then
    print_result "GET /categories (List)" "PASS" "Categories retrieved"
else
    print_result "GET /categories (List)" "FAIL" "$response"
fi

echo ""

# Test 4: Create Product
echo "Testing Product Management..."
PRODUCT_DATA='{
    "sku": "TEST-001",
    "name": "Test Product",
    "description": "Product for quickstart validation",
    "category_id": "'$CATEGORY_ID'",
    "selling_price": 19.99,
    "cost_price": 10.00,
    "tax_rate": 0.10,
    "stock_quantity": 50,
    "reorder_level": 10
}'

response=$(api_call "POST" "/api/v1/products" "$PRODUCT_DATA")
if echo "$response" | grep -q "id"; then
    PRODUCT_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    print_result "POST /products (Create)" "PASS" "Product ID: $PRODUCT_ID"
else
    print_result "POST /products (Create)" "FAIL" "$response"
    PRODUCT_ID="00000000-0000-0000-0000-000000000003"
fi

echo ""

# Test 5: Get Product by ID
echo "Testing Product Retrieval..."
response=$(api_call "GET" "/api/v1/products/$PRODUCT_ID")
if echo "$response" | grep -q "TEST-001"; then
    print_result "GET /products/{id}" "PASS" "Product retrieved"
else
    print_result "GET /products/{id}" "FAIL" "$response"
fi

echo ""

# Test 6: List Products
echo "Testing Product List..."
response=$(api_call "GET" "/api/v1/products")
if echo "$response" | grep -q "TEST-001"; then
    print_result "GET /products (List)" "PASS" "Products retrieved"
else
    print_result "GET /products (List)" "FAIL" "$response"
fi

echo ""

# Test 7: Search Products
echo "Testing Product Search..."
response=$(api_call "GET" "/api/v1/products?search=Test")
if echo "$response" | grep -q "TEST-001"; then
    print_result "GET /products?search=Test" "PASS" "Search results returned"
else
    print_result "GET /products?search=Test" "FAIL" "$response"
fi

echo ""

# Test 8: Filter by Category
echo "Testing Category Filter..."
response=$(api_call "GET" "/api/v1/products?category_id=$CATEGORY_ID")
if echo "$response" | grep -q "TEST-001"; then
    print_result "GET /products?category_id={id}" "PASS" "Filtered results returned"
else
    print_result "GET /products?category_id={id}" "FAIL" "$response"
fi

echo ""

# Test 9: Update Product
echo "Testing Product Update..."
UPDATE_DATA='{
    "sku": "TEST-001",
    "name": "Test Product (Updated)",
    "description": "Updated description",
    "category_id": "'$CATEGORY_ID'",
    "selling_price": 24.99,
    "cost_price": 12.00,
    "tax_rate": 0.10,
    "stock_quantity": 50,
    "reorder_level": 10
}'

response=$(api_call "PUT" "/api/v1/products/$PRODUCT_ID" "$UPDATE_DATA")
if echo "$response" | grep -q "Updated"; then
    print_result "PUT /products/{id}" "PASS" "Product updated"
else
    print_result "PUT /products/{id}" "FAIL" "$response"
fi

echo ""

# Test 10: Stock Adjustment
echo "Testing Stock Adjustment..."
ADJUSTMENT_DATA='{
    "quantity_delta": 25,
    "reason": "supplier_delivery",
    "notes": "Quickstart validation test"
}'

response=$(api_call "POST" "/api/v1/products/$PRODUCT_ID/stock" "$ADJUSTMENT_DATA")
if echo "$response" | grep -q "supplier_delivery"; then
    print_result "POST /products/{id}/stock" "PASS" "Stock adjusted"
else
    print_result "POST /products/{id}/stock" "FAIL" "$response"
fi

echo ""

# Test 11: Get Adjustment History
echo "Testing Adjustment History..."
response=$(api_call "GET" "/api/v1/products/$PRODUCT_ID/adjustments")
if echo "$response" | grep -q "supplier_delivery"; then
    print_result "GET /products/{id}/adjustments" "PASS" "History retrieved"
else
    print_result "GET /products/{id}/adjustments" "FAIL" "$response"
fi

echo ""

# Test 12: Inventory Summary
echo "Testing Inventory Summary..."
response=$(api_call "GET" "/api/v1/inventory/summary")
if echo "$response" | grep -q "total_products"; then
    print_result "GET /inventory/summary" "PASS" "Summary retrieved"
else
    print_result "GET /inventory/summary" "FAIL" "$response"
fi

echo ""

# Test 13: Low Stock Products
echo "Testing Low Stock Filter..."
# First, create a low stock product
LOW_STOCK_DATA='{
    "sku": "LOW-001",
    "name": "Low Stock Product",
    "description": "Product with low stock",
    "category_id": "'$CATEGORY_ID'",
    "selling_price": 9.99,
    "cost_price": 5.00,
    "tax_rate": 0.10,
    "stock_quantity": 5,
    "reorder_level": 10
}'

response=$(api_call "POST" "/api/v1/products" "$LOW_STOCK_DATA")
if echo "$response" | grep -q "id"; then
    LOW_STOCK_PRODUCT_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    print_result "POST /products (Low Stock)" "PASS" "Low stock product created"
    
    # Now check low stock filter
    response=$(api_call "GET" "/api/v1/products?low_stock=true")
    if echo "$response" | grep -q "LOW-001"; then
        print_result "GET /products?low_stock=true" "PASS" "Low stock filter works"
    else
        print_result "GET /products?low_stock=true" "FAIL" "$response"
    fi
else
    print_result "POST /products (Low Stock)" "FAIL" "$response"
    print_result "GET /products?low_stock=true" "SKIP" "Skipped due to previous failure"
fi

echo ""

# Test 14: Archive Product
echo "Testing Archive/Restore..."
response=$(api_call "PATCH" "/api/v1/products/$PRODUCT_ID/archive")
if [ -z "$response" ] || echo "$response" | grep -q "success"; then
    print_result "PATCH /products/{id}/archive" "PASS" "Product archived"
    
    # Verify archived product doesn't appear in default list
    response=$(api_call "GET" "/api/v1/products")
    if ! echo "$response" | grep -q "TEST-001"; then
        print_result "GET /products (Archived excluded)" "PASS" "Archived products excluded"
    else
        print_result "GET /products (Archived excluded)" "FAIL" "Archived product still visible"
    fi
    
    # Restore product
    response=$(api_call "PATCH" "/api/v1/products/$PRODUCT_ID/restore")
    if [ -z "$response" ] || echo "$response" | grep -q "success"; then
        print_result "PATCH /products/{id}/restore" "PASS" "Product restored"
    else
        print_result "PATCH /products/{id}/restore" "FAIL" "$response"
    fi
else
    print_result "PATCH /products/{id}/archive" "FAIL" "$response"
fi

echo ""

# Test 15: Delete Product (cleanup)
echo "Testing Product Deletion..."
response=$(api_call "DELETE" "/api/v1/products/$PRODUCT_ID")
if [ -z "$response" ] || ! echo "$response" | grep -q "error"; then
    print_result "DELETE /products/{id}" "PASS" "Product deleted"
else
    print_result "DELETE /products/{id}" "FAIL" "$response"
fi

# Cleanup low stock product if created
if [ ! -z "$LOW_STOCK_PRODUCT_ID" ]; then
    api_call "DELETE" "/api/v1/products/$LOW_STOCK_PRODUCT_ID" > /dev/null 2>&1
fi

# Cleanup category
api_call "DELETE" "/api/v1/categories/$CATEGORY_ID" > /dev/null 2>&1

echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed.${NC}"
    exit 1
fi
