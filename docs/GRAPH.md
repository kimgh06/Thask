# Graph Engine

Thask uses [Cytoscape.js](https://js.cytoscape.org/) with the **fCOSE** force-directed layout and **edgehandles** extension for interactive edge creation.

Frontend: SvelteKit + Svelte 5 components.

---

## Node Types

| Type | Shape | Color | Description |
|---|---|---|---|
| FLOW | round-rectangle | blue `#3b82f6` | Product flow / user journey |
| BRANCH | diamond | purple `#8b5cf6` | Conditional branch / decision |
| TASK | rectangle | cyan `#06b6d4` | Work item / task |
| BUG | hexagon | red `#ef4444` | Bug / defect |
| API | barrel | orange `#f97316` | API endpoint |
| UI | ellipse | green `#10b981` | UI component / screen |
| GROUP | round-rectangle (dashed) | slate `#64748b` | Container for grouping nodes |

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
| depends_on | orange | solid | A depends on B |
| blocks | red | dashed | A blocks B |
| triggers | blue | solid | A triggers B |
| related | gray | solid | General relation |
| parent_child | purple | dashed | Parent-child hierarchy |

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
1. Click an edge — EdgeColorPopover appears at the edge's position
2. Select a new edge type from the 5 options
3. Edit the label (debounced auto-save)
4. Or click delete to remove the edge

### Node Selection
- **Click node:** Select node, opens NodeDetailPanel
- **Click canvas:** Clear selection
- **Click edge:** Select edge, opens EdgeColorPopover

### Group Drag
- Dragging a GROUP moves all descendant nodes together
- Child offsets are preserved during drag

### Search & Focus
- Click "Search" or press `Ctrl+F`
- Type to filter — matching nodes get a pulse highlight (orange, 2s)
- Graph animates to center on the focused node
- Press Enter to cycle through matches

---

## File Map

| File | Responsibility |
|---|---|
| `frontend/src/lib/cytoscape/styles.ts` | 60+ Cytoscape style rules |
| `frontend/src/lib/cytoscape/layouts.ts` | fCOSE and preset layout configurations |
| `frontend/src/lib/cytoscape/groupHelpers.ts` | `getChildNodes()`, `getDescendantNodes()`, `getDescendantIdSet()` |
| `frontend/src/lib/cytoscape/impact.ts` | `activateImpactMode()`, `deactivateImpactMode()` |
| `backend/internal/service/waterfall.go` | `ComputeWaterfall()` — BFS status propagation |
| `backend/internal/service/impact.go` | `ComputeImpact()` — bidirectional BFS |
| `frontend/src/lib/components/CytoscapeCanvas.svelte` | Main canvas with all interactions |
| `frontend/src/lib/components/GraphToolbar.svelte` | Toolbar with zoom, layout, filters, search |
| `frontend/src/lib/components/AddNodeModal.svelte` | Node creation modal |
| `frontend/src/lib/components/EdgeColorPopover.svelte` | Edge type/label editing popover |
| `frontend/src/lib/components/NodeDetailPanel.svelte` | Slide-out detail panel with tabs |
| `frontend/src/lib/stores/graph.svelte.ts` | Selection, filters, impact mode, collapsed groups |
| `frontend/src/lib/types.ts` | GraphNode, GraphEdge, NodeType, NodeStatus, EdgeType |
