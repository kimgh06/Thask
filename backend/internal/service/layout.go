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
	layerGapX   = 60.0 // horizontal gap between layers
	layerGapY   = 40.0 // vertical gap between nodes in same layer
	groupGapY   = 80.0 // vertical gap around group nodes
	groupLaneGapY = 48.0 // compact same-layer gap when a group is involved
	minGroupW   = 160.0
	minGroupH   = 100.0
	gridSize    = 40.0 // snap positions to this grid for metro-style alignment

	// Compact rectangular child templates keep common 4-8 node groups readable
	// without making the rendered group box excessively wide.
	rectCompactCornerX = 56.0
	rectCompactCornerY = 56.0
	rectCompactSideX   = 88.0
	rectCompactSideY   = 84.0
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

type longEdgeSpan struct {
	Src   string
	Tgt   string
	SrcDX float64
	SrcDY float64
	TgtDX float64
	TgtDY float64
}

type verticalBand struct {
	Top    float64
	Bottom float64
}

type rectBox struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

type predictedRoute struct {
	SrcTop string
	TgtTop string
	Points []Point
}

// snapToGrid rounds a value to the nearest grid point.
func snapToGrid(v float64) float64 {
	return math.Round(v/gridSize) * gridSize
}

// siftingCrossMin reorders nodes within each layer to minimize edge crossings.
// For each node, it tries ALL possible positions in its layer and picks the one
// with the fewest crossings. Much more powerful than barycentric/adjacent exchange.
func siftingCrossMin(layerNodes map[int][]string, maxLayer int, adj map[string]map[string]bool) {
	layerOf := make(map[string]int)
	for l, ids := range layerNodes {
		for _, id := range ids {
			layerOf[id] = l
		}
	}

	// Multiple sweeps for convergence (forward + backward)
	for sweep := 0; sweep < 12; sweep++ {
		startL, endL, stepL := 0, maxLayer, 1
		if sweep%2 == 1 {
			startL, endL, stepL = maxLayer, 0, -1
		}
		for l := startL; ; l += stepL {
			ids := layerNodes[l]
			if len(ids) > 1 {
				// For each node, find its optimal position
				for ni := 0; ni < len(ids); ni++ {
					node := ids[ni]
					// Remove node from current position
					remaining := make([]string, 0, len(ids)-1)
					remaining = append(remaining, ids[:ni]...)
					remaining = append(remaining, ids[ni+1:]...)

					// Try each position and count crossings
					bestPos := 0
					bestCross := math.MaxInt64
					for pos := 0; pos <= len(remaining); pos++ {
						trial := make([]string, 0, len(ids))
						trial = append(trial, remaining[:pos]...)
						trial = append(trial, node)
						trial = append(trial, remaining[pos:]...)
						cross := countLayerCrossings(trial, l, adj, layerNodes, layerOf)
						if cross < bestCross {
							bestCross = cross
							bestPos = pos
						}
					}

					// Place node at best position
					newIds := make([]string, 0, len(ids))
					newIds = append(newIds, remaining[:bestPos]...)
					newIds = append(newIds, node)
					newIds = append(newIds, remaining[bestPos:]...)
					copy(ids, newIds)
				}
				layerNodes[l] = ids
			}
			if l == endL {
				break
			}
		}
	}
}

// countLayerCrossings counts crossings between edges from layer l to adjacent layers
func countLayerCrossings(ids []string, l int, adj map[string]map[string]bool, layerNodes map[int][]string, layerOf map[string]int) int {
	indexOf := make(map[string]int)
	for i, id := range ids {
		indexOf[id] = i
	}
	// Also index adjacent layers
	for _, adjL := range []int{l - 1, l + 1} {
		if adjIds, ok := layerNodes[adjL]; ok {
			for i, id := range adjIds {
				indexOf[id] = i
			}
		}
	}

	crossings := 0
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			u, v := ids[i], ids[j]
			for _, adjL := range []int{l - 1, l + 1} {
				for uN := range adj[u] {
					if layerOf[uN] != adjL {
						continue
					}
					for vN := range adj[v] {
						if layerOf[vN] != adjL {
							continue
						}
						// u is above v (i < j). If uN is below vN, crossing.
						if indexOf[uN] > indexOf[vN] {
							crossings++
						}
					}
				}
			}
		}
	}
	return crossings
}

// brandesKopfAssign computes Y coordinates that minimize edge bends by
// aligning connected nodes vertically where possible.
func brandesKopfAssign(layerNodes map[int][]string, maxLayer int, adj map[string]map[string]bool, groupSizes map[string][2]float64, layerOf map[string]int) map[string]float64 {
	// Phase 1: Vertical alignment — choose one neighbor per node to align with.
	// Do 4 passes (upper-left, upper-right, lower-left, lower-right) and take median.
	results := make([]map[string]float64, 4)

	type direction struct {
		layerStart, layerEnd, layerStep int
		pickNeighbor                    func(neighbors []string, indexOf map[string]int) string
	}

	dirs := []direction{
		// Upper-left: layers left to right, pick upper median neighbor
		{0, maxLayer, 1, func(ns []string, idx map[string]int) string {
			if len(ns) == 0 {
				return ""
			}
			sort.Slice(ns, func(i, j int) bool { return idx[ns[i]] < idx[ns[j]] })
			return ns[(len(ns)-1)/2] // upper median
		}},
		// Upper-right: layers right to left, pick upper median
		{maxLayer, -1, -1, func(ns []string, idx map[string]int) string {
			if len(ns) == 0 {
				return ""
			}
			sort.Slice(ns, func(i, j int) bool { return idx[ns[i]] < idx[ns[j]] })
			return ns[(len(ns)-1)/2]
		}},
		// Lower-left: layers left to right, pick lower median
		{0, maxLayer, 1, func(ns []string, idx map[string]int) string {
			if len(ns) == 0 {
				return ""
			}
			sort.Slice(ns, func(i, j int) bool { return idx[ns[i]] < idx[ns[j]] })
			return ns[len(ns)/2] // lower median
		}},
		// Lower-right: layers right to left, pick lower median
		{maxLayer, -1, -1, func(ns []string, idx map[string]int) string {
			if len(ns) == 0 {
				return ""
			}
			sort.Slice(ns, func(i, j int) bool { return idx[ns[i]] < idx[ns[j]] })
			return ns[len(ns)/2]
		}},
	}

	for d, dir := range dirs {
		// Build index of each node's position within its layer
		indexOf := make(map[string]int)
		for _, ids := range layerNodes {
			for i, id := range ids {
				indexOf[id] = i
			}
		}

		// Alignment: root[u] = the root of u's block, align[u] = next node in block
		root := make(map[string]string)
		align := make(map[string]string)
		for l := 0; l <= maxLayer; l++ {
			for _, id := range layerNodes[l] {
				root[id] = id
				align[id] = id
			}
		}

		// For each layer in the sweep direction, align nodes with chosen neighbors
		for l := dir.layerStart; l != dir.layerEnd+dir.layerStep; l += dir.layerStep {
			prevL := l - dir.layerStep
			if prevL < 0 || prevL > maxLayer {
				continue
			}
			for _, u := range layerNodes[l] {
				// Find neighbors in previous layer
				var neighbors []string
				for n := range adj[u] {
					if layerOf[n] == prevL {
						neighbors = append(neighbors, n)
					}
				}
				chosen := dir.pickNeighbor(neighbors, indexOf)
				if chosen == "" {
					continue
				}
				// Align u with chosen
				if align[chosen] == chosen { // chosen not yet aligned
					align[chosen] = u
					root[u] = root[chosen]
				}
			}
		}

		// Phase 2: Compaction — assign Y based on blocks
		yPos := make(map[string]float64)
		placed := make(map[string]bool)

		// Place root nodes first, then propagate to block members
		for l := 0; l <= maxLayer; l++ {
			ids := layerNodes[l]
			y := 0.0
			for _, id := range ids {
				h := nodeH
				if sz, ok := groupSizes[id]; ok {
					h = sz[1]
				}
				if !placed[root[id]] {
					yPos[root[id]] = y + h/2
					placed[root[id]] = true
				}
				// Propagate root's Y to this node
				yPos[id] = yPos[root[id]]
				y += h
				// Gap
				_, isGroup := groupSizes[id]
				nextIsGroup := false
				if i := indexOf[id]; i < len(ids)-1 {
					_, nextIsGroup = groupSizes[ids[i+1]]
				}
				if isGroup || nextIsGroup {
					y += groupGapY
				} else {
					y += layerGapY
				}
			}
		}

		// Center the layout around y=0
		minY, maxYVal := math.MaxFloat64, -math.MaxFloat64
		for _, y := range yPos {
			if y < minY {
				minY = y
			}
			if y > maxYVal {
				maxYVal = y
			}
		}
		center := (minY + maxYVal) / 2
		for id := range yPos {
			yPos[id] -= center
		}

		results[d] = yPos
	}

	// Phase 3: Balance — take median of 4 results for each node
	finalY := make(map[string]float64)
	for l := 0; l <= maxLayer; l++ {
		for _, id := range layerNodes[l] {
			vals := make([]float64, 4)
			for d := 0; d < 4; d++ {
				vals[d] = results[d][id]
			}
			sort.Float64s(vals)
			finalY[id] = (vals[1] + vals[2]) / 2 // median of 4 = average of middle 2
		}
	}

	return finalY
}

func collectLongEdgeSpans(
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	topSet map[string]bool,
	layers map[string]int,
	childRelPos map[string][2]float64,
) []longEdgeSpan {
	spans := make([]longEdgeSpan, 0, len(edges))
	for _, e := range edges {
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		src := resolveTopLevel(e.SourceID, nodeMap)
		tgt := resolveTopLevel(e.TargetID, nodeMap)
		if src == tgt || !topSet[src] || !topSet[tgt] {
			continue
		}
		srcDX, srcDY := edgeEndpointOffset(e.SourceID, src, nodeMap, childRelPos)
		tgtDX, tgtDY := edgeEndpointOffset(e.TargetID, tgt, nodeMap, childRelPos)
		srcL, tgtL := layers[src], layers[tgt]
		if srcL == tgtL {
			continue
		}
		if srcL > tgtL {
			src, tgt = tgt, src
			srcDX, tgtDX = tgtDX, srcDX
			srcDY, tgtDY = tgtDY, srcDY
			srcL, tgtL = tgtL, srcL
		}
		spans = append(spans, longEdgeSpan{
			Src:   src,
			Tgt:   tgt,
			SrcDX: srcDX,
			SrcDY: srcDY,
			TgtDX: tgtDX,
			TgtDY: tgtDY,
		})
	}
	return spans
}

func edgeEndpointOffset(
	nodeID string,
	topID string,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) (float64, float64) {
	n, ok := nodeMap[nodeID]
	if !ok || n.ParentID == nil {
		return 0, 0
	}
	if resolveTopLevel(nodeID, nodeMap) != topID {
		return 0, 0
	}
	rel, ok := childRelPos[nodeID]
	if !ok {
		return 0, 0
	}
	return rel[0], rel[1]
}

func compute8DirWaypoints(src, tgt Point) []Point {
	dx := tgt.X - src.X
	dy := tgt.Y - src.Y
	absDx := math.Abs(dx)
	absDy := math.Abs(dy)

	minBend := nodeW / 2

	if absDx < minBend && absDy >= minBend {
		midY := (src.Y + tgt.Y) / 2
		return []Point{
			{X: src.X, Y: midY},
			{X: tgt.X, Y: midY},
		}
	}

	if absDy < minBend && absDx >= minBend {
		midX := (src.X + tgt.X) / 2
		return []Point{
			{X: midX, Y: src.Y},
			{X: midX, Y: tgt.Y},
		}
	}

	if math.Abs(absDx-absDy) < minBend {
		return nil
	}

	if absDx >= absDy {
		return []Point{{X: src.X + math.Copysign(absDy, dx), Y: tgt.Y}}
	}
	return []Point{{X: tgt.X, Y: src.Y + math.Copysign(absDx, dy)}}
}

func topLevelSize(id string, groupSizes map[string][2]float64) (float64, float64) {
	if sz, ok := groupSizes[id]; ok {
		return sz[0], sz[1]
	}
	return nodeW, nodeH
}

func topLevelRect(id string, positions map[string][2]float64, groupSizes map[string][2]float64, padding float64) rectBox {
	pos := positions[id]
	w, h := topLevelSize(id, groupSizes)
	return rectBox{
		Left:   pos[0] - w/2 - padding,
		Right:  pos[0] + w/2 + padding,
		Top:    pos[1] - h/2 - padding,
		Bottom: pos[1] + h/2 + padding,
	}
}

func pointInRect(p Point, box rectBox) bool {
	return p.X >= box.Left && p.X <= box.Right && p.Y >= box.Top && p.Y <= box.Bottom
}

func orientation(a, b, c Point) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func onSegment(a, b, p Point) bool {
	const eps = 1e-6
	return p.X >= math.Min(a.X, b.X)-eps &&
		p.X <= math.Max(a.X, b.X)+eps &&
		p.Y >= math.Min(a.Y, b.Y)-eps &&
		p.Y <= math.Max(a.Y, b.Y)+eps
}

func segmentsIntersect(a1, a2, b1, b2 Point) bool {
	const eps = 1e-6
	o1 := orientation(a1, a2, b1)
	o2 := orientation(a1, a2, b2)
	o3 := orientation(b1, b2, a1)
	o4 := orientation(b1, b2, a2)

	if (o1 > eps && o2 < -eps || o1 < -eps && o2 > eps) &&
		(o3 > eps && o4 < -eps || o3 < -eps && o4 > eps) {
		return true
	}
	if math.Abs(o1) <= eps && onSegment(a1, a2, b1) {
		return true
	}
	if math.Abs(o2) <= eps && onSegment(a1, a2, b2) {
		return true
	}
	if math.Abs(o3) <= eps && onSegment(b1, b2, a1) {
		return true
	}
	if math.Abs(o4) <= eps && onSegment(b1, b2, a2) {
		return true
	}
	return false
}

func segmentIntersectsRect(a, b Point, box rectBox) bool {
	if pointInRect(a, box) || pointInRect(b, box) {
		return true
	}

	topLeft := Point{X: box.Left, Y: box.Top}
	topRight := Point{X: box.Right, Y: box.Top}
	bottomRight := Point{X: box.Right, Y: box.Bottom}
	bottomLeft := Point{X: box.Left, Y: box.Bottom}

	return segmentsIntersect(a, b, topLeft, topRight) ||
		segmentsIntersect(a, b, topRight, bottomRight) ||
		segmentsIntersect(a, b, bottomRight, bottomLeft) ||
		segmentsIntersect(a, b, bottomLeft, topLeft)
}

func polylineIntersectsRect(points []Point, box rectBox) bool {
	for i := 0; i < len(points)-1; i++ {
		if segmentIntersectsRect(points[i], points[i+1], box) {
			return true
		}
	}
	return false
}

func pointsApproxEqual(a, b Point) bool {
	const eps = 1e-6
	return math.Abs(a.X-b.X) <= eps && math.Abs(a.Y-b.Y) <= eps
}

func actualNodePoint(
	nodeID string,
	positions map[string][2]float64,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) (Point, bool) {
	n, ok := nodeMap[nodeID]
	if !ok {
		return Point{}, false
	}
	if n.ParentID == nil {
		pos, ok := positions[nodeID]
		if !ok {
			return Point{}, false
		}
		return Point{X: pos[0], Y: pos[1]}, true
	}
	topID := resolveTopLevel(nodeID, nodeMap)
	parentPos, ok := positions[topID]
	if !ok {
		return Point{}, false
	}
	rel := childRelPos[nodeID]
	return Point{X: parentPos[0] + rel[0], Y: parentPos[1] + rel[1]}, true
}

func buildPredictedRoutes(
	edges []model.Edge,
	positions map[string][2]float64,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) []predictedRoute {
	routes := make([]predictedRoute, 0, len(edges))
	for _, e := range edges {
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		srcTop := resolveTopLevel(e.SourceID, nodeMap)
		tgtTop := resolveTopLevel(e.TargetID, nodeMap)
		if srcTop == tgtTop {
			continue
		}
		src, ok := actualNodePoint(e.SourceID, positions, nodeMap, childRelPos)
		if !ok {
			continue
		}
		tgt, ok := actualNodePoint(e.TargetID, positions, nodeMap, childRelPos)
		if !ok {
			continue
		}
		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		routes = append(routes, predictedRoute{
			SrcTop: srcTop,
			TgtTop: tgtTop,
			Points: points,
		})
	}
	return routes
}

func interpolateYAtX(x1, y1, x2, y2, x float64) float64 {
	if x2 == x1 {
		return (y1 + y2) / 2
	}
	t := (x - x1) / (x2 - x1)
	return y1 + (y2-y1)*t
}

func edgeYRangeAcrossRect(x1, y1, x2, y2, left, right float64) (float64, float64, bool) {
	if x1 == x2 {
		if x1 < left || x1 > right {
			return 0, 0, false
		}
		if y1 < y2 {
			return y1, y2, true
		}
		return y2, y1, true
	}
	segLeft := math.Max(left, math.Min(x1, x2))
	segRight := math.Min(right, math.Max(x1, x2))
	if segLeft > segRight {
		return 0, 0, false
	}
	yLeft := interpolateYAtX(x1, y1, x2, y2, segLeft)
	yRight := interpolateYAtX(x1, y1, x2, y2, segRight)
	if yLeft < yRight {
		return yLeft, yRight, true
	}
	return yRight, yLeft, true
}

func enforceLayerSpacing(ids []string, yCoords map[string]float64, groupSizes map[string][2]float64) {
	if len(ids) <= 1 {
		return
	}
	for i := 1; i < len(ids); i++ {
		prevH := nodeH
		if sz, ok := groupSizes[ids[i-1]]; ok {
			prevH = sz[1]
		}
		currH := nodeH
		if sz, ok := groupSizes[ids[i]]; ok {
			currH = sz[1]
		}
		gap := spacingBetween(ids[i-1], ids[i], groupSizes)
		minY := yCoords[ids[i-1]] + prevH/2 + gap + currH/2
		if yCoords[ids[i]] < minY {
			yCoords[ids[i]] = minY
		}
	}
}

func mergeVerticalBands(bands []verticalBand) []verticalBand {
	if len(bands) == 0 {
		return nil
	}
	sort.Slice(bands, func(i, j int) bool { return bands[i].Top < bands[j].Top })
	merged := []verticalBand{bands[0]}
	for _, band := range bands[1:] {
		last := &merged[len(merged)-1]
		if band.Top <= last.Bottom {
			if band.Bottom > last.Bottom {
				last.Bottom = band.Bottom
			}
			continue
		}
		merged = append(merged, band)
	}
	return merged
}

func nodeHeightForLayout(id string, groupSizes map[string][2]float64) float64 {
	if sz, ok := groupSizes[id]; ok {
		return sz[1]
	}
	return nodeH
}

func computeLayerXPositions(layerNodes map[int][]string, maxLayer int, groupSizes map[string][2]float64) map[int]float64 {
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

	layerX := make(map[int]float64)
	cumX := 0.0
	for l := 0; l <= maxLayer; l++ {
		layerX[l] = cumX + snapToGrid(layerMaxW[l]/2)
		cumX += snapToGrid(layerMaxW[l]) + math.Ceil(layerGapX/gridSize)*gridSize
	}
	return layerX
}

func weightedSatelliteAnchor(
	sources map[string]int,
	positions map[string][2]float64,
	fallbackX, fallbackY float64,
) (float64, float64) {
	if len(sources) == 0 {
		return fallbackX, fallbackY
	}
	sumX, sumY, totalW := 0.0, 0.0, 0
	for srcG, w := range sources {
		if p, ok := positions[srcG]; ok {
			sumX += p[0] * float64(w)
			sumY += p[1] * float64(w)
			totalW += w
		}
	}
	if totalW == 0 {
		return fallbackX, fallbackY
	}
	return sumX / float64(totalW), sumY / float64(totalW)
}

func buildTopLevelPositions(layerNodes map[int][]string, maxLayer int, layerX map[int]float64, yCoords map[string]float64) map[string][2]float64 {
	positions := make(map[string][2]float64)
	for l := 0; l <= maxLayer; l++ {
		x := snapToGrid(layerX[l])
		for _, id := range layerNodes[l] {
			positions[id] = [2]float64{x, snapToGrid(yCoords[id])}
		}
	}
	return positions
}

func spacingBetween(prevID, nextID string, groupSizes map[string][2]float64) float64 {
	_, prevIsGroup := groupSizes[prevID]
	_, nextIsGroup := groupSizes[nextID]
	if prevIsGroup || nextIsGroup {
		return groupLaneGapY
	}
	return layerGapY
}

func centerBlockedByBands(center, height float64, bands []verticalBand) bool {
	top := center - height/2
	bottom := center + height/2
	for _, band := range bands {
		if math.Min(bottom, band.Bottom) > math.Max(top, band.Top) {
			return true
		}
	}
	return false
}

func chooseNearestValidCenter(ideal, minY, maxY, height float64, bands []verticalBand) float64 {
	candidates := []float64{}
	if !math.IsInf(minY, 1) && !math.IsInf(maxY, -1) && minY > maxY {
		return ideal
	}

	clamped := ideal
	if !math.IsInf(minY, -1) && clamped < minY {
		clamped = minY
	}
	if !math.IsInf(maxY, 1) && clamped > maxY {
		clamped = maxY
	}
	candidates = append(candidates, clamped)
	for _, band := range bands {
		candidates = append(candidates, band.Top-height/2, band.Bottom+height/2)
	}

	best := clamped
	bestCost := math.MaxFloat64
	found := false
	for _, candidate := range candidates {
		if !math.IsInf(minY, -1) && candidate < minY {
			continue
		}
		if !math.IsInf(maxY, 1) && candidate > maxY {
			continue
		}
		if centerBlockedByBands(candidate, height, bands) {
			continue
		}
		cost := math.Abs(candidate - ideal)
		if !found || cost < bestCost {
			best = candidate
			bestCost = cost
			found = true
		}
	}
	if found {
		return best
	}
	return clamped
}

func barycenterIdealY(
	id string,
	fallback float64,
	adj map[string]map[string]bool,
	layers map[string]int,
	yCoords map[string]float64,
) float64 {
	neighbors, ok := adj[id]
	if !ok || len(neighbors) == 0 {
		return fallback
	}

	sum := 0.0
	totalWeight := 0.0
	idLayer := layers[id]
	for neighbor := range neighbors {
		neighborY, ok := yCoords[neighbor]
		if !ok {
			continue
		}
		layerGap := math.Abs(float64(layers[neighbor] - idLayer))
		if layerGap == 0 {
			continue
		}
		weight := 1.0 / layerGap
		sum += neighborY * weight
		totalWeight += weight
	}
	if totalWeight == 0 {
		return fallback
	}
	return fallback*0.75 + (sum/totalWeight)*0.25
}

func routedBandsAcrossRect(src, tgt Point, left, right float64) []verticalBand {
	points := []Point{src}
	points = append(points, compute8DirWaypoints(src, tgt)...)
	points = append(points, tgt)

	bands := make([]verticalBand, 0, len(points))
	for i := 0; i < len(points)-1; i++ {
		yTop, yBottom, ok := edgeYRangeAcrossRect(
			points[i].X,
			points[i].Y,
			points[i+1].X,
			points[i+1].Y,
			left,
			right,
		)
		if !ok {
			continue
		}
		bands = append(bands, verticalBand{
			Top:    yTop - layerGapY/2,
			Bottom: yBottom + layerGapY/2,
		})
	}
	return mergeVerticalBands(bands)
}

func corridorBandsForNode(
	id string,
	layer int,
	layerX map[int]float64,
	yCoords map[string]float64,
	layers map[string]int,
	groupSizes map[string][2]float64,
	spans []longEdgeSpan,
) []verticalBand {
	width := nodeW
	if sz, ok := groupSizes[id]; ok {
		width = sz[0]
	}
	left := layerX[layer] - width/2
	right := layerX[layer] + width/2
	bands := make([]verticalBand, 0, len(spans))
	for _, span := range spans {
		if id == span.Src || id == span.Tgt {
			continue
		}
		src, tgt := span.Src, span.Tgt
		srcL, tgtL := layers[src], layers[tgt]
		if srcL > tgtL {
			src, tgt = tgt, src
			srcL, tgtL = tgtL, srcL
		}
		if layer < srcL || layer > tgtL {
			continue
		}
		srcPt := Point{X: layerX[srcL] + span.SrcDX, Y: yCoords[src] + span.SrcDY}
		tgtPt := Point{X: layerX[tgtL] + span.TgtDX, Y: yCoords[tgt] + span.TgtDY}
		bands = append(bands, routedBandsAcrossRect(srcPt, tgtPt, left, right)...)
	}
	return mergeVerticalBands(bands)
}

func copyPositionsMap(positions map[string][2]float64) map[string][2]float64 {
	cloned := make(map[string][2]float64, len(positions))
	for id, pos := range positions {
		cloned[id] = pos
	}
	return cloned
}

func sortedPositionIDs(positions map[string][2]float64) []string {
	ids := make([]string, 0, len(positions))
	for id := range positions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func countTopLevelBoxOverlaps(positions map[string][2]float64, groupSizes map[string][2]float64) int {
	ids := sortedPositionIDs(positions)
	overlaps := 0
	for i := 0; i < len(ids); i++ {
		a := topLevelRect(ids[i], positions, groupSizes, 8)
		for j := i + 1; j < len(ids); j++ {
			b := topLevelRect(ids[j], positions, groupSizes, 8)
			if math.Min(a.Right, b.Right) > math.Max(a.Left, b.Left) &&
				math.Min(a.Bottom, b.Bottom) > math.Max(a.Top, b.Top) {
				overlaps++
			}
		}
	}
	return overlaps
}

func countPredictedRouteBoxIntersections(
	edges []model.Edge,
	positions map[string][2]float64,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) (int, map[string]int) {
	total := 0
	byNode := make(map[string]int)
	boxIDs := sortedPositionIDs(positions)
	for _, route := range buildPredictedRoutes(edges, positions, nodeMap, childRelPos) {
		for _, id := range boxIDs {
			if id == route.SrcTop || id == route.TgtTop {
				continue
			}
			box := topLevelRect(id, positions, groupSizes, 8)
			if polylineIntersectsRect(route.Points, box) {
				total++
				byNode[id]++
			}
		}
	}

	return total, byNode
}

func countPredictedRouteCrossings(
	edges []model.Edge,
	positions map[string][2]float64,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) (int, map[string]int) {
	total := 0
	byNode := make(map[string]int)
	routes := buildPredictedRoutes(edges, positions, nodeMap, childRelPos)

	for i := 0; i < len(routes); i++ {
		a := routes[i]
		for j := i + 1; j < len(routes); j++ {
			b := routes[j]
			if a.SrcTop == b.SrcTop || a.SrcTop == b.TgtTop || a.TgtTop == b.SrcTop || a.TgtTop == b.TgtTop {
				continue
			}
			crossed := false
			for ai := 0; ai < len(a.Points)-1 && !crossed; ai++ {
				for bi := 0; bi < len(b.Points)-1; bi++ {
					a1, a2 := a.Points[ai], a.Points[ai+1]
					b1, b2 := b.Points[bi], b.Points[bi+1]
					if pointsApproxEqual(a1, b1) || pointsApproxEqual(a1, b2) || pointsApproxEqual(a2, b1) || pointsApproxEqual(a2, b2) {
						continue
					}
					if segmentsIntersect(a1, a2, b1, b2) {
						crossed = true
						break
					}
				}
			}
			if crossed {
				total++
				byNode[a.SrcTop]++
				byNode[a.TgtTop]++
				byNode[b.SrcTop]++
				byNode[b.TgtTop]++
			}
		}
	}

	return total, byNode
}

func layoutViolationCost(
	positions map[string][2]float64,
	basePositions map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) (float64, map[string]int) {
	boxOverlaps := countTopLevelBoxOverlaps(positions, groupSizes)
	intersections, byNode := countPredictedRouteBoxIntersections(edges, positions, nodeMap, childRelPos, groupSizes)
	crossings, crossingNodes := countPredictedRouteCrossings(edges, positions, nodeMap, childRelPos)
	for id, count := range crossingNodes {
		byNode[id] += count
	}
	displacement := 0.0
	for id, pos := range positions {
		base := basePositions[id]
		displacement += math.Abs(pos[0]-base[0]) + math.Abs(pos[1]-base[1])
	}
	cost := float64(boxOverlaps)*1_000_000_000 + float64(intersections)*1_000_000 + float64(crossings)*50_000 + displacement*500
	return cost, byNode
}

func uniqueSortedColumns(positions map[string][2]float64) []float64 {
	seen := make(map[float64]bool, len(positions))
	cols := make([]float64, 0, len(positions))
	for _, pos := range positions {
		x := pos[0]
		if seen[x] {
			continue
		}
		seen[x] = true
		cols = append(cols, x)
	}
	sort.Float64s(cols)
	return cols
}

func adjacentColumnCandidates(currentX float64, columns []float64) []float64 {
	idx := sort.SearchFloat64s(columns, currentX)
	out := make([]float64, 0, 2)
	if idx > 0 {
		out = append(out, columns[idx-1])
	}
	if idx < len(columns) && columns[idx] == currentX {
		if idx+1 < len(columns) {
			out = append(out, columns[idx+1])
		}
	} else if idx < len(columns) {
		out = append(out, columns[idx])
	}
	return out
}

func cleanupCandidatePositions(id string, current, base [2]float64, positions map[string][2]float64, groupSizes map[string][2]float64) [][2]float64 {
	w, h := topLevelSize(id, groupSizes)
	dx := snapToGrid(math.Max(nodeW, w/2) + layerGapX)
	dy := snapToGrid(math.Max(nodeH, h/2) + layerGapY)
	raw := [][2]float64{
		current,
		{current[0], current[1] - dy},
		{current[0], current[1] + dy},
		{current[0] - dx, current[1]},
		{current[0] + dx, current[1]},
		{current[0] - dx, current[1] - dy},
		{current[0] - dx, current[1] + dy},
		{current[0] + dx, current[1] - dy},
		{current[0] + dx, current[1] + dy},
	}
	columns := uniqueSortedColumns(positions)
	adjacentXs := adjacentColumnCandidates(current[0], columns)
	maxXShift := dx
	for _, x := range adjacentXs {
		if shift := math.Abs(x - base[0]); shift > maxXShift {
			maxXShift = shift
		}
		raw = append(raw,
			[2]float64{x, current[1]},
			[2]float64{x, current[1] - dy},
			[2]float64{x, current[1] + dy},
		)
	}

	candidates := make([][2]float64, 0, len(raw))
	seen := make(map[[2]float64]bool, len(raw))
	for _, candidate := range raw {
		candidate[0] = snapToGrid(candidate[0])
		candidate[1] = snapToGrid(candidate[1])
		if math.Abs(candidate[0]-base[0]) > maxXShift+1 || math.Abs(candidate[1]-base[1]) > dy+1 {
			continue
		}
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}
	return candidates
}

func cleanupRouteBoxIntersections(
	positions map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) {
	if len(positions) <= 1 {
		return
	}

	basePositions := copyPositionsMap(positions)
	currentCost, offenders := layoutViolationCost(positions, basePositions, edges, nodeMap, childRelPos, groupSizes)
	if currentCost < 1_000_000 {
		return
	}

	for iter := 0; iter < 4; iter++ {
		order := sortedPositionIDs(positions)
		sort.SliceStable(order, func(i, j int) bool {
			left := offenders[order[i]]
			right := offenders[order[j]]
			if left != right {
				return left > right
			}
			return order[i] < order[j]
		})

		moved := false
		for _, id := range order {
			if offenders[id] == 0 {
				continue
			}

			original := positions[id]
			best := original
			bestCost := currentCost
			for _, candidate := range cleanupCandidatePositions(id, original, basePositions[id], positions, groupSizes) {
				if candidate == original {
					continue
				}
				positions[id] = candidate
				candidateCost, _ := layoutViolationCost(positions, basePositions, edges, nodeMap, childRelPos, groupSizes)
				if candidateCost < bestCost {
					best = candidate
					bestCost = candidateCost
				}
			}
			positions[id] = best
			if best != original {
				moved = true
				currentCost = bestCost
			}
		}

		if !moved {
			return
		}
		currentCost, offenders = layoutViolationCost(positions, basePositions, edges, nodeMap, childRelPos, groupSizes)
		if currentCost < 1_000_000 {
			return
		}
	}
}

func clampCenterShift(target, ideal, limit float64) float64 {
	if target > ideal+limit {
		return ideal + limit
	}
	if target < ideal-limit {
		return ideal - limit
	}
	return target
}

func repackLayerAgainstCorridors(
	layer int,
	ids []string,
	layerX map[int]float64,
	yCoords map[string]float64,
	groupSizes map[string][2]float64,
	layers map[string]int,
	adj map[string]map[string]bool,
	spans []longEdgeSpan,
	bandTargets map[string]float64,
) bool {
	if len(ids) == 0 || len(spans) == 0 {
		return false
	}

	before := make(map[string]float64, len(ids))
	ideal := make(map[string]float64, len(ids))
	anchor := make(map[string]float64, len(ids))
	shiftLimit := make(map[string]float64, len(ids))
	for _, id := range ids {
		before[id] = yCoords[id]
		base := yCoords[id]
		anchor[id] = yCoords[id]
		shiftLimit[id] = 240
		if target, ok := bandTargets[id]; ok {
			base = target
			anchor[id] = target
			shiftLimit[id] = 160
		}
		ideal[id] = barycenterIdealY(id, base, adj, layers, yCoords)
	}

	for i, id := range ids {
		minY := math.Inf(-1)
		if i > 0 {
			prevID := ids[i-1]
			minY = yCoords[prevID] + nodeHeightForLayout(prevID, groupSizes)/2 +
				spacingBetween(prevID, id, groupSizes) +
				nodeHeightForLayout(id, groupSizes)/2
		}
		target := chooseNearestValidCenter(
			ideal[id],
			minY,
			math.Inf(1),
			nodeHeightForLayout(id, groupSizes),
			corridorBandsForNode(id, layer, layerX, yCoords, layers, groupSizes, spans),
		)
		yCoords[id] = clampCenterShift(target, anchor[id], shiftLimit[id])
	}

	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		maxY := math.Inf(1)
		if i < len(ids)-1 {
			nextID := ids[i+1]
			maxY = yCoords[nextID] - nodeHeightForLayout(nextID, groupSizes)/2 -
				spacingBetween(id, nextID, groupSizes) -
				nodeHeightForLayout(id, groupSizes)/2
		}
		target := chooseNearestValidCenter(
			ideal[id],
			math.Inf(-1),
			maxY,
			nodeHeightForLayout(id, groupSizes),
			corridorBandsForNode(id, layer, layerX, yCoords, layers, groupSizes, spans),
		)
		yCoords[id] = clampCenterShift(target, anchor[id], shiftLimit[id])
	}

	enforceLayerSpacing(ids, yCoords, groupSizes)

	moved := false
	for _, id := range ids {
		if math.Abs(yCoords[id]-before[id]) > 0.5 {
			moved = true
		}
	}
	return moved
}

func repackLayersAgainstEdgeCorridors(
	layerNodes map[int][]string,
	maxLayer int,
	layerX map[int]float64,
	yCoords map[string]float64,
	groupSizes map[string][2]float64,
	layers map[string]int,
	adj map[string]map[string]bool,
	spans []longEdgeSpan,
	bandTargets map[string]float64,
) {
	if len(spans) == 0 {
		return
	}

	for iter := 0; iter < 6; iter++ {
		moved := false
		for l := 0; l <= maxLayer; l++ {
			if repackLayerAgainstCorridors(l, layerNodes[l], layerX, yCoords, groupSizes, layers, adj, spans, bandTargets) {
				moved = true
			}
		}
		if !moved {
			return
		}
	}
}

func buildInEdges(topLevel []string, outEdges map[string][]string) map[string][]string {
	inEdges := make(map[string][]string, len(topLevel))
	for _, id := range topLevel {
		inEdges[id] = []string{}
	}
	for src, targets := range outEdges {
		for _, tgt := range targets {
			inEdges[tgt] = append(inEdges[tgt], src)
		}
	}
	return inEdges
}

func isGroupNode(id string, groupSizes map[string][2]float64) bool {
	_, ok := groupSizes[id]
	return ok
}

func topLevelBandPriority(id string, outEdges, inEdges map[string][]string, groupSizes map[string][2]float64) int {
	inCount := len(inEdges[id])
	outCount := len(outEdges[id])
	isGroup := isGroupNode(id, groupSizes)

	switch {
	case inCount > 0 && outCount > 0 && isGroup:
		return 0
	case inCount > 0 && outCount > 0:
		return 1
	case (inCount > 0 || outCount > 0) && isGroup:
		return 2
	case inCount > 0 || outCount > 0:
		return 3
	default:
		return 4
	}
}

func orderedBandCandidates(id string, outEdges, inEdges map[string][]string) []int {
	inCount := len(inEdges[id])
	outCount := len(outEdges[id])
	switch {
	case inCount > 0 && outCount > 0:
		return []int{0, -1, 1, -2, 2, -3, 3}
	case inCount == 0 && outCount > 0:
		return []int{-1, 1, 0, -2, 2, -3, 3}
	case outCount == 0 && inCount > 0:
		return []int{-1, 1, 0, -2, 2, -3, 3}
	default:
		return []int{0, -1, 1, -2, 2}
	}
}

func scoreBandCandidate(
	id string,
	band int,
	assigned map[string]int,
	topLevel []string,
	layers map[string]int,
	outEdges, inEdges map[string][]string,
	groupSizes map[string][2]float64,
) float64 {
	score := 0.0
	inCount := len(inEdges[id])
	outCount := len(outEdges[id])
	absBand := math.Abs(float64(band))

	switch {
	case inCount > 0 && outCount > 0:
		score += absBand * 120
	default:
		if band == 0 {
			score += 180
		}
		score += math.Abs(absBand-1) * 65
	}

	if isGroupNode(id, groupSizes) {
		score += absBand * 35
	}

	for other, otherBand := range assigned {
		layerGap := math.Abs(float64(layers[id] - layers[other]))
		bandGap := math.Abs(float64(band - otherBand))
		if layerGap == 0 {
			if bandGap < 0.5 {
				score += 5000
			} else {
				score += 400 / bandGap
			}
			continue
		}
		if bandGap < 0.5 {
			score += 60 / layerGap
		}
	}

	neighbors := append([]string{}, outEdges[id]...)
	neighbors = append(neighbors, inEdges[id]...)
	for _, neighbor := range neighbors {
		neighborBand, ok := assigned[neighbor]
		if !ok {
			continue
		}
		score += math.Abs(float64(band-neighborBand)) * 90

		idLayer := layers[id]
		neighborLayer := layers[neighbor]
		startBand := band
		endBand := neighborBand
		if idLayer > neighborLayer {
			idLayer, neighborLayer = neighborLayer, idLayer
			startBand, endBand = endBand, startBand
		}
		if idLayer == neighborLayer {
			continue
		}
		for other, otherBand := range assigned {
			otherLayer := layers[other]
			if other == neighbor || otherLayer <= idLayer || otherLayer >= neighborLayer {
				continue
			}
			t := float64(otherLayer-idLayer) / float64(neighborLayer-idLayer)
			interpBand := float64(startBand) + float64(endBand-startBand)*t
			diff := math.Abs(interpBand - float64(otherBand))
			switch {
			case diff < 0.6:
				score += 900
			case diff < 1.2:
				score += 220
			}
		}
	}

	return score
}

func computeTopLevelBandTargets(
	topLevel []string,
	layers map[string]int,
	outEdges map[string][]string,
	groupSizes map[string][2]float64,
) map[string]float64 {
	if len(topLevel) == 0 {
		return nil
	}

	inEdges := buildInEdges(topLevel, outEdges)
	ordered := append([]string(nil), topLevel...)
	sort.Slice(ordered, func(i, j int) bool {
		pi := topLevelBandPriority(ordered[i], outEdges, inEdges, groupSizes)
		pj := topLevelBandPriority(ordered[j], outEdges, inEdges, groupSizes)
		if pi != pj {
			return pi < pj
		}
		di := len(outEdges[ordered[i]]) + len(inEdges[ordered[i]])
		dj := len(outEdges[ordered[j]]) + len(inEdges[ordered[j]])
		if di != dj {
			return di > dj
		}
		hi := nodeHeightForLayout(ordered[i], groupSizes)
		hj := nodeHeightForLayout(ordered[j], groupSizes)
		if hi != hj {
			return hi > hj
		}
		return ordered[i] < ordered[j]
	})

	assigned := make(map[string]int, len(topLevel))
	for _, id := range ordered {
		bestBand := 0
		bestScore := math.MaxFloat64
		for _, band := range orderedBandCandidates(id, outEdges, inEdges) {
			score := scoreBandCandidate(id, band, assigned, topLevel, layers, outEdges, inEdges, groupSizes)
			if score < bestScore {
				bestScore = score
				bestBand = band
			}
		}
		assigned[id] = bestBand
	}

	maxHeight := nodeH
	for _, id := range topLevel {
		if h := nodeHeightForLayout(id, groupSizes); h > maxHeight {
			maxHeight = h
		}
	}
	pitch := snapToGrid(math.Max(120, maxHeight*0.45+groupGapY*0.35))
	targets := make(map[string]float64, len(topLevel))
	for _, id := range topLevel {
		targets[id] = float64(assigned[id]) * pitch
	}
	return targets
}

func filterAnchoredBandTargets(
	topLevel []string,
	bandTargets map[string]float64,
	outEdges map[string][]string,
	groupSizes map[string][2]float64,
) map[string]float64 {
	if len(bandTargets) == 0 {
		return nil
	}

	inEdges := buildInEdges(topLevel, outEdges)
	filtered := make(map[string]float64, len(bandTargets))
	for _, id := range topLevel {
		target, ok := bandTargets[id]
		if !ok {
			continue
		}
		if isGroupNode(id, groupSizes) && len(inEdges[id]) > 0 && len(outEdges[id]) > 0 {
			filtered[id] = target
		}
	}
	return filtered
}

func reorderLayerByDesiredY(ids []string, desiredY, currentY map[string]float64) {
	sort.SliceStable(ids, func(i, j int) bool {
		left := currentY[ids[i]]
		if target, ok := desiredY[ids[i]]; ok {
			left = target
		}
		right := currentY[ids[j]]
		if target, ok := desiredY[ids[j]]; ok {
			right = target
		}
		if left != right {
			return left < right
		}
		return ids[i] < ids[j]
	})
}

func relaxLayerTowardDesiredY(
	ids []string,
	yCoords map[string]float64,
	desiredY map[string]float64,
	groupSizes map[string][2]float64,
) {
	if len(ids) == 0 {
		return
	}

	placed := make([]float64, len(ids))
	for i, id := range ids {
		placed[i] = yCoords[id]
		if target, ok := desiredY[id]; ok {
			placed[i] = target
		}
	}

	for iter := 0; iter < 3; iter++ {
		for i := 1; i < len(ids); i++ {
			prevID, id := ids[i-1], ids[i]
			minY := placed[i-1] + nodeHeightForLayout(prevID, groupSizes)/2 +
				spacingBetween(prevID, id, groupSizes) +
				nodeHeightForLayout(id, groupSizes)/2
			if placed[i] < minY {
				placed[i] = minY
			}
		}
		for i := len(ids) - 2; i >= 0; i-- {
			id, nextID := ids[i], ids[i+1]
			maxY := placed[i+1] - nodeHeightForLayout(nextID, groupSizes)/2 -
				spacingBetween(id, nextID, groupSizes) -
				nodeHeightForLayout(id, groupSizes)/2
			if placed[i] > maxY {
				placed[i] = maxY
			}
		}
	}

	for i, id := range ids {
		yCoords[id] = placed[i]
	}
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

func refineLayerOrderByChildConnections(
	layerNodes map[int][]string,
	maxLayer int,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) {
	for l := 0; l <= maxLayer; l++ {
		ids := layerNodes[l]
		if len(ids) <= 1 {
			continue
		}
		changed := true
		for changed {
			changed = false
			for i := 0; i < len(ids)-1; i++ {
				u, v := ids[i], ids[i+1]
				uAvg, uCnt := childConnAvgRelY(u, edges, nodeMap, childRelPos)
				vAvg, vCnt := childConnAvgRelY(v, edges, nodeMap, childRelPos)
				if uCnt > 0 && vCnt > 0 && uAvg > vAvg {
					ids[i], ids[i+1] = v, u
					changed = true
				}
			}
		}
	}
}

func finalizeGroupLayoutsWithExternalPulls(
	nodes []model.Node,
	children map[string][]string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) {
	for _, n := range nodes {
		if n.Type != model.NodeTypeGroup {
			continue
		}
		groupPos, gpOk := groupPositions[n.ID]
		if !gpOk {
			continue
		}
		kids := children[n.ID]
		if len(kids) <= 1 {
			continue
		}

		pullMap := childExternalPulls(n.ID, groupPos, kids, edges, nodeMap, groupPositions)

		if w, h, ok := layoutChildrenRectangular(kids, edges, childRelPos, pullMap); ok {
			groupSizes[n.ID] = [2]float64{w, h}
			continue
		}

		// Bucket kids by their current relX (produced by the child-layout pass).
		// If any column holds more than one child, the layout reflects a real
		// BFS flow (leftmost column = roots, rightmost = sinks). In that case we
		// preserve column assignments and only reorder Y within each column so
		// BFS roots stay on the intake side of the group. For layouts where
		// every column holds a single child (grids, sparse rectangular pools),
		// we fall back to the global sort which lets external pulls swap kids
		// freely across columns.
		type colBucket struct {
			x     float64
			slots []float64
			kids  []string
		}
		colMap := make(map[float64]*colBucket)
		for _, kid := range kids {
			rel := childRelPos[kid]
			b, ok := colMap[rel[0]]
			if !ok {
				b = &colBucket{x: rel[0]}
				colMap[rel[0]] = b
			}
			b.slots = append(b.slots, rel[1])
			b.kids = append(b.kids, kid)
		}
		layered := false
		for _, b := range colMap {
			if len(b.kids) > 1 {
				layered = true
				break
			}
		}

		if layered {
			for _, b := range colMap {
				sort.Float64s(b.slots)
				sort.SliceStable(b.kids, func(i, j int) bool {
					ki, kj := b.kids[i], b.kids[j]
					relI, relJ := childRelPos[ki], childRelPos[kj]
					pullI, pullJ := pullMap[ki], pullMap[kj]
					desiredYI := relI[1]
					if pullI.count > 0 {
						desiredYI = (relI[1] + pullI.avgY) / 2
					}
					desiredYJ := relJ[1]
					if pullJ.count > 0 {
						desiredYJ = (relJ[1] + pullJ.avgY) / 2
					}
					if desiredYI != desiredYJ {
						return desiredYI < desiredYJ
					}
					return relI[1] < relJ[1]
				})
				for i, kid := range b.kids {
					childRelPos[kid] = [2]float64{b.x, b.slots[i]}
				}
			}
		} else {
			slots := make([][2]float64, len(kids))
			for i, kid := range kids {
				slots[i] = childRelPos[kid]
			}
			sort.Slice(slots, func(i, j int) bool {
				if slots[i][0] != slots[j][0] {
					return slots[i][0] < slots[j][0]
				}
				return slots[i][1] < slots[j][1]
			})

			sortedKids := make([]string, len(kids))
			copy(sortedKids, kids)
			sort.SliceStable(sortedKids, func(i, j int) bool {
				relI, relJ := childRelPos[sortedKids[i]], childRelPos[sortedKids[j]]
				pullI, pullJ := pullMap[sortedKids[i]], pullMap[sortedKids[j]]

				desiredXI := relI[0]
				desiredYI := relI[1]
				if pullI.count > 0 {
					desiredXI = (relI[0] + pullI.avgX*2) / 3
					desiredYI = (relI[1] + pullI.avgY) / 2
				}

				desiredXJ := relJ[0]
				desiredYJ := relJ[1]
				if pullJ.count > 0 {
					desiredXJ = (relJ[0] + pullJ.avgX*2) / 3
					desiredYJ = (relJ[1] + pullJ.avgY) / 2
				}

				if desiredXI != desiredXJ {
					return desiredXI < desiredXJ
				}
				if desiredYI != desiredYJ {
					return desiredYI < desiredYJ
				}
				if relI[0] != relJ[0] {
					return relI[0] < relJ[0]
				}
				return relI[1] < relJ[1]
			})

			for i, kid := range sortedKids {
				if i < len(slots) {
					childRelPos[kid] = slots[i]
				}
			}
		}
	}
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

	// Detect "satellite" top-level nodes: non-group nodes whose resolved edges
	// all target groups (never other plain top-level nodes). These are visually
	// attached to a dominant group and should be placed adjacent to it rather
	// than allocated their own layer column, which tends to create awkward
	// gaps. If a node has edges to multiple groups but >50% go to one group,
	// we still classify it as a "cross-group satellite" of the dominant group
	// and place it BELOW that group, biasing its X toward the weighted mean of
	// its source groups so far-left sources get a shorter connection.
	satelliteDominant := make(map[string]string)
	satelliteSide := make(map[string]int)                // -1 = left, +1 = right, 0 = below
	satelliteSources := make(map[string]map[string]int)  // cross-group only: groupID → edge count
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

	layerNodes := make(map[int][]string)
	maxLayer = 0
	for _, id := range topLevel {
		l := layers[id]
		layerNodes[l] = append(layerNodes[l], id)
		if l > maxLayer {
			maxLayer = l
		}
	}

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
	finalizeGroupLayoutsWithExternalPulls(nodes, children, edges, nodeMap, coarsePositions, childRelPos, groupSizes)
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

	// Final hard cleanup: if the predicted route between top-level elements still
	// crosses unrelated group/node boxes, locally move the offending boxes.
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

	if w, h, ok := layoutChildrenRectangular(childIDs, edges, relPos, nil); ok {
		return w, h
	}
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

type rectSlot struct {
	X, Y     float64
	NX, NY   float64
	Priority float64
}

type childRectMetric struct {
	internalDegree int
	internalIn     int
	internalOut    int
	externalDegree int
	externalIn     int
	externalOut    int
	pull           childExternalPull
}

func shouldUseRectangularChildLayout(count int) bool {
	return count >= 4 && count <= 8
}

func rectangularSlotPool(count int) []rectSlot {
	return rectangularSlotPools(count)[0]
}

// rectangularSlotPools returns candidate slot arrangements for a given child
// count. `layoutChildrenRectangular` evaluates all candidates and picks the
// one with the lowest total assignment cost, so each candidate should be a
// geometrically-coherent rectangle (2x3, 3x2, 3x3, etc.) instead of a single
// pool that allows mixing orthogonal midpoints.
func rectangularSlotPools(count int) [][]rectSlot {
	offsetY := (groupPadTop - groupPadBot) / 2
	if count <= 4 {
		return [][]rectSlot{{
			{X: -rectCompactCornerX, Y: -rectCompactCornerY + offsetY, NX: -1, NY: -1, Priority: 0},
			{X: rectCompactCornerX, Y: -rectCompactCornerY + offsetY, NX: 1, NY: -1, Priority: 0},
			{X: -rectCompactCornerX, Y: rectCompactCornerY + offsetY, NX: -1, NY: 1, Priority: 0},
			{X: rectCompactCornerX, Y: rectCompactCornerY + offsetY, NX: 1, NY: 1, Priority: 0},
		}}
	}

	corners := []rectSlot{
		{X: -rectCompactSideX, Y: -rectCompactSideY + offsetY, NX: -1, NY: -1, Priority: 0},
		{X: rectCompactSideX, Y: -rectCompactSideY + offsetY, NX: 1, NY: -1, Priority: 0},
		{X: -rectCompactSideX, Y: rectCompactSideY + offsetY, NX: -1, NY: 1, Priority: 0},
		{X: rectCompactSideX, Y: rectCompactSideY + offsetY, NX: 1, NY: 1, Priority: 0},
	}
	topMid := rectSlot{X: 0, Y: -rectCompactSideY + offsetY, NX: 0, NY: -1, Priority: 8}
	botMid := rectSlot{X: 0, Y: rectCompactSideY + offsetY, NX: 0, NY: 1, Priority: 8}
	leftMid := rectSlot{X: -rectCompactSideX, Y: offsetY, NX: -1, NY: 0, Priority: 8}
	rightMid := rectSlot{X: rectCompactSideX, Y: offsetY, NX: 1, NY: 0, Priority: 8}
	center := rectSlot{X: 0, Y: offsetY, NX: 0, NY: 0, Priority: 18}

	if count == 5 {
		// 5 children: one row gets 3 nodes via a single midpoint. Keep the
		// midpoint on the bottom so trailing children sit beneath siblings,
		// matching the previous behavior before this refactor.
		return [][]rectSlot{
			append(append([]rectSlot(nil), corners...),
				rectSlot{X: 0, Y: rectCompactSideY + offsetY, NX: 0, NY: 1, Priority: 6},
			),
			append(append([]rectSlot(nil), corners...),
				rectSlot{X: 0, Y: -rectCompactSideY + offsetY, NX: 0, NY: -1, Priority: 6},
			),
			append(append([]rectSlot(nil), corners...),
				rectSlot{X: -rectCompactSideX, Y: offsetY, NX: -1, NY: 0, Priority: 6},
			),
			append(append([]rectSlot(nil), corners...),
				rectSlot{X: rectCompactSideX, Y: offsetY, NX: 1, NY: 0, Priority: 6},
			),
		}
	}

	if count == 6 {
		// 2 rows of 3 (wide) or 3 rows of 2 (tall). Each candidate keeps
		// its paired midpoints on opposite sides of the group center.
		wide := append(append([]rectSlot(nil), corners...), topMid, botMid)
		tall := append(append([]rectSlot(nil), corners...), leftMid, rightMid)
		return [][]rectSlot{wide, tall}
	}

	if count == 7 {
		// 7 = corners + one full midpoint pair + one lone midpoint.
		// The lone midpoint must be orthogonal to the pair so the result
		// still reads as a rectangle with an extra center-line node.
		h1 := append(append([]rectSlot(nil), corners...), topMid, botMid, leftMid)
		h2 := append(append([]rectSlot(nil), corners...), topMid, botMid, rightMid)
		v1 := append(append([]rectSlot(nil), corners...), leftMid, rightMid, topMid)
		v2 := append(append([]rectSlot(nil), corners...), leftMid, rightMid, botMid)
		return [][]rectSlot{h1, h2, v1, v2}
	}

	// 8 children = 4 corners + 4 midpoints (all edge slots in use).
	// 9 children = above + center.
	pool := append(append([]rectSlot(nil), corners...), topMid, botMid, leftMid, rightMid)
	if count > 8 {
		pool = append(pool, center)
	}
	return [][]rectSlot{pool}
}

func computeChildRectMetrics(
	childIDs []string,
	edges []model.Edge,
	pulls map[string]childExternalPull,
) map[string]childRectMetric {
	childSet := make(map[string]bool, len(childIDs))
	metrics := make(map[string]childRectMetric, len(childIDs))
	for _, id := range childIDs {
		childSet[id] = true
		metrics[id] = childRectMetric{}
	}

	for _, e := range edges {
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		srcIn := childSet[e.SourceID]
		tgtIn := childSet[e.TargetID]
		switch {
		case srcIn && tgtIn:
			m := metrics[e.SourceID]
			m.internalDegree++
			m.internalOut++
			metrics[e.SourceID] = m
			m = metrics[e.TargetID]
			m.internalDegree++
			m.internalIn++
			metrics[e.TargetID] = m
		case srcIn:
			m := metrics[e.SourceID]
			m.externalDegree++
			m.externalOut++
			metrics[e.SourceID] = m
		case tgtIn:
			m := metrics[e.TargetID]
			m.externalDegree++
			m.externalIn++
			metrics[e.TargetID] = m
		}
	}

	for id, pull := range pulls {
		if m, ok := metrics[id]; ok {
			m.pull = pull
			metrics[id] = m
		}
	}

	return metrics
}

func childRectUrgency(id string, metrics map[string]childRectMetric) (int, int) {
	m := metrics[id]
	return m.externalDegree + m.internalOut + m.externalOut, m.internalDegree + m.internalIn
}

func isCornerSlot(slot rectSlot) bool {
	return math.Abs(slot.NX) > 0.5 && math.Abs(slot.NY) > 0.5
}

func isBottomSlot(slot rectSlot) bool {
	return slot.NY > 0.5
}

func isTopSlot(slot rectSlot) bool {
	return slot.NY < -0.5
}

func isCenterXSlot(slot rectSlot) bool {
	return math.Abs(slot.NX) < 0.25
}

func scoreRectSlotAssignment(id string, slot rectSlot, metrics map[string]childRectMetric, count int) float64 {
	m := metrics[id]
	score := slot.Priority
	boundary := math.Max(math.Abs(slot.NX), math.Abs(slot.NY))
	extWeight := float64(m.externalDegree)
	intWeight := float64(m.internalDegree)
	isBroker := m.internalIn > 0 && m.internalOut > 0
	isSink := m.internalIn > m.internalOut
	isSource := m.internalOut > m.internalIn
	isEntryLike := isSource && m.externalIn > 0
	isBoundaryProvider := isSource && m.externalOut > 0 && !isEntryLike

	if extWeight > 0 {
		score += (1 - boundary) * 150 * extWeight
		if slot.NX == 0 && slot.NY == 0 {
			score += 120 * extWeight
		}

		pullMag := math.Hypot(m.pull.avgX, m.pull.avgY)
		slotMag := math.Hypot(slot.NX, slot.NY)
		if pullMag > 1 && slotMag > 0 {
			dot := (slot.NX*m.pull.avgX + slot.NY*m.pull.avgY) / (slotMag * pullMag)
			if dot > 1 {
				dot = 1
			}
			if dot < -1 {
				dot = -1
			}
			score += (1 - dot) * 90 * extWeight

			mainAxis := math.Max(math.Abs(m.pull.avgX), math.Abs(m.pull.avgY))
			minorAxis := math.Min(math.Abs(m.pull.avgX), math.Abs(m.pull.avgY))
			if mainAxis > 0 {
				ratio := minorAxis / mainAxis
				if ratio > 0.45 && (slot.NX == 0 || slot.NY == 0) {
					score += 18 * extWeight
				}
				if ratio < 0.25 && slot.NX != 0 && slot.NY != 0 {
					score += 10 * extWeight
				}
			}
		}
	}

	centerAffinity := intWeight*2 - extWeight*3
	if centerAffinity > 0 {
		score += boundary * centerAffinity * 18
	} else if centerAffinity < 0 {
		score += (1 - boundary) * math.Abs(centerAffinity) * 14
	}

	if m.externalDegree == 0 && m.internalDegree == 0 {
		score += boundary * 8
	}

	if isBroker {
		if isCornerSlot(slot) {
			score += 160
		}
		if isCenterXSlot(slot) && isBottomSlot(slot) {
			score -= 24
		} else if isCenterXSlot(slot) {
			score += 12
		}
	}

	if isEntryLike {
		if !isBottomSlot(slot) {
			score += 70
		}
		if slot.NX > 0.25 {
			score += 26
		}
	}

	if isBoundaryProvider {
		if !isTopSlot(slot) {
			score += 55
		}
	}

	if isSink {
		if !isBottomSlot(slot) {
			score += 42
		}
		if isCenterXSlot(slot) && count == 5 {
			score += 30
		}
	}

	return score
}

func bestRectangularSlotAssignment(
	kids []string,
	slots []rectSlot,
	metrics map[string]childRectMetric,
) map[string]rectSlot {
	assigned, _ := bestRectangularSlotAssignmentWithCost(kids, slots, metrics)
	return assigned
}

func bestRectangularSlotAssignmentWithCost(
	kids []string,
	slots []rectSlot,
	metrics map[string]childRectMetric,
) (map[string]rectSlot, float64) {
	type state struct {
		index int
		mask  int
	}

	memo := make(map[state]float64)
	choice := make(map[state]int)
	var solve func(index, mask int) float64
	solve = func(index, mask int) float64 {
		if index == len(kids) {
			return 0
		}
		key := state{index: index, mask: mask}
		if val, ok := memo[key]; ok {
			return val
		}

		best := math.MaxFloat64
		bestSlot := -1
		for slotIdx := 0; slotIdx < len(slots); slotIdx++ {
			if mask&(1<<slotIdx) != 0 {
				continue
			}
			cost := scoreRectSlotAssignment(kids[index], slots[slotIdx], metrics, len(kids)) + solve(index+1, mask|(1<<slotIdx))
			if cost < best {
				best = cost
				bestSlot = slotIdx
			}
		}

		memo[key] = best
		choice[key] = bestSlot
		return best
	}

	total := solve(0, 0)
	assigned := make(map[string]rectSlot, len(kids))
	mask := 0
	for index, id := range kids {
		key := state{index: index, mask: mask}
		slotIdx := choice[key]
		if slotIdx < 0 {
			continue
		}
		assigned[id] = slots[slotIdx]
		mask |= 1 << slotIdx
	}
	return assigned, total
}

func layoutChildrenRectangular(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
) (float64, float64, bool) {
	if !shouldUseRectangularChildLayout(len(childIDs)) {
		return 0, 0, false
	}

	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	kids := append([]string(nil), childIDs...)
	sort.SliceStable(kids, func(i, j int) bool {
		extI, intI := childRectUrgency(kids[i], metrics)
		extJ, intJ := childRectUrgency(kids[j], metrics)
		if extI != extJ {
			return extI > extJ
		}
		if intI != intJ {
			return intI > intJ
		}
		return kids[i] < kids[j]
	})

	var assigned map[string]rectSlot
	bestCost := math.MaxFloat64
	for _, pool := range rectangularSlotPools(len(kids)) {
		if len(pool) < len(kids) {
			continue
		}
		cand, cost := bestRectangularSlotAssignmentWithCost(kids, pool, metrics)
		if len(cand) != len(kids) {
			continue
		}
		if cost < bestCost {
			bestCost = cost
			assigned = cand
		}
	}
	if assigned == nil {
		return 0, 0, false
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, id := range kids {
		slot := assigned[id]
		relPos[id] = [2]float64{slot.X, slot.Y}
		if slot.X < minX {
			minX = slot.X
		}
		if slot.X > maxX {
			maxX = slot.X
		}
		if slot.Y < minY {
			minY = slot.Y
		}
		if slot.Y > maxY {
			maxY = slot.Y
		}
	}

	groupW := math.Max((maxX-minX)+nodeW+groupPadX*2, minGroupW)
	groupH := math.Max((maxY-minY)+nodeH+groupPadTop+groupPadBot, minGroupH)
	return groupW, groupH, true
}

type childExternalPull struct {
	avgX  float64
	avgY  float64
	count int
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
