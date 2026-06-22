# MCP Integration Guide

Thask exposes a full-featured MCP (Model Context Protocol) server that lets AI coding agents like Claude Code and Cursor interact with your project graphs directly.

---

## What is MCP?

Model Context Protocol is a standard for connecting AI agents to external tools and data sources. With Thask's MCP server, Claude Code, Cursor, and other agents can:

- Create, read, update, and delete nodes and edges
- Analyze cascade impact of changes
- Import and export graph data
- Batch update multiple nodes at once

All without leaving the editor.

---

## Prerequisites

1. **Thask server running** — locally via `docker compose up` or a remote instance
2. **Thask CLI installed** — `npm install -g @thask-org/cli` or build from source (`cd cli && go build -o thask ./cmd/thask`)
3. **API token** — easiest is:

   ```bash
   thask config set url <your-thask-url>
   thask login   # opens browser, click Approve, token saved
   ```

   This writes the token to `~/.thask/config.json` and the MCP server below
   reads it automatically — no need to put the key in `.claude/mcp.json`.
   See [CLI Reference > login](./CLI.md#login) for details on the flow.
   For SSH / headless setups, create a key in the web UI (Settings > API Keys)
   and run `thask config set token <key>` instead.

---

## Setup

### Claude Code

After `thask login` (see Prerequisites), the MCP server picks up your URL +
token from `~/.thask/config.json`. Minimal `.claude/mcp.json`:

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

Override URL/token inline if you want a different identity for this MCP than
your CLI default:

```json
{
  "mcpServers": {
    "thask": {
      "command": "thask",
      "args": ["mcp", "serve", "--url", "http://localhost:7244", "--token", "YOUR_API_KEY"]
    }
  }
}
```

Or use npx (no global install needed):

```json
{
  "mcpServers": {
    "thask": {
      "command": "npx",
      "args": ["-y", "@thask-org/cli", "mcp", "serve"]
    }
  }
}
```

Alternatively, if you've already configured the CLI with `thask config set url` and `thask config set token`, use the simpler form:

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

Then restart Claude Code to activate the server.

### Cursor

Use the same configuration in `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "thask": {
      "command": "thask",
      "args": ["mcp", "serve", "--url", "http://localhost:7244", "--token", "YOUR_API_KEY"]
    }
  }
}
```

Restart Cursor after adding the configuration.

---

## Available Tools

The MCP server exposes 24 tools organized into 7 categories.

### Node Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.node.list` | List nodes in a project | `projectId` | `type`, `status` |
| `thask.node.create` | Create a new node | `projectId`, `type`, `title` | `description`, `status`, `tags`, `positionX`, `positionY` |
| `thask.node.get` | Get node details, connected edges, and history | `projectId`, `nodeId` | — |
| `thask.node.update` | Update node fields | `projectId`, `nodeId` | `title`, `status`, `type`, `description`, `tags`, `parentId`, `assigneeId` |
| `thask.node.delete` | Delete a node | `projectId`, `nodeId` | — |
| `thask.node.batch_status` | Batch update status for multiple nodes | `projectId`, `ids`, `status` | — |
| `thask.node.batch_update` | Partial-update up to 200 nodes in one call (atomic on permission / cycle) | `projectId`, `updates[]` | — |
| `thask.node.suggest_update` | Queue a description change for human review (agent-safe alternative to `update`) | `projectId`, `nodeId`, `proposedValue` | `fieldName`, `rationale`, `evidence` |
| `thask.node.verify` | Stamp "still correct as of now" on a node | `projectId`, `nodeId` | `commit` |

> `parentId` / `assigneeId` accept an empty string to unparent / unassign;
> omit to leave untouched. Agent-kind keys are blocked from writing
> `description` and `tags` via `update` by default — use `suggest_update`.

### Edge Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.edge.list` | List all edges in a project | `projectId` | — |
| `thask.edge.create` | Create a relationship between nodes | `projectId`, `sourceId`, `targetId` | `edgeType`, `label` |
| `thask.edge.delete` | Delete an edge | `projectId`, `edgeId` | — |
| `thask.edge.batch_create` | Insert up to 500 edges in one call (skip reasons: self_reference, duplicate, invalid_endpoint) | `projectId`, `edges[]` | — |
| `thask.edge.batch_delete` | Delete up to 500 edges by id (skip reason: not_found) | `projectId`, `edgeIds[]` | — |

### Graph Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.graph.get` | Get full graph snapshot (all nodes and edges) | `projectId` | — |
| `thask.graph.import` | Import graph from JSON — **creates NEW node ids**, does NOT patch existing nodes by id (use `node.update` / `node.batch_update` for that) | `projectId`, `mode`, `nodes`, `edges` | — |
| `thask.graph.layout` | Auto-layout the project graph. Repositions all nodes and auto-sizes GROUP nodes. | `projectId` | `algorithm` (dagre \| grid, default: dagre) |
| `thask.graph.analyze` | Detect dependency cycles and find the critical path (longest dependency chain) | `projectId` | — |

### Analysis Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.impact.analyze` | Analyze cascade impact of changing a node | `projectId`, `nodeId` | — |

### Scan Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.scan.run` | Scan a project's internal dependencies and import them as graph nodes/edges. Auto-detects Go / TypeScript via `go.mod` / `package.json`. | `projectId`, `path` | `language` (`auto` \| `go` \| `ts`), `maxFiles` (default 500) |

### Suggestions

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.suggestions.list` | List pending agent-proposed updates awaiting human review | `projectId` | `limit` |
| `thask.suggestions.decide` | Accept or reject a pending suggestion (server enforces `user_interactive` actor for `accepted`) | `projectId`, `suggestionId`, `status` | `reason` |

### Meta

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.guide` | Get the full AI agent guide for Thask. Call this before your first interaction with a Thask project. | — | `projectId` |
| `thask.mistake.record` | Record an agent mistake as a BUG node under the project's "실수 기록" GROUP (auto-created). Surfaced by `thask.guide` in future sessions. | `projectId`, `title`, `lesson` | `cause`, `fix` |

### Local Telemetry of MCP Calls (v0.5.15+)

Every `tools/call` dispatch through `thask mcp serve` appends an `mcp_call` event to `~/.thask/events.jsonl` with `tool_name`, `duration_ms`, `ok`, in/out payload size buckets, and `parent` pointing to the serve session's invocation event. The agent never sees this — it is the human's own usage log. The serve session itself also gets one `invocation` event (`cmd: "thask mcp serve"`) on shutdown. Inspect via `thask reflog` (the serve invocation) or directly with `jq 'select(.kind=="mcp_call")' ~/.thask/events.jsonl` for the per-tool rows. Opt out with `thask telemetry disable`. Raw request/response bodies stay local-only and require `thask telemetry config set capture_payloads true` to be recorded at all.

---

## Enum Values

### Node Types
`FLOW`, `BRANCH`, `TASK`, `BUG`, `API`, `UI`, `GROUP`

### Node Statuses
`PASS`, `FAIL`, `IN_PROGRESS`, `BLOCKED`

### Edge Types
`depends_on`, `blocks`, `related`, `parent_child`, `triggers`

### Import Modes
`replace` (overwrite existing graph), `merge` (add alongside existing)

---

## Example Workflow

Here's a realistic conversation with Claude Code:

```
User: "Map the dependencies for my Express.js API"

Claude Code:
  thask.node.create({ projectId: "...", type: "API", title: "GET /users", status: "PASS" })
  thask.node.create({ projectId: "...", type: "API", title: "POST /users", status: "PASS" })
  thask.node.create({ projectId: "...", type: "UI", title: "User List Page", status: "IN_PROGRESS" })
  thask.edge.create({ projectId: "...", sourceId: "<GET /users ID>", targetId: "<User List Page ID>", edgeType: "blocks" })

User: "What would break if I change POST /users?"

Claude Code:
  thask.impact.analyze({ projectId: "...", nodeId: "<POST /users ID>" })
  → Returns list of affected nodes and edges downstream
```

---

## Troubleshooting

| Error | Cause | Solution |
|---|---|---|
| `connection refused` | Thask server not running | Check server is up: `docker compose up` or verify remote URL |
| `401 unauthorized` | Invalid or expired API token | Create new token in web UI (Settings > API Keys) |
| `tool not found` | MCP server not loaded | Restart Claude Code or Cursor after updating `mcp.json` |
| `invalid projectId` | Project ID doesn't exist or user lacks access | Verify project ID in web UI |

### Debugging

MCP server logs go to stderr. Check your editor's terminal or debug console for detailed error messages.

**First stop for any setup problem:** run `thask doctor`. It walks the full
stack (binary, config, URL, server reachability, DB + migration version, token
validity, token permissions, Claude / Cursor / Codex MCP entries) and prints
`✓` / `⚠` / `✗` with a remediation hint per check. See
[CLI.md#doctor](./CLI.md#doctor) for the full reference.

```bash
thask doctor
```

Manual probes (rarely needed once `doctor` exists):

```bash
thask node list --project <projectId>            # CLI connection
thask config show                                # what is set
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:7244/api/auth/me           # token validity
```

---

## Configuration via Environment

You can also set URL and token via environment variables instead of CLI config:

```bash
export THASK_URL=http://localhost:7244
export THASK_TOKEN=your_api_key
```

Then use the simpler `mcp.json`:

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

---

## Provenance Tools (v0.5.9)

To prevent agents from silently writing hallucinated descriptions into the graph,
v0.5.9 adds four MCP tools and gates writes by per-key permissions.

| Tool | Purpose | Required permission |
|---|---|---|
| `thask.node.suggest_update` | Queue a description change for human review | `suggest` |
| `thask.suggestions.list` | List pending proposals for a project | `read` |
| `thask.suggestions.decide` | Accept/reject a proposal | `write_semantic` (when accepting) |
| `thask.node.verify` | Stamp "still correct as of now" on a node | `verify` |

### Agent workflow

The recommended flow for an agent that wants to revise a node:

```
1. thask.graph.get → read existing description + provenance
2. Re-derive from code: open the files the node covers
3. If description should change:
   thask.node.suggest_update {
     projectId, nodeId,
     proposedValue: "...",
     rationale: "src/auth.ts:42 added refreshToken() — quote the line",
     evidence: { codeCommit: "abc1234", sourcePaths: ["src/auth.ts"], confidence: "high" }
   }
4. Human reviews via the UI or `thask.suggestions.list / decide`
```

Agent keys are blocked from `thask.node.update {description: ...}` and
`thask.node.verify` by default. The CLI/MCP backend returns a 403 with
guidance pointing at the suggest_update path.

When composing a `proposedValue` for `suggest_update`, prefer the **Handoff Convention** from [GRAPH.md](./GRAPH.md#handoff-convention-v0513) — structure the description with `## Why`, `## Q&A`, and `## Gotchas` headings. This makes the description useful for both onboarding humans and future agents reading the graph.

### Provenance in read responses

Every node returned by `thask.node.get`, `thask.node.list`, `thask.graph.get`,
and `thask.impact.analyze` includes:

```json
{
  "id": "...",
  "title": "...",
  "description": "...",
  "descriptionSource": "agent",
  "descriptionAuthoredBy": "<user_id>",
  "descriptionAuthoredAt": "2026-05-23T...",
  "descriptionAgentModel": "claude-opus-4-7",
  "lastVerifiedAt": null,
  "lastVerifiedBy": null,
  "lastVerifiedCommit": null
}
```

When reading another agent's content, treat `descriptionSource: "agent"` with
`lastVerifiedAt: null` as **hint, not ground truth** — verify against code
before acting on it.

### X-Thask-Client header

The MCP server automatically sets `X-Thask-Client: thask-mcp/<ver> model=<client> session=<uuid>`
on every outbound request. `model` comes from MCP `clientInfo` (e.g.
`claude-code/0.1.0`), `session` is a per-server-instance UUID.

---

## Bulk Operations (v0.5.10)

Three tools cut down on per-call agent context overhead when touching many
nodes or edges at once.

| Tool | Limit | Atomic on | Best-effort on |
|---|---|---|---|
| `thask.node.batch_update` | 200 | permission, cycle, DB | not_found, no_change |
| `thask.edge.batch_create` | 500 | permission, DB | self_reference, duplicate, invalid_endpoint |
| `thask.edge.batch_delete` | 500 | permission, DB | not_found |

Each call returns the result set + `skipped[]` (when relevant) + a shared
`batchId` that groups the audit_log rows generated by this batch.

### When to use which tool

| Scenario | Tool |
|---|---|
| One node, multiple field changes | `thask.node.update` |
| Many nodes, partial field changes (e.g. re-parent 67 nodes) | `thask.node.batch_update` |
| Wire up many edges between existing nodes | `thask.edge.batch_create` |
| Remove many edges | `thask.edge.batch_delete` |
| Migrate an external graph (creates NEW node ids) | `thask.graph.import` |
| Propose a description change (agent keys) | `thask.node.suggest_update` |

### Response shape

`200 OK` when every item applied; **`207 Multi-Status`** when any items were
skipped (other items still applied successfully):

```json
{
  "data": {
    "updated": [
      { "nodeId": "uuid", "fieldsChanged": ["title", "parent_id"] }
    ],
    "skipped": [
      { "nodeId": "uuid", "reason": "not_found" }
    ],
    "batchId": "11111111-1111-4111-8111-111111111111"
  }
}
```

`403 Forbidden` when the calling key lacks the required permission for any
field in the batch — atomic: the whole batch is rejected (no partial writes).

---

## See Also

- [API Reference](./API.md) — Full REST API documentation, suggestion endpoints, bulk endpoints
- [Database Schema](./DATABASE.md) — `audit_log`, `node_suggestions`, provenance columns
- [Graph Model](./GRAPH.md) — Node types, edge types, and semantics
- [Architecture](./ARCHITECTURE.md) — How Thask is structured
