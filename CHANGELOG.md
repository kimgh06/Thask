# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.5.13] - 2026-06-12

### Added
- **Handoff / onboarding flow (Phase 10).** Node detail panel now shows a subtle `Created by {email} · {relative date}` footer (click the email to add `?author=<uuid>` to the URL — filter consumer arrives later). New CLI command `thask graph export --format md` renders the project graph as a markdown handoff document: H1 project header, one H2 section per node with type/status/tags badge, description as-is, GROUP children list, `Depends on` / `Blocks` link sections (anchor-linked, GitHub-style slugs), and a creator footer. Nodes sorted by `createdAt` for deterministic output; golden-file test in `cli/internal/output/markdown_test.go`. New project template "Team Handoff Starter" with three example nodes demonstrating the convention.
- **Handoff Convention (docs/GRAPH.md).** New `## Handoff Convention (v0.5.13+)` section recommending node descriptions follow a structured markdown shape — `## Why` / `## Q&A` / `## Gotchas` / `## See also` — so descriptions stay useful for both onboarding humans and agents reading the graph. Three realistic example descriptions (FLOW: Stripe webhook handler, TASK: payment retry, BUG: duplicate charge). One-paragraph hint in `docs/MCP.md` near the `suggest_update` agent workflow pointing agents at the convention when composing `proposedValue`.
- **Backend: `creator_email` on Node read responses.** All node read paths (`FindByID`, `FindByProjectID`, `FindChangedSince`, `FindFailOrBug`, `FindByIDs`, `FindByProjectIDPaginated`) now LEFT JOIN `users` and return `creatorEmail` in the JSON payload. Migration `011_nodes_created_by.sql` adds `nodes.created_by` (nullable, FK to `users` with `ON DELETE SET NULL`) and backfills from the earliest `node_history` row with `action='created'` so existing projects pick up provenance without a re-import. Index `idx_nodes_created_by` for the eventual `?author=<uuid>` filter.

### Upgrade notes
- **Migration 011 lock window.** The backfill (`UPDATE nodes SET created_by = (SELECT ... FROM node_history ...) WHERE created_by IS NULL`) runs as a single statement and acquires row locks on every previously-existing node. On large tables (~100k+ nodes) this can block concurrent writes for several seconds. Deploy during a maintenance window if you have a busy instance, or chunk the backfill by hand before running migrations. For Thask's current scale this completes in well under a second.
- **`creatorEmail` is `""` until you create new nodes.** Existing nodes without any `node_history` row of `action='created'` remain with NULL `created_by` and an empty `creatorEmail`. The NodeDetailView footer renders "Created by unknown · {date}" for these. Future node creations carry their author end-to-end.

### Fixed
- **`nodeRepo.Create` and `Graph.Import` now populate `created_by`.** Migration 011 only added the column + backfilled history — write paths still left it NULL for brand-new nodes. Both the single-node create handler (`POST /api/projects/:pid/nodes`) and the bulk graph import handler now read `userID` from the request context and pass it to the INSERT, so every node created after this release carries its author. Anonymous shared-access creates still write NULL (no session user available — expected).
- **`thask graph export --format md` header showed the project UUID instead of the project name.** The CLI assumed `/api/projects/:id` always wraps responses in `{data: {...}}` but the endpoint returns a bare `{name: ...}` shape on some paths. Now tries both envelope and bare formats before falling back to the UUID.

### Docs
- **Phase 9 positioning rollout.** README hero gains a "Before vs After" comparison table and "Works with Claude Code / Cursor / MCP" badges to land the dependency-graph-for-AI-agents positioning above the fold. New `marketing/server.json` follows the MCP Registry 2025-09-29 schema (npm package `@thask-org/cli`, stdio transport, `THASK_URL` + `THASK_TOKEN` env, tool_count=24, category tags) for submission to the official Anthropic MCP Registry. New `marketing/show-hn.md` draft with three title options, body, posting checklist, and follow-up channel sequence.

## [0.5.12] - 2026-06-03

### Added
- **`thask doctor`** — single-command diagnostic that walks the full stack (binary version → config file → URL → server reachability → DB + migration version → token validity → token permissions → default team/project → MCP binary on PATH → Claude Code / Cursor / Codex MCP entries). Each check reports `✓` / `⚠` / `✗` with a remediation hint; exit code is `0` on all-pass, `1` when any check is critical. Replaces the previous workflow of correlating 401s, "tools not visible" and silent MCP failures across separate commands.
- **`GET /api/health` enhanced response** (and `/health` alias): now returns `{status, version, db, migrationVersion, migrationsApplied, uptime}` instead of `{"status":"ok"}`. Returns `503` with `dbError` populated when the DB ping fails so monitors can distinguish "server up, DB down" from "fully healthy". `thask doctor` consumes this; external uptime monitors can too. Backend `Version` is overridable via `-ldflags "-X github.com/thask/backend/internal/handler.Version=X.Y.Z"`.
- **`thask init` end-of-run hint** now points users at `thask doctor` for verification alongside the existing `thask guide` reference.

### Docs
- **End-to-end doc refresh for v0.5.9–v0.5.11.** README tagline / Quick Start / Features / Roadmap rewritten to reflect the AI-agent-context-layer positioning, v0.5.9 provenance + permissions, v0.5.10 bulk ops + self-update, and v0.5.11 `thask login`. `docs/ARCHITECTURE.md` now documents the new `internal/audit` package, `AuditRepo` / `SuggestionRepo`, `X-Thask-Client` header parsing, per-key permission gate, and the agent → suggestion-queue → human-approve data flow. `docs/CLI.md` gains `## login`, `## self-update`, an `Outbound Headers` subsection, and a Quick start block. `docs/MCP.md` tools table corrected from 16 → 24 across 7 categories with `parentId` / `assigneeId` on `node.update` and `edge.batch_create` / `edge.batch_delete`. `docs/API.md` clarifies `parentId` / `assigneeId` empty-string semantics on single PATCH + references the `207 Multi-Status` semantics. `docs/CLAUDE_CODE_PLUGIN.md` promotes `thask login` as the standard onboarding flow. `docs/GRAPH.md` gains a new "Provenance & Authoring (v0.5.9+)" section covering field classes, the suggestion queue, and verification.

### Internal
- **Release-gate rule recorded in `CLAUDE.md` + `AGENTS.md`**: docs must be fully up to date before `make release-cli` runs — treat it as a gate, not a follow-up. Pre-release sweep covers CHANGELOG, README, ARCHITECTURE, API.md, CLI.md, MCP.md, DATABASE.md, CLAUDE_CODE_PLUGIN.md, GRAPH.md. CHANGELOG alone is necessary but not sufficient.

## [0.5.11] - 2026-06-01

### Added
- **`thask login` browser-based authentication**: replaces the previous "open the web UI, generate a key, copy the 64-char string, paste it into the terminal" dance with a single command. `thask login` spawns a short-lived loopback HTTP server on a free port in `7400-7500`, opens `<your-thask>/cli/auth?callback_port=<port>&state=<csrf>` in the system browser, waits for the user to click Approve, captures the freshly-minted `user_interactive` API key from the redirect query, and writes it to `~/.thask/config.json`. CSRF protection via random `state`; loopback bound to `127.0.0.1` only (never LAN-reachable); 5-minute timeout; `--force` to replace an existing token; `--url` to override the persisted URL for this login.
- **`/cli/auth` page** (`frontend/src/routes/cli/auth/+page.svelte`): the browser landing target. If the user isn't logged in, transparently redirects to `/login?next=/cli/auth?…` and back. Approve clicks `POST /api/auth/api-keys` (no new backend route — reuses the existing v0.5.9 endpoint) and hands the response to the CLI's loopback server.
- **`/login` `?next=` support**: post-login navigation now honors a `?next=` query (same-origin only, open-redirect guarded) so deep-link flows like `thask login` return to where the user came from instead of always dumping into `/dashboard`.

### Changed
- **`thask mcp serve` missing-token error message** points at `thask login` instead of generically reporting "token not configured" — most MCP users hit this from inside Claude Code where pasting a 64-char string is awkward.
- **URL auto-normalization**: `thask config set url localhost:7243` (and `THASK_URL=localhost:7243`, and `thask login --url localhost:7243`) now silently prepends `http://` so the Go HTTP client stops rejecting the request with `unsupported protocol scheme "localhost"`. Existing scheme-less URLs already stored in `~/.thask/config.json` self-heal on next load.

### Internal
- New CLI dependency: `github.com/pkg/browser` (small, well-tested cross-platform launcher; tarball size unchanged for users since they download platform binaries from GitHub Releases).

### Known Limitations
- **Device flow not implemented.** SSH / headless / CI sessions can't open a browser; they still use `thask config set token <key>` after manual web-UI key creation. A `--device` flag (GitHub-style code entry) is planned for v0.5.12+ if demand surfaces.
- **`/cli/auth` always creates a `user_interactive` key** with the full default permission preset. Agent / service keys still need to go through the web settings UI where the kind picker and per-permission checkboxes live.

## [0.5.10] - 2026-05-26

### Added
- **`thask.node.batch_update` MCP tool + `PATCH /api/projects/:pid/nodes/batch-update`**: partial-update up to 200 nodes in one call. Replaces "loop thask.node.update 67 times" — saves agent context (1 call vs N), writes audit_log rows under one shared `batch_id`, atomic on permission / cycle failure, best-effort per-item on not-found / no-change (returned in `skipped[]`). Server responds **HTTP 207 Multi-Status** when any items skip.
- **`thask.edge.batch_create` + `POST /api/projects/:pid/edges/batch-create`**: insert up to 500 edges in one transaction. Self-references, duplicates (UNIQUE source/target/edgeType), and invalid endpoints land in `skipped[]` with reasons.
- **`thask.edge.batch_delete` + `POST /api/projects/:pid/edges/batch-delete`**: delete up to 500 edges by id. Missing ids → `skipped[]` with `not_found` reason.
- **Whole-project cycle detector** (`detectProjectCycleTx`) runs once at commit time when a batch touches any `parent_id`. O(N) walk over `(id, parent_id)` pairs; first node id in a cycle is reported and the batch rolls back.
- **`thask self-update` CLI command**: fetches the latest published release from GitHub, downloads the platform tarball or `.exe`, extracts the binary, and atomically replaces the running thask binary in place via `os.Rename` within the same directory (POSIX-atomic — concurrent invocations never see a half-written binary). Flags: `--check` (exit non-zero if update available, install nothing), `--version X` (pin to a specific version). Downloads are size-capped at 200 MB.

### Changed
- **`thask.node.update` MCP schema** now exposes `parentId` and `assigneeId`. Empty string unparents / unassigns; omission leaves the field untouched. Backend already accepted these — only the MCP-side schema was missing.
- **`thask.graph.import` description** rewritten to make the create-vs-patch distinction explicit: it creates NEW nodes (with new UUIDs) and does NOT patch existing nodes by id. Validation errors on missing `type`/`title` now point agents at `node.batch_update` (for partial node patches) or `edge.batch_create` (for adding edges between existing nodes).

### Internal
- `EdgeRepo.Pool()` accessor added for the same tx-from-handler pattern `NodeRepo` already uses.
- All bulk endpoints share `newBatchID()` so audit_log rows from a single bulk request carry the same `batch_id`.

### Known Limitations
- Bulk endpoints live on `/api/projects/...` paths and **do not yet honor `Idempotency-Key`**. That middleware is currently wired only on the `/api/v1/...` surface. Idempotent bulk arrives when v1 gets these endpoints (planned v0.5.11+).
- `graph.import mode: "patch"` (id-matched partial updates via the import payload) intentionally deferred — `node.batch_update` covers the same use case without the edge-patching ambiguity.

## [0.5.9] - 2026-05-23

### Added
- **Audit log with 6-dimension provenance**: every write now records `actor_kind` (user_interactive/agent/service/scanner/system), `client_type` + `client_version` (parsed from `X-Thask-Client` header), `agent_model` + `agent_session_id`, `mutation_kind` (structural/semantic/meta), `trigger`, `batch_id`, optional evidence (`code_commit`, `source_path`, `confidence`). Single `audit_log` table replaces `node_history` over a deprecation window — historical rows backfilled in place via migration 010.
- **Node provenance snapshot**: `description_source` (human/agent/scanner/import/unknown), `description_authored_by`, `description_authored_at`, `description_agent_model`, `last_verified_at`, `last_verified_by`, `last_verified_commit`, `field_provenance` JSONB columns on the `nodes` table. Read responses surface these so downstream agents can tell "another agent wrote this, not verified" from "human-authored".
- **Per-API-key permissions**: `api_keys.kind` (`user_interactive` / `agent` / `service`) + `permissions` JSONB with seven flags (`read`, `write_structural`, `write_semantic`, `write_meta`, `verify`, `delete`, `suggest`). Agent-kind keys default to `write_semantic=false`, `verify=false` — they cannot write descriptions or mark nodes verified without an explicit opt-in. Settings UI offers a kind preset selector plus a collapsible per-flag checkbox panel.
- **Suggestion queue**: agents proposing description changes go through `node_suggestions` instead of mutating directly. Pending suggestions are visible to human reviewers; accepting copies the proposed value into the node and credits the deciding human as the author (the agent is only mentioned in audit metadata). Duplicates on the same node+field auto-supersede.
- **`thask.node.suggest_update` / `thask.suggestions.list` / `thask.suggestions.decide` / `thask.node.verify` MCP tools** — the human-in-the-loop pipeline surfaced to AI agents.
- **`X-Thask-Client` header**: CLI sends `thask-cli/<ver>`; MCP sends `thask-mcp/<ver> model=<client> session=<uuid>` (model parsed from MCP `clientInfo`, session is a per-server-instance UUID). Backend parses into audit_log channel fields.
- **Permission denial response format**: `403` with the missing permission flag name and (where relevant) the alternative tool to use, e.g. *"This API key lacks the 'write_semantic' permission. Use thask.node.suggest_update to propose this change for human review."* Denials are themselves recorded in `audit_log` with `action='write_denied'` so key owners can audit attempted writes.
- **Audit coverage extended to all writes**: previously only node create/update/batch_status logged. v0.5.9 covers node delete, node import (with `batch_id`), edge create/update/delete, graph layout, batch deletes, batch positions, batch status — closing the gaps surfaced during the pre-implementation audit.

### Changed
- **`audit.RequirePermission` gates write paths**: every node/edge handler now classifies the operation (structural/semantic/meta) and checks the caller's per-key flag before mutating. Field-level for `node.update` (description → semantic, type/parent_id → structural, status/position/tags → meta).
- **`api_keys` Create signature**: now requires `kind` and `permissions` parameters. Existing keys are migrated to `kind='user_interactive'` with all-true permissions to preserve current behavior.

### Internal
- New `internal/audit` package — lives outside `service` to avoid an import cycle with `middleware`. `Logger.Log()` extracts actor/channel from echo context; `RequirePermission()` returns a 403 echo response and records the denial.
- New `repository/audit.go`, `repository/suggestion.go`, `handler/suggestion.go`, `dto.SuggestNodeUpdateRequest` / `DecideSuggestionRequest` / `VerifyNodeRequest`.
- `NodeRepo.UpdateDescriptionProvenance()` and `NodeRepo.MarkVerified()` stamp the snapshot columns; both are called by handlers after a successful mutation.
- Migrations 006–010 added; `audit_log` indexed on `(project_id, created_at DESC)`, `(entity_type, entity_id)`, `(actor_kind, agent_model)`, and a partial `(batch_id) WHERE batch_id IS NOT NULL`.
- Tests pass: `go test ./...` clean on backend + CLI, `svelte-check` 0 errors on frontend.

### Docs
- `docs/DATABASE.md` — new "Provenance & Audit" section covering `audit_log`, `node_suggestions`, the node provenance columns, the `api_keys.kind/permissions` extension, and the backfill migration.
- `docs/API.md` — `X-Thask-Client` header spec, suggestion queue endpoints, `verify` endpoint, permission denial response shape, extended `Create API Key` body.
- `docs/MCP.md` — four new tools with required permissions, the recommended agent workflow ("read → re-derive from code → suggest_update"), provenance fields in read responses, and a note that agent keys are blocked from `thask.node.update {description}` and `thask.node.verify` by default.

### Improved
- **Edge routing rewrite**: new client-side A\* grid pathfinder (`frontend/src/lib/cytoscape/gridRouter.ts`) routes edges around node and group obstacles with configurable bend / backtrack penalties. `edgeRouter.ts` delegates obstacle avoidance to the grid router and short-circuits the 8-direction waypoint helper for nearly-axis-aligned or 45°-diagonal pairs to avoid degenerate routes.
- **Backend layout simplification**: removed three predict-and-refine passes that probed candidate routes during placement (`refineTopLevelLayersByPredictedRoutes`, `refineLayerOrderByPredictedRoutes`, `refineLayerCentersByPredictedRoutes`) along with the related cleanup. The new client-side grid router handles obstacle avoidance, so the server stops paying the cost of guessing routes that the client will recompute anyway. Net −250 lines across `layout.go`, `layout_child.go`, `layout_child_slots.go`, `layout_group.go`.

## [0.5.8] - 2026-05-11

### Added
- **TypeScript/JavaScript scanner**: `thask scan run --language ts` walks `.ts/.tsx/.js/.jsx/.mjs/.cjs`, parses imports (including multi-line and dynamic `import()`), and emits directory-granular nodes with `depends_on` edges. Auto-detects via `package.json` when `--language` is omitted.
- **Alias resolution**: tsconfig.json / jsconfig.json `compilerOptions.paths`, plus SvelteKit `$lib` auto-detection (`$app`, `$env`, `$service-worker` silently ignored as runtime modules). Resolved paths that escape the scan root are dropped.
- **MCP `language` param**: `thask.scan.run` now accepts `language: "auto" | "go" | "ts"`.
- **`LanguageScanner` interface** in `cli/internal/scan/`: clean extension point for future languages without touching the Go scanner path.

### Improved
- **Landing page**: repositioned as "AI agent context layer" — Before/After block, Works With Claude Code/Cursor/Codex, `thask init` + `/plugin marketplace add` snippets surfaced.
- **Dashboard flicker eliminated**: `+layout.server.ts` seeds the user on first paint (no more "Loading..." flash); inline `<head>` script applies persisted theme before hydration (no light/dark FOUC); login submit no longer races `goto()` against a reactive `$effect`.
- **Edge bridge overlay**: canvas + SVG overlay now clip overflow so bridge arcs stay inside the viewport; same-direction diagonals are correctly skipped while opposite-slope diagonals get a soft-bypass.

### Internal
- `backend/internal/service/layout.go` split from one 8.5k-line file into ten focused files (geometry, crossmin, edge routing, top-level, group, child, slots, lines, boundary) — zero behavior change, all tests pass.

## [0.5.6] - 2026-04-26

### Added
- **Graph image capture**: `GET /api/v1/projects/:id/graph/capture` renders a project as PNG (via Playwright worker) or SVG (server-side). New `thask graph capture` CLI command with `--format/--width/--height/--padding/--scale/--crop/--out` flags
- **Capture worker** (`capture/`): standalone Node.js + Playwright service on port 7241. Connects to a Browserless Chrome instance; opens `frontend/src/routes/capture/+page.svelte` to render Cytoscape and stream the screenshot back. Hardened: read-only filesystem, dropped capabilities, no-new-privileges, optional `CAPTURE_INTERNAL_SECRET`, URL allowlist, body/dimension/scale clamping
- **Edge bridge overlay**: SVG bridges and soft-bypasses drawn at edge crossings so overlapping edges remain readable (`frontend/src/lib/cytoscape/edgeBridgeOverlay.ts`)
- `make dev-services` / `make dev-capture` Makefile targets, `--profile capture` Docker Compose profile

### Improved
- **Layout algorithm**: group-aware orientation selection, corridor routing for inter-group edges, tighter intra-group placement (4,500+ line rewrite of `backend/internal/service/layout.go`)
- Frontend dev server now binds `0.0.0.0` and allows `host.docker.internal`, so the dockerised capture worker can reach it

### Configuration
- New backend env vars: `CAPTURE_URL`, `CAPTURE_INTERNAL_SECRET`, `CAPTURE_TIMEOUT_SECONDS`
- New compose env vars: `CAPTURE_PORT`, `CAPTURE_FRONTEND_URL`, `BROWSER_WS_ENDPOINT`

## [0.5.5] - 2026-04-26

### Added
- **CLI auto-update notification**: every command checks `~/.thask/update-check.json` and prints a yellow stderr alert when a newer GitHub Release exists. Refresh runs in the background and is awaited via a deferred cleanup so the cache reliably updates even on fast commands. Skips in CI, non-TTY, when `THASK_NO_UPDATE_CHECK=1`, or when running `thask mcp serve`

## [0.5.4] - 2026-04-26

### Added
- **Homebrew distribution**: `Formula/thask.rb` is committed in this repo; install via `brew tap kimgh06/thask https://github.com/kimgh06/Thask && brew install kimgh06/thask/thask`
- `make release-cli` automates the full pipeline: parallel cross-platform build → tarballs → SHA256 + formula update → commit/tag/push → npm publish → GitHub Release in one shot. Reads version from `cli/package.json`. npm 2FA bypass-token in `~/.npmrc` removes the OTP prompt

## [0.5.3] - 2026-04-25

### Fixed
- `thask guide` command: replaced `fmt.Println` with `fmt.Print` to fix a `go vet` warning about a redundant trailing newline

## [0.5.2] - 2026-04-19

### Added
- `thask guide` CLI command and `thask.guide` MCP tool — full AI agent interaction guide (single source of truth in `guide.go`)
- `Thask.md` canonical guide document for reference
- Team delete: owner can delete team from Members page UI
- npm distribution: `npm install -g @thask-org/cli` — auto-downloads platform binary from GitHub Releases
- Air hot-reload for backend development (`make dev-backend` auto-rebuilds on `.go` changes)

### Improved
- Layout algorithm: role-aware rectangular child layout with bitmask slot assignment
- Layout algorithm: corridor-aware group repack to avoid long-edge corridors
- Layout algorithm: route-box intersection cleanup — pushes blocking nodes off edge paths
- Layout algorithm: actual group widths used for layer X spacing (`computeLayerXPositions`)
- Layout algorithm: Sugiyama dummy node insertion, sink push, median barycenter, adjacent exchange, Y-blend
- Edge routing: Z-path waypoints for nearly-vertical and nearly-horizontal edges
- Edge routing: parallel edge offset — spread overlapping edges apart
- Panel UX: collapse/expand toggle (`]` shortcut), relations direction (upstream/downstream), activity feed improvements
- Sidebar: collapsible to icon-only mini mode
- Node/edge selection: visual feedback fix, tap priority (node > edge > group)

### Fixed
- `window.__cy` debug exposure now only active in dev builds (`import.meta.env.DEV`)

## [0.5.0] - 2026-03-28

### Added
- CLI tool (Go/Cobra) with full graph operations: node, edge, team, project, graph, impact commands
- MCP server mode for AI agent integration (Claude Code, Cursor) with 12 tools
- Realtime updates via Server-Sent Events (SSE) — 8 event types (node/edge CRUD, layout, import)
- Server-side auto-layout endpoint (`POST /graph/layout`) with dagre and grid algorithms
- CLI `graph layout` command for server-side auto-layout
- API key authentication (`Authorization: Bearer <key>`) alongside session cookies
- Role-based access control: owner, admin, member, viewer with `RequireRole` middleware
- Team member management: invite, role change, remove, leave, transfer ownership
- Team members page in frontend UI
- User settings page in frontend UI
- Shell aliases (`thask aliases install`) for common CLI commands
- `thask install` / `thask uninstall` for system-wide binary installation
- Project sharing: link sharing (viewer/editor mode) with public share URLs
- Project member invitations with editor/viewer roles
- Shared view page with realtime SSE updates and conditional editing
- CLI sharing commands: `project share/unshare/members/invite/kick`
- CLI `graph export` command for JSON file export
- SharedAccess middleware with 30-second cache and rate limiting (5 req/sec)
- Route registration refactored to `routes.go`

## [0.1.0] - 2026-03-20

### Added
- Graph CRUD: 7 node types (FLOW, BRANCH, TASK, BUG, API, UI, GROUP), 4 statuses, 5 edge types
- Interactive graph editor with Cytoscape.js and fCOSE force-directed layout
- Drag-and-drop grouping with compound nodes (collapse/expand, resize)
- Graph export (PNG, JSON) and import (replace/merge mode with transaction support)
- QA Impact Mode with bidirectional BFS and direction-aware edge traversal
- Waterfall status propagation (max depth 10) with parent aggregation
- Node search with pulse highlight animation
- Keyboard shortcuts (N, G, Delete, I, L, Ctrl+Z, etc.)
- Fixed side panel replacing popups: node detail, edge detail, multi-select batch operations
- Session-based authentication with bcrypt (cost 12) and HTTP-only cookies
- Team management with projects
- Team rename with inline edit UI
- Dashboard with project listing and team overview
- Batch operations: position update, status update, delete
- Node history audit log
- Docker Compose deployment (PostgreSQL 17 + Go backend + SvelteKit frontend)
- Playwright E2E tests (13 tests) and Go unit tests (18 tests)
- "Amber Precision" dark design system
