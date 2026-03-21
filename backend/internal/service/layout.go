package service

import (
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

const (
	nodeW       = 72.0
	nodeH       = 72.0
	cellPad     = 40.0
	groupPadX   = 30.0
	groupPadTop = 45.0
	groupPadBot = 30.0
	layerSpaceX = 250.0
	layerSpaceY = 180.0
	minGroupW   = 160.0
	minGroupH   = 100.0
	routePad    = 30.0 // padding around obstacles for edge routing
)

type LayoutPosition struct {
	ID     string
	X, Y   float64
	Width  *float64
	Height *float64
}

type Point struct {
	X, Y float64
}

type EdgeRoute struct {
	ID        string
	Waypoints []Point // nil = straight line
}

type LayoutResult struct {
	Positions  []LayoutPosition
	EdgeRoutes []EdgeRoute
}

func CalculateLayout(nodes []model.Node, edges []model.Edge, algorithm string) LayoutResult {
	var positions []LayoutPosition
	if algorithm == "grid" {
		positions = gridLayout(nodes)
	} else {
		positions = dagreLayout(nodes, edges)
	}
	edgeRoutes := routeEdges(positions, edges, nodes)
	return LayoutResult{Positions: positions, EdgeRoutes: edgeRoutes}
}

// --- Edge routing: obstacle avoidance ---

type bbox struct {
	minX, minY, maxX, maxY float64
}

func routeEdges(positions []LayoutPosition, edges []model.Edge, nodes []model.Node) []EdgeRoute {
	// Build set of top-level node IDs (exclude children inside groups)
	topLevel := make(map[string]bool)
	for _, n := range nodes {
		if n.ParentID == nil {
			topLevel[n.ID] = true
		}
	}

	// Build bounding boxes for top-level nodes only
	posMap := make(map[string]Point)
	boxes := make(map[string]bbox)
	for _, lp := range positions {
		posMap[lp.ID] = Point{lp.X, lp.Y}
		if !topLevel[lp.ID] {
			continue // skip child nodes as obstacles
		}
		w := nodeW
		h := nodeH
		if lp.Width != nil {
			w = *lp.Width
		}
		if lp.Height != nil {
			h = *lp.Height
		}
		boxes[lp.ID] = bbox{
			minX: lp.X - w/2,
			minY: lp.Y - h/2,
			maxX: lp.X + w/2,
			maxY: lp.Y + h/2,
		}
	}

	routes := make([]EdgeRoute, 0, len(edges))
	for _, edge := range edges {
		srcPos, srcOk := posMap[edge.SourceID]
		tgtPos, tgtOk := posMap[edge.TargetID]
		if !srcOk || !tgtOk {
			routes = append(routes, EdgeRoute{ID: edge.ID})
			continue
		}

		// Find obstacles: nodes whose bbox intersects the source→target line
		var obstacles []bbox
		for id, box := range boxes {
			if id == edge.SourceID || id == edge.TargetID {
				continue
			}
			// Expand box slightly for padding
			expanded := bbox{
				minX: box.minX - routePad/2,
				minY: box.minY - routePad/2,
				maxX: box.maxX + routePad/2,
				maxY: box.maxY + routePad/2,
			}
			if lineIntersectsBox(srcPos, tgtPos, expanded) {
				obstacles = append(obstacles, box)
			}
		}

		if len(obstacles) == 0 {
			routes = append(routes, EdgeRoute{ID: edge.ID})
			continue
		}

		// Sort obstacles by distance from source
		sort.Slice(obstacles, func(i, j int) bool {
			di := math.Hypot(((obstacles[i].minX+obstacles[i].maxX)/2)-srcPos.X, ((obstacles[i].minY+obstacles[i].maxY)/2)-srcPos.Y)
			dj := math.Hypot(((obstacles[j].minX+obstacles[j].maxX)/2)-srcPos.X, ((obstacles[j].minY+obstacles[j].maxY)/2)-srcPos.Y)
			return di < dj
		})

		waypoints := generateDetour(srcPos, tgtPos, obstacles, boxes)
		routes = append(routes, EdgeRoute{ID: edge.ID, Waypoints: waypoints})
	}
	return routes
}

func generateDetour(src, tgt Point, obstacles []bbox, allBoxes map[string]bbox) []Point {
	var waypoints []Point
	avgY := (src.Y + tgt.Y) / 2

	for _, obs := range obstacles {
		midY := (obs.minY + obs.maxY) / 2

		if avgY < midY {
			waypoints = append(waypoints,
				Point{(obs.minX + obs.maxX) / 2, obs.minY - routePad},
			)
		} else {
			waypoints = append(waypoints,
				Point{(obs.minX + obs.maxX) / 2, obs.maxY + routePad},
			)
		}
	}

	// Validate: check each segment for remaining intersections (1 pass max to avoid infinite loops)
	allPoints := make([]Point, 0, len(waypoints)+2)
	allPoints = append(allPoints, src)
	allPoints = append(allPoints, waypoints...)
	allPoints = append(allPoints, tgt)

	var extra []Point
	for i := 0; i < len(allPoints)-1; i++ {
		for _, box := range allBoxes {
			expanded := bbox{
				minX: box.minX - routePad/2,
				minY: box.minY - routePad/2,
				maxX: box.maxX + routePad/2,
				maxY: box.maxY + routePad/2,
			}
			if lineIntersectsBox(allPoints[i], allPoints[i+1], expanded) {
				midY := (box.minY + box.maxY) / 2
				segAvgY := (allPoints[i].Y + allPoints[i+1].Y) / 2
				if segAvgY < midY {
					extra = append(extra, Point{(box.minX + box.maxX) / 2, box.minY - routePad})
				} else {
					extra = append(extra, Point{(box.minX + box.maxX) / 2, box.maxY + routePad})
				}
			}
		}
	}

	if len(extra) > 0 {
		waypoints = append(waypoints, extra...)
		// Sort all waypoints by X to maintain left-to-right order
		sort.Slice(waypoints, func(i, j int) bool {
			return waypoints[i].X < waypoints[j].X
		})
	}

	return waypoints
}

// lineIntersectsBox checks if line segment p1→p2 intersects axis-aligned bounding box.
// Uses parametric line-AABB intersection (slab method).
func lineIntersectsBox(p1, p2 Point, box bbox) bool {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y

	var tmin, tmax float64
	tmin = 0
	tmax = 1

	const eps = 1e-9
	if math.Abs(dx) > eps {
		tx1 := (box.minX - p1.X) / dx
		tx2 := (box.maxX - p1.X) / dx
		if tx1 > tx2 {
			tx1, tx2 = tx2, tx1
		}
		tmin = math.Max(tmin, tx1)
		tmax = math.Min(tmax, tx2)
	} else if p1.X < box.minX || p1.X > box.maxX {
		return false
	}

	if math.Abs(dy) > eps {
		ty1 := (box.minY - p1.Y) / dy
		ty2 := (box.maxY - p1.Y) / dy
		if ty1 > ty2 {
			ty1, ty2 = ty2, ty1
		}
		tmin = math.Max(tmin, ty1)
		tmax = math.Min(tmax, ty2)
	} else if p1.Y < box.minY || p1.Y > box.maxY {
		return false
	}

	return tmin <= tmax
}

// --- Hierarchical layout (dagre-style) ---

func dagreLayout(nodes []model.Node, edges []model.Edge) []LayoutPosition {
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	nodeMap := make(map[string]*model.Node)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	var topLevel []string
	children := make(map[string][]string)
	for _, n := range nodes {
		if n.ParentID == nil {
			topLevel = append(topLevel, n.ID)
		} else {
			children[*n.ParentID] = append(children[*n.ParentID], n.ID)
		}
	}

	groupSizes := make(map[string][2]float64)
	childRelPos := make(map[string][2]float64)

	for _, n := range nodes {
		if n.Type != model.NodeTypeGroup {
			continue
		}
		kids := children[n.ID]
		w, h := autoSizeGroup(kids)
		groupSizes[n.ID] = [2]float64{w, h}
		positionChildrenRelative(kids, w, h, childRelPos)
	}

	inDegree := make(map[string]int)
	outEdges := make(map[string][]string)
	topSet := make(map[string]bool)
	for _, id := range topLevel {
		topSet[id] = true
		inDegree[id] = 0
	}

	for _, e := range edges {
		src, tgt := e.SourceID, e.TargetID
		src = resolveTopLevel(src, nodeMap)
		tgt = resolveTopLevel(tgt, nodeMap)
		if src == tgt || !topSet[src] || !topSet[tgt] {
			continue
		}
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		outEdges[src] = append(outEdges[src], tgt)
		inDegree[tgt]++
	}

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

	for _, id := range topLevel {
		if _, ok := layers[id]; !ok {
			layers[id] = 0
		}
	}

	layerNodes := make(map[int][]string)
	for _, id := range topLevel {
		l := layers[id]
		layerNodes[l] = append(layerNodes[l], id)
	}

	positions := make(map[string][2]float64)
	for layer, ids := range layerNodes {
		count := len(ids)
		for i, id := range ids {
			x := float64(layer) * layerSpaceX
			y := (float64(i) - float64(count-1)/2.0) * layerSpaceY
			positions[id] = [2]float64{x, y}
		}
	}

	result := make([]LayoutPosition, 0, len(nodes))
	for _, n := range nodes {
		lp := LayoutPosition{ID: n.ID}

		if n.ParentID != nil {
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

func gridLayout(nodes []model.Node) []LayoutPosition {
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
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
			w, h := autoSizeGroup(nil)
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

	startX := -float64(cols-1) * cellW / 2.0
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
