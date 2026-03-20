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
2. **API token** — create via web UI (Settings > API Keys) or set via CLI config
3. **Thask CLI installed** — `go install` or download binary from releases

---

## Setup

### Claude Code

Create or edit `.claude/mcp.json` in your project root:

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

The MCP server exposes 12 tools organized into 4 categories.

### Node Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.node.list` | List nodes in a project | `projectId` | `type`, `status` |
| `thask.node.create` | Create a new node | `projectId`, `type`, `title` | `description`, `status`, `tags`, `positionX`, `positionY` |
| `thask.node.get` | Get node details, connected edges, and history | `projectId`, `nodeId` | — |
| `thask.node.update` | Update node fields | `projectId`, `nodeId` | `title`, `status`, `type`, `description`, `tags` |
| `thask.node.delete` | Delete a node | `projectId`, `nodeId` | — |
| `thask.node.batch_status` | Batch update status for multiple nodes | `projectId`, `ids`, `status` | — |

### Edge Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.edge.list` | List all edges in a project | `projectId` | — |
| `thask.edge.create` | Create a relationship between nodes | `projectId`, `sourceId`, `targetId` | `edgeType`, `label` |
| `thask.edge.delete` | Delete an edge | `projectId`, `edgeId` | — |

### Graph Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.graph.get` | Get full graph snapshot (all nodes and edges) | `projectId` | — |
| `thask.graph.import` | Import graph from JSON (replace or merge) | `projectId`, `mode`, `nodes`, `edges` | — |

### Analysis Tools

| Tool | Description | Required | Optional |
|---|---|---|---|
| `thask.impact.analyze` | Analyze cascade impact of changing a node | `projectId`, `nodeId` | — |

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

For CLI troubleshooting:

```bash
# Test CLI connection
thask node list --project <projectId>

# Verify config
thask config

# Check token validity
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:7244/api/auth/me
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

## See Also

- [API Reference](./API.md) — Full REST API documentation
- [Graph Model](./GRAPH.md) — Node types, edge types, and semantics
- [Architecture](./ARCHITECTURE.md) — How Thask is structured
