<div align="center">

<table><tr>
<td align="center" width="120"><img src="public/icon.svg" alt="Thask" width="80" /></td>
<td align="center"><h1>Thask</h1><em>Thask it, done.</em></td>
<td align="center" width="160"><img src="public/mascot.png" alt="Thask Mascot" width="140" /></td>
</tr></table>

**Visualize product flows, tasks, and bugs as a linked node graph.**
<br />
Built for QA risk management and change impact analysis.

<br />

[![MIT License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
&nbsp;
[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
&nbsp;
[![SvelteKit](https://img.shields.io/badge/SvelteKit-Svelte_5-FF3E00?logo=svelte&logoColor=white)](https://svelte.dev/)
&nbsp;
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
&nbsp;
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

<!-- TODO: Add hero screenshot/GIF here -->

</div>

---

## Why Thask?

Spreadsheets lose context. Linear issue trackers hide relationships. **Thask maps your product as a living graph** — so you can see what breaks before it breaks.

<table>
<tr>
<td width="25%" align="center">
<h4>Graph-first Thinking</h4>
<p>Every flow, task, and bug is a node. Every dependency is a visible edge. No more hidden connections.</p>
</td>
<td width="25%" align="center">
<h4>Impact at a Glance</h4>
<p>One click shows which nodes are affected by recent changes. Catch regressions before they ship.</p>
</td>
<td width="25%" align="center">
<h4>Self-hosted</h4>
<p><code>docker compose up</code> — that's it. Your data stays on your infrastructure. No vendor lock-in.</p>
</td>
<td width="25%" align="center">
<h4>Team-ready</h4>
<p>Multi-team projects with role-based access. Everyone sees the same graph.</p>
</td>
</tr>
</table>

---

## Features

### Interactive Graph Editor

Drag-and-drop nodes with **7 types** — Flow, Branch, Task, Bug, API, UI, and Group. Connect them by hovering and dragging the edge handle. Auto-layout with the fCOSE force-directed algorithm.

### QA Impact Mode

Toggle Impact Mode to instantly highlight **changed nodes** and their **downstream dependencies**. Dimmed nodes are safe; glowing nodes need attention.

### Group Nodes

Organize related nodes into collapsible groups. Drag nodes in and out. Resize groups freely. Double-click to collapse with a child count badge.

### Status Tracking & Filters

Track every node as `PASS` / `FAIL` / `IN_PROGRESS` / `BLOCKED` with color-coded visuals. Filter the graph by node type or status to focus on what matters.

### Node Detail Panel

Slide-out panel with full editing — title, description, type, status, tags, connected nodes, and a complete change history audit log.

### Edge Relationships

Five edge types with distinct colors: `depends_on`, `blocks`, `related`, `parent_child`, `triggers`. Click any edge to change its type or delete it.

---

## Quick Start

### Docker (recommended)

```bash
docker compose up
```

Open [http://localhost:7243](http://localhost:7243) and create an account.

### Local Development (macOS / Linux)

```bash
# 1. Start PostgreSQL
make dev-db

# 2. Set up backend
cd backend
cp .env.example .env
go run ./cmd/server

# 3. Set up frontend (in another terminal)
cd frontend
cp .env.example .env
npm install
npm run dev -- --port 7243
```

Or simply:

```bash
make dev   # starts DB, backend, and frontend together
```

### Local Development (Windows)

Prerequisites: [Go 1.23+](https://go.dev/dl/), [Node.js 22+](https://nodejs.org/), [Docker Desktop](https://www.docker.com/products/docker-desktop/)

```powershell
# Terminal 1 — Start PostgreSQL
docker compose -f docker-compose.dev.yml up -d

# Terminal 2 — Start backend
cd backend
copy .env.example .env
go run ./cmd/server

# Terminal 3 — Start frontend
cd frontend
copy .env.example .env
npm install
npm run dev -- --port 7243
```

> **Tip:** To use `make` on Windows, install via `scoop install make` or `choco install make`.

Open [http://localhost:7243](http://localhost:7243)

---

## Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go 1.23 (Echo v4) |
| **Frontend** | SvelteKit + Svelte 5 (runes) |
| **Graph Engine** | Cytoscape.js + fCOSE layout + edgehandles |
| **Styling** | Tailwind CSS v4 |
| **State** | Svelte 5 runes ($state, $derived, $effect) |
| **Database** | PostgreSQL 17 + pgx/v5 (raw SQL) |
| **Auth** | Session-based (bcrypt + HTTP-only cookies) |
| **Testing** | Go test (unit) + Playwright (E2E) |
| **Deploy** | Docker Compose (3 services) |

---

## Project Structure

```
backend/
  cmd/server/           # Go entrypoint
  internal/
    config/             # Environment configuration
    dto/                # Request/response structs
    handler/            # HTTP handlers (auth, team, node, edge, impact)
    middleware/         # Auth & project access middleware
    model/              # Domain models & enums
    repository/         # Database access layer (pgx)
    service/            # Business logic (waterfall, impact, auth)
  migrations/           # SQL migration files

frontend/
  src/
    routes/
      login/            # Login page
      register/         # Register page
      dashboard/        # Dashboard & team pages
        [teamSlug]/
          [projectId]/  # Graph editor page
    lib/
      api.ts            # Typed API client
      types.ts          # TypeScript type definitions
      stores/           # Svelte 5 rune stores (auth, graph)
      components/       # CytoscapeCanvas, GraphToolbar, AddNodeModal,
                        # EdgeColorPopover, NodeDetailPanel
      cytoscape/        # Styles, layouts, impact mode, group helpers
  e2e/                  # Playwright E2E tests
```

---

## Data Model

```
Users ──< TeamMembers >── Teams ──< Projects ──< Nodes ──< NodeHistory
                                                    │
                                                    └──< Edges
```

**Node types:** `FLOW` `BRANCH` `TASK` `BUG` `API` `UI` `GROUP`
**Node statuses:** `PASS` `FAIL` `IN_PROGRESS` `BLOCKED`
**Edge types:** `depends_on` `blocks` `related` `parent_child` `triggers`

---

## Environment Variables

### Backend (`backend/.env`)

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://thask:thask_dev_password@localhost:7242/thask` |
| `SESSION_SECRET` | Random string for session signing | — |
| `PORT` | Backend server port | `7244` |
| `FRONTEND_URL` | Frontend URL for CORS | `http://localhost:7243` |

### Frontend (`frontend/.env`)

| Variable | Description | Default |
|---|---|---|
| `PUBLIC_API_URL` | Backend API URL | `http://localhost:7244` |

---

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — Layers, data flow, directory structure
- [Database](docs/DATABASE.md) — ER diagram, tables, indexes, relations
- [API Reference](docs/API.md) — 22+ endpoints with request/response examples
- [Graph Engine](docs/GRAPH.md) — Node types, edge types, GROUP, impact mode
- [Keyboard Shortcuts](docs/SHORTCUTS.md) — All shortcuts and interactions

---

## Makefile Commands

| Command | Description |
|---|---|
| `make dev` | Start DB + backend + frontend |
| `make dev-db` | Start PostgreSQL only |
| `make dev-backend` | Start Go backend |
| `make dev-frontend` | Start SvelteKit frontend |
| `make build` | Build backend + frontend |
| `make test` | Run Go unit tests + frontend checks |
| `make test-backend` | Run Go unit tests (verbose) |
| `make test-e2e` | Run Playwright E2E tests |
| `make up` | Docker Compose full stack |
| `make down` | Stop Docker Compose |
| `make clean` | Remove build artifacts |

---

## Roadmap

### v0.1 — Foundation (Done)
- [x] Graph CRUD (nodes, edges, groups)
- [x] 7 node types & 4 statuses with visual styling
- [x] fCOSE auto-layout & manual positioning
- [x] Drag-and-drop grouping & compound nodes
- [x] Node search & keyboard shortcuts
- [x] QA impact analysis (BFS-based)
- [x] Status waterfall propagation
- [x] Session-based auth & team management
- [x] Docker Compose one-command deploy
- [x] Go backend with 18 unit tests
- [x] Playwright E2E tests (13 tests)

### v0.2 — Collaboration & Export
- [ ] Real-time collaboration (WebSocket / SSE)
- [ ] Graph snapshot — export as PNG / SVG
- [ ] PDF report generation (impact summary)
- [ ] Activity feed (recent changes across team)
- [ ] Comment threads on nodes

### v0.3 — Automation & Integration
- [ ] API token auth for CI/CD pipelines
- [ ] Webhook triggers on status change
- [ ] GitHub / GitLab issue sync
- [ ] Slack / Discord notifications
- [ ] Bulk import / export (JSON, CSV)

### v0.4 — UX & Templates
- [ ] Graph templates (preset flow patterns)
- [ ] Light / dark mode toggle
- [ ] Custom node colors & icons
- [ ] Minimap
- [ ] Mobile responsive layout

### Future
- [ ] Version history & graph diffing
- [ ] Role-based permissions (view-only, edit, admin)
- [ ] Plugin system for custom node types
- [ ] AI-assisted impact prediction
- [ ] Self-hosted SSO (SAML / OIDC)

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions and guidelines.

---

<div align="center">

## License

[MIT](LICENSE) &copy; Thask Contributors

**Thask it, done.**

[Report Bug](../../issues) &middot; [Request Feature](../../issues)

</div>
