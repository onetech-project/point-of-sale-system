#!/bin/bash
# Dev Container Health Check Script

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🏥 Point of Sale System - Dev Container Health Check${NC}\n"

PASS=0
FAIL=0
WARN=0

# Function to check a command
check_command() {
  local cmd=$1
  local name=$2
  
  if command -v "$cmd" &> /dev/null; then
    local version=$("$cmd" --version 2>&1 | head -n1)
    echo -e "${GREEN}✅${NC} $name: $version"
    ((PASS++))
  else
    echo -e "${RED}❌${NC} $name: Not found"
    ((FAIL++))
  fi
}

# Function to check a service
check_service() {
  local host=$1
  local port=$2
  local name=$3
  
  if nc -z -w2 "$host" "$port" 2>/dev/null; then
    echo -e "${GREEN}✅${NC} $name ($host:$port): Running"
    ((PASS++))
  else
    echo -e "${RED}❌${NC} $name ($host:$port): Not responding"
    ((FAIL++))
  fi
}

# Function to check database connection
check_database() {
  if psql postgresql://pos_user:pos_password@postgres:5432/pos_db -c "SELECT version();" &>/dev/null; then
    local version=$(psql -t -c "SELECT version();" postgresql://pos_user:pos_password@postgres:5432/pos_db 2>/dev/null | head -c 30)
    echo -e "${GREEN}✅${NC} PostgreSQL: Connected ($version...)"
    ((PASS++))
  else
    echo -e "${RED}❌${NC} PostgreSQL: Connection failed"
    ((FAIL++))
  fi
}

# Function to check Redis connection
check_redis() {
  if redis-cli -h redis -a pos_password ping &>/dev/null | grep -q PONG; then
    echo -e "${GREEN}✅${NC} Redis: Connected (PONG)"
    ((PASS++))
  else
    echo -e "${RED}❌${NC} Redis: Connection failed"
    ((FAIL++))
  fi
}

echo -e "${BLUE}📦 Development Tools${NC}"
check_command "go" "Go"
check_command "node" "Node.js"
check_command "npm" "npm"
check_command "psql" "PostgreSQL Client"
check_command "redis-cli" "Redis CLI"
check_command "docker" "Docker"
check_command "git" "Git"

echo ""
echo -e "${BLUE}🗄️  Databases${NC}"
check_service "postgres" 5432 "PostgreSQL"
check_database
check_service "redis" 6379 "Redis"
check_redis

echo ""
echo -e "${BLUE}🚀 Backend Services${NC}"
check_service "localhost" 8080 "API Gateway"
check_service "localhost" 8081 "Tenant Service"
check_service "localhost" 8082 "Auth Service"
check_service "localhost" 8083 "User Service"
check_service "localhost" 8084 "Order Service"
check_service "localhost" 8085 "Product Service"
check_service "localhost" 8086 "Notification Service"

echo ""
echo -e "${BLUE}📱 Frontend & Other Services${NC}"
check_service "localhost" 3000 "Frontend"
check_service "localhost" 9000 "MinIO"
check_service "localhost" 9001 "MinIO Console"
check_service "localhost" 5555 "Mailhog"
check_service "localhost" 8200 "Vault"
check_service "kafka" 9092 "Kafka"

echo ""
echo -e "${BLUE}📂 Project Structure${NC}"
if [ -d "frontend" ]; then
  echo -e "${GREEN}✅${NC} Frontend directory found"
  ((PASS++))
else
  echo -e "${RED}❌${NC} Frontend directory not found"
  ((FAIL++))
fi

if [ -d "backend" ]; then
  echo -e "${GREEN}✅${NC} Backend directory found"
  ((PASS++))
else
  echo -e "${RED}❌${NC} Backend directory not found"
  ((FAIL++))
fi

if [ -f "docker-compose.yml" ]; then
  echo -e "${GREEN}✅${NC} docker-compose.yml found"
  ((PASS++))
else
  echo -e "${RED}❌${NC} docker-compose.yml not found"
  ((FAIL++))
fi

echo ""
echo -e "${BLUE}📊 Results${NC}"
echo -e "  ${GREEN}Passed: $PASS${NC}"
echo -e "  ${RED}Failed: $FAIL${NC}"
echo -e "  ${YELLOW}Warnings: $WARN${NC}"

echo ""
if [ $FAIL -eq 0 ]; then
  echo -e "${GREEN}✨ Dev container is healthy!${NC}"
  echo ""
  echo "Next steps:"
  echo "  1. Start backend: cd backend && ./scripts/start-all.sh"
  echo "  2. Start frontend: cd frontend && npm run dev"
  echo "  3. Run migrations: ./scripts/run-migrations.sh"
  exit 0
else
  echo -e "${RED}⚠️  Some checks failed. Please review above.${NC}"
  echo ""
  echo "Common fixes:"
  echo "  - Wait 30 seconds for services to be ready"
  echo "  - Rebuild container: Ctrl+Shift+P → 'Dev Containers: Rebuild Container'"
  echo "  - Check logs: docker logs -f <service>"
  echo "  - Restart services: docker compose -f .devcontainer/docker-compose.yml restart"
  exit 1
fi
