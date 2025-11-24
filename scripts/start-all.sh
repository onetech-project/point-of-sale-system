#!/bin/bash

# Start All POS Services
# This script starts all backend services and the frontend in development mode
# It loads environment variables from .env files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🚀 Starting Point of Sale System Services"
echo "=========================================="

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
echo "🔍 Checking service configuration files..."
services_to_check=(
    "api-gateway/.env"
    "backend/auth-service/.env"
    "backend/tenant-service/.env"
    "backend/user-service/.env"
)

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

# # Check if Docker is running
# if ! docker info > /dev/null 2>&1; then
#     echo "⚠️  Warning: Docker is not running. Database and Redis will not be available."
#     echo "    Services will attempt to start but may fail without database connectivity."
#     echo ""
# fi

# # Start Docker services if available
# if docker info > /dev/null 2>&1; then
#     echo "📦 Starting Docker services (PostgreSQL & Redis)..."
#     cd "$PROJECT_ROOT"
#     docker-compose up -d
#     echo "✅ Docker services started"
#     echo ""
    
#     # Wait for PostgreSQL to be ready
#     echo "⏳ Waiting for PostgreSQL to be ready..."
#     for i in {1..30}; do
#         if docker-compose exec -T postgres pg_isready -U pos_user -d pos_db > /dev/null 2>&1; then
#             echo "✅ PostgreSQL is ready"
#             break
#         fi
#         if [ $i -eq 30 ]; then
#             echo "❌ PostgreSQL did not become ready in time"
#             exit 1
#         fi
#         sleep 1
#     done
#     echo ""
# fi

# Build all services
echo "🔨 Building services..."
cd "$PROJECT_ROOT/api-gateway" && go build -o api-gateway main.go &
cd "$PROJECT_ROOT/backend/auth-service" && go build -o auth-service main.go &
cd "$PROJECT_ROOT/backend/tenant-service" && go build -o tenant-service main.go &
cd "$PROJECT_ROOT/backend/user-service" && go build -o user-service main.go &
wait
echo "✅ All services built"
echo ""

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
    
    ./"$binary_name" > "$log_file" 2>&1 &
    local pid=$!
    echo "✅ $service_name started (PID: $pid)"
    
    # Store PID for later
    echo $pid >> /tmp/pos-services.pid
}

# Create/clear PID file
> /tmp/pos-services.pid

# Start API Gateway
start_service_with_env "API Gateway" "$PROJECT_ROOT/api-gateway" "api-gateway" "/tmp/api-gateway.log"

# Start Tenant Service
start_service_with_env "Tenant Service" "$PROJECT_ROOT/backend/tenant-service" "tenant-service" "/tmp/tenant-service.log"

# Start Auth Service
start_service_with_env "Auth Service" "$PROJECT_ROOT/backend/auth-service" "auth-service" "/tmp/auth-service.log"

# Start User Service
start_service_with_env "User Service" "$PROJECT_ROOT/backend/user-service" "user-service" "/tmp/user-service.log"

# Wait a moment for services to start
sleep 2

# # Start frontend
# echo ""
# echo "🎨 Starting frontend..."
# cd "$PROJECT_ROOT/frontend"
# npm run dev > /tmp/frontend.log 2>&1 &
# echo "✅ Frontend started on port 3000 (PID: $!)"

echo ""
echo "=========================================="
echo "✨ All services started successfully!"
echo ""
echo "📍 Service URLs:"
echo "   API Gateway:      http://localhost:${API_GATEWAY_PORT:-8080}"
echo "   Tenant Service:   http://localhost:${TENANT_SERVICE_PORT:-8084}"
echo "   Auth Service:     http://localhost:${AUTH_SERVICE_PORT:-8082}"
echo "   User Service:     http://localhost:${USER_SERVICE_PORT:-8083}"
echo "   Frontend:         http://localhost:${FRONTEND_PORT:-3000}"
echo ""
echo "📋 Health Checks:"
echo "   curl http://localhost:${API_GATEWAY_PORT:-8080}/health"
echo "   curl http://localhost:${TENANT_SERVICE_PORT:-8084}/health"
echo "   curl http://localhost:${AUTH_SERVICE_PORT:-8082}/health"
echo "   curl http://localhost:${USER_SERVICE_PORT:-8083}/health"
echo ""
echo "📝 Logs:"
echo "   tail -f /tmp/api-gateway.log"
echo "   tail -f /tmp/tenant-service.log"
echo "   tail -f /tmp/auth-service.log"
echo "   tail -f /tmp/user-service.log"
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
