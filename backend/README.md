# POS System Backend

**Technology Stack**: Go (Golang) 1.21+  
**Architecture**: Microservices with event-driven communication  
**Database**: PostgreSQL 15+  
**Message Queue**: Apache Kafka  
**Secrets Management**: HashiCorp Vault  
**Caching**: Redis

---

## Table of Contents

1. [Services Overview](#services-overview)
2. [Quick Start](#quick-start)
3. [UU PDP Compliance Setup](#uu-pdp-compliance-setup)
4. [Development Workflow](#development-workflow)
5. [Testing](#testing)
6. [Deployment](#deployment)
7. [Monitoring](#monitoring)
8. [Troubleshooting](#troubleshooting)

---

## Services Overview

### Microservices

| Service                | Port | Description                    | Key Features                                      |
| ---------------------- | ---- | ------------------------------ | ------------------------------------------------- |
| `api-gateway`          | 8080 | API Gateway & Routing          | Rate limiting, JWT validation, request logging    |
| `auth-service`         | 8081 | Authentication & Authorization | Login, registration, session management           |
| `user-service`         | 8082 | User Management                | CRUD operations, encryption, data cleanup         |
| `tenant-service`       | 8083 | Tenant Management              | Multi-tenancy, tenant configuration               |
| `product-service`      | 8084 | Product Catalog                | Inventory management, pricing                     |
| `order-service`        | 8085 | Order Processing               | Order creation, guest orders, payment integration |
| `notification-service` | 8086 | Email Notifications            | SMTP integration, template rendering, retry logic |
| `audit-service`        | 8087 | Audit Logging                  | Immutable event storage, partition management     |

### Supporting Infrastructure

- **PostgreSQL**: Primary database for all services
- **Kafka**: Event streaming between services
- **Redis**: Distributed locking, caching, session storage
- **Vault**: Encryption key management (Transit Engine)
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Observability dashboards

---

## Quick Start

### Prerequisites

- **Go**: 1.21 or higher
- **Docker & Docker Compose**: Latest version
- **Make**: Build automation
- **PostgreSQL**: 15+ (or use Docker Compose)
- **Redis**: 7+ (or use Docker Compose)

### Initial Setup

```bash
# 1. Clone repository
git clone <repository-url>
cd point-of-sale-system/backend

# 2. Install dependencies for all services
make deps

# 3. Start infrastructure (PostgreSQL, Kafka, Redis, Vault)
docker-compose up -d postgres kafka redis vault

# 4. Initialize database
make migrate-up

# 5. Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# 6. Start all services
make dev
```

### Verify Installation

```bash
# Check service health
curl http://localhost:8080/health  # API Gateway
curl http://localhost:8081/health  # Auth Service
curl http://localhost:8082/health  # User Service

# Check database connection
psql -U pos_user -d pos_db -c "SELECT 1;"

# Check Vault status
docker exec -it vault vault status
```

---

## UU PDP Compliance Setup

The system implements Indonesian Personal Data Protection Law (UU PDP No.27/2022) compliance features:

- **Encryption at Rest**: All PII encrypted with AES-256-GCM via Vault Transit Engine
- **Consent Management**: Purpose-based consent collection and revocation
- **Audit Trail**: Immutable logging of all data access (7-year retention)
- **Data Rights**: User data access, export, and deletion capabilities
- **Data Retention**: Automated cleanup with configurable retention policies
- **Privacy Policy**: Bilingual policy (Indonesian + English)

### Quick Start Guide

For complete setup instructions, see: **[/docs/quickstart.md](../docs/quickstart.md)**

**30-Minute Setup**:

1. Start Vault and initialize Transit Engine
2. Generate encryption keys
3. Run database migrations (including retention policies)
4. Configure SMTP for deletion notifications
5. Verify compliance features

### Key Environment Variables

```bash
# Vault Configuration
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=your_vault_root_token
VAULT_TRANSIT_KEY_NAME=pos-keys

# Encryption (Development only - use Vault in production)
ENCRYPTION_KEY_PATH=./encryption_keys/master.key

# SMTP (for deletion notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
SMTP_FROM=noreply@your-domain.com

# Redis (for distributed locking in cleanup jobs)
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Testing Compliance Features

```bash
# 1. Test encryption
cd user-service
go test ./src/utils -run TestEncryption -v

# 2. Test consent validation
cd auth-service
go test ./src/handlers -run TestConsentRequired -v

# 3. Test audit logging
cd audit-service
go test ./src/repository -run TestAuditImmutability -v

# 4. Test data cleanup
cd user-service
go test ./src/jobs -run TestCleanupOrchestrator -v

# 5. Run compliance verification script
cd ../../scripts
./verify-uu-pdp-compliance.sh
```

### Compliance Documentation

- **Feature Guide**: [/docs/UU_PDP_COMPLIANCE.md](../docs/UU_PDP_COMPLIANCE.md)
- **API Documentation**: [/docs/API.md](../docs/API.md) (see "UU PDP Compliance API" section)
- **Runbooks**: [/docs/RUNBOOKS.md](../docs/RUNBOOKS.md) (operational procedures)

---

## Development Workflow

### Project Structure

```
backend/
├── api-gateway/          # API Gateway service
│   ├── middleware/       # Rate limiting, auth, logging
│   ├── observability/    # Metrics, tracing
│   └── main.go
├── auth-service/         # Authentication service
│   └── src/
│       ├── handlers/     # HTTP handlers
│       ├── services/     # Business logic
│       ├── models/       # Data models
│       └── utils/        # Encryption, validation
├── user-service/         # User management service
│   └── src/
│       ├── jobs/         # Background jobs (cleanup, notifications)
│       ├── scheduler/    # Job scheduling
│       └── ...
├── migrations/           # Database migrations (shared)
│   ├── 000001_initial_schema.up.sql
│   ├── 000055_create_retention_policies.up.sql
│   └── ...
└── docker-compose.yml    # Infrastructure setup
```

### Adding a New Service

1. **Create service directory**:

```bash
mkdir -p new-service/src/{handlers,services,models,repository}
cd new-service
go mod init github.com/pos/new-service
```

2. **Add dependencies**:

```bash
go get github.com/gorilla/mux
go get github.com/lib/pq
go get github.com/rs/zerolog
```

3. **Create main.go**:

```go
package main

import (
    "github.com/gorilla/mux"
    "github.com/rs/zerolog/log"
)

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/health", HealthCheck).Methods("GET")

    log.Info().Msg("New service starting on :8088")
    http.ListenAndServe(":8088", router)
}
```

4. **Update docker-compose.yml**:

```yaml
new-service:
  build: ./new-service
  ports:
    - '8088:8088'
  environment:
    - DATABASE_URL=postgresql://pos_user:password@postgres:5432/pos_db
  depends_on:
    - postgres
```

### Database Migrations

Using [golang-migrate](https://github.com/golang-migrate/migrate):

```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_feature

# Apply migrations
migrate -path migrations -database "postgresql://pos_user:password@localhost:5432/pos_db?sslmode=disable" up

# Rollback last migration
migrate -path migrations -database "postgresql://..." down 1

# Check migration version
migrate -path migrations -database "postgresql://..." version
```

### Code Conventions

See: **[/docs/BACKEND_CONVENTIONS.md](../docs/BACKEND_CONVENTIONS.md)**

**Key Principles**:

- Use standard `database/sql` (not sqlx) for consistency
- Encrypt all PII fields using `EncryptionService`
- Publish audit events to Kafka for all data modifications
- Use structured logging with `zerolog`
- Follow repository pattern for database access
- Use context for cancellation and timeouts

---

## Testing

### Unit Tests

```bash
# Test specific service
cd user-service
go test ./... -v

# Test with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Test specific package
go test ./src/services -v
```

### Integration Tests

```bash
# Start test database
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test ./tests/integration -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Benchmarks

```bash
# Run encryption benchmarks
cd user-service
go test ./src/utils -bench=BenchmarkEncrypt -benchmem

# Expected results:
# BenchmarkEncrypt-8    10000    2500 ns/op    512 B/op    5 allocs/op
# BenchmarkDecrypt-8    10000    2300 ns/op    480 B/op    4 allocs/op
```

---

## Deployment

### Production Checklist

- [ ] Environment variables configured (no defaults)
- [ ] Vault Transit Engine initialized and unsealed
- [ ] Database migrations applied
- [ ] SMTP credentials configured for notifications
- [ ] Redis cluster set up for distributed locking
- [ ] Prometheus alerts configured
- [ ] Backup schedule configured (daily database backups)
- [ ] SSL/TLS certificates installed
- [ ] Rate limiting tuned for expected load

### Docker Build

```bash
# Build all services
docker-compose build

# Build specific service
docker-compose build user-service

# Push to registry
docker tag user-service:latest registry.example.com/user-service:v1.0.0
docker push registry.example.com/user-service:v1.0.0
```

### Kubernetes Deployment

```bash
# Apply configurations
kubectl apply -f k8s/namespace.yml
kubectl apply -f k8s/secrets.yml
kubectl apply -f k8s/user-service-deployment.yml

# Check rollout status
kubectl rollout status deployment/user-service -n pos

# View logs
kubectl logs -f deployment/user-service -n pos
```

---

## Monitoring

### Prometheus Metrics

Each service exposes metrics at `/metrics`:

**Common Metrics**:

- `http_requests_total{method, endpoint, status}` - HTTP request count
- `http_request_duration_seconds{method, endpoint}` - Request duration
- `db_connections_active` - Active database connections
- `encryption_operations_total{operation}` - Encryption/decryption count
- `cleanup_records_processed_total{table}` - Data cleanup metrics

**Service-Specific Metrics**:

- `notification_email_sent_total` - Successful email deliveries
- `audit_events_persisted_total` - Audit events stored
- `cleanup_duration_seconds{table}` - Cleanup job duration

### Grafana Dashboards

Import pre-built dashboards from `observability/grafana/dashboards/`:

- `pos_system_overview.json` - System-wide metrics
- `encryption_performance.json` - Encryption operations
- `data_cleanup_status.json` - Retention policy compliance

### Health Checks

```bash
# Quick health check script
./scripts/health-check.sh

# Or check individual services
curl http://localhost:8080/health  # Returns: {"status": "healthy"}
```

---

## Troubleshooting

### Common Issues

#### 1. Service Cannot Connect to Database

**Error**: `pq: password authentication failed`

**Solution**:

```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Verify credentials
psql -U pos_user -d pos_db -h localhost -p 5432

# Update .env file with correct credentials
DATABASE_URL=postgresql://pos_user:correct_password@localhost:5432/pos_db
```

#### 2. Encryption Failures

**Error**: `Failed to encrypt field: connection refused`

**Solution**:

```bash
# Check Vault is running and unsealed
vault status

# If sealed, unseal with key shares
vault operator unseal <key_share_1>
vault operator unseal <key_share_2>
vault operator unseal <key_share_3>

# Verify Transit Engine mounted
vault secrets list | grep transit
```

#### 3. Kafka Connection Issues

**Error**: `kafka: client has run out of available brokers`

**Solution**:

```bash
# Check Kafka is running
docker ps | grep kafka

# Test connectivity
kafka-topics --bootstrap-server localhost:9092 --list

# Restart Kafka if needed
docker-compose restart kafka
```

#### 4. Cleanup Jobs Not Running

**Error**: `CleanupJobsStalled` alert firing

**Solution**:

```bash
# Check Redis locks
redis-cli KEYS "cleanup:lock:*"

# Release stuck lock
redis-cli DEL "cleanup:lock:users:deleted"

# Manually trigger cleanup
curl -X POST http://localhost:8082/admin/cleanup/run-now
```

### Logs

```bash
# View logs for specific service
docker logs user-service --tail 100 -f

# Search logs for errors
docker logs user-service | grep -i error

# Export logs for analysis
docker logs user-service > /tmp/user-service.log
```

### Debug Mode

Enable debug logging in `.env`:

```bash
LOG_LEVEL=debug  # Options: debug, info, warn, error
```

---

## Additional Resources

- **API Documentation**: [/docs/API.md](../docs/API.md)
- **Backend Conventions**: [/docs/BACKEND_CONVENTIONS.md](../docs/BACKEND_CONVENTIONS.md)
- **Quick Start Guide**: [/docs/quickstart.md](../docs/quickstart.md)
- **Runbooks**: [/docs/RUNBOOKS.md](../docs/RUNBOOKS.md)
- **UU PDP Compliance**: [/docs/UU_PDP_COMPLIANCE.md](../docs/UU_PDP_COMPLIANCE.md)

---

## Contributing

1. Follow code conventions ([BACKEND_CONVENTIONS.md](../docs/BACKEND_CONVENTIONS.md))
2. Write tests for new features
3. Ensure all tests pass: `make test`
4. Update documentation for API changes
5. Submit pull request with clear description

---

## License

[License Type] - See LICENSE file for details

---

**Questions?** Contact the development team or create an issue in the repository.
