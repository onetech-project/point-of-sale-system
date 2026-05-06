#!/bin/bash

# Development Environment Helper
# Provides utility commands for development environment management
# Usage: ./scripts/dev-helper.sh <command> [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VAULT_DIR="$PROJECT_ROOT/vault"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }
print_info() { echo -e "${YELLOW}ℹ $1${NC}"; }
print_header() { echo -e "\n${BLUE}=== $1 ===${NC}\n"; }

# Commands
cmd_status() {
    print_header "Development Environment Status"
    
    echo "Docker Containers:"
    docker-compose ps
    
    echo ""
    echo "Environment Files:"
    find "$PROJECT_ROOT" -maxdepth 3 -name ".env" ! -name ".env.example" | sort
    
    if [ -f "$VAULT_DIR/tls/vault.crt" ]; then
        print_success "Vault TLS certificates exist"
    else
        print_error "Vault TLS certificates not found"
    fi
    
    if [ -f "$VAULT_DIR/data/init-keys.txt" ]; then
        print_success "Vault initialized"
    else
        print_info "Vault not initialized yet"
    fi
}

cmd_show_vault_token() {
    print_header "Vault Credentials"
    
    if [ -f "$VAULT_DIR/data/init-keys.txt" ]; then
        echo "File: $VAULT_DIR/data/init-keys.txt"
        echo ""
        cat "$VAULT_DIR/data/init-keys.txt"
    else
        print_error "Vault init-keys.txt not found"
        print_info "Initialize Vault first: ./scripts/quick-setup.sh"
        exit 1
    fi
}

cmd_vault_login() {
    print_header "Vault Login"
    
    if [ ! -f "$VAULT_DIR/data/init-keys.txt" ]; then
        print_error "Vault init-keys.txt not found"
        exit 1
    fi
    
    VAULT_TOKEN=$(grep 'Initial Root Token:' "$VAULT_DIR/data/init-keys.txt" | awk '{print $NF}')
    export VAULT_ADDR="https://localhost:8200"
    export VAULT_TOKEN="$VAULT_TOKEN"
    export VAULT_SKIP_VERIFY=true
    
    print_info "Vault environment variables set:"
    echo "  VAULT_ADDR=$VAULT_ADDR"
    echo "  VAULT_TOKEN=$VAULT_TOKEN (root)"
    echo "  VAULT_SKIP_VERIFY=$VAULT_SKIP_VERIFY"
    echo ""
    print_info "Run: vault status"
}

cmd_check_services() {
    print_header "Checking Service Health"
    
    local services=("postgres" "redis" "vault" "kafka" "zookeeper")
    
    for service in "${services[@]}"; do
        if docker-compose ps "$service" 2>/dev/null | grep -q "Up"; then
            print_success "$service is running"
        else
            print_error "$service is not running"
        fi
    done
}

cmd_logs() {
    local service="$1"
    
    if [ -z "$service" ]; then
        print_error "Please specify a service"
        echo "Available services:"
        docker-compose ps --services
        exit 1
    fi
    
    print_header "Logs for $service"
    docker-compose logs -f "$service"
}

cmd_restart_service() {
    local service="$1"
    
    if [ -z "$service" ]; then
        print_error "Please specify a service"
        exit 1
    fi
    
    print_info "Restarting $service..."
    docker-compose restart "$service"
    print_success "$service restarted"
}

cmd_rebuild() {
    print_header "Rebuilding Containers"
    
    cd "$PROJECT_ROOT"
    docker-compose down
    docker-compose build --no-cache
    docker-compose up -d
    
    print_success "Containers rebuilt and started"
}

cmd_clean() {
    print_header "Cleaning Up Development Environment"
    
    echo "This will:"
    echo "  1. Stop all containers"
    echo "  2. Remove all volumes (databases, etc.)"
    echo "  3. Keep .env files and TLS certificates"
    echo ""
    read -p "Continue? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        print_info "Cancelled"
        return
    fi
    
    print_info "Stopping services..."
    cd "$PROJECT_ROOT"
    docker-compose down -v
    
    print_success "Cleanup complete"
}

cmd_verify_encryption() {
    print_header "Verifying Encryption Setup"
    
    if [ ! -f "$SCRIPT_DIR/verify-deterministic-encryption.sh" ]; then
        print_error "Encryption verification script not found"
        exit 1
    fi
    
    bash "$SCRIPT_DIR/verify-deterministic-encryption.sh"
}

cmd_verify_env() {
    print_header "Verifying Environment Configuration"
    
    if [ ! -f "$SCRIPT_DIR/verify-env.sh" ]; then
        print_error "Environment verification script not found"
        exit 1
    fi
    
    bash "$SCRIPT_DIR/verify-env.sh"
}

cmd_help() {
    cat <<EOF
POS Development Helper

Usage: ./scripts/dev-helper.sh <command> [options]

Commands:

  status                  Show status of development environment
  vault-token             Display Vault credentials (token, keys, etc.)
  vault-login             Set environment variables for Vault CLI access
  check-services          Check health of all services
  logs <service>          View logs for a service (e.g., postgres, redis)
  restart <service>       Restart a specific service
  rebuild                 Rebuild all containers
  clean                   Stop containers and remove volumes
  verify-encryption       Verify encryption setup
  verify-env              Verify environment configuration
  help                    Show this help message

Examples:

  ./scripts/dev-helper.sh status
  ./scripts/dev-helper.sh logs postgres
  ./scripts/dev-helper.sh restart vault
  ./scripts/dev-helper.sh verify-encryption
  source <(./scripts/dev-helper.sh vault-login)

EOF
}

main() {
    local command="$1"
    
    case "$command" in
        status)
            cmd_status
            ;;
        vault-token)
            cmd_show_vault_token
            ;;
        vault-login)
            cmd_vault_login
            ;;
        check-services|check)
            cmd_check_services
            ;;
        logs)
            cmd_logs "$2"
            ;;
        restart)
            cmd_restart_service "$2"
            ;;
        rebuild)
            cmd_rebuild
            ;;
        clean)
            cmd_clean
            ;;
        verify-encryption)
            cmd_verify_encryption
            ;;
        verify-env)
            cmd_verify_env
            ;;
        help|-h|--help)
            cmd_help
            ;;
        *)
            if [ -z "$command" ]; then
                cmd_help
            else
                print_error "Unknown command: $command"
                cmd_help
                exit 1
            fi
            ;;
    esac
}

cd "$PROJECT_ROOT"
main "$@"
