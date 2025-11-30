#!/bin/bash

# Stop All POS Services
# This script stops all backend services, frontend, and Docker containers
# It loads port configuration from .env files
#
# Usage:
#   ./stop-all.sh                     # Stop all services
#   ./stop-all.sh gateway             # Stop only API Gateway
#   ./stop-all.sh auth                # Stop only Auth Service
#   ./stop-all.sh user                # Stop only User Service
#   ./stop-all.sh tenant              # Stop only Tenant Service
#   ./stop-all.sh notification        # Stop only Notification Service
#   ./stop-all.sh frontend            # Stop only Frontend
#   ./stop-all.sh auth user tenant    # Stop multiple services

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
TARGET_SERVICES=()
STOP_ALL=false

if [ $# -eq 0 ]; then
    STOP_ALL=true
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
            frontend|web)
                TARGET_SERVICES+=("frontend")
                ;;
            all)
                STOP_ALL=true
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
                echo "  frontend         - Frontend (Next.js)"
                echo "  all              - All services (default)"
                echo ""
                exit 1
                ;;
        esac
    done
fi

# Helper function to check if service should stop
should_stop_service() {
    local service=$1
    if [ "$STOP_ALL" = true ]; then
        return 0
    fi
    for target in "${TARGET_SERVICES[@]}"; do
        if [ "$target" = "$service" ]; then
            return 0
        fi
    done
    return 1
}

echo "🛑 Stopping Point of Sale System Services"
echo "=========================================="

if [ "$STOP_ALL" = true ]; then
    echo "🎯 Target: All services"
else
    echo "🎯 Target: ${TARGET_SERVICES[*]}"
fi
echo ""

# Load environment variables from root .env if it exists
if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
fi

# Stop services using PID file if it exists
if [ -f "/tmp/pos-services.pid" ]; then
    echo "Stopping services from PID file..."
    while read pid; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            echo "✅ Stopped service (PID: $pid)"
        fi
    done < /tmp/pos-services.pid
    rm -f /tmp/pos-services.pid
    echo ""
fi

# Stop services by port (fallback method)
echo "Stopping services by port..."

# Use ports from environment or defaults
API_GATEWAY_PORT=${API_GATEWAY_PORT:-8080}
AUTH_SERVICE_PORT=${AUTH_SERVICE_PORT:-8082}
USER_SERVICE_PORT=${USER_SERVICE_PORT:-8083}
TENANT_SERVICE_PORT=${TENANT_SERVICE_PORT:-8084}
NOTIFICATION_SERVICE_PORT=${NOTIFICATION_SERVICE_PORT:-8085}
FRONTEND_PORT=${FRONTEND_PORT:-3000}

# Map ports to services
declare -A PORT_SERVICE_MAP
PORT_SERVICE_MAP[$API_GATEWAY_PORT]="gateway"
PORT_SERVICE_MAP[$AUTH_SERVICE_PORT]="auth"
PORT_SERVICE_MAP[$USER_SERVICE_PORT]="user"
PORT_SERVICE_MAP[$TENANT_SERVICE_PORT]="tenant"
PORT_SERVICE_MAP[$NOTIFICATION_SERVICE_PORT]="notification"
PORT_SERVICE_MAP[$FRONTEND_PORT]="frontend"

STOPPED_PORTS=()

for port in $API_GATEWAY_PORT $AUTH_SERVICE_PORT $USER_SERVICE_PORT $TENANT_SERVICE_PORT $NOTIFICATION_SERVICE_PORT $FRONTEND_PORT; do
    service_name=${PORT_SERVICE_MAP[$port]}
    
    if should_stop_service "$service_name"; then
        PID=$(lsof -ti:$port 2>/dev/null || echo "")
        if [ -n "$PID" ]; then
            kill $PID 2>/dev/null || true
            echo "✅ Stopped $service_name on port $port (PID: $PID)"
            STOPPED_PORTS+=($port)
        fi
    fi
done

if [ ${#STOPPED_PORTS[@]} -eq 0 ]; then
    echo "ℹ️  No services running on configured ports"
fi

# Stop Next.js processes if frontend should stop
if should_stop_service "frontend"; then
    echo ""
    echo "Stopping Next.js processes..."
    pkill -f "next dev" 2>/dev/null && echo "✅ Stopped Next.js dev server" || echo "ℹ️  No Next.js dev server running"
    pkill -f "next-server" 2>/dev/null && echo "✅ Stopped Next.js server" || true
    
    # Clean up Next.js lock file
    if [ -f "$PROJECT_ROOT/frontend/.next/dev/lock" ]; then
        rm -f "$PROJECT_ROOT/frontend/.next/dev/lock"
        echo "✅ Removed Next.js lock file"
    fi
fi

# Clean up log files
echo ""
echo "📝 Log files management..."

# Build list of log files based on stopped services
LOG_FILES=()
if should_stop_service "gateway"; then
    LOG_FILES+=("/tmp/api-gateway.log")
fi
if should_stop_service "auth"; then
    LOG_FILES+=("/tmp/auth-service.log")
fi
if should_stop_service "user"; then
    LOG_FILES+=("/tmp/user-service.log")
fi
if should_stop_service "tenant"; then
    LOG_FILES+=("/tmp/tenant-service.log")
fi
if should_stop_service "notification"; then
    LOG_FILES+=("/tmp/notification-service.log")
fi
if should_stop_service "frontend"; then
    LOG_FILES+=("/tmp/frontend.log")
fi

# Check if any log files exist
log_files_exist=false
for log_file in "${LOG_FILES[@]}"; do
    if [ -f "$log_file" ]; then
        log_files_exist=true
        break
    fi
done

if [ "$log_files_exist" = true ]; then
    echo "Found log files:"
    for log_file in "${LOG_FILES[@]}"; do
        if [ -f "$log_file" ]; then
            size=$(du -h "$log_file" | cut -f1)
            echo "  - $(basename $log_file) ($size)"
        fi
    done
    
    echo ""
    echo "Remove log files? (y/N): "
    read -r -t 10 REPLY || REPLY="N"
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        for log_file in "${LOG_FILES[@]}"; do
            rm -f "$log_file"
        done
        echo "✅ Log files removed"
    else
        echo "ℹ️  Log files kept in /tmp/"
    fi
else
    echo "ℹ️  No log files found"
fi

echo ""
echo "=========================================="
echo "✨ Services stopped successfully!"
echo ""

if [ "$STOP_ALL" = true ]; then
    echo "📊 Summary:"
    echo "   Stopped ports: $API_GATEWAY_PORT, $AUTH_SERVICE_PORT, $USER_SERVICE_PORT, $TENANT_SERVICE_PORT, $NOTIFICATION_SERVICE_PORT, $FRONTEND_PORT"
else
    if [ ${#STOPPED_PORTS[@]} -gt 0 ]; then
        echo "📊 Summary:"
        echo "   Stopped ports: ${STOPPED_PORTS[*]}"
    fi
fi

echo ""
echo "🚀 To start services again, run: ./scripts/start-all.sh"
echo ""
