# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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
