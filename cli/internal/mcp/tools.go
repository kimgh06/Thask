package mcp

func AllTools() []ToolDef {
	return []ToolDef{
		{
			Name:        "thask.node.list",
			Description: "List nodes in a Thask project graph. Filter by type (TASK, BUG, FLOW, BRANCH, API, UI, GROUP) or status (PASS, FAIL, IN_PROGRESS, BLOCKED).",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"type":      propEnum("string", "Node type filter", []string{"TASK", "BUG", "FLOW", "BRANCH", "API", "UI", "GROUP", "REQUIREMENT", "DECISION", "EXPERIMENT", "PERSON"}),
				"status":    propEnum("string", "Status filter", []string{"PASS", "FAIL", "IN_PROGRESS", "BLOCKED"}),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.node.create",
			Description: "Create a new node in the project graph. Types: FLOW, BRANCH, TASK, BUG, API, UI, GROUP. Status defaults to IN_PROGRESS. Prefer thask.graph.import(merge) when creating 3+ nodes. Always call thask.graph.layout after creation.",
			InputSchema: objectSchema(map[string]any{
				"projectId":   prop("string", "Project UUID"),
				"type":        propEnum("string", "Node type", []string{"FLOW", "BRANCH", "TASK", "BUG", "API", "UI", "GROUP", "REQUIREMENT", "DECISION", "EXPERIMENT", "PERSON"}),
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
			Description: "Update node fields. Pass only the fields you want to change — all are optional except projectId+nodeId. Empty string on parentId/assigneeId unparents / unassigns. For >1 node at once, prefer thask.node.batch_update (saves agent context, atomic on cycle/permission).",
			InputSchema: objectSchema(map[string]any{
				"projectId":   prop("string", "Project UUID"),
				"nodeId":      prop("string", "Node UUID"),
				"title":       prop("string", "New title"),
				"status":      propEnum("string", "New status", []string{"PASS", "FAIL", "IN_PROGRESS", "BLOCKED"}),
				"type":        propEnum("string", "New type", []string{"FLOW", "BRANCH", "TASK", "BUG", "API", "UI", "GROUP", "REQUIREMENT", "DECISION", "EXPERIMENT", "PERSON"}),
				"description": prop("string", "New description (supports markdown). Blocked by default on agent keys — use thask.node.suggest_update."),
				"tags":        propArray("string", "New tags"),
				"parentId":       prop("string", "Re-parent the node. UUID = set parent, empty string = unparent. Server rejects self-parent / cycles."),
				"assigneeId":     prop("string", "Assignee user UUID, or empty string to unassign."),
				"lifecycleState": prop("string", "Domain lifecycle phase for REQUIREMENT/DECISION/EXPERIMENT/PERSON (e.g. APPROVED, RUNNING). Free-form text; server stamps lifecycle_state_changed_at automatically."),
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
				"edgeType":  propEnum("string", "Relationship type", []string{"depends_on", "blocks", "related", "parent_child", "triggers", "realizes", "conflicts", "drives", "supersedes", "tests", "produced", "owns", "decided", "reported"}),
				"label":     prop("string", "Optional label"),
			}, []string{"projectId", "sourceId", "targetId"}),
		},
		{
			Name:        "thask.edge.update",
			Description: "Update an existing edge. Only fields supplied are touched; unsupplied fields keep their current value (server COALESCEs). Useful for renaming a relationship type or attaching a label without recreating the edge.",
			InputSchema: objectSchema(map[string]any{
				"projectId":   prop("string", "Project UUID"),
				"edgeId":      prop("string", "Edge UUID"),
				"edgeType":    propEnum("string", "New relationship type", []string{"depends_on", "blocks", "related", "parent_child", "triggers", "realizes", "conflicts", "drives", "supersedes", "tests", "produced", "owns", "decided", "reported"}),
				"metadata":    prop("object", "Free-form JSON metadata attached to the edge (v0.6.0). Used by the Knowledge OS entity types — e.g. supersedes carries `{ reason: '...' }`, produced carries `{ outcome_summary: '...' }`."),
				"label":       prop("string", "New label"),
				"sourcePort":  prop("string", "New source port"),
				"targetPort":  prop("string", "New target port"),
				"waypoints":   propArray("object", "New waypoints array; pass [] to clear"),
			}, []string{"projectId", "edgeId"}),
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
			Description: "Import nodes/edges as NEW entities (every node gets a fresh UUID). 'replace' deletes all existing data first; 'merge' appends to existing. This is for migrations / external→Thask transfers — it does NOT patch existing nodes by id. To partial-update existing nodes use thask.node.update (single) or thask.node.batch_update (many). To add only edges between existing nodes use thask.edge.batch_create. Each item requires type + title. Always call thask.graph.layout(dagre) after import.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"mode":      propEnum("string", "Import mode (creates new nodes either way)", []string{"replace", "merge"}),
				"nodes":     propArray("object", "Array of node objects — each MUST include type and title"),
				"edges":     propArray("object", "Array of edge objects referencing imported node ids"),
			}, []string{"projectId", "mode", "nodes", "edges"}),
		},
		{
			Name:        "thask.node.batch_update",
			Description: "Partial-update up to 200 nodes in one call. Use INSTEAD of looping thask.node.update — saves agent context (1 call vs N) and writes one batch_id grouping the audit_log rows. Atomic on permission / cycle failure, best-effort on per-item not-found and no-change (returned in skipped[]). Server responds 207 Multi-Status when any items skip. Each update sends only the fields to change; empty string on parentId/assigneeId unparents / unassigns.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"updates": propArray("object", "Array of { nodeId, ...fields }. Max 200. Fields: type, title, description, status, tags, parentId, assigneeId."),
			}, []string{"projectId", "updates"}),
		},
		{
			Name:        "thask.edge.batch_create",
			Description: "Create up to 500 edges between existing nodes in one call. Atomic on permission failure, per-item skip for self-references / duplicates / invalid endpoints (returned in skipped[]). Server responds 207 when any items skip.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"edges": propArray("object", "Array of { sourceId, targetId, edgeType?, label?, sourcePort?, targetPort?, waypoints? }. Max 500."),
			}, []string{"projectId", "edges"}),
		},
		{
			Name:        "thask.edge.batch_delete",
			Description: "Delete up to 500 edges by id in one call. Missing ids are returned in skipped[] (not_found). Server responds 207 when any items skip.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"edgeIds":   propArray("string", "Array of edge UUIDs. Max 500."),
			}, []string{"projectId", "edgeIds"}),
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
			Description: "Scan a project's internal dependencies (Go and TypeScript/JavaScript supported) and import them as graph nodes/edges",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Target project ID"),
				"path":      prop("string", "Path to project root (go.mod for Go, package.json for TS/JS)"),
				"maxFiles":  prop("number", "Max files to scan (default 500)"),
				"language":  propEnum("string", "Force scanner language (default auto)", []string{"auto", "go", "ts"}),
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
			Name:        "thask.node.suggest_update",
			Description: "Propose a description change for human review instead of writing directly. Use this when your API key lacks write_semantic permission (agent keys block direct description writes by default — see thask.guide). Always quote the code/file that motivated the change in 'rationale' so the reviewer can verify.",
			InputSchema: objectSchema(map[string]any{
				"projectId":     prop("string", "Project UUID"),
				"nodeId":        prop("string", "Node UUID to propose changes for"),
				"fieldName":     propEnum("string", "Which field to update (default 'description')", []string{"description", "title", "tags"}),
				"proposedValue": prop("string", "The new value you want the human to consider"),
				"rationale":     prop("string", "Why the change is needed. MUST quote the code/file/commit that triggered it — reviewers reject suggestions without provenance."),
				"evidence":      prop("object", "Optional structured evidence: { codeCommit, sourcePaths[], confidence }"),
			}, []string{"projectId", "nodeId", "proposedValue"}),
		},
		{
			Name:        "thask.suggestions.list",
			Description: "List pending agent-proposed updates awaiting human review for a project.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"limit":     prop("integer", "Max rows (default 50, max 200)"),
			}, []string{"projectId"}),
		},
		{
			Name:        "thask.suggestions.decide",
			Description: "Accept or reject a pending suggestion. Accepting copies the proposed value into the target node and credits the deciding human as the author. Reserved for user_interactive keys.",
			InputSchema: objectSchema(map[string]any{
				"projectId":    prop("string", "Project UUID"),
				"suggestionId": prop("string", "Suggestion UUID returned by suggestions.list"),
				"status":       propEnum("string", "Decision", []string{"accepted", "rejected"}),
				"reason":       prop("string", "Optional rationale for the decision"),
			}, []string{"projectId", "suggestionId", "status"}),
		},
		{
			Name:        "thask.node.verify",
			Description: "Stamp 'still correct as of now' on a node — sets last_verified_at / by / commit. Requires permissions.verify. Use sparingly; verifying agent-authored content without re-reading the code defeats the safety guarantee.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"nodeId":    prop("string", "Node UUID to mark verified"),
				"commit":    prop("string", "Optional git commit SHA you verified against"),
			}, []string{"projectId", "nodeId"}),
		},
		{
			Name:        "thask.mistake.record",
			Description: "Record an agent mistake as a BUG node under the project's '실수 기록' GROUP (auto-created if missing). Use whenever the user corrects you, you get a tool/command wrong, or you repeat a prior error. The recorded mistake will be surfaced by thask.guide in future sessions to prevent recurrence.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Project UUID"),
				"title":     prop("string", "Short label for the mistake (e.g. '/plugin install ?path= 구문 발명')"),
				"cause":     prop("string", "Why it happened — the wrong assumption or missing check"),
				"fix":       prop("string", "How it was corrected this time"),
				"lesson":    prop("string", "Rule for future sessions to avoid repeating it"),
			}, []string{"projectId", "title", "lesson"}),
		},
		{
			Name:        "thask.guide",
			Description: "Get the full AI agent guide for Thask. Pass projectId to also receive recent mistakes, in-progress work, and blockers from that project — call this at the start of every session to load user-specific context.",
			InputSchema: objectSchema(map[string]any{
				"projectId": prop("string", "Optional project UUID. When provided, the response includes a 'Your Project Context' section with recent mistakes, in-progress nodes, and blockers."),
			}, []string{}),
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
