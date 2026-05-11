package service

import (
	"fmt"
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

const (
	nodeW         = 72.0
	nodeH         = 72.0
	cellPad       = 60.0
	groupPadX     = 30.0
	groupPadTop   = 45.0
	groupPadBot   = 30.0
	layerGapX     = 60.0 // horizontal gap between layers
	layerGapY     = 40.0 // vertical gap between nodes in same layer
	groupGapY     = 80.0 // vertical gap around group nodes
	groupLaneGapY = 48.0 // compact same-layer gap when a group is involved
	minGroupW     = 160.0
	minGroupH     = 100.0
	gridSize      = 40.0 // snap positions to this grid for metro-style alignment

	// Compact rectangular child templates keep common 4-8 node groups readable
	// without making the rendered group box excessively wide.
	rectCompactCornerX = 56.0
	rectCompactCornerY = 56.0
	rectCompactSideX   = 88.0
	rectCompactSideY   = 84.0
	rectDenseCornerX   = 46.0
	rectDenseCornerY   = 44.0
	rectDenseSideX     = 70.0
	rectDenseSideY     = 66.0

	// Group-internal layout needs a larger effective footprint than the bare
	// 72x72 Cytoscape body because long wrapped labels and thick outlines make
	// nodes look overlapped much earlier than the raw geometry suggests.
	childLayoutNodeW         = 104.0
	childLayoutNodeH         = 92.0
	childLayoutCellPad       = 60.0
	childRoutePad            = 10.0
	childGrowFactor          = 1.10
	childGrowPasses          = 6
	childCompactFactor       = 0.92
	childCompactPasses       = 4
	childCompactNodeW        = 82.0
	childCompactNodeH        = 76.0
	childCompactPad          = 3.0
	childCompactRoutePad     = 4.0
	childCorridorPad         = 18.0
	childCompactCorridorPad  = 10.0
	childHeaderCorridorPad   = 10.0
	passThroughLaneGap       = 34.0
	passThroughStep          = 96.0
	lineMainOffset           = 36.0
	lineSidecarOffset        = 152.0
	lineBoundaryMainX        = 136.0
	lineBoundarySideX        = 84.0
	lineAxisStep             = 116.0
	lineCompactMainOffset    = 28.0
	lineCompactSidecarOffset = 104.0
	lineCompactBoundaryMainX = 96.0
	lineCompactBoundarySideX = 62.0
	lineCompactAxisStep      = 84.0
	twoColumnLeftX           = -84.0
	twoColumnRightX          = 116.0
	twoColumnStepY           = 120.0
	twoColumnCompactLeftX    = -62.0
	twoColumnCompactRightX   = 82.0
	twoColumnCompactStepY    = 88.0
	routeHotspotCell         = gridSize * 3
	routeHotspotStep         = gridSize
	routeHotspotTrim         = nodeW
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

	// Detect "satellite" top-level nodes: non-group nodes whose resolved edges
	// all target groups (never other plain top-level nodes). These are visually
	// attached to a dominant group and should be placed adjacent to it rather
	// than allocated their own layer column, which tends to create awkward
	// gaps. If a node has edges to multiple groups but >50% go to one group,
	// we still classify it as a "cross-group satellite" of the dominant group
	// and place it BELOW that group, biasing its X toward the weighted mean of
	// its source groups so far-left sources get a shorter connection.
	satelliteDominant := make(map[string]string)
	satelliteSide := make(map[string]int)               // -1 = left, +1 = right, 0 = below
	satelliteSources := make(map[string]map[string]int) // cross-group only: groupID → edge count
	{
		groupEdgeCount := make(map[string]map[string]int)
		outToGroup := make(map[string]map[string]int)
		inFromGroup := make(map[string]map[string]int)
		hasNonGroupAdj := make(map[string]bool)
		for id := range topSet {
			if _, isGroup := groupSizes[id]; isGroup {
				continue
			}
			groupEdgeCount[id] = map[string]int{}
			outToGroup[id] = map[string]int{}
			inFromGroup[id] = map[string]int{}
		}
		for _, e := range edges {
			if e.EdgeType == model.EdgeTypeRelated {
				continue
			}
			src := resolveTopLevel(e.SourceID, nodeMap)
			tgt := resolveTopLevel(e.TargetID, nodeMap)
			if src == tgt || !topSet[src] || !topSet[tgt] {
				continue
			}
			if counts, ok := groupEdgeCount[src]; ok {
				if _, isG := groupSizes[tgt]; isG {
					counts[tgt]++
					outToGroup[src][tgt]++
				} else {
					hasNonGroupAdj[src] = true
				}
			}
			if counts, ok := groupEdgeCount[tgt]; ok {
				if _, isG := groupSizes[src]; isG {
					counts[src]++
					inFromGroup[tgt][src]++
				} else {
					hasNonGroupAdj[tgt] = true
				}
			}
		}
		for id, counts := range groupEdgeCount {
			if hasNonGroupAdj[id] || len(counts) == 0 {
				continue
			}
			var (
				total int
				bestG string
				bestN int
			)
			for g, n := range counts {
				total += n
				if n > bestN {
					bestN = n
					bestG = g
				}
			}
			dominant := bestN*2 > total // strict majority (>50%)
			isSoleGroup := len(counts) == 1
			if !isSoleGroup && !dominant {
				continue
			}
			satelliteDominant[id] = bestG

			outN := outToGroup[id][bestG]
			inN := inFromGroup[id][bestG]

			if isSoleGroup {
				switch {
				case outN > 0 && inN == 0:
					satelliteSide[id] = -1
				case inN > 0 && outN == 0:
					satelliteSide[id] = +1
				default:
					satelliteSide[id] = 0
				}
			} else {
				// Cross-group satellites still benefit from a semantic side:
				// pure sources read better on the left, pure sinks on the right,
				// and only mixed-direction satellites fall back below.
				totalOut, totalIn := 0, 0
				for _, n := range outToGroup[id] {
					totalOut += n
				}
				for _, n := range inFromGroup[id] {
					totalIn += n
				}
				switch {
				case totalOut > 0 && totalIn == 0:
					satelliteSide[id] = -1
				case totalIn > 0 && totalOut == 0:
					satelliteSide[id] = +1
				default:
					satelliteSide[id] = 0
				}
				src := make(map[string]int, len(counts))
				for g, n := range counts {
					src[g] = n
				}
				satelliteSources[id] = src
			}
		}
	}

	if len(satelliteDominant) > 0 {
		filtered := make([]string, 0, len(topLevel))
		for _, id := range topLevel {
			if _, isSat := satelliteDominant[id]; isSat {
				delete(topSet, id)
				delete(inDegree, id)
				continue
			}
			filtered = append(filtered, id)
		}
		topLevel = filtered
		for sat := range satelliteDominant {
			delete(outEdges, sat)
		}
		for src, targets := range outEdges {
			kept := make([]string, 0, len(targets))
			for _, tgt := range targets {
				if _, isSat := satelliteDominant[tgt]; !isSat {
					kept = append(kept, tgt)
				}
			}
			outEdges[src] = kept
		}
		for id := range inDegree {
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

	// Compute maxLayer for sink push
	maxLayer := 0
	for _, id := range topLevel {
		if l := layers[id]; l > maxLayer {
			maxLayer = l
		}
	}

	// Sink push: move each node to the latest possible layer (minimize edge span)
	for i := len(topLevel) - 1; i >= 0; i-- {
		id := topLevel[i]
		targets := outEdges[id]
		if len(targets) == 0 {
			continue
		}
		minSuccLayer := maxLayer + 1
		for _, tgt := range targets {
			if l, ok := layers[tgt]; ok && l < minSuccLayer {
				minSuccLayer = l
			}
		}
		if minSuccLayer > 0 && minSuccLayer-1 > layers[id] {
			layers[id] = minSuccLayer - 1
		}
	}

	// Separate isolated nodes (no dependency edges) for compact placement
	hasDepEdge := make(map[string]bool)
	for src, targets := range outEdges {
		if topSet[src] {
			hasDepEdge[src] = true
			for _, tgt := range targets {
				hasDepEdge[tgt] = true
			}
		}
	}
	var isolatedTopLevel []string
	filteredTop := make([]string, 0, len(topLevel))
	for _, id := range topLevel {
		if hasDepEdge[id] {
			filteredTop = append(filteredTop, id)
		} else {
			isolatedTopLevel = append(isolatedTopLevel, id)
		}
	}
	topLevel = filteredTop

	// Let top-level nodes/groups move by one adjacent layer before we freeze the
	// column structure. This is the first place where the placement itself can
	// react to predicted route crossings rather than only reordering inside a
	// fixed layer.
	refineTopLevelLayersByPredictedRoutes(topLevel, layers, outEdges, edges, nodeMap, childRelPos, groupSizes)

	layerNodes, maxLayer := buildTopLevelLayerNodes(topLevel, layers)

	layerX := computeLayerXPositions(layerNodes, maxLayer, groupSizes)

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
	// Build layerOf map for sifting
	layerOf := make(map[string]int)
	for l, ids := range layerNodes {
		for _, id := range ids {
			layerOf[id] = l
		}
	}

	// Crossing minimization via sifting
	siftingCrossMin(layerNodes, maxLayer, fullAdj)
	refineLayerOrderByChildConnections(layerNodes, maxLayer, edges, nodeMap, childRelPos)

	// Coordinate assignment via Brandes-Kopf
	yCoords := brandesKopfAssign(layerNodes, maxLayer, fullAdj, groupSizes, layerOf)

	// Overlap removal: enforce minimum Y spacing within each layer
	// Use sifting-determined order (layerNodes), NOT Y-sorted order
	for l := 0; l <= maxLayer; l++ {
		ids := layerNodes[l]
		if len(ids) <= 1 {
			continue
		}
		enforceLayerSpacing(ids, yCoords, groupSizes)
		// Re-center around 0
		first := yCoords[ids[0]]
		last := yCoords[ids[len(ids)-1]]
		offset := (first + last) / 2
		for _, id := range ids {
			yCoords[id] -= offset
		}
	}

	// Freeze child layouts before the final top-level passes. This keeps
	// top-level corridor and spacing decisions in sync with the child geometry
	// that will actually be rendered.
	coarsePositions := buildTopLevelPositions(layerNodes, maxLayer, layerX, yCoords)
	// Inject provisional satellite positions so child-layout external pulls can
	// see them. Final satellite placement happens after main positioning.
	for sat, g := range satelliteDominant {
		gPos, ok := coarsePositions[g]
		if !ok {
			continue
		}
		gSz := groupSizes[g]
		anchorX, anchorY := gPos[0], gPos[1]
		if sources, ok := satelliteSources[sat]; ok {
			anchorX, anchorY = weightedSatelliteAnchor(sources, coarsePositions, anchorX, anchorY)
		}
		switch satelliteSide[sat] {
		case -1:
			coarsePositions[sat] = [2]float64{gPos[0] - gSz[0]/2.0 - layerGapX - nodeW/2.0, anchorY}
		case +1:
			coarsePositions[sat] = [2]float64{gPos[0] + gSz[0]/2.0 + layerGapX + nodeW/2.0, anchorY}
		default:
			coarsePositions[sat] = [2]float64{anchorX, gPos[1] + gSz[1]/2.0 + groupGapY + nodeH/2.0}
		}
	}
	finalizeGroupLayoutsWithExternalPulls(nodes, children, edges, nodeMap, coarsePositions, childRelPos, groupSizes, false, false)
	layerX = computeLayerXPositions(layerNodes, maxLayer, groupSizes)
	refineLayerOrderByChildConnections(layerNodes, maxLayer, edges, nodeMap, childRelPos)
	yCoords = brandesKopfAssign(layerNodes, maxLayer, fullAdj, groupSizes, layerOf)
	longSpans := collectLongEdgeSpans(edges, nodeMap, topSet, layers, childRelPos)
	for l := 0; l <= maxLayer; l++ {
		ids := layerNodes[l]
		if len(ids) <= 1 {
			continue
		}
		enforceLayerSpacing(ids, yCoords, groupSizes)
		first := yCoords[ids[0]]
		last := yCoords[ids[len(ids)-1]]
		offset := (first + last) / 2
		for _, id := range ids {
			yCoords[id] -= offset
		}
	}

	// Pre-reserve corridors for long-span edges: ensure intermediate layers
	// have gaps at the Y coordinates where cross-layer edges will route.
	reserveEdgeCorridors(layerNodes, maxLayer, yCoords, groupSizes, longSpans, layers)

	// Reserve coarse top/mid/bottom channels for the top-level graph before the
	// finer corridor pass. This gives large groups room away from major flows.
	bandTargets := computeTopLevelBandTargets(topLevel, layers, outEdges, groupSizes)
	anchoredBandTargets := filterAnchoredBandTargets(topLevel, bandTargets, outEdges, groupSizes)
	for l := 0; l <= maxLayer; l++ {
		reorderLayerByDesiredY(layerNodes[l], anchoredBandTargets, yCoords)
		relaxLayerTowardDesiredY(layerNodes[l], yCoords, anchoredBandTargets, groupSizes)
	}

	// Re-pack each layer against the straight corridor of cross-layer edges.
	// This makes the node placement itself respect likely edge paths.
	repackLayersAgainstEdgeCorridors(layerNodes, maxLayer, layerX, yCoords, groupSizes, layers, fullAdj, longSpans, anchoredBandTargets)

	// The earlier sifting pass only sees abstract adjacency crossings.
	// Run one more local search against the actual predicted top-level routes so
	// unrelated edges stop piling into the same crossing hotspot.
	refineLayerOrderByPredictedRoutes(layerNodes, maxLayer, layerX, yCoords, groupSizes, edges, nodeMap, childRelPos)

	// After the ordering is settled, refine each node/group center within its
	// layer against the predicted routes themselves so placement chooses gaps
	// that avoid crossing-heavy corridors instead of only reacting afterwards.
	refineLayerCentersByPredictedRoutes(layerNodes, maxLayer, layerX, yCoords, groupSizes, edges, nodeMap, childRelPos, bandTargets)

	// Re-enforce minimum spacing after corridor repacking, which may have
	// shifted nodes into their neighbours' space.
	for l := 0; l <= maxLayer; l++ {
		ids := layerNodes[l]
		if len(ids) <= 1 {
			continue
		}
		enforceLayerSpacing(ids, yCoords, groupSizes)
	}

	// Build positions map
	positions := buildTopLevelPositions(layerNodes, maxLayer, layerX, yCoords)

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

	// Place satellite nodes (group-bound top-level nodes) adjacent to their
	// dominant group, honoring edge direction: upstream → left, downstream →
	// right, bidirectional → below the group's bounding box.
	if len(satelliteDominant) > 0 {
		type satBucket struct {
			left, right, below []string
		}
		byGroup := make(map[string]*satBucket)
		for sat, g := range satelliteDominant {
			b, ok := byGroup[g]
			if !ok {
				b = &satBucket{}
				byGroup[g] = b
			}
			switch satelliteSide[sat] {
			case -1:
				b.left = append(b.left, sat)
			case +1:
				b.right = append(b.right, sat)
			default:
				b.below = append(b.below, sat)
			}
		}
		for g, b := range byGroup {
			gPos, ok := positions[g]
			if !ok {
				continue
			}
			gSz := groupSizes[g]
			sort.Strings(b.left)
			sort.Strings(b.right)
			sort.Strings(b.below)

			if len(b.left) > 0 {
				x := gPos[0] - gSz[0]/2.0 - layerGapX - nodeW/2.0
				totalH := float64(len(b.left))*nodeH + float64(len(b.left)-1)*cellPad
				y := gPos[1] - totalH/2.0 + nodeH/2.0
				for _, id := range b.left {
					yPos := y
					if sources, ok := satelliteSources[id]; ok {
						_, yPos = weightedSatelliteAnchor(sources, positions, x, yPos)
					}
					positions[id] = [2]float64{snapToGrid(x), snapToGrid(yPos)}
					y += nodeH + cellPad
				}
			}
			if len(b.right) > 0 {
				x := gPos[0] + gSz[0]/2.0 + layerGapX + nodeW/2.0
				totalH := float64(len(b.right))*nodeH + float64(len(b.right)-1)*cellPad
				y := gPos[1] - totalH/2.0 + nodeH/2.0
				for _, id := range b.right {
					yPos := y
					if sources, ok := satelliteSources[id]; ok {
						_, yPos = weightedSatelliteAnchor(sources, positions, x, yPos)
					}
					positions[id] = [2]float64{snapToGrid(x), snapToGrid(yPos)}
					y += nodeH + cellPad
				}
			}
			if len(b.below) > 0 {
				y := gPos[1] + gSz[1]/2.0 + groupGapY + nodeH/2.0
				for _, id := range b.below {
					baseX, _ := weightedSatelliteAnchor(satelliteSources[id], positions, gPos[0], y)
					positions[id] = [2]float64{snapToGrid(baseX), snapToGrid(y)}
					y += nodeH + cellPad
				}
			}
		}
	}

	// Place isolated nodes (no dependency edges) in a compact grid below the main layout
	if len(isolatedTopLevel) > 0 {
		maxY := 0.0
		for _, pos := range positions {
			if pos[1] > maxY {
				maxY = pos[1]
			}
		}
		startY := snapToGrid(maxY + groupGapY)
		cols := int(math.Ceil(math.Sqrt(float64(len(isolatedTopLevel)))))
		if cols < 1 {
			cols = 1
		}
		cellW := snapToGrid(nodeW + layerGapX)
		cellH := snapToGrid(nodeH + layerGapY)
		for _, id := range isolatedTopLevel {
			if sz, ok := groupSizes[id]; ok {
				if sz[0]+layerGapX > cellW {
					cellW = snapToGrid(sz[0] + layerGapX)
				}
				if sz[1]+layerGapY > cellH {
					cellH = snapToGrid(sz[1] + layerGapY)
				}
			}
		}
		for i, id := range isolatedTopLevel {
			col := i % cols
			row := i / cols
			positions[id] = [2]float64{
				snapToGrid(float64(col) * cellW),
				snapToGrid(startY + float64(row)*cellH),
			}
		}
	}

	// Dedicated overlap resolution: push overlapping top-level boxes apart
	// along their smallest overlap axis before the general route cleanup.
	resolveTopLevelOverlaps(positions, groupSizes)

	// Final hard cleanup: if the predicted route between top-level elements still
	// crosses unrelated group/node boxes, locally move the offending boxes.
	cleanupRouteBoxIntersections(positions, edges, nodeMap, childRelPos, groupSizes)

	// Re-run child layout finalization with the FINAL top-level positions so
	// children within groups align toward their actual external connections
	// (the first run used coarse positions which may differ significantly).
	finalizeGroupLayoutsWithExternalPulls(nodes, children, edges, nodeMap, positions, childRelPos, groupSizes, true, true)

	// Micro-adjust children to steer cross-group routes through gaps.
	adjustChildrenToAvoidRouteCrossings(positions, edges, nodeMap, childRelPos, groupSizes)

	// Second cleanup pass: the child adjustments may have fixed most
	// route-box intersections; mop up any remaining violations.
	cleanupRouteBoxIntersections(positions, edges, nodeMap, childRelPos, groupSizes)

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
