# Makefile for Dev Container Common Tasks

.PHONY: help setup frontend backend db-migrate db-reset logs-order logs-auth logs-gateway clean restart all

help:
	@echo "Available commands:"
	@echo "  make setup              - Initial setup (install deps, build services, migrate DB)"
	@echo "  make frontend           - Start frontend dev server"
	@echo "  make backend            - Start all backend services"
	@echo "  make backend-order      - Start order service only"
	@echo "  make backend-auth       - Start auth service only"
	@echo "  make backend-stop       - Stop all backend services"
	@echo "  make db-migrate         - Run database migrations"
	@echo "  make db-reset           - Reset database (careful!)"
	@echo "  make logs-order         - View order service logs"
	@echo "  make logs-auth          - View auth service logs"
	@echo "  make logs-gateway       - View API gateway logs"
	@echo "  make db-shell           - Open PostgreSQL shell"
	@echo "  make redis-cli          - Open Redis CLI"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make restart            - Restart all services"
	@echo "  make all                - Full setup, build, and start"

setup:
	@echo "🚀 Setting up dev environment..."
	@cd frontend && npm install
	@cd ../backend && for dir in */; do [ -f "$$dir/main.go" ] && (cd $$dir && go build -o $${dir%/}.bin main.go && echo "✅ Built $$dir") || true; done
	@make db-migrate
	@echo "✨ Setup complete!"

frontend:
	@echo "Starting frontend dev server..."
	@cd frontend && npm run dev

backend:
	@echo "Starting all backend services..."
	@cd backend && bash ./scripts/start-all.sh

backend-order:
	@echo "Starting order service..."
	@cd backend && bash ./scripts/start-all.sh order

backend-auth:
	@echo "Starting auth service..."
	@cd backend && bash ./scripts/start-all.sh auth

backend-stop:
	@echo "Stopping all backend services..."
	@cd backend && bash ./scripts/stop-all.sh

db-migrate:
	@echo "Running database migrations..."
	@bash ./scripts/run-migrations.sh

db-reset:
	@echo "⚠️  Resetting database (all data will be lost)..."
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		psql postgresql://pos_user:pos_password@postgres:5432/pos_db -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" && \
		make db-migrate && \
		echo "✅ Database reset complete"; \
	else \
		echo "Cancelled"; \
	fi

logs-order:
	@tail -f /tmp/order-service.log 2>/dev/null || echo "Order service not running"

logs-auth:
	@tail -f /tmp/auth-service.log 2>/dev/null || echo "Auth service not running"

logs-gateway:
	@tail -f /tmp/api-gateway.log 2>/dev/null || echo "API gateway not running"

db-shell:
	@psql postgresql://pos_user:pos_password@postgres:5432/pos_db

redis-cli:
	@redis-cli -h redis -a pos_password

clean:
	@echo "Cleaning build artifacts..."
	@cd backend && for dir in */; do [ -f "$$dir/$${dir%/}.bin" ] && rm "$$dir/$${dir%/}.bin" && echo "Removed $$dir$${dir%/}.bin"; done
	@echo "✅ Cleanup complete"

restart:
	@echo "Restarting services..."
	@make backend-stop
	@sleep 2
	@make backend
	@echo "✅ Services restarted"

all: setup backend frontend
	@echo "✨ All services started!"

.DEFAULT_GOAL := help
