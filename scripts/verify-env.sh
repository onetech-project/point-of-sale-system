#!/bin/bash

# Script to verify environment configuration
# Usage: ./scripts/verify-env.sh

set -e

echo "========================================="
echo "Verifying Environment Configuration"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if all .env files exist
echo "1. Checking .env files existence..."
echo ""

files=(
    ".env"
    "api-gateway/.env"
    "backend/auth-service/.env"
    "backend/user-service/.env"
    "backend/tenant-service/.env"
    "backend/notification-service/.env"
    "backend/product-service/.env"
    "backend/order-service/.env"
    "frontend/.env.local"
)

all_exist=true
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  ${GREEN}✓${NC} $file"
    else
        echo -e "  ${RED}✗${NC} $file ${RED}(missing)${NC}"
        all_exist=false
    fi
done

echo ""

if [ "$all_exist" = false ]; then
    echo -e "${YELLOW}⚠️  Some .env files are missing. Run: ./scripts/setup-env.sh${NC}"
    exit 1
fi

# Check JWT_SECRET consistency
echo "2. Checking JWT_SECRET consistency..."
echo ""

jwt_secret_root=$(grep "^JWT_SECRET=" .env | cut -d= -f2)
jwt_secret_gateway=$(grep "^JWT_SECRET=" api-gateway/.env | cut -d= -f2)
jwt_secret_auth=$(grep "^JWT_SECRET=" backend/auth-service/.env | cut -d= -f2)

if [ "$jwt_secret_root" = "$jwt_secret_gateway" ] && [ "$jwt_secret_gateway" = "$jwt_secret_auth" ]; then
    echo -e "  ${GREEN}✓${NC} JWT_SECRET is consistent across services"
    echo "    Value: $jwt_secret_root"
else
    echo -e "  ${RED}✗${NC} JWT_SECRET mismatch detected!"
    echo "    Root:        $jwt_secret_root"
    echo "    API Gateway: $jwt_secret_gateway"
    echo "    Auth:        $jwt_secret_auth"
    exit 1
fi

echo ""

# Check if using default JWT_SECRET
if [ "$jwt_secret_root" = "default-secret-change-in-production" ]; then
    echo -e "  ${YELLOW}⚠️  WARNING: Using default JWT_SECRET${NC}"
    echo -e "     ${YELLOW}Change this for production!${NC}"
fi

echo ""

# Check database configuration
echo "3. Checking database configuration..."
echo ""

db_url=$(grep "^DATABASE_URL=" backend/auth-service/.env | cut -d= -f2)
if [[ $db_url == *"pos_user:pos_password"* ]]; then
    echo -e "  ${GREEN}✓${NC} Database URL configured"
    if [[ $db_url == *"pos_password"* ]]; then
        echo -e "  ${YELLOW}⚠️  WARNING: Using default database password${NC}"
        echo -e "     ${YELLOW}Change this for production!${NC}"
    fi
else
    echo -e "  ${RED}✗${NC} Database URL not properly configured"
    exit 1
fi

echo ""

# Check Redis configuration
echo "4. Checking Redis configuration..."
echo ""

redis_host=$(grep "^REDIS_HOST=" backend/auth-service/.env | cut -d= -f2)
if [ -n "$redis_host" ]; then
    echo -e "  ${GREEN}✓${NC} Redis host configured: $redis_host"
else
    echo -e "  ${RED}✗${NC} Redis host not configured"
    exit 1
fi

echo ""

# Check API URL in frontend
echo "5. Checking frontend API URL..."
echo ""

api_url=$(grep "^NEXT_PUBLIC_API_URL=" frontend/.env.local | cut -d= -f2)
if [ -n "$api_url" ]; then
    echo -e "  ${GREEN}✓${NC} API URL configured: $api_url"
else
    echo -e "  ${RED}✗${NC} API URL not configured in frontend"
    exit 1
fi

echo ""

# Summary
echo "========================================="
echo -e "${GREEN}✓ Environment configuration verified!${NC}"
echo "========================================="
echo ""
echo "📝 Recommendations:"
echo ""
echo "  1. Review and update JWT_SECRET for production"
echo "  2. Update database password for production"
echo "  3. Configure SMTP settings for email"
echo "  4. Set ENVIRONMENT=production when deploying"
echo ""
echo "📖 For more details, see: docs/ENVIRONMENT.md"
echo ""
