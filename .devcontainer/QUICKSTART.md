# Dev Container Quick Reference

## 🚀 Getting Started

1. Open in VS Code: `code .`
2. Reopen in Container: `Ctrl+Shift+P` → "Dev Containers: Reopen in Container"
3. Wait for first-time setup (~5-10 minutes)

## ⌨️ Common Commands

| Task                     | Command                                                        |
| ------------------------ | -------------------------------------------------------------- |
| **Frontend Development** | `cd frontend && npm run dev`                                   |
| **All Backend Services** | `cd backend && ./scripts/start-all.sh`                         |
| **Single Service**       | `cd backend && ./scripts/start-all.sh order`                   |
| **Stop Services**        | `cd backend && ./scripts/stop-all.sh`                          |
| **Database Migrations**  | `./scripts/run-migrations.sh`                                  |
| **Database Shell**       | `psql postgresql://pos_user:pos_password@postgres:5432/pos_db` |
| **Redis CLI**            | `redis-cli -h redis -a pos_password`                           |

## 📱 Service URLs

| Service     | URL                      | Credentials                 |
| ----------- | ------------------------ | --------------------------- |
| Frontend    | `http://localhost:3000`  | -                           |
| API Gateway | `http://localhost:8080`  | -                           |
| MinIO       | `http://localhost:9001`  | `minioadmin` / `minioadmin` |
| Mailhog     | `http://localhost:5555`  | -                           |
| Vault       | `https://localhost:8200` | See initialization          |

## 🔧 Using Makefile (Simplified Commands)

```bash
make help              # Show all available commands
make setup             # Initial setup
make frontend          # Start frontend
make backend           # Start all backend services
make db-migrate        # Run migrations
make db-shell          # Open PostgreSQL
make redis-cli         # Open Redis
make logs-order        # View order service logs
make clean             # Clean builds
```

## 🐛 Debugging

### View Logs

```bash
# Backend service logs
tail -f /tmp/order-service.log
tail -f /tmp/auth-service.log

# Docker container logs
docker logs -f pos-dev
docker logs -f pos-dev-postgres
```

### Database Queries

```bash
psql postgresql://pos_user:pos_password@postgres:5432/pos_db
> SELECT * FROM users LIMIT 5;
> \dt  # List tables
> \q  # Quit
```

### Check Service Health

```bash
curl http://localhost:8080/api/v1/health
curl http://localhost:8082/api/v1/health
```

## 🔄 Rebuilding Container

If you modify `.devcontainer/Dockerfile` or `devcontainer.json`:

- `Ctrl+Shift+P` → "Dev Containers: Rebuild Container"
- Or: `docker compose -f .devcontainer/docker-compose.yml build --no-cache`

## 💾 Important Volumes

- **Workspace**: `/workspace` - All project files (live sync)
- **Go extensions**: Persisted for faster starts
- **Node modules**: Persisted to avoid reinstalling

## 🧹 Cleanup

```bash
# Remove all dev container volumes
docker volume prune

# Reset database
docker compose -f .devcontainer/docker-compose.yml down -v

# Full reset (nuclear option)
docker system prune -a --volumes
```

## ⚡ Tips & Tricks

1. **Use tmux for multiple terminals**: `tmux new-session -s dev`
2. **Port forwarding works automatically** - no extra setup needed
3. **Database persists** between container restarts
4. **Node modules cached** in volume for faster npm install
5. **Go build cache** persists for faster builds

## 📚 Documentation

- Full docs: [.devcontainer/README.md](.devcontainer/README.md)
- Dev Container official: https://code.visualstudio.com/docs/devcontainers/containers
- Docker Compose: https://docs.docker.com/compose/

## 🚨 Common Issues

| Issue                 | Solution                                           |
| --------------------- | -------------------------------------------------- |
| Port already in use   | `docker ps` then `docker kill <container>`         |
| PostgreSQL not ready  | Wait 30s, check: `docker logs pos-dev-postgres`    |
| npm install fails     | `docker volume rm devcontainer-node-modules`       |
| Container won't start | Rebuild: `Ctrl+Shift+P` → "Rebuild Container"      |
| Git auth issues       | Ensure SSH keys available: `ssh-add ~/.ssh/id_rsa` |

## 💡 Need Help?

1. Check [.devcontainer/README.md](.devcontainer/README.md) for detailed docs
2. View service logs: `docker logs -f <container_name>`
3. Check VS Code's Dev Containers extension status
4. Verify Docker has sufficient resources allocated

---

For more details, see the comprehensive guide in `.devcontainer/README.md`
