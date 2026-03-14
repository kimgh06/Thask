# Contributing to Thask

## Prerequisites

- [Go 1.23+](https://go.dev/dl/)
- [Node.js 22+](https://nodejs.org/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for PostgreSQL)
- (macOS/Linux) `make` is pre-installed
- (Windows) Optional: `scoop install make` or `choco install make`

## Development Setup

1. Fork and clone the repo

### macOS / Linux

2. Start PostgreSQL:
   ```bash
   make dev-db
   ```

3. Set up the backend:
   ```bash
   cd backend
   cp .env.example .env
   go run ./cmd/server
   ```

4. Set up the frontend (in another terminal):
   ```bash
   cd frontend
   cp .env.example .env
   npm install
   npm run dev -- --port 7243
   ```

Or run everything together:
```bash
make dev
```

### Windows (PowerShell)

2. Start PostgreSQL:
   ```powershell
   docker compose -f docker-compose.dev.yml up -d
   ```

3. Set up the backend:
   ```powershell
   cd backend
   copy .env.example .env
   go run ./cmd/server
   ```

4. Set up the frontend (in another terminal):
   ```powershell
   cd frontend
   copy .env.example .env
   npm install
   npm run dev -- --port 7243
   ```

## Running Tests

```bash
# Go unit tests (macOS/Linux)
make test-backend
# Windows alternative:
cd backend && go test -v ./...

# Frontend type check
cd frontend && npm run check

# Playwright E2E tests (requires running backend + frontend)
# macOS/Linux:
make test-e2e
# Windows:
cd frontend && npx playwright test
```

## Project Structure

- `backend/` — Go API server (Echo v4 + pgx/v5)
- `frontend/` — SvelteKit app (Svelte 5 + Tailwind v4)
- `docs/` — Documentation

## Code Style

### Backend (Go)
- Follow standard Go conventions
- Use `slog` for structured logging
- Keep handlers thin — business logic goes in services
- Database queries go in the repository layer

### Frontend (Svelte)
- Use Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`)
- Keep components focused — one responsibility per component
- Use the typed API client (`$lib/api.ts`)
- Run `npm run check` to verify no type errors

## Pull Requests

- Create a feature branch from `main`
- Keep PRs focused on a single concern
- Ensure all tests pass: `make test-backend` and `cd frontend && npm run check`
- Include a clear description of the change

## Reporting Bugs

Use GitHub Issues with a clear description and steps to reproduce.
