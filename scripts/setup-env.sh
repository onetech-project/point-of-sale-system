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
echo "  - .env (database credentials, JWT secret)"
echo "  - api-gateway/.env (JWT secret must match)"
echo "  - backend/auth-service/.env (JWT secret must match)"
echo "  - backend/user-service/.env (email configuration)"
echo "  - backend/tenant-service/.env"
echo "  - backend/notification-service/.env (email, SMS, Kafka)"
echo "  - frontend/.env.local (API URL)"
echo ""
echo "📖 For detailed configuration instructions, see: docs/ENVIRONMENT.md"
echo ""
