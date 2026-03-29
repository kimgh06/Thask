# @thask-org/cli

Graph-based dependency visualization for AI-assisted development.

## Install

```bash
npm install -g @thask-org/cli
```

Or use directly with npx:

```bash
npx @thask-org/cli version
```

## Quick Start

```bash
# Configure
thask config set url http://localhost:7244
thask config set token <your-api-key>

# Scan a Go project
thask scan --path . --dry-run

# Analyze dependencies
thask graph analyze -p <projectId>
```

## MCP Integration (Claude Code / Cursor)

Add to `.claude/mcp.json`:

```json
{
  "mcpServers": {
    "thask": {
      "command": "npx",
      "args": ["-y", "@thask-org/cli", "mcp", "serve", "--url", "http://localhost:7244", "--token", "<key>"]
    }
  }
}
```

## Links

- [GitHub](https://github.com/kimgh06/Thask)
- [Documentation](https://github.com/kimgh06/Thask/tree/main/docs)
- [CLI Reference](https://github.com/kimgh06/Thask/blob/main/docs/CLI.md)
- [MCP Guide](https://github.com/kimgh06/Thask/blob/main/docs/MCP.md)
