<div align="center">

<table><tr>
<td align="center" width="120"><img src="public/icon.svg" alt="Thask" width="80" /></td>
<td align="center"><h1>Thask</h1><em>Thask it, done.</em></td>
<td align="center" width="120"><img src="public/mascot.png" alt="Thask Mascot" width="100" /></td>
</tr></table>

**Visualize product flows, tasks, and bugs as a linked node graph.**
<br />
Built for QA risk management and change impact analysis.

<br />

[![MIT License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
&nbsp;
[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](https://nextjs.org/)
&nbsp;
[![TypeScript](https://img.shields.io/badge/TypeScript-5.8-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
&nbsp;
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
&nbsp;
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

<!-- TODO: Add hero screenshot/GIF here -->

</div>

---

## Why Thask?

Spreadsheets lose context. Linear issue trackers hide relationships. **Thask maps your product as a living graph** — so you can see what breaks before it breaks.

<table>
<tr>
<td width="25%" align="center">
<h4>Graph-first Thinking</h4>
<p>Every flow, task, and bug is a node. Every dependency is a visible edge. No more hidden connections.</p>
</td>
<td width="25%" align="center">
<h4>Impact at a Glance</h4>
<p>One click shows which nodes are affected by recent changes. Catch regressions before they ship.</p>
</td>
<td width="25%" align="center">
<h4>Self-hosted</h4>
<p><code>docker compose up</code> — that's it. Your data stays on your infrastructure. No vendor lock-in.</p>
</td>
<td width="25%" align="center">
<h4>Team-ready</h4>
<p>Multi-team projects with role-based access. Everyone sees the same graph.</p>
</td>
</tr>
</table>

---

## Features

### Interactive Graph Editor

Drag-and-drop nodes with **7 types** — Flow, Branch, Task, Bug, API, UI, and Group. Connect them by hovering and dragging the edge handle. Auto-layout with the fCOSE force-directed algorithm.

<!-- ![Graph Editor](docs/images/graph-editor.png) -->

### QA Impact Mode

Toggle Impact Mode to instantly highlight **changed nodes** and their **downstream dependencies**. Dimmed nodes are safe; glowing nodes need attention.

<!-- ![Impact Mode](docs/images/impact-mode.png) -->

### Group Nodes

Organize related nodes into collapsible groups. Drag nodes in and out. Resize groups freely. Double-click to collapse with a child count badge.

<!-- ![Group Nodes](docs/images/group-nodes.png) -->

### Status Tracking & Filters

Track every node as `PASS` / `FAIL` / `IN_PROGRESS` / `BLOCKED` with color-coded visuals. Filter the graph by node type or status to focus on what matters.

### Node Detail Panel

Slide-out panel with full editing — title, description, type, status, group membership, connected nodes, and a complete change history audit log.

### Edge Relationships

Five edge types with distinct colors: `depends_on`, `blocks`, `related`, `parent_child`, `triggers`. Click any edge to change its type or delete it.

---

## Quick Start

### Docker (recommended)

```bash
docker compose up
```

Open [http://localhost:7243](http://localhost:7243) and create an account.

### Local Development

```bash
# 1. Start PostgreSQL
docker compose -f docker-compose.dev.yml up -d

# 2. Install & configure
npm install
cp .env.example .env

# 3. Set up database
npm run db:push

# 4. Start dev server
npm run dev
```

Open [http://localhost:7243](http://localhost:7243)

---

## Tech Stack

| Layer | Technology |
|---|---|
| **Framework** | Next.js 15 (App Router, Turbopack) |
| **Language** | TypeScript 5.8 (strict) |
| **Graph Engine** | Cytoscape.js + fCOSE layout + edgehandles |
| **Styling** | Tailwind CSS v4 |
| **State** | Zustand (UI) + TanStack Query (server) |
| **Database** | PostgreSQL 17 + Drizzle ORM |
| **Auth** | Session-based (bcrypt + HTTP-only cookies) |
| **Validation** | Zod |
| **Deploy** | Docker Compose |

---

## Project Structure

```
src/
  app/
    (auth)/          # Login & register pages
    (dashboard)/     # Dashboard & project graph pages
    api/             # REST API routes (auth, teams, projects, nodes, edges)
  components/
    auth/            # LoginForm, RegisterForm
    graph/           # CytoscapeCanvas, GraphToolbar, AddNodeModal, EdgeColorPopover
    layout/          # Header, Sidebar, Providers
    panels/          # NodeDetailPanel
    ui/              # ConfirmDialog and shared UI
  lib/
    auth/            # Session management, password hashing, route guards
    cytoscape/       # Graph styles, layouts, impact mode logic
    db/              # Drizzle schema & connection
  stores/            # Zustand stores (auth, graph state)
  types/             # TypeScript type definitions
```

---

## Data Model

```
Users ──< TeamMembers >── Teams ──< Projects ──< Nodes ──< NodeHistory
                                                    │
                                                    └──< Edges
```

**Node types:** `FLOW` `BRANCH` `TASK` `BUG` `API` `UI` `GROUP`
**Node statuses:** `PASS` `FAIL` `IN_PROGRESS` `BLOCKED`
**Edge types:** `depends_on` `blocks` `related` `parent_child` `triggers`

---

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://thask:thask_dev_password@localhost:7242/thask` |
| `SESSION_SECRET` | Random string for session signing | — |
| `NEXT_PUBLIC_APP_URL` | Public app URL | `http://localhost:7243` |
| `NODE_ENV` | Environment | `development` |

See [.env.example](.env.example) for a ready-to-copy template.

---

## Roadmap

- [ ] Real-time collaboration (WebSocket)
- [ ] Export graph as image / PDF
- [ ] Node search & keyboard shortcuts
- [ ] API token auth for CI/CD integration
- [ ] Graph templates (preset flow patterns)
- [ ] Dark mode

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions and guidelines.

```bash
npm run lint        # Lint check
npm run format      # Format code
npm run type-check  # TypeScript check
```

---

<div align="center">

## License

[MIT](LICENSE) &copy; Thask Contributors

**Thask it, done.**

[Report Bug](../../issues) &middot; [Request Feature](../../issues)

</div>