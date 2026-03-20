# Thask CLI

Command-line interface and MCP server for Thask.

## Quick Start

```bash
# Build
go build -o thask ./cmd/thask

# Configure
./thask config set url http://localhost:7244
./thask config set token <your-api-key>

# Use
./thask node list --pretty
./thask impact --node <nodeId>
./thask mcp serve  # Start MCP server for Claude Code
```

## Documentation

- [CLI Reference](../docs/CLI.md) — Full command reference, flags, output formats
- [MCP Guide](../docs/MCP.md) — AI agent integration (Claude Code, Cursor)

## Structure

```
cmd/thask/          Entry point
internal/
  cmd/              Cobra commands (node, edge, team, project, graph, impact, etc.)
  mcp/              MCP server (stdio protocol, 12 tools)
  client/           HTTP client for backend API
  config/           Config file management (~/.config/thask/)
  output/           Output formatting (JSON, table, quiet)
```
