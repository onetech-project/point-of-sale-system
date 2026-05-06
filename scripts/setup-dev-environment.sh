#!/bin/bash

# Comprehensive Development Environment Setup Script
# This script sets up everything needed to start development:
# - Generates Vault TLS certificates
# - Creates all .env files from examples
# - Initializes Docker infrastructure
# - Sets up Vault encryption keys
# 
# Usage: ./scripts/setup-dev-environment.sh [--skip-docker] [--skip-vault]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VAULT_DIR="$PROJECT_ROOT/vault"

# Parse arguments
SKIP_DOCKER=false
SKIP_VAULT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --skip-vault)
            SKIP_VAULT=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Helper functions
print_header() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    local missing=false
    
    # Check docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        missing=true
    else
        print_success "Docker found"
    fi
    
    # Check docker-compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "docker-compose is not installed"
        missing=true
    else
        print_success "docker-compose found"
    fi
    
    # Check openssl
    if ! command -v openssl &> /dev/null; then
        print_error "openssl is not installed"
        missing=true
    else
        print_success "openssl found"
    fi
    
    if [ "$missing" = true ]; then
        print_error "Please install missing prerequisites"
        exit 1
    fi
    
    print_success "All prerequisites met"
}

# Generate Vault TLS certificates
setup_vault_tls() {
    if [ "$SKIP_VAULT" = true ]; then
        print_info "Skipping Vault TLS setup"
        return
    fi
    
    print_header "Setting up Vault TLS Certificates"
    
    TLS_DIR="$VAULT_DIR/tls"
    
    # Check if TLS certificates already exist
    if [ -f "$TLS_DIR/vault.crt" ] && [ -f "$TLS_DIR/vault.key" ] && [ -f "$TLS_DIR/ca.crt" ]; then
        print_info "Vault TLS certificates already exist"
        print_info "Remove $TLS_DIR to regenerate"
        return
    fi
    
    # Create TLS directory
    mkdir -p "$TLS_DIR"
    
    local COMMON_NAME_CA="Vault Internal CA"
    local CA_KEY="$TLS_DIR/ca.key"
    local CA_CRT="$TLS_DIR/ca.crt"
    local CSR_CONFIG="$TLS_DIR/vault_csr.cnf"
    local VAULT_KEY="$TLS_DIR/vault.key"
    local VAULT_CRT="$TLS_DIR/vault.crt"
    local VAULT_CSR="$TLS_DIR/vault.csr"
    local DAYS_VALID=825
    local BIT=2048
    
    # Generate CA key and Self-signed certificate
    print_info "Generating CA key and self-signed certificate..."
    openssl genrsa -out "$CA_KEY" $BIT 2>/dev/null
    openssl req -new -x509 -key "$CA_KEY" -out "$CA_CRT" -days $DAYS_VALID -subj "/CN=$COMMON_NAME_CA" 2>/dev/null
    print_success "CA certificate generated"
    
    # Generate Vault certificate
    print_info "Generating Vault server key and certificate signing request..."
    openssl genrsa -out "$VAULT_KEY" $BIT 2>/dev/null
    
    # Create CSR config file
    cat > "$CSR_CONFIG" <<EOF
[req]
default_bits = $BIT
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext

[dn]
CN = vault

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = vault
DNS.2 = vault.internal
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF
    
    # Generate CSR
    openssl req -new -key "$VAULT_KEY" -out "$VAULT_CSR" -config "$CSR_CONFIG" 2>/dev/null
    
    # Sign Vault certificate with CA
    openssl x509 -req -in "$VAULT_CSR" \
      -CA "$CA_CRT" -CAkey "$CA_KEY" -CAcreateserial \
      -out "$VAULT_CRT" -days $DAYS_VALID -sha256 \
      -extensions req_ext -extfile "$CSR_CONFIG" 2>/dev/null
    
    # Set proper permissions
    chmod 600 "$VAULT_KEY"
    chmod 644 "$VAULT_CRT"
    chmod 644 "$CA_CRT"
    
    # Clean up temporary files
    rm -f "$VAULT_CSR" "$CSR_CONFIG"
    
    print_success "Vault TLS certificates generated in $TLS_DIR"
}

# Setup environment files
setup_env_files() {
    print_header "Setting up Environment Files"
    
    # Root .env
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        print_info "Creating root .env file..."
        cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
        print_success "Created root .env"
    else
        print_info "Root .env already exists"
    fi
    
    # Vault .env
    if [ ! -f "$VAULT_DIR/.env" ]; then
        print_info "Creating vault .env file..."
        cp "$VAULT_DIR/.env.example" "$VAULT_DIR/.env"
        print_success "Created vault .env"
    else
        print_info "Vault .env already exists"
    fi
    
    # API Gateway .env
    if [ ! -f "$PROJECT_ROOT/api-gateway/.env" ]; then
        print_info "Creating api-gateway .env..."
        cp "$PROJECT_ROOT/api-gateway/.env.example" "$PROJECT_ROOT/api-gateway/.env"
        print_success "Created api-gateway .env"
    else
        print_info "api-gateway .env already exists"
    fi
    
    # Backend services
    local services=(
        "auth-service"
        "user-service"
        "tenant-service"
        "notification-service"
        "product-service"
        "order-service"
        "audit-service"
        "analytics-service"
        "billing-service"
    )
    
    for service in "${services[@]}"; do
        local service_dir="$PROJECT_ROOT/backend/$service"
        if [ -d "$service_dir" ]; then
            if [ ! -f "$service_dir/.env" ]; then
                if [ -f "$service_dir/.env.example" ]; then
                    print_info "Creating backend/$service .env..."
                    cp "$service_dir/.env.example" "$service_dir/.env"
                    print_success "Created backend/$service .env"
                fi
            else
                print_info "backend/$service .env already exists"
            fi
        fi
    done
    
    # Frontend .env
    if [ -d "$PROJECT_ROOT/frontend" ]; then
        if [ ! -f "$PROJECT_ROOT/frontend/.env.local" ]; then
            if [ -f "$PROJECT_ROOT/frontend/.env.example" ]; then
                print_info "Creating frontend .env.local..."
                cp "$PROJECT_ROOT/frontend/.env.example" "$PROJECT_ROOT/frontend/.env.local"
                print_success "Created frontend .env.local"
            fi
        else
            print_info "frontend .env.local already exists"
        fi
    fi
}

# Start Docker infrastructure
start_docker_infrastructure() {
    if [ "$SKIP_DOCKER" = true ]; then
        print_info "Skipping Docker infrastructure setup"
        return
    fi
    
    print_header "Starting Docker Infrastructure"
    
    cd "$PROJECT_ROOT"
    
    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        print_info "Please start Docker and try again"
        exit 1
    fi
    
    print_info "Starting Docker Compose services..."
    docker-compose up -d
    
    print_info "Waiting for services to be healthy (30 seconds)..."
    sleep 30
    
    # Check service health
    print_info "Checking service health..."
    docker-compose ps
    
    print_success "Docker infrastructure started"
}

# Initialize Vault
initialize_vault() {
    if [ "$SKIP_VAULT" = true ]; then
        print_info "Skipping Vault initialization"
        return
    fi
    
    print_header "Initializing Vault"
    
    print_info "Waiting for Vault to be ready..."
    sleep 10
    
    # Check if Vault is running
    if ! docker-compose ps vault | grep -q "Up"; then
        print_error "Vault container is not running"
        return
    fi
    
    # Run Vault init script
    print_info "Running Vault initialization script..."
    docker-compose exec -T vault sh /vault/vault-init.sh development staging production
    
    # Extract Vault token from init-keys.txt
    if [ -f "$VAULT_DIR/data/init-keys.txt" ]; then
        VAULT_TOKEN=$(grep 'Initial Root Token:' "$VAULT_DIR/data/init-keys.txt" | awk '{print $NF}')
        if [ -n "$VAULT_TOKEN" ]; then
            print_success "Vault initialized with root token"
            print_info "Vault token saved in vault/data/init-keys.txt"
        fi
    else
        print_info "Vault initialization keys file not found - Vault may already be initialized"
    fi
}

# Run database migrations
run_migrations() {
    print_header "Running Database Migrations"
    
    if [ ! -f "$SCRIPT_DIR/db-migrate.sh" ]; then
        print_info "Database migration script not found"
        return
    fi
    
    print_info "Running database migrations..."
    bash "$SCRIPT_DIR/db-migrate.sh"
    print_success "Database migrations completed"
}

# Initialize MinIO buckets
init_minio() {
    print_header "Initializing MinIO Buckets"
    
    if [ ! -f "$SCRIPT_DIR/init-minio-bucket.sh" ]; then
        print_info "MinIO initialization script not found"
        return
    fi
    
    print_info "Initializing MinIO buckets..."
    bash "$SCRIPT_DIR/init-minio-bucket.sh"
    print_success "MinIO buckets initialized"
}

# Display summary and next steps
show_summary() {
    print_header "Setup Complete!"
    
    cat <<EOF

${GREEN}Environment Setup Summary:${NC}

✓ Prerequisites verified
✓ Vault TLS certificates generated
✓ Environment files created
✓ Docker infrastructure started
✓ Vault initialized
✓ Database migrations completed

${BLUE}Next Steps:${NC}

1. Review and configure .env files if needed:
   - Root: ${PROJECT_ROOT}/.env
   - Vault: ${VAULT_DIR}/.env
   - Services: ${PROJECT_ROOT}/backend/*/env
   - API Gateway: ${PROJECT_ROOT}/api-gateway/.env

2. Save your Vault credentials (${VAULT_DIR}/data/init-keys.txt):
   - Keep the unseal keys and root token secure
   - Required for unsealing Vault after restart

3. Start all services:
   $ ./scripts/start-all.sh

4. Start specific service:
   $ ./scripts/start-all.sh auth
   $ ./scripts/start-all.sh frontend
   $ ./scripts/start-all.sh all with-vault

5. Stop all services:
   $ ./scripts/stop-all.sh

6. View Vault data:
   export VAULT_TOKEN=<root-token>
   export VAULT_ADDR=https://localhost:8200
   export VAULT_SKIP_VERIFY=true
   vault secrets list
   vault read transit/keys/pos-development-key

${YELLOW}Important Notes:${NC}

- Vault is initialized with 3 key shares, requiring 3 to unseal
- TLS certificates are self-signed (for development only)
- Unseal keys and root token are stored in: ${VAULT_DIR}/data/init-keys.txt
- Keep the init-keys.txt file safe - it's needed to unseal Vault

${BLUE}Useful Commands:${NC}

# View running services
docker-compose ps

# View logs
docker-compose logs -f postgres
docker-compose logs -f vault
docker-compose logs -f redis

# Access database
psql -h localhost -U pos_user -d pos_db

# Verify environment
./scripts/verify-env.sh

# Verify encryption
./scripts/verify-deterministic-encryption.sh

EOF
}

# Main execution
main() {
    print_header "POS Development Environment Setup"
    
    print_info "Project root: $PROJECT_ROOT"
    print_info "Vault directory: $VAULT_DIR"
    
    check_prerequisites
    setup_vault_tls
    setup_env_files
    start_docker_infrastructure
    initialize_vault
    run_migrations
    init_minio
    show_summary
}

# Run main function
main
