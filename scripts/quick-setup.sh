#!/bin/bash

# Quick Development Environment Setup
# Simplified wrapper for setup-dev-environment.sh with interactive options
# Usage: ./scripts/quick-setup.sh

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
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  POS Development Environment Setup     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
}

print_menu() {
    echo ""
    echo -e "${YELLOW}What do you want to do?${NC}"
    echo ""
    echo "  1) Full setup (recommended for first time)"
    echo "     - Generate TLS certificates"
    echo "     - Create .env files"
    echo "     - Start Docker services"
    echo "     - Initialize Vault"
    echo "     - Run migrations"
    echo ""
    echo "  2) Setup without Docker (manual Docker management)"
    echo "     - Generate TLS certificates"
    echo "     - Create .env files"
    echo "     - Initialize Vault"
    echo ""
    echo "  3) Setup without Vault (for lightweight development)"
    echo "     - Generate TLS certificates"
    echo "     - Create .env files"
    echo "     - Start Docker services"
    echo ""
    echo "  4) View setup options"
    echo ""
    echo "  5) Reset everything (clean start)"
    echo ""
    echo "  6) Exit"
    echo ""
}

show_options() {
    echo -e "${BLUE}Available setup options:${NC}"
    echo ""
    echo "Full setup:"
    echo "  ./scripts/setup-dev-environment.sh"
    echo ""
    echo "Without Docker:"
    echo "  ./scripts/setup-dev-environment.sh --skip-docker"
    echo ""
    echo "Without Vault:"
    echo "  ./scripts/setup-dev-environment.sh --skip-vault"
    echo ""
    echo "Without both:"
    echo "  ./scripts/setup-dev-environment.sh --skip-docker --skip-vault"
    echo ""
}

reset_environment() {
    echo ""
    echo -e "${RED}This will reset your development environment${NC}"
    echo ""
    echo "This will:"
    echo "  1. Stop all Docker containers"
    echo "  2. Remove all volumes"
    echo "  3. Delete generated TLS certificates"
    echo "  4. Delete all .env files (keeping .env.example)"
    echo ""
    read -p "Are you sure? Type 'yes' to confirm: " confirm
    
    if [ "$confirm" != "yes" ]; then
        echo -e "${YELLOW}Reset cancelled${NC}"
        return
    fi
    
    echo ""
    echo -e "${YELLOW}Resetting environment...${NC}"
    
    # Stop and remove containers
    echo "Stopping Docker containers..."
    cd "$PROJECT_ROOT"
    docker-compose down -v 2>/dev/null || true
    
    # Remove TLS certificates
    echo "Removing TLS certificates..."
    rm -rf "$PROJECT_ROOT/vault/tls" || true
    
    # Remove .env files but keep .env.example
    echo "Removing .env files..."
    find "$PROJECT_ROOT" -name ".env" ! -name ".env.example" -delete 2>/dev/null || true
    
    # Remove Vault data
    echo "Removing Vault data..."
    rm -rf "$PROJECT_ROOT/vault/data" || true
    mkdir -p "$PROJECT_ROOT/vault/data"
    
    echo -e "${GREEN}✓ Environment reset complete${NC}"
    echo ""
    echo "Run './scripts/quick-setup.sh' again to set up from scratch"
}

main() {
    print_header
    
    while true; do
        print_menu
        read -p "Enter your choice (1-6): " choice
        
        case $choice in
            1)
                echo ""
                echo -e "${BLUE}Starting full setup...${NC}"
                bash "$SCRIPT_DIR/setup-dev-environment.sh"
                break
                ;;
            2)
                echo ""
                echo -e "${BLUE}Starting setup without Docker...${NC}"
                bash "$SCRIPT_DIR/setup-dev-environment.sh" --skip-docker
                break
                ;;
            3)
                echo ""
                echo -e "${BLUE}Starting setup without Vault...${NC}"
                bash "$SCRIPT_DIR/setup-dev-environment.sh" --skip-vault
                break
                ;;
            4)
                show_options
                ;;
            5)
                reset_environment
                ;;
            6)
                echo -e "${YELLOW}Goodbye!${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}Invalid choice. Please select 1-6.${NC}"
                ;;
        esac
    done
}

main
