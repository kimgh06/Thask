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
				var candidate string
				switch edge.EdgeType {
				case model.EdgeTypeBlocks, model.EdgeTypeTriggers:
					// forward: source changed → target affected
					if edge.SourceID == fid {
						candidate = edge.TargetID
					}
				case model.EdgeTypeDependsOn:
					// backward: A depends_on B, B changed → A affected
					if edge.TargetID == fid {
						candidate = edge.SourceID
					}
				case model.EdgeTypeRelated:
					// bidirectional
					if edge.SourceID == fid {
						candidate = edge.TargetID
					} else if edge.TargetID == fid {
						candidate = edge.SourceID
					}
				case model.EdgeTypeParentChild:
					continue // structural, not causal
				}
				if candidate != "" && !impacted[candidate] {
					impacted[candidate] = true
					next = append(next, candidate)
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
