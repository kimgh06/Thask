package mcp

func AllTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "thask.node.list",
			Description: "List nodes in a Thask project graph. Filter by type (TASK, BUG, FLOW, BRANCH, API, UI, GROUP) or status (PASS, FAIL, IN_PROGRESS, BLOCKED).",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"type":      propEnum("string", "Node type filter", []string{"TASK", "BUG", "FLOW", "BRANCH", "API", "UI", "GROUP"}),
				"status":    propEnum("string", "Status filter", []string{"PASS", "FAIL", "IN_PROGRESS", "BLOCKED"}),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.node.create",
			Description: "Create a new node in the project graph. Types: FLOW, BRANCH, TASK, BUG, API, UI, GROUP. Status defaults to IN_PROGRESS. Prefer thask.graph.import(merge) when creating 3+ nodes. Always call thask.graph.layout after creation.",
			InputSchema: objectSchema(map[string]any{
				"projectId":   prop("string", "Project UUID"),
				"type":        propEnum("string", "Node type", []string{"FLOW", "BRANCH", "TASK", "BUG", "API", "UI", "GROUP"}),
				"title":       prop("string", "Node title"),
				"description": prop("string", "Optional description (supports markdown)"),
				"status":      propEnum("string", "Optional status", []string{"PASS", "FAIL", "IN_PROGRESS", "BLOCKED"}),
				"tags":        propArray("string", "Optional tags"),
				"positionX":   prop("number", "X position (default 0)"),
				"positionY":   prop("number", "Y position (default 0)"),
			}, []string{"projectId", "type", "title"}),
		},
		{
			Name:        "thask.node.get",
			Description: "Get node details including connected edges and history.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"nodeId":    prop("string", "Node UUID"),
			}, []string{"projectId", "nodeId"}),
		},
		{
			Name:        "thask.node.update",
			Description: "Update node fields (title, status, type, description, tags).",
			InputSchema: objectSchema(map[string]any{
				"projectId":   prop("string", "Project UUID"),
				"nodeId":      prop("string", "Node UUID"),
				"title":       prop("string", "New title"),
				"status":      propEnum("string", "New status", []string{"PASS", "FAIL", "IN_PROGRESS", "BLOCKED"}),
				"type":        propEnum("string", "New type", []string{"FLOW", "BRANCH", "TASK", "BUG", "API", "UI", "GROUP"}),
				"description": prop("string", "New description (supports markdown)"),
				"tags":        propArray("string", "New tags"),
			}, []string{"projectId", "nodeId"}),
		},
		{
			Name:        "thask.node.delete",
			Description: "Delete a node from the project graph.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"nodeId":    prop("string", "Node UUID"),
			}, []string{"projectId", "nodeId"}),
		},
		{
			Name:        "thask.node.batch_status",
			Description: "Batch update status for multiple nodes at once.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"ids":       propArray("string", "Node UUIDs to update"),
				"status":    propEnum("string", "New status", []string{"PASS", "FAIL", "IN_PROGRESS", "BLOCKED"}),
			}, []string{"projectId", "ids", "status"}),
		},
		{
			Name:        "thask.edge.list",
			Description: "List all edges (relationships) in a project.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.edge.create",
			Description: "Create a relationship between two nodes. Types: depends_on (source NEEDS target), blocks (source PREVENTS target), triggers (source STARTS target), related, parent_child. Direction matters — check thask.guide if unsure.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"sourceId":  prop("string", "Source node UUID"),
				"targetId":  prop("string", "Target node UUID"),
				"edgeType":  propEnum("string", "Relationship type", []string{"depends_on", "blocks", "related", "parent_child", "triggers"}),
				"label":     prop("string", "Optional label"),
			}, []string{"projectId", "sourceId", "targetId"}),
		},
		{
			Name:        "thask.edge.delete",
			Description: "Delete an edge from the project graph.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"edgeId":    prop("string", "Edge UUID"),
			}, []string{"projectId", "edgeId"}),
		},
		{
			Name:        "thask.graph.get",
			Description: "Get full graph snapshot (all nodes + edges) for analysis.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.graph.import",
			Description: "Import a graph from JSON data. Mode 'replace' overwrites ALL existing data — use 'merge' by default. Always call thask.graph.get first to check existing state. Call thask.graph.layout(dagre) after import.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"mode":      propEnum("string", "Import mode", []string{"replace", "merge"}),
				"nodes":     propArray("object", "Array of node objects"),
				"edges":     propArray("object", "Array of edge objects"),
			}, []string{"projectId", "mode", "nodes", "edges"}),
		},
		{
			Name:        "thask.impact.analyze",
			Description: "Analyze cascade impact of changing a node's status. Shows which downstream nodes would be affected.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"nodeId":    prop("string", "Node UUID to analyze"),
			}, []string{"projectId", "nodeId"}),
		},
		{
			Name:        "thask.graph.layout",
			Description: "Auto-layout the project graph. Repositions all nodes and auto-sizes GROUP nodes based on their children.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"algorithm": propEnum("string", "Layout algorithm", []string{"dagre", "grid"}),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.scan.run",
			Description: "Scan a Go project's internal dependencies and import them as graph nodes/edges",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Target project ID"),
				"path":      prop("string", "Path to Go project root"),
				"maxFiles":  prop("number", "Max files to scan (default 500)"),
			}, []string{"projectId", "path"}),
		},
		{
			Name:        "thask.graph.analyze",
			Description: "Detect dependency cycles and find the critical path (longest dependency chain) in a project graph",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project ID to analyze"),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.guide",
			Description: "Get the full AI agent guide for Thask. Call this before your first interaction with a Thask project to learn conventions, rules, and workflows.",
			InputSchema: objectSchema(map[string]any{}, []string{}),
		},
	}
}

// Schema helpers
func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func prop(t, desc string) map[string]any {
	return map[string]any{"type": t, "description": desc}
}

func propEnum(t, desc string, values []string) map[string]any {
	return map[string]any{"type": t, "description": desc, "enum": values}
}

func propArray(itemType, desc string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": itemType}}
}
