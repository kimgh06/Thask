package handler

import (
	"context"
	"fmt"
	cryptoRand "crypto/rand"

	"github.com/jackc/pgx/v5"
	"github.com/thask/backend/internal/audit"
	"github.com/thask/backend/internal/dto"
	"github.com/thask/backend/internal/model"
)

func strPtr(s string) *string { return &s }

func getOldValue(n *model.Node, field string) *string {
	var v string
	switch field {
	case "type":
		v = string(n.Type)
	case "title":
		v = n.Title
	case "description":
		if n.Description != nil {
			v = *n.Description
		}
	case "status":
		v = string(n.Status)
	case "assignee_id":
		if n.AssigneeID != nil {
			v = *n.AssigneeID
		}
	case "parent_id":
		if n.ParentID != nil {
			v = *n.ParentID
		}
	default:
		return nil
	}
	return &v
}

// mutationKindForNodeField maps a node field to the permission class that
// gates writes to it. Keep in sync with audit.MutationKind constants.
//
//	semantic    — claims about reality (description, "why" content)
//	structural  — graph topology / typing (type, parent_id)
//	meta        — operational state (status, position, tags, assignee)
func mutationKindForNodeField(field string) string {
	switch field {
	case "description":
		return audit.MutationSemantic
	case "type", "parent_id":
		return audit.MutationStructural
	default:
		return audit.MutationMeta
	}
}

// newBatchID returns a UUID-format string suitable for audit_log.batch_id. We
// avoid adding a new dependency by composing 16 random bytes into the standard
// 8-4-4-4-12 format. Collision risk is negligible for batch grouping.
func newBatchID() string {
	var b [16]byte
	if _, err := cryptoRand.Read(b[:]); err != nil {
		// Extremely unlikely; the worst-case is duplicate batch IDs, not an
		// auth/security issue, so we silently fall back to a static value.
		return "00000000-0000-0000-0000-000000000000"
	}
	// RFC 4122 variant + version 4
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// nonNilFields extracts the populated fields from a BatchNodeUpdateItem into
// the column-name → value map the UPDATE builder expects. Mirrors the
// per-field handling in NodeHandler.Update.
func nonNilFields(u dto.BatchNodeUpdateItem) map[string]any {
	f := map[string]any{}
	if u.Type != nil {
		f["type"] = *u.Type
	}
	if u.Title != nil {
		f["title"] = *u.Title
	}
	if u.Description != nil {
		f["description"] = *u.Description
	}
	if u.Status != nil {
		f["status"] = *u.Status
	}
	if u.Tags != nil {
		f["tags"] = u.Tags
	}
	if u.AssigneeID != nil {
		if *u.AssigneeID == "" {
			f["assignee_id"] = nil
		} else {
			f["assignee_id"] = *u.AssigneeID
		}
	}
	if u.ParentID != nil {
		if *u.ParentID == "" {
			f["parent_id"] = nil
		} else {
			f["parent_id"] = *u.ParentID
		}
	}
	return f
}

// buildSetClauses returns the SET fragment + ordered args for a dynamic UPDATE.
// Always appends `updated_at = now()` last in the clause list.
func buildSetClauses(fields map[string]any) ([]string, []any) {
	clauses := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields))
	i := 1
	for col, val := range fields {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}
	clauses = append(clauses, "updated_at = now()")
	return clauses, args
}

func joinClauses(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

// detectProjectCycleTx walks every node's parent chain inside the current
// transaction state. Returns the first node id that participates in a cycle,
// or "" when the parent graph is a forest. Real cost is O(N) via 3-color DFS:
// once a node is BLACK we know its entire chain is acyclic and skip re-walking.
// Naive per-node walk would be O(N²) on long chains.
const (
	cycleWhite = 0 // unseen
	cycleGray  = 1 // in current walk
	cycleBlack = 2 // walk completed, no cycle
)

func detectProjectCycleTx(ctx context.Context, tx pgx.Tx, projectID string) (string, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, parent_id FROM nodes WHERE project_id = $1 AND parent_id IS NOT NULL`,
		projectID,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	parent := map[string]string{}
	for rows.Next() {
		var id, pid string
		if err := rows.Scan(&id, &pid); err != nil {
			return "", err
		}
		parent[id] = pid
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	color := make(map[string]int, len(parent))
	for start := range parent {
		if color[start] == cycleBlack {
			continue
		}
		path := []string{}
		cur := start
		for color[cur] == cycleWhite {
			color[cur] = cycleGray
			path = append(path, cur)
			next, ok := parent[cur]
			if !ok {
				break // reached a root
			}
			cur = next
		}
		if color[cur] == cycleGray {
			return cur, nil
		}
		for _, n := range path {
			color[n] = cycleBlack
		}
	}
	return "", nil
}
