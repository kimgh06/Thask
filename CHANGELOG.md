# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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
