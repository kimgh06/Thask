package service

import (
	"math"

	"github.com/thask/backend/internal/model"
)

type childExternalPull struct {
	avgX  float64
	avgY  float64
	count int
}

type childBoundaryDemand struct {
	leftOrder  float64
	rightOrder float64
	topOrder   float64
	botOrder   float64
	leftCount  int
	rightCount int
	topCount   int
	botCount   int
}

type childExternalLink struct {
	routeID string
	childID string
	dx      float64
	dy      float64
	inbound bool
}

func childExternalPulls(
	groupID string,
	groupPos [2]float64,
	kids []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	positions map[string][2]float64,
) map[string]childExternalPull {
	childSet := make(map[string]bool, len(kids))
	pulls := make(map[string]childExternalPull, len(kids))
	for _, kid := range kids {
		childSet[kid] = true
		pulls[kid] = childExternalPull{}
	}

	for _, e := range edges {
		var kid, external string
		if childSet[e.SourceID] && !childSet[e.TargetID] {
			kid = e.SourceID
			external = resolveTopLevel(e.TargetID, nodeMap)
		} else if childSet[e.TargetID] && !childSet[e.SourceID] {
			kid = e.TargetID
			external = resolveTopLevel(e.SourceID, nodeMap)
		}
		if kid == "" || external == groupID {
			continue
		}

		extPos, ok := positions[external]
		if !ok {
			continue
		}

		pull := pulls[kid]
		pull.avgX += extPos[0] - groupPos[0]
		pull.avgY += extPos[1] - groupPos[1]
		pull.count++
		pulls[kid] = pull
	}

	for kid, pull := range pulls {
		if pull.count == 0 {
			continue
		}
		pull.avgX /= float64(pull.count)
		pull.avgY /= float64(pull.count)
		pulls[kid] = pull
	}

	return pulls
}

func buildChildExternalLinks(
	groupID string,
	groupPos [2]float64,
	kids []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	positions map[string][2]float64,
) []childExternalLink {
	childSet := make(map[string]bool, len(kids))
	for _, kid := range kids {
		childSet[kid] = true
	}

	links := make([]childExternalLink, 0, len(edges))
	for _, e := range edges {
		var kid, external string
		inbound := false
		switch {
		case childSet[e.TargetID] && !childSet[e.SourceID]:
			kid = e.TargetID
			external = resolveTopLevel(e.SourceID, nodeMap)
			inbound = true
		case childSet[e.SourceID] && !childSet[e.TargetID]:
			kid = e.SourceID
			external = resolveTopLevel(e.TargetID, nodeMap)
		default:
			continue
		}
		if kid == "" || external == "" || external == groupID {
			continue
		}

		extPos, ok := positions[external]
		if !ok {
			continue
		}

		links = append(links, childExternalLink{
			routeID: e.ID,
			childID: kid,
			dx:      extPos[0] - groupPos[0],
			dy:      extPos[1] - groupPos[1],
			inbound: inbound,
		})
	}

	return links
}

func buildChildBoundaryDemands(
	groupID string,
	groupPos [2]float64,
	kids []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	positions map[string][2]float64,
) map[string]childBoundaryDemand {
	childSet := make(map[string]bool, len(kids))
	demands := make(map[string]childBoundaryDemand, len(kids))
	for _, kid := range kids {
		childSet[kid] = true
		demands[kid] = childBoundaryDemand{}
	}

	for _, e := range edges {
		var kid, external string
		if childSet[e.SourceID] && !childSet[e.TargetID] {
			kid = e.SourceID
			external = resolveTopLevel(e.TargetID, nodeMap)
		} else if childSet[e.TargetID] && !childSet[e.SourceID] {
			kid = e.TargetID
			external = resolveTopLevel(e.SourceID, nodeMap)
		}
		if kid == "" || external == groupID {
			continue
		}

		extPos, ok := positions[external]
		if !ok {
			continue
		}

		demand := demands[kid]
		dx := extPos[0] - groupPos[0]
		dy := extPos[1] - groupPos[1]
		if math.Abs(dx) >= math.Abs(dy) {
			if dx < 0 {
				demand.leftOrder += dy
				demand.leftCount++
			} else {
				demand.rightOrder += dy
				demand.rightCount++
			}
		} else {
			if dy < 0 {
				demand.topOrder += dx
				demand.topCount++
			} else {
				demand.botOrder += dx
				demand.botCount++
			}
		}
		demands[kid] = demand
	}

	for kid, demand := range demands {
		if demand.leftCount > 0 {
			demand.leftOrder /= float64(demand.leftCount)
		}
		if demand.rightCount > 0 {
			demand.rightOrder /= float64(demand.rightCount)
		}
		if demand.topCount > 0 {
			demand.topOrder /= float64(demand.topCount)
		}
		if demand.botCount > 0 {
			demand.botOrder /= float64(demand.botCount)
		}
		demands[kid] = demand
	}

	return demands
}

func boundaryDesiredY(id string, demands map[string]childBoundaryDemand, pull childExternalPull, fallback float64) float64 {
	if demand, ok := demands[id]; ok {
		total := demand.leftCount + demand.rightCount
		if total > 0 {
			sum := demand.leftOrder*float64(demand.leftCount) + demand.rightOrder*float64(demand.rightCount)
			return sum / float64(total)
		}
	}
	if pull.count > 0 {
		return pull.avgY
	}
	return fallback
}

func boundaryDesiredX(id string, demands map[string]childBoundaryDemand, pull childExternalPull, fallback float64) float64 {
	if demand, ok := demands[id]; ok {
		total := demand.topCount + demand.botCount
		if total > 0 {
			sum := demand.topOrder*float64(demand.topCount) + demand.botOrder*float64(demand.botCount)
			return sum / float64(total)
		}
	}
	if pull.count > 0 {
		return pull.avgX
	}
	return fallback
}

// childConnAvgRelY returns the average relative Y of group children that node `id` connects to.
// Positive = lower children, negative = upper. Returns (avg, count).
func childConnAvgRelY(id string, edges []model.Edge, nodeMap map[string]*model.Node, childRelPos map[string][2]float64) (float64, int) {
	sum := 0.0
	count := 0
	for _, e := range edges {
		var child string
		if e.SourceID == id {
			child = e.TargetID
		} else if e.TargetID == id {
			child = e.SourceID
		} else {
			continue
		}
		// Check if child is inside a group (has a parent)
		if n, ok := nodeMap[child]; ok && n.ParentID != nil {
			if rel, ok := childRelPos[child]; ok {
				sum += rel[1]
				count++
			}
		}
	}
	if count == 0 {
		return 0, 0
	}
	return sum / float64(count), count
}

func resolveTopLevel(nodeID string, nodeMap map[string]*model.Node) string {
	n, ok := nodeMap[nodeID]
	if !ok {
		return nodeID
	}
	for n.ParentID != nil {
		parent, ok := nodeMap[*n.ParentID]
		if !ok {
			break
		}
		n = parent
	}
	return n.ID
}
