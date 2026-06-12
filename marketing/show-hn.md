# Show HN — draft

## Title options (pick one, ~70 char max)

1. **Show HN: Thask – MCP server that gives Claude Code your dependency graph**
2. Show HN: Stop your AI agent from guessing what your code depends on
3. Show HN: Dependency-graph MCP server with provenance guards for AI agents

## Body draft

Hi HN — I built Thask because every AI coding agent I use has the same blind spot.

You ask Claude Code "refactor this payment function" and it sees the file. It does not see the three other flows that touch the same DB row, the UI component that re-reads the cached price, or the cron job that retries on the old shape. So it writes confident code that breaks production.

Thask is a self-hosted dependency graph + MCP server. You map your system once (or let `thask scan` do it for Go and TypeScript), and then Claude Code / Cursor / Codex query it as a tool:

- `thask.impact.analyze` — "what breaks downstream of this node?"
- `thask.graph.get` — "show me the structural neighborhood"
- `thask.node.suggest_update` — agents can *propose* description changes, but a human approves before the change lands

That last part is the piece I have not seen elsewhere. Every API key has a `kind` (user_interactive | agent | service) and seven independent permission flags. Agent keys default to **blocking semantic writes** — they can update structural facts (file paths, edges) and metadata (status, position), but not the "why this exists" prose. Every write records 6-dimension provenance (actor, channel, agent model, mutation kind, trigger, evidence) to a single `audit_log` table.

So when the next agent reads the graph, it sees `description_source: "human"` vs `"agent"` and `last_verified_commit`. It knows what to trust and what to re-derive from code.

### Quickstart

```bash
docker compose up -d              # self-host
npm install -g @thask-org/cli
thask config set url http://localhost:7244
thask login                       # browser flow, 5 seconds
```

Then add to `.claude/mcp.json`:

```json
{ "mcpServers": { "thask": { "command": "thask", "args": ["mcp", "serve"] } } }
```

### Stack

Go 1.26 backend (Echo + pgx), SvelteKit 2 / Svelte 5 frontend (Cytoscape.js for the graph editor), PostgreSQL 17, CLI in Go with built-in MCP server (stdio). Apache 2.0 MIT. ~25k LOC.

### Live demo (read-only)

- Documentation graph of Thask itself: https://thask.kimgh06.com/shared/562a734b85200bbdd65a65e18e066dc377b7dcfd3c86edb1eab4e6aece9a9bbf
- Architecture graph: https://thask.kimgh06.com/shared/de5cb3d3587d2479d6c875f873bd6479f031187e4054fd8cc93eafca8c840691

Repo: https://github.com/kimgh06/Thask

I would especially love feedback on:

1. The `kind` + permissions model — is the default of "agent keys cannot write description" too restrictive, or right?
2. The 6-dimension audit schema — anything obvious missing?
3. Scanner coverage — currently Go + TS/JS. Python next?

Thanks.

---

## Posting checklist

- [ ] Tuesday or Wednesday, ~9am ET (HN traffic peak for Show HN)
- [ ] Have a 30s demo GIF ready to link in first comment
- [ ] Watch first 2 hours, answer every comment within 15 min
- [ ] Do not crosspost to /r/programming the same day — wait 48h
- [ ] LinkedIn version delayed by 1 week with different framing (less technical)

## Follow-up channels (if Show HN traction is decent)

- Anthropic MCP Registry: submit `marketing/server.json` via PR to modelcontextprotocol/registry
- r/ClaudeAI weekly thread
- /r/cursor showcase
- Indie Hackers (community-builder angle, not technical)
- Personal blog post on the "agents hallucinating descriptions" failure mode — the audit_log design rationale is the meat

## Conversion goal

Plan target: 300 → 1000 npm downloads within 30 days. Track:
- npm download counts (npmjs.com weekly stats)
- GitHub stars delta
- MCP Registry install events (if available)
- thask.kimgh06.com signups
