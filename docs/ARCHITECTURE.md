# Architecture

## Overview

Thask uses a **monorepo with separate backend, frontend, and CLI** services. The backend is a Go API server; the frontend is a SvelteKit application; the CLI is a Go tool with an embedded MCP server. All communicate via REST/stdio and are deployed as independent Docker containers or binaries.

```
┌──────────────────────────────────────────────────────────┐
│  CLI (Go + Cobra)                                        │
│  thask node · thask edge · thask graph · thask impact    │
├──────────────────────────────────────────────────────────┤
│  MCP Server (stdio)                                      │
│  16 tools · thask.node.* · thask.edge.* · thask.graph.* · thask.scan.* · thask.guide  │
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
│  CORS · RateLimiter · Auth · ProjectAccess · TeamAccess · SharedAccess  │
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

## Local Ports

| Service | Port | Notes |
|---|---:|---|
| Capture worker | `7241` | Playwright-based graph capture worker |
| PostgreSQL | `7242` | Local Docker database |
| Frontend | `7243` | SvelteKit app |
| Backend | `7244` | Go API server |

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
│   ├── mcp/                       # MCP server (stdio, 24 tools incl. provenance + bulk)
│   ├── client/                    # HTTP client for backend API
│   ├── config/                    # Config file (~/.config/thask/)
│   └── output/                    # Output formatting (JSON, table, quiet)
├── go.mod
└── go.sum

backend/
├── cmd/server/
│   ├── main.go                    # Entrypoint: DB, migrations, graceful shutdown
│   └── routes.go                  # Route registration and middleware wiring
├── internal/
│   ├── config/
│   │   └── config.go              # Env var loading (DATABASE_URL, SESSION_SECRET, etc.)
│   ├── dto/
│   │   ├── request.go             # Request validation structs (validate tags)
│   │   └── response.go            # OK(data) / Err(message) helpers
│   ├── audit/                     # Provenance & permission logger (v0.5.9)
│   │   └── audit.go               # Logger.Log() + RequirePermission() — actor/channel extraction
│   ├── handler/
│   │   ├── auth.go                # Register, Login (session rotation), Me, Logout
│   │   ├── team.go                # List, Create, GetBySlug, Delete, Members, Projects
│   │   ├── node.go                # CRUD, BatchUpdate, Import, Layout, waterfall, cycle detector
│   │   ├── edge.go                # CRUD + BatchCreate / BatchDelete with skip reasons
│   │   ├── suggestion.go          # Suggest / List / Decide / Verify (v0.5.9 human-in-the-loop)
│   │   ├── impact.go              # BFS-based impact analysis
│   │   ├── summary.go             # Dashboard summary counts
│   │   ├── api_key.go             # Create (kind + permissions), List, Delete API keys
│   │   └── validator.go           # Custom validator (slug regex)
│   ├── middleware/
│   │   ├── auth.go                # Cookie/Bearer → user context + actor_kind + permissions + X-Thask-Client parsing
│   │   ├── role.go                # TeamAccess (slug resolution + membership), RequireRole (minimum role check)
│   │   ├── project_access.go      # Team membership verification (centralized)
│   │   ├── idempotency.go         # Idempotency-Key replay cache (v1 API only)
│   │   └── shared_access.go       # SharedAccess: share token validation, 30-second cache, role mapping
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
│   │   ├── history.go             # Create, FindByNodeID (with user join) — being superseded by audit_log
│   │   ├── api_key.go             # API key storage incl. kind + permissions (v0.5.9)
│   │   ├── audit.go               # AuditRepo — insert/list audit_log rows (v0.5.9)
│   │   └── suggestion.go          # SuggestionRepo — pending queue + decide + supersede (v0.5.9)
│   └── service/
│       ├── auth.go                # bcrypt (cost 12), token generation, session expiry
│       ├── waterfall.go           # BFS status propagation (max depth 10)
│       └── impact.go              # Bidirectional BFS impact analysis
├── migrations/
│   ├── 001_initial.sql            # Full schema: enums, tables, indexes, constraints
│   ├── 002_api_keys.sql           # API key storage (token hash + last_used_at)
│   ├── 003_project_access.sql     # Per-project members + share token
│   ├── 004_edge_routing.sql       # Edge waypoints / port persistence
│   ├── 005_v1_api.sql             # Idempotency keys + pagination indexes
│   ├── 006_audit_log.sql          # audit_log table (6-dimension provenance) — v0.5.9
│   ├── 007_node_provenance.sql    # nodes: description_source, last_verified_*, field_provenance
│   ├── 008_api_key_kind.sql       # api_keys: kind enum + permissions JSONB
│   ├── 009_suggestions.sql        # node_suggestions queue
│   └── 010_backfill_audit_from_history.sql  # one-time backfill
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

**Code generation (v0.5.14+).** Repositories are thin wrappers over `internal/dbgen/`, a `sqlc`-generated layer compiled from `backend/db/queries/*.sql` + `backend/migrations/*.sql`. Adding a field to a migration + the SELECT list of the query that returns it forces `make sqlc-gen` to update the generated row struct, which then surfaces as a compile error at every call site until the model / converter / handler is updated. SQL-to-Go drift is no longer silent.

Static queries (fixed shape: Create, FindByID, list-without-filters, simple Update/Delete) go through sqlc. Dynamic queries (Update setClauses built from a `map[string]any`, pagination cursor WHERE, `pgx.Batch` pipelining, multi-statement transactions) stay hand-written — these account for ~11 of 60+ queries across the repo. Wrappers convert the generated row types into the existing `model.*` structs at the boundary so handlers and services are unchanged.

JSONB columns (`nodes.metadata`, `nodes.field_provenance`, `edges.metadata`, `audit_log.metadata`) map to `json.RawMessage` in the generated layer; the wrapper marshals `any` → `json.RawMessage` on writes so the handler-facing shape stays `any`. `api_keys.permissions` overrides directly to `model.APIKeyPermissions` so the existing `Scan`/`Value` methods continue to drive serialization.

| Repository | Responsibility |
|---|---|
| `UserRepo` | User CRUD, lookup by email/ID |
| `SessionRepo` | Token-based sessions, validate, cleanup expired |
| `TeamRepo` | Team CRUD, membership management |
| `ProjectRepo` | Project CRUD, `VerifyAccess` (centralized auth check) |
| `ProjectMemberRepo` | Per-project member CRUD, list with user join |
| `NodeRepo` | Node CRUD, batch positions, filtered queries, status updates, provenance snapshot writes (`UpdateDescriptionProvenance`, `MarkVerified`) |
| `EdgeRepo` | Edge CRUD, find connected edges, `Pool()` accessor for tx-from-handler batch writes |
| `HistoryRepo` | Legacy node_history (read-only after v0.5.9; new writes go to `AuditRepo`) |
| `APIKeyRepo` | API key storage incl. `kind` + `permissions` JSONB (v0.5.9) |
| `AuditRepo` | `audit_log` insert + `ListByEntity` for the new 6-dimension provenance trail |
| `SuggestionRepo` | `node_suggestions` queue: Create, ListPending, Decide, SupersedePendingForNode |
| `IdempotencyRepo` | Replay cache for v1 API `Idempotency-Key` header |

### Backend — Service / Cross-cutting Layer

| Package | Responsibility |
|---|---|
| `service/auth` | Password hashing (bcrypt cost 12), token generation, session expiry |
| `service/waterfall` | BFS status propagation across edges (max depth 10) |
| `service/impact` | Bidirectional BFS from changed nodes with configurable depth |
| `service/eventhub` | Pub/sub hub for SSE realtime events (node/edge CRUD, layout, import) |
| `service/layout` | Server-side graph layout algorithms (dagre, grid) with GROUP auto-sizing |
| `audit` (own package) | `Logger.Log()` — write to `audit_log` with actor/channel pulled from echo context. `RequirePermission()` — gate handler by `permissions.write_semantic / write_structural / write_meta / verify / suggest / delete / read`. Lives outside `service` to avoid the `service ↔ repository ↔ middleware` import cycle. |

### Capture Worker (`capture/`)

Out-of-process Node.js + Playwright service that turns a project graph into a PNG.

```
CLI / API caller
   │  GET /api/v1/projects/:id/graph/capture?format=png
   ▼
Backend (NodeHandler.Capture)
   │  POST http://capture:7241/capture/graph  { nodes, edges, width, ... }
   │  X-Thask-Capture-Secret: <CAPTURE_INTERNAL_SECRET>
   ▼
Capture worker (capture/src/server.js)
   │  chromium.connectOverCDP(BROWSER_WS_ENDPOINT)
   ▼
Browserless Chrome → opens FRONTEND_URL/capture
   │  window.__thaskCapture.load({ nodes, edges })  → fit/zoom → metrics
   ▼
PNG bytes ─────────────────────────────► caller
```

- Sandbox: `read_only` rootfs, `cap_drop: ALL`, `no-new-privileges`, optional shared secret
- Input clamps: width/height ≤ 4096, scale ≤ 4, body ≤ 8 MB, URL allowlist (frontend + `data:` / `blob:` only)
- 503 if `CAPTURE_URL` is empty; SVG path renders inline in `og_image.go` and never needs the worker

### Frontend — Stores (Svelte 5 Runes)

| Store | Responsibility |
|---|---|
| `AuthStore` | User session, login/register/logout, `$state` user |
| `GraphStore` | Node/edge selection, type/status filters, impact mode, collapsed groups |
| `RealtimeStore` | SSE connection to `/api/projects/:id/events`, debounced graph refresh |

### Auth Flow

Every request first goes through `Auth` middleware, which establishes
**identity** (who) and **provenance** (how it got here). The provenance
fields are what `audit_log` ultimately stores.

```
Browser → SvelteKit route → api.ts (fetch w/ cookie + X-Thask-Client: thask-web/<sha>)
       → Auth middleware: ValidateToken()
                              → sessions table (token lookup)
                              → users table (join)
                              → context: user_id, actor_kind="user_interactive",
                                         client_type="thask-web", client_version

CLI → Authorization: Bearer <api_key> + X-Thask-Client: thask-cli/<ver>
   → Auth middleware: ValidateAPIKey()
                              → api_keys table (hash lookup) → kind + permissions
                              → users table (join)
                              → context: user_id, api_key_id,
                                         actor_kind=<kind>, permissions=<JSONB>,
                                         client_type="thask-cli", client_version

MCP (Claude Code) → Bearer <api_key> + X-Thask-Client: thask-mcp/<ver> model=<client> session=<uuid>
   → Same as CLI, plus context: agent_model + agent_session_id
```

Pulled from echo context downstream:
- **Handlers** call `audit.RequirePermission(c, mutationKind, action, evt)`
  before mutating. Denial → 403 with the missing flag name + a pointer at
  the right alternative tool (e.g. "use `thask.node.suggest_update`").
- **Audit logger** writes one row per field change to `audit_log` with
  every identity + provenance dimension populated. Failures are
  best-effort (`context.Background()` detach + warn log) so audit issues
  never roll back the user's primary write.

Auth fundamentals (unchanged):
- Session tokens: 32-byte hex, 7-day expiry
- Storage: HttpOnly cookie (`thask_session`)
- API keys: SHA256 hash, per-user, optional expiration, **per-key `kind` + `permissions` JSONB** (v0.5.9)
- Passwords: bcrypt with cost 12
- Session rotation: all previous sessions deleted on login
- Rate limiting: 20 req/s per client; v1 API additionally supports `Idempotency-Key` replay

### Access Control

```
User → TeamMembers → Teams → Projects → Nodes/Edges
```

Every API route verifies:
1. Valid session or API key (`Auth` middleware)
2. Team membership for the project (`ProjectAccess` middleware — centralized, not duplicated)
3. Minimum role (if required) via `RequireRole` middleware (owner, admin, member, viewer)
4. **(v0.5.9+)** Per-key `permissions` flag for the touched field's `mutation_kind`, via `audit.RequirePermission()` inside the handler

### Permission gate & suggestion queue (v0.5.9+)

Each field a write touches is classified as `semantic` / `structural` / `meta`
(see [GRAPH.md > Provenance & Authoring](GRAPH.md#provenance--authoring-v059)).
The calling key's `permissions` JSONB gates each class independently:

```
agent-kind key tries to update node.description
   │
   ▼
audit.RequirePermission(c, "semantic", "write", evt)
   │  permissions.write_semantic == false (default for agent kind)
   ▼
audit_log: action="write_denied", required="write_semantic"
   │
   ▼
403 + "Use thask.node.suggest_update to propose this change for human review"
```

Approved-path data flow for agent-proposed semantic changes:

```
Agent → thask.node.suggest_update     (permissions.suggest required)
      → POST /api/.../nodes/:id/suggestions
      → SuggestionRepo.Create(status="pending")
      → audit_log: action="suggested"

Human  → /dashboard browse pending list
       → thask.suggestions.decide {status:"accepted"}      ← actor_kind must be user_interactive
       → SuggestionRepo.Decide(decided_by=<human user_id>)
       → NodeRepo.Update(description=proposed_value)
       → NodeRepo.UpdateDescriptionProvenance(authored_by=<human>, source="human", agent_model=<original>)
       → audit_log: action="suggestion_decided" + later "updated"
```

The deciding human becomes the author of record on the node — the agent
that originally proposed is credited only in `audit_log.metadata`. This
is the design knob that prevents "agent writes → human rubber-stamps →
graph fills with confidently-wrong content" loops.

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
