.PHONY: dev dev-db dev-backend dev-frontend build test test-e2e clean

# Development
dev: dev-db
	@echo "Starting backend and frontend..."
	@make -j2 dev-backend dev-frontend

dev-db:
	docker compose -f docker-compose.dev.yml up -d

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev -- --port 7243

# Build
build:
	cd backend && go build -o bin/server ./cmd/server
	cd frontend && npm run build

# Test
test:
	cd backend && go test ./...
	cd frontend && npm test

test-backend:
	cd backend && go test -v ./...

test-e2e:
	cd frontend && npx playwright test

# Docker
up: .env
	docker compose up --build

down:
	docker compose down

.env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		sed -i '' "s/^SESSION_SECRET=$$/SESSION_SECRET=$$(openssl rand -hex 32)/" .env; \
		echo "Created .env with generated SESSION_SECRET"; \
	fi

# Database
db-up:
	docker compose -f docker-compose.dev.yml up -d

db-down:
	docker compose -f docker-compose.dev.yml down

# Clean
clean:
	rm -rf backend/bin
	rm -rf frontend/build frontend/.svelte-kit
