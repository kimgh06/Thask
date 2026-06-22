<div align="center">

<table><tr>
<td align="center" width="120"><img src="public/icon.svg" alt="Thask" width="80" /></td>
<td align="center"><h1>Thask</h1><em>Thask it, done.</em></td>
<td align="center" width="160"><img src="public/mascot.png" alt="Thask Mascot" width="140" /></td>
</tr></table>

**The dependency graph layer for AI-assisted development.**
<br />
Map what depends on what, then let Claude Code / Cursor / Codex query it through MCP — with provenance guards so agents can't silently land hallucinated descriptions on your graph.

<br />

[![MIT License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
&nbsp;
[![Works with Claude Code](https://img.shields.io/badge/Works_with-Claude_Code-D97706?logo=anthropic&logoColor=white)](docs/CLAUDE_CODE_PLUGIN.md)
&nbsp;
[![Works with Cursor](https://img.shields.io/badge/Works_with-Cursor-000000?logo=cursor&logoColor=white)](docs/MCP.md)
&nbsp;
[![MCP](https://img.shields.io/badge/MCP-24_tools-7C3AED)](docs/MCP.md)
&nbsp;
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
&nbsp;
[![SvelteKit](https://img.shields.io/badge/SvelteKit-Svelte_5-FF3E00?logo=svelte&logoColor=white)](https://svelte.dev/)
&nbsp;
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

<!-- TODO: Add hero screenshot/GIF here -->

[**Live Demo** — Documentation Graph](https://thask.kimgh06.com/shared/562a734b85200bbdd65a65e18e066dc377b7dcfd3c86edb1eab4e6aece9a9bbf) · [**Architecture Graph**](https://thask.kimgh06.com/shared/de5cb3d3587d2479d6c875f873bd6479f031187e4054fd8cc93eafca8c840691)

</div>

---

## Before vs After

| | **Without Thask** | **With Thask** |
|---|---|---|
| You ask | *"Refactor this payment function."* | *"Refactor this payment function."* |
| Agent sees | Just the open file. | The file **plus** every node that `depends_on` it across your graph. |
| Agent answers | Plausible code. Three downstream flows quietly break. | "This change touches 3 flows and 2 UIs — I will update them together, or stop and ask." |
| Description drift | Agent rewrites the "why" prose on confidence, you read it, the next agent treats it as ground truth. | Agent keys default to **blocking semantic writes**. They propose to a queue; a human approves. Source-of-record stays human. |

Every change is recorded with **6-dimension provenance** (actor, channel, agent model, mutation kind, trigger, evidence) so the next agent knows what to trust and what to re-derive from code.

---

## Why Thask?

Spreadsheets lose context. Linear issue trackers hide relationships. **Thask maps your product as a living graph** — so you can see what breaks before it breaks.

<table>
<tr>
<td width="25%" align="center">
<h4>AI-Native</h4>
<p>Ship with <code>thask mcp serve</code> — Claude Code and Cursor read your dependency graph as a tool. Ask "what breaks if I change this?" and get real answers.</p>
</td>
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

Slide-out panel with full editing — title, description (with markdown rendering), type, status, tags, connected nodes, and a complete change history audit log.

### Edge Relationships

Five edge types with distinct colors: `depends_on`, `blocks`, `related`, `parent_child`, `triggers`. Draggable waypoints for edge routing. Click any edge to change its type or delete it.

### CLI & MCP Integration

Full CLI for terminal workflows (`npm install -g @thask-org/cli`). 24 MCP tools for AI agent integration — Claude Code and Cursor can query and modify your graph directly. One-step browser login (`thask login`), in-place upgrades (`thask self-update`). [CLI Reference](docs/CLI.md) · [MCP Guide](docs/MCP.md)

### Per-Key Permissions & Provenance (v0.5.9+)

Every API key is classified as `user_interactive`, `agent`, or `service` with seven independent permission flags. Agent keys default to **blocking semantic writes** (description, "why" content) and node verification — so a hallucinated description can't silently land on your graph. Every write records 6-dimension provenance (actor, channel, agent model, mutation kind, trigger, evidence) to a single `audit_log` table. [DATABASE.md > Provenance](docs/DATABASE.md#provenance--audit-migrations-006010)

### Suggestion Queue

Agents wanting to revise a description post to `node_suggestions` and a human approves before the change lands. The deciding human becomes the author of record — the agent is credited only in audit metadata. Server-enforced: `accepted` decisions require a `user_interactive` actor regardless of permission flags.

### Bulk Operations (v0.5.10+)

Three endpoints cut N round-trips down to one — `node.batch_update` (up to 200), `edge.batch_create` / `edge.batch_delete` (up to 500). Atomic on permission / cycle failure; per-item skip reasons in `skipped[]`; HTTP `207 Multi-Status` when any item skips. Saves substantial agent context (1 call vs N).

### Local-First CLI Telemetry (v0.5.15+)

Every CLI invocation, MCP tool call, and HTTP response appends a single JSONL line to `~/.thask/events.jsonl` — on your machine only, no upload. Inspect with `thask usage` (30-day summary, p50/p95 latency, top commands), `thask reflog` / `thask history` (recent events, full-text search), or `tail -f` the file directly. Raw bodies are opt-in (`thask telemetry config set capture_payloads true`); the default captures only metadata. Tokens, URL credentials, JWT and cookies are masked at write time.

### Go Dependency Scanner

Scan Go codebases to auto-generate dependency graphs. `thask scan --path .` parses `go.mod` and imports, creating nodes and edges automatically. Extensible via [plugin system](docs/PLUGINS.md).

### Graph Analysis

Detect dependency cycles (Tarjan DFS) and find the critical path (longest `depends_on`/`blocks` chain). Toggle Analysis Mode (`Shift+A`) to visualize cycles and critical path on the canvas.

### External API (v1)

Versioned REST API at `/api/v1/` for third-party integrations. OpenAPI 3.1 spec, interactive Scalar docs, structured error responses, and idempotency support. [API Guide](backend/api/README.md)

### Role-Based Access

Four team roles — Owner, Admin, Member, Viewer — with granular permissions. Per-project roles (Editor, Viewer). API key authentication for programmatic access.

### Project Sharing

Share projects via link with viewer or editor access. Manage per-project members with granular roles. Public shared views support realtime collaboration. Embeddable graph views and OG image generation.

### Templates

Start new projects from built-in templates: API Flow, Microservice Map, Sprint Board. One-click apply from the project creation flow.

### Theme System

Light and dark mode with system detection. Persisted per user. Design system uses CSS variables throughout.

---

## How It Works

Thask has two parts:

| | Server (self-hosted) | CLI (local) |
|---|---|---|
| **What** | Web UI + REST API + PostgreSQL | Terminal commands + MCP server |
| **Install** | `docker compose up` | `npm install -g @thask-org/cli` |
| **Used by** | Humans (browser) | Humans (terminal) + AI agents (MCP) |
| **Data** | Stores everything (nodes, edges, users) | Reads/writes via server API |

```
Browser ──→ Thask Server (Docker) ──→ PostgreSQL
                ↑
CLI / MCP ──────┘
```

The **server** runs your graph database and web UI. The **CLI** talks to the server's API — you can create nodes, run scans, and analyze graphs from the terminal. AI agents (Claude Code, Cursor) use the CLI's built-in MCP server.

---

## Quick Start for AI Agents

Get Thask working with Claude Code in 2 minutes:

1. **Start Thask** (if not running):
   ```bash
   make up
   ```

2. **Install the CLI + log in via browser:**
   ```bash
   npm install -g @thask-org/cli
   thask config set url http://localhost:7244
   thask login   # opens browser, click Approve, token saved
   ```
   `thask login` (v0.5.11+) replaces the old "make a key in Settings,
   copy a 64-char string, paste it" dance. The MCP server reads the
   same `~/.thask/config.json`, so this single login covers Claude Code
   too. For headless / SSH sessions: create a key in the web UI and
   run `thask config set token <key>` instead.

3. **Add to Claude Code** (`.claude/mcp.json`):
   ```json
   {
     "mcpServers": {
       "thask": {
         "command": "thask",
         "args": ["mcp", "serve"]
       }
     }
   }
   ```

Now Claude Code can read and modify your dependency graph — with v0.5.9+
permission gates so agent keys can't silently land hallucinated
descriptions. See [MCP Guide](docs/MCP.md) for details and the
[official Claude Code plugin](docs/CLAUDE_CODE_PLUGIN.md) for a zero-setup
install.

---

## Quick Start

### Docker (recommended)

```bash
make up   # auto-generates .env with SESSION_SECRET on first run
```

Or manually:

```bash
cp .env.example .env
# Edit .env and set SESSION_SECRET (or let make generate it)
docker compose up --build
```

Open [http://localhost:7243](http://localhost:7243) and create an account.

### Local Development (macOS / Linux)

The Makefile is the source of truth — every dev workflow has a target.

```bash
make dev          # one-shot: starts DB + capture worker, then backend + frontend in parallel
```

Or run pieces individually in separate terminals:

```bash
make dev-db       # PostgreSQL only (docker compose)
make dev-capture  # Playwright capture worker (docker compose)
make dev-backend  # Go backend (requires air: go install github.com/air-verse/air@latest)
make dev-frontend # SvelteKit frontend on :7243
```

If `air` isn't installed, run the backend directly:

```bash
cd backend && cp .env.example .env && go run ./cmd/server
```

### Local Development (Windows)

Prerequisites: [Go 1.26+](https://go.dev/dl/), [Node.js 22+](https://nodejs.org/), [Docker Desktop](https://www.docker.com/products/docker-desktop/)

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
| **Backend** | Go 1.26 (Echo v4) |
| **Frontend** | SvelteKit + Svelte 5 (runes) |
| **CLI** | Go (Cobra) + MCP server |
| **Graph Engine** | Cytoscape.js + fCOSE layout + edgehandles |
| **Styling** | Tailwind CSS v4 |
| **State** | Svelte 5 runes ($state, $derived, $effect) |
| **Database** | PostgreSQL 17 + pgx/v5 (raw SQL) |
| **Auth** | Session-based (bcrypt + HTTP-only cookies) |
| **Testing** | Go test (unit + bench) + Playwright (E2E) |
| **Deploy** | Docker Compose (3 services) |
| **npm** | `@thask-org/cli` (esbuild pattern, 5 platforms) |

---

## Project Structure

```
backend/
  cmd/server/           # Go entrypoint, route registration
  internal/
    config/             # Environment configuration
    dto/                # Request/response structs, v1 errors, pagination
    handler/            # HTTP handlers (auth, team, node, edge, impact,
                        #   graph_analysis, activity, events, og_image, api_key)
    middleware/         # Auth, role, project access, shared access,
                        #   idempotency, v1 response
    model/              # Domain models & enums
    repository/         # Database access layer (pgx, 11 repos)
    service/            # Business logic (waterfall, impact, graph_analysis,
                        #   layout, eventhub, auth)
    migrate/            # Migration runner
  migrations/           # SQL migration files (001~005)
  api/                  # OpenAPI spec + API guide

frontend/
  src/
    routes/
      login/            # Login page
      register/         # Register page
      dashboard/        # Dashboard & team pages
        [teamSlug]/
          [projectId]/  # Graph editor page
          members/      # Team member management
        settings/       # User settings
      shared/[shareToken]/  # Public shared view
      embed/[shareToken]/   # Embeddable graph
      docs/             # Documentation page
    lib/
      api.ts            # Typed API client
      types.ts          # TypeScript type definitions
      markdown.ts       # Markdown renderer (marked + DOMPurify)
      stores/           # Svelte 5 rune stores (auth, graph, teams,
                        #   members, realtime, theme, undo)
      components/       # CytoscapeCanvas, GraphToolbar, DetailSidePanel,
                        #   MarkdownEditor, ActivityFeed, TemplateSelector,
                        #   ShareDialog, SearchBar, ProjectMenu, AddNodeModal
      cytoscape/        # Styles, layouts, impact, sync, resize,
                        #   portOverlay, groupHelpers, statusDot
      managers/         # nodeCrud, edgeCrud
  e2e/                  # Playwright E2E tests
  static/templates/     # Built-in project templates (3)

cli/
  cmd/thask/            # CLI entrypoint
  internal/
    cmd/                # Cobra commands (node, edge, team, project,
                        #   graph, impact, scan, mcp, auth, config,
                        #   aliases, install, version)
    mcp/                # MCP server (stdio protocol, 12+ tools)
    scan/               # Go dependency scanner + plugin runner
    client/             # HTTP client for backend API
    config/             # Config file management (~/.config/thask/)
    output/             # Output formatting (JSON, table, quiet)

npm/                    # npm distribution (@thask-org/cli, 5 platforms)
```

---

## Data Model

```
Users ──< TeamMembers >── Teams ──< Projects ──< Nodes ──< NodeHistory
  │                                    │            │
  └──< ApiKeys                         │            └──< Edges
                                       │
                                       └──< ProjectMembers >── Users
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
| `CAPTURE_URL` | Internal Playwright capture worker URL | `http://localhost:7241` |
| `CAPTURE_INTERNAL_SECRET` | Optional shared secret for backend → capture worker calls | — |
| `CAPTURE_TIMEOUT_SECONDS` | Capture worker request timeout | `30` |
| `V1_ALLOWED_ORIGINS` | Comma-separated CORS origins for `/api/v1/` | `*` |
| `MAX_REQUEST_BODY_BYTES` | Max request body size for v1 routes (bytes) | `1048576` (1MB) |

### Frontend (`frontend/.env`)

| Variable | Description | Default |
|---|---|---|
| `BACKEND_URL` | Backend API URL (server-side proxy) | `http://localhost:7244` |

### Docker Compose (`.env`)

| Variable | Description | Default |
|---|---|---|
| `SESSION_SECRET` | **Required.** Random 64+ char string for session signing | — |
| `APP_URL` | Public URL of the application | `http://localhost:7243` |
| `BACKEND_URL` | Backend URL for frontend proxy | `http://backend:7244` |
| `POSTGRES_PASSWORD` | PostgreSQL password | `thask_password` |
| `CAPTURE_PORT` | Local/dev host port for the Playwright capture worker | `7241` |
| `CAPTURE_INTERNAL_SECRET` | Optional shared secret for backend → capture worker calls | — |
| `CAPTURE_FRONTEND_URL` | URL the capture worker opens in Chromium | `http://frontend:7243` |
| `BROWSER_WS_ENDPOINT` | Browserless Chrome WebSocket endpoint used by the capture worker | `ws://browserless:3000` |

### Deploying to a Custom Domain

Set `APP_URL` in `.env` to your public URL:

```bash
# .env
APP_URL=https://thask.example.com
SESSION_SECRET=your-random-64-char-string
```

This configures CORS and CSRF protection automatically. `BACKEND_URL` does **not** need to change — the frontend server proxies API requests to the backend over the internal Docker network.

```
Browser ──https──▶ Reverse Proxy (nginx/Cloudflare)
                        │
                        ▼ :7243
                   Frontend (SvelteKit)
                        │
                        ▼ http://backend:7244 (Docker internal)
                   Backend (Go)
                        │
                        ▼ http://capture:7241 (Docker internal, optional)
                   Capture Worker (Playwright)
                        │
                        ▼ postgres:5432 (Docker internal)
                   PostgreSQL
```

Place a reverse proxy (e.g. nginx, Caddy, Cloudflare Tunnel) in front to handle SSL termination.

---

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — Layers, data flow, directory structure
- [Database](docs/DATABASE.md) — ER diagram, tables, indexes, relations
- [API Reference](docs/API.md) — 30+ endpoints with request/response examples
- [Graph Engine](docs/GRAPH.md) — Node types, edge types, GROUP, impact mode
- [Keyboard Shortcuts](docs/SHORTCUTS.md) — All shortcuts and interactions
- [CLI Reference](docs/CLI.md) — Installation, commands, shell aliases
- [MCP Guide](docs/MCP.md) — AI agent integration (Claude Code, Cursor)
- [Claude Code Plugin](docs/CLAUDE_CODE_PLUGIN.md) — Auto-inject project context + register MCP server in every session
- [Scanner Plugins](docs/PLUGINS.md) — Scanner plugin system for new languages
- [V1 API Guide](backend/api/README.md) — External API quickstart for third-party developers

---

## Makefile Commands

The full surface — `make` is the canonical entrypoint for dev, build, test, and release.

### Development
| Command | Description |
|---|---|
| `make dev` | Start DB + capture worker, then backend + frontend in parallel |
| `make dev-services` | Start DB + capture worker (docker compose) |
| `make dev-db` | Start PostgreSQL only |
| `make dev-backend` | Run Go backend with air hot reload |
| `make dev-frontend` | Run SvelteKit frontend on `:7243` |
| `make dev-capture` | Build + start the Playwright capture worker |
| `make db-up` / `make db-down` | Start / stop the dev PostgreSQL container |

### Build & Release
| Command | Description |
|---|---|
| `make build` | Build CLI + backend (`backend/bin/server`) + frontend |
| `make build-cli` | Build CLI binary into `bin/thask` (with version + commit ldflags) |
| `make release-cli` | Cross-compile CLI for 5 platforms, tag, push, npm publish, GitHub Release. Set `CLI_VERSION=x.y.z` (or read from `cli/package.json`). Optional `THASKOTP=...` for npm 2FA. |

### Test
| Command | Description |
|---|---|
| `make test` | Backend Go tests + frontend checks |
| `make test-backend` | Backend Go tests (verbose) |
| `make test-cli` | CLI Go tests |
| `make test-e2e` | Playwright E2E tests |
| `make bench` | Scanner + graph analysis benchmarks |

### Docker
| Command | Description |
|---|---|
| `make up` | Full stack via Docker Compose (auto-generates `.env` with `SESSION_SECRET` on first run) |
| `make down` | Stop Docker Compose |
| `make clean` | Remove `backend/bin`, `bin/`, `dist/`, `frontend/build`, `frontend/.svelte-kit` |

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
- [x] CLI tool with full graph operations
- [x] MCP server for AI agent integration (12 tools)
- [x] API key authentication (Bearer token)
- [x] Role-based access control (owner/admin/member/viewer)
- [x] Team member management (invite, roles, transfer)
- [x] Project sharing with public links (viewer/editor modes)
- [x] CLI sharing commands (share, unshare, invite, kick)

### v0.2 — External API & Collaboration (Done)
- [x] Real-time collaboration via SSE
- [x] Graph export as PNG / JSON
- [x] Graph import (replace/merge mode)
- [x] Versioned external API (`/api/v1/`) with OpenAPI spec
- [x] Interactive API docs (Scalar UI at `/api/v1/docs`)
- [x] Structured error responses for v1
- [x] Idempotency support for API consumers
- [x] CORS configuration for external domains
- [x] Edge waypoints (draggable bend points)
- [x] SVG edge rendering with snap guides
- [x] Embeddable graph views (`/embed/:shareToken`)
- [x] OG image generation for shared links

### v0.3 — Scanner + Graph Intelligence (Done)
- [x] Go dependency scanner (`thask scan`) with `go/ast` parsing
- [x] Cycle detection (Tarjan DFS) and critical path analysis
- [x] Analysis Mode frontend (Shift+A toggle, cycles/critical path highlighting)
- [x] MCP tools: `thask.scan.run`, `thask.graph.analyze`
- [x] GitHub Actions CI (backend tests, CLI tests, frontend check)
- [x] Scanner plugin system with documentation

### v0.4 — Community & Ecosystem (Done)
- [x] npm distribution (`@thask-org/cli`, 5 platform binaries, esbuild pattern)
- [x] Scanner plugin interface (any executable outputting ImportGraphRequest JSON)
- [x] Homebrew formula template
- [x] Performance benchmarks (scanner + graph analysis)

### v0.5 — Visual Polish & Analytics (Done)
- [x] Light/dark theme system with system detection
- [x] Activity feed (recent changes with user attribution)
- [x] Project templates (API Flow, Microservice Map, Sprint Board)
- [x] Amber Precision design system fully applied
- [x] Markdown description rendering (marked + DOMPurify)
- [x] **v0.5.6** — Graph image capture (PNG/SVG via Playwright worker)
- [x] **v0.5.8** — TypeScript/JavaScript dependency scanner (alias resolution, SvelteKit `$lib`)
- [x] **v0.5.9** — Per-key permissions + suggestion queue + 6-dimension `audit_log` (anti-hallucination guards)
- [x] **v0.5.10** — Bulk endpoints (`node.batch_update`, `edge.batch_*`) with HTTP 207 partial-success; `thask self-update`
- [x] **v0.5.11** — `thask login` browser-based authentication; URL auto-normalization

### Future
- [ ] Graph version snapshots & diffing
- [ ] Node lifecycle analytics (time-in-status, bottleneck detection)
- [ ] Webhook triggers on graph changes
- [ ] GitHub repo sync (auto-update graph on push)
- [ ] Slack / Discord notifications
- [ ] Comment threads on nodes
- [ ] Mobile responsive layout
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
