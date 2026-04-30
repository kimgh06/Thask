# Thask Claude Code Plugin

Auto-injects your Thask project graph context (recent mistakes, in-progress tasks, dependencies) into every Claude Code session, and registers the Thask MCP server.

## Install

This plugin lives in a monorepo subdirectory, so install via the bundled marketplace:

```
/plugin marketplace add kimgh06/Thask
/plugin install thask@thask
```

The first command registers `plugin/thask-claudecode/.claude-plugin/marketplace.json`; the second installs the `thask` plugin from that marketplace.

### Local development

```
claude --plugin-dir ./plugin/thask-claudecode
```

Use `/reload-plugins` after editing files in this directory.

## Prerequisites

- `thask` CLI on PATH (`npm i -g @thask-org/cli`)
- `thask config set url <your-thask-instance>` and `thask config set token <api-key>`

If either is missing, the plugin no-ops silently.

## What it does

- **SessionStart hook** runs `thask guide` and prepends the output as session context.
- **MCP server** registers `thask mcp serve` so the agent can query/edit your graph.

## Versioning

Plugin version follows the CLI version (single source of truth: git tag).
After tagging a release, run:

```
./scripts/sync-version.sh
```

to bump `plugin.json` to match.
