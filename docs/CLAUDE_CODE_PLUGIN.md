# Claude Code Plugin

Thask ships an official Claude Code plugin that injects your project graph context into every session and registers the Thask MCP server. Once installed, your AI agent automatically receives Thask conventions, available tools, and (when [T4 dynamic guide](#roadmap) lands) recent mistakes and in-progress tasks at session start.

## Why use it

Without the plugin, every session starts with an agent that doesn't know:
- What Thask is or how to use its tools
- Which nodes/edges already exist in your project
- What you've previously gotten wrong

With the plugin, all of this is in the system context before the first user message.

## Architecture

```
plugin/thask-claudecode/
├── .claude-plugin/
│   ├── plugin.json          # Plugin metadata (version synced with CLI)
│   └── marketplace.json     # git-subdir source for monorepo install
├── hooks/
│   ├── hooks.json           # Hook registration
│   └── session-start.sh     # Runs `thask guide`, prints to stdout
├── .mcp.json                # Registers `thask mcp serve` as an MCP server
├── scripts/
│   └── sync-version.sh      # Bumps plugin.json from git tag
└── README.md
```

## Install

The plugin lives in the Thask monorepo (`plugin/thask-claudecode/`), so installation goes through the bundled marketplace.

```
/plugin marketplace add kimgh06/Thask
/plugin install thask@thask
```

After install, restart Claude Code or run `/reload-plugins`.

### Local development

To iterate on the plugin without pushing to GitHub:

```
claude --plugin-dir /path/to/Thask/plugin/thask-claudecode
```

Edit files, then `/reload-plugins` to pick up changes.

## Prerequisites

The plugin no-ops silently if the CLI is not configured. Before installation, run:

```bash
npm install -g @thask-org/cli
thask config set url <your-thask-url>
thask config set token <your-api-key>
thask config set project <your-project-id>
```

See [CLI.md](CLI.md) for full configuration reference.

## What the plugin does

### SessionStart hook

On every new session, `hooks/session-start.sh` runs:

```bash
thask guide
```

The output is prepended to the session as additional context, so the agent knows:
- The 16 MCP tools and their parameters
- Node/edge type conventions (FLOW, BRANCH, TASK, BUG, API, UI, GROUP)
- Status enum (PASS, FAIL, IN_PROGRESS, BLOCKED) — agents won't try invalid values like `TODO`
- Recommended workflows (impact analysis, batch updates)
- Known pitfalls

If the CLI isn't installed or configured, the hook exits cleanly with no output.

### MCP server registration

`.mcp.json` registers `thask mcp serve` so Claude Code spawns the Thask MCP server automatically. This exposes 16 tools to the agent:

| Category | Tools |
|----------|-------|
| Node | `list`, `create`, `get`, `update`, `delete`, `batch_status` |
| Edge | `list`, `create`, `delete` |
| Graph | `get`, `import`, `layout`, `analyze` |
| Analysis | `impact.analyze`, `scan.run` |
| Meta | `guide` |

See [MCP.md](MCP.md) for tool-level details.

## Versioning

The plugin version is locked to the CLI version (single source of truth: git tag). After tagging a CLI release:

```bash
cd plugin/thask-claudecode
./scripts/sync-version.sh
```

This bumps `.claude-plugin/plugin.json` to match `git describe --tags`.

## Verifying installation

After installing, in a new Claude Code session:

1. The session context should contain a `## Thask project context` block at the top with the 138-line guide
2. The tool list should include `mcp__thask__*` entries (16 tools)
3. Asking the agent "what Thask projects do I have?" should produce a `thask.project.list`-style call

If neither shows up, check:
- `thask config show` returns a configured URL/token
- `thask guide` prints output on its own
- The plugin is in the active plugin directory (`/plugin list`)

## Roadmap

The current plugin is v0.1 — it injects a static guide. Planned upgrades (tracked in the Thask `todo` project):

- **Dynamic `thask.guide`** — include user's recent BUG nodes (mistakes), IN_PROGRESS tasks, and FAIL-status items
- **PreToolUse hook** — warn before commands match a previously-recorded mistake pattern
- **UserPromptSubmit hook** — auto-detect "그게 아니고", "왜 X 했냐" patterns to surface mistake-recording prompts
- **`thask init` command** — single-command setup that installs the plugin, configures the CLI, and patches CLAUDE.md

## Related docs

- [CLI.md](CLI.md) — CLI command reference
- [MCP.md](MCP.md) — MCP tool reference and Cursor setup
- [ARCHITECTURE.md](ARCHITECTURE.md) — System architecture
