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
thask login                         # browser flow, writes token to ~/.thask/config.json
thask config set project <your-project-id>
```

`thask login` (v0.5.11+) replaces the old "create a key in the web UI, copy
the 64-char string, run `thask config set token <paste>`" dance. The MCP
server reads the same `~/.thask/config.json`, so once `thask login`
succeeds the plugin works without additional setup.

For SSH / headless sessions where a browser isn't available, fall back to
the manual path: create a key in the web UI (Settings > API Keys) and run
`thask config set token <key>`.

See [CLI.md](CLI.md) for the full configuration reference.

## What the plugin does

### SessionStart hook

On every new session, `hooks/session-start.sh` runs:

```bash
thask guide
```

The output is prepended to the session as additional context, so the agent knows:
- The 25 MCP tools and their parameters (incl. v0.5.9 suggestion queue, v0.5.10 bulk endpoints, v0.5.11 login flow, v0.5.16 edge.update + CLI parity, v0.6.0 Knowledge OS types and lifecycle state)
- Node type conventions — 11 as of v0.6.0: `FLOW`, `BRANCH`, `TASK`, `BUG`, `API`, `UI`, `GROUP`, plus `REQUIREMENT`, `DECISION`, `EXPERIMENT`, `PERSON`
- Edge type conventions — 14 as of v0.6.0: `depends_on`, `blocks`, `related`, `parent_child`, `triggers`, plus `realizes`, `conflicts`, `drives`, `supersedes`, `tests`, `produced`, `owns`, `decided`, `reported`
- Status enum (PASS, FAIL, IN_PROGRESS, BLOCKED) — agents won't try invalid values like `TODO`
- Lifecycle state (v0.6.0) — free-form text on the four Knowledge OS types, orthogonal to `status`
- Recommended workflows (impact analysis, batch updates, suggest-then-verify)
- Known pitfalls (incl. why agent-kind keys can't write descriptions directly)

If the CLI isn't installed or configured, the hook exits cleanly with no output.

### MCP server registration

`.mcp.json` registers `thask mcp serve` so Claude Code spawns the Thask MCP server automatically. This exposes 25 tools to the agent:

| Category | Tools |
|----------|-------|
| Node | `list`, `create`, `get`, `update`, `delete`, `batch_status`, `batch_update`, `suggest_update`, `verify` |
| Edge | `list`, `create`, `update`, `delete`, `batch_create`, `batch_delete` |
| Graph | `get`, `import`, `layout`, `analyze` |
| Analysis | `impact.analyze`, `scan.run` |
| Suggestions | `list`, `decide` |
| Meta | `guide`, `mistake.record` |

`node.update` blocks writes to semantic fields (`description`, `tags`) for
agent-kind API keys by default — agents must go through `node.suggest_update`
and have a human run `suggestions.decide`. See
[MCP.md > Provenance Tools](MCP.md#provenance-tools-v059) for the design
rationale.

See [MCP.md](MCP.md) for tool-level details.

## Versioning

The plugin version is locked to the CLI version (single source of truth: git tag). Current release: **v0.6.0** (`plugin/thask-claudecode/.claude-plugin/plugin.json`). After tagging a CLI release:

```bash
cd plugin/thask-claudecode
./scripts/sync-version.sh
```

This bumps `.claude-plugin/plugin.json` to match `git describe --tags`.

## Verifying installation

After installing, in a new Claude Code session:

1. The session context should contain a `## Thask project context` block at the top with the 138-line guide
2. The tool list should include `mcp__thask__*` entries (25 tools)
3. Asking the agent "what Thask projects do I have?" should produce a `thask.project.list`-style call

**One-shot check from the terminal:** `thask doctor` walks the full stack
(binary, config, URL, server reachability + DB + migration version, token
validity + permissions, MCP entries in `~/.claude.json`, `~/.cursor/mcp.json`,
and `~/.codex/config.toml`) and prints `✓` / `⚠` / `✗` per check with a hint
on every failure. Always the first command to run when the plugin "isn't
working." Full reference: [CLI.md#doctor](./CLI.md#doctor).

If `doctor` itself is missing some context, manual probes:
- `thask config show` returns a configured URL/token
- `thask guide` prints output on its own
- The plugin is in the active plugin directory (`/plugin list`)

## Roadmap

The current plugin is v0.1 — it injects a static guide. Planned upgrades (tracked in the Thask `todo` project):

- **PreToolUse hook** — warn before commands match a previously-recorded mistake pattern
- **UserPromptSubmit hook** — auto-detect "그게 아니고", "왜 X 했냐" patterns to surface mistake-recording prompts

Shipped: dynamic `thask.guide` (v0.5.9, includes recent mistakes / IN_PROGRESS / blockers), `thask init` (v0.5.x, configures CLI + patches CLAUDE.md + Cursor/Codex MCP entries), `thask login` (v0.5.11, browser-based auth), `thask doctor` (v0.5.12, full-stack diagnostic).

## Related docs

- [CLI.md](CLI.md) — CLI command reference
- [MCP.md](MCP.md) — MCP tool reference and Cursor setup
- [ARCHITECTURE.md](ARCHITECTURE.md) — System architecture
