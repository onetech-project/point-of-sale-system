# Start/Stop Scripts Usage Guide

This guide explains how to use the enhanced `start-all.sh` and `stop-all.sh` scripts for managing POS system services.

## Quick Start

```bash
# Start all services (default)
./scripts/start-all.sh

# Stop all services (default)
./scripts/stop-all.sh
```

## Service Names

You can refer to services using short or full names:

| Short Name | Full Name | Service |
|------------|-----------|---------|
| `gateway` | `api-gateway` | API Gateway |
| `auth` | `auth-service` | Auth Service |
| `user` | `user-service` | User Service |
| `tenant` | `tenant-service` | Tenant Service |
| `notification` | `notification-service` | Notification Service |
| `frontend` | `web` | Frontend (Next.js) |
| `all` | - | All services |

## Start Services

### Start All Services
```bash
./scripts/start-all.sh
# or
./scripts/start-all.sh all
```

### Start Specific Services
```bash
# Single service
./scripts/start-all.sh auth

# Multiple services
./scripts/start-all.sh auth user tenant

# Gateway and one backend
./scripts/start-all.sh gateway user
```

### Start Only Frontend
```bash
./scripts/start-all.sh frontend
```
**Use Case**: Frontend development with backend on remote server

### Start Backend Services Only
```bash
./scripts/start-all.sh gateway auth user tenant notification
```
**Use Case**: API testing without frontend

## Stop Services

### Stop All Services
```bash
./scripts/stop-all.sh
# or
./scripts/stop-all.sh all
```

### Stop Specific Services
```bash
# Single service
./scripts/stop-all.sh auth

# Multiple services
./scripts/stop-all.sh auth user

# Stop frontend only
./scripts/stop-all.sh frontend
```

## Common Workflows

### Frontend Development
```bash
# Start only frontend for UI work
./scripts/start-all.sh frontend

# When done
./scripts/stop-all.sh frontend
```

### Backend Service Development
```bash
# Working on auth service
./scripts/stop-all.sh auth          # Stop it
# Make your changes...
./scripts/start-all.sh auth         # Restart only auth

# Or restart with dependencies
./scripts/stop-all.sh auth tenant
./scripts/start-all.sh auth tenant
```

### API Gateway Testing
```bash
# Test gateway with minimal backend
./scripts/start-all.sh gateway user
```

### Full Stack Development
```bash
# Start everything
./scripts/start-all.sh

# Stop everything when done
./scripts/stop-all.sh
```

### Restart a Service
```bash
# Quick restart of single service
./scripts/stop-all.sh user && ./scripts/start-all.sh user
```

## Features

### Smart Building
- ✅ Only builds services you're starting
- ✅ Parallel compilation for speed
- ✅ Skip build if only starting frontend

### Smart Dependencies
- ✅ Only starts Docker if backend services requested
- ✅ Only waits for PostgreSQL if backend services need it
- ✅ Only checks .env files for services being started

### Resource Management
- ✅ Only uses resources for services you need
- ✅ Faster iteration when working on specific services
- ✅ Cleaner testing environment

### Log Management
- ✅ Only manages logs for affected services
- ✅ Separate log files per service
- ✅ Optional cleanup after stopping

## Service Ports

| Service | Port |
|---------|------|
| API Gateway | 8080 |
| Auth Service | 8082 |
| User Service | 8083 |
| Tenant Service | 8084 |
| Notification Service | 8085 |
| Frontend | 3000 |

## Logs

View logs for running services:

```bash
# All logs
tail -f /tmp/api-gateway.log
tail -f /tmp/auth-service.log
tail -f /tmp/user-service.log
tail -f /tmp/tenant-service.log
tail -f /tmp/notification-service.log
tail -f /tmp/frontend.log

# Or all at once
tail -f /tmp/*.log
```

## Troubleshooting

### Service Won't Start
```bash
# Check logs
tail -50 /tmp/[service-name].log

# Check if port is in use
lsof -i :[port]

# Check if Docker is running
docker ps
```

### Service Won't Stop
```bash
# Force kill by port
lsof -ti:[port] | xargs kill -9

# Or use process name
pkill -f [service-name]
```

### Clean Restart
```bash
# Stop everything, clean logs, restart
./scripts/stop-all.sh
# Select 'y' to remove logs
./scripts/start-all.sh
```

## Environment Variables

Services load configuration from:
1. `PROJECT_ROOT/.env` - Global settings
2. Service-specific `.env` files in each service directory

Example:
- `backend/auth-service/.env`
- `backend/user-service/.env`
- `frontend/.env.local`

## Error Messages

### "Missing: [service]/.env"
**Solution**: Run `./scripts/setup-env.sh` to create environment files

### "Docker is not running"
**Solution**: Start Docker Desktop or Docker daemon

### "PostgreSQL did not become ready in time"
**Solution**: Check Docker logs: `docker-compose logs postgres`

## Tips

1. **Use short names** for faster typing: `auth` instead of `auth-service`
2. **Start minimal services** for faster development cycles
3. **Check logs** if a service fails to start
4. **Clean logs regularly** to save disk space
5. **Use specific service start/stop** to save system resources

## Examples by Role

### UI/UX Developer
```bash
# Only need frontend
./scripts/start-all.sh frontend
```

### Backend Developer (Auth)
```bash
# Auth service with its dependencies
./scripts/start-all.sh gateway auth tenant
```

### Backend Developer (User Management)
```bash
# User service with dependencies
./scripts/start-all.sh gateway user tenant auth
```

### Full Stack Developer
```bash
# Everything
./scripts/start-all.sh
```

### DevOps/Testing
```bash
# All backend, no frontend
./scripts/start-all.sh gateway auth user tenant notification
```

## CI/CD Integration

```bash
# In CI pipeline
./scripts/start-all.sh all
# Run tests
npm test
# Cleanup
./scripts/stop-all.sh all
```

## Performance Notes

- Starting all services: ~15-30 seconds
- Starting single service: ~5-10 seconds
- Starting frontend only: ~5-8 seconds
- Stopping services: ~2-5 seconds

---

**Last Updated**: 2024-11-30
**Version**: 2.0 (Selective Service Management)
