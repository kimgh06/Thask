# Architecture

## Overview

Thask uses a **monorepo with separate backend, frontend, and CLI** services. The backend is a Go API server; the frontend is a SvelteKit application; the CLI is a Go tool with an embedded MCP server. All communicate via REST/stdio and are deployed as independent Docker containers or binaries.

```
┌──────────────────────────────────────────────────────────┐
│  CLI (Go + Cobra)                                        │
│  thask node · edge · graph · impact · usage · reflog · telemetry  │
├──────────────────────────────────────────────────────────┤
│  MCP Server (stdio)                                      │
│  25 tools · thask.node.* · thask.edge.* · thask.graph.* · thask.scan.* · thask.guide  │
├──────────────────────────────────────────────────────────┤
│  Local State (~/.thask/, v0.5.15+)                       │
│  config.json · events.jsonl (append-only) · payloads/ (opt-in)  │
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
   │  12 tables, 5 enums, 15+ indexes       │
   └──────────────────────────────────────────┘
```

---

## Directory Structure

```
cli/
├── cmd/thask/
│   └── main.go                    # CLI entrypoint
├── internal/
│   ├── cmd/                       # Cobra commands (node, edge, team, telemetry, usage, reflog, ...)
│   ├── mcp/                       # MCP server (stdio, 25 tools incl. provenance + bulk)
│   ├── client/                    # HTTP client for backend API (records HTTP outcome → telemetry)
│   ├── config/                    # Config file (~/.thask/config.json)
│   ├── output/                    # Output formatting (JSON, table, quiet)
│   ├── telemetry/                 # Local-first event log (~/.thask/events.jsonl) — v0.5.15+
│   ├── scan/                      # Go codebase dependency scanner
│   └── update/                    # Background self-update check
├── go.mod
└── go.sum

# CLI runtime state (per-user, never tracked in git)
~/.thask/
├── config.json                    # URL, token, default project/team
├── events.jsonl                   # Append-only telemetry log (v0.5.15+)
├── payloads/                      # Opt-in raw request/response blobs (0600 each)
├── telemetry.json                 # install_id, capture_payloads, first_run_at
├── telemetry-tombstone            # Presence = telemetry disabled
└── uploaded.jsonl                 # Reserved for Phase 14 prod-upload ledger

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
│   ├── 010_backfill_audit_from_history.sql  # one-time backfill
│   ├── 011_nodes_created_by.sql             # nodes.created_by + backfill from history
│   └── 012_v060_knowledge_os.sql            # v0.6.0 Knowledge OS: 4 node types, 9 edge types, lifecycle_state, edges.metadata, node_comments, node_attachments, project_tags, api_keys.project_id
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
| `HistoryRepo` | Legacy node_history — reads only (v0.6.0 dropped all writes; table retained for backward-compat, DROP planned in v0.7.0) |
| `CommentRepo` | v0.6.0 threaded node comments (`node_comments`) — list / create / update / resolve / delete |
| `AttachmentRepo` | v0.6.0 per-node file metadata (`node_attachments`); blob storage handled by the handler under `THASK_ATTACHMENT_DIR` |
| `ProjectTagRepo` | v0.6.0 canonical project-scoped tag palette (`project_tags`) |
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
| middleware shim | A 4-line `cmd/server/main.go` middleware stamps `X-Thask-Server-Version: <handler.Version>` on every `/api/*` response so CLI telemetry (v0.5.15+) can record the backend it talked to. Static / non-API routes are excluded. |

### CLI — Local-First Telemetry (`cli/internal/telemetry/`)

New in v0.5.15. Every CLI invocation, HTTP round-trip, and MCP tool call appends one JSONL line to `~/.thask/events.jsonl` — local-only, append-only (single explicit purge command rewrites via temp file + `os.Rename`). Hooks live in three places: `cmd/root.go` `PersistentPreRunE` calls `telemetry.Begin()` and `Execute()` calls `Finalize()`; `client/client.go do()` records the HTTP outcome on every response; `mcp/handler.go HandleToolCall` wraps each tool dispatch with an `mcp_call` event whose `parent` references the in-flight invocation. Failures inside the telemetry stack are swallowed (`defer recover()`) so the CLI never breaks on telemetry pressure. Raw request/response bodies are opt-in (`capture_payloads`); the default captures only metadata. Inspection commands (`thask usage`, `thask reflog` / `history`, `thask telemetry status`) full-scan the file (~30 ms / year) — no SQL, no index.

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
5. **(v0.6.0+)** If the API key was minted with `projectId=X`, `ProjectAccess` middleware 403s any request against a different project. NULL scope keeps the pre-v0.6.0 behavior (all of the user's projects).

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

## Node / Edge Catalog

### Node types (11 as of v0.6.0)

| Type | Purpose | Notes |
|---|---|---|
| `FLOW` | End-to-end user or system flow | Original set (v0.1) |
| `BRANCH` | Conditional split | Original set |
| `TASK` | Unit of work | Original set |
| `BUG` | Defect / incident | Original set |
| `API` | API endpoint / contract | Original set |
| `UI` | Screen / component | Original set |
| `GROUP` | Container for compound layout | Original set |
| `REQUIREMENT` | Product / spec requirement | v0.6.0 Knowledge OS |
| `DECISION` | Architecture / product decision | v0.6.0 Knowledge OS |
| `EXPERIMENT` | Time-boxed test / A/B / spike | v0.6.0 Knowledge OS |
| `PERSON` | Owner / stakeholder / responsible individual | v0.6.0 Knowledge OS |

### Node status vs. lifecycle state (v0.6.0)

`status` and `lifecycle_state` are **orthogonal** and both stay on every node:

- `status` (existing enum: `PASS` / `FAIL` / `IN_PROGRESS` / `BLOCKED`) — QA
  / execution state. Waterfall propagation consumes this.
- `lifecycle_state` (new, free-form `TEXT NULL`) — domain phase specific to
  entity type. Examples: REQUIREMENT `DRAFT` → `APPROVED` → `IMPLEMENTED`;
  EXPERIMENT `PLANNED` → `RUNNING` → `CONCLUDED`; DECISION `PROPOSED` →
  `DECIDED` → `SUPERSEDED`; PERSON `ACTIVE` / `INACTIVE`. The server writes
  `lifecycle_state_changed_at = now()` whenever this field is updated, so a
  simple diff query answers "which requirements advanced this week".

Legacy types (`FLOW`, `TASK`, `BUG`, `API`, `UI`, `GROUP`, `BRANCH`) leave
`lifecycle_state = NULL` — the frontend Lifecycle state input only renders
for the four Knowledge OS types.

### Edge types (14 as of v0.6.0)

| Type | Semantic |
|---|---|
| `depends_on` | Source needs target — waterfall propagation follows this |
| `blocks` | Source prevents target — impact analysis follows this |
| `related` | Weak association (default) |
| `parent_child` | Legacy compound-graph link (superseded by `parent_id`) |
| `triggers` | Source causes target |
| `realizes` (v0.6.0) | Node implements a REQUIREMENT / DECISION |
| `conflicts` (v0.6.0) | Two nodes are mutually incompatible |
| `drives` (v0.6.0) | Motivating force (e.g. PERSON `drives` REQUIREMENT) |
| `supersedes` (v0.6.0) | New replaces old (metadata `{reason}`) |
| `tests` (v0.6.0) | Source verifies target (TEST → CODE) |
| `produced` (v0.6.0) | EXPERIMENT produced DECISION / FINDING (metadata `{outcome_summary}`) |
| `owns` (v0.6.0) | PERSON owns node |
| `decided` (v0.6.0) | DECISION concluded a TASK / FLOW |
| `reported` (v0.6.0) | PERSON reported BUG |

Every edge carries `metadata JSONB NOT NULL DEFAULT '{}'` (v0.6.0). MCP
`edge.update` exposes it directly.

### Adjacent tables (v0.6.0)

- `node_comments` — threaded discussion per node (`parent_id` self-reference).
  Author-scoped edit/delete on the write side; `resolved_at`/`resolved_by`
  close threads without deletion.
- `node_attachments` — per-node files, backed by `THASK_ATTACHMENT_DIR` on
  local FS today (MinIO/S3 planned). SHA256 stored per row; storage sharded
  by `{projectId}/{rand}-{safeName}`.
- `project_tags` — canonical metadata for the free-form `nodes.tags[]`
  strings (color, description, creator). `nodes.tags[]` remains the source
  of truth for what tags a given node carries; this table just decorates
  known tags at the project level.

### API-key project scoping (v0.6.0)

`api_keys.project_id` (nullable) constrains a key to one project. NULL
retains v0.5.x behavior (all of the user's projects). `ProjectAccess`
middleware verifies the header and 403s on mismatch.

### `node_history` deprecation

Writes to `node_history` stopped in v0.6.0 — `HistoryRepo.Create` /
`BatchCreateStatusChanges` are no-ops. `audit_log` is now the single source
of truth. The table is retained through v0.6.x for read compatibility
(activity feed, older reports) and will be DROP'd in v0.7.0.

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
