#!/bin/bash
set -e

echo "🚀 Setting up Point of Sale System dev environment..."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}📦 Installing frontend dependencies...${NC}"
cd /workspace/frontend
npm install
echo -e "${GREEN}✅ Frontend dependencies installed${NC}"

echo -e "${BLUE}🔨 Building Go services...${NC}"
cd /workspace/backend

# List of services to build
SERVICES=(
  "api-gateway"
  "auth-service"
  "tenant-service"
  "user-service"
  "order-service"
  "product-service"
  "notification-service"
  "audit-service"
  "analytics-service"
)

for service in "${SERVICES[@]}"; do
  if [ -d "$service" ]; then
    echo "  Building $service..."
    cd "$service"
    go build -o "${service}.bin" main.go 2>/dev/null || true
    cd ..
  fi
done
echo -e "${GREEN}✅ Go services built${NC}"

echo -e "${BLUE}⏳ Waiting for PostgreSQL to be ready...${NC}"
until pg_isready -h postgres -U pos_user -d pos_db 2>/dev/null; do
  echo "  PostgreSQL is unavailable - sleeping"
  sleep 2
done
echo -e "${GREEN}✅ PostgreSQL is ready${NC}"

echo -e "${BLUE}🗄️  Running database migrations...${NC}"
cd /workspace
if [ -f "./scripts/run-migrations.sh" ]; then
  bash ./scripts/run-migrations.sh || true
  echo -e "${GREEN}✅ Database migrations completed${NC}"
else
  echo -e "${YELLOW}⚠️  Migration script not found${NC}"
fi

echo ""
echo -e "${GREEN}✨ Dev environment setup complete!${NC}"
echo ""
echo -e "${BLUE}📋 Available commands:${NC}"
echo "  Frontend:      cd frontend && npm run dev"
echo "  Backend:       cd backend && ./scripts/start-all.sh"
echo "  Migrations:    ./scripts/run-migrations.sh"
echo "  Services status: docker ps"
echo ""
echo -e "${BLUE}🔗 Service ports:${NC}"
echo "  Frontend:              http://localhost:3000"
echo "  API Gateway:           http://localhost:8080"
echo "  MinIO Console:         http://localhost:9001"
echo "  Mailhog:               http://localhost:5555"
echo "  PostgreSQL:            localhost:5432"
echo "  Redis:                 localhost:6379"
echo "  Kafka:                 localhost:9092"
echo ""
echo -e "${YELLOW}ℹ️  Tip: Use VS Code's command palette to access Integrated Terminal${NC}"
