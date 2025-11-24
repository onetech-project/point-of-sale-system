#!/bin/bash

# Stop All POS Services
# This script stops all backend services, frontend, and Docker containers

set -e

echo "🛑 Stopping Point of Sale System Services"
echo "=========================================="

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

# Stop Go services by port (fallback method)
echo "Stopping services by port (fallback)..."
for port in 8080 8082 8083 8084 8085; do
    PID=$(lsof -ti:$port 2>/dev/null || echo "")
    if [ -n "$PID" ]; then
        kill $PID 2>/dev/null || true
        echo "✅ Stopped service on port $port (PID: $PID)"
    fi
done

# Stop frontend
PID=$(lsof -ti:3000 2>/dev/null || echo "")
if [ -n "$PID" ]; then
    kill $PID 2>/dev/null || true
    echo "✅ Stopped frontend on port 3000 (PID: $PID)"
fi

# Clean up log files (optional)
echo ""
read -p "Remove log files? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -f /tmp/api-gateway.log /tmp/auth-service.log /tmp/user-service.log /tmp/tenant-service.log /tmp/frontend.log
    echo "✅ Log files removed"
fi

# Stop Docker services
if docker info > /dev/null 2>&1; then
    echo ""
    echo "📦 Stopping Docker services..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
    cd "$PROJECT_ROOT"
    docker-compose down
    echo "✅ Docker services stopped"
fi

echo ""
echo "✨ All services stopped successfully!"
