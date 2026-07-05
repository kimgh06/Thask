# Graph Engine

Thask uses [Cytoscape.js](https://js.cytoscape.org/) with the **fCOSE** force-directed layout and **edgehandles** extension for interactive edge creation.

Frontend: SvelteKit + Svelte 5 components.

---

## Node Types

| Type | Shape | Color | Description |
|---|---|---|---|
| FLOW | round-rectangle | yellow-gold `#e2b340` | Product flow / user journey |
| BRANCH | diamond | purple `#a78bfa` | Conditional branch / decision |
| TASK | rectangle | steel blue `#7ca3c4` | Work item / task |
| BUG | hexagon | red `#e05252` | Bug / defect |
| API | barrel | green `#5ea87a` | API endpoint |
| UI | ellipse | orange `#d4915a` | UI component / screen |
| GROUP | round-rectangle (dashed) | gray `#7c7570` | Container for grouping nodes |
| REQUIREMENT | round-rectangle | teal `#4a9fb0` | Product / user requirement (v0.6.0). Lifecycle: PROPOSED → ACCEPTED → IN_PROGRESS → DELIVERED → DEPRECATED |
| DECISION | diamond | amber `#c9942e` | Architecture decision record (v0.6.0). Lifecycle: PROPOSED → APPROVED → SUPERSEDED → REVERSED |
| EXPERIMENT | octagon | violet `#9370c4` | Hypothesis-driven probe (v0.6.0). Lifecycle: PROPOSED → RUNNING → COMPLETED → ABANDONED |
| PERSON | ellipse | warm neutral `#c48a4d` | Owner / decider / reporter (v0.6.0). Graph-visible so `BLOCKED → owns⁻¹ → PERSON` traversal works |

## Node Statuses

| Status | Border Color | Background | Description |
|---|---|---|---|
| PASS | green | light green | Verified / passing |
| FAIL | red | light red | Failed / broken |
| IN_PROGRESS | yellow | light yellow | Currently being worked on |
| BLOCKED | gray | light gray | Blocked by dependency |

**Lifecycle state (v0.6.0)** — a second, orthogonal state field on `REQUIREMENT`/`DECISION`/`EXPERIMENT`/`PERSON`. `status` tracks operational progress (PASS/FAIL/IN_PROGRESS/BLOCKED); `lifecycle_state` tracks domain phase (e.g. DECISION `APPROVED`). Server auto-stamps `lifecycle_state_changed_at` on every write to the field.

## Edge Types

| Type | Color | Style | Description |
|---|---|---|---|
| depends_on | purple `#a78bfa` | solid | A depends on B |
| blocks | red `#e05252` | dashed | A blocks B |
| triggers | yellow-gold `#e2b340` | solid | A triggers B |
| related | gray `#7c7570` | solid | General relation |
| parent_child | blue `#7ca3c4` | dashed | Parent-child hierarchy |
| realizes | teal `#4a9fb0` | solid | TASK → REQUIREMENT (v0.6.0) |
| conflicts | red `#e05252` | dashed | REQUIREMENT ↔ REQUIREMENT (v0.6.0) |
| drives | amber `#c9942e` | solid | DECISION → TASK / API / UI (v0.6.0) |
| supersedes | amber `#c9942e` | dashed | DECISION → DECISION (v0.6.0). metadata: `{reason}` |
| tests | violet `#9370c4` | solid | EXPERIMENT → REQUIREMENT / DECISION (v0.6.0) |
| produced | violet `#9370c4` | solid | EXPERIMENT → BUG / TASK (v0.6.0). metadata: `{outcome_summary}` |
| owns | warm neutral `#c48a4d` | solid | PERSON → TASK / API / UI / REQUIREMENT (v0.6.0) |
| decided | warm neutral `#c48a4d` | solid | PERSON → DECISION (v0.6.0) |
| reported | warm neutral `#c48a4d` | solid | PERSON → BUG (v0.6.0) |

**Edge metadata (v0.6.0)** — every edge now carries a `metadata` JSONB blob. Frontend / CLI is responsible for the per-edge-type shape; the server stores it as-is.

---

## GROUP Nodes

Groups are compound nodes that contain other nodes.

### Creating Groups
- **Toolbar:** Click "+ Group" button
- **API:** POST node with `type: "GROUP"`

### Adding Nodes to Groups
- **Drag & Drop:** Drag a node over a GROUP — visual drop target highlight
- **API:** PATCH node with `parentId: "group-id"`

### Removing from Group
- **Detail Panel:** Update parentId to null
- **API:** PATCH node with `parentId: null`

### Collapsing
- **Double-click** a GROUP to collapse/expand
- Collapsed groups show a child count badge
- Children and their edges are hidden when collapsed

### Resizing
- 8-directional resize handles on GROUP nodes (corners + edges)
- Minimum size enforced based on child node positions
- Positions saved automatically after resize

---

## Layouts

### fCOSE (Force-directed)

Used for automatic layout arrangement.

```
nodeRepulsion: 8000
idealEdgeLength: 120
gravity: 0.25
numIter: 2500
animate: true (500ms)
```

### Preset (Manual)

Reads stored `positionX` / `positionY` from the database. Used when loading an existing graph that has saved positions.

### Server-side Layout

The backend also provides auto-layout via `POST /api/projects/:projectId/graph/layout`:

| Algorithm | Description |
|---|---|
| `dagre` | Directed acyclic graph layout (default). Respects edge direction, layers nodes hierarchically. |
| `grid` | Simple grid layout. Nodes arranged in rows/columns. |

Both algorithms auto-size GROUP nodes to contain their children. Positions are saved to the database and broadcast via SSE.

---

## Realtime Updates (SSE)

When multiple users view the same project, changes are broadcast via Server-Sent Events:

| Event | Trigger |
|---|---|
| `node.created` | Node added |
| `node.updated` | Node title, status, type, etc. changed |
| `node.deleted` | Node removed |
| `edge.created` | Edge added |
| `edge.updated` | Edge type or label changed |
| `edge.deleted` | Edge removed |
| `graph.layout` | Auto-layout applied |
| `graph.import` | Graph imported |

Frontend connects via `EventSource` at `/api/projects/:projectId/events`. Changes trigger a debounced graph refresh (300ms).

---

## Shared View

Projects can be shared via public links. Shared views support two modes:

| Mode | Access | URL |
|---|---|---|
| `viewer` | Read-only: view graph, pan, zoom, search | `/shared/{token}` |
| `editor` | Full access: create/edit/delete nodes and edges | `/shared/{token}` |

### Features
- Full Cytoscape canvas (same as authenticated view)
- Realtime updates via SSE (debounced 300ms)
- Export to PNG/JSON (editor mode)
- Toolbar hidden in viewer mode
- Sign-in button for anonymous users
- Managed via ShareDialog component or CLI (`thask project share`)

### Security
- Share tokens are hashed (SHA256) in the database
- Disabling sharing invalidates old tokens (new token on re-enable)
- Public routes rate-limited to 5 requests/second
- 30-second middleware cache to prevent token enumeration

---

## Impact Mode

Highlights nodes affected by recent changes for QA risk assessment.

### How It Works

1. **Activation:** Toggle Impact Mode in the toolbar
2. **API call:** `GET /api/projects/:id/impact?since=7d&depth=2`
3. **Changed nodes:** Orange glow border (5px)
4. **Affected nodes:** Orange border (4px) — downstream via BFS
5. **Unaffected nodes:** Dimmed to 15% opacity
6. **Deactivation:** Toggle off — all classes removed

### Status Propagation (Waterfall)

When a node's status changes to PASS or FAIL, the waterfall algorithm propagates status changes downstream. Implemented in Go (`backend/internal/service/waterfall.go`):

1. Find all edges where the changed node is the source
2. For each target node, re-evaluate status based on all incoming edges
3. Recurse up to depth 10 (cycle prevention)
4. Parent GROUP nodes re-evaluate based on children's statuses

**Edge type behavior:**
- `blocks`: FAIL on source → BLOCKED on target; source resolves → target unblocked
- `depends_on`: all dependencies must PASS for target to be unblocked
- `triggers`: FAIL/PASS propagates forward

---

## Interactive Features

### Edge Creation (Port Overlay)
1. Hover over a node — 4 port dots appear (top, right, bottom, left)
2. Drag from a port dot to another node
3. Edge is created with default type `related`

### Edge Editing
1. Click an edge — EdgeDetailView opens in the side panel
2. Select a new edge type from the 14 options
3. Edit the label (debounced auto-save)
4. Or click delete to remove the edge

### Node Selection
- **Click node:** Select node, opens DetailSidePanel → NodeDetailView
- **Click canvas:** Clear selection
- **Click edge:** Select edge, opens DetailSidePanel → EdgeDetailView
- **Box select / Ctrl+Click:** Multi-select, opens MultiSelectView with batch operations

### Group Drag
- Dragging a GROUP moves all descendant nodes together
- Child offsets are preserved during drag

### Search & Focus
- Click "Search" or press `Ctrl+F`
- Type to filter — matching nodes get a pulse highlight (orange, 2s)
- Graph animates to center on the focused node
- Press Enter to cycle through matches

### Edge Bridge Overlay

When edges cross each other in a complex graph, Thask draws visual bridges to keep the diagram readable.

Two rendering modes are used depending on crossing geometry:

| Mode | Appearance | When Used |
|---|---|---|
| Bridge | Arc over the crossing (like a road overpass) | Edges cross at a sharp angle |
| Soft-bypass | Arc that gently routes around the crossing | Edges cross at a shallow angle |

The overlay is an SVG layer rendered on top of the Cytoscape canvas. It updates automatically when nodes move, edges change, or the viewport is zoomed/panned. No configuration is needed — bridges appear wherever crossings are detected.

---

## Export & Import

### PNG Export (client-side)
- **Toolbar:** Export button → PNG
- Full-graph capture with dark background (#131214), 2x scale
- Implemented in `frontend/src/lib/export.ts`

### Server-side Capture (CLI / API)
- **CLI:** `thask graph capture -p <projectId> --out graph.png` (also `--format svg`)
- **API:** `GET /api/v1/projects/:id/graph/capture?format=png&width=1600&height=1000&padding=80&scale=2&crop=true`
- PNG path goes through the **capture worker** (`capture/`): a Playwright + Browserless service that opens `/capture` and screenshots the rendered Cytoscape canvas. Returns 503 if `CAPTURE_URL` is not configured
- SVG path is rendered inline by the backend in `backend/internal/handler/og_image.go` — lightweight, no headless browser required
- Use SVG for small previews (OG cards, embed thumbnails); use PNG for full-fidelity exports

### JSON Export
- **Toolbar:** Export button → JSON
- Exports all nodes and edges as pretty-printed JSON with timestamp

### JSON Import
- **Toolbar:** Import button → select file
- **Replace mode:** Overwrites entire graph with imported data (transaction-safe)
- **Merge mode:** Adds imported nodes alongside existing ones, offset to the right
- **API:** `POST /api/projects/:projectId/graph/import`

---

## Batch Operations

When multiple nodes are selected (box select or Ctrl+Click), the MultiSelectView panel offers:

| Operation | Description |
|---|---|
| Batch delete | Delete all selected nodes |
| Batch status | Set status for all selected nodes |
| Batch type | Change type for all selected nodes |
| Batch add tag | Add a tag to all selected nodes |
| Create group | Create a GROUP containing all selected nodes |

---

## Provenance & Authoring (v0.5.9+)

Every node carries a small set of "who wrote this" columns alongside its
content. These don't change the visual graph model, but they shape who is
*allowed* to change which fields and how downstream consumers interpret
unfamiliar content.

### Field classes

Each field a write touches falls into one of three classes (`mutation_kind`
in the audit log):

| Class | Examples | Why it matters |
|---|---|---|
| **semantic** | `description`, "why", gotchas | Carries truth claims that don't fall out of code automatically — wrong values poison downstream agents |
| **structural** | `type`, `parent_id`, edges, file mappings | Topology and typing — derivable from code, generally safe for automation |
| **meta** | `status`, `position_x/y`, `tags`, `assignee_id` | Operational state — workflow and UI, low blast radius |

### Description authorship

Nodes track who authored the current description:

- `description_source` — `human` / `agent` / `scanner` / `import` / `unknown`
- `description_authored_by` — user id
- `description_authored_at` — timestamp
- `description_agent_model` — populated when an agent (e.g. `claude-opus-4-7`) wrote it
- `last_verified_at` / `last_verified_by` / `last_verified_commit` — when a human last confirmed it still matches the code

The detail panel surfaces these so a reader can tell **"some other agent
wrote this last week, no human has verified it"** apart from **"a teammate
wrote this with a referenced commit"** at a glance — important when an
agent is about to base a refactor decision on the description.

### Who can write what

Writes go through a per-API-key permission gate (see
[DATABASE.md](DATABASE.md#api-key-kind--per-key-permissions-migration-008)).
Agent-kind keys default to **blocking semantic writes** — they can keep
the graph's structural shape and operational state up to date, but they
can't quietly land hallucinated `description` text.

Instead of writing directly, an agent posts to the **suggestion queue**
(`node_suggestions` table; `thask.node.suggest_update` MCP tool). A human
later approves or rejects via the UI / `thask.suggestions.decide`. On
approve, the human becomes the author of record — the agent is only
credited in `audit_log.metadata`.

### Verification

`thask.node.verify` (or the verify button in the detail panel) stamps the
`last_verified_*` columns. Default-blocked for agent keys: the whole point
is that a human looked at the code and said "still true". Agents can be
granted `permissions.verify=true` if you accept the trade-off.

### Practical impact on graph editing

- **Bulk node creation by agents stays fine** — `node.batch_update` and
  `graph.import` continue to work for agent keys as long as they're not
  setting `description` (structural fields only).
- **Description edits become a two-step dance** in agent flows: propose,
  then have a human approve. Most agent runs that *would* have edited a
  description in v0.5.8 now suggest instead.
- **The detail panel's history tab** surfaces both the `audit_log` (who
  did what, when, via which client / MCP / agent model) and any pending
  suggestions for the node.

---

## Handoff Convention (v0.5.13+)

When a team member leaves or a new engineer joins, the Thask graph should be
the first thing they open — not a stale Confluence page. For that to work,
node descriptions need enough context to be self-explanatory.

### Recommended description structure

Use this markdown skeleton in the `description` field of any node that owns
non-obvious logic:

```markdown
## Why
Why does this exist? What problem does it solve? (1–3 sentences)

## Q&A
- **What triggers this?** — ...
- **What can go wrong?** — ...
- **Who owns this?** — ...

## Gotchas
- Non-obvious constraint or footgun
- Hidden dependency that doesn't appear in the graph

## See also  *(optional)*
- [Slack thread](https://...) — decision from 2025-11
- Linear ticket ENG-4201
```

The headings are a convention, not enforced by the schema. Feel free to add or
drop sections — the point is that a new engineer reading the node can answer
their first three questions without pinging anyone.

### Example: FLOW node — Stripe webhook handler

```markdown
## Why
Stripe delivers payment events (charge.succeeded, payment_intent.failed, etc.)
asynchronously. This flow receives those webhooks, verifies the signature, and
routes them to the correct domain handler. Without it, subscriptions would
never activate and failed payments would go unnoticed.

## Q&A
- **Why not handle Stripe events inline in the checkout API?** — Checkout is
  synchronous; Stripe can retry webhooks for hours, so they need their own
  idempotency layer.
- **What happens if the handler returns a non-2xx?** — Stripe retries with
  exponential backoff for up to 72 hours. After that the event is dropped.
- **How do I test locally?** — `stripe listen --forward-to localhost:3000/webhooks/stripe`

## Gotchas
- Signature verification uses the **endpoint-specific** signing secret, not the
  API key. Using the wrong secret silently rejects all events.
- `charge.succeeded` and `payment_intent.succeeded` can both fire for the same
  payment. The handler deduplicates by `stripeEventId` in the `webhook_events`
  table.

## See also
- [Stripe webhook docs](https://stripe.com/docs/webhooks)
- Slack #payments — "why we moved off Sidekiq for webhooks" (2025-09-03)
```

### Example: TASK node — Migrate payment retry to Sidekiq

```markdown
## Why
Our current retry loop is a cron job that polls the DB every minute. Under
load it causes lock contention on the `invoices` table. Moving retries to
Sidekiq gives per-job backoff and removes the polling.

## Q&A
- **Is this safe to deploy incrementally?** — Yes. The cron and Sidekiq
  workers can coexist; set `SIDEKIQ_RETRY_ENABLED=true` in staging first.
- **What queue does this use?** — `payments_retry` — separate from the default
  queue so a retry storm doesn't starve other jobs.
- **Who approved the queue naming?** — Platform team, see Linear ENG-5102.

## Gotchas
- The `max_attempts` column in `invoices` maps to Sidekiq's `retry:` option —
  keep them in sync when changing retry limits.
- Sidekiq's dead-job TTL is 6 months by default; adjust `dead_timeout_in_seconds`
  or dead jobs accumulate unboundedly.
```

### Example: BUG node — Duplicate charge race condition

```markdown
## Why
Under concurrent checkout (two tabs, rapid double-click), two
`payment_intent.create` calls can race and produce two charges for the same
order. Reported by three users in June 2026; reproduced in staging.

## Q&A
- **Is this in production right now?** — Yes, low frequency (~1/week). We have
  a manual refund runbook in Notion.
- **What's the root cause?** — `orders.payment_intent_id` lacks a UNIQUE
  constraint, so the idempotency check is a read-then-write race.
- **What's the fix?** — Add the DB constraint + a Redis lock around
  `checkout.confirm`. Draft in branch `fix/double-charge-lock`.

## Gotchas
- The Redis lock approach needs a timeout — don't set it below 10 s or mobile
  users on slow connections will hit false positives.
- Stripe's own idempotency key helps but doesn't fully cover the case where the
  client generates a new key on retry.

## See also
- Notion: Payment Incident Runbook
- Linear ENG-5311 (tracks the fix)
```

### Why this matters

Graphs go stale in two ways: nodes get deleted when they should be kept, and
descriptions stop matching the code. The second failure mode is the dangerous
one for the AI-agent-context-layer use case — an agent reading a stale
description makes confident, wrong decisions.

A consistent structural shape (`## Why / ## Q&A / ## Gotchas`) makes
descriptions easier to maintain: each section has a single clear owner
(domain knowledge, anticipated questions, footguns). It also makes them easier
for agents to extract context from — a model can answer "what are the gotchas
for this node?" with a targeted read rather than parsing free-form prose.

The convention pairs with the provenance system (v0.5.9+): human-authored
descriptions with a recent `last_verified_at` timestamp are the ground truth;
agent-authored descriptions with no verification are signals worth double-checking.

---

## File Map

| File | Responsibility |
|---|---|
| `frontend/src/lib/cytoscape/edgeBridgeOverlay.ts` | SVG bridge/soft-bypass overlay drawn over edge crossings |
| `frontend/src/lib/cytoscape/styles.ts` | 60+ Cytoscape style rules |
| `frontend/src/lib/cytoscape/layouts.ts` | fCOSE and preset layout configurations |
| `frontend/src/lib/cytoscape/groupHelpers.ts` | `getChildNodes()`, `getDescendantNodes()`, `getDescendantIdSet()` |
| `frontend/src/lib/cytoscape/impact.ts` | `activateImpactMode()`, `deactivateImpactMode()` |
| `frontend/src/lib/cytoscape/handlers/selection.ts` | Node/edge/multi-select tap handlers |
| `frontend/src/lib/cytoscape/handlers/edgeCreation.ts` | Edge drawing with port overlay |
| `frontend/src/lib/cytoscape/handlers/groupDrag.ts` | Drag nodes into/out of groups |
| `frontend/src/lib/managers/nodeCrud.svelte.ts` | Node CRUD + batch operations |
| `frontend/src/lib/managers/edgeCrud.svelte.ts` | Edge CRUD operations |
| `frontend/src/lib/export.ts` | PNG/JSON export, JSON import |
| `backend/internal/service/waterfall.go` | `ComputeWaterfall()` — BFS status propagation |
| `backend/internal/service/impact.go` | `ComputeImpact()` — bidirectional BFS |
| `frontend/src/lib/components/CytoscapeCanvas.svelte` | Main canvas with all interactions |
| `frontend/src/lib/components/GraphToolbar.svelte` | Toolbar with zoom, layout, filters, search |
| `frontend/src/lib/components/DetailSidePanel.svelte` | Fixed side panel (node/edge/multi-select) |
| `frontend/src/lib/components/panel/NodeDetailView.svelte` | Node detail with tabs: Info, Connected, History |
| `frontend/src/lib/components/panel/EdgeDetailView.svelte` | Edge type/label editing |
| `frontend/src/lib/components/panel/MultiSelectView.svelte` | Batch operations for multi-selection |
| `frontend/src/lib/components/AddNodeModal.svelte` | Node creation modal |
| `frontend/src/lib/components/SearchBar.svelte` | Node search with pulse highlight |
| `frontend/src/lib/stores/graph.svelte.ts` | Selection, filters, impact mode, collapsed groups |
| `frontend/src/lib/stores/undo.svelte.ts` | Undo/redo command stack |
| `frontend/src/lib/types.ts` | GraphNode, GraphEdge, NodeType, NodeStatus, EdgeType |
