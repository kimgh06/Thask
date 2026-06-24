# Thask CLI

Command-line interface and MCP server for Thask.

## Quick Start

```bash
# Install
npm i -g @thask-org/cli

# Configure (interactive — also patches Claude Code / Cursor / Codex configs)
thask init

# Use
thask node list --pretty
thask impact --node <nodeId>
thask mcp serve  # Start MCP server for Claude Code / Cursor / Codex

# Inspect your own CLI usage (v0.5.15+, local-only — no upload)
thask usage              # 30-day summary
thask reflog -n 20       # alias: thask history
thask telemetry status   # what's collected + opt-in state
```

Build from source instead:

```bash
go build -o thask ./cmd/thask
```

## Documentation

- [CLI Reference](../docs/CLI.md) — Full command reference, flags, output formats
- [MCP Guide](../docs/MCP.md) — AI agent integration (Claude Code, Cursor)

## Structure

```
cmd/thask/          Entry point
internal/
  cmd/              Cobra commands (node, edge, team, project, graph, impact, telemetry, usage, reflog, etc.)
  mcp/              MCP server (stdio protocol, 24 tools)
  client/           HTTP client for backend API
  config/           Config file management (~/.thask/config.json)
  output/           Output formatting (JSON, table, quiet)
  telemetry/        Local-first event log (~/.thask/events.jsonl) — v0.5.15+
  scan/             Go codebase dependency scanner
  update/           Background update-check
```
