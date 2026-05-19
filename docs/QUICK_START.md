# Quick Start and Deployment Guide

This guide covers the fastest way to run the Point of Sale system locally, the developer workflow, and the production deployment checklist.

## Prerequisites

Install these tools before running the project:

```bash
go version
node --version
npm --version
docker --version
docker compose version
```

Recommended versions:

- Go 1.23 or newer
- Node.js 20 LTS or newer
- Docker Engine with Docker Compose v2
- `migrate` CLI is optional because `scripts/run-migrations.sh` can use a Dockerized fallback

## Service Map

| Component | Local URL | Notes |
| --- | --- | --- |
| Frontend | http://localhost:3000 | Next.js app |
| API Gateway | http://localhost:8080 | Public API entrypoint |
| Auth Service | http://localhost:8082 | Authentication and JWT |
| User Service | http://localhost:8083 | Team and user management |
| Tenant Service | http://localhost:8084 | Multi-tenant business setup |
| Notification Service | http://localhost:8085 | Email notification workflow |
| Product Service | http://localhost:8086 | Product and inventory APIs |
| Order Service | http://localhost:8087 | Online and offline orders |
| Audit Service | http://localhost:8088 | Audit log and observability events |
| Analytics Service | http://localhost:8089 | Reporting and analytics |
| Billing Service | http://localhost:8090 | Subscriptions, invoices, and billing webhooks |
| PostgreSQL | localhost:5432 | Main relational database |
| Redis | localhost:6379 | Cache, sessions, and rate limiting |
| Kafka | localhost:9092 | Event streaming |
| MinIO | http://localhost:9001 | Object storage console |
| MailHog | http://localhost:8025 | Local email inbox |
| Vault | http://localhost:8200 | Optional encryption/secrets service |
| Grafana | http://localhost:3001 | Optional observability dashboard |

## First-Time Setup

From the repository root:

```bash
./scripts/setup-env.sh
./scripts/verify-env.sh
```

Review the generated environment files before starting services:

- `.env`
- `api-gateway/.env`
- `backend/*-service/.env`
- `frontend/.env.local`
- `vault/.env`
- `observability/.env`

At minimum, make sure these values are intentional:

- `JWT_SECRET` must match between the root `.env`, API Gateway, and Auth Service.
- Database credentials must match `docker-compose.yml`.
- `NEXT_PUBLIC_API_URL` should point to `http://localhost:8080` for local development.
- SMTP can use MailHog locally: host `localhost`, port `1025`.
- Payment, map, production SMTP, and Vault secrets should be replaced before staging or production use.
- Billing defaults include a 7-day grace period and 30-day operational-data retention window.

## Option A: Docker Infrastructure Only

The root `docker-compose.yml` currently starts infrastructure only. Use it for PostgreSQL, Redis, Kafka, MinIO, and MailHog, then run application services with `./scripts/start-all.sh all` or your own process manager.

```bash
docker network inspect pos-network >/dev/null 2>&1 || docker network create pos-network
docker compose up -d postgres redis kafka minio mailhog
./scripts/run-migrations.sh
```

Check the infrastructure stack:

```bash
docker compose ps
docker compose exec -T postgres pg_isready -U pos_user -d pos_db
```

View infrastructure logs:

```bash
docker compose logs -f postgres
docker compose logs -f kafka
```

Stop the stack:

```bash
docker compose down
```

Use `docker compose down -v` only when you intentionally want to delete local PostgreSQL, Redis, Kafka, and MinIO data.

## Option B: Local Development Mode

Use this when you want infrastructure in Docker but Go services and the frontend running as local processes.

```bash
./scripts/start-all.sh all
```

The script will:

- Load root and service-specific `.env` files
- Apply root `.env` local overrides for service ports, localhost infrastructure, and gateway service URLs
- Start PostgreSQL, Redis, Kafka, MinIO, and MailHog in Docker
- Run database migrations
- Build and start Go services locally, including Billing Service
- Start the Next.js frontend locally
- Write logs to `/tmp/*.log`
- Store process IDs in `/tmp/pos-services.pid`

Useful variants:

```bash
./scripts/start-all.sh auth user tenant
./scripts/start-all.sh all with-vault
./scripts/start-all.sh all with-observability
./scripts/start-all.sh all with-vault with-observability
```

Stop local processes:

```bash
./scripts/stop-all.sh
```

Stop Docker infrastructure as well:

```bash
docker compose down
```

## Optional Services

Start Vault manually:

```bash
docker network inspect pos-network >/dev/null 2>&1 || docker network create pos-network
docker compose -f vault/docker-compose.yml up -d
```

Start observability manually:

```bash
docker network inspect pos-network >/dev/null 2>&1 || docker network create pos-network
docker compose -f observability/docker-compose.yml up -d
```

Start the reverse proxy manually:

```bash
docker network inspect pos-network >/dev/null 2>&1 || docker network create pos-network
docker compose -f proxy/docker-compose.yml up -d
```

## Health Checks

Run these after startup:

```bash
curl http://localhost:8080/health
curl http://localhost:8082/health
curl http://localhost:8083/health
curl http://localhost:8084/health
curl http://localhost:8085/health
curl http://localhost:8086/health
curl http://localhost:8087/health
curl http://localhost:8088/health
curl http://localhost:8089/health
curl http://localhost:8090/health
```

For local-process logs:

```bash
tail -f /tmp/api-gateway.log
tail -f /tmp/auth-service.log
tail -f /tmp/order-service.log
tail -f /tmp/billing-service.log
tail -f /tmp/frontend.log
```

For container logs:

```bash
docker compose logs -f api-gateway
docker compose logs -f auth-service
docker compose logs -f order-service
```

Those application container log commands only apply when app containers are enabled in Compose or in your deployment platform.

## Database Migrations

Apply migrations:

```bash
./scripts/run-migrations.sh
```

Check the current migration version when the `migrate` CLI is installed:

```bash
migrate -path backend/migrations \
  -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
  version
```

The migration script uses root `.env` database settings when available. By default it targets:

- host: `localhost`
- port: `5432`
- database: `pos_db`
- user: `pos_user`

## Production Deployment Checklist

Use the same deployment order for staging and production:

1. Prepare environment files or platform secrets.
2. Build service images.
3. Start infrastructure services.
4. Run database migrations.
5. Start backend services.
6. Start API Gateway and frontend.
7. Enable proxy, TLS, Vault, and observability.
8. Run health checks and smoke tests.

Example local deployment bootstrap:

```bash
./scripts/verify-env.sh
docker network inspect pos-network >/dev/null 2>&1 || docker network create pos-network
docker compose up -d postgres redis kafka minio mailhog
./scripts/run-migrations.sh
./scripts/start-all.sh all
docker compose -f observability/docker-compose.yml up -d
docker compose -f proxy/docker-compose.yml up -d
```

Before production release, verify:

- Default passwords and development secrets are replaced.
- `JWT_SECRET` is long, random, and consistent across services that validate tokens.
- PostgreSQL has backups, retention, and restore testing.
- Vault is initialized, unsealed, persisted, and not using development tokens.
- Kafka data is persisted and topic retention matches audit/order requirements.
- MinIO or S3 buckets, credentials, and retention policies are configured.
- SMTP credentials and sender domains are production-ready.
- Payment provider keys and webhook URLs are production-ready.
- Subscription retention policy and Terms of Service version are reviewed before launch.
- CORS and frontend public URLs point to production domains.
- TLS is enabled at the proxy/load balancer.
- Grafana, Prometheus, Loki, Tempo, and OpenTelemetry endpoints are reachable.
- Logs do not expose credentials, tokens, or encrypted PII plaintext.

## Smoke Test

After deployment:

```bash
curl http://localhost:8080/health
curl http://localhost:3000
docker compose ps
```

Then test the main workflow in the browser:

1. Register or sign in to a tenant.
2. Create a team member.
3. Create a product and adjust stock.
4. Create an online order.
5. Create an offline order.
6. Verify email output in MailHog or the configured SMTP provider.
7. Check audit logs and service metrics.

## Troubleshooting

Port already in use:

```bash
lsof -i :8080
lsof -i :3000
./scripts/stop-all.sh
docker compose down
```

PostgreSQL is not ready:

```bash
docker compose ps postgres
docker compose logs -f postgres
docker compose exec -T postgres pg_isready -U pos_user -d pos_db
```

Migrations fail:

```bash
./scripts/run-migrations.sh
docker compose logs -f postgres
```

Frontend cannot reach the API:

```bash
grep NEXT_PUBLIC_API_URL frontend/.env.local
curl http://localhost:8080/health
```

JWT validation fails:

```bash
grep JWT_SECRET .env api-gateway/.env backend/auth-service/.env
```

Kafka issues:

```bash
docker compose ps kafka
docker compose logs -f kafka
```

Vault or observability cannot start:

```bash
docker network inspect pos-network >/dev/null 2>&1 || docker network create pos-network
docker compose -f vault/docker-compose.yml logs -f
docker compose -f observability/docker-compose.yml logs -f
```
