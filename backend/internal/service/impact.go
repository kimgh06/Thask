package service

import "github.com/thask/backend/internal/model"

// ComputeImpact performs BFS from changed nodes to find impacted nodes up to given depth.
func ComputeImpact(changedIDs []string, allEdges []model.Edge, depth int) []string {
	impacted := make(map[string]bool)
	for _, id := range changedIDs {
		impacted[id] = true
	}

	frontier := make([]string, len(changedIDs))
	copy(frontier, changedIDs)

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []string
		for _, edge := range allEdges {
			for _, fid := range frontier {
				if edge.SourceID == fid && !impacted[edge.TargetID] {
					impacted[edge.TargetID] = true
					next = append(next, edge.TargetID)
				}
				if edge.TargetID == fid && !impacted[edge.SourceID] {
					impacted[edge.SourceID] = true
					next = append(next, edge.SourceID)
				}
			}
		}
		frontier = next
	}

	// Return only non-changed IDs
	changedSet := make(map[string]bool)
	for _, id := range changedIDs {
		changedSet[id] = true
	}
	var result []string
	for id := range impacted {
		if !changedSet[id] {
			result = append(result, id)
		}
	}
	return result
}
