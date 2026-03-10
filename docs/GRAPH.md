# Graph Engine

Thask uses [Cytoscape.js](https://js.cytoscape.org/) with the **fCOSE** force-directed layout and **edgehandles** extension for interactive edge creation.

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
- **Toolbar:** Click "Add Group" button
- **Keyboard:** Select 2+ nodes → `Ctrl+G`
- **API:** POST node with `type: "GROUP"`

### Adding Nodes to Groups
- **Drag & Drop:** Drag a node over a GROUP → visual drop target highlight
- **API:** PATCH node with `parentId: "group-id"`

### Removing from Group
- **Detail Panel:** Click "Remove" in group membership section
- **Keyboard:** Select nodes → `Ctrl+Shift+G`
- **API:** PATCH node with `parentId: null`

### Collapsing
- **Double-click** a GROUP to collapse/expand
- Collapsed groups show a child count badge
- Children are hidden when collapsed

### Resizing
- Drag the resize handle (bottom-right corner) on a GROUP node
- Minimum size: 130×80px

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

Reads stored `positionX` / `positionY` from the database. Used when loading an existing graph.

---

## Impact Mode

Highlights nodes affected by recent changes for QA risk assessment.

### How It Works

1. **Activation:** Toggle Impact Mode in the toolbar
2. **API call:** `GET /api/projects/[id]/impact?since=7d&depth=2`
3. **Changed nodes:** Orange glow border (5px)
4. **Affected nodes:** Orange border (4px) — downstream via BFS
5. **Unaffected nodes:** Dimmed to 15% opacity
6. **Deactivation:** Toggle off → all classes removed

### Status Propagation (Waterfall)

When a node's status changes to PASS or FAIL, the waterfall algorithm propagates status changes downstream:

1. Find all edges where the changed node is the source
2. For each target node, re-evaluate status based on all incoming edges
3. Recurse up to depth 10 (cycle prevention)
4. Parent GROUP nodes re-evaluate based on children's statuses

---

## Undo/Redo

Stack-based system with max 50 entries.

### Supported Actions

| Action | Undo Behavior | Redo Behavior |
|---|---|---|
| `addNode` | Delete the node | Re-add the node |
| `deleteNode` | Re-add the node + edges | Delete again |
| `updateNode` | Revert to old values | Apply new values |
| `addEdge` | Delete the edge | Re-add the edge |
| `deleteEdge` | Re-add the edge | Delete again |
| `updateEdgeType` | Revert edge type | Apply new type |
| `updateEdgeLabel` | Revert label | Apply new label |
| `dropOnGroup` | Remove from group | Re-add to group |
| `groupNodes` | Ungroup all + delete GROUP | Re-group |
| `ungroupNodes` | Re-add to parent GROUP | Ungroup again |

### Keyboard Shortcuts

- `Ctrl+Z` — Undo
- `Ctrl+Shift+Z` — Redo

---

## Interactive Features

### Edge Creation
1. Hover over a node → edge handle appears (small circle)
2. Drag from handle to another node
3. Edge is created with default type `related`

### Edge Editing
1. Click an edge → EdgeColorPopover appears
2. Select a new edge type from the 5 options
3. Or click delete to remove the edge

### Node Selection
- **Click:** Select single node → opens NodeDetailPanel
- **Ctrl+Click:** Toggle multi-selection
- **Click canvas:** Clear selection

### Search & Focus
- Type in the toolbar search bar
- Matching nodes get a pulse highlight (orange, 2s)
- Graph pans and zooms to the focused node

---

## File Map

| File | Responsibility |
|---|---|
| `src/lib/cytoscape/styles.ts` | 60+ Cytoscape style rules for nodes, edges, and interaction classes |
| `src/lib/cytoscape/layouts.ts` | fCOSE and preset layout configurations |
| `src/lib/cytoscape/groupHelpers.ts` | `getChildNodes()`, `getDescendantNodes()`, `getDescendantIdSet()` |
| `src/lib/cytoscape/impact.ts` | `activateImpactMode()`, `deactivateImpactMode()` |
| `src/lib/waterfall.ts` | `computeWaterfall()` — status propagation algorithm |
| `src/components/graph/CytoscapeCanvas.tsx` | Main canvas component with all interactions |
| `src/components/graph/GraphToolbar.tsx` | Toolbar with zoom, layout, filters, search, undo/redo |
| `src/components/graph/GraphMinimap.tsx` | Overview minimap |
| `src/components/graph/AddNodeModal.tsx` | Node creation modal |
| `src/components/graph/EdgeColorPopover.tsx` | Edge type selection popover |
| `src/stores/useGraphStore.ts` | Selection, filters, impact mode, collapsed groups |
| `src/stores/useUndoStore.ts` | Undo/redo stack management |
| `src/hooks/useGraphData.ts` | React Query CRUD for nodes and edges |
| `src/hooks/useUndoRedo.ts` | Undo/redo execution logic |
| `src/types/graph.ts` | GraphNode, GraphEdge, NodeType, NodeStatus, EdgeType |
| `src/types/undo.ts` | UndoEntry union type (10 actions) |
