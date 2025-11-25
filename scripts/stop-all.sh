#!/bin/bash

# Stop All POS Services
# This script stops all backend services, frontend, and Docker containers
# It loads port configuration from .env files

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🛑 Stopping Point of Sale System Services"
echo "=========================================="

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
echo "Stopping services by port (fallback)..."

# Use ports from environment or defaults
API_GATEWAY_PORT=${API_GATEWAY_PORT:-8080}
AUTH_SERVICE_PORT=${AUTH_SERVICE_PORT:-8082}
USER_SERVICE_PORT=${USER_SERVICE_PORT:-8083}
TENANT_SERVICE_PORT=${TENANT_SERVICE_PORT:-8084}
NOTIFICATION_SERVICE_PORT=${NOTIFICATION_SERVICE_PORT:-8085}
FRONTEND_PORT=${FRONTEND_PORT:-3000}

for port in $API_GATEWAY_PORT $AUTH_SERVICE_PORT $USER_SERVICE_PORT $TENANT_SERVICE_PORT $NOTIFICATION_SERVICE_PORT $FRONTEND_PORT; do
    PID=$(lsof -ti:$port 2>/dev/null || echo "")
    if [ -n "$PID" ]; then
        kill $PID 2>/dev/null || true
        echo "✅ Stopped service on port $port (PID: $PID)"
    fi
done

# Also stop Next.js dev server by process name (in case port check missed it)
echo ""
echo "Stopping Next.js processes..."
pkill -f "next dev" 2>/dev/null && echo "✅ Stopped Next.js dev server" || echo "ℹ️  No Next.js dev server running"
pkill -f "next-server" 2>/dev/null && echo "✅ Stopped Next.js server" || true

# Clean up Next.js lock file
if [ -f "$PROJECT_ROOT/frontend/.next/dev/lock" ]; then
    rm -f "$PROJECT_ROOT/frontend/.next/dev/lock"
    echo "✅ Removed Next.js lock file"
fi

# Clean up log files
echo ""
echo "📝 Log files management..."
LOG_FILES=(
    "/tmp/api-gateway.log"
    "/tmp/auth-service.log"
    "/tmp/user-service.log"
    "/tmp/tenant-service.log"
    "/tmp/notification-service.log"
    "/tmp/frontend.log"
)

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
echo "✨ All services stopped successfully!"
echo ""
echo "📊 Summary:"
echo "   Stopped ports: $API_GATEWAY_PORT, $AUTH_SERVICE_PORT, $USER_SERVICE_PORT, $TENANT_SERVICE_PORT, $NOTIFICATION_SERVICE_PORT, $FRONTEND_PORT"
echo ""
echo "🚀 To start services again, run: ./scripts/start-all.sh"
echo ""
