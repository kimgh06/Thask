package service

import (
	"fmt"
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
	layerGapX   = 80.0  // horizontal gap between layers
	layerGapY   = 120.0 // vertical gap between nodes in same layer
	groupGapY   = 200.0 // vertical gap around group nodes
	minGroupW   = 160.0
	minGroupH   = 100.0
	gridSize    = 40.0 // snap positions to this grid for metro-style alignment
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

type LayoutResult struct {
	Positions []LayoutPosition
}

// snapToGrid rounds a value to the nearest grid point.
func snapToGrid(v float64) float64 {
	return math.Round(v/gridSize) * gridSize
}

// reorderByBarycenter reorders nodes within layers to minimize edge crossings
// using the barycenter heuristic with forward/backward sweeps.
func reorderByBarycenter(layerNodes map[int][]string, maxLayer int, adj map[string]map[string]bool) {
	// Build index: node → layer, node → position within its layer
	layerOf := make(map[string]int)
	indexOf := make(map[string]int)
	for l, ids := range layerNodes {
		for i, id := range ids {
			layerOf[id] = l
			indexOf[id] = i
		}
	}

	// 12 forward+backward sweep pairs for convergence
	for sweep := 0; sweep < 12; sweep++ {
		// Forward sweep: layer 1 → maxLayer
		for l := 1; l <= maxLayer; l++ {
			ids := layerNodes[l]
			bary := make(map[string]float64, len(ids))
			for _, id := range ids {
				var neighborPositions []float64
				for nid := range adj[id] {
					if layerOf[nid] == l-1 {
						neighborPositions = append(neighborPositions, float64(indexOf[nid]))
					}
				}
				count := len(neighborPositions)
				if count > 0 {
					sort.Float64s(neighborPositions)
					bary[id] = neighborPositions[len(neighborPositions)/2]
				} else {
					bary[id] = float64(indexOf[id])
				}
			}
			sort.SliceStable(ids, func(i, j int) bool {
				return bary[ids[i]] < bary[ids[j]]
			})
			for i, id := range ids {
				indexOf[id] = i
			}
		}
		// Backward sweep: layer maxLayer-1 → 0
		for l := maxLayer - 1; l >= 0; l-- {
			ids := layerNodes[l]
			bary := make(map[string]float64, len(ids))
			for _, id := range ids {
				var neighborPositions []float64
				for nid := range adj[id] {
					if layerOf[nid] == l+1 {
						neighborPositions = append(neighborPositions, float64(indexOf[nid]))
					}
				}
				count := len(neighborPositions)
				if count > 0 {
					sort.Float64s(neighborPositions)
					bary[id] = neighborPositions[len(neighborPositions)/2]
				} else {
					bary[id] = float64(indexOf[id])
				}
			}
			sort.SliceStable(ids, func(i, j int) bool {
				return bary[ids[i]] < bary[ids[j]]
			})
			for i, id := range ids {
				indexOf[id] = i
			}
		}
	}
}

// adjacentExchange swaps adjacent pairs in each layer if it reduces edge crossings.
func adjacentExchange(layerNodes map[int][]string, maxLayer int, adj map[string]map[string]bool) {
	layerOf := make(map[string]int)
	indexOf := make(map[string]int)
	for l, ids := range layerNodes {
		for i, id := range ids {
			layerOf[id] = l
			indexOf[id] = i
		}
	}

	for improved := true; improved; {
		improved = false
		for l := 0; l <= maxLayer; l++ {
			ids := layerNodes[l]
			for i := 0; i < len(ids)-1; i++ {
				u, v := ids[i], ids[i+1]
				crossBefore := countPairCrossings(u, v, l, adj, indexOf, layerOf)
				crossAfter := countPairCrossings(v, u, l, adj, indexOf, layerOf)
				if crossAfter < crossBefore {
					ids[i], ids[i+1] = ids[i+1], ids[i]
					indexOf[ids[i]] = i
					indexOf[ids[i+1]] = i + 1
					improved = true
				}
			}
		}
	}
}

// countPairCrossings counts edge crossings involving two adjacent nodes u (at position uPos) and v (at position vPos).
func countPairCrossings(u, v string, layer int, adj map[string]map[string]bool, indexOf map[string]int, layerOf map[string]int) int {
	crossings := 0
	uIdx := indexOf[u]
	vIdx := indexOf[v]

	for _, adjLayer := range []int{layer - 1, layer + 1} {
		var uNeighbors, vNeighbors []int
		for nid := range adj[u] {
			if layerOf[nid] == adjLayer {
				uNeighbors = append(uNeighbors, indexOf[nid])
			}
		}
		for nid := range adj[v] {
			if layerOf[nid] == adjLayer {
				vNeighbors = append(vNeighbors, indexOf[nid])
			}
		}
		for _, a := range uNeighbors {
			for _, b := range vNeighbors {
				if (uIdx < vIdx && a > b) || (uIdx > vIdx && a < b) {
					crossings++
				}
			}
		}
	}
	return crossings
}

// findSCCs returns strongly connected components using Tarjan's algorithm.
func findSCCs(nodeIDs []string, outEdges map[string][]string) [][]string {
	index := 0
	stack := make([]string, 0)
	onStack := make(map[string]bool)
	indices := make(map[string]int)
	lowlinks := make(map[string]int)
	defined := make(map[string]bool)
	var sccs [][]string

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlinks[v] = index
		defined[v] = true
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range outEdges[v] {
			if !defined[w] {
				strongConnect(w)
				if lowlinks[w] < lowlinks[v] {
					lowlinks[v] = lowlinks[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlinks[v] {
					lowlinks[v] = indices[w]
				}
			}
		}

		if lowlinks[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	for _, v := range nodeIDs {
		if !defined[v] {
			strongConnect(v)
		}
	}
	return sccs
}

// computeCircleRadius returns a radius so that n nodes don't overlap on a circle.
func computeCircleRadius(n int) float64 {
	minArc := nodeW + cellPad/2
	circumference := float64(n) * minArc
	radius := circumference / (2 * math.Pi)
	if radius < nodeW {
		radius = nodeW
	}
	return snapToGrid(radius)
}

// circularLayout positions nodeIDs evenly on a circle, starting from the top.
func circularLayout(nodeIDs []string, centerX, centerY, radius float64) map[string][2]float64 {
	positions := make(map[string][2]float64, len(nodeIDs))
	n := len(nodeIDs)
	for i, id := range nodeIDs {
		angle := 2*math.Pi*float64(i)/float64(n) - math.Pi/2
		x := snapToGrid(centerX + radius*math.Cos(angle))
		y := snapToGrid(centerY + radius*math.Sin(angle))
		positions[id] = [2]float64{x, y}
	}
	return positions
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

	seenEdge := make(map[string]bool)
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
		// Deduplicate resolved edges and prevent false cycles
		// (e.g., childA→nodeB + nodeB→childB resolves to GroupX→nodeB + nodeB→GroupX)
		edgeKey := src + "|" + tgt
		if seenEdge[edgeKey] {
			continue
		}
		reverseKey := tgt + "|" + src
		if seenEdge[reverseKey] {
			continue // would create false cycle between group and outside node
		}
		seenEdge[edgeKey] = true
		outEdges[src] = append(outEdges[src], tgt)
		inDegree[tgt]++
	}

	// Detect cycles using Tarjan's SCC and collapse them
	sccs := findSCCs(topLevel, outEdges)
	repCycle := make(map[string][]string) // representative → cycle members
	cycleNodeSet := make(map[string]bool) // all cycle member nodes
	for _, scc := range sccs {
		if len(scc) <= 1 {
			continue
		}
		rep := scc[0]
		repCycle[rep] = scc
		for _, id := range scc {
			cycleNodeSet[id] = true
		}
		// Reserve space for circular layout (only if rep is not a real GROUP node)
		if _, isRealGroup := groupSizes[rep]; !isRealGroup {
			radius := computeCircleRadius(len(scc))
			diameter := 2*radius + nodeW
			groupSizes[rep] = [2]float64{diameter, diameter}
		}
	}

	// Remove non-representative cycle nodes from topLevel for BFS
	if len(cycleNodeSet) > 0 {
		filtered := make([]string, 0, len(topLevel))
		for _, id := range topLevel {
			if cycleNodeSet[id] && repCycle[id] == nil {
				// non-representative cycle node → skip
				delete(topSet, id)
				continue
			}
			filtered = append(filtered, id)
		}
		topLevel = filtered
		// Redirect edges that target non-rep cycle nodes to their representative
		for src, targets := range outEdges {
			for i, tgt := range targets {
				if cycleNodeSet[tgt] && repCycle[tgt] == nil {
					// Find representative for this cycle node
					for rep, members := range repCycle {
						for _, m := range members {
							if m == tgt {
								outEdges[src][i] = rep
								break
							}
						}
					}
				}
			}
		}
		// Recalculate inDegree for filtered topLevel
		for id := range inDegree {
			delete(inDegree, id)
		}
		for _, id := range topLevel {
			inDegree[id] = 0
		}
		for src, targets := range outEdges {
			if !topSet[src] {
				continue
			}
			for _, tgt := range targets {
				if !topSet[tgt] || src == tgt {
					continue
				}
				inDegree[tgt]++
			}
		}
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

	// Dynamic X spacing: compute per-layer max width (before adding dummies)
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

	// Cumulative X positioning (grid-aligned)
	layerX := make(map[int]float64)
	cumX := 0.0
	for l := 0; l <= maxLayer; l++ {
		layerX[l] = cumX + snapToGrid(layerMaxW[l]/2)
		cumX += snapToGrid(layerMaxW[l]) + math.Ceil(layerGapX/gridSize)*gridSize
	}

	// Insert dummy nodes for edges spanning 2+ layers (Sugiyama)
	dummySet := make(map[string]bool)
	dummyAdj := make(map[string][]string)
	for _, e := range edges {
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		src := resolveTopLevel(e.SourceID, nodeMap)
		tgt := resolveTopLevel(e.TargetID, nodeMap)
		if src == tgt || !topSet[src] || !topSet[tgt] {
			continue
		}
		srcL, tgtL := layers[src], layers[tgt]
		if srcL > tgtL {
			srcL, tgtL = tgtL, srcL
			src, tgt = tgt, src
		}
		if tgtL-srcL <= 1 {
			continue
		}
		prev := src
		for l := srcL + 1; l < tgtL; l++ {
			did := fmt.Sprintf("__d_%s_%d", e.ID, l)
			dummySet[did] = true
			layers[did] = l
			layerNodes[l] = append(layerNodes[l], did)
			dummyAdj[prev] = append(dummyAdj[prev], did)
			dummyAdj[did] = append(dummyAdj[did], prev)
			prev = did
		}
		dummyAdj[prev] = append(dummyAdj[prev], tgt)
		dummyAdj[tgt] = append(dummyAdj[tgt], prev)
	}

	// Build full adjacency map (real edges + dummy edges)
	fullAdj := make(map[string]map[string]bool)
	for _, e := range edges {
		src := resolveTopLevel(e.SourceID, nodeMap)
		tgt := resolveTopLevel(e.TargetID, nodeMap)
		if src == tgt || e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		if fullAdj[src] == nil {
			fullAdj[src] = make(map[string]bool)
		}
		if fullAdj[tgt] == nil {
			fullAdj[tgt] = make(map[string]bool)
		}
		fullAdj[src][tgt] = true
		fullAdj[tgt][src] = true
	}
	for src, targets := range dummyAdj {
		if fullAdj[src] == nil {
			fullAdj[src] = make(map[string]bool)
		}
		for _, tgt := range targets {
			fullAdj[src][tgt] = true
			if fullAdj[tgt] == nil {
				fullAdj[tgt] = make(map[string]bool)
			}
			fullAdj[tgt][src] = true
		}
	}
	reorderByBarycenter(layerNodes, maxLayer, fullAdj)
	adjacentExchange(layerNodes, maxLayer, fullAdj)

	// Dynamic Y positioning per layer — pack using individual node heights
	// Use wider gaps around group nodes
	positions := make(map[string][2]float64)
	for layer, ids := range layerNodes {
		nodeHts := make([]float64, len(ids))
		isGroup := make([]bool, len(ids))
		for i, id := range ids {
			h := nodeH
			if sz, ok := groupSizes[id]; ok {
				h = sz[1]
				isGroup[i] = true
			}
			nodeHts[i] = h
		}
		totalH := 0.0
		for i, h := range nodeHts {
			totalH += h
			if i < len(ids)-1 {
				if isGroup[i] || isGroup[i+1] {
					totalH += groupGapY
				} else {
					totalH += layerGapY
				}
			}
		}
		y := -totalH / 2
		x := snapToGrid(layerX[layer])
		for i, id := range ids {
			cy := snapToGrid(y + nodeHts[i]/2)
			positions[id] = [2]float64{x, cy}
			gap := layerGapY
			if i < len(ids)-1 && (isGroup[i] || isGroup[i+1]) {
				gap = groupGapY
			}
			y += nodeHts[i] + gap
		}
	}

	// Y-coordinate optimization: pull nodes toward median neighbor Y to straighten edges
	for iter := 0; iter < 8; iter++ {
		for l := 0; l <= maxLayer; l++ {
			ids := layerNodes[l]
			for i, id := range ids {
				var neighborYs []float64
				for nid := range fullAdj[id] {
					if pos, ok := positions[nid]; ok {
						neighborYs = append(neighborYs, pos[1])
					}
				}
				if len(neighborYs) == 0 {
					continue
				}
				sort.Float64s(neighborYs)
				idealY := neighborYs[len(neighborYs)/2]

				h := nodeH
				if sz, ok := groupSizes[id]; ok {
					h = sz[1]
				}

				// Clamp to maintain ordering with neighbors in same layer
				_, idIsGroup := groupSizes[id]
				if i > 0 {
					prevH := nodeH
					_, prevIsGroup := groupSizes[ids[i-1]]
					if sz, ok := groupSizes[ids[i-1]]; ok {
						prevH = sz[1]
					}
					gap := layerGapY / 2
					if idIsGroup || prevIsGroup {
						gap = groupGapY / 2
					}
					minY := positions[ids[i-1]][1] + prevH/2 + gap + h/2
					if idealY < minY {
						idealY = minY
					}
				}
				if i < len(ids)-1 {
					nextH := nodeH
					_, nextIsGroup := groupSizes[ids[i+1]]
					if sz, ok := groupSizes[ids[i+1]]; ok {
						nextH = sz[1]
					}
					gap := layerGapY / 2
					if idIsGroup || nextIsGroup {
						gap = groupGapY / 2
					}
					maxY := positions[ids[i+1]][1] - nextH/2 - gap - h/2
					if idealY > maxY {
						idealY = maxY
					}
				}

				// Cross-layer group avoidance: don't overlap with groups in adjacent layers
				for nid := range fullAdj[id] {
					if npos, ok := positions[nid]; ok {
						if sz, gok := groupSizes[nid]; gok {
							nLayer := layers[nid]
							if nLayer != l {
								halfH := sz[1]/2 + groupGapY/2
								if idealY > npos[1]-halfH && idealY < npos[1]+halfH {
									if idealY < npos[1] {
										idealY = npos[1] - halfH
									} else {
										idealY = npos[1] + halfH
									}
								}
							}
						}
					}
				}

				positions[id] = [2]float64{positions[id][0], snapToGrid(idealY)}
			}
		}
	}

	// Remove dummy nodes from positions (they only influenced ordering)
	for did := range dummySet {
		delete(positions, did)
	}

	// Expand cycle representatives into circular positions
	for rep, members := range repCycle {
		center := positions[rep]
		radius := computeCircleRadius(len(members))
		circPos := circularLayout(members, center[0], center[1], radius)
		for id, pos := range circPos {
			positions[id] = pos
		}
		// Remove the artificial groupSize so it doesn't emit Width/Height
		delete(groupSizes, rep)
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
