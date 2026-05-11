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
            analytics|analytics-service)
                TARGET_SERVICES+=("analytics")
                ;;
            billing|billing-service)
                TARGET_SERVICES+=("billing")
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
                echo "  analytics        - Analytics Service"
                echo "  billing          - Billing Service"
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

# Keep root values available after service-specific .env files are loaded.
ROOT_POSTGRES_DB="${POSTGRES_DB:-pos_db}"
ROOT_POSTGRES_USER="${POSTGRES_USER:-pos_user}"
ROOT_POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-pos_password}"
ROOT_POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
ROOT_POSTGRES_PORT="${POSTGRES_PORT:-5432}"
ROOT_REDIS_HOST="${REDIS_HOST:-localhost}"
ROOT_REDIS_PORT="${REDIS_PORT:-6379}"
ROOT_REDIS_PASSWORD="${REDIS_PASSWORD:-pos_password}"
ROOT_KAFKA_BROKER="${KAFKA_BROKER:-localhost:9092}"
ROOT_API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
ROOT_AUTH_SERVICE_PORT="${AUTH_SERVICE_PORT:-8082}"
ROOT_USER_SERVICE_PORT="${USER_SERVICE_PORT:-8083}"
ROOT_TENANT_SERVICE_PORT="${TENANT_SERVICE_PORT:-8084}"
ROOT_NOTIFICATION_SERVICE_PORT="${NOTIFICATION_SERVICE_PORT:-8085}"
ROOT_PRODUCT_SERVICE_PORT="${PRODUCT_SERVICE_PORT:-8086}"
ROOT_ORDER_SERVICE_PORT="${ORDER_SERVICE_PORT:-8087}"
ROOT_AUDIT_SERVICE_PORT="${AUDIT_SERVICE_PORT:-8088}"
ROOT_ANALYTICS_SERVICE_PORT="${ANALYTICS_SERVICE_PORT:-8089}"
ROOT_BILLING_SERVICE_PORT="${BILLING_SERVICE_PORT:-8090}"
ROOT_BILLING_SERVICE_URL="${BILLING_SERVICE_URL:-http://localhost:${ROOT_BILLING_SERVICE_PORT}}"

local_redis_addr() {
    if [[ "$ROOT_REDIS_HOST" == *":"* ]]; then
        echo "$ROOT_REDIS_HOST"
    else
        echo "${ROOT_REDIS_HOST}:${ROOT_REDIS_PORT}"
    fi
}

local_redis_host() {
    if [[ "$ROOT_REDIS_HOST" == *":"* ]]; then
        echo "${ROOT_REDIS_HOST%%:*}"
    else
        echo "$ROOT_REDIS_HOST"
    fi
}

local_redis_url() {
    local redis_addr
    redis_addr="$(local_redis_addr)"
    if [ -n "$ROOT_REDIS_PASSWORD" ]; then
        echo "redis://:${ROOT_REDIS_PASSWORD}@${redis_addr}/0"
    else
        echo "redis://${redis_addr}/0"
    fi
}

local_database_url() {
    echo "postgresql://${ROOT_POSTGRES_USER}:${ROOT_POSTGRES_PASSWORD}@${ROOT_POSTGRES_HOST}:${ROOT_POSTGRES_PORT}/${ROOT_POSTGRES_DB}?sslmode=disable"
}

apply_local_runtime_overrides() {
    local service_key=$1
    local service_port=$2
    local redis_addr
    local redis_url
    local db_url

    redis_addr="$(local_redis_addr)"
    redis_url="$(local_redis_url)"
    db_url="$(local_database_url)"

    export PORT="$service_port"
    export DATABASE_URL="$db_url"
    export DB_HOST="$ROOT_POSTGRES_HOST"
    export DB_PORT="$ROOT_POSTGRES_PORT"
    export DB_USER="$ROOT_POSTGRES_USER"
    export DB_PASSWORD="$ROOT_POSTGRES_PASSWORD"
    export DB_NAME="$ROOT_POSTGRES_DB"
    export KAFKA_BROKERS="$ROOT_KAFKA_BROKER"
    export REDIS_PASSWORD="$ROOT_REDIS_PASSWORD"
    export REDIS_DB="${REDIS_DB:-0}"

    case "$service_key" in
        gateway)
            export REDIS_HOST="$redis_addr"
            export AUTH_SERVICE_URL="http://localhost:${ROOT_AUTH_SERVICE_PORT}"
            export USER_SERVICE_URL="http://localhost:${ROOT_USER_SERVICE_PORT}"
            export TENANT_SERVICE_URL="http://localhost:${ROOT_TENANT_SERVICE_PORT}"
            export PRODUCT_SERVICE_URL="http://localhost:${ROOT_PRODUCT_SERVICE_PORT}"
            export ORDER_SERVICE_URL="http://localhost:${ROOT_ORDER_SERVICE_PORT}"
            export NOTIFICATION_SERVICE_URL="http://localhost:${ROOT_NOTIFICATION_SERVICE_PORT}"
            export AUDIT_SERVICE_URL="http://localhost:${ROOT_AUDIT_SERVICE_PORT}"
            export ANALYTICS_SERVICE_URL="http://localhost:${ROOT_ANALYTICS_SERVICE_PORT}"
            export BILLING_SERVICE_URL="$ROOT_BILLING_SERVICE_URL"
            ;;
        auth|user|tenant|notification|product)
            export REDIS_HOST="$redis_addr"
            export REDIS_URL="$redis_url"
            ;;
        order)
            export REDIS_HOST="$(local_redis_host)"
            export REDIS_PORT="$ROOT_REDIS_PORT"
            export REDIS_URL="$redis_url"
            export TENANT_SERVICE_URL="http://localhost:${ROOT_TENANT_SERVICE_PORT}"
            export MIDTRANS_WEBHOOK_URL="http://localhost:${ROOT_API_GATEWAY_PORT}/api/v1/webhooks/payments/midtrans/notification"
            ;;
        analytics)
            export REDIS_HOST="$(local_redis_host)"
            export REDIS_PORT="$ROOT_REDIS_PORT"
            ;;
    esac

    if [ "$service_key" = "tenant" ]; then
        export NOTIFICATION_SERVICE_URL="http://localhost:${ROOT_NOTIFICATION_SERVICE_PORT}"
    fi

    if [ "$service_key" = "notification" ] && [ "$SMTP_HOST" = "mailhog" ]; then
        export SMTP_HOST="localhost"
    fi

    if [ "$service_key" = "product" ] && [ "$S3_ENDPOINT" = "minio:9000" ]; then
        export S3_ENDPOINT="localhost:9000"
    fi
}

# Check if service .env files exist
if [ "$START_ALL" = true ] || should_start_service "gateway" || should_start_service "auth" || should_start_service "user" || should_start_service "tenant" || should_start_service "notification" || should_start_service "product" || should_start_service "order" || should_start_service "audit" || should_start_service "analytics" || should_start_service "frontend" || should_start_service "billing"; then
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
    if [ "$START_ALL" = true ] || should_start_service "order"; then
        services_to_check+=("backend/order-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "audit"; then
        services_to_check+=("backend/audit-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "analytics"; then
        services_to_check+=("backend/analytics-service/.env")
    fi
    if [ "$START_ALL" = true ] || should_start_service "frontend"; then
        services_to_check+=("frontend/.env.local")
    fi

    if [ "$START_ALL" = true ] || should_start_service "billing"; then
        services_to_check+=("backend/billing-service/.env")
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
if [ "$START_ALL" = true ] || should_start_service "gateway" || should_start_service "auth" || should_start_service "user" || should_start_service "tenant" || should_start_service "notification" || should_start_service "product" || should_start_service "order" || should_start_service "audit" || should_start_service "analytics" || should_start_service "billing"; then
    if ! docker info > /dev/null 2>&1; then
        echo "⚠️  Warning: Docker is not running. Database and Redis will not be available."
        echo "    Services will attempt to start but may fail without database connectivity."
        echo ""
    fi

    # Start Docker services if available
    if docker info > /dev/null 2>&1; then
        docker network inspect pos-network > /dev/null 2>&1 || docker network create pos-network > /dev/null

        echo "📦 Starting Docker services (PostgreSQL, Redis, Kafka, Minio, Mailhog)..."
        cd "$PROJECT_ROOT"
        docker compose up -d postgres redis kafka minio mailhog
        echo "✅ Docker services started"
        echo ""

        # Start vault from directory /vault if available
        if [ "$WITH_VAULT" = true ] && [ -d "$PROJECT_ROOT/vault" ]; then
            echo "🔐 Starting Vault server..."
            cd "$PROJECT_ROOT/vault"
            docker compose up -d
            echo "✅ Vault server started"
            echo ""
        fi

        if [ "$WITH_OBSERVABILITY" = true ] && [ -d "$PROJECT_ROOT/observability" ]; then
            echo "📊 Starting Observability stack (Prometheus & Grafana)..."
            cd "$PROJECT_ROOT/observability"
            docker compose up -d
            echo "✅ Observability stack started"
            echo ""
        fi
        
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

        # Apply latest database schema before services boot.
        echo "🗃️  Running database migrations..."
        if [ -x "$PROJECT_ROOT/scripts/run-migrations.sh" ]; then
            "$PROJECT_ROOT/scripts/run-migrations.sh"
        else
            chmod +x "$PROJECT_ROOT/scripts/run-migrations.sh"
            "$PROJECT_ROOT/scripts/run-migrations.sh"
        fi
        echo ""
    fi
fi

# Build services
if [ "$START_ALL" = true ] || should_start_service "gateway" || should_start_service "auth" || should_start_service "user" || should_start_service "tenant" || should_start_service "notification" || should_start_service "product" || should_start_service "order" || should_start_service "audit" || should_start_service "analytics" || should_start_service "billing"; then
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
    if [ "$START_ALL" = true ] || should_start_service "analytics"; then
        cd "$PROJECT_ROOT/backend/analytics-service" && go build -o analytics-service.bin main.go &
    fi
    if [ "$START_ALL" = true ] || should_start_service "billing"; then
        cd "$PROJECT_ROOT/backend/billing-service" && go build -o billing-service.bin main.go &
    fi
    
    wait
    echo "✅ Services built"
    echo ""
fi

# Start services in background with .env files
echo "🎯 Starting services..."

# Helper function to start service with .env
start_service_with_env() {
    local service_key=$1
    local service_name=$2
    local service_dir=$3
    local binary_name=$4
    local log_file=$5
    local service_port=$6
    
    cd "$service_dir"
    
    # Load service-specific .env if it exists
    if [ -f ".env" ]; then
        export $(grep -v '^#' .env | xargs)
    fi

    # Local development runs binaries on the host while Compose starts infra only.
    # Override Docker-network defaults from service .env files with root .env values.
    apply_local_runtime_overrides "$service_key" "$service_port"
    
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
    start_service_with_env "gateway" "API Gateway" "$PROJECT_ROOT/api-gateway" "api-gateway" "/tmp/api-gateway.log" "$ROOT_API_GATEWAY_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "tenant"; then
    start_service_with_env "tenant" "Tenant Service" "$PROJECT_ROOT/backend/tenant-service" "tenant-service" "/tmp/tenant-service.log" "$ROOT_TENANT_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "auth"; then
    start_service_with_env "auth" "Auth Service" "$PROJECT_ROOT/backend/auth-service" "auth-service" "/tmp/auth-service.log" "$ROOT_AUTH_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "user"; then
    start_service_with_env "user" "User Service" "$PROJECT_ROOT/backend/user-service" "user-service" "/tmp/user-service.log" "$ROOT_USER_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "notification"; then
    start_service_with_env "notification" "Notification Service" "$PROJECT_ROOT/backend/notification-service" "notification-service" "/tmp/notification-service.log" "$ROOT_NOTIFICATION_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "product"; then
    start_service_with_env "product" "Product Service" "$PROJECT_ROOT/backend/product-service" "product-service" "/tmp/product-service.log" "$ROOT_PRODUCT_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "order"; then
    start_service_with_env "order" "Order Service" "$PROJECT_ROOT/backend/order-service" "order-service" "/tmp/order-service.log" "$ROOT_ORDER_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "audit"; then
    start_service_with_env "audit" "Audit Service" "$PROJECT_ROOT/backend/audit-service" "audit-service" "/tmp/audit-service.log" "$ROOT_AUDIT_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "analytics"; then
    start_service_with_env "analytics" "Analytics Service" "$PROJECT_ROOT/backend/analytics-service" "analytics-service" "/tmp/analytics-service.log" "$ROOT_ANALYTICS_SERVICE_PORT"
fi

if [ "$START_ALL" = true ] || should_start_service "billing"; then
    start_service_with_env "billing" "Billing Service" "$PROJECT_ROOT/backend/billing-service" "billing-service" "/tmp/billing-service.log" "$ROOT_BILLING_SERVICE_PORT"
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
echo "   Analytics Service:    http://localhost:${ANALYTICS_SERVICE_PORT:-8089}"
echo "   Billing Service:      http://localhost:${BILLING_SERVICE_PORT:-8090}"
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
echo "   curl http://localhost:${ANALYTICS_SERVICE_PORT:-8089}/health"
echo "   curl http://localhost:${BILLING_SERVICE_PORT:-8090}/health"
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
echo "   tail -f /tmp/analytics-service.log"
echo "   tail -f /tmp/billing-service.log"
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
