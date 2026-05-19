#!/bin/bash

# Script to set up environment files for all services
# Usage: ./scripts/setup-env.sh

set -e

echo "========================================="
echo "Setting up environment files"
echo "========================================="

# Root directory
if [ ! -f ".env" ]; then
    echo "Creating root .env file..."
    cp .env.example .env
    echo "✓ Created .env"
else
    echo "✓ Root .env already exists"
fi

# API Gateway
if [ ! -f "api-gateway/.env" ]; then
    echo "Creating api-gateway/.env file..."
    cp api-gateway/.env.example api-gateway/.env
    echo "✓ Created api-gateway/.env"
else
    echo "✓ api-gateway/.env already exists"
fi

# Auth Service
if [ ! -f "backend/auth-service/.env" ]; then
    echo "Creating backend/auth-service/.env file..."
    cp backend/auth-service/.env.example backend/auth-service/.env
    echo "✓ Created backend/auth-service/.env"
else
    echo "✓ backend/auth-service/.env already exists"
fi

# User Service
if [ ! -f "backend/user-service/.env" ]; then
    echo "Creating backend/user-service/.env file..."
    cp backend/user-service/.env.example backend/user-service/.env
    echo "✓ Created backend/user-service/.env"
else
    echo "✓ backend/user-service/.env already exists"
fi

# Tenant Service
if [ ! -f "backend/tenant-service/.env" ]; then
    echo "Creating backend/tenant-service/.env file..."
    cp backend/tenant-service/.env.example backend/tenant-service/.env
    echo "✓ Created backend/tenant-service/.env"
else
    echo "✓ backend/tenant-service/.env already exists"
fi

# Notification Service
if [ ! -f "backend/notification-service/.env" ]; then
    echo "Creating backend/notification-service/.env file..."
    cp backend/notification-service/.env.example backend/notification-service/.env
    echo "✓ Created backend/notification-service/.env"
else
    echo "✓ backend/notification-service/.env already exists"
fi

# Product Service
if [ ! -f "backend/product-service/.env" ]; then
    echo "Creating backend/product-service/.env file..."
    cp backend/product-service/.env.example backend/product-service/.env
    echo "✓ Created backend/product-service/.env"
else
    echo "✓ backend/product-service/.env already exists"
fi

# Order Service
if [ ! -f "backend/order-service/.env" ]; then
    echo "Creating backend/order-service/.env file..."
    cp backend/order-service/.env.example backend/order-service/.env
    echo "✓ Created backend/order-service/.env"
else
    echo "✓ backend/order-service/.env already exists"
fi

# Audit Service
if [ ! -f "backend/audit-service/.env" ]; then
    echo "Creating backend/audit-service/.env file..."
    cp backend/audit-service/.env.example backend/audit-service/.env
    echo "✓ Created backend/audit-service/.env"
else
    echo "✓ backend/audit-service/.env already exists"
fi

# Analytics Service
if [ ! -f "backend/analytics-service/.env" ]; then
    echo "Creating backend/analytics-service/.env file..."
    cp backend/analytics-service/.env.example backend/analytics-service/.env
    echo "✓ Created backend/analytics-service/.env"
else
    echo "✓ backend/analytics-service/.env already exists"
fi

# Billing Service
if [ ! -f "backend/billing-service/.env" ]; then
    echo "Creating backend/billing-service/.env file..."
    cp backend/billing-service/.env.example backend/billing-service/.env
    echo "✓ Created backend/billing-service/.env"
else
    echo "✓ backend/billing-service/.env already exists"
fi
if ! grep -q "^KAFKA_AUDIT_TOPIC=" backend/billing-service/.env; then
    echo "KAFKA_AUDIT_TOPIC=audit-events" >> backend/billing-service/.env
    echo "✓ Added KAFKA_AUDIT_TOPIC to backend/billing-service/.env"
fi
if ! grep -q "^PLAN_RETENTION_DAYS=" backend/billing-service/.env; then
    echo "PLAN_RETENTION_DAYS=30" >> backend/billing-service/.env
    echo "✓ Added PLAN_RETENTION_DAYS to backend/billing-service/.env"
fi
if ! grep -q "^REDIS_HOST=" backend/billing-service/.env; then
    echo "REDIS_HOST=localhost:6379" >> backend/billing-service/.env
    echo "✓ Added REDIS_HOST to backend/billing-service/.env"
fi
if ! grep -q "^REDIS_PASSWORD=" backend/billing-service/.env; then
    echo "REDIS_PASSWORD=pos_password" >> backend/billing-service/.env
    echo "✓ Added REDIS_PASSWORD to backend/billing-service/.env"
fi

# Platform Service
if [ ! -f "backend/platform-service/.env" ]; then
    echo "Creating backend/platform-service/.env file..."
    cp backend/platform-service/.env.example backend/platform-service/.env
    echo "✓ Created backend/platform-service/.env"
else
    echo "✓ backend/platform-service/.env already exists"
fi

# Observability
if [ ! -f "observability/.env" ]; then
    echo "Creating observability/.env file..."
    cp observability/.env.example observability/.env
    echo "✓ Created observability/.env"
else
    echo "✓ observability/.env already exists"
fi

# Vault
if [ ! -f "vault/.env" ]; then
    echo "Creating vault/.env file..."
    cp vault/.env.example vault/.env
    echo "✓ Created vault/.env"
else
    echo "✓ vault/.env already exists"
fi

# Frontend
if [ ! -f "frontend/.env.local" ]; then
    echo "Creating frontend/.env.local file..."
    cp frontend/.env.example frontend/.env.local
    echo "✓ Created frontend/.env.local"
else
    echo "✓ frontend/.env.local already exists"
fi

echo ""
echo "========================================="
echo "Environment files setup complete!"
echo "========================================="
echo ""
echo "⚠️  IMPORTANT: Review and update the following files with your configuration:"
echo ""
echo "  - .env (database credentials, JWT secret, Vault token)"
echo "  - api-gateway/.env (JWT secret must match)"
echo "  - backend/auth-service/.env (JWT secret must match, Vault config)"
echo "  - backend/user-service/.env (email configuration, Vault config)"
echo "  - backend/tenant-service/.env (Vault config)"
echo "  - backend/notification-service/.env (email, SMS, Kafka, Vault config)"
echo "  - backend/product-service/.env (Vault config)"
echo "  - backend/order-service/.env (Midtrans, Google Maps, Vault config)"
echo "  - backend/audit-service/.env (Kafka, Vault config)"
echo "  - backend/analytics-service/.env (analytics DB, Vault config)"
echo "  - backend/billing-service/.env (Midtrans, billing plan config)"
echo "  - backend/platform-service/.env (platform admin bootstrap credentials)"
echo "  - observability/.env (Grafana, MinIO credentials)"
echo "  - vault/.env (Vault token)"
echo "  - frontend/.env.local (API URL)"
echo ""
echo "📖 For detailed configuration instructions, see: docs/ENVIRONMENT.md"
echo ""
