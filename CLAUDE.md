# Thask — Project Instructions for AI Agents

Canonical agent guide: [`Thask.md`](Thask.md) (MCP `thask.guide` / CLI `thask guide`).
This file holds **project-level conventions** that the canonical guide does not cover.

## Release tracking convention

When a new version of Thask is published to npm / GitHub, record it in the prod
Thask graph (project `32919acf-2a6f-4249-819f-e5bc153246aa`) using this pattern:

1. Create a `TASK` node titled `Release vX.Y.Z`, status `PASS`, tag `release`.
   Description = the matching `CHANGELOG.md` section (full markdown, including npm + GitHub Release links at top).
2. For every existing node this release ships (e.g. a feature TASK that was `IN_PROGRESS`), update its status to `PASS`.
3. Add a `related` edge from the release node to each shipped node with `label: "ships"`.
4. Call `thask.graph.layout(dagre)` once at the end.

Why: `thask.impact.analyze --node <release-id>` then answers "what areas did this release touch" without re-reading commits. CHANGELOG.md stays the human-facing source of truth; the graph adds the structural view for agents.

## Build / test / release

- Backend: `cd backend && go build ./... && go test ./...`
- Frontend: `cd frontend && npx svelte-check`
- CLI: `cd cli && go build ./... && go test ./...`
- Code generation: `make sqlc-gen` (whenever you touch `backend/db/queries/*.sql`
  or `backend/migrations/*.sql`). `make sqlc-check` validates queries against
  the schema without writing anything — useful in CI.
- Release pipeline (needs npm OTP): `make release-cli CLI_VERSION=X.Y.Z THASKOTP=<6-digit>`

## Release rules

- **Docs must be fully up to date before `make release-cli` runs.** Treat
  it as a gate, not a follow-up. Pre-release sweep covers:
  CHANGELOG, README, ARCHITECTURE, API.md, CLI.md, MCP.md, DATABASE.md,
  CLAUDE_CODE_PLUGIN.md, GRAPH.md. CHANGELOG alone is necessary but not
  sufficient — README and ARCHITECTURE are the easy-to-forget ones.
- If you're blocked on an external dependency mid-release (npm token,
  OTP, gh auth), use the wait window to do the doc sweep rather than
  skipping it.
- **`backend/internal/dbgen/` must be regenerated and committed alongside
  any change to `backend/db/queries/*.sql` or `backend/migrations/*.sql`.**
  CI doesn't run `sqlc generate` — the diff is the contract. If `make
  sqlc-check` passes but the generated files are stale in git, builds
  will reference old types and fail unpredictably.

## Commit rules

- Never commit without an explicit per-task user request. Prior authorization does not carry over.
- Split commits by functional unit (refactor / feat / fix / docs separately), not bundled.
- Run code review before committing when the user asks for a commit.

## Pitfalls (project-specific)

- `git push` may fail with 403 if credentials cache holds the wrong GitHub account. Retry once before reporting blocked.
- `cli/package.json` is the version source of truth. Platform packages under `npm/cli-*/` are stale (v0.5.0) and not auto-bumped — leave them alone.
- The published MCP server lags behind local CLI code by one release. To test newly added MCP tools / params, run the local binary directly, not via Claude Code MCP.
