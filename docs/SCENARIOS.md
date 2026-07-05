# Thask — Scenario Inventory

Living catalog of every observable behavior in Thask, derived directly from
the source (routes, CLI commands, MCP tools, frontend routes). Each scenario
has an ID so E2E tests, manual QA, and bug reports can reference precisely.

**Convention:**
- `H` = Human user via web UI
- `C` = Human via CLI
- `A` = AI agent via MCP
- `S` = External system / cron / scanner
- Scenarios are positive-path **and** known failure modes.
- `→` means "expected outcome / observable result".

**Date of last full audit:** 2026-07-04 (v0.6.0 release).

---

## 1. Authentication & Identity

### 1.1 Session-based (cookie)

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| AUTH-01 | H | `POST /api/auth/register` with new email/password | 201 with session cookie set, `User` row inserted |
| AUTH-02 | H | `POST /api/auth/register` with duplicate email | 400/409, no insert |
| AUTH-03 | H | `POST /api/auth/login` with valid creds | 200 + session cookie, `Session` row inserted with future `expires_at` |
| AUTH-04 | H | `POST /api/auth/login` with wrong password | 401, no session created |
| AUTH-05 | H | `GET /api/auth/me` with valid cookie | 200 with `{id, email, displayName}` |
| AUTH-06 | H | `GET /api/auth/me` with no cookie | 401 |
| AUTH-07 | H | `POST /api/auth/logout` | 200, session row deleted, cookie cleared |
| AUTH-08 | S | `Session.expires_at < now()` accessed | 401, expired row cleaned by background job |

### 1.2 API key (Bearer)

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| AUTH-09 | H | `POST /api/auth/api-keys` with `{name, kind: user_interactive}` | 201, plain key returned ONCE in response, hash stored |
| AUTH-10 | H | `POST /api/auth/api-keys` with `kind: agent` (default permissions) | 201, `write_semantic=false`, `verify=false`, others true |
| AUTH-11 | H | `POST /api/auth/api-keys` with `kind: service` + custom permissions JSON | 201, permissions stored as supplied |
| AUTH-12 | H | `GET /api/auth/api-keys` | 200 list (no plain key, only prefix + metadata + kind + permissions) |
| AUTH-13 | H | `DELETE /api/auth/api-keys/:keyId` | 200, row removed, subsequent requests with that key 401 |
| AUTH-14 | A | Bearer header with valid `agent` key, mutation=structural | 200, audit_log records `actor_kind=agent` + `client_type` + `agent_model` |
| AUTH-15 | A | Bearer header with `agent` key, attempts description PATCH (semantic) | 403 with `"required":"write_semantic"`, audit_log records `action=write_denied` |
| AUTH-16 | A | Bearer header with `agent` key, attempts `POST /verify` | 403 with `"required":"verify"` |
| AUTH-17 | S | Bearer header with expired/revoked key | 401 |

### 1.3 CLI login flow (Pattern A — loopback)

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| AUTH-18 | C | `thask login` (no existing token) — browser launches | Loopback HTTP server on port 7400-7500, browser opens `/cli/auth?callback_port=...&state=...` |
| AUTH-19 | C | User not logged in on web → page redirects to `/login?next=...` | After login, browser lands on `/cli/auth`, Approve button visible |
| AUTH-20 | C | User clicks Approve | API key created via `POST /api/auth/api-keys`, redirect to `localhost:<port>?token=...&state=...`, CLI captures, saves to `~/.thask/config.json`, prints "✓ Logged in" |
| AUTH-21 | C | User clicks Cancel | Redirect with `error=denied`, CLI prints denial + exit non-zero |
| AUTH-22 | C | `thask login --force` with existing token | Replaces token, prior token still valid until manually revoked |
| AUTH-23 | C | `thask login` times out (5 minutes) | Local server shutdown, friendly message |
| AUTH-24 | C | CSRF: callback URL has wrong `state` | "Security check failed (state mismatch)", no token saved |
| AUTH-25 | C | Port range 7400-7500 fully occupied | "All ports busy" message, exit non-zero |
| AUTH-26 | C | `thask login` with no URL in config | "Run `thask config set url <url>` first" |

---

## 2. Teams

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| TEAM-01 | H | `POST /api/teams` with `{name, slug}` | 201, current user becomes Owner |
| TEAM-02 | H | `POST /api/teams` with duplicate slug | 400/409 |
| TEAM-03 | H | `GET /api/teams` | 200 list of teams user belongs to (any role) |
| TEAM-04 | H | `GET /api/teams/:slug` as member | 200 with `name`, `members[]`, `projects[]` |
| TEAM-05 | H | `GET /api/teams/:slug` as non-member | 403 |
| TEAM-06 | H | `GET /api/teams/:slug/members` | 200 with email + role per member |
| TEAM-07 | H | `GET /api/teams/:slug/projects` | 200 with project list |
| TEAM-08 | H | `POST /api/teams/:slug/leave` as member | 200, team_members row removed |
| TEAM-09 | H | `POST /api/teams/:slug/leave` as last owner | 400 ("transfer ownership first") |
| TEAM-10 | H | `PATCH /api/teams/:slug` as admin | 200, name updated |
| TEAM-11 | H | `PATCH /api/teams/:slug` as member | 403 |
| TEAM-12 | H | `POST /api/teams/:slug/members` (invite email) as admin | 201, team_members row added with role |
| TEAM-13 | H | `PATCH /api/teams/:slug/members/:userId` (role change) as admin | 200 |
| TEAM-14 | H | `DELETE /api/teams/:slug/members/:userId` as admin | 200 |
| TEAM-15 | H | `DELETE /api/teams/:slug` as owner | 200, cascade deletes projects + nodes + edges |
| TEAM-16 | H | `DELETE /api/teams/:slug` as admin | 403 |
| TEAM-17 | H | `POST /api/teams/:slug/transfer` (owner→other) as owner | 200, roles swap |
| TEAM-18 | H | `POST /api/teams/:slug/projects` as member+ | 201 |

---

## 3. Projects

### 3.1 CRUD

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| PROJ-01 | H | `POST /api/teams/:slug/projects` with `{name, description}` | 201 |
| PROJ-02 | H | `GET /api/projects/:pid` as team member | 200 with metadata + sharing state |
| PROJ-03 | H | `GET /api/projects/:pid` as non-member | 403 |
| PROJ-04 | H | `PATCH /api/projects/:pid` (rename) as member+ | 200 |
| PROJ-05 | H | `DELETE /api/projects/:pid` as member+ | 200, cascades nodes/edges/history/audit |
| PROJ-06 | H | `GET /api/projects/summary` | 200, list of projects with counts (per user) |

### 3.2 Sharing (admin+)

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| PROJ-07 | H | `GET /api/projects/:pid/sharing` | 200 with `linkSharing`, `shareToken` (plain, once), members[] |
| PROJ-08 | H | `PUT /api/projects/:pid/sharing` with `{linkSharing: "viewer"}` | 200, token generated and returned |
| PROJ-09 | H | `PUT /api/projects/:pid/sharing` with `{linkSharing: "editor"}` | 200, write-enabled shared mode |
| PROJ-10 | H | `PUT /api/projects/:pid/sharing` with `{linkSharing: "off"}` | 200, share_token cleared |
| PROJ-11 | H | `POST /api/projects/:pid/sharing/members` (invite by email) | 201, project_members row added |
| PROJ-12 | H | `PATCH /api/projects/:pid/sharing/members/:userId` (role) | 200 |
| PROJ-13 | H | `DELETE /api/projects/:pid/sharing/members/:userId` | 200 |

### 3.3 Public shared access (no auth)

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| PROJ-14 | S | `GET /api/shared/:token` valid token | 200 with project metadata |
| PROJ-15 | S | `GET /api/shared/:token` invalid/expired token | 404 |
| PROJ-16 | S | Rate limit: >5 req/s from same IP | 429 |
| PROJ-17 | S | `GET /api/shared/:token/graph` viewer mode | 200 read-only graph |
| PROJ-18 | S | `POST /api/shared/:token/nodes` viewer mode | 403 |
| PROJ-19 | S | `POST /api/shared/:token/nodes` editor mode | 201 (anonymous user, `created_by=NULL`) |
| PROJ-20 | S | `GET /api/shared/:token/og-image` | 200 PNG image stream |
| PROJ-21 | S | `GET /api/shared/:token/events` | SSE stream, public events only |

---

## 4. Nodes

### 4.1 Read

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| NODE-01 | H/A | `GET /api/projects/:pid/nodes` | 200 with `creatorEmail` per node (LEFT JOIN users) |
| NODE-02 | H/A | `GET /api/projects/:pid/nodes?type=TASK` | filtered list |
| NODE-03 | H/A | `GET /api/projects/:pid/nodes?status=FAIL` | filtered list |
| NODE-04 | H/A | `GET /api/projects/:pid/nodes?cursor=...&limit=20` | paginated, cursor (createdAt, id) keyset |
| NODE-05 | H/A | `GET /api/projects/:pid/nodes/:nid` | 200 with full Node + `creatorEmail` |
| NODE-06 | H/A | `GET /api/projects/:pid/nodes/:nid` non-existent | 404 |

### 4.2 Create

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| NODE-07 | H/A | `POST /api/projects/:pid/nodes` with `{type, title, status, tags, description?}` | 201, `createdBy` set from session/key user, audit_log entry |
| NODE-08 | A | Create with description and agent key lacking `write_semantic` | 403 |
| NODE-09 | H/A | Create with invalid `type` enum | 400 |
| NODE-10 | H/A | Create with `parentId` cycle attempt | 400 (cycle detected) |
| NODE-11 | S | Shared editor anonymous create | 201, `createdBy=NULL` (anonymous) |

### 4.3 Update (PATCH)

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| NODE-12 | H/A | `PATCH /api/projects/:pid/nodes/:nid` with `{title, status, tags}` | 200, response includes new values + `creatorEmail` via re-fetch |
| NODE-13 | A | Update description (semantic) with agent key | 403 |
| NODE-14 | H/A | Update with `parentId` creating cycle | 400 |
| NODE-15 | H/A | Update with `parentId: null` | 200, detached from parent |
| NODE-16 | H/A | Update triggers waterfall (FAIL propagates downstream) | downstream nodes auto-updated via BFS, audit_log records cascade |

### 4.4 Delete

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| NODE-17 | H/A | `DELETE /api/projects/:pid/nodes/:nid` | 200, children un-parented, connected edges removed, node removed |
| NODE-18 | H/A | Delete non-existent | 404 |

### 4.5 Batch

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| NODE-19 | H/A | `PATCH /api/projects/:pid/nodes/positions` with `[{id, x, y, width?, height?}, ...]` | 200, pgx.Batch pipelined updates |
| NODE-20 | H/A | `POST /api/projects/:pid/nodes/batch-delete` with `{ids: [...]}` | 200, multi-statement transaction |
| NODE-21 | H/A | `PATCH /api/projects/:pid/nodes/batch-status` with `{ids, status}` | 200 |
| NODE-22 | H/A | `PATCH /api/projects/:pid/nodes/batch-update` (v0.5.10, up to 200) | 200 or **207 Multi-Status** when any items skip (per-item `skipped[]` reasons) |
| NODE-23 | A | batch-update with one item missing permission | 207 with `skipped[]` reason `forbidden:write_semantic` |
| NODE-24 | A | batch-update with cycle on one item | atomic rollback OR 207 (see code; current = atomic on permission/cycle/db failure) |

### 4.6 Provenance writes

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| NODE-25 | H/A | Description PATCH | `description_source`, `_authored_by`, `_authored_at`, `_agent_model` columns updated |
| NODE-26 | H | `POST /api/projects/:pid/nodes/:nid/verify` with `{commit?}` (human only) | 200 with rows affected, `last_verified_at/by/commit` updated |
| NODE-27 | A | `verify` attempted with agent key (default permissions) | 403 |
| NODE-28 | H/A | Verify non-existent node | 404 |

---

## 5. Edges

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| EDGE-01 | H/A | `GET /api/projects/:pid/edges` | 200 list |
| EDGE-02 | H/A | `POST /api/projects/:pid/edges` with `{sourceId, targetId, edgeType, label?, waypoints?}` | 201 |
| EDGE-03 | H/A | Create edge with sourceId == targetId | 400 (no self-edges) |
| EDGE-04 | H/A | Create edge across projects | 400 (cross-project) |
| EDGE-05 | H/A | `PATCH /api/projects/:pid/edges/:eid` with `{edgeType?, waypoints?, label?, sourcePort?, targetPort?}` | 200 (COALESCE for optional fields) |
| EDGE-06 | H/A | `DELETE /api/projects/:pid/edges/:eid` | 200 |
| EDGE-07 | H/A | `POST /api/projects/:pid/edges/batch-create` (v0.5.10, up to 500) | 201 or 207. **v0.5.15:** items with `waypoints == nil` are normalised to `[]` so the `edges.waypoints JSONB NOT NULL DEFAULT '[]'` constraint never aborts the whole tx. |
| EDGE-08 | H/A | `POST /api/projects/:pid/edges/batch-delete` | 200 or 207 |
| EDGE-09 | H/A | depends_on edge created on FAIL-status source | waterfall triggers downstream cascade |
| EDGE-10 | H/A | blocks edge enforces directional impact analysis |

---

## 6. Graph operations

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| GRAPH-01 | H/A | `GET /api/projects/:pid/graph` | 200 with `{nodes, edges}` arrays, `creatorEmail` populated |
| GRAPH-02 | H/A | `POST /api/projects/:pid/graph/import` with `merge` mode | 200, nodes upserted by title (or id), edges added |
| GRAPH-03 | H/A | `POST /api/projects/:pid/graph/import` with `replace` mode | 200, all existing nodes/edges deleted first, then inserted |
| GRAPH-04 | H/A | `POST /api/projects/:pid/graph/layout` with `{algorithm: dagre\|grid}` | 200, all positions updated, GROUP auto-sized |
| GRAPH-05 | H/A | `GET /api/projects/:pid/graph/analyze` | 200 with `{cycles: [...], criticalPath: [...]}` (Tarjan SCC, longest depends_on/blocks chain) |
| GRAPH-06 | H/A | `GET /api/projects/:pid/impact?nodeId=...` | 200 with `{changedNodes, affectedNodeIds}` (bidirectional BFS) |
| GRAPH-07 | H/A | Impact analysis on isolated node | empty `affectedNodeIds` |

---

## 7. Suggestions queue

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| SUGG-01 | A | `POST /api/projects/:pid/nodes/:nid/suggestions` with `{field, proposedValue, rationale?, evidence?}` | 201, row in `node_suggestions` with `status=pending` |
| SUGG-02 | H | `GET /api/projects/:pid/suggestions` | 200 list filtered to pending by default |
| SUGG-03 | H | `PATCH /api/projects/:pid/suggestions/:sid` with `{decision: accepted}` | 200, target node's field updated to `proposedValue`, suggestion `status=accepted`, audit_log records human as authoring user |
| SUGG-04 | H | `PATCH /api/projects/:pid/suggestions/:sid` with `{decision: rejected, reason}` | 200, `status=rejected`, no node mutation |
| SUGG-05 | A | Attempt `accepted` decision via agent key | 403 (server-enforced `actor_kind=user_interactive`) |
| SUGG-06 | H | New suggestion for same node+field while a pending one exists | older one `status=superseded` |

---

## 8. Activity feed & SSE

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| ACTIVITY-01 | H | `GET /api/projects/:pid/activity?limit=20` | 200 list of recent history + audit entries |
| ACTIVITY-02 | H/A | `GET /api/projects/:pid/events` (SSE) | text/event-stream, sends `node_created`, `node_updated`, `edge_created`, etc. as they occur |
| ACTIVITY-03 | H | Multiple clients on same project | each receives same broadcasts |
| ACTIVITY-04 | H | Client disconnects mid-stream | hub cleans up subscription |

---

## 9. Health & meta

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| HEALTH-01 | S | `GET /api/health` (or `/health`) DB up | 200 `{status:"ok", version, db:"ok", migrationVersion, migrationsApplied, uptime}` |
| HEALTH-02 | S | DB ping fails | 503 with `dbError:"db ping failed"` (no DSN leak) |
| HEALTH-03 | S | `schema_migrations` table missing/unreadable | 200 `degraded`, `dbError:"schema_migrations unreadable"` |
| HEALTH-04 | S | Any `/api/*` response (v0.5.15+) | Echo middleware stamps `X-Thask-Server-Version: <semver>` header on every reply; CLI telemetry records it as `backend_version`. Static and non-API routes are excluded. |

---

## 10. CLI scenarios

### 10.1 Meta

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-01 | C | `thask --version` | stdout `thask v0.5.15 (<sha>)`, exit 0 |
| CLI-02 | C | `thask -v` | same as CLI-01 |
| CLI-03 | C | `thask version` (subcmd) | same output |
| CLI-04 | C | `thask --help` | usage info, exit 0 |
| CLI-05 | C | `thask --bogus-flag` | stderr `Error: unknown flag...\nRun 'thask --help' for usage.`, exit 2 |
| CLI-06 | C | `thask nodd` (typo) | stderr unknown command + "Did you mean: node" suggestion, exit 2 |
| CLI-07 | C | `thask node list` with unset token | stdout JSON `{"error":"Authentication required"}`, exit 1 |
| CLI-08 | C | `thask doctor` | 14 checks ✓/⚠/✗ + remediation hints, exit 0 all-pass / 1 critical |
| CLI-09 | C | `thask self-update` | downloads latest tarball, replaces binary, prints version |
| CLI-10 | C | `thask init` | sets default config, prints `thask doctor` hint |
| CLI-11 | C | `thask guide` | prints embedded agent guide markdown |
| CLI-12 | C | `thask mistake record --cause "..." --fix "..."` | inserts mistake row + memory |

### 10.2 Config

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-13 | C | `thask config set url localhost:7244` (no scheme) | persists `http://localhost:7244` (auto-normalize) |
| CLI-14 | C | `thask config set token thsk_...` | saved to `~/.thask/config.json` (mode 0600) |
| CLI-15 | C | `thask config show` | prints url, token prefix, team, project |

### 10.3 Output format

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-16 | C | `thask node list --format json` (default) | stdout JSON array |
| CLI-17 | C | `thask node list --format table` or `--pretty` | column-aligned text |
| CLI-18 | C | `thask node list --format quiet` or `-q` | one ID per line |

### 10.4 Aliases & shell install

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-19 | C | `thask aliases show` | prints recommended shell aliases |
| CLI-20 | C | `thask aliases install` | writes aliases to `~/.zshrc` or `~/.bashrc` (idempotent) |
| CLI-21 | C | `thask aliases uninstall` | removes block |

### 10.5 Domain commands (mirror REST)

For each: `--url` `--token` `-p` `--team` `-f` `--pretty` `-q` flags compose.

| ID | Command | Maps to |
|---|---|---|
| CLI-22 | `thask auth whoami` | `GET /api/auth/me` |
| CLI-23 | `thask team list` | `GET /api/teams` |
| CLI-24 | `thask team create --name ... --slug ...` | `POST /api/teams` |
| CLI-25 | `thask project list/get/create` | `/api/teams/:slug/projects` |
| CLI-26 | `thask project share --mode viewer\|editor` | `PUT /api/projects/:pid/sharing` |
| CLI-27 | `thask project unshare` | `PUT sharing linkSharing=off` |
| CLI-28 | `thask project members` | `GET sharing` |
| CLI-29 | `thask project invite --email ...` | `POST sharing/members` |
| CLI-30 | `thask project kick --user ...` | `DELETE sharing/members/:uid` |
| CLI-31 | `thask node list/get/create/update/delete/batch-status` | `/nodes` family |
| CLI-32 | `thask edge list/create/update/delete` | `/edges` family |
| CLI-33 | `thask graph get` | `/graph` |
| CLI-34 | `thask graph export --format md` | client-side render to markdown |
| CLI-35 | `thask graph export --format json` | server `/graph` raw |
| CLI-36 | `thask graph export --format png` | capture endpoint, PNG stream |
| CLI-37 | `thask graph import --file ... --mode merge\|replace` | `/graph/import` |
| CLI-38 | `thask graph layout --algorithm dagre\|grid` | `/graph/layout` |
| CLI-39 | `thask graph analyze` | `/graph/analyze` |
| CLI-40 | `thask graph capture` | local capture worker invocation |
| CLI-41 | `thask impact --node ...` | `/impact?nodeId=` |
| CLI-42 | `thask scan --path ./...` | local scanner (Go/TS) → `/graph/import` merge |
| CLI-60 | `thask edge batch-create --file ./edges.json` (v0.5.16) | `POST /edges/batch-create`, 201 or 207 |
| CLI-61 | `thask edge batch-delete --file ./ids.json` (v0.5.16) | `POST /edges/batch-delete`, 200 or 207 |
| CLI-62 | `thask node batch-update --file ./updates.json` (v0.5.16) | `PATCH /nodes/batch-update`, 200 or 207 |
| CLI-63 | `thask suggestions list [--status pending\|accepted\|rejected] [--limit N]` (v0.5.16) | `GET /suggestions[?status=&limit=]` |
| CLI-64 | `thask suggestions decide <id> --accept\|--reject [--reason ...]` (v0.5.16) | `PATCH /suggestions/:sid` with `{status,reason?}`; server enforces actor_kind=user_interactive — agent keys 403 |

### 10.6 MCP serve

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-43 | C | `thask mcp serve` | starts MCP stdio loop, no human output to stdout (JSON-RPC only) |
| CLI-44 | C | `mcp serve` with missing token | nice error message pointing at `thask login` |
| CLI-45 | C | `mcp serve` with --url override | uses that URL |

### 10.7 Login

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-46 | C | `thask login` | loopback flow as in AUTH-18~26 |
| CLI-47 | C | `thask login --force` | replaces existing token |

### 10.8 Local-first telemetry (v0.5.15+)

Backing file: `~/.thask/events.jsonl` (append-only, single explicit `purge` rewrites via temp file + `os.Rename`). Bodies are opt-in via `capture_payloads`; redaction is applied at write time. See [SYS-10](#13-background--system) for the write path.

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| CLI-48 | C | `thask usage` | 30-day summary: invocation count, success %, p50/p95 latency, top commands, backends |
| CLI-49 | C | `thask usage top --limit 5 --since "1 week ago"` | command / count / avg-ms / ok% table for the window |
| CLI-50 | C | `thask usage errors` | failure funnel grouped by `error_class` (auth / network / timeout / server / usage / other) |
| CLI-51 | C | `thask reflog -n 20 [--source mcp\|--errors-only\|--since "yesterday"]` | most-recent invocations in time-reverse order |
| CLI-52 | C | `thask history` | alias for `reflog` |
| CLI-53 | C | `thask reflog --show <id-prefix>` | pretty-printed full event (any kind) by ULID prefix |
| CLI-54 | C | `thask reflog search "<q>"` | `bytes.Contains` grep over raw lines, pretty-prints matches |
| CLI-55 | C | `thask telemetry status` | collection state, payload opt-in flag, install_id, data dir |
| CLI-56 | C | `thask telemetry show schema\|last` | event schema + redaction policy, or most-recent raw event |
| CLI-57 | C | `thask telemetry config set capture_payloads <bool>` | flips opt-in flag, emits 1 `config_change` event |
| CLI-58 | C | `thask telemetry disable` then any command × N | exactly 1 `config_change` row recorded (the disable itself), then 0 events on subsequent runs while the tombstone exists; `enable` lifts it |
| CLI-59 | C | `thask telemetry purge --before "30 days ago" \| --all \| --payloads-only` | atomic rewrite via temp file + `os.Rename`; only path outside append-only invariant |

---

## 11. MCP tools (25)

Each MCP tool is invoked via `thask mcp serve` stdio + JSON-RPC `tools/call`.

| ID | Tool | Maps to | Notes |
|---|---|---|---|
| MCP-01 | `thask.guide` | embedded guide | Returns agent operating manual |
| MCP-02 | `thask.node.list` | `GET /nodes` | type/status filter, pagination |
| MCP-03 | `thask.node.get` | `GET /nodes/:nid` | |
| MCP-04 | `thask.node.create` | `POST /nodes` | description=>semantic check |
| MCP-05 | `thask.node.update` | `PATCH /nodes/:nid` | |
| MCP-06 | `thask.node.delete` | `DELETE /nodes/:nid` | |
| MCP-07 | `thask.node.batch_status` | `PATCH /nodes/batch-status` | |
| MCP-08 | `thask.node.batch_update` | `PATCH /nodes/batch-update` (v0.5.10) | up to 200 |
| MCP-09 | `thask.node.suggest_update` | `POST /nodes/:nid/suggestions` | semantic write fallback |
| MCP-10 | `thask.node.verify` | `POST /nodes/:nid/verify` | requires `verify` permission |
| MCP-11 | `thask.edge.list` | `GET /edges` | |
| MCP-12 | `thask.edge.create` | `POST /edges` | |
| MCP-13 | `thask.edge.delete` | `DELETE /edges/:eid` | |
| MCP-25 | `thask.edge.update` | `PATCH /edges/:eid` (v0.5.16) | server COALESCE — unsupplied fields kept |
| MCP-14 | `thask.edge.batch_create` | `POST /edges/batch-create` (v0.5.10) | up to 500 |
| MCP-15 | `thask.edge.batch_delete` | `POST /edges/batch-delete` | up to 500 |
| MCP-16 | `thask.graph.get` | `GET /graph` | full nodes+edges payload |
| MCP-17 | `thask.graph.import` | `POST /graph/import` | mode=merge default |
| MCP-18 | `thask.graph.layout` | `POST /graph/layout` | dagre/grid |
| MCP-19 | `thask.graph.analyze` | `GET /graph/analyze` | cycles + critical path |
| MCP-20 | `thask.impact.analyze` | `GET /impact?nodeId=` | bidirectional BFS |
| MCP-21 | `thask.scan.run` | local scanner | Go/TS dependency graph → import |
| MCP-22 | `thask.suggestions.list` | `GET /suggestions` | filter status |
| MCP-23 | `thask.suggestions.decide` | `PATCH /suggestions/:sid` | server enforces actor_kind=user_interactive |
| MCP-24 | `thask.mistake.record` | inserts mistake + memory entry | feedback loop |

---

## 12. Frontend pages (13)

| ID | Route | Scenario | Expected |
|---|---|---|---|
| FE-01 | `/` | Marketing landing | Static page, no auth required |
| FE-02 | `/register` | New account form | submit → AUTH-01 |
| FE-03 | `/login` | Login form | submit → AUTH-03 |
| FE-04 | `/login?next=/cli/auth?...` | Login with redirect param | After login, navigates to `next` (same-origin only) |
| FE-05 | `/dashboard` | Dashboard home (auth required) | List teams with selectable cards |
| FE-06 | `/dashboard/settings` | User settings page | API keys CRUD, profile edit, theme toggle |
| FE-07 | `/dashboard/[teamSlug]` | Team page | List projects, member count |
| FE-08 | `/dashboard/[teamSlug]/members` | Team member management | Invite, role change, remove (admin+) |
| FE-09 | `/dashboard/[teamSlug]/[projectId]` | Graph editor | Cytoscape canvas, side panel, search, toolbars |
| FE-10 | `/cli/auth?callback_port=...&state=...` | CLI auth approval page | AUTH-19~26 |
| FE-11 | `/shared/[shareToken]` | Public shared view | viewer/editor modes, realtime via SSE |
| FE-12 | `/embed/[shareToken]` | Embeddable iframe view | Minimal chrome, dark/light auto |
| FE-13 | `/docs` | Internal documentation page | Renders selected docs/* files |
| FE-14 | `/capture` | Capture trigger endpoint (worker uses) | Worker iframe loads this to snapshot graph |

### 12.1 Graph editor (FE-09) sub-scenarios

| ID | Scenario | Expected |
|---|---|---|
| FE-09.01 | Pan canvas with V/Space hold or middle mouse | Cytoscape pan mode |
| FE-09.02 | Drag to create node (toolbar) | AddNodeModal → POST nodes |
| FE-09.03 | Hover node, drag edge handle to another | POST edge |
| FE-09.04 | Double-click GROUP node | Collapse/expand with child count badge |
| FE-09.05 | Drag node into GROUP | parent_id set, group resizes |
| FE-09.06 | Shift+drag | multi-select (roadmap) |
| FE-09.07 | Click node → DetailPanel slide out | shows description (markdown), type/status, tags, creator footer, history |
| FE-09.08 | Edit description in panel (markdown) | PATCH on blur, audit_log written |
| FE-09.09 | Impact Mode toggle (button) | dim non-affected nodes, highlight changed + downstream |
| FE-09.10 | Analysis Mode (Shift+A) | cycles outlined red, critical path highlighted |
| FE-09.11 | Search bar | filter nodes by title/tag |
| FE-09.12 | Filter by type/status (toolbar) | hide non-matching |
| FE-09.13 | Share dialog (admin+) | toggle linkSharing, copy share URL |
| FE-09.14 | Activity feed open | recent changes with user attribution |
| FE-09.15 | Template selector on empty project | apply built-in template (4 incl. handoff-starter) |
| FE-09.16 | Realtime: another user creates node | SSE event → canvas updates without refresh |
| FE-09.17 | Theme toggle (system / light / dark) | persists per user |

---

## 13. Background / system

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| SYS-01 | S | Container startup | `migrate.Run()` applies pending migrations in order, advances `schema_migrations.version` |
| SYS-02 | S | Migration 011 backfill on large `nodes` table | runs as single UPDATE — known lock window on >100k rows (documented) |
| SYS-03 | S | Session cleanup tick | expired rows deleted |
| SYS-04 | S | Idempotency cache eviction | rows older than TTL removed |
| SYS-05 | S | EventHub: project subscribers | each Publish() fan-outs to all SSE clients of that project |
| SYS-06 | S | Waterfall propagation | BFS up to depth 10, audit_log records cascade per affected node |
| SYS-07 | S | Capture worker | Playwright Chromium loads `/capture?...&token=...`, snapshots PNG/SVG, posts back |
| SYS-08 | S | sqlc generate | `make sqlc-gen` regenerates `backend/internal/dbgen/`; idempotent (re-run produces zero diff) |
| SYS-09 | S | sqlc compile-check | `make sqlc-check` validates queries against migrations without writing files |
| SYS-10 | C | **CLI telemetry write path (v0.5.15+).** Each CLI invocation → `cli/internal/telemetry/log.go Log()` → POSIX `O_APPEND` write under 4 KB to `~/.thask/events.jsonl` (multi-process safe via PIPE_BUF atomicity). Redaction (`--token`, `Authorization: Bearer`, URL credentials, JWT, Cookie, `thsk_`-prefixed strings) runs at marshal time. Failure modes — missing dir, permission denied, disk full, internal panic — are swallowed via `defer recover()` so the CLI's exit code is unaffected. | One row appended (or zero if tombstone exists / unrecoverable I/O); CLI behaviour unchanged regardless. |

---

## 14. Error / edge cases (cross-cutting)

| ID | Category | Scenario | Expected |
|---|---|---|---|
| ERR-01 | Auth | API key with `permissions.write_semantic=false` PATCHes description | 403 with `{"error","required":"write_semantic","current_permissions":{...}}` |
| ERR-02 | Auth | Session cookie + Bearer header both present | Bearer wins (more specific) |
| ERR-03 | Validation | Required field missing on POST | 400 with field name |
| ERR-04 | Validation | Type enum invalid | 400 |
| ERR-05 | Cycle | Parent cycle on create | 400 |
| ERR-06 | Cycle | Parent cycle on update | 400 |
| ERR-07 | Cycle | depends_on cycle reported by `/graph/analyze` | reported in `cycles[]`, not blocked |
| ERR-08 | Permission | Member tries owner-only op | 403 |
| ERR-09 | Permission | Non-member tries any project op | 403 |
| ERR-10 | Permission | Anonymous tries `viewer` share write | 403 |
| ERR-11 | Rate limit | Shared endpoint >5 req/s | 429 |
| ERR-12 | Body size | v1 request exceeds `MAX_REQUEST_BODY_BYTES` | 413 |
| ERR-13 | CORS | v1 request from disallowed origin | preflight 403 |
| ERR-14 | Idempotency | v1 retry with same `Idempotency-Key` | cached response replayed |
| ERR-15 | Audit | Permission denial recorded | audit_log entry `action=write_denied` |
| ERR-16 | DB | Connection pool exhausted | 503 (handler returns "service unavailable") |
| ERR-17 | DB | unique constraint violation | 409 |
| ERR-18 | CLI | Usage errors: unknown flag / unknown command / wrong arg count (`accepts N arg(s)`, `flag needs an argument`, `requires at least`, `accepts at most`, `accepts no arg(s)`) / pflag `strconv.*` parse failures | human stderr + exit 2 (v0.5.14 introduced, v0.5.15 widened the matcher) |
| ERR-19 | CLI | Runtime auth fail | JSON stderr + exit 1 |
| ERR-20 | MCP | Tool input schema fails validation | JSON-RPC error response with details |
| ERR-21 | MCP | Backend unreachable from MCP context | JSON-RPC error mapping HTTP status |
| ERR-22 | Sharing | Token regenerated while client cached old | next request 404, client must re-fetch |
| ERR-23 | Realtime | EventHub Publish during shutdown | drained gracefully |

---

## 15. Provenance & audit semantics (v0.5.9+)

Cross-cutting invariants every mutation must satisfy:

| ID | Invariant | Verification |
|---|---|---|
| PROV-01 | Every `nodes` / `edges` write has an `audit_log` row | query `SELECT count(*) FROM audit_log WHERE created_at > <test_start>` after each scenario |
| PROV-02 | `audit_log.actor_kind` matches API key's `kind` | verify per scenario AUTH-14~17 |
| PROV-03 | `audit_log.client_type` matches `X-Thask-Client` header (or `unknown`) | header parsing tested in MCP-* scenarios |
| PROV-04 | `audit_log.mutation_kind` correctly classifies field changes (structural/semantic/meta) | description→semantic, status→meta, file_paths→structural |
| PROV-05 | `audit_log.batch_id` groups multi-item operations | NODE-19~24, EDGE-07~08 |
| PROV-06 | Suggestion accept's `audit_log.user_id` = decider's user_id, NOT proposer | SUGG-03 enforced server-side |
| PROV-07 | Verify action sets `last_verified_at/by/commit` and writes `audit_log.action=verified` | NODE-26 |

---

## 17. Knowledge OS Foundation (v0.6.0)

The v0.6.0 release adds four first-class entity types and three side tables
without introducing new MCP tools — enum widening + additive fields on
`node.update` / `edge.update` handle everything. Scenarios below reuse the
`NODE-` / `EDGE-` prefixes for lookups but are tagged `v0.6.0`.

### 17.1 New entity types

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-01 | H/A | `POST /api/projects/:pid/nodes` with `{type:"REQUIREMENT", title, lifecycleState:"DRAFT"}` | 201, `type` accepted, `lifecycle_state_changed_at` stamped |
| KOS-02 | H/A | Same with `type:"DECISION"` / `"EXPERIMENT"` / `"PERSON"` | 201 (v0.6.0 enum values) |
| KOS-03 | H/A | `PATCH /nodes/:nid` with `{lifecycleState:"APPROVED"}` on a REQUIREMENT | 200, `lifecycle_state_changed_at = now()` |
| KOS-04 | H/A | `PATCH /nodes/:nid` re-writing the same `lifecycleState` | still bumps `lifecycle_state_changed_at` (v0.6.0 unconditional; `IS DISTINCT FROM` optimisation planned for v0.6.1) |
| KOS-05 | H/A | Frontend NodeDetailView on REQUIREMENT/DECISION/EXPERIMENT/PERSON | Lifecycle state text input renders; other types hide it |

### 17.2 New edge verbs + metadata

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-06 | H/A | `POST /edges` with `{edgeType:"supersedes", metadata:{"reason":"conflicts with DEC-42"}}` | 201, metadata JSONB persisted |
| KOS-07 | H/A | `POST /edges` with any of `realizes`, `conflicts`, `drives`, `tests`, `produced`, `owns`, `decided`, `reported` | 201 (v0.6.0 enum values) |
| KOS-08 | H/A | `PATCH /edges/:eid` with `{metadata:{"outcome_summary":"..."}}` alone | 200; COALESCE means other fields untouched |
| KOS-09 | H/A | `MCP thask.edge.update {metadata: {...}}` | Same as HTTP; matches wider enum on `edgeType` |

### 17.3 Threaded comments

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-10 | H | `POST /nodes/:nid/comments` with `{body:"..."}` | 201, `authorId` = caller |
| KOS-11 | H | `POST /nodes/:nid/comments` with `{body:"...", parentId:"<comment-uuid>"}` | 201, threaded reply |
| KOS-12 | H | `GET /nodes/:nid/comments` | 200 chronological list |
| KOS-13 | H | `PATCH /comments/:cid` as author | 200, body updated |
| KOS-14 | H | `PATCH /comments/:cid` as non-author | 404 (author-scoped write) |
| KOS-15 | H | `POST /comments/:cid/resolve` | 200, `resolvedAt`/`resolvedBy` set, row retained |
| KOS-16 | H | `DELETE /comments/:cid` as author | 200 |
| KOS-17 | A | Agent posts comment (`meta` classification) | 201 with default agent-key permissions |

### 17.4 Attachments

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-18 | H | `POST /nodes/:nid/attachments` (multipart, `file` field) with `THASK_ATTACHMENT_DIR` set | 201, file at `{dir}/{projectId}/{hex}-{name}`, SHA256 stored |
| KOS-19 | H | Same with `THASK_ATTACHMENT_DIR=""` (default) | 503 "Attachments disabled: THASK_ATTACHMENT_DIR unset" |
| KOS-20 | H | Upload larger than `THASK_ATTACHMENT_MAX_BYTES` (default 10 MB) | 413 |
| KOS-21 | H | `GET /attachments/:aid` | 200 file stream, `Content-Disposition: inline; filename=...` |
| KOS-22 | H | `GET /attachments/:aid` with tampered `storageKey` in DB pointing outside `attachmentDir` | 403 (path-traversal guard) |
| KOS-23 | H | `DELETE /attachments/:aid` | 200; DB row removed first, then blob unlinked (leak on unlink failure is acceptable) |
| KOS-24 | H | Delete project → attachments cascade | rows deleted, blobs remain on disk (cleanup by ops) |

### 17.5 Project tags

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-25 | H | `GET /tags` | 200 array of `{tag, color, description, createdBy}` |
| KOS-26 | H | `PUT /tags/backend` with `{color:"#D97706", description:"..."}` | 200, idempotent upsert on `(projectId, tag)` |
| KOS-27 | H | `DELETE /tags/backend` | 200; existing `nodes.tags[]` values untouched — palette entry only |

### 17.6 API-key project scope

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-28 | H | `POST /api/auth/api-keys` with `{projectId:"X"}` | 201, key scope stored |
| KOS-29 | A | Use scoped key against `/api/projects/X/...` | 200 |
| KOS-30 | A | Use scoped key against `/api/projects/Y/...` (X ≠ Y) | 403 (ProjectAccess middleware) |
| KOS-31 | H | `POST /api/auth/api-keys` with no `projectId` | 201, user-scope key (all projects the user can access, unchanged) |

### 17.7 node_history deprecation

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-32 | S | `PATCH /nodes/:nid` on v0.6.0 | 0 rows written to `node_history`, 1+ rows in `audit_log` |
| KOS-33 | H | Activity feed | still renders from mixed source (`audit_log` + legacy `node_history` reads) — no user-visible regression |
| KOS-34 | S | v0.7.0 upgrade | `node_history` table DROP planned; any external tooling must migrate to `audit_log` before then |

### 17.8 End-to-end lifecycle

| ID | Actor | Scenario | Expected |
|---|---|---|---|
| KOS-35 | H/A | Create REQUIREMENT (`DRAFT`) → linked TASK via `realizes` → EXPERIMENT `produced` DECISION → PERSON `owns` REQUIREMENT | Full v0.6.0 vocabulary exercised; `thask.graph.get` returns all types + all edges; layout(dagre) still succeeds |
| KOS-36 | A | Agent walks a project's KOS-35 graph and reports "which requirements are still DRAFT" | Filter `nodes` by `type=REQUIREMENT AND lifecycle_state='DRAFT'` — no MCP tool change needed |

---

## 16. How to use this catalog

### For E2E test selection
Pick scenarios by ID, group into a run plan. Example: smoke = AUTH-01, AUTH-03, NODE-07, NODE-12, NODE-17, GRAPH-01, CLI-01, CLI-05.

### For PR review
Cite scenario IDs when explaining what a change affects ("regression in NODE-22 because batch-update no longer returns 207").

### For bug reports
"Reproduces NODE-13 expected 403 but got 200" is cheaper than rewriting the steps.

### For CHANGELOG entries
Note `Covers SUGG-05` / `Adds CLI-46` to show what's freshly verified.

### When this catalog drifts
Add a TODO comment in the relevant scenario row pointing at the new/changed code path. Full re-audits should happen at each minor release (`make scenarios-audit` — not yet implemented, manual `grep` for now).

### Audit checklist (rerun at each release)
1. `routes.go` route count matches Section 1-9 count
2. `cli/internal/cmd/*.go` `Use:` count matches Section 10
3. `cli/internal/mcp/tools.go` `Name: "thask.*"` count matches Section 11
4. `frontend/src/routes/**/+page.svelte` count matches Section 12
5. New migration → new SYS-* or PROV-* invariant added if behavior changes
