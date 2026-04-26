.PHONY: dev dev-db dev-backend dev-frontend build build-cli release-cli test test-backend test-cli test-e2e clean

# Development
dev: dev-db
	@echo "Starting backend and frontend..."
	@make -j2 dev-backend dev-frontend

dev-db:
	docker compose -f docker-compose.dev.yml up -d

dev-backend:
	cd backend && $$(go env GOPATH)/bin/air

dev-frontend:
	cd frontend && npm run dev -- --port 7243

# Build
build: build-cli
	cd backend && go build -o bin/server ./cmd/server
	cd frontend && npm run build

build-cli:
	cd cli && go build -ldflags "-X github.com/thask/cli/internal/cmd.Version=$$(git describe --tags --always 2>/dev/null || echo dev) -X github.com/thask/cli/internal/cmd.Commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) -X github.com/thask/cli/internal/mcp.Version=$$(git describe --tags --always 2>/dev/null || echo dev)" -o ../bin/thask ./cmd/thask

# Test
test:
	cd backend && go test ./...
	cd frontend && npm test

test-backend:
	cd backend && go test -v ./...

CLI_LDFLAGS = -X github.com/thask/cli/internal/cmd.Version=$$(git describe --tags --always 2>/dev/null || echo dev) \
             -X github.com/thask/cli/internal/cmd.Commit=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) \
             -X github.com/thask/cli/internal/mcp.Version=$$(git describe --tags --always 2>/dev/null || echo dev)

CLI_VERSION ?= $(shell node -p "require('./cli/package.json').version" 2>/dev/null)

release-cli:
	@if [ -z "$(CLI_VERSION)" ]; then echo "Error: no git tag found. Run: git tag v0.5.4 or make release-cli CLI_VERSION=0.5.4"; exit 1; fi
	THASKOTP=$(THASKOTP) ./scripts/release.sh v$(CLI_VERSION)

test-cli:
	cd cli && go test ./...

bench:
	@echo "=== Scanner Benchmark ==="
	cd cli && go test ./internal/scan/ -bench=. -benchmem -count=1
	@echo ""
	@echo "=== Graph Analysis Benchmark ==="
	cd backend && go test ./internal/service/ -bench=. -benchmem -count=1

test-e2e:
	cd frontend && npx playwright test

# Docker
up: .env
	docker compose up --build -d

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
	rm -rf backend/bin bin/ dist/
	rm -rf frontend/build frontend/.svelte-kit
