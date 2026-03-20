package service

import (
	"math"

	"github.com/thask/backend/internal/model"
)

const (
	nodeW      = 72.0
	nodeH      = 72.0
	cellPad    = 40.0
	groupPadX  = 30.0
	groupPadTop = 45.0
	groupPadBot = 30.0
	layerSpaceX = 250.0
	layerSpaceY = 180.0
	minGroupW   = 160.0
	minGroupH   = 100.0
)

type LayoutPosition struct {
	ID     string
	X, Y   float64
	Width  *float64
	Height *float64
}

func CalculateLayout(nodes []model.Node, edges []model.Edge, algorithm string) []LayoutPosition {
	if algorithm == "grid" {
		return gridLayout(nodes)
	}
	return dagreLayout(nodes, edges)
}

// dagreLayout implements a hierarchical layout with GROUP auto-sizing
func dagreLayout(nodes []model.Node, edges []model.Edge) []LayoutPosition {
	nodeMap := make(map[string]*model.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	// Classify: top-level vs grouped
	var topLevel []string
	children := make(map[string][]string) // groupID -> childIDs
	for _, n := range nodes {
		if n.ParentID == nil {
			topLevel = append(topLevel, n.ID)
		} else {
			children[*n.ParentID] = append(children[*n.ParentID], n.ID)
		}
	}

	// Step 1: Auto-size GROUPs and position children relative to center (0,0)
	groupSizes := make(map[string][2]float64) // id -> {w, h}
	childRelPos := make(map[string][2]float64) // childID -> {relX, relY}

	for _, n := range nodes {
		if n.Type != model.NodeTypeGroup {
			continue
		}
		kids := children[n.ID]
		w, h := autoSizeGroup(kids)
		groupSizes[n.ID] = [2]float64{w, h}
		positionChildrenRelative(kids, w, h, childRelPos)
	}

	// Step 2: Build adjacency for top-level nodes (directional edges only)
	inDegree := make(map[string]int)
	outEdges := make(map[string][]string) // source -> targets
	topSet := make(map[string]bool)
	for _, id := range topLevel {
		topSet[id] = true
		inDegree[id] = 0
	}

	for _, e := range edges {
		src, tgt := e.SourceID, e.TargetID
		// Resolve to top-level parent if node is grouped
		src = resolveTopLevel(src, nodeMap)
		tgt = resolveTopLevel(tgt, nodeMap)
		if src == tgt || !topSet[src] || !topSet[tgt] {
			continue
		}
		// Only use directional edges for layering
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		outEdges[src] = append(outEdges[src], tgt)
		inDegree[tgt]++
	}

	// Step 3: Topological sort (Kahn's algorithm) for layer assignment
	layers := make(map[string]int)
	var queue []string
	for _, id := range topLevel {
		if inDegree[id] == 0 {
			queue = append(queue, id)
			layers[id] = 0
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, tgt := range outEdges[cur] {
			if layers[cur]+1 > layers[tgt] {
				layers[tgt] = layers[cur] + 1
			}
			inDegree[tgt]--
			if inDegree[tgt] == 0 {
				queue = append(queue, tgt)
			}
		}
	}

	// Assign unvisited nodes (cycles) to layer 0
	for _, id := range topLevel {
		if _, ok := layers[id]; !ok {
			layers[id] = 0
		}
	}

	// Step 4: Group by layer
	layerNodes := make(map[int][]string)
	for _, id := range topLevel {
		l := layers[id]
		layerNodes[l] = append(layerNodes[l], id)
	}

	// Step 5: Position top-level nodes
	positions := make(map[string][2]float64)
	for layer, ids := range layerNodes {
		count := len(ids)
		for i, id := range ids {
			x := float64(layer) * layerSpaceX
			y := (float64(i) - float64(count-1)/2.0) * layerSpaceY
			positions[id] = [2]float64{x, y}
		}
	}

	// Step 6: Build result
	result := make([]LayoutPosition, 0, len(nodes))
	for _, n := range nodes {
		lp := LayoutPosition{ID: n.ID}

		if n.ParentID != nil {
			// Child node: parent position + relative offset
			parentPos := positions[*n.ParentID]
			rel := childRelPos[n.ID]
			lp.X = parentPos[0] + rel[0]
			lp.Y = parentPos[1] + rel[1]
		} else {
			pos := positions[n.ID]
			lp.X = pos[0]
			lp.Y = pos[1]
		}

		if n.Type == model.NodeTypeGroup {
			if sz, ok := groupSizes[n.ID]; ok {
				w, h := sz[0], sz[1]
				lp.Width = &w
				lp.Height = &h
			}
		}

		result = append(result, lp)
	}

	return result
}

// gridLayout arranges all nodes in a simple grid
func gridLayout(nodes []model.Node) []LayoutPosition {
	cols := int(math.Ceil(math.Sqrt(float64(len(nodes)))))
	result := make([]LayoutPosition, len(nodes))

	for i, n := range nodes {
		col := i % cols
		row := i / cols
		result[i] = LayoutPosition{
			ID: n.ID,
			X:  float64(col) * layerSpaceX,
			Y:  float64(row) * layerSpaceY,
		}
		if n.Type == model.NodeTypeGroup {
			w, h := autoSizeGroup(nil) // default size
			result[i].Width = &w
			result[i].Height = &h
		}
	}

	return result
}

func autoSizeGroup(childIDs []string) (float64, float64) {
	count := len(childIDs)
	if count == 0 {
		return minGroupW, minGroupH
	}

	cols := int(math.Ceil(math.Sqrt(float64(count))))
	rows := int(math.Ceil(float64(count) / float64(cols)))

	cellW := nodeW + cellPad
	cellH := nodeH + cellPad

	gridW := float64(cols) * cellW
	gridH := float64(rows) * cellH

	w := math.Max(gridW+groupPadX*2, minGroupW)
	h := math.Max(gridH+groupPadTop+groupPadBot, minGroupH)

	return w, h
}

func positionChildrenRelative(childIDs []string, groupW, groupH float64, relPos map[string][2]float64) {
	count := len(childIDs)
	if count == 0 {
		return
	}

	cols := int(math.Ceil(math.Sqrt(float64(count))))
	cellW := nodeW + cellPad
	cellH := nodeH + cellPad

	startX := -float64(cols-1)*cellW/2.0
	rows := int(math.Ceil(float64(count) / float64(cols)))
	startY := -float64(rows-1)*cellH/2.0 + (groupPadTop-groupPadBot)/2.0

	for i, id := range childIDs {
		col := i % cols
		row := i / cols
		relPos[id] = [2]float64{
			startX + float64(col)*cellW,
			startY + float64(row)*cellH,
		}
	}
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
