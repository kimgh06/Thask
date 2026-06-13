# CLI Reference

The Thask CLI provides command-line access to projects, nodes, edges, and graphs. All commands output JSON by default.

**Installation:**

```bash
# Option 1: npm (recommended)
npm install -g @thask-org/cli

# Option 2: Build from source
cd cli && go build -o thask ./cmd/thask && ./thask install
```

**Quick start:**

```bash
thask config set url https://thask.kimgh06.com   # point at your instance
thask login                                       # browser flow, no copy/paste
thask auth whoami                                 # verify
```

See [`login`](#login) for details. Manual token entry via
`thask config set token <key>` still works for headless / SSH / CI sessions.

**Base config:** `~/.config/thask/config.json` (keys: `url`, `token`, `team`, `project`)

---

## Global Flags

These flags apply to all commands:

| Flag | Env Var | Default | Description |
|---|---|---|---|
| `--url` | `THASK_URL` | - | Backend URL (e.g., `http://localhost:7244`) |
| `--token` | `THASK_TOKEN` | - | API authentication token |
| `-p, --project` | `THASK_PROJECT` | - | Project ID (UUID) |
| `--team` | `THASK_TEAM` | - | Team slug |
| `-f, --format` | - | `json` | Output format: `json`, `table`, `quiet` |
| `--pretty` | - | - | Shorthand for `--format table` |
| `-q, --quiet` | - | - | Shorthand for `--format quiet` |

**Priority:** CLI flags > environment variables > config file

### Config File

Located at `~/.config/thask/config.json`:

```json
{
  "url": "http://localhost:7244",
  "token": "your-api-token",
  "team": "my-team",
  "project": "project-uuid"
}
```

### Auto-Update Notifications

Every command checks for a newer release on GitHub and prints a one-line warning to stderr when one is available:

```
🆕 thask v0.6.0 available  (current: v0.5.6)
   brew upgrade thask  ·  npm i -g @thask-org/cli
```

**Skip conditions** — the check is silently skipped when:
- Running in CI (`CI` environment variable is set)
- stderr is not a TTY (piped or redirected output)
- `THASK_NO_UPDATE_CHECK=1` is set
- Running `thask mcp serve` (to avoid polluting stdio transport)

**Cache** — version info is cached at `~/.thask/update-check.json` and refreshed at most once per 24 hours in the background.

To disable permanently:
```bash
export THASK_NO_UPDATE_CHECK=1
```

### Outbound Headers

Every outbound request sets `X-Thask-Client` so the backend `audit_log`
records which channel a write came from. Set automatically — no flag
required:

| Invocation | Header value |
|---|---|
| `thask <cmd>` (direct CLI) | `thask-cli/<version>` |
| `thask mcp serve` | `thask-mcp/<version> model=<client> session=<uuid>` |

`model` comes from MCP `clientInfo` (e.g. `claude-code/0.1.0`); `session`
is a per-server-instance UUID. This populates the `client_type`,
`agent_model`, and `agent_session_id` columns the backend uses to attribute
writes — see [API.md > Provenance & Suggestion Queue](API.md#provenance--suggestion-queue-v059).

### Error output split (v0.5.14+)

Thask routes two classes of failures to different formats so that humans get
readable terminal output while scripts and MCP clients keep a stable JSON
contract.

| Class | Examples | Where it goes | Exit |
|---|---|---|---|
| **Usage** | unknown flag, unknown command, wrong arg count, missing required flag | `stderr`, plain text: `Error: ...\nRun 'thask --help' for usage.` (Cobra also adds "Did you mean" suggestions where possible) | `2` |
| **Runtime / API** | auth failure, 4xx/5xx from backend, network error, validation failure from the server | `stderr`, JSON: `{"error":"..."}` | `1` (or whatever `client.ExitCode` maps the error to) |

Why the split: a first-time user typing `thask --version` should get
`thask v0.5.13 (...)` not `{"error":"unknown flag"}`. But a script running
`thask node create ...` and parsing `.error` to retry should keep getting
JSON. The match is by error message prefix — `unknown flag`, `unknown
command`, `unknown shorthand flag`, `flag provided but not defined`,
`required flag(s)`, `invalid argument`, `bad flag syntax`, plus phrase
matches for `accepts at most`, `requires at least`, etc.

`thask mcp serve` is unaffected — its stdio loop is MCP-protocol JSON-RPC,
not the regular CLI output path.

---

## config

Manage local configuration.

### config set \<key\> \<value\>

Set a configuration value. Keys: `url`, `token`, `team`, `project`.

```bash
thask config set url http://localhost:7244
thask config set token abc123def456
thask config set team my-team
thask config set project 550e8400-e29b-41d4-a716-446655440000
```

### config show

Display current configuration. The token value is masked for security.

```bash
thask config show
```

Output:
```json
{
  "url": "http://localhost:7244",
  "token": "abc123...***",
  "team": "my-team",
  "project": "550e8400..."
}
```

---

## login

Browser-based one-step authentication. Opens your configured Thask URL's
`/cli/auth` page, lets you click Approve, and writes the freshly-minted API
key into `~/.thask/config.json` — no copy/paste from the web UI required.

```bash
thask config set url https://thask.kimgh06.com   # one-time
thask login                                       # browser flow
```

**Flags:**
- `--url <url>` — Override the configured URL for this login (and persist it).
- `--force` — Replace an existing token without prompting.

**Behavior:**
- Spawns a loopback HTTP server on `127.0.0.1` (port in `7400-7500`).
- Generates a random `state` for CSRF protection.
- Opens the browser to `<url>/cli/auth?callback_port=<port>&state=<state>` and
  prints the URL on stderr so you can paste it manually if the browser doesn't
  auto-open.
- If you aren't logged into the web UI yet, you'll be sent through `/login`
  first and back to the authorization page automatically.
- The web page mints a new `user_interactive` API key (full default permissions)
  via the existing `POST /api/auth/api-keys` endpoint and redirects to the
  loopback server with the token in the query string.
- Server validates `state`, writes the token to config, and exits.
- Times out after 5 minutes if no callback arrives.

**Not yet supported:** SSH / headless / CI sessions (no browser available) —
fall back to the manual `thask config set token <key>` flow for those.
A device-code flag (`--device`) is planned for v0.5.12+.

---

## doctor

Single-command diagnostic for the full CLI + MCP + backend stack. Use this
when something is broken and you cannot tell whether the URL, token, server,
migrations, or MCP wiring is at fault.

```bash
thask doctor
```

**Checks (in order):**

| # | Check | Source |
|---|---|---|
| 1 | Binary version (`thask <ver> (<commit>)`) | linker `-ldflags` |
| 2 | Config file (`~/.thask/config.json`) exists with `mode 600` | filesystem |
| 3 | `url` is configured | config |
| 4 | Server reachable + backend version + uptime | `GET /api/health` |
| 5 | DB ping + applied migration count + latest migration version | `GET /api/health` |
| 6 | `token` is configured | config |
| 7 | Token valid (resolves to a user) | `GET /api/auth/me` |
| 8 | Token kind + permission flags | `GET /api/auth/api-keys` |
| 9 | Default `team` set (warn-only) | config |
| 10 | Default `project` set (warn-only) | config |
| 11 | `thask` binary on `PATH` (needed for MCP `command`) | `exec.LookPath` |
| 12 | Claude Code MCP entry (`~/.claude.json`) | filesystem scan |
| 13 | Cursor MCP entry (`~/.cursor/mcp.json`) | filesystem scan |
| 14 | Codex MCP entry (`~/.codex/config.toml`) | filesystem scan |

Each line prints `✓` / `⚠` / `✗` with a one-line hint when the check fails.

**Exit codes:**
- `0` — all critical checks passed (warnings OK)
- `1` — one or more critical checks failed

**Example output:**
```
  ✓ Binary version             thask 0.5.12 (a1b2c3d)
  ✓ Config file                /Users/kim/.thask/config.json (mode 600)
  ✓ URL                        http://localhost:7244
  ✓ Server reachable           backend 0.5.12, uptime 2m30s
  ✓ Server DB                  migrations applied: 10 (latest version 10)
  ✓ Token                      thsk_c2fcb02...
  ✓ Token valid                Kim <kkh061101@gmail.com>
  ✓ Token permissions          kind=user_interactive, read,write_structural,write_semantic,write_meta,verify,delete,suggest
  ✓ Default team               thask
  ✓ Default project            a9116af4-d6d1-416f-8c40-de31be0f5f49
  ✓ MCP binary on PATH         /Users/kim/.local/bin/thask
  ✓ Claude Code MCP            /Users/kim/.claude.json
  ⚠ Cursor MCP (global)        not configured
       → Run `thask init` to patch it
  ✓ Codex MCP                  /Users/kim/.codex/config.toml

All critical checks passed. 1 warning(s).
```

The `Server reachable` and `Server DB` rows come from the `/api/health`
endpoint enhanced in v0.5.12 — older backend instances that still respond with
the legacy `{"status":"ok"}` body are accepted as `version=unknown, db=ok`.

---

## auth

Authenticate and manage sessions.

### auth whoami

Display the currently authenticated user.

```bash
thask auth whoami
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "displayName": "User Name"
  }
}
```

---

## team (alias: t)

Team management.

### team list | ls

List all teams the authenticated user is a member of.

```bash
thask team list
thask t ls
```

Output (JSON):
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Engineering",
      "slug": "engineering",
      "projects": [...]
    }
  ]
}
```

Output (table format):
```bash
thask team list --pretty
```

### team get \<slug\>

Get details for a specific team.

```bash
thask team get engineering
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Engineering",
    "slug": "engineering"
  }
}
```

### team create --name \<name\> --slug \<slug\>

Create a new team. The creator becomes owner.

```bash
thask team create --name "My Team" --slug "my-team"
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Team",
    "slug": "my-team"
  }
}
```

### team members \<slug\>

List members of a team.

```bash
thask team members engineering
thask team members engineering --pretty
```

Output (table):

| ID | NAME | EMAIL | ROLE |
|---|---|---|---|
| 550e8400... | Alice | alice@example.com | owner |
| 550e8400... | Bob | bob@example.com | member |

### team invite \<slug\> --email \<email\> [\--role \<role\>]

Invite a user to a team. Roles: `admin`, `member`, `viewer` (default: `member`).

```bash
thask team invite engineering --email newmember@example.com
thask team invite engineering --email newadmin@example.com --role admin
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "newmember@example.com",
    "role": "member"
  }
}
```

---

## project (alias: p)

Project management.

### project list | ls

List all projects in a team. Requires `--team`.

```bash
thask project list --team engineering
thask p ls --team engineering --pretty
```

Output:
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Backend",
      "description": "Backend services",
      "teamId": "550e8400-e29b-41d4-a716-446655440001"
    }
  ]
}
```

### project create --name \<name\> [\--description \<desc\>]

Create a project in a team. Requires `--team`.

```bash
thask project create --team engineering --name "Backend" --description "Backend services"
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Backend",
    "description": "Backend services"
  }
}
```

### project get [\<projectId\>]

Get project details. Uses `--project` if no ID provided.

```bash
thask project get 550e8400-e29b-41d4-a716-446655440000
thask project get  # uses --project flag
```

### project share [\--mode viewer\|editor]

Enable link sharing for a project. Default mode is `viewer`.

```bash
thask project share -p <projectId>
thask project share -p <projectId> --mode editor
```

| Flag | Default | Description |
|---|---|---|
| `--mode` | `viewer` | Sharing mode: `viewer` (read-only) or `editor` (full access) |

### project unshare

Disable link sharing. Clears the share token (old links become invalid).

```bash
thask project unshare -p <projectId>
```

### project members

List project sharing settings and members.

```bash
thask project members -p <projectId>
```

### project invite \--email \<email\> [\--role editor\|viewer]

Invite a user to the project.

```bash
thask project invite -p <projectId> --email user@example.com
thask project invite -p <projectId> --email user@example.com --role editor
```

| Flag | Default | Description |
|---|---|---|
| `--email` | (required) | Email of user to invite |
| `--role` | `viewer` | Role: `editor` or `viewer` |

### project kick \--user \<userId\>

Remove a user from the project.

```bash
thask project kick -p <projectId> --user <userId>
```

---

## node (alias: n)

Node operations.

### node list | ls

List all nodes in a project. Requires `--project`. Optional filters: `--type`, `--status`.

```bash
thask node list --project 550e8400-e29b-41d4-a716-446655440000
thask node list --pretty
thask node list --type TASK --status IN_PROGRESS
thask n ls --quiet  # IDs only
```

Output (table):

| ID | TYPE | TITLE | STATUS |
|---|---|---|---|
| 550e8400... | TASK | Setup database | IN_PROGRESS |
| 550e8400... | BUG | Fix login error | PASS |

### node get \<nodeId\> | g

Get node details with history and connected edges.

```bash
thask node get 550e8400-e29b-41d4-a716-446655440000
thask n g 550e8400-e29b-41d4-a716-446655440000
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "TASK",
    "title": "Setup database",
    "description": "Initialize PostgreSQL",
    "status": "IN_PROGRESS",
    "positionX": 100,
    "positionY": 200,
    "tags": ["backend", "infrastructure"],
    "connectedEdges": [...],
    "connectedNodeIds": ["550e8400..."],
    "history": [
      {
        "id": "550e8400...",
        "action": "updated",
        "fieldName": "status",
        "oldValue": "TODO",
        "newValue": "IN_PROGRESS",
        "timestamp": "2025-03-20T10:30:00Z"
      }
    ]
  }
}
```

### node create | c

Create a new node. Requires `--type` and `--title`.

```bash
thask node create --type TASK --title "Setup database" --description "Initialize PostgreSQL" --status IN_PROGRESS --tags "backend,infrastructure" --x 100 --y 200
thask n c --type BUG --title "Fix login error"
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "TASK",
    "title": "Setup database",
    "status": "TODO"
  }
}
```

### node update \<nodeId\> | u

Update a node. All fields are optional.

```bash
thask node update 550e8400-e29b-41d4-a716-446655440000 --title "Updated title" --status PASS
thask n u 550e8400-e29b-41d4-a716-446655440000 --type BUG --tags "urgent,backend"
thask n u 550e8400-e29b-41d4-a716-446655440000 --parent <groupId>   # move into GROUP
thask n u 550e8400-e29b-41d4-a716-446655440000 --parent none        # remove from GROUP
```

| Flag | Description |
|---|---|
| `--title` | New title |
| `--status` | New status: PASS, FAIL, IN\_PROGRESS, BLOCKED |
| `--type` | New type: FLOW, BRANCH, TASK, BUG, API, UI, GROUP |
| `--description` | New description |
| `--tags` | Comma-separated tags |
| `--parent` | Parent GROUP node ID, or `none` to unparent |

### node delete \<nodeId\> | d, rm

Delete a node. If it's a GROUP, children are unparented but preserved.

```bash
thask node delete 550e8400-e29b-41d4-a716-446655440000
thask n rm 550e8400-e29b-41d4-a716-446655440000
```

### node batch-status --ids \<comma-sep\> --status \<status\>

Update status for multiple nodes at once.

```bash
thask node batch-status --ids "550e8400-e29b-41d4-a716-446655440000,550e8400-e29b-41d4-a716-446655440001" --status PASS
```

---

## edge (alias: e)

Edge operations.

### edge list | ls

List all edges in a project.

```bash
thask edge list --pretty
thask e ls --quiet
```

Output (table):

| ID | SOURCE | TARGET | TYPE | LABEL |
|---|---|---|---|---|
| 550e8400... | 550e8400... | 550e8400... | depends_on | Blocked by DB setup |
| 550e8400... | 550e8400... | 550e8400... | related | - |

### edge create | c

Create an edge between two nodes. Requires `--source` and `--target`.

Edge types: `depends_on`, `blocks`, `related`, `parent_child`, `triggers`

```bash
thask edge create --source 550e8400-e29b-41d4-a716-446655440000 --target 550e8400-e29b-41d4-a716-446655440001 --type depends_on
thask e c --source 550e8400... --target 550e8400... --type related --label "Related work"
```

Output:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "sourceId": "550e8400-e29b-41d4-a716-446655440001",
    "targetId": "550e8400-e29b-41d4-a716-446655440002",
    "edgeType": "depends_on",
    "label": "Blocked by DB setup"
  }
}
```

### edge update \<edgeId\>

Update an edge's type or label.

```bash
thask edge update 550e8400-e29b-41d4-a716-446655440000 --type blocks --label "Blocking deployment"
```

### edge delete \<edgeId\> | d, rm

Delete an edge.

```bash
thask edge delete 550e8400-e29b-41d4-a716-446655440000
thask e rm 550e8400-e29b-41d4-a716-446655440000
```

---

## graph (alias: g)

Graph operations.

### graph get

Export the full graph (all nodes and edges) for a project.

```bash
thask graph get --project 550e8400-e29b-41d4-a716-446655440000 > graph.json
thask graph get > graph.json  # uses --project flag
```

Output:
```json
{
  "data": {
    "nodes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "type": "TASK",
        "title": "Setup database",
        "status": "IN_PROGRESS"
      }
    ],
    "edges": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "sourceId": "550e8400-e29b-41d4-a716-446655440001",
        "targetId": "550e8400-e29b-41d4-a716-446655440002",
        "edgeType": "depends_on"
      }
    ]
  }
}
```

### graph export [\--format json\|md] [\--file \<path\>]

Export graph to a JSON file or a markdown handoff document.

```bash
thask graph export -p <projectId>
thask graph export -p <projectId> --file my-graph.json
thask graph export -p <projectId> --format md > handoff.md
thask graph export -p <projectId> --format md --file handoff.md
```

Use `--format md` to render the project graph as a markdown handoff document, one H2 section per node with dependency links and a creator footer. Suitable for attaching to a Slack message or PR.

| Flag | Default | Description |
|---|---|---|
| `--file` | `graph.json` (json) / stdout (md) | Output file path |
| `--format` | `json` | Output format: `json` or `md` |

### graph capture [\--out \<path\>] [\--width 1600] [\--height 1000]

Capture the current graph as a PNG image through the Playwright capture worker. Use `--format svg` for the lightweight server-side preview renderer.

```bash
thask graph capture -p <projectId>
thask graph capture -p <projectId> --out autolayout-after.png --width 1600 --height 1000 --scale 2
thask graph capture -p <projectId> --format svg --out graph.svg
```

| Flag | Default | Description |
|---|---|---|
| `--out` | `graph.png` | Output image path |
| `--file` | — | Deprecated alias for `--out` |
| `--format` | `png` | Image format: `png` or `svg` |
| `--width` | `1600` | Image width in pixels |
| `--height` | `1000` | Image height in pixels |
| `--padding` | `80` | Fit padding in pixels |
| `--scale` | `2` | Output pixel scale, from `1` to `4` |
| `--crop` | `true` | Crop PNG output to graph bounds |

### graph import --file \<path\> [\--mode merge\|replace]

Import a graph from a JSON file. Default mode is `merge` (adds to existing graph). Use `replace` to overwrite.

```bash
thask graph import --file graph.json --mode merge
thask graph import --file graph.json --mode replace
```

### graph layout [\--algorithm dagre\|grid]

Run server-side auto-layout. Repositions all nodes and auto-sizes GROUPs.

```bash
thask graph layout -p <projectId>
thask graph layout -p <projectId> --algorithm grid
```

| Flag | Default | Description |
|---|---|---|
| `--algorithm` | `dagre` | Layout algorithm: `dagre`, `grid` |

### graph analyze [\--format json]

Detect dependency cycles and find the critical path (longest dependency chain). Only traverses `depends_on` and `blocks` edges.

```bash
thask graph analyze -p <projectId>
thask graph analyze -p <projectId> --format json
```

Output (default):
```
Cycles: 2
  1. auth -> service
  2. auth -> service -> model -> auth

Critical Path: 4 nodes
  main -> handler -> service -> model
```

Output (JSON):
```json
{
  "data": {
    "cycles": [["auth", "service"], ["auth", "service", "model"]],
    "criticalPath": ["main", "handler", "service", "model"]
  }
}
```

---

## scan

Scan a project's source code and import the dependency graph automatically.

### scan [--path .] [--max-files 500]

Scan the Go project at `--path` and import all package-level dependencies as nodes and edges (merge mode). Requires `--project` or `THASK_PROJECT` to be set.

```bash
thask scan -p <projectId>
thask scan -p <projectId> --path ./myservice
thask scan -p <projectId> --max-files 200
```

### scan --dry-run

Print the discovered nodes/edges as JSON to stdout without importing anything.

```bash
thask scan --path . --dry-run
thask scan --plugin ./my-scanner.py --path . --dry-run | jq '.nodes | length'
```

### scan --plugin \<path\>

Use an external scanner plugin instead of the built-in Go scanner. The plugin must accept `--path <dir>` and output JSON matching the import format.

```bash
thask scan -p <projectId> --plugin ./python-scanner.py --path ./myapp
```

See [docs/PLUGINS.md](PLUGINS.md) for the plugin contract and an example Python scanner.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--path` | `.` | Path to project root |
| `--max-files` | `500` | Maximum source files to process |
| `--dry-run` | `false` | Print JSON to stdout, skip import |
| `--plugin` | — | Path to external scanner executable |

---

## impact

Impact analysis.

### impact --node \<nodeId\>

Run impact analysis on a node. Shows changed nodes and their downstream dependencies.

```bash
thask impact --node 550e8400-e29b-41d4-a716-446655440000 --project 550e8400-e29b-41d4-a716-446655440000
```

Output:
```json
{
  "data": {
    "changedNodes": [...],
    "impactedNodes": [...],
    "failNodes": [...],
    "impactEdges": [...]
  }
}
```

---

## mcp

MCP server integration.

### mcp serve

Start the MCP server on stdio for Claude Code integration.

```bash
thask mcp serve
```

---

## aliases

Shell alias management for faster command entry.

### aliases show

Print all available aliases to stdout.

```bash
thask aliases show
```

Available aliases:

```
t=thask
tn=thask node
tnls=thask node list --pretty
tnc=thask node create
tnu=thask node update
tnd=thask node delete
tng=thask node get
te=thask edge
tels=thask edge list --pretty
tec=thask edge create
ted=thask edge delete
tt=thask team
ttls=thask team list --pretty
tp=thask project
tpls=thask project list --pretty
tg=thask graph get
tim=thask impact
```

### aliases install [\--file \<path\>]

Append aliases to your shell rc file (`~/.zshrc` or `~/.bashrc`). Auto-detects shell.

```bash
thask aliases install
thask aliases install --file ~/.bashrc
```

### aliases uninstall [\--file \<path\>]

Remove Thask aliases from your shell rc file.

```bash
thask aliases uninstall
thask aliases uninstall --file ~/.bashrc
```

---

## install / uninstall

Manage the CLI binary installation.

### install [\--dir \<path\>]

Install the Thask binary to PATH.

- **macOS/Linux default:** `/usr/local/bin`
- **Windows default:** `%LOCALAPPDATA%\thask`

```bash
thask install
thask install --dir ~/.local/bin
```

### uninstall

Remove Thask from PATH.

```bash
thask uninstall
```

---

## version

Display version and commit information.

```bash
thask version
```

Output:
```
Thask v0.1.0 (commit: abc123def456)
```

---

## self-update

Download the latest published release from GitHub and replace the running
binary in place. Skips Homebrew / npm — direct binary swap. Useful when
you installed via `curl | sh` or `go build` and just want to follow
along with releases.

```bash
thask self-update                  # install latest
thask self-update --check          # check only, exit 1 if update available
thask self-update --version 0.5.10 # pin to a specific tag
```

**Flags:**
- `--check` — Print whether an update is available, exit non-zero if so
  (no install). Lets shell scripts gate on update availability.
- `--version <X>` — Install a specific release tag instead of latest.

**Mechanics:**
- Resolves the running binary path via `os.Executable()`.
- Picks the right asset for `runtime.GOOS / GOARCH` — tarballs on unix,
  `.exe` on Windows.
- Streams the download into a temp file **in the same directory as the
  binary** so the eventual `os.Rename` is atomic — concurrent `thask`
  invocations never see a half-written binary.
- 200 MB ceiling on the download + tarball extraction so a hostile or
  corrupted release can't fill the install directory.
- Permission errors hint at `sudo` when the install dir is system-owned.

**Not yet supported (planned for a future release):**
- SHA256 / checksum verification of the downloaded asset. HTTPS + GitHub
  ACLs are the only integrity checks today.

---

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error |
| HTTP status codes | API errors (e.g., 404 for not found, 401 for unauthorized) |

---

## Output Formats

### JSON (default)

Raw JSON from the API. Suitable for piping to `jq` or other tools.

```bash
thask node list | jq '.data[] | select(.status == "IN_PROGRESS")'
```

### Table (--pretty or --format table)

Human-readable table output. Useful for interactive use.

```bash
thask node list --pretty
```

### Quiet (-q or --format quiet)

IDs only, one per line. Useful for piping to other commands.

```bash
thask node list --quiet | xargs -I {} thask node get {}
```

---

## Common Workflows

### Setup and Authentication

1. Set backend URL:
   ```bash
   thask config set url http://localhost:7244
   ```

2. Authenticate (get token from server or API):
   ```bash
   thask config set token your-api-token
   ```

3. Verify authentication:
   ```bash
   thask auth whoami
   ```

### Working with Projects

1. List teams:
   ```bash
   thask team list --pretty
   ```

2. Create a project:
   ```bash
   thask project create --team engineering --name "Backend"
   ```

3. Set default project:
   ```bash
   thask config set project 550e8400-e29b-41d4-a716-446655440000
   ```

### Creating and Linking Tasks

1. Create a task:
   ```bash
   thask node create --type TASK --title "Setup database"
   ```

2. Create another task:
   ```bash
   thask node create --type TASK --title "Configure ORM"
   ```

3. Link them with an edge:
   ```bash
   thask edge create --source <node1-id> --target <node2-id> --type depends_on
   ```

4. View the task:
   ```bash
   thask node get <node-id> --pretty
   ```

### Batch Operations

1. Get all task IDs in quiet format:
   ```bash
   thask node list --type TASK --quiet
   ```

2. Batch update status:
   ```bash
   TASK_IDS=$(thask node list --type TASK --quiet | tr '\n' ',' | sed 's/,$//')
   thask node batch-status --ids "$TASK_IDS" --status PASS
   ```

### Exporting and Importing Graphs

1. Export full graph:
   ```bash
   thask graph get > my-graph.json
   ```

2. Import to another project:
   ```bash
   thask graph import --file my-graph.json --mode merge --project <another-project-id>
   ```

### Impact Analysis Workflow

1. Make changes to a task:
   ```bash
   thask node update <node-id> --status PASS
   ```

2. Analyze impact:
   ```bash
   thask impact --node <node-id> --pretty
   ```

3. Find affected downstream tasks and verify status updates.

### Quick Commands with Aliases

After running `thask aliases install`:

```bash
# List nodes with table format
tnls

# Create a node
tnc --type TASK --title "My task"

# Get node details
tng <node-id>

# Update node
tnu <node-id> --status PASS

# List edges
tels

# Create edge
tec --source <id1> --target <id2> --type depends_on

# List teams
ttls

# View graph
tg

# Run impact analysis
tim --node <node-id>
```

---

## Error Handling

Common error scenarios and responses:

```bash
# Missing required field
thask node create --type TASK
# Error: Title is required

# Unauthorized
thask node list
# Error: Authentication required

# Not found
thask node get 550e8400-e29b-41d4-a716-446655440000
# Error: Node not found

# Invalid format
thask node list --format invalid
# Error: Format must be one of: json, table, quiet
```

Use `thask auth whoami` to verify authentication before debugging API errors.
