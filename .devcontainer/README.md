# Dev Container Setup - Point of Sale System

This directory contains all configuration files for the VS Code Dev Container development environment. The Dev Container provides a fully isolated and reproducible development setup with all required dependencies, databases, and services pre-configured.

## 📋 What's Included

### Development Environment

- **Go 1.23** - Backend services development
- **Node.js LTS** - Frontend development with npm, yarn, and pnpm
- **Git LFS** - Large file support
- **Development Tools**: postgresql-client, redis-tools, jq, curl, etc.

### Databases & Services

- **PostgreSQL 14** - Primary data store (port 5432)
- **Redis 8** - Session storage and caching (port 6379)
- **Apache Kafka** - Event streaming (port 9092)
- **MinIO** - Object storage for product photos (port 9000)
- **Mailhog** - Email testing and debugging (port 5555)
- **HashiCorp Vault** - Secrets management (port 8200)

### VS Code Extensions

- Go development tools
- ESLint and Prettier
- Tailwind CSS support
- Docker support
- Prisma ORM support
- GitLens
- GitHub Copilot

## 🚀 Quick Start

### Prerequisites

- VS Code with "Dev Containers" extension installed (`ms-vscode-remote.remote-containers`)
- Docker and Docker Compose installed on your system
- At least 4GB available RAM for containers

### Opening in Dev Container

1. **Open the workspace in VS Code:**

   ```bash
   code /path/to/point-of-sale-system
   ```

2. **Reopen in Dev Container:**
   - Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on Mac)
   - Search for "Dev Containers: Reopen in Container"
   - Select it

   Alternatively, VS Code may prompt you with a notification offering to reopen in the container.

3. **Wait for setup:**
   - The first time will take 5-10 minutes to build the image and install dependencies
   - Subsequent starts are much faster (1-2 minutes)

### Verification

Once the dev container is running, verify everything is set up:

```bash
# Check Go version
go version

# Check Node version
node --version
npm --version

# Check PostgreSQL connection
psql postgresql://pos_user:pos_password@postgres:5432/pos_db -c "SELECT version();"

# Check Redis connection
redis-cli -h redis -a pos_password ping

# Check Kafka connectivity
kafka-broker-api-versions --bootstrap-server kafka:29092

# Check Vault connectivity
curl -k https://localhost:8200/v1/sys/health
```

## 📝 Environment Variables

Environment variables are configured in `.devcontainer/.env`:

- **Database**: PostgreSQL connection details
- **Cache**: Redis credentials
- **Services**: Port mappings for all microservices
- **External APIs**: Placeholders for Midtrans and Google Maps keys

To customize, edit `.devcontainer/.env`:

```bash
MIDTRANS_SERVER_KEY=your_sandbox_key
GOOGLE_MAPS_API_KEY=your_api_key
```

## 🏗️ Project Structure

```
.devcontainer/
├── devcontainer.json        # Main Dev Container configuration
├── Dockerfile               # Custom dev environment image
├── docker-compose.yml       # Services definition (PostgreSQL, Redis, etc.)
├── post-create.sh          # Initialization script (runs after container creation)
├── .env                     # Environment variables (customize as needed)
├── .env.example             # Environment variables template
└── README.md                # This file
```

## 🛠️ Common Development Tasks

### Running Frontend

```bash
cd frontend
npm install    # First time only
npm run dev    # Start development server
```

Frontend will be available at: http://localhost:3000

### Running Backend Services

```bash
cd backend
./scripts/start-all.sh        # Start all services
# or
./scripts/start-all.sh order  # Start specific service
```

Available services:

- API Gateway (8080)
- Auth Service (8082)
- Tenant Service (8081)
- User Service (8083)
- Order Service (8084)
- Product Service (8085)
- Notification Service (8086)
- Audit Service (8087)

### Running Database Migrations

```bash
./scripts/run-migrations.sh
```

### Stopping Services

```bash
cd backend
./scripts/stop-all.sh         # Stop all services
# or
./scripts/stop-all.sh order   # Stop specific service
```

## 🌐 Accessing Services

| Service         | URL                    | Credentials             |
| --------------- | ---------------------- | ----------------------- |
| Frontend        | http://localhost:3000  | -                       |
| API Gateway     | http://localhost:8080  | -                       |
| MinIO Console   | http://localhost:9001  | minioadmin / minioadmin |
| Mailhog         | http://localhost:5555  | -                       |
| HashiCorp Vault | https://localhost:8200 | See initialization      |
| PostgreSQL      | localhost:5432         | pos_user / pos_password |
| Redis           | localhost:6379         | Password: pos_password  |

## 🐛 Debugging

### View Service Logs

```bash
# Check running containers
docker ps

# View logs for a service
docker logs -f pos-dev  # Dev container logs
docker logs -f pos-dev-postgres
docker logs -f pos-dev-redis

# Or check local log files
tail -f /tmp/order-service.log
tail -f /tmp/auth-service.log
```

### Database Inspection

```bash
psql postgresql://pos_user:pos_password@postgres:5432/pos_db
\dt                     # List tables
\d table_name          # Describe table
SELECT * FROM users;   # Query data
```

### Redis Inspection

```bash
redis-cli -h redis -a pos_password
> KEYS *                 # List all keys
> GET key_name          # Get a value
> MONITOR               # Monitor operations
```

## 🔄 Container Lifecycle

### Starting

```bash
# VS Code automatically starts containers when you reopen them
# Or manually in terminal:
docker compose -f .devcontainer/docker-compose.yml up -d
```

### Stopping

```bash
# Containers stop when VS Code closes
# Or manually:
docker compose -f .devcontainer/docker-compose.yml down
```

### Rebuilding (if Dockerfile changes)

- Press `Ctrl+Shift+P` → "Dev Containers: Rebuild Container"
- Or: `docker compose -f .devcontainer/docker-compose.yml build --no-cache`

### Cleaning Up

```bash
# Remove all dev container volumes
docker volume prune

# Or specific volumes:
docker volume rm devcontainer-go-extensions devcontainer-node-modules
```

## 💾 Volume Mounts

- **Workspace**: Entire project synced in real-time
- **Go Extensions**: Persisted for faster subsequent container starts
- **Node Modules**: Persisted across container restarts

This ensures fast iteration and avoids reinstalling dependencies.

## ⚙️ Customization

### Adding Go Tools

Edit `.devcontainer/Dockerfile` and add to the `RUN go install` line:

```dockerfile
RUN go install github.com/some/tool@latest && \
    go install github.com/another/tool@latest
```

Then rebuild the container:

```bash
# Ctrl+Shift+P → "Dev Containers: Rebuild Container"
```

### Adding Node Packages Globally

Edit `.devcontainer/Dockerfile` and add to the `RUN npm install -g` line:

```dockerfile
RUN npm install -g package-name
```

### Adding VS Code Extensions

Edit `devcontainer.json`, section `customizations.vscode.extensions`:

```json
"extensions": [
  "existing.extensions",
  "publisher.new-extension"
]
```

## 🚨 Troubleshooting

### Container won't start

1. Check Docker is running: `docker ps`
2. Check available disk space: `df -h`
3. Rebuild container: `Ctrl+Shift+P` → "Dev Containers: Rebuild Container"

### Port already in use

If services fail to start due to port conflicts:

```bash
# Find process using port
lsof -i :8080

# Either kill the process or change port in .devcontainer/docker-compose.yml
```

### Database connection fails

```bash
# Check PostgreSQL is healthy
docker ps -a | grep postgres

# Check logs
docker logs pos-dev-postgres

# Restart PostgreSQL
docker restart pos-dev-postgres
```

### Node modules permission issues

```bash
# Clear node modules volume
docker volume rm devcontainer-node-modules

# Reinstall
cd frontend && npm install
```

### Go build failures

```bash
# Clear build cache
go clean -cache
cd backend/order-service && go build -o order-service.bin main.go
```

## 📚 Resources

- [VS Code Dev Containers Documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [Docker Documentation](https://docs.docker.com/)
- [Go Development Tools](https://github.com/golang/tools/wiki)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)

## 🤝 Contributing

When working in the dev container:

1. Changes to source code are automatically reflected
2. Use formatting tools: `gofmt`, `prettier`, `eslint`
3. Run tests before committing
4. Keep the `.devcontainer` configuration updated with new dependencies

## 📞 Support

For issues or questions:

1. Check troubleshooting section above
2. Review service logs
3. Check VS Code's Dev Containers extension status
4. Verify Docker daemon is running and has sufficient resources

---

Happy coding! 🎉
