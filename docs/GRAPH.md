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

## Node Statuses

| Status | Border Color | Background | Description |
|---|---|---|---|
| PASS | green | light green | Verified / passing |
| FAIL | red | light red | Failed / broken |
| IN_PROGRESS | yellow | light yellow | Currently being worked on |
| BLOCKED | gray | light gray | Blocked by dependency |

## Edge Types

| Type | Color | Style | Description |
|---|---|---|---|
| depends_on | purple `#a78bfa` | solid | A depends on B |
| blocks | red `#e05252` | dashed | A blocks B |
| triggers | yellow-gold `#e2b340` | solid | A triggers B |
| related | gray `#7c7570` | solid | General relation |
| parent_child | blue `#7ca3c4` | dashed | Parent-child hierarchy |

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
2. Select a new edge type from the 5 options
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

## File Map

| File | Responsibility |
|---|---|
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
