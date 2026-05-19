#!/bin/bash

# Generate .env Files with Customizable Values
# Allows users to create .env files with specific values
# Usage: ./scripts/generate-env-values.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "\n${BLUE}╔════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  Generate .env Files with Values  ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════╝${NC}\n"
}

print_info() { echo -e "${YELLOW}ℹ $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }

# Prompt for value with default
prompt_value() {
    local prompt_text="$1"
    local default_value="$2"
    local input_value
    
    if [ -z "$default_value" ]; then
        read -p "$prompt_text: " input_value
    else
        read -p "$prompt_text [$default_value]: " input_value
        input_value="${input_value:-$default_value}"
    fi
    
    echo "$input_value"
}

generate_root_env() {
    echo ""
    print_info "Configuring Root Environment (.env)"
    
    local db_host=$(prompt_value "Database Host" "localhost")
    local db_port=$(prompt_value "Database Port" "5432")
    local db_user=$(prompt_value "Database User" "pos_user")
    local db_password=$(prompt_value "Database Password" "pos_password")
    local db_name=$(prompt_value "Database Name" "pos_db")
    
    local redis_host=$(prompt_value "Redis Host" "localhost")
    local redis_port=$(prompt_value "Redis Port" "6379")
    local redis_password=$(prompt_value "Redis Password" "pos_password")
    
    local kafka_broker=$(prompt_value "Kafka Broker" "localhost:9092")
    
    local jwt_secret=$(prompt_value "JWT Secret (press Enter to generate)" "")
    if [ -z "$jwt_secret" ]; then
        jwt_secret=$(openssl rand -base64 32)
    fi
    
    local api_gateway_port=$(prompt_value "API Gateway Port" "8080")
    local frontend_port=$(prompt_value "Frontend Port" "3000")
    local frontend_url=$(prompt_value "Frontend URL" "http://localhost:$frontend_port")
    local api_url=$(prompt_value "API URL" "http://localhost:$api_gateway_port")
    
    local environment=$(prompt_value "Environment" "development")
    local log_level=$(prompt_value "Log Level" "debug")
    
    # Create .env file
    cat > "$PROJECT_ROOT/.env" <<EOF
# Root Environment Configuration
# Generated: $(date)

# Database Configuration
POSTGRES_DB=$db_name
POSTGRES_USER=$db_user
POSTGRES_PASSWORD=$db_password
POSTGRES_HOST=$db_host
POSTGRES_PORT=$db_port

# Redis Configuration
REDIS_HOST=$redis_host
REDIS_PORT=$redis_port
REDIS_PASSWORD=$redis_password

# Kafka Configuration
KAFKA_BROKER=$kafka_broker
ZOOKEEPER_HOST=localhost:2181

# JWT Configuration (shared across all services)
JWT_SECRET=$jwt_secret

# Service Ports
API_GATEWAY_PORT=$api_gateway_port
AUTH_SERVICE_PORT=8082
USER_SERVICE_PORT=8083
TENANT_SERVICE_PORT=8084
NOTIFICATION_SERVICE_PORT=8085
PRODUCT_SERVICE_PORT=8086
ORDER_SERVICE_PORT=8087
AUDIT_SERVICE_PORT=8088
ANALYTICS_SERVICE_PORT=8089
BILLING_SERVICE_PORT=8090
BILLING_SERVICE_URL=http://localhost:8090

# Billing plan defaults
PLAN_MONTHLY_PRICE_IDR=299000
PLAN_ANNUAL_DISCOUNT_PCT=20
PLAN_TRIAL_DAYS=7
PLAN_GRACE_PERIOD_DAYS=7

# Frontend
FRONTEND_PORT=$frontend_port
NEXT_PUBLIC_API_URL=$api_url

# Environment
ENVIRONMENT=$environment
LOG_LEVEL=$log_level

# Email Configuration (shared)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=
SMTP_FROM=noreply@example.com

# App URLs
APP_URL=$frontend_url
API_URL=$api_url
EOF
    
    print_success "Root .env created"
}

generate_vault_env() {
    echo ""
    print_info "Configuring Vault (.env)"
    
    # Check if Vault environment already exists
    if [ -f "$PROJECT_ROOT/vault/.env" ]; then
        read -p "Vault .env already exists. Overwrite? (y/n): " overwrite
        if [ "$overwrite" != "y" ]; then
            print_info "Skipped vault .env generation"
            return
        fi
    fi
    
    local hcl_config=$(prompt_value "HCL Config File" "vault-config-raft-storage.hcl")
    local vault_env=$(prompt_value "Vault Environment" "development")
    
    cat > "$PROJECT_ROOT/vault/.env" <<EOF
# Vault Configuration
# Generated: $(date)

HCL_CONFIG=./$hcl_config
VAULT_ENV=$vault_env
EOF
    
    print_success "Vault .env created"
}

generate_service_env() {
    local service_name="$1"
    local service_path="$2"
    
    # Check if .env.example exists
    if [ ! -f "$service_path/.env.example" ]; then
        print_info "Skipping $service_name (no .env.example found)"
        return
    fi
    
    # Copy from example if .env doesn't exist
    if [ ! -f "$service_path/.env" ]; then
        cp "$service_path/.env.example" "$service_path/.env"
        print_success "Created $service_name .env from example"
    else
        print_info "$service_name .env already exists"
    fi
}

main() {
    print_header
    
    echo "This script will generate .env files with custom values."
    echo ""
    echo "Available options:"
    echo "  1) Generate all .env files with custom values"
    echo "  2) Only generate root .env (advanced)"
    echo "  3) Only generate vault .env (advanced)"
    echo "  4) Use defaults for all (quick)"
    echo ""
    
    read -p "Choose option (1-4) [1]: " option
    option="${option:-1}"
    
    case $option in
        1)
            echo ""
            echo "=== Custom .env Generation ==="
            echo ""
            echo "Answer the prompts below. Press Enter to use defaults."
            echo ""
            
            generate_root_env
            generate_vault_env
            
            # Generate backend services using examples
            echo ""
            print_info "Generating backend service .env files..."
            local services=(
                "auth-service:$PROJECT_ROOT/backend/auth-service"
                "user-service:$PROJECT_ROOT/backend/user-service"
                "tenant-service:$PROJECT_ROOT/backend/tenant-service"
                "notification-service:$PROJECT_ROOT/backend/notification-service"
                "product-service:$PROJECT_ROOT/backend/product-service"
                "order-service:$PROJECT_ROOT/backend/order-service"
                "audit-service:$PROJECT_ROOT/backend/audit-service"
                "analytics-service:$PROJECT_ROOT/backend/analytics-service"
                "billing-service:$PROJECT_ROOT/backend/billing-service"
            )
            
            for service_info in "${services[@]}"; do
                IFS=':' read -r service_name service_path <<< "$service_info"
                generate_service_env "$service_name" "$service_path"
            done
            
            # API Gateway
            if [ -d "$PROJECT_ROOT/api-gateway" ]; then
                generate_service_env "api-gateway" "$PROJECT_ROOT/api-gateway"
            fi
            
            echo ""
            print_success "All .env files generated!"
            echo ""
            echo "Next steps:"
            echo "  1. Review the generated .env files"
            echo "  2. Run: ./scripts/setup-dev-environment.sh"
            ;;
            
        2)
            generate_root_env
            print_success "Root .env generated"
            ;;
            
        3)
            generate_vault_env
            print_success "Vault .env generated"
            ;;
            
        4)
            echo ""
            print_info "Using default values for all .env files..."
            
            # Root
            cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env" 2>/dev/null || true
            print_success "Root .env created from example"
            
            # Vault
            cp "$PROJECT_ROOT/vault/.env.example" "$PROJECT_ROOT/vault/.env" 2>/dev/null || true
            print_success "Vault .env created from example"
            
            # Services
            local services=(
                "$PROJECT_ROOT/api-gateway"
                "$PROJECT_ROOT/backend/auth-service"
                "$PROJECT_ROOT/backend/user-service"
                "$PROJECT_ROOT/backend/tenant-service"
                "$PROJECT_ROOT/backend/notification-service"
                "$PROJECT_ROOT/backend/product-service"
                "$PROJECT_ROOT/backend/order-service"
                "$PROJECT_ROOT/backend/audit-service"
                "$PROJECT_ROOT/backend/analytics-service"
                "$PROJECT_ROOT/backend/billing-service"
            )
            
            for service_path in "${services[@]}"; do
                if [ -f "$service_path/.env.example" ] && [ ! -f "$service_path/.env" ]; then
                    cp "$service_path/.env.example" "$service_path/.env"
                    echo "✓ Created $(basename $service_path)/.env"
                fi
            done
            
            print_success "All .env files created from defaults"
            ;;
            
        *)
            echo "Invalid option"
            exit 1
            ;;
    esac
}

main
