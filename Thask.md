# Thask — AI Agent Guide

Thask is a graph-based dependency visualization tool. This guide defines how AI agents
should create, update, and query Thask graphs via MCP tools or CLI.

> This guide is also available via the MCP tool `thask.guide` and the CLI command `thask guide`.

## MCP Tools (16)

### Node (6)
| Tool                    | Required                    | Optional                                    |
|-------------------------|-----------------------------|---------------------------------------------|
| thask.node.list         | projectId                   | type, status                                |
| thask.node.create       | projectId, type, title      | description, status, tags, positionX/Y      |
| thask.node.get          | projectId, nodeId           | —                                           |
| thask.node.update       | projectId, nodeId           | title, status, type, description, tags      |
| thask.node.delete       | projectId, nodeId           | —                                           |
| thask.node.batch_status | projectId, ids, status      | —                                           |

### Edge (3)
| Tool               | Required                       | Optional        |
|--------------------|--------------------------------|-----------------|
| thask.edge.list    | projectId                      | —               |
| thask.edge.create  | projectId, sourceId, targetId  | edgeType, label |
| thask.edge.delete  | projectId, edgeId              | —               |

### Graph (4)
| Tool                | Required                          | Optional  |
|---------------------|-----------------------------------|-----------|
| thask.graph.get     | projectId                         | —         |
| thask.graph.import  | projectId, mode, nodes, edges     | —         |
| thask.graph.layout  | projectId                         | algorithm |
| thask.graph.analyze | projectId                         | —         |

### Analysis & Scan (2)
| Tool                 | Required          | Optional |
|----------------------|-------------------|----------|
| thask.impact.analyze | projectId, nodeId | —        |
| thask.scan.run       | projectId, path   | maxFiles |

### Meta (1)
| Tool         | Required | Optional |
|--------------|----------|----------|
| thask.guide  | —        | —        |

---

## Node Types — when to use each

| Type   | Use for                                  |
|--------|------------------------------------------|
| TASK   | Work items, to-dos, implementation tasks |
| FLOW   | User journeys, product flows             |
| BUG    | Defects, issues                          |
| API    | REST/GraphQL endpoints                   |
| UI     | Frontend pages, components               |
| BRANCH | Conditional decisions, if/else           |
| GROUP  | Logical container (folder-like grouping) |

## Statuses

| Status      | Meaning                      |
|-------------|------------------------------|
| IN_PROGRESS | Being worked on (default)    |
| PASS        | Verified, complete           |
| FAIL        | Broken — triggers waterfall  |
| BLOCKED     | Waiting on a dependency      |

**Waterfall**: Setting a node to FAIL automatically propagates BLOCKED to downstream nodes.

## Edge Types — direction matters

| Type         | source -> target means               | Example                                      |
|--------------|--------------------------------------|----------------------------------------------|
| depends_on   | source NEEDS target                  | "Login Page" depends_on "Auth API"           |
| blocks       | source PREVENTS target               | "DB Migration Bug" blocks "Deploy"           |
| triggers     | source STARTS target                 | "Payment Complete" triggers "Send Receipt"   |
| related      | general association (no direction)   | "Search UI" related "Filter UI"              |
| parent_child | hierarchy (rarely used — prefer GROUP parentId) |                                     |

---

## Rules

1. **Read first**: Always call `thask.graph.get` before modifying an existing graph.
2. **Import for bulk**: Use `thask.graph.import` (merge) when creating 3+ nodes. Individual create only for 1-2 nodes.
3. **Layout after create**: Agents cannot see the canvas. Always call `thask.graph.layout` (dagre) after creating nodes.
4. **Merge by default**: Never use import mode "replace" unless the user explicitly asks to rebuild from scratch.
5. **No duplicates**: Check existing nodes/edges via `graph.get` before creating new ones.
6. **Status defaults**: Omitted status = IN_PROGRESS. Always set PASS explicitly for completed items.
7. **Edge direction**: `depends_on` and `blocks` are NOT interchangeable. A depends_on B != A blocks B.

---

## Workflows

### Create a dependency map (most common)
```
1. thask.graph.get           -> read existing graph
2. thask.graph.import(merge) -> batch create nodes + edges
3. thask.graph.layout(dagre) -> auto-position everything
```

Import example:
```json
{
  "projectId": "...",
  "mode": "merge",
  "nodes": [
    { "id": "tmp-1", "type": "API",  "title": "POST /orders" },
    { "id": "tmp-2", "type": "UI",   "title": "Order Page" },
    { "id": "tmp-3", "type": "TASK", "title": "Payment Integration" }
  ],
  "edges": [
    { "sourceId": "tmp-2", "targetId": "tmp-1", "edgeType": "depends_on" },
    { "sourceId": "tmp-1", "targetId": "tmp-3", "edgeType": "depends_on" }
  ]
}
```
`tmp-*` IDs are remapped to real UUIDs by the server.

### Update existing nodes
```
1. thask.graph.get             -> find node IDs
2. thask.node.update           -> individual update
   OR thask.node.batch_status  -> bulk status change
```

### Analyze impact before changes
```
1. thask.impact.analyze -> downstream cascade for a specific node
2. thask.graph.analyze  -> cycle detection + critical path
```

---

## Pitfalls

- DO NOT confuse `depends_on` with `blocks` — check the direction table above
- DO NOT create nodes one-by-one when you have 3+ — use `graph.import`
- DO NOT use "replace" mode unless explicitly asked — it deletes ALL existing data
- DO NOT skip `graph.layout` after bulk creation — nodes stack at (0,0)
- DO NOT create self-loop edges — server rejects them
- DO NOT create duplicate edges between the same node pair
- Deleting a GROUP node orphans its children (parentId -> null)
