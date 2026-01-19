#!/bin/bash

# Start All POS Services
# This script starts all backend services and the frontend in development mode
# It loads environment variables from .env files
#
# Usage:
#   ./start-all.sh                     # Start all services
#   ./start-all.sh gateway             # Start only API Gateway
#   ./start-all.sh auth                # Start only Auth Service
#   ./start-all.sh user                # Start only User Service
#   ./start-all.sh tenant              # Start only Tenant Service
#   ./start-all.sh notification        # Start only Notification Service
#   ./start-all.sh frontend            # Start only Frontend
#   ./start-all.sh auth user tenant    # Start multiple services
#   ./start-all.sh all with-vault          # Start all services with Vault
#   ./start-all.sh all with-observability  # Start all services with Observability
#   ./start-all.sh all with-vault with-observability # Start all services with Vault and Observability

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
TARGET_SERVICES=()
START_ALL=false
WITH_VAULT=false
WITH_OBSERVABILITY=false

if [ $# -eq 0 ]; then
    START_ALL=true
else
    for arg in "$@"; do
        case $arg in
            gateway|api-gateway)
                TARGET_SERVICES+=("gateway")
                ;;
            auth|auth-service)
                TARGET_SERVICES+=("auth")
                ;;
            user|user-service)
                TARGET_SERVICES+=("user")
                ;;
            tenant|tenant-service)
                TARGET_SERVICES+=("tenant")
                ;;
            notification|notification-service)
                TARGET_SERVICES+=("notification")
                ;;
            product|product-service)
                TARGET_SERVICES+=("product")
                ;;
            order|order-service)
                TARGET_SERVICES+=("order")
                ;;
            audit|audit-service)
                TARGET_SERVICES+=("audit")
                ;;
            frontend|web)
                TARGET_SERVICES+=("frontend")
                ;;
            all)
                START_ALL=true
                ;;
            with-vault)
                WITH_VAULT=true
                ;;
            with-observability)
                WITH_OBSERVABILITY=true
                ;;
            *)
                echo "❌ Unknown service: $arg"
                echo ""
                echo "Available services:"
                echo "  gateway          - API Gateway"
                echo "  auth             - Auth Service"
                echo "  user             - User Service"
                echo "  tenant           - Tenant Service"
                echo "  notification     - Notification Service"
                echo "  product          - Product Service"
                echo "  order            - Order Service"
                echo "  audit            - Audit Service"
                echo "  frontend         - Frontend (Next.js)"
                echo "  all              - All services (default)"
                echo ""
                exit 1
                ;;
        esac
    done
fi

# Helper function to check if service should start
should_start_service() {
    local service=$1
    if [ "$START_ALL" = true ]; then
        return 0
    fi
    for target in "${TARGET_SERVICES[@]}"; do
        if [ "$target" = "$service" ]; then
            return 0
        fi
    done
    return 1
}

echo "🚀 Starting Point of Sale System Services"
echo "=========================================="

if [ "$START_ALL" = true ]; then
    echo "🎯 Target: All services"
else
    echo "🎯 Target: ${TARGET_SERVICES[*]}"
fi
echo ""

if [ "$WITH_VAULT" = true ]; then
    echo "🔐 Vault integration: Enabled"
else
    echo "🔐 Vault integration: Disabled"
fi
echo ""

if [ "$WITH_OBSERVABILITY" = true ]; then
    echo "📊 Observability stack: Enabled"
else
    echo "📊 Observability stack: Disabled"
fi
echo ""

sleep 1

# Load environment variables from root .env if it exists
if [ -f "$PROJECT_ROOT/.env" ]; then
    echo "📋 Loading environment variables from .env"
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
    echo "✅ Environment variables loaded"
    echo ""
else
    echo "⚠️  Warning: .env file not found in project root"
    echo "    Run: ./scripts/setup-env.sh to create environment files"
    echo ""
fi

# Check if service .env files exist
if [ "$START_ALL" = true ] || should_start_service "gateway" || should_start_service "auth" || should_start_service "user" || should_start_service "tenant" || should_start_service "notification" || should_start_service "product" || should_start_service "audit"; then
    echo "🔍 Checking service configuration files..."
    services_to_check=()
    
    if [ "$START_ALL" = true ] || should_start_service "gateway"; then
        services_to_check+=("api-gateway/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "auth"; then
        services_to_check+=("backend/auth-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "tenant"; then
        services_to_check+=("backend/tenant-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "user"; then
        services_to_check+=("backend/user-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "notification"; then
        services_to_check+=("backend/notification-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "product"; then
        services_to_check+=("backend/product-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "audit"; then
        services_to_check+=("backend/audit-service/.env")
    fi

    missing_files=false
    for service_env in "${services_to_check[@]}"; do
        if [ ! -f "$PROJECT_ROOT/$service_env" ]; then
            echo "❌ Missing: $service_env"
            missing_files=true
        else
            echo "✅ Found: $service_env"
        fi
    done

    if [ "$missing_files" = true ]; then
        echo ""
        echo "⚠️  Some .env files are missing!"
        echo "    Run: ./scripts/setup-env.sh to create them"
        echo ""
        exit 1
    fi
    
    echo ""
fi

# Check if Docker is running (only if starting backend services)
if [ "$START_ALL" = true ] || should_start_service "gateway" || should_start_service "auth" || should_start_service "user" || should_start_service "tenant" || should_start_service "notification" || should_start_service "product" || should_start_service "audit"; then
    if ! docker info > /dev/null 2>&1; then
        echo "⚠️  Warning: Docker is not running. Database and Redis will not be available."
        echo "    Services will attempt to start but may fail without database connectivity."
        echo ""
    fi

    # Start vault from directory /vault if available
    if [ "$WITH_VAULT" = true ] && [ -d "$PROJECT_ROOT/vault" ]; then
        echo "🔐 Starting Vault server..."
        cd "$PROJECT_ROOT/vault"
        docker compose up -d &
        echo "✅ Vault server started"
        echo ""
    fi

    if [ "$WITH_OBSERVABILITY" = true ] && [ -d "$PROJECT_ROOT/observability" ]; then
        echo "📊 Starting Observability stack (Prometheus & Grafana)..."
        cd "$PROJECT_ROOT/observability"
        docker compose up -d &
        echo "✅ Observability stack started"
        echo ""
    fi

    # Start Docker services if available
    if docker info > /dev/null 2>&1; then
        echo "📦 Starting Docker services (PostgreSQL & Redis)..."
        cd "$PROJECT_ROOT"
        docker compose up -d
        echo "✅ Docker services started"
        echo ""
        
        # Wait for PostgreSQL to be ready
        echo "⏳ Waiting for PostgreSQL to be ready..."
        for i in {1..30}; do
            if docker compose exec -T postgres pg_isready -U pos_user -d pos_db > /dev/null 2>&1; then
                echo "✅ PostgreSQL is ready"
                break
            fi
            if [ $i -eq 30 ]; then
                echo "❌ PostgreSQL did not become ready in time"
                exit 1
            fi
            sleep 1
        done
        echo ""
    fi
fi

# Build services
if [ "$START_ALL" = true ] || should_start_service "gateway" || should_start_service "auth" || should_start_service "user" || should_start_service "tenant" || should_start_service "notification" || should_start_service "product" || should_start_service "audit"; then
    echo "🔨 Building services..."
    
    if [ "$START_ALL" = true ] || should_start_service "gateway"; then
        cd "$PROJECT_ROOT/api-gateway" && go build -o api-gateway.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "auth"; then
        cd "$PROJECT_ROOT/backend/auth-service" && go build -o auth-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "tenant"; then
        cd "$PROJECT_ROOT/backend/tenant-service" && go build -o tenant-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "user"; then
        cd "$PROJECT_ROOT/backend/user-service" && go build -o user-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "notification"; then
        cd "$PROJECT_ROOT/backend/notification-service" && go build -o notification-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "product"; then
        cd "$PROJECT_ROOT/backend/product-service" && go build -o product-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "order"; then
        cd "$PROJECT_ROOT/backend/order-service" && go build -o order-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "audit"; then
        cd "$PROJECT_ROOT/backend/audit-service" && go build -o audit-service.bin main.go &
    fi
    
    wait
    echo "✅ Services built"
    echo ""
fi

# Start services in background with .env files
echo "🎯 Starting services..."

# Helper function to start service with .env
start_service_with_env() {
    local service_name=$1
    local service_dir=$2
    local binary_name=$3
    local log_file=$4
    
    cd "$service_dir"
    
    # Load service-specific .env if it exists
    if [ -f ".env" ]; then
        export $(grep -v '^#' .env | xargs)
    fi
    
    ./"$binary_name".bin > "$log_file" 2>&1 &
    local pid=$!
    echo "✅ $service_name started (PID: $pid)"
    
    # Store PID for later
    echo $pid >> /tmp/pos-services.pid
}

# Create/clear PID file
> /tmp/pos-services.pid

# Start services based on arguments
if [ "$START_ALL" = true ] || should_start_service "gateway"; then
    start_service_with_env "API Gateway" "$PROJECT_ROOT/api-gateway" "api-gateway" "/tmp/api-gateway.log"
fi

if [ "$START_ALL" = true ] || should_start_service "tenant"; then
    start_service_with_env "Tenant Service" "$PROJECT_ROOT/backend/tenant-service" "tenant-service" "/tmp/tenant-service.log"
fi

if [ "$START_ALL" = true ] || should_start_service "auth"; then
    start_service_with_env "Auth Service" "$PROJECT_ROOT/backend/auth-service" "auth-service" "/tmp/auth-service.log"
fi

if [ "$START_ALL" = true ] || should_start_service "user"; then
    start_service_with_env "User Service" "$PROJECT_ROOT/backend/user-service" "user-service" "/tmp/user-service.log"
fi

if [ "$START_ALL" = true ] || should_start_service "notification"; then
    start_service_with_env "Notification Service" "$PROJECT_ROOT/backend/notification-service" "notification-service" "/tmp/notification-service.log"
fi

if [ "$START_ALL" = true ] || should_start_service "product"; then
    start_service_with_env "Product Service" "$PROJECT_ROOT/backend/product-service" "product-service" "/tmp/product-service.log"
fi

if [ "$START_ALL" = true ] || should_start_service "order"; then
    start_service_with_env "Order Service" "$PROJECT_ROOT/backend/order-service" "order-service" "/tmp/order-service.log"
fi

if [ "$START_ALL" = true ] || should_start_service "audit"; then
    start_service_with_env "Audit Service" "$PROJECT_ROOT/backend/audit-service" "audit-service" "/tmp/audit-service.log"
fi

# Wait a moment for services to start
sleep 2

# Start frontend
if [ "$START_ALL" = true ] || should_start_service "frontend"; then
    echo ""
    echo "🎨 Starting frontend..."
    cd "$PROJECT_ROOT/frontend"
    
    # Load frontend .env if it exists
    if [ -f ".env.local" ]; then
        export $(grep -v '^#' .env.local | xargs)
    fi
    
    # Clear PORT variable to use Next.js default (3000)
    unset PORT
    
    npm run dev > /tmp/frontend.log 2>&1 &
    frontend_pid=$!
    echo $frontend_pid >> /tmp/pos-services.pid
    echo "✅ Frontend started (PID: $frontend_pid)"
fi

echo ""
echo "=========================================="
echo "✨ All services started successfully!"
echo ""
echo "📍 Service URLs:"
echo "   API Gateway:          http://localhost:${API_GATEWAY_PORT:-8080}"
echo "   Auth Service:         http://localhost:${AUTH_SERVICE_PORT:-8082}"
echo "   User Service:         http://localhost:${USER_SERVICE_PORT:-8083}"
echo "   Tenant Service:       http://localhost:${TENANT_SERVICE_PORT:-8084}"
echo "   Notification Service: http://localhost:${NOTIFICATION_SERVICE_PORT:-8085}"
echo "   Product Service:      http://localhost:${PRODUCT_SERVICE_PORT:-8086}"
echo "   Order Service:        http://localhost:${ORDER_SERVICE_PORT:-8087}"
echo "   Audit Service:        http://localhost:${AUDIT_SERVICE_PORT:-8088}"
echo "   Frontend:             http://localhost:${FRONTEND_PORT:-3000}"
echo ""
echo "📋 Health Checks:"
echo "   curl http://localhost:${API_GATEWAY_PORT:-8080}/health"
echo "   curl http://localhost:${AUTH_SERVICE_PORT:-8082}/health"
echo "   curl http://localhost:${USER_SERVICE_PORT:-8083}/health"
echo "   curl http://localhost:${TENANT_SERVICE_PORT:-8084}/health"
echo "   curl http://localhost:${NOTIFICATION_SERVICE_PORT:-8085}/health"
echo "   curl http://localhost:${PRODUCT_SERVICE_PORT:-8086}/health"
echo "   curl http://localhost:${ORDER_SERVICE_PORT:-8087}/health"
echo "   curl http://localhost:${AUDIT_SERVICE_PORT:-8088}/health"
echo ""
echo "📝 Logs:"
echo "   tail -f /tmp/api-gateway.log"
echo "   tail -f /tmp/auth-service.log"
echo "   tail -f /tmp/user-service.log"
echo "   tail -f /tmp/tenant-service.log"
echo "   tail -f /tmp/notification-service.log"
echo "   tail -f /tmp/product-service.log"
echo "   tail -f /tmp/order-service.log"
echo "   tail -f /tmp/audit-service.log"
echo "   tail -f /tmp/frontend.log"
echo ""
echo "🔧 Configuration:"
echo "   Using .env files from service directories"
echo "   JWT_SECRET: ${JWT_SECRET:0:20}..." 
echo "   Database: ${POSTGRES_DB:-pos_db}@${POSTGRES_HOST:-localhost}:${POSTGRES_PORT:-5432}"
echo "   Redis: ${REDIS_HOST:-localhost:6379}"
echo ""
echo "🛑 To stop all services, run: ./scripts/stop-all.sh"
echo ""
