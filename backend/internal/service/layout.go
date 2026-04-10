package service

import (
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

const (
	nodeW       = 72.0
	nodeH       = 72.0
	cellPad     = 60.0
	groupPadX   = 30.0
	groupPadTop = 45.0
	groupPadBot = 30.0
	layerGapX   = 80.0 // horizontal gap between layers
	layerGapY   = 60.0 // vertical gap between nodes in same layer
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
	return LayoutResult{Positions: positions}
}

// --- Edge routing: Manhattan-style orthogonal routing ---

type bbox struct {
	minX, minY, maxX, maxY float64
}

func routeEdges(positions []LayoutPosition, edges []model.Edge, nodes []model.Node) []EdgeRoute {
	topLevel := make(map[string]bool)
	for _, n := range nodes {
		if n.ParentID == nil {
			topLevel[n.ID] = true
		}
	}

	posMap := make(map[string]Point)
	boxes := make(map[string]bbox)
	for _, lp := range positions {
		posMap[lp.ID] = Point{lp.X, lp.Y}
		if !topLevel[lp.ID] {
			continue
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

		var obstacles []bbox
		for id, box := range boxes {
			if id == edge.SourceID || id == edge.TargetID {
				continue
			}
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

		sort.Slice(obstacles, func(i, j int) bool {
			di := math.Hypot(((obstacles[i].minX+obstacles[i].maxX)/2)-srcPos.X, ((obstacles[i].minY+obstacles[i].maxY)/2)-srcPos.Y)
			dj := math.Hypot(((obstacles[j].minX+obstacles[j].maxX)/2)-srcPos.X, ((obstacles[j].minY+obstacles[j].maxY)/2)-srcPos.Y)
			return di < dj
		})

		waypoints := manhattanRoute(srcPos, tgtPos, obstacles)
		routes = append(routes, EdgeRoute{ID: edge.ID, Waypoints: waypoints})
	}
	return routes
}

// manhattanRoute generates orthogonal (L/Z-shaped) waypoints around obstacles.
func manhattanRoute(src, tgt Point, obstacles []bbox) []Point {
	if len(obstacles) == 0 {
		return nil
	}

	// Merge all obstacle bounding boxes into a single avoidance zone
	combined := obstacles[0]
	for _, obs := range obstacles[1:] {
		combined.minX = math.Min(combined.minX, obs.minX)
		combined.minY = math.Min(combined.minY, obs.minY)
		combined.maxX = math.Max(combined.maxX, obs.maxX)
		combined.maxY = math.Max(combined.maxY, obs.maxY)
	}

	// Determine whether to route above or below the obstacle zone
	midY := (combined.minY + combined.maxY) / 2
	avgY := (src.Y + tgt.Y) / 2

	var routeY float64
	if avgY < midY {
		routeY = combined.minY - routePad
	} else {
		routeY = combined.maxY + routePad
	}

	// Manhattan route: src → horizontal out → vertical to routeY → horizontal to tgt.X → down to tgt
	var waypoints []Point

	// Only add waypoints if they actually change direction
	if math.Abs(src.Y-routeY) > 1 || math.Abs(tgt.Y-routeY) > 1 {
		waypoints = append(waypoints,
			Point{src.X, routeY},  // go vertical from source
			Point{tgt.X, routeY},  // go horizontal at avoidance level
		)
	}

	return waypoints
}

// lineIntersectsBox checks if line segment p1→p2 intersects axis-aligned bounding box.
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

// --- Hierarchical layout (dagre-style) with dynamic spacing ---

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
		w, h := computeGroupSizeAndLayout(kids, edges, nodeMap, childRelPos)
		groupSizes[n.ID] = [2]float64{w, h}
	}

	// BFS layer assignment
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
	maxLayer := 0
	for _, id := range topLevel {
		l := layers[id]
		layerNodes[l] = append(layerNodes[l], id)
		if l > maxLayer {
			maxLayer = l
		}
	}

	// Dynamic X spacing: compute per-layer max width
	layerMaxW := make(map[int]float64)
	for l := 0; l <= maxLayer; l++ {
		maxW := nodeW
		for _, id := range layerNodes[l] {
			if sz, ok := groupSizes[id]; ok && sz[0] > maxW {
				maxW = sz[0]
			}
		}
		layerMaxW[l] = maxW
	}

	// Cumulative X positioning
	layerX := make(map[int]float64)
	cumX := 0.0
	for l := 0; l <= maxLayer; l++ {
		layerX[l] = cumX + layerMaxW[l]/2
		cumX += layerMaxW[l] + layerGapX
	}

	// Dynamic Y spacing per layer
	positions := make(map[string][2]float64)
	for layer, ids := range layerNodes {
		// Compute max height for nodes in this layer
		maxH := nodeH
		for _, id := range ids {
			if sz, ok := groupSizes[id]; ok && sz[1] > maxH {
				maxH = sz[1]
			}
		}
		spacing := maxH + layerGapY
		count := len(ids)
		for i, id := range ids {
			x := layerX[layer]
			y := (float64(i) - float64(count-1)/2.0) * spacing
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
	spacing := nodeW + cellPad + layerGapX
	result := make([]LayoutPosition, len(nodes))

	for i, n := range nodes {
		col := i % cols
		row := i / cols
		result[i] = LayoutPosition{
			ID: n.ID,
			X:  float64(col) * spacing,
			Y:  float64(row) * spacing,
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

// computeGroupSizeAndLayout computes group dimensions AND positions children.
// Returns (width, height) of the group. Positions are written to relPos.
func computeGroupSizeAndLayout(childIDs []string, edges []model.Edge, nodeMap map[string]*model.Node, relPos map[string][2]float64) (float64, float64) {
	count := len(childIDs)
	if count == 0 {
		return minGroupW, minGroupH
	}

	childSet := make(map[string]bool, count)
	for _, id := range childIDs {
		childSet[id] = true
	}

	var childEdges []model.Edge
	for _, e := range edges {
		if childSet[e.SourceID] && childSet[e.TargetID] && e.EdgeType != model.EdgeTypeRelated {
			childEdges = append(childEdges, e)
		}
	}

	cellW := nodeW + cellPad
	cellH := nodeH + cellPad

	if len(childEdges) > 0 {
		return layoutChildrenByLayers(childIDs, childEdges, cellW, cellH, relPos)
	}
	return layoutChildrenGrid(childIDs, cellW, cellH, relPos)
}

// layoutChildrenByLayers does mini BFS layering. Returns (groupW, groupH).
func layoutChildrenByLayers(childIDs []string, childEdges []model.Edge, cellW, cellH float64, relPos map[string][2]float64) (float64, float64) {
	inDeg := make(map[string]int)
	succ := make(map[string][]string)
	for _, id := range childIDs {
		inDeg[id] = 0
	}
	for _, e := range childEdges {
		succ[e.SourceID] = append(succ[e.SourceID], e.TargetID)
		inDeg[e.TargetID]++
	}

	var layerOrder [][]string
	var queue []string
	for _, id := range childIDs {
		if inDeg[id] == 0 {
			queue = append(queue, id)
		}
	}
	placed := make(map[string]bool)
	for len(queue) > 0 {
		layerOrder = append(layerOrder, queue)
		for _, id := range queue {
			placed[id] = true
		}
		var next []string
		for _, id := range queue {
			for _, tgt := range succ[id] {
				inDeg[tgt]--
				if inDeg[tgt] == 0 && !placed[tgt] {
					next = append(next, tgt)
				}
			}
		}
		queue = next
	}
	for _, id := range childIDs {
		if !placed[id] {
			layerOrder = append(layerOrder, []string{id})
			placed[id] = true
		}
	}

	numLayers := len(layerOrder)
	maxPerLayer := 0
	for _, layer := range layerOrder {
		if len(layer) > maxPerLayer {
			maxPerLayer = len(layer)
		}
	}

	// Size group to fit ALL children without overlap
	gridW := float64(numLayers) * cellW
	gridH := float64(maxPerLayer) * cellH
	groupW := math.Max(gridW+groupPadX*2, minGroupW)
	groupH := math.Max(gridH+groupPadTop+groupPadBot, minGroupH)

	// Position children: layers left-to-right, nodes top-to-bottom
	startX := -float64(numLayers-1) * cellW / 2

	for li, layer := range layerOrder {
		n := len(layer)
		startY := -float64(n-1)*cellH/2 + (groupPadTop-groupPadBot)/2

		for ni, id := range layer {
			relPos[id] = [2]float64{
				startX + float64(li)*cellW,
				startY + float64(ni)*cellH,
			}
		}
	}

	return groupW, groupH
}

// layoutChildrenGrid arranges children in a grid. Returns (groupW, groupH).
func layoutChildrenGrid(childIDs []string, cellW, cellH float64, relPos map[string][2]float64) (float64, float64) {
	count := len(childIDs)
	cols := int(math.Ceil(math.Sqrt(float64(count))))
	rows := int(math.Ceil(float64(count) / float64(cols)))

	gridW := float64(cols) * cellW
	gridH := float64(rows) * cellH
	groupW := math.Max(gridW+groupPadX*2, minGroupW)
	groupH := math.Max(gridH+groupPadTop+groupPadBot, minGroupH)

	startX := -float64(cols-1) * cellW / 2.0
	startY := -float64(rows-1)*cellH/2.0 + (groupPadTop-groupPadBot)/2.0

	for i, id := range childIDs {
		col := i % cols
		row := i / cols
		relPos[id] = [2]float64{
			startX + float64(col)*cellW,
			startY + float64(row)*cellH,
		}
	}

	return groupW, groupH
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
