# Architecture

## Overview

Thask uses a **monorepo with separate backend, frontend, and CLI** services. The backend is a Go API server; the frontend is a SvelteKit application; the CLI is a Go tool with an embedded MCP server. All communicate via REST/stdio and are deployed as independent Docker containers or binaries.

```
┌──────────────────────────────────────────────────────────┐
│  CLI (Go + Cobra)                                        │
│  thask node · thask edge · thask graph · thask impact    │
├──────────────────────────────────────────────────────────┤
│  MCP Server (stdio)                                      │
│  12 tools · thask.node.* · thask.edge.* · thask.graph.*  │
╠══════════════════════════════════════════════════════════╣
│  Frontend (SvelteKit + Svelte 5)                         │
│  CytoscapeCanvas · GraphToolbar · NodeDetailPanel · ...  │
├──────────────────────────────────────────────────────────┤
│  State Layer (Svelte 5 Runes)                            │
│  AuthStore · GraphStore                                  │
├──────────────────────────────────────────────────────────┤
│  API Client (fetch + credentials: include)               │
│  api.get · api.post · api.patch · api.delete             │
╠══════════════════════════════════════════════════════════╣
│  Backend (Go + Echo v4)                                  │
├──────────────────────────────────────────────────────────┤
│  Middleware Layer                                        │
│  CORS · RateLimiter · Auth · ProjectAccess · TeamAccess  │
├──────────────────────────────────────────────────────────┤
│  Handler Layer (HTTP → Business Logic)                   │
│  auth · team · node · edge · impact · summary · api_key  │
├──────────────────────────────────────────────────────────┤
│  Service Layer (Pure Logic)                              │
│  waterfall · impact · auth (bcrypt, tokens)              │
├──────────────────────────────────────────────────────────┤
│  Repository Layer (pgx/v5 → PostgreSQL 17)               │
│  users · sessions · teams · projects · nodes · edges     │
└──────────────────────────────────────────────────────────┘
```

---

## Data Flow

```
Svelte 5 Stores                   API Client
  AuthStore ──┐                    api.get/post/patch/delete
  GraphStore ─┘                          │
        │                                │
        ▼                                ▼
   ┌──────────────────────────────────────────┐
   │         Svelte Components (UI)          │
   │  +page.svelte (graph editor)            │
   │   ├─ CytoscapeCanvas (graph render)     │
   │   ├─ GraphToolbar (controls)            │
   │   ├─ NodeDetailPanel (editor)           │
   │   ├─ AddNodeModal (create)              │
   │   └─ EdgeColorPopover (edge type)       │
   └──────────────┬──────────────────────────┘
                  │ HTTP (fetch w/ cookies)
                  ▼
   ┌──────────────────────────────────────────┐
   │         Go Backend (Echo v4)            │
   │  Middleware → Handler → Service → Repo  │
   │  22+ endpoints, rate limited, logged    │
   └──────────────┬──────────────────────────┘
                  │ pgx/v5 queries
                  ▼
   ┌──────────────────────────────────────────┐
   │         PostgreSQL 17                   │
   │  9 tables, 5 enums, 12+ indexes        │
   └──────────────────────────────────────────┘
```

---

## Directory Structure

```
cli/
├── cmd/thask/
│   └── main.go                    # CLI entrypoint
├── internal/
│   ├── cmd/                       # Cobra commands (node, edge, team, etc.)
│   ├── mcp/                       # MCP server (stdio, 12 tools)
│   ├── client/                    # HTTP client for backend API
│   ├── config/                    # Config file (~/.config/thask/)
│   └── output/                    # Output formatting (JSON, table, quiet)
├── go.mod
└── go.sum

backend/
├── cmd/server/
│   └── main.go                    # Entrypoint: DB, migrations, routes, graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go              # Env var loading (DATABASE_URL, SESSION_SECRET, etc.)
│   ├── dto/
│   │   ├── request.go             # Request validation structs (validate tags)
│   │   └── response.go            # OK(data) / Err(message) helpers
│   ├── handler/
│   │   ├── auth.go                # Register, Login (session rotation), Me, Logout
│   │   ├── team.go                # List, Create, GetBySlug, Delete, Members, Projects
│   │   ├── node.go                # CRUD, BatchUpdatePositions, waterfall propagation
│   │   ├── edge.go                # CRUD with self-reference check
│   │   ├── impact.go              # BFS-based impact analysis
│   │   ├── summary.go             # Dashboard summary counts
│   │   ├── api_key.go             # Create, List, Delete API keys
│   │   └── validator.go           # Custom validator (slug regex)
│   ├── middleware/
│   │   ├── auth.go                # Cookie → session → user context injection
│   │   ├── role.go                # TeamAccess (slug resolution + membership), RequireRole (minimum role check)
│   │   └── project_access.go      # Team membership verification (centralized)
│   ├── model/
│   │   ├── enums.go               # NodeType, NodeStatus, EdgeType, TeamRole constants
│   │   └── models.go              # All data models with JSON/DB tags
│   ├── repository/
│   │   ├── db.go                  # pgxpool connection (max 20 conns)
│   │   ├── user.go                # Create, FindByEmail, FindByID
│   │   ├── session.go             # Create, ValidateToken, Delete*, DeleteExpired
│   │   ├── team.go                # CRUD, AddMember, IsMember, GetMembers
│   │   ├── project.go             # CRUD, VerifyAccess (centralized)
│   │   ├── node.go                # CRUD, BatchPositions, FindChangedSince, UpdateStatus
│   │   ├── edge.go                # CRUD, FindConnected
│   │   ├── history.go             # Create, FindByNodeID (with user join)
│   │   └── api_key.go             # API key storage and validation
│   └── service/
│       ├── auth.go                # bcrypt (cost 12), token generation, session expiry
│       ├── waterfall.go           # BFS status propagation (max depth 10)
│       └── impact.go              # Bidirectional BFS impact analysis
├── migrations/
│   └── 001_initial.sql            # Full schema: enums, tables, indexes, constraints
├── Dockerfile                     # Multi-stage build (golang:1.23 → alpine:3.20)
├── go.mod
└── go.sum

frontend/
├── src/
│   ├── app.css                    # Tailwind v4 + CSS custom properties (dark theme)
│   ├── routes/
│   │   ├── +layout.svelte         # Root layout, CSS import, user fetch
│   │   ├── +page.svelte           # Auth-aware redirect
│   │   ├── login/+page.svelte     # Login form
│   │   ├── register/+page.svelte  # Registration form
│   │   └── dashboard/
│   │       ├── +layout.svelte     # Protected layout, sidebar navigation
│   │       ├── +page.svelte       # Team listing + create team
│   │       └── [teamSlug]/
│   │           ├── +page.svelte   # Project listing + create project
│   │           └── [projectId]/
│   │               └── +page.svelte   # Graph editor (main page)
│   └── lib/
│       ├── api.ts                 # Typed API client (credentials: include)
│       ├── types.ts               # All TypeScript types matching Go models
│       ├── stores/
│       │   ├── auth.svelte.ts     # AuthStore ($state runes)
│       │   └── graph.svelte.ts    # GraphStore (selection, filters, impact, collapse)
│       ├── components/
│       │   ├── CytoscapeCanvas.svelte    # Full graph rendering + interactions
│       │   ├── GraphToolbar.svelte       # Toolbar with filters, search, impact
│       │   ├── AddNodeModal.svelte       # Node creation modal
│       │   ├── EdgeColorPopover.svelte   # Edge type/label editing popover
│       │   └── NodeDetailPanel.svelte    # Slide-out detail panel with tabs
│       └── cytoscape/
│           ├── styles.ts          # 60+ Cytoscape style rules
│           ├── layouts.ts         # fCOSE + preset layout configs
│           ├── groupHelpers.ts    # Child/descendant queries
│           ├── impact.ts          # Impact mode activate/deactivate
│           └── extensions.d.ts    # Ambient type declarations
├── e2e/                           # Playwright E2E tests
│   ├── helpers.ts                 # Test utilities (register, login)
│   ├── auth.spec.ts               # Auth flow tests (5)
│   ├── team-project.spec.ts       # Team/project tests (3)
│   └── graph.spec.ts              # Graph editor tests (5)
├── playwright.config.ts
├── svelte.config.js               # adapter-node
├── vite.config.ts                 # Tailwind v4 plugin
├── Dockerfile                     # Multi-stage build (node:22 → alpine)
└── package.json
```

---

## Key Layers

### Backend — Repository Layer

| Repository | Responsibility |
|---|---|
| `UserRepo` | User CRUD, lookup by email/ID |
| `SessionRepo` | Token-based sessions, validate, cleanup expired |
| `TeamRepo` | Team CRUD, membership management |
| `ProjectRepo` | Project CRUD, `VerifyAccess` (centralized auth check) |
| `NodeRepo` | Node CRUD, batch positions, filtered queries, status updates |
| `EdgeRepo` | Edge CRUD, find connected edges |
| `HistoryRepo` | Audit log creation and retrieval |

### Backend — Service Layer

| Service | Responsibility |
|---|---|
| `auth` | Password hashing (bcrypt cost 12), token generation, session expiry |
| `waterfall` | BFS status propagation across edges (max depth 10) |
| `impact` | Bidirectional BFS from changed nodes with configurable depth |
| `eventhub` | Pub/sub hub for SSE realtime events (node/edge CRUD, layout, import) |
| `layout` | Server-side graph layout algorithms (dagre, grid) with GROUP auto-sizing |

### Frontend — Stores (Svelte 5 Runes)

| Store | Responsibility |
|---|---|
| `AuthStore` | User session, login/register/logout, `$state` user |
| `GraphStore` | Node/edge selection, type/status filters, impact mode, collapsed groups |
| `RealtimeStore` | SSE connection to `/api/projects/:id/events`, debounced graph refresh |

### Auth Flow

```
Browser → SvelteKit route → api.ts (fetch w/ cookie)
       → Go middleware: Auth → ValidateToken()
                              → sessions table (token lookup)
                              → users table (join)
                              → inject user into Echo context

CLI/MCP → Authorization header (Bearer <api_key>)
        → Go middleware: Auth → ValidateAPIKey()
                              → api_keys table (hash lookup)
                              → users table (join)
                              → inject user into Echo context
```

- Session tokens: 32-byte hex, 7-day expiry
- Storage: HttpOnly cookie (`thask_session`)
- API keys: SHA256 hash, per-user, with optional expiration
- Passwords: bcrypt with cost 12
- Session rotation: all previous sessions deleted on login
- Rate limiting: 20 req/s per client

### Access Control

```
User → TeamMembers → Teams → Projects → Nodes/Edges
```

Every API route verifies:
1. Valid session or API key (`Auth` middleware)
2. Team membership for the project (`ProjectAccess` middleware — centralized, not duplicated)
3. Minimum role (if required) via `RequireRole` middleware (owner, admin, member, viewer)

---

## Testing

| Layer | Framework | Count | Command |
|---|---|---|---|
| Backend unit tests | Go `testing` | 18 | `make test-backend` |
| Frontend type check | svelte-check | 299 files | `cd frontend && npm run check` |
| E2E tests | Playwright | 13 | `make test-e2e` |

### Unit Test Coverage

- `waterfall_test.go` — 10 tests: BFS propagation, parent aggregation, depth limits
- `impact_test.go` — 4 tests: single/multi-depth, bidirectional BFS
- `auth_test.go` — 4 tests: hash roundtrip, token generation
