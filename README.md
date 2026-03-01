# Thask

Open-source web tool for visualizing product flows, branches, tasks, bugs, and follow-up work as a linked node graph. Designed for QA risk management and change impact analysis.

## Features

- **Interactive Graph Editor** - Drag-and-drop nodes with 6 types (Flow, Branch, Task, Bug, API, UI)
- **QA Impact Mode** - One-click view of which nodes are affected by recent changes
- **Status Tracking** - PASS / FAIL / IN_PROGRESS / BLOCKED with color-coded visualization
- **Force-directed Layout** - Auto-arranges nodes using fcose algorithm
- **Node Detail Panel** - Edit properties, view connections, and change history
- **Team Collaboration** - Team-based projects with role management
- **Self-hosted** - Deploy with Docker Compose

## Quick Start

### With Docker (recommended)

```bash
docker compose up
```

Open http://localhost:3000

### Local Development

```bash
# Start PostgreSQL
docker compose -f docker-compose.dev.yml up -d

# Install dependencies
npm install

# Set up database
cp .env.example .env
npm run db:push

# Start dev server
npm run dev
```

Open http://localhost:3000

## Tech Stack

- **Frontend**: Next.js 15 (App Router), TypeScript, Cytoscape.js, Tailwind CSS v4
- **Backend**: Next.js API Routes, Drizzle ORM
- **Database**: PostgreSQL 17
- **State**: Zustand + TanStack Query
- **Deploy**: Docker Compose

## Project Structure

```
src/
  app/           # Pages and API routes (Next.js App Router)
  components/    # React components (auth, graph, layout, panels)
  hooks/         # Custom React hooks
  lib/           # Core libraries (db, auth, cytoscape, validators)
  stores/        # Zustand state stores
  types/         # TypeScript type definitions
```

## Environment Variables

See `.env.example` for required variables.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
