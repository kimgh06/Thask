package service

import "github.com/thask/backend/internal/model"

const maxWaterfallDepth = 10

type StatusChange struct {
	NodeID    string           `json:"nodeId"`
	OldStatus model.NodeStatus `json:"oldStatus"`
	NewStatus model.NodeStatus `json:"newStatus"`
}

type WaterfallNode struct {
	ID       string
	Status   model.NodeStatus
	ParentID *string
}

type WaterfallEdge struct {
	SourceID string
	TargetID string
	EdgeType model.EdgeType
}

// ComputeWaterfall calculates cascading status changes triggered by a node's status update.
// Pure function — does not touch the DB.
func ComputeWaterfall(changedNodeID string, newStatus model.NodeStatus, allNodes []WaterfallNode, allEdges []WaterfallEdge) []StatusChange {
	if newStatus != model.NodeStatusPass && newStatus != model.NodeStatusFail {
		return nil
	}

	nodeMap := make(map[string]*WaterfallNode, len(allNodes))
	for i := range allNodes {
		n := allNodes[i]
		nodeMap[n.ID] = &WaterfallNode{ID: n.ID, Status: n.Status, ParentID: n.ParentID}
	}

	// Apply the triggering change
	if trigger, ok := nodeMap[changedNodeID]; ok {
		trigger.Status = newStatus
	}

	var changes []StatusChange
	visited := map[string]bool{changedNodeID: true}

	type queueItem struct {
		id    string
		depth int
	}
	queue := []queueItem{{changedNodeID, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxWaterfallDepth {
			continue
		}

		node, ok := nodeMap[current.id]
		if !ok {
			continue
		}

		derived := deriveChanges(current.id, node.Status, nodeMap, allEdges)
		for _, change := range derived {
			if visited[change.NodeID] {
				continue
			}
			visited[change.NodeID] = true

			target, ok := nodeMap[change.NodeID]
			if !ok || target.Status == change.NewStatus {
				continue
			}

			change.OldStatus = target.Status
			target.Status = change.NewStatus
			changes = append(changes, change)
			queue = append(queue, queueItem{change.NodeID, current.depth + 1})
		}

		// Parent-child aggregation
		if node.ParentID != nil && !visited[*node.ParentID] {
			if parentChange := evaluateParent(*node.ParentID, nodeMap, allNodes); parentChange != nil {
				visited[parentChange.NodeID] = true
				if parent, ok := nodeMap[parentChange.NodeID]; ok {
					parent.Status = parentChange.NewStatus
					changes = append(changes, *parentChange)
					queue = append(queue, queueItem{parentChange.NodeID, current.depth + 1})
				}
			}
		}
	}

	return changes
}

func deriveChanges(nodeID string, status model.NodeStatus, nodeMap map[string]*WaterfallNode, allEdges []WaterfallEdge) []StatusChange {
	var results []StatusChange

	for _, edge := range allEdges {
		if edge.EdgeType == model.EdgeTypeRelated || edge.EdgeType == model.EdgeTypeParentChild {
			continue
		}

		// blocks: A --blocks--> B
		if edge.EdgeType == model.EdgeTypeBlocks && edge.SourceID == nodeID {
			target, ok := nodeMap[edge.TargetID]
			if !ok {
				continue
			}
			if status == model.NodeStatusPass && target.Status == model.NodeStatusBlocked {
				allResolved := true
				for _, e := range allEdges {
					if e.EdgeType == model.EdgeTypeBlocks && e.TargetID == edge.TargetID && e.SourceID != nodeID {
						if src, ok := nodeMap[e.SourceID]; ok && src.Status != model.NodeStatusPass {
							allResolved = false
							break
						}
					}
				}
				if allResolved {
					results = append(results, StatusChange{NodeID: edge.TargetID, OldStatus: target.Status, NewStatus: model.NodeStatusInProgress})
				}
			} else if status == model.NodeStatusFail && target.Status == model.NodeStatusInProgress {
				results = append(results, StatusChange{NodeID: edge.TargetID, OldStatus: target.Status, NewStatus: model.NodeStatusBlocked})
			}
		}

		// depends_on: A --depends_on--> B (A depends on B)
		if edge.EdgeType == model.EdgeTypeDependsOn && edge.TargetID == nodeID {
			dependent, ok := nodeMap[edge.SourceID]
			if !ok {
				continue
			}
			if status == model.NodeStatusPass && dependent.Status == model.NodeStatusBlocked {
				allMet := true
				for _, e := range allEdges {
					if e.EdgeType == model.EdgeTypeDependsOn && e.SourceID == edge.SourceID {
						if dep, ok := nodeMap[e.TargetID]; ok && dep.Status != model.NodeStatusPass {
							allMet = false
							break
						}
					}
				}
				if allMet {
					results = append(results, StatusChange{NodeID: edge.SourceID, OldStatus: dependent.Status, NewStatus: model.NodeStatusInProgress})
				}
			} else if status == model.NodeStatusFail && dependent.Status == model.NodeStatusInProgress {
				results = append(results, StatusChange{NodeID: edge.SourceID, OldStatus: dependent.Status, NewStatus: model.NodeStatusBlocked})
			}
		}

		// triggers: A --triggers--> B
		if edge.EdgeType == model.EdgeTypeTriggers && edge.SourceID == nodeID {
			target, ok := nodeMap[edge.TargetID]
			if !ok {
				continue
			}
			if status == model.NodeStatusPass && target.Status == model.NodeStatusBlocked {
				results = append(results, StatusChange{NodeID: edge.TargetID, OldStatus: target.Status, NewStatus: model.NodeStatusInProgress})
			}
		}
	}

	return results
}

func evaluateParent(parentID string, nodeMap map[string]*WaterfallNode, allNodes []WaterfallNode) *StatusChange {
	parent, ok := nodeMap[parentID]
	if !ok {
		return nil
	}

	var children []WaterfallNode
	for _, n := range allNodes {
		if n.ParentID != nil && *n.ParentID == parentID {
			children = append(children, n)
		}
	}
	if len(children) == 0 {
		return nil
	}

	allPass := true
	anyFail := false
	for _, c := range children {
		s := c.Status
		if node, ok := nodeMap[c.ID]; ok {
			s = node.Status
		}
		if s != model.NodeStatusPass {
			allPass = false
		}
		if s == model.NodeStatusFail {
			anyFail = true
		}
	}

	var newStatus model.NodeStatus
	if allPass {
		newStatus = model.NodeStatusPass
	} else if anyFail {
		newStatus = model.NodeStatusFail
	} else {
		newStatus = model.NodeStatusInProgress
	}

	if newStatus == parent.Status {
		return nil
	}

	return &StatusChange{NodeID: parentID, OldStatus: parent.Status, NewStatus: newStatus}
}
