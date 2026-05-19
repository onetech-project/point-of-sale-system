#!/bin/bash
# POS Development Environment - Quick Reference Card
# Print this with: ./scripts/print-help.sh

cat <<'EOF'

╔════════════════════════════════════════════════════════════════════════════╗
║                  POS Development Setup - Quick Reference                  ║
║                        One Script to Start Them All!                       ║
╚════════════════════════════════════════════════════════════════════════════╝

█ FIRST TIME SETUP (New Device)
═══════════════════════════════════════════════════════════════════════════

  Run ONE command to setup everything:
  
  $ ./scripts/quick-setup.sh
  
  This will:
  ✓ Generate Vault TLS certificates
  ✓ Create all .env files
  ✓ Start Docker containers (PostgreSQL, Redis, Vault, etc.)
  ✓ Initialize Vault with encryption keys
  ✓ Run database migrations
  ✓ Initialize MinIO buckets
  
  Time: ~3-5 minutes


█ QUICK START REFERENCE
═══════════════════════════════════════════════════════════════════════════

  SETUP & INITIALIZE
  ──────────────────
  ./scripts/quick-setup.sh              Interactive setup menu
  ./scripts/setup-dev-environment.sh    Full automated setup
  ./scripts/generate-env-values.sh      Custom .env generator
  
  DEVELOPMENT WORKFLOW
  ────────────────────
  ./scripts/start-all.sh                Start all services
  ./scripts/stop-all.sh                 Stop all services
  ./scripts/dev-helper.sh status        Show environment status
  
  DAILY UTILITIES
  ───────────────
  ./scripts/dev-helper.sh logs postgres           View service logs
  ./scripts/dev-helper.sh restart vault           Restart service
  ./scripts/dev-helper.sh check-services          Health check
  ./scripts/dev-helper.sh vault-token             Show Vault credentials
  source <(./scripts/dev-helper.sh vault-login)  Set Vault CLI env


█ COMMON TASKS
═══════════════════════════════════════════════════════════════════════════

  Check everything is running
  ──────────────────────────
  $ ./scripts/dev-helper.sh status
  
  Result shows: Docker containers, .env files, TLS certs, Vault status

  Debug service issue
  ──────────────────
  $ ./scripts/dev-helper.sh logs postgres
  $ ./scripts/dev-helper.sh logs vault
  $ ./scripts/dev-helper.sh logs redis

  Restart service after config change
  ────────────────────────────────────
  $ ./scripts/dev-helper.sh restart vault

  Access Vault CLI
  ────────────────
  $ source <(./scripts/dev-helper.sh vault-login)
  $ vault status
  $ vault secrets list

  Clean up and reset
  ──────────────────
  $ ./scripts/quick-setup.sh
  Choose option 5: Reset everything

  Start specific service
  ──────────────────────
  $ ./scripts/start-all.sh auth
  $ ./scripts/start-all.sh frontend
  $ ./scripts/start-all.sh gateway


█ ENVIRONMENT FILES
═════════════════════════════════════════════════════════════════════════════

  Auto-generated in these locations:
  
  .env                              Root configuration
  vault/.env                        Vault configuration
  vault/tls/                        TLS certificates (generated)
  api-gateway/.env                  API Gateway config
  backend/*/env                     Service configs
  
  DON'T commit .env files!
  These are automatically created from .env.example


█ VAULT CREDENTIALS
═══════════════════════════════════════════════════════════════════════════

  View credentials
  ────────────────
  $ ./scripts/dev-helper.sh vault-token
  
  Output:
  Unseal Key 1: xxx
  Unseal Key 2: yyy
  Unseal Key 3: zzz
  Initial Root Token: token_xxx
  
  File location: vault/data/init-keys.txt
  
  ⚠️  IMPORTANT: Keep this file secure!
      It's needed to unseal Vault after restart.


█ TROUBLESHOOTING
═════════════════════════════════════════════════════════════════════════════

  Service not starting?
  ────────────────────
  $ ./scripts/dev-helper.sh logs <service>
  $ ./scripts/dev-helper.sh status
  $ ./scripts/dev-helper.sh check-services

  Port already in use?
  ────────────────────
  $ lsof -i :8200  # Vault
  $ lsof -i :5432  # PostgreSQL
  $ lsof -i :6379  # Redis
  $ kill -9 <PID>

  Docker container crashed?
  ──────────────────────────
  $ docker-compose ps
  $ docker-compose logs -f <service>

  Reset everything?
  ─────────────────
  $ ./scripts/dev-helper.sh clean
  $ rm -rf vault/tls vault/data
  $ ./scripts/setup-dev-environment.sh


█ DOCKER MANAGEMENT
═════════════════════════════════════════════════════════════════════════════

  Container Status
  ────────────────
  $ docker-compose ps

  View Logs
  ─────────
  $ docker-compose logs -f postgres
  $ docker-compose logs -f vault
  $ docker-compose logs postgres  # Show and exit

  Stop Everything
  ───────────────
  $ docker-compose down

  Stop and Remove Data
  ────────────────────
  $ docker-compose down -v

  Rebuild Containers
  ──────────────────
  $ docker-compose build --no-cache


█ FILE LOCATIONS
═════════════════════════════════════════════════════════════════════════════

  Setup Scripts
  $PROJECT_ROOT/scripts/
  ├── quick-setup.sh                      ← START HERE
  ├── setup-dev-environment.sh            ← Full automated setup
  ├── dev-helper.sh                       ← Daily utilities
  └── generate-env-values.sh              ← Custom config
  
  Configuration
  $PROJECT_ROOT/
  ├── .env                                ← Root config (auto-generated)
  ├── .env.example                        ← Template (don't edit)
  └── vault/
      ├── .env                            ← Vault config
      ├── tls/                            ← TLS certs (auto-generated)
      └── data/
          └── init-keys.txt               ← Vault credentials ⚠️

  Documentation
  $PROJECT_ROOT/scripts/
  ├── README.md                           ← All scripts documented
  ├── DEVELOPMENT_SETUP.md                ← Detailed setup guide
  └── print-help.sh                       ← This file


█ USEFUL COMMANDS SUMMARY
═════════════════════════════════════════════════════════════════════════════

  Status                      ./scripts/dev-helper.sh status
  Logs                        ./scripts/dev-helper.sh logs <service>
  Health Check                ./scripts/dev-helper.sh check-services
  Start Services              ./scripts/start-all.sh
  Stop Services               ./scripts/stop-all.sh
  Restart Service             ./scripts/dev-helper.sh restart <service>
  Vault Token                 ./scripts/dev-helper.sh vault-token
  Vault Login (CLI)           source <(./scripts/dev-helper.sh vault-login)
  Verify Config               ./scripts/dev-helper.sh verify-env
  Verify Encryption           ./scripts/dev-helper.sh verify-encryption
  Clean Up                    ./scripts/dev-helper.sh clean
  Full Reset                  ./scripts/quick-setup.sh (option 5)


█ ENVIRONMENT
═════════════════════════════════════════════════════════════════════════════

  Database
  ────────
  Host: localhost (from .env)
  Port: 5432 (from .env)
  User: pos_user (from .env)
  Pass: pos_password (from .env)
  
  Redis
  ─────
  Host: localhost (from .env)
  Port: 6379 (from .env)
  Pass: pos_password (from .env)
  
  Vault
  ─────
  URL: https://localhost:8200
  TLS: Self-signed (development)
  
  API Gateway
  ───────────
  URL: http://localhost:8080
  
  Frontend
  ────────
  URL: http://localhost:3000


█ SERVICES RUNNING AFTER SETUP
════════════════════════════════════════════════════════════════════════════

  Backend Infrastructure
  ──────────────────────
  ✓ PostgreSQL          (localhost:5432)
  ✓ Redis               (localhost:6379)
  ✓ Vault               (https://localhost:8200)
  ✓ Kafka               (localhost:9092)
  ✓ Zookeeper           (localhost:2181)
  ✓ MinIO               (localhost:9000)
  
  Backend Services
  ────────────────
  See docs/API.md for API documentation


█ NEXT STEPS AFTER SETUP
════════════════════════════════════════════════════════════════════════════

  1. Verify setup
     $ ./scripts/dev-helper.sh status
  
  2. Check services running
     $ ./scripts/dev-helper.sh check-services
  
  3. Start development
     $ ./scripts/start-all.sh
  
  4. View logs
     $ ./scripts/dev-helper.sh logs <service>
  
  5. Stop when done
     $ ./scripts/stop-all.sh


█ DOCUMENTATION
═════════════════════════════════════════════════════════════════════════════

  Setup & Installation
  ────────────────────
  ./scripts/README.md                 All scripts documented
  ./scripts/DEVELOPMENT_SETUP.md       Comprehensive setup guide
  
  Configuration
  ──────────────
  docs/ENVIRONMENT.md                 Environment variables
  
  Development
  ───────────
  docs/BACKEND_CONVENTIONS.md         Backend best practices
  docs/FRONTEND_CONVENTIONS.md        Frontend best practices
  docs/API.md                         API documentation
  
  Vault & Security
  ────────────────
  vault/VAULT_TOKEN_MANAGEMENT.md    Vault management


█ SUPPORT
═════════════════════════════════════════════════════════════════════════════

  Still having issues?

  1. Check logs:          ./scripts/dev-helper.sh logs <service>
  2. Verify config:       ./scripts/dev-helper.sh verify-env
  3. Check services:      ./scripts/dev-helper.sh check-services
  4. View this help:      cat ./scripts/print-help.sh
  5. Reset & retry:       ./scripts/quick-setup.sh (option 5)


════════════════════════════════════════════════════════════════════════════

Quick Start:  $ ./scripts/quick-setup.sh
Daily Work:   $ ./scripts/start-all.sh
Logs:         $ ./scripts/dev-helper.sh logs <service>
Status:       $ ./scripts/dev-helper.sh status
Reset:        $ ./scripts/quick-setup.sh (choose option 5)

════════════════════════════════════════════════════════════════════════════
Last Updated: May 2026
EOF
