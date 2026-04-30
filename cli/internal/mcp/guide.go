package mcp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/thask/cli/internal/client"
)

// GuideText is the single source of truth for AI agent interaction guidance.
// Used by: MCP thask.guide tool, CLI `thask guide` command, MCP server instructions.
const GuideText = `# Thask — AI Agent Guide

Thask is a graph-based dependency visualization tool. This guide defines how AI agents
should create, update, and query Thask graphs via MCP tools or CLI.

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
| Tool                | Required              | Optional  |
|---------------------|-----------------------|-----------|
| thask.graph.get     | projectId             | —         |
| thask.graph.import  | projectId, mode, nodes, edges | —  |
| thask.graph.layout  | projectId             | algorithm |
| thask.graph.analyze | projectId             | —         |

### Analysis & Scan (2)
| Tool                 | Required          | Optional |
|----------------------|-------------------|----------|
| thask.impact.analyze | projectId, nodeId | —        |
| thask.scan.run       | projectId, path   | maxFiles |

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

| Type         | source → target means              | Example                                      |
|--------------|------------------------------------|----------------------------------------------|
| depends_on   | source NEEDS target                | "Login Page" depends_on "Auth API"           |
| blocks       | source PREVENTS target             | "DB Migration Bug" blocks "Deploy"           |
| triggers     | source STARTS target               | "Payment Complete" triggers "Send Receipt"   |
| related      | general association (no direction) | "Search UI" related "Filter UI"              |
| parent_child | hierarchy (rarely used — prefer GROUP parentId) |                                   |

---

## Rules

1. **Read first**: Always call thask.graph.get before modifying an existing graph.
2. **Import for bulk**: Use thask.graph.import (merge) when creating 3+ nodes. Individual create only for 1-2 nodes.
3. **Layout after create**: Agents cannot see the canvas. Always call thask.graph.layout (dagre) after creating nodes.
4. **Merge by default**: Never use import mode "replace" unless the user explicitly asks to rebuild from scratch.
5. **No duplicates**: Check existing nodes/edges via graph.get before creating new ones.
6. **Status defaults**: Omitted status = IN_PROGRESS. Always set PASS explicitly for completed items.
7. **Edge direction**: depends_on and blocks are NOT interchangeable. A depends_on B ≠ A blocks B.

---

## Workflows

### Create a dependency map (most common)
` + "`" + `` + "`" + `` + "`" + `
1. thask.graph.get           → read existing graph
2. thask.graph.import(merge) → batch create nodes + edges
3. thask.graph.layout(dagre) → auto-position everything
` + "`" + `` + "`" + `` + "`" + `

Import example:
` + "`" + `` + "`" + `` + "`" + `json
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
` + "`" + `` + "`" + `` + "`" + `
tmp-* IDs are remapped to real UUIDs by the server.

### Update existing nodes
` + "`" + `` + "`" + `` + "`" + `
1. thask.graph.get             → find node IDs
2. thask.node.update           → individual update
   OR thask.node.batch_status  → bulk status change
` + "`" + `` + "`" + `` + "`" + `

### Analyze impact before changes
` + "`" + `` + "`" + `` + "`" + `
1. thask.impact.analyze → downstream cascade for a specific node
2. thask.graph.analyze  → cycle detection + critical path
` + "`" + `` + "`" + `` + "`" + `

---

## Pitfalls

- DO NOT confuse depends_on with blocks — check the direction table above
- DO NOT create nodes one-by-one when you have 3+ — use graph.import
- DO NOT use "replace" mode unless explicitly asked — it deletes ALL existing data
- DO NOT skip graph.layout after bulk creation — nodes stack at (0,0)
- DO NOT create self-loop edges — server rejects them
- DO NOT create duplicate edges between the same node pair
- Deleting a GROUP node orphans its children (parentId → null)
`

// InstructionsText is a compact version for MCP server initialize response.
// Should be short — it loads into every agent session automatically.
const InstructionsText = `Thask graph tool. Key rules: (1) Always graph.get before modifying. (2) Use graph.import(merge) for 3+ nodes, not individual creates. (3) Always graph.layout(dagre) after creation. (4) Edge direction: depends_on = source NEEDS target, blocks = source PREVENTS target. (5) Never use import mode "replace" unless user explicitly asks. Call thask.guide for the full agent guide.`

// RenderInstructions builds the short, init-time instructions string that MCP
// servers return on the `initialize` handshake. It is intentionally compact
// (every word counts — the client may inject this into its system prompt) and
// focuses on the agent's own past mistakes since those are the most actionable
// per-user context. Falls back to InstructionsText alone if no project is set
// or any fetch fails.
func RenderInstructions(c *client.Client, projectID string) string {
	if c == nil || projectID == "" {
		return InstructionsText
	}
	mistakes := fetchGuideNodes(c, projectID, "BUG", "FAIL")
	if len(mistakes) == 0 {
		return InstructionsText
	}
	sortByCreatedDesc(mistakes)
	if len(mistakes) > 5 {
		mistakes = mistakes[:5]
	}

	var sb strings.Builder
	sb.WriteString(InstructionsText)
	sb.WriteString("\n\nThis user's recent mistakes — DO NOT repeat:")
	for _, m := range mistakes {
		sb.WriteString("\n• ")
		sb.WriteString(m.Title)
		if lesson := extractLesson(m.Description); lesson != "" {
			sb.WriteString(" → ")
			sb.WriteString(lesson)
		}
	}
	sb.WriteString("\n\nCall thask.guide with projectId='")
	sb.WriteString(projectID)
	sb.WriteString("' for full state including in-progress work and blockers. Call thask.mistake.record whenever you slip up so the next session sees it.")
	return sb.String()
}

// RenderGuide returns the static GuideText optionally appended with a
// project-specific context block (recent mistakes, in-progress work, blockers).
// Falls back to GuideText alone if any fetch fails or projectID is empty.
func RenderGuide(c *client.Client, projectID string) string {
	if c == nil || projectID == "" {
		return GuideText
	}
	dyn := buildProjectContext(c, projectID)
	if dyn == "" {
		return GuideText
	}
	return GuideText + "\n\n" + dyn
}

type guideNode struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
}

func buildProjectContext(c *client.Client, pid string) string {
	mistakes := fetchGuideNodes(c, pid, "BUG", "FAIL")
	active := fetchGuideNodes(c, pid, "", "IN_PROGRESS")
	failing := fetchGuideNodes(c, pid, "", "FAIL")
	blocked := fetchGuideNodes(c, pid, "", "BLOCKED")

	if len(mistakes) == 0 && len(active) == 0 && len(failing) == 0 && len(blocked) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("---\n\n## 📍 Your Project Context\n\n")
	sb.WriteString("Generated from your Thask project's current state. Read before acting.\n\n")

	if len(mistakes) > 0 {
		sortByCreatedDesc(mistakes)
		if len(mistakes) > 5 {
			mistakes = mistakes[:5]
		}
		sb.WriteString("### Recent mistakes — DO NOT repeat\n\n")
		for _, m := range mistakes {
			sb.WriteString("- **")
			sb.WriteString(m.Title)
			sb.WriteString("**")
			if lesson := extractLesson(m.Description); lesson != "" {
				sb.WriteString(" — ")
				sb.WriteString(lesson)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// GROUP nodes are containers, not actionable work — filter them out.
	actionable := active[:0]
	for _, n := range active {
		if n.Type != "GROUP" {
			actionable = append(actionable, n)
		}
	}
	if len(actionable) > 0 {
		sortByCreatedDesc(actionable)
		if len(actionable) > 10 {
			actionable = actionable[:10]
		}
		sb.WriteString(fmt.Sprintf("### In progress (%d)\n\n", len(actionable)))
		for _, n := range actionable {
			sb.WriteString(fmt.Sprintf("- %s _(%s, %s)_\n", n.Title, n.Type, shortID(n.ID)))
		}
		sb.WriteString("\n")
	}

	// Filter out BUG nodes already shown under "Recent mistakes".
	nonBugFailing := failing[:0]
	for _, n := range failing {
		if n.Type != "BUG" {
			nonBugFailing = append(nonBugFailing, n)
		}
	}
	if len(nonBugFailing) > 0 {
		sortByCreatedDesc(nonBugFailing)
		if len(nonBugFailing) > 5 {
			nonBugFailing = nonBugFailing[:5]
		}
		sb.WriteString("### Currently failing\n\n")
		for _, n := range nonBugFailing {
			sb.WriteString(fmt.Sprintf("- %s _(%s)_\n", n.Title, n.Type))
		}
		sb.WriteString("\n")
	}

	if len(blocked) > 0 {
		sortByCreatedDesc(blocked)
		if len(blocked) > 5 {
			blocked = blocked[:5]
		}
		sb.WriteString("### Blocked\n\n")
		for _, n := range blocked {
			sb.WriteString(fmt.Sprintf("- %s _(%s)_\n", n.Title, n.Type))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Call `thask.mistake.record` whenever the user corrects you, you misuse a tool, or you repeat any item above.\n")
	return sb.String()
}

func fetchGuideNodes(c *client.Client, pid, typ, status string) []guideNode {
	path := "/api/projects/" + pid + "/nodes"
	var qs []string
	if typ != "" {
		qs = append(qs, "type="+typ)
	}
	if status != "" {
		qs = append(qs, "status="+status)
	}
	if len(qs) > 0 {
		path += "?" + strings.Join(qs, "&")
	}
	raw, err := c.Get(path)
	if err != nil {
		return nil
	}
	var nodes []guideNode
	if err := json.Unmarshal(raw, &nodes); err != nil {
		return nil
	}
	return nodes
}

func sortByCreatedDesc(nodes []guideNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CreatedAt > nodes[j].CreatedAt
	})
}

// extractLesson pulls the "**교훈:**" or "**Lesson:**" line from a mistake
// description. Returns the trimmed value, or empty if neither marker is found.
func extractLesson(desc string) string {
	for _, marker := range []string{"**교훈:**", "**Lesson:**"} {
		idx := strings.Index(desc, marker)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(desc[idx+len(marker):])
		if nl := strings.Index(rest, "\n\n"); nl >= 0 {
			rest = rest[:nl]
		}
		rest = strings.ReplaceAll(rest, "\n", " ")
		if len(rest) > 200 {
			rest = rest[:197] + "..."
		}
		return strings.TrimSpace(rest)
	}
	return ""
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}
