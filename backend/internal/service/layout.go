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

type routeCell struct {
	X int
	Y int
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
				// Propagate root's Y to this node, but clamp so
				// the node never overlaps with previously placed
				// nodes in this layer.
				minCenter := y + h/2
				aligned := yPos[root[id]]
				if aligned < minCenter {
					aligned = minCenter
				}
				yPos[id] = aligned
				y = yPos[id] + h/2
				// Gap — use the same dimension-aware spacing that
				// enforceLayerSpacing applies later.
				nextID := ""
				if i := indexOf[id]; i < len(ids)-1 {
					nextID = ids[i+1]
				}
				if nextID != "" {
					y += spacingBetween(id, nextID, groupSizes)
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

func polylineLength(points []Point) float64 {
	total := 0.0
	for i := 0; i < len(points)-1; i++ {
		total += math.Hypot(points[i+1].X-points[i].X, points[i+1].Y-points[i].Y)
	}
	return total
}

func interpolatePoint(a, b Point, t float64) Point {
	return Point{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
	}
}

func hotspotCellAt(p Point) routeCell {
	return routeCell{
		X: int(math.Round(p.X / routeHotspotCell)),
		Y: int(math.Round(p.Y / routeHotspotCell)),
	}
}

func predictedRouteHotspotCells(points []Point) map[routeCell]bool {
	if len(points) < 2 {
		return nil
	}

	totalLen := polylineLength(points)
	if totalLen <= 0 {
		return nil
	}

	trim := math.Min(routeHotspotTrim, totalLen*0.2)
	cells := make(map[routeCell]bool)
	walked := 0.0

	for i := 0; i < len(points)-1; i++ {
		a := points[i]
		b := points[i+1]
		segLen := math.Hypot(b.X-a.X, b.Y-a.Y)
		if segLen <= 0 {
			continue
		}

		start := math.Max(0, trim-walked)
		end := math.Min(segLen, totalLen-trim-walked)
		if end < 0 || start > segLen || start > end {
			walked += segLen
			continue
		}

		for d := start; d <= end+1e-6; d += routeHotspotStep {
			t := d / segLen
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			cells[hotspotCellAt(interpolatePoint(a, b, t))] = true
		}

		cells[hotspotCellAt(interpolatePoint(a, b, end/segLen))] = true
		walked += segLen
	}

	return cells
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

// reserveEdgeCorridors ensures intermediate layers have gaps at the Y
// coordinates where long-span edges will route. Instead of reactively
// pushing nodes away from corridors (repackLayersAgainstEdgeCorridors),
// this proactively inserts gaps before the fine-tuning passes.
func reserveEdgeCorridors(
	layerNodes map[int][]string,
	maxLayer int,
	yCoords map[string]float64,
	groupSizes map[string][2]float64,
	spans []longEdgeSpan,
	layers map[string]int,
) {
	if len(spans) == 0 {
		return
	}
	minCorridorW := nodeH + 16 // minimum gap width for edge passage

	// Collect all corridor Y values per intermediate layer.
	type corridor struct{ y float64 }
	layerCorridors := make(map[int][]corridor)
	for _, span := range spans {
		srcL, tgtL := layers[span.Src], layers[span.Tgt]
		if srcL > tgtL {
			srcL, tgtL = tgtL, srcL
		}
		if tgtL-srcL < 2 {
			continue // only for edges that skip at least one layer
		}
		srcY := yCoords[span.Src] + span.SrcDY
		tgtY := yCoords[span.Tgt] + span.TgtDY

		for l := srcL + 1; l < tgtL; l++ {
			t := float64(l-srcL) / float64(tgtL-srcL)
			routeY := srcY + t*(tgtY-srcY)
			layerCorridors[l] = append(layerCorridors[l], corridor{y: routeY})
		}
	}

	for l, corridors := range layerCorridors {
		ids := layerNodes[l]
		if len(ids) == 0 {
			continue
		}

		// Deduplicate nearby corridors.
		sort.Slice(corridors, func(i, j int) bool { return corridors[i].y < corridors[j].y })
		merged := []float64{corridors[0].y}
		for _, c := range corridors[1:] {
			if c.y-merged[len(merged)-1] > float64(minCorridorW) {
				merged = append(merged, c.y)
			}
		}

		for _, cy := range merged {
			halfCW := float64(minCorridorW) / 2
			for _, id := range ids {
				h := nodeHeightForLayout(id, groupSizes)
				top := yCoords[id] - h/2
				bot := yCoords[id] + h/2

				// Check if the node's box blocks this corridor.
				if cy <= top-halfCW || cy >= bot+halfCW {
					continue
				}

				// Push node toward the side that requires less movement.
				distUp := cy - top + halfCW   // how far to push node up
				distDown := bot - cy + halfCW // how far to push node down
				if distUp <= distDown {
					yCoords[id] = cy - halfCW - h/2
				} else {
					yCoords[id] = cy + halfCW + h/2
				}
			}
		}

		// Re-sort the layer and re-enforce spacing.
		sort.SliceStable(ids, func(i, j int) bool {
			return yCoords[ids[i]] < yCoords[ids[j]]
		})
		enforceLayerSpacing(ids, yCoords, groupSizes)
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
	prevSz, prevIsGroup := groupSizes[prevID]
	nextSz, nextIsGroup := groupSizes[nextID]
	if !prevIsGroup && !nextIsGroup {
		return layerGapY
	}
	// Scale the gap with the larger neighbour so that edge corridors
	// between big groups get enough room.
	maxH := nodeH
	if prevIsGroup && prevSz[1] > maxH {
		maxH = prevSz[1]
	}
	if nextIsGroup && nextSz[1] > maxH {
		maxH = nextSz[1]
	}
	return groupLaneGapY + (groupGapY-groupLaneGapY)*math.Min(1, maxH/300.0)
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
			Top:    yTop - nodeH/2,
			Bottom: yBottom + nodeH/2,
		})
	}
	return mergeVerticalBands(bands)
}

func polylineBandsAcrossRect(points []Point, left, right float64) []verticalBand {
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
			Top:    yTop - nodeH/2,
			Bottom: yBottom + nodeH/2,
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

func countTopLevelBoxOverlaps(positions map[string][2]float64, groupSizes map[string][2]float64) (int, map[string]int) {
	ids := sortedPositionIDs(positions)
	overlaps := 0
	byNode := make(map[string]int)
	for i := 0; i < len(ids); i++ {
		a := topLevelRect(ids[i], positions, groupSizes, 8)
		for j := i + 1; j < len(ids); j++ {
			b := topLevelRect(ids[j], positions, groupSizes, 8)
			if math.Min(a.Right, b.Right) > math.Max(a.Left, b.Left) &&
				math.Min(a.Bottom, b.Bottom) > math.Max(a.Top, b.Top) {
				overlaps++
				byNode[ids[i]]++
				byNode[ids[j]]++
			}
		}
	}
	return overlaps, byNode
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
		routeBlocked := false
		for _, id := range boxIDs {
			if _, ok := nodeMap[id]; !ok {
				continue
			}
			if id == route.SrcTop || id == route.TgtTop {
				continue
			}
			box := topLevelRect(id, positions, groupSizes, 8)
			if polylineIntersectsRect(route.Points, box) {
				total++
				byNode[id]++
				routeBlocked = true
			}
		}
		// Also mark source/target parents as movable so the cleanup can
		// shift them to reroute around the blocking node.
		if routeBlocked {
			if route.SrcTop != "" {
				byNode[route.SrcTop]++
			}
			if route.TgtTop != "" {
				byNode[route.TgtTop]++
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

func countPredictedRouteHotspots(
	edges []model.Edge,
	positions map[string][2]float64,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) (int, map[string]int) {
	type routeInfo struct {
		predictedRoute
		cells map[routeCell]bool
	}

	routes := buildPredictedRoutes(edges, positions, nodeMap, childRelPos)
	infos := make([]routeInfo, 0, len(routes))
	for _, route := range routes {
		cells := predictedRouteHotspotCells(route.Points)
		if len(cells) == 0 {
			continue
		}
		infos = append(infos, routeInfo{
			predictedRoute: route,
			cells:          cells,
		})
	}

	total := 0
	byNode := make(map[string]int)
	for i := 0; i < len(infos); i++ {
		a := infos[i]
		for j := i + 1; j < len(infos); j++ {
			b := infos[j]
			if a.SrcTop == b.SrcTop || a.SrcTop == b.TgtTop || a.TgtTop == b.SrcTop || a.TgtTop == b.TgtTop {
				continue
			}

			shared := 0
			smaller, larger := a.cells, b.cells
			if len(smaller) > len(larger) {
				smaller, larger = larger, smaller
			}
			for cell := range smaller {
				if larger[cell] {
					shared++
				}
			}
			if shared == 0 {
				continue
			}

			penalty := shared * shared
			total += penalty
			byNode[a.SrcTop] += penalty
			byNode[a.TgtTop] += penalty
			byNode[b.SrcTop] += penalty
			byNode[b.TgtTop] += penalty
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
	boxOverlaps, overlapNodes := countTopLevelBoxOverlaps(positions, groupSizes)
	intersections, byNode := countPredictedRouteBoxIntersections(edges, positions, nodeMap, childRelPos, groupSizes)
	crossings, crossingNodes := countPredictedRouteCrossings(edges, positions, nodeMap, childRelPos)
	for id, count := range overlapNodes {
		byNode[id] += count
	}
	for id, count := range crossingNodes {
		byNode[id] += count
	}
	displacement := 0.0
	for id, pos := range positions {
		base := basePositions[id]
		displacement += math.Abs(pos[0]-base[0]) + math.Abs(pos[1]-base[1])
	}
	cost := float64(boxOverlaps)*1_000_000_000 +
		float64(intersections)*1_000_000 +
		float64(crossings)*50_000 +
		displacement*500
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
	// Scan range: wide enough for the node to clear its own height
	// plus comfortable corridor margin.
	scanRange := snapToGrid(math.Max(dy*2, h+groupGapY*2))
	raw := [][2]float64{current}

	// Systematic Y scan: try every gridSize step within the full range.
	// This ensures the search finds the optimal gap position even when
	// the gap is at an unusual offset.
	for yOff := -scanRange; yOff <= scanRange; yOff += gridSize {
		raw = append(raw, [2]float64{current[0], current[1] + yOff})
	}

	// Horizontal shifts + diagonal combinations at key Y offsets.
	raw = append(raw,
		[2]float64{current[0] - dx, current[1]},
		[2]float64{current[0] + dx, current[1]},
		[2]float64{current[0] - dx, current[1] - dy},
		[2]float64{current[0] - dx, current[1] + dy},
		[2]float64{current[0] + dx, current[1] - dy},
		[2]float64{current[0] + dx, current[1] + dy},
	)
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
		if math.Abs(candidate[0]-base[0]) > maxXShift+1 || math.Abs(candidate[1]-base[1]) > scanRange+1 {
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

// adjustChildrenToAvoidRouteCrossings shifts endpoint GROUPS (not
// individual children) so that cross-group edge routes pass through
// gaps between intermediate groups. This preserves the internal child
// layout within each group, preventing child-child overlaps.
func adjustChildrenToAvoidRouteCrossings(
	positions map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) {
	for iter := 0; iter < 4; iter++ {
		routes := buildPredictedRoutes(edges, positions, nodeMap, childRelPos)
		boxIDs := sortedPositionIDs(positions)
		adjusted := false
		// Track which groups have already been shifted this iteration
		// to avoid contradictory moves.
		shifted := make(map[string]bool)

		for _, route := range routes {
			var blockerRect rectBox
			found := false
			for _, id := range boxIDs {
				if id == route.SrcTop || id == route.TgtTop {
					continue
				}
				r := topLevelRect(id, positions, groupSizes, 4)
				if polylineIntersectsRect(route.Points, r) {
					blockerRect = r
					found = true
					break
				}
			}
			if !found {
				continue
			}

			srcPt := route.Points[0]
			tgtPt := route.Points[len(route.Points)-1]
			avgY := (srcPt.Y + tgtPt.Y) / 2
			gapAbove := blockerRect.Top - nodeH/2
			gapBelow := blockerRect.Bottom + nodeH/2
			targetGapY := gapAbove
			if math.Abs(avgY-gapBelow) < math.Abs(avgY-gapAbove) {
				targetGapY = gapBelow
			}

			// Shift both endpoint groups toward the gap Y.
			// Only shift the GROUP position — never touch childRelPos.
			for _, topID := range []string{route.SrcTop, route.TgtTop} {
				if shifted[topID] {
					continue
				}
				pos, ok := positions[topID]
				if !ok {
					continue
				}
				// Compute the average child-relative Y of children
				// involved in this route's edges to estimate where
				// the group center should be.
				shift := targetGapY - avgY
				// Apply a fraction: move halfway toward the ideal to
				// avoid overshooting when multiple routes conflict.
				newY := snapToGrid(pos[1] + shift/2)
				if math.Abs(newY-pos[1]) < gridSize {
					continue
				}
				positions[topID] = [2]float64{pos[0], newY}
				shifted[topID] = true
				adjusted = true
			}
		}

		if !adjusted {
			return
		}

		// After shifting groups, resolve any new overlaps.
		resolveTopLevelOverlaps(positions, groupSizes)
	}
}

// resolveTopLevelOverlaps pushes overlapping top-level boxes apart along
// their smallest overlap axis. This runs before the general route cleanup
// so that groups never visually overlap.
func resolveTopLevelOverlaps(positions map[string][2]float64, groupSizes map[string][2]float64) {
	for iter := 0; iter < 6; iter++ {
		overlaps, _ := countTopLevelBoxOverlaps(positions, groupSizes)
		if overlaps == 0 {
			return
		}
		ids := sortedPositionIDs(positions)
		moved := false
		for i := 0; i < len(ids); i++ {
			a := topLevelRect(ids[i], positions, groupSizes, 8)
			for j := i + 1; j < len(ids); j++ {
				b := topLevelRect(ids[j], positions, groupSizes, 8)
				overlapX := math.Min(a.Right, b.Right) - math.Max(a.Left, b.Left)
				overlapY := math.Min(a.Bottom, b.Bottom) - math.Max(a.Top, b.Top)
				if overlapX <= 0 || overlapY <= 0 {
					continue
				}
				posA := positions[ids[i]]
				posB := positions[ids[j]]
				// Push apart along the smaller overlap axis.
				if overlapY <= overlapX {
					shift := snapToGrid(overlapY/2 + groupLaneGapY/2)
					if posA[1] <= posB[1] {
						positions[ids[i]] = [2]float64{posA[0], snapToGrid(posA[1] - shift)}
						positions[ids[j]] = [2]float64{posB[0], snapToGrid(posB[1] + shift)}
					} else {
						positions[ids[i]] = [2]float64{posA[0], snapToGrid(posA[1] + shift)}
						positions[ids[j]] = [2]float64{posB[0], snapToGrid(posB[1] - shift)}
					}
				} else {
					shift := snapToGrid(overlapX/2 + layerGapX/2)
					if posA[0] <= posB[0] {
						positions[ids[i]] = [2]float64{snapToGrid(posA[0] - shift), posA[1]}
						positions[ids[j]] = [2]float64{snapToGrid(posB[0] + shift), posB[1]}
					} else {
						positions[ids[i]] = [2]float64{snapToGrid(posA[0] + shift), posA[1]}
						positions[ids[j]] = [2]float64{snapToGrid(posB[0] - shift), posB[1]}
					}
				}
				moved = true
			}
		}
		if !moved {
			return
		}
	}
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

	for iter := 0; iter < 8; iter++ {
		order := sortedPositionIDs(positions)
		sort.SliceStable(order, func(i, j int) bool {
			left := offenders[order[i]]
			right := offenders[order[j]]
			if left != right {
				return left > right
			}
			return order[i] < order[j]
		})

		routes := buildPredictedRoutes(edges, positions, nodeMap, childRelPos)
		moved := false
		for _, id := range order {
			if offenders[id] == 0 {
				continue
			}

			original := positions[id]
			best := original
			bestCost := currentCost
			candidates := cleanupCandidatePositions(id, original, basePositions[id], positions, groupSizes)
			rect := topLevelRect(id, positions, groupSizes, 8)
			h := nodeHeightForLayout(id, groupSizes)
			for _, route := range routes {
				srcPt := route.Points[0]
				tgtPt := route.Points[len(route.Points)-1]

				if id == route.SrcTop || id == route.TgtTop {
					// Y-alignment: move this group so its child aligns
					// vertically with the other endpoint, making the
					// route more horizontal and less likely to cross
					// intermediate groups.
					var shift float64
					if id == route.SrcTop {
						shift = tgtPt.Y - srcPt.Y
					} else {
						shift = srcPt.Y - tgtPt.Y
					}
					candidates = append(candidates,
						[2]float64{original[0], snapToGrid(original[1] + shift)},
						[2]float64{original[0], snapToGrid(original[1] + shift/2)},
					)
					continue
				}

				if !polylineIntersectsRect(route.Points, rect) {
					continue
				}
				// Corridor-aware: place node just above or below the
				// route that currently crosses it.
				routeY := (srcPt.Y + tgtPt.Y) / 2
				candidates = append(candidates,
					[2]float64{original[0], snapToGrid(routeY - h/2 - groupLaneGapY)},
					[2]float64{original[0], snapToGrid(routeY + h/2 + groupLaneGapY)},
				)
			}
			for _, candidate := range candidates {
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
		h := nodeHeightForLayout(id, groupSizes)
		shiftLimit[id] = math.Max(240, h)
		if target, ok := bandTargets[id]; ok {
			base = target
			anchor[id] = target
			shiftLimit[id] = math.Max(160, h*0.75)
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

func topLevelPredictedRouteCost(
	layerNodes map[int][]string,
	maxLayer int,
	layerX map[int]float64,
	yCoords map[string]float64,
	groupSizes map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) float64 {
	positions := buildTopLevelPositions(layerNodes, maxLayer, layerX, yCoords)
	intersections, _ := countPredictedRouteBoxIntersections(edges, positions, nodeMap, childRelPos, groupSizes)
	crossings, _ := countPredictedRouteCrossings(edges, positions, nodeMap, childRelPos)
	hotspots, _ := countPredictedRouteHotspots(edges, positions, nodeMap, childRelPos)
	return float64(intersections)*1_000_000_000 + float64(crossings)*250_000 + float64(hotspots)*18_000
}

func topLevelPredictedRoutePlacementCost(
	layerNodes map[int][]string,
	maxLayer int,
	layerX map[int]float64,
	yCoords map[string]float64,
	baseY map[string]float64,
	groupSizes map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) float64 {
	cost := topLevelPredictedRouteCost(layerNodes, maxLayer, layerX, yCoords, groupSizes, edges, nodeMap, childRelPos)
	drift := 0.0
	for id, y := range yCoords {
		base, ok := baseY[id]
		if !ok {
			continue
		}
		drift += math.Abs(y - base)
	}
	return cost + drift*260
}

func topLevelRoutePlacementShiftLimit(id string, groupSizes map[string][2]float64) float64 {
	return math.Max(200, nodeHeightForLayout(id, groupSizes)*0.9)
}

func buildTopLevelLayerNodes(topLevel []string, layers map[string]int) (map[int][]string, int) {
	layerNodes := make(map[int][]string)
	maxLayer := 0
	for _, id := range topLevel {
		l := layers[id]
		layerNodes[l] = append(layerNodes[l], id)
		if l > maxLayer {
			maxLayer = l
		}
	}
	return layerNodes, maxLayer
}

func buildTopLevelAdjacency(topLevel []string, outEdges map[string][]string) map[string]map[string]bool {
	topSet := make(map[string]bool, len(topLevel))
	for _, id := range topLevel {
		topSet[id] = true
	}

	fullAdj := make(map[string]map[string]bool, len(topLevel))
	for _, id := range topLevel {
		fullAdj[id] = make(map[string]bool)
	}
	for src, targets := range outEdges {
		if !topSet[src] {
			continue
		}
		for _, tgt := range targets {
			if !topSet[tgt] || src == tgt {
				continue
			}
			fullAdj[src][tgt] = true
			fullAdj[tgt][src] = true
		}
	}
	return fullAdj
}

func layerAssignmentOrderAndCost(
	topLevel []string,
	layers map[string]int,
	outEdges map[string][]string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) float64 {
	if len(topLevel) == 0 {
		return 0
	}

	layerNodes, maxLayer := buildTopLevelLayerNodes(topLevel, layers)
	fullAdj := buildTopLevelAdjacency(topLevel, outEdges)
	layerOf := make(map[string]int, len(topLevel))
	for l, ids := range layerNodes {
		for _, id := range ids {
			layerOf[id] = l
		}
	}

	siftingCrossMin(layerNodes, maxLayer, fullAdj)
	refineLayerOrderByChildConnections(layerNodes, maxLayer, edges, nodeMap, childRelPos)

	layerX := computeLayerXPositions(layerNodes, maxLayer, groupSizes)
	yCoords := brandesKopfAssign(layerNodes, maxLayer, fullAdj, groupSizes, layerOf)
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

	bandTargets := computeTopLevelBandTargets(topLevel, layers, outEdges, groupSizes)
	anchoredBandTargets := filterAnchoredBandTargets(topLevel, bandTargets, outEdges, groupSizes)
	for l := 0; l <= maxLayer; l++ {
		reorderLayerByDesiredY(layerNodes[l], anchoredBandTargets, yCoords)
		relaxLayerTowardDesiredY(layerNodes[l], yCoords, anchoredBandTargets, groupSizes)
	}

	cost := topLevelPredictedRouteCost(layerNodes, maxLayer, layerX, yCoords, groupSizes, edges, nodeMap, childRelPos)
	spanPenalty := 0.0
	for src, targets := range outEdges {
		for _, tgt := range targets {
			gap := layers[tgt] - layers[src]
			if gap < 1 {
				return math.MaxFloat64
			}
			if gap > 1 {
				extra := float64(gap - 1)
				spanPenalty += extra * extra * 16_000
			}
		}
	}

	return cost + spanPenalty + float64(maxLayer)*6_000
}

func preferredLayerDirection(id string, layers map[string]int, outEdges, inEdges map[string][]string) int {
	current := layers[id]
	idealSum := 0.0
	idealCount := 0.0
	for _, pred := range inEdges[id] {
		idealSum += float64(layers[pred] + 1)
		idealCount++
	}
	for _, succ := range outEdges[id] {
		idealSum += float64(layers[succ] - 1)
		idealCount++
	}
	if idealCount == 0 {
		return 0
	}
	ideal := idealSum / idealCount
	switch {
	case ideal < float64(current)-0.25:
		return -1
	case ideal > float64(current)+0.25:
		return 1
	default:
		return 0
	}
}

func layerMoveBounds(
	id string,
	layers map[string]int,
	outEdges, inEdges map[string][]string,
	maxLayer int,
) (int, int) {
	minLayer := 0
	for _, pred := range inEdges[id] {
		if candidate := layers[pred] + 1; candidate > minLayer {
			minLayer = candidate
		}
	}
	maxAllowed := maxLayer
	if succs := outEdges[id]; len(succs) > 0 {
		for _, succ := range succs {
			candidate := layers[succ] - 1
			if candidate < maxAllowed {
				maxAllowed = candidate
			}
		}
	}
	return minLayer, maxAllowed
}

func topLevelLayerMoveCandidates(
	id string,
	layers map[string]int,
	outEdges, inEdges map[string][]string,
	maxLayer int,
) []int {
	current := layers[id]
	minLayer, maxAllowed := layerMoveBounds(id, layers, outEdges, inEdges, maxLayer)
	if minLayer > maxAllowed {
		return nil
	}

	dir := preferredLayerDirection(id, layers, outEdges, inEdges)
	raw := []int{current}
	switch dir {
	case -1:
		raw = append(raw, current-1, current+1)
	case 1:
		raw = append(raw, current+1, current-1)
	default:
		raw = append(raw, current-1, current+1)
	}

	candidates := make([]int, 0, len(raw))
	seen := make(map[int]bool, len(raw))
	for _, candidate := range raw {
		if candidate < minLayer || candidate > maxAllowed {
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

func refineTopLevelLayersByPredictedRoutes(
	topLevel []string,
	layers map[string]int,
	outEdges map[string][]string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	groupSizes map[string][2]float64,
) {
	if len(topLevel) <= 1 {
		return
	}

	inEdges := buildInEdges(topLevel, outEdges)
	order := append([]string(nil), topLevel...)
	sort.SliceStable(order, func(i, j int) bool {
		leftDegree := len(outEdges[order[i]]) + len(inEdges[order[i]])
		rightDegree := len(outEdges[order[j]]) + len(inEdges[order[j]])
		if leftDegree != rightDegree {
			return leftDegree > rightDegree
		}
		return order[i] < order[j]
	})

	for sweep := 0; sweep < 5; sweep++ {
		moved := false
		currentCost := layerAssignmentOrderAndCost(topLevel, layers, outEdges, edges, nodeMap, childRelPos, groupSizes)
		maxLayer := 0
		for _, id := range topLevel {
			if l := layers[id]; l > maxLayer {
				maxLayer = l
			}
		}

		for _, id := range order {
			currentLayer := layers[id]
			bestLayer := currentLayer
			bestCost := currentCost
			for _, candidate := range topLevelLayerMoveCandidates(id, layers, outEdges, inEdges, maxLayer) {
				if candidate == currentLayer {
					continue
				}
				layers[id] = candidate
				trialCost := layerAssignmentOrderAndCost(topLevel, layers, outEdges, edges, nodeMap, childRelPos, groupSizes)
				if trialCost+1e-6 < bestCost {
					bestCost = trialCost
					bestLayer = candidate
				}
			}
			layers[id] = bestLayer
			if bestLayer != currentLayer {
				currentCost = bestCost
				moved = true
			}
		}

		if !moved {
			return
		}
	}
}

func predictedRouteBandsForTopLevelNode(
	id string,
	positions map[string][2]float64,
	routes []predictedRoute,
	groupSizes map[string][2]float64,
) []verticalBand {
	rect := topLevelRect(id, positions, groupSizes, 8)
	bands := make([]verticalBand, 0, len(routes))
	for _, route := range routes {
		if id == route.SrcTop || id == route.TgtTop {
			continue
		}
		bands = append(bands, polylineBandsAcrossRect(route.Points, rect.Left, rect.Right)...)
	}
	return mergeVerticalBands(bands)
}

func topLevelRouteIdealY(
	id string,
	currentY float64,
	routes []predictedRoute,
	bandTargets map[string]float64,
) float64 {
	sum := currentY * 2
	weight := 2.0
	if target, ok := bandTargets[id]; ok {
		sum += target * 1.5
		weight += 1.5
	}
	for _, route := range routes {
		switch {
		case id == route.SrcTop && len(route.Points) > 0:
			sum += route.Points[len(route.Points)-1].Y * 2
			weight += 2
		case id == route.TgtTop && len(route.Points) > 0:
			sum += route.Points[0].Y * 2
			weight += 2
		}
	}
	return snapToGrid(sum / weight)
}

func topLevelRouteCandidateCenters(
	id string,
	currentY float64,
	height float64,
	minY, maxY float64,
	blockedBands []verticalBand,
	routes []predictedRoute,
	bandTargets map[string]float64,
) []float64 {
	ideal := topLevelRouteIdealY(id, currentY, routes, bandTargets)
	raw := []float64{
		currentY,
		ideal,
		snapToGrid((currentY + ideal) / 2),
		currentY - 2*gridSize,
		currentY - gridSize,
		currentY + gridSize,
		currentY + 2*gridSize,
	}
	if target, ok := bandTargets[id]; ok {
		raw = append(raw, target, snapToGrid((currentY+target)/2))
	}
	for _, route := range routes {
		switch {
		case id == route.SrcTop && len(route.Points) > 0:
			endY := route.Points[len(route.Points)-1].Y
			raw = append(raw, endY, snapToGrid((currentY+endY)/2))
		case id == route.TgtTop && len(route.Points) > 0:
			startY := route.Points[0].Y
			raw = append(raw, startY, snapToGrid((currentY+startY)/2))
		}
	}
	for _, band := range blockedBands {
		raw = append(raw,
			band.Top-height/2-groupLaneGapY,
			band.Bottom+height/2+groupLaneGapY,
		)
	}

	candidates := make([]float64, 0, len(raw))
	seen := make(map[float64]bool, len(raw))
	for _, candidate := range raw {
		candidate = snapToGrid(candidate)
		candidate = chooseNearestValidCenter(candidate, minY, maxY, height, blockedBands)
		candidate = snapToGrid(candidate)
		if !math.IsInf(minY, -1) && candidate < minY {
			continue
		}
		if !math.IsInf(maxY, 1) && candidate > maxY {
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

func refineLayerOrderByPredictedRoutes(
	layerNodes map[int][]string,
	maxLayer int,
	layerX map[int]float64,
	yCoords map[string]float64,
	groupSizes map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
) {
	baseY := make(map[string]float64, len(yCoords))
	for id, y := range yCoords {
		baseY[id] = y
	}
	currentCost := topLevelPredictedRoutePlacementCost(layerNodes, maxLayer, layerX, yCoords, baseY, groupSizes, edges, nodeMap, childRelPos)

	for sweep := 0; sweep < 8; sweep++ {
		changed := false
		startL, endL, stepL := 0, maxLayer, 1
		if sweep%2 == 1 {
			startL, endL, stepL = maxLayer, 0, -1
		}

		for l := startL; ; l += stepL {
			ids := layerNodes[l]
			if len(ids) > 1 {
				for pass := 0; pass < len(ids); pass++ {
					layerChanged := false
					for i := 0; i < len(ids)-1; i++ {
						upperID := ids[i]
						lowerID := ids[i+1]
						upperY := yCoords[upperID]
						lowerY := yCoords[lowerID]
						if math.Abs(lowerY-baseY[upperID]) > topLevelRoutePlacementShiftLimit(upperID, groupSizes)+1 ||
							math.Abs(upperY-baseY[lowerID]) > topLevelRoutePlacementShiftLimit(lowerID, groupSizes)+1 {
							continue
						}

						ids[i], ids[i+1] = lowerID, upperID
						yCoords[upperID], yCoords[lowerID] = lowerY, upperY

						trialCost := topLevelPredictedRoutePlacementCost(layerNodes, maxLayer, layerX, yCoords, baseY, groupSizes, edges, nodeMap, childRelPos)
						if trialCost+1e-6 < currentCost {
							currentCost = trialCost
							layerChanged = true
							changed = true
							continue
						}

						ids[i], ids[i+1] = upperID, lowerID
						yCoords[upperID], yCoords[lowerID] = upperY, lowerY
					}
					if !layerChanged {
						break
					}
				}
			}

			if l == endL {
				break
			}
		}

		if !changed {
			return
		}
	}
}

func refineLayerCentersByPredictedRoutes(
	layerNodes map[int][]string,
	maxLayer int,
	layerX map[int]float64,
	yCoords map[string]float64,
	groupSizes map[string][2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	childRelPos map[string][2]float64,
	bandTargets map[string]float64,
) {
	baseY := make(map[string]float64, len(yCoords))
	for id, y := range yCoords {
		baseY[id] = y
	}

	currentCost := topLevelPredictedRoutePlacementCost(layerNodes, maxLayer, layerX, yCoords, baseY, groupSizes, edges, nodeMap, childRelPos)

	for sweep := 0; sweep < 6; sweep++ {
		changed := false
		startL, endL, stepL := 0, maxLayer, 1
		if sweep%2 == 1 {
			startL, endL, stepL = maxLayer, 0, -1
		}

		for l := startL; ; l += stepL {
			ids := layerNodes[l]
			if len(ids) > 0 {
				for i, id := range ids {
					positions := buildTopLevelPositions(layerNodes, maxLayer, layerX, yCoords)
					routes := buildPredictedRoutes(edges, positions, nodeMap, childRelPos)
					blockedBands := predictedRouteBandsForTopLevelNode(id, positions, routes, groupSizes)
					height := nodeHeightForLayout(id, groupSizes)

					minY := math.Inf(-1)
					if i > 0 {
						prevID := ids[i-1]
						minY = yCoords[prevID] + nodeHeightForLayout(prevID, groupSizes)/2 +
							spacingBetween(prevID, id, groupSizes) +
							height/2
					}
					maxY := math.Inf(1)
					if i < len(ids)-1 {
						nextID := ids[i+1]
						maxY = yCoords[nextID] - nodeHeightForLayout(nextID, groupSizes)/2 -
							spacingBetween(id, nextID, groupSizes) -
							height/2
					}

					currentY := yCoords[id]
					base := baseY[id]
					maxShift := topLevelRoutePlacementShiftLimit(id, groupSizes)
					bestY := currentY
					bestCost := currentCost
					for _, candidate := range topLevelRouteCandidateCenters(id, currentY, height, minY, maxY, blockedBands, routes, bandTargets) {
						if math.Abs(candidate-currentY) < 0.5 {
							continue
						}
						if math.Abs(candidate-base) > maxShift+1 {
							continue
						}
						yCoords[id] = candidate
						trialCost := topLevelPredictedRoutePlacementCost(layerNodes, maxLayer, layerX, yCoords, baseY, groupSizes, edges, nodeMap, childRelPos)
						if trialCost+1e-6 < bestCost {
							bestCost = trialCost
							bestY = candidate
						}
					}
					yCoords[id] = bestY
					if math.Abs(bestY-currentY) > 0.5 {
						currentCost = bestCost
						changed = true
					}
				}
			}

			if l == endL {
				break
			}
		}

		if !changed {
			return
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
	includeExternalRoutes bool,
	preferBestTemplate bool,
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
		boundaryDemands := buildChildBoundaryDemands(n.ID, groupPos, kids, edges, nodeMap, groupPositions)
		var externalLinks []childExternalLink
		if includeExternalRoutes {
			externalLinks = buildChildExternalLinks(n.ID, groupPos, kids, edges, nodeMap, groupPositions)
		}
		childEdges := childInternalEdges(kids, edges)
		metrics := computeChildRectMetrics(kids, edges, pullMap)

		if preferBestTemplate {
			type layoutCandidate struct {
				pos  map[string][2]float64
				w    float64
				h    float64
				cost float64
			}
			var best *layoutCandidate
			tryCandidate := func(build func(map[string][2]float64) (float64, float64, bool)) {
				trial := make(map[string][2]float64, len(kids))
				for _, kid := range kids {
					trial[kid] = childRelPos[kid]
				}
				w, h, ok := build(trial)
				if !ok {
					return
				}
				cost := childLayoutCost(kids, childEdges, externalLinks, trial) +
					scoreChildPlacementAffinity(kids, trial, metrics, boundaryDemands)
				if len(kids) <= 8 {
					cost += renderedChildLayoutCost(groupPos, kids, edges, nodeMap, groupPositions, trial, childRelPos)
				}
				if best == nil || cost < best.cost {
					best = &layoutCandidate{
						pos:  trial,
						w:    w,
						h:    h,
						cost: cost,
					}
				}
			}

			tryCandidate(func(rel map[string][2]float64) (float64, float64, bool) {
				return layoutChildrenPassThroughCorridor(kids, edges, rel, pullMap, boundaryDemands, childEdges, externalLinks)
			})
			tryCandidate(func(rel map[string][2]float64) (float64, float64, bool) {
				return layoutChildrenTwoColumnFlow(kids, edges, rel, pullMap, boundaryDemands, childEdges, externalLinks)
			})
			tryCandidate(func(rel map[string][2]float64) (float64, float64, bool) {
				return layoutChildrenVerticalLine(kids, edges, rel, pullMap, boundaryDemands, childEdges, externalLinks)
			})
			tryCandidate(func(rel map[string][2]float64) (float64, float64, bool) {
				return layoutChildrenHorizontalLine(kids, edges, rel, pullMap, boundaryDemands, childEdges, externalLinks)
			})
			tryCandidate(func(rel map[string][2]float64) (float64, float64, bool) {
				return layoutChildrenExternalPullBoundary(kids, edges, rel, pullMap, boundaryDemands, childEdges, externalLinks)
			})
			tryCandidate(func(rel map[string][2]float64) (float64, float64, bool) {
				return layoutChildrenRectangular(kids, edges, rel, pullMap, boundaryDemands, externalLinks)
			})

			if best != nil {
				for _, kid := range kids {
					childRelPos[kid] = best.pos[kid]
				}
				optimizeChildAssignmentsForRenderedRoutes(
					groupPos,
					kids,
					edges,
					nodeMap,
					groupPositions,
					childRelPos,
					childRelPos,
					childEdges,
					externalLinks,
					metrics,
					boundaryDemands,
				)
				groupW, groupH, _ := compactChildLayoutTightestClear(
					kids,
					childEdges,
					externalLinks,
					childRelPos,
					groupPos,
					edges,
					nodeMap,
					groupPositions,
					childRelPos,
				)
				groupSizes[n.ID] = [2]float64{groupW, groupH}
				continue
			}
		} else {
			if w, h, ok := layoutChildrenPassThroughCorridor(kids, edges, childRelPos, pullMap, boundaryDemands, childEdges, nil); ok {
				groupSizes[n.ID] = [2]float64{w, h}
				continue
			}

			if w, h, ok := layoutChildrenTwoColumnFlow(kids, edges, childRelPos, pullMap, boundaryDemands, childEdges, nil); ok {
				groupSizes[n.ID] = [2]float64{w, h}
				continue
			}

			if w, h, ok := layoutChildrenVerticalLine(kids, edges, childRelPos, pullMap, boundaryDemands, childEdges, nil); ok {
				groupSizes[n.ID] = [2]float64{w, h}
				continue
			}

			if w, h, ok := layoutChildrenHorizontalLine(kids, edges, childRelPos, pullMap, boundaryDemands, childEdges, nil); ok {
				groupSizes[n.ID] = [2]float64{w, h}
				continue
			}

			if w, h, ok := layoutChildrenExternalPullBoundary(kids, edges, childRelPos, pullMap, boundaryDemands, childEdges, nil); ok {
				groupSizes[n.ID] = [2]float64{w, h}
				continue
			}

			if w, h, ok := layoutChildrenRectangular(kids, edges, childRelPos, pullMap, boundaryDemands, nil); ok {
				groupSizes[n.ID] = [2]float64{w, h}
				continue
			}
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
					desiredYI := (relI[1] + boundaryDesiredY(ki, boundaryDemands, pullI, relI[1])) / 2
					desiredYJ := (relJ[1] + boundaryDesiredY(kj, boundaryDemands, pullJ, relJ[1])) / 2
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
				desiredXI = (relI[0] + boundaryDesiredX(sortedKids[i], boundaryDemands, pullI, relI[0])*2) / 3
				desiredYI = (relI[1] + boundaryDesiredY(sortedKids[i], boundaryDemands, pullI, relI[1])) / 2

				desiredXJ := relJ[0]
				desiredYJ := relJ[1]
				desiredXJ = (relJ[0] + boundaryDesiredX(sortedKids[j], boundaryDemands, pullJ, relJ[0])*2) / 3
				desiredYJ = (relJ[1] + boundaryDesiredY(sortedKids[j], boundaryDemands, pullJ, relJ[1])) / 2

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

		groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, childRelPos)
		optimizeChildAssignmentsForRenderedRoutes(
			groupPos,
			kids,
			edges,
			nodeMap,
			groupPositions,
			childRelPos,
			childRelPos,
			childEdges,
			externalLinks,
			metrics,
			boundaryDemands,
		)
		groupW, groupH, _ = compactChildLayoutTightestClear(
			kids,
			childEdges,
			externalLinks,
			childRelPos,
			groupPos,
			edges,
			nodeMap,
			groupPositions,
			childRelPos,
		)
		groupSizes[n.ID] = [2]float64{groupW, groupH}
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

func autoSizeGroup(childIDs []string) (float64, float64) {
	count := len(childIDs)
	if count == 0 {
		return minGroupW, minGroupH
	}

	cols := int(math.Ceil(math.Sqrt(float64(count))))
	rows := int(math.Ceil(float64(count) / float64(cols)))

	cellW := childLayoutNodeW + childLayoutCellPad
	cellH := childLayoutNodeH + childLayoutCellPad

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

	cellW := childLayoutNodeW + childLayoutCellPad
	cellH := childLayoutNodeH + childLayoutCellPad

	if w, h, ok := layoutChildrenRectangular(childIDs, edges, relPos, nil, nil, nil); ok {
		return w, h
	}
	if len(childEdges) > 0 {
		return layoutChildrenByLayers(childIDs, childEdges, cellW, cellH, relPos)
	}
	return layoutChildrenGrid(childIDs, cellW, cellH, relPos)
}

func buildChildPredictedRoutesFromRelPos(childEdges []model.Edge, relPos map[string][2]float64) [][]Point {
	routes := make([][]Point, 0, len(childEdges))
	for _, e := range childEdges {
		srcPos, ok := relPos[e.SourceID]
		if !ok {
			continue
		}
		tgtPos, ok := relPos[e.TargetID]
		if !ok {
			continue
		}
		src := Point{X: srcPos[0], Y: srcPos[1]}
		tgt := Point{X: tgtPos[0], Y: tgtPos[1]}
		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		routes = append(routes, points)
	}
	return routes
}

type childPredictedRouteInfo struct {
	srcID  string
	tgtID  string
	points []Point
}

func buildChildPredictedRouteInfosFromRelPos(childEdges []model.Edge, relPos map[string][2]float64) []childPredictedRouteInfo {
	routes := make([]childPredictedRouteInfo, 0, len(childEdges))
	for _, e := range childEdges {
		srcPos, ok := relPos[e.SourceID]
		if !ok {
			continue
		}
		tgtPos, ok := relPos[e.TargetID]
		if !ok {
			continue
		}
		src := Point{X: srcPos[0], Y: srcPos[1]}
		tgt := Point{X: tgtPos[0], Y: tgtPos[1]}
		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		routes = append(routes, childPredictedRouteInfo{
			srcID:  e.SourceID,
			tgtID:  e.TargetID,
			points: points,
		})
	}
	return routes
}

func childGroupRouteRects(childIDs []string, relPos map[string][2]float64) (rectBox, rectBox) {
	return childGroupRouteRectsWithSize(childIDs, relPos, childLayoutNodeW, childLayoutNodeH)
}

func childCompactGroupRouteRects(childIDs []string, relPos map[string][2]float64) (rectBox, rectBox) {
	return childGroupRouteRectsWithSize(childIDs, relPos, childCompactNodeW, childCompactNodeH)
}

func childGroupRouteRectsWithSize(childIDs []string, relPos map[string][2]float64, nodeW, nodeH float64) (rectBox, rectBox) {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, id := range childIDs {
		pos := relPos[id]
		if pos[0] < minX {
			minX = pos[0]
		}
		if pos[0] > maxX {
			maxX = pos[0]
		}
		if pos[1] < minY {
			minY = pos[1]
		}
		if pos[1] > maxY {
			maxY = pos[1]
		}
	}

	groupRect := rectBox{
		Left:   minX - nodeW/2 - groupPadX,
		Right:  maxX + nodeW/2 + groupPadX,
		Top:    minY - nodeH/2 - groupPadTop,
		Bottom: maxY + nodeH/2 + groupPadBot,
	}
	headerRect := rectBox{
		Left:   groupRect.Left,
		Right:  groupRect.Right,
		Top:    groupRect.Top,
		Bottom: groupRect.Top + groupPadTop*0.85,
	}
	return groupRect, headerRect
}

func clampFloat(v, minV, maxV float64) float64 {
	if minV > maxV {
		return (minV + maxV) / 2
	}
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func childExternalBoundaryPoint(groupRect, headerRect rectBox, link childExternalLink) Point {
	leftX := groupRect.Left + childRoutePad
	rightX := groupRect.Right - childRoutePad
	topY := headerRect.Bottom + childRoutePad
	bottomY := groupRect.Bottom - childRoutePad

	switch {
	case math.Abs(link.dx) >= math.Abs(link.dy):
		y := clampFloat(link.dy, topY, bottomY)
		if link.dx < 0 {
			return Point{X: leftX, Y: y}
		}
		return Point{X: rightX, Y: y}
	case link.dy < 0:
		x := clampFloat(link.dx, leftX, rightX)
		return Point{X: x, Y: topY}
	default:
		x := clampFloat(link.dx, leftX, rightX)
		return Point{X: x, Y: bottomY}
	}
}

func buildChildExternalRouteInfosFromRelPos(
	childIDs []string,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) []childPredictedRouteInfo {
	if len(childIDs) == 0 || len(externalLinks) == 0 {
		return nil
	}

	groupRect, headerRect := childGroupRouteRects(childIDs, relPos)
	routes := make([]childPredictedRouteInfo, 0, len(externalLinks))
	for _, link := range externalLinks {
		childPos, ok := relPos[link.childID]
		if !ok {
			continue
		}

		anchor := childExternalBoundaryPoint(groupRect, headerRect, link)
		childPt := Point{X: childPos[0], Y: childPos[1]}
		start, end := anchor, childPt
		srcID, tgtID := link.routeID, link.childID
		if !link.inbound {
			start, end = childPt, anchor
			srcID, tgtID = link.childID, link.routeID
		}

		points := []Point{start}
		points = append(points, compute8DirWaypoints(start, end)...)
		points = append(points, end)
		routes = append(routes, childPredictedRouteInfo{
			srcID:  srcID,
			tgtID:  tgtID,
			points: points,
		})
	}
	return routes
}

func childRelPositionForLayout(
	nodeID string,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
) ([2]float64, bool) {
	if pos, ok := localRelPos[nodeID]; ok {
		return pos, true
	}
	if pos, ok := allChildRelPos[nodeID]; ok {
		return pos, true
	}
	return [2]float64{}, false
}

func absoluteNodePointForRenderedChildLayout(
	nodeID string,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
) (Point, bool) {
	n, ok := nodeMap[nodeID]
	if !ok || n.ParentID == nil {
		pos, ok := groupPositions[nodeID]
		if !ok {
			return Point{}, false
		}
		return Point{X: pos[0], Y: pos[1]}, true
	}

	parentPos, ok := groupPositions[*n.ParentID]
	if !ok {
		return Point{}, false
	}
	rel, ok := childRelPositionForLayout(nodeID, localRelPos, allChildRelPos)
	if !ok {
		return Point{}, false
	}
	return Point{X: parentPos[0] + rel[0], Y: parentPos[1] + rel[1]}, true
}

func buildRenderedChildRouteInfos(
	childIDs []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
) []childPredictedRouteInfo {
	if len(childIDs) == 0 {
		return nil
	}

	childSet := make(map[string]bool, len(childIDs))
	for _, id := range childIDs {
		childSet[id] = true
	}

	routes := make([]childPredictedRouteInfo, 0, len(edges))
	for _, e := range edges {
		if e.EdgeType == model.EdgeTypeRelated {
			continue
		}
		srcIn := childSet[e.SourceID]
		tgtIn := childSet[e.TargetID]
		if !srcIn && !tgtIn {
			continue
		}

		src, ok := absoluteNodePointForRenderedChildLayout(e.SourceID, nodeMap, groupPositions, localRelPos, allChildRelPos)
		if !ok {
			continue
		}
		tgt, ok := absoluteNodePointForRenderedChildLayout(e.TargetID, nodeMap, groupPositions, localRelPos, allChildRelPos)
		if !ok {
			continue
		}

		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		routes = append(routes, childPredictedRouteInfo{
			srcID:  e.SourceID,
			tgtID:  e.TargetID,
			points: points,
		})
	}
	return routes
}

func countRenderedChildRouteNodeIntersections(
	childIDs []string,
	routes []childPredictedRouteInfo,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
) int {
	return countRenderedChildRouteNodeIntersectionsWithPadding(
		childIDs,
		routes,
		nodeMap,
		groupPositions,
		localRelPos,
		allChildRelPos,
		childRoutePad,
	)
}

func countRenderedChildRouteNodeIntersectionsWithPadding(
	childIDs []string,
	routes []childPredictedRouteInfo,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
	padding float64,
) int {
	total := 0
	for _, route := range routes {
		for _, id := range childIDs {
			if id == route.srcID || id == route.tgtID {
				continue
			}
			pos, ok := absoluteNodePointForRenderedChildLayout(id, nodeMap, groupPositions, localRelPos, allChildRelPos)
			if !ok {
				continue
			}
			if polylineIntersectsRect(route.points, childRectBoxAt([2]float64{pos.X, pos.Y}, padding)) {
				total++
			}
		}
	}
	return total
}

func translateRectBox(box rectBox, dx, dy float64) rectBox {
	return rectBox{
		Left:   box.Left + dx,
		Right:  box.Right + dx,
		Top:    box.Top + dy,
		Bottom: box.Bottom + dy,
	}
}

func inflateRectBox(box rectBox, padX, padY float64) rectBox {
	return rectBox{
		Left:   box.Left - padX,
		Right:  box.Right + padX,
		Top:    box.Top - padY,
		Bottom: box.Bottom + padY,
	}
}

func countRenderedChildHeaderIntersections(
	groupPos [2]float64,
	childIDs []string,
	routes []childPredictedRouteInfo,
	localRelPos map[string][2]float64,
) int {
	return countRenderedChildHeaderIntersectionsWithPadding(groupPos, childIDs, routes, localRelPos, 0)
}

func countRenderedChildHeaderIntersectionsWithPadding(
	groupPos [2]float64,
	childIDs []string,
	routes []childPredictedRouteInfo,
	localRelPos map[string][2]float64,
	padding float64,
) int {
	if len(childIDs) == 0 || len(routes) == 0 {
		return 0
	}
	_, headerRect := childGroupRouteRects(childIDs, localRelPos)
	headerRect = translateRectBox(headerRect, groupPos[0], groupPos[1])
	if padding > 0 {
		headerRect = inflateRectBox(headerRect, padding, padding)
	}
	hits := 0
	for _, route := range routes {
		if polylineIntersectsRect(route.points, headerRect) {
			hits++
		}
	}
	return hits
}

func renderedChildLayoutCost(
	groupPos [2]float64,
	childIDs []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
) float64 {
	routes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, localRelPos, allChildRelPos)
	if len(routes) == 0 {
		return 0
	}

	nodeHits := countRenderedChildRouteNodeIntersections(childIDs, routes, nodeMap, groupPositions, localRelPos, allChildRelPos)
	headerHits := countRenderedChildHeaderIntersections(groupPos, childIDs, routes, localRelPos)
	corridorHits := countRenderedChildRouteNodeIntersectionsWithPadding(
		childIDs,
		routes,
		nodeMap,
		groupPositions,
		localRelPos,
		allChildRelPos,
		childCorridorPad,
	)
	headerCorridorHits := countRenderedChildHeaderIntersectionsWithPadding(
		groupPos,
		childIDs,
		routes,
		localRelPos,
		childHeaderCorridorPad,
	)
	softNodeHits := max(0, corridorHits-nodeHits)
	softHeaderHits := max(0, headerCorridorHits-headerHits)
	return float64(nodeHits)*6_000_000 +
		float64(headerHits)*350_000 +
		float64(softNodeHits)*1_400_000 +
		float64(softHeaderHits)*80_000
}

func renderedChildRouteCrossings(
	childIDs []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
) int {
	routes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, localRelPos, allChildRelPos)
	return countPredictedChildRouteCrossings(routes)
}

func optimizeChildAssignmentsForRenderedRoutes(
	groupPos [2]float64,
	kids []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	metrics map[string]childRectMetric,
	demands map[string]childBoundaryDemand,
) bool {
	if len(kids) < 2 {
		return false
	}

	layoutCost := func() float64 {
		renderedCost := renderedChildLayoutCost(groupPos, kids, edges, nodeMap, groupPositions, localRelPos, allChildRelPos)
		crossingCost := float64(renderedChildRouteCrossings(kids, edges, nodeMap, groupPositions, localRelPos, allChildRelPos)) * 420
		affinityCost := scoreChildPlacementAffinity(kids, localRelPos, metrics, demands)
		if len(childEdges) == 0 && len(kids) >= 8 {
			return renderedCost + crossingCost + affinityCost*0.12
		}

		predictedWeight := 1.0
		if len(kids) <= 7 {
			predictedWeight = 0.22
		} else {
			predictedWeight = 0.38
		}
		return childLayoutCost(kids, childEdges, externalLinks, localRelPos)*predictedWeight +
			affinityCost +
			renderedCost +
			crossingCost
	}

	if len(kids) <= 8 {
		slots := make([][2]float64, len(kids))
		for i, id := range kids {
			slots[i] = localRelPos[id]
		}

		original := make(map[string][2]float64, len(kids))
		for _, id := range kids {
			original[id] = localRelPos[id]
		}

		bestCost := layoutCost()
		bestLayout := make(map[string][2]float64, len(kids))
		for _, id := range kids {
			bestLayout[id] = localRelPos[id]
		}

		used := make([]bool, len(slots))
		var dfs func(int)
		dfs = func(index int) {
			if index == len(kids) {
				cost := layoutCost()
				if cost+1e-6 < bestCost {
					bestCost = cost
					for _, id := range kids {
						bestLayout[id] = localRelPos[id]
					}
				}
				return
			}

			id := kids[index]
			for slotIdx, slot := range slots {
				if used[slotIdx] {
					continue
				}
				used[slotIdx] = true
				localRelPos[id] = slot
				dfs(index + 1)
				used[slotIdx] = false
			}
		}

		dfs(0)
		improved := false
		for _, id := range kids {
			if localRelPos[id] != bestLayout[id] {
				improved = true
			}
			localRelPos[id] = bestLayout[id]
		}
		if improved {
			return true
		}
		for _, id := range kids {
			localRelPos[id] = original[id]
		}
	}

	bestCost := layoutCost()
	improved := false
	for pass := 0; pass < 6; pass++ {
		passImproved := false
		for i := 0; i < len(kids); i++ {
			for j := i + 1; j < len(kids); j++ {
				leftID, rightID := kids[i], kids[j]
				leftPos := localRelPos[leftID]
				rightPos := localRelPos[rightID]
				if leftPos == rightPos {
					continue
				}

				localRelPos[leftID], localRelPos[rightID] = rightPos, leftPos
				cost := layoutCost()
				if cost+1e-6 < bestCost {
					bestCost = cost
					passImproved = true
					improved = true
					continue
				}
				localRelPos[leftID], localRelPos[rightID] = leftPos, rightPos
			}
		}
		if !passImproved {
			break
		}
	}

	return improved
}

func childRectBoxAt(pos [2]float64, padding float64) rectBox {
	return childRectBoxAtSize(pos, childLayoutNodeW, childLayoutNodeH, padding)
}

func childCompactRectBoxAt(pos [2]float64, padding float64) rectBox {
	return childRectBoxAtSize(pos, childCompactNodeW, childCompactNodeH, padding)
}

func childRectBoxAtSize(pos [2]float64, nodeW, nodeH, padding float64) rectBox {
	return rectBox{
		Left:   pos[0] - nodeW/2 - padding,
		Right:  pos[0] + nodeW/2 + padding,
		Top:    pos[1] - nodeH/2 - padding,
		Bottom: pos[1] + nodeH/2 + padding,
	}
}

func childRectsOverlap(a, b rectBox) bool {
	return a.Left < b.Right && a.Right > b.Left && a.Top < b.Bottom && a.Bottom > b.Top
}

func countChildBoxOverlaps(childIDs []string, relPos map[string][2]float64, padding float64) int {
	total := 0
	for i := 0; i < len(childIDs); i++ {
		a := childRectBoxAt(relPos[childIDs[i]], padding)
		for j := i + 1; j < len(childIDs); j++ {
			b := childRectBoxAt(relPos[childIDs[j]], padding)
			if childRectsOverlap(a, b) {
				total++
			}
		}
	}
	return total
}

type childLayoutViolations struct {
	overlapCount       int
	overlapX           float64
	overlapY           float64
	routeIntersections int
	routeHorizontal    int
	routeVertical      int
	headerHits         int
	corridorHits       int
	corridorHorizontal int
	corridorVertical   int
	headerCorridorHits int
	internalRoutes     int
	externalRoutes     int
	internalHeaders    int
	externalHeaders    int
	spanW              float64
	spanH              float64
}

type childRouteIntersectionStats struct {
	total      int
	horizontal int
	vertical   int
}

func measureChildLayoutViolations(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) childLayoutViolations {
	v := childLayoutViolations{}
	if len(childIDs) == 0 {
		return v
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for i := 0; i < len(childIDs); i++ {
		a := childRectBoxAt(relPos[childIDs[i]], 4)
		if a.Left < minX {
			minX = a.Left
		}
		if a.Top < minY {
			minY = a.Top
		}
		if a.Right > maxX {
			maxX = a.Right
		}
		if a.Bottom > maxY {
			maxY = a.Bottom
		}
		for j := i + 1; j < len(childIDs); j++ {
			b := childRectBoxAt(relPos[childIDs[j]], 4)
			if !childRectsOverlap(a, b) {
				continue
			}
			v.overlapCount++
			v.overlapX += math.Min(a.Right, b.Right) - math.Max(a.Left, b.Left)
			v.overlapY += math.Min(a.Bottom, b.Bottom) - math.Max(a.Top, b.Top)
		}
	}

	internalRouteStats := countPredictedChildRouteNodeIntersectionsDetailed(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
	)
	externalRouteStats := countPredictedChildRouteNodeIntersectionsDetailed(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
	)
	v.routeIntersections = internalRouteStats.total + externalRouteStats.total
	v.routeHorizontal = internalRouteStats.horizontal + externalRouteStats.horizontal
	v.routeVertical = internalRouteStats.vertical + externalRouteStats.vertical
	v.internalRoutes = internalRouteStats.total
	v.externalRoutes = externalRouteStats.total
	v.internalHeaders = countChildHeaderIntersections(childIDs, childEdges, relPos)
	v.externalHeaders = countChildExternalHeaderIntersections(childIDs, externalLinks, relPos)
	v.headerHits = v.internalHeaders + v.externalHeaders
	internalCorridorStats := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
		func(pos [2]float64) rectBox {
			return childRectBoxAt(pos, childCorridorPad)
		},
	)
	externalCorridorStats := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
		func(pos [2]float64) rectBox {
			return childRectBoxAt(pos, childCorridorPad)
		},
	)
	v.corridorHits = max(0, internalCorridorStats.total+externalCorridorStats.total-v.routeIntersections)
	v.corridorHorizontal = max(0, internalCorridorStats.horizontal+externalCorridorStats.horizontal-v.routeHorizontal)
	v.corridorVertical = max(0, internalCorridorStats.vertical+externalCorridorStats.vertical-v.routeVertical)
	internalHeaderCorridor := countPredictedChildHeaderIntersectionsWithPadding(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
		childLayoutNodeW,
		childLayoutNodeH,
		childHeaderCorridorPad,
	)
	externalHeaderCorridor := countPredictedChildHeaderIntersectionsWithPadding(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
		childLayoutNodeW,
		childLayoutNodeH,
		childHeaderCorridorPad,
	)
	v.headerCorridorHits = max(0, internalHeaderCorridor+externalHeaderCorridor-v.headerHits)
	v.spanW = maxX - minX
	v.spanH = maxY - minY
	return v
}

func measureCompactChildLayoutViolations(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) childLayoutViolations {
	v := childLayoutViolations{}
	if len(childIDs) == 0 {
		return v
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for i := 0; i < len(childIDs); i++ {
		a := childCompactRectBoxAt(relPos[childIDs[i]], childCompactPad)
		if a.Left < minX {
			minX = a.Left
		}
		if a.Top < minY {
			minY = a.Top
		}
		if a.Right > maxX {
			maxX = a.Right
		}
		if a.Bottom > maxY {
			maxY = a.Bottom
		}
		for j := i + 1; j < len(childIDs); j++ {
			b := childCompactRectBoxAt(relPos[childIDs[j]], childCompactPad)
			if !childRectsOverlap(a, b) {
				continue
			}
			v.overlapCount++
			v.overlapX += math.Min(a.Right, b.Right) - math.Max(a.Left, b.Left)
			v.overlapY += math.Min(a.Bottom, b.Bottom) - math.Max(a.Top, b.Top)
		}
	}

	internalRouteStats := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
		func(pos [2]float64) rectBox {
			return childCompactRectBoxAt(pos, childCompactRoutePad)
		},
	)
	externalRouteStats := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
		func(pos [2]float64) rectBox {
			return childCompactRectBoxAt(pos, childCompactRoutePad)
		},
	)
	v.routeIntersections = internalRouteStats.total + externalRouteStats.total
	v.routeHorizontal = internalRouteStats.horizontal + externalRouteStats.horizontal
	v.routeVertical = internalRouteStats.vertical + externalRouteStats.vertical
	v.internalRoutes = internalRouteStats.total
	v.externalRoutes = externalRouteStats.total
	v.internalHeaders = countCompactChildHeaderIntersections(childIDs, childEdges, relPos)
	v.externalHeaders = countCompactChildExternalHeaderIntersections(childIDs, externalLinks, relPos)
	v.headerHits = v.internalHeaders + v.externalHeaders
	internalCorridorStats := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
		func(pos [2]float64) rectBox {
			return childCompactRectBoxAt(pos, childCompactCorridorPad)
		},
	)
	externalCorridorStats := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
		func(pos [2]float64) rectBox {
			return childCompactRectBoxAt(pos, childCompactCorridorPad)
		},
	)
	v.corridorHits = max(0, internalCorridorStats.total+externalCorridorStats.total-v.routeIntersections)
	v.corridorHorizontal = max(0, internalCorridorStats.horizontal+externalCorridorStats.horizontal-v.routeHorizontal)
	v.corridorVertical = max(0, internalCorridorStats.vertical+externalCorridorStats.vertical-v.routeVertical)
	internalHeaderCorridor := countPredictedChildHeaderIntersectionsWithPadding(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
		childCompactNodeW,
		childCompactNodeH,
		childHeaderCorridorPad,
	)
	externalHeaderCorridor := countPredictedChildHeaderIntersectionsWithPadding(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
		childCompactNodeW,
		childCompactNodeH,
		childHeaderCorridorPad,
	)
	v.headerCorridorHits = max(0, internalHeaderCorridor+externalHeaderCorridor-v.headerHits)
	v.spanW = maxX - minX
	v.spanH = maxY - minY
	return v
}

func scaleChildLayoutAxes(childIDs []string, relPos map[string][2]float64, factorX, factorY float64) {
	for _, id := range childIDs {
		pos := relPos[id]
		relPos[id] = [2]float64{pos[0] * factorX, pos[1] * factorY}
	}
}

func childLayoutViolationsClear(v childLayoutViolations) bool {
	return v.overlapCount == 0 &&
		v.routeIntersections == 0 &&
		v.headerHits == 0 &&
		v.corridorHits == 0 &&
		v.headerCorridorHits == 0
}

func compactChildLayoutAxis(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
	axis byte,
) bool {
	if len(childIDs) == 0 {
		return false
	}

	bestSnapshot := cloneChildLayout(childIDs, relPos)
	bestCost := childLayoutCost(childIDs, childEdges, externalLinks, relPos)
	improved := false
	factors := []float64{0.76, 0.82, 0.88, 0.94, 0.97}

	for round := 0; round < 5; round++ {
		base := cloneChildLayout(childIDs, bestSnapshot)
		roundBest := bestSnapshot
		roundBestCost := bestCost
		roundImproved := false

		for _, factor := range factors {
			restoreChildLayout(relPos, base)
			switch axis {
			case 'x':
				scaleChildLayoutAxes(childIDs, relPos, factor, 1)
			case 'y':
				scaleChildLayoutAxes(childIDs, relPos, 1, factor)
			default:
				restoreChildLayout(relPos, bestSnapshot)
				return improved
			}
			recenterAndMeasureChildLayout(childIDs, relPos)

			if !childLayoutViolationsClear(measureChildLayoutViolations(childIDs, childEdges, externalLinks, relPos)) {
				continue
			}

			cost := childLayoutCost(childIDs, childEdges, externalLinks, relPos)
			if cost+1e-6 < roundBestCost {
				roundBest = cloneChildLayout(childIDs, relPos)
				roundBestCost = cost
				roundImproved = true
			}
		}

		if !roundImproved {
			break
		}

		bestSnapshot = roundBest
		bestCost = roundBestCost
		improved = true
		restoreChildLayout(relPos, bestSnapshot)
	}

	restoreChildLayout(relPos, bestSnapshot)
	recenterAndMeasureChildLayout(childIDs, relPos)
	return improved
}

func compactChildLayoutUniform(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) bool {
	if len(childIDs) == 0 {
		return false
	}

	bestSnapshot := cloneChildLayout(childIDs, relPos)
	bestCost := childLayoutCost(childIDs, childEdges, externalLinks, relPos)
	improved := false
	factors := []float64{0.72, 0.8, 0.88, 0.94, 0.97}

	for round := 0; round < 5; round++ {
		base := cloneChildLayout(childIDs, bestSnapshot)
		roundBest := bestSnapshot
		roundBestCost := bestCost
		roundImproved := false

		for _, factor := range factors {
			restoreChildLayout(relPos, base)
			scaleChildLayoutAxes(childIDs, relPos, factor, factor)
			recenterAndMeasureChildLayout(childIDs, relPos)

			if !childLayoutViolationsClear(measureChildLayoutViolations(childIDs, childEdges, externalLinks, relPos)) {
				continue
			}

			cost := childLayoutCost(childIDs, childEdges, externalLinks, relPos)
			if cost+1e-6 < roundBestCost {
				roundBest = cloneChildLayout(childIDs, relPos)
				roundBestCost = cost
				roundImproved = true
			}
		}

		if !roundImproved {
			break
		}

		bestSnapshot = roundBest
		bestCost = roundBestCost
		improved = true
		restoreChildLayout(relPos, bestSnapshot)
	}

	restoreChildLayout(relPos, bestSnapshot)
	recenterAndMeasureChildLayout(childIDs, relPos)
	return improved
}

func compactChildLayoutTightestClear(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
	groupPos [2]float64,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	allChildRelPos map[string][2]float64,
) (float64, float64, bool) {
	if len(childIDs) == 0 {
		return minGroupW, minGroupH, false
	}

	bestSnapshot := cloneChildLayout(childIDs, relPos)
	bestW, bestH := recenterAndMeasureCompactChildLayout(childIDs, relPos)
	baseExternalHits := childCompactExternalHitCount(childIDs, externalLinks, relPos, 0)
	baseRenderedHits := childCompactRenderedHitCount(childIDs, edges, nodeMap, groupPositions, relPos, allChildRelPos, 0)
	bestScore := math.Inf(1)
	if compactChildLayoutAcceptable(childIDs, childEdges, relPos) {
		bestScore = childCompactAreaScore(bestW, bestH) +
			childCompactExternalSoftPenalty(childIDs, externalLinks, relPos) +
			childCompactRenderedSoftPenalty(childIDs, edges, nodeMap, groupPositions, relPos, allChildRelPos, groupPos)
	}
	improved := false

	type shrinkCandidate struct {
		x float64
		y float64
	}
	candidates := []shrinkCandidate{
		{x: 0.94, y: 0.94},
		{x: 0.90, y: 0.90},
		{x: 0.86, y: 0.86},
		{x: 0.82, y: 0.82},
		{x: 0.78, y: 0.78},
		{x: 0.74, y: 0.74},
		{x: 0.70, y: 0.70},
		{x: 0.66, y: 0.66},
		{x: 0.60, y: 0.60},
		{x: 0.55, y: 0.55},
		{x: 0.50, y: 0.50},
		{x: 0.45, y: 0.45},
		{x: 0.40, y: 0.40},
		{x: 0.35, y: 0.35},
		{x: 0.94, y: 1.00},
		{x: 0.90, y: 1.00},
		{x: 0.86, y: 1.00},
		{x: 0.82, y: 1.00},
		{x: 0.78, y: 1.00},
		{x: 0.74, y: 1.00},
		{x: 0.70, y: 1.00},
		{x: 0.60, y: 1.00},
		{x: 0.50, y: 1.00},
		{x: 0.45, y: 1.00},
		{x: 0.40, y: 1.00},
		{x: 1.00, y: 0.94},
		{x: 1.00, y: 0.90},
		{x: 1.00, y: 0.86},
		{x: 1.00, y: 0.82},
		{x: 1.00, y: 0.78},
		{x: 1.00, y: 0.74},
		{x: 1.00, y: 0.70},
		{x: 1.00, y: 0.60},
		{x: 1.00, y: 0.50},
		{x: 1.00, y: 0.45},
		{x: 1.00, y: 0.40},
		{x: 0.90, y: 0.96},
		{x: 0.96, y: 0.90},
		{x: 0.86, y: 0.94},
		{x: 0.94, y: 0.86},
		{x: 0.82, y: 0.94},
		{x: 0.94, y: 0.82},
		{x: 0.78, y: 0.90},
		{x: 0.90, y: 0.78},
		{x: 0.74, y: 0.86},
		{x: 0.86, y: 0.74},
		{x: 0.70, y: 0.82},
		{x: 0.82, y: 0.70},
		{x: 0.60, y: 0.78},
		{x: 0.78, y: 0.60},
		{x: 0.50, y: 0.74},
		{x: 0.74, y: 0.50},
		{x: 0.45, y: 0.70},
		{x: 0.70, y: 0.45},
		{x: 0.40, y: 0.66},
		{x: 0.66, y: 0.40},
	}

	for pass := 0; pass < 8; pass++ {
		base := cloneChildLayout(childIDs, bestSnapshot)
		passBest := bestSnapshot
		passBestW, passBestH := bestW, bestH
		passBestScore := bestScore
		passImproved := false

		for _, candidate := range candidates {
			restoreChildLayout(relPos, base)
			scaleChildLayoutAxes(childIDs, relPos, candidate.x, candidate.y)
			groupW, groupH := recenterAndMeasureCompactChildLayout(childIDs, relPos)

			if !compactChildLayoutAcceptable(childIDs, childEdges, relPos) {
				continue
			}
			if childCompactExternalHitCount(childIDs, externalLinks, relPos, 0) > baseExternalHits {
				continue
			}
			if childCompactRenderedHitCount(childIDs, edges, nodeMap, groupPositions, relPos, allChildRelPos, 0) > baseRenderedHits {
				continue
			}
			score := childCompactAreaScore(groupW, groupH) +
				childCompactExternalSoftPenalty(childIDs, externalLinks, relPos) +
				childCompactRenderedSoftPenalty(childIDs, edges, nodeMap, groupPositions, relPos, allChildRelPos, groupPos)
			if score+1e-6 < passBestScore {
				passBest = cloneChildLayout(childIDs, relPos)
				passBestW, passBestH = groupW, groupH
				passBestScore = score
				passImproved = true
			}
		}

		if !passImproved {
			break
		}

		bestSnapshot = passBest
		bestW, bestH = passBestW, passBestH
		bestScore = passBestScore
		improved = true
		restoreChildLayout(relPos, bestSnapshot)
	}

	restoreChildLayout(relPos, bestSnapshot)
	bestW, bestH = recenterAndMeasureCompactChildLayout(childIDs, relPos)
	return bestW, bestH, improved
}

func compactChildLayoutAcceptable(
	childIDs []string,
	childEdges []model.Edge,
	relPos map[string][2]float64,
) bool {
	internal := measureCompactChildLayoutViolations(childIDs, childEdges, nil, relPos)
	return childLayoutViolationsClear(internal)
}

func childCompactExternalSoftPenalty(
	childIDs []string,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) float64 {
	externalRoutes := buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos)
	externalHits := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		externalRoutes,
		relPos,
		func(pos [2]float64) rectBox {
			return childCompactRectBoxAt(pos, childCompactRoutePad)
		},
	)
	headerHits := countCompactChildExternalHeaderIntersections(childIDs, externalLinks, relPos)
	return float64(externalHits.total)*8_000_000 + float64(headerHits)*650_000
}

func childCompactExternalHitCount(
	childIDs []string,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
	padding float64,
) int {
	externalRoutes := buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos)
	externalHits := countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		externalRoutes,
		relPos,
		func(pos [2]float64) rectBox {
			return childCompactRectBoxAt(pos, padding)
		},
	)
	return externalHits.total
}

func childCompactRenderedHitCount(
	childIDs []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
	padding float64,
) int {
	routes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, localRelPos, allChildRelPos)
	return countRenderedCompactChildRouteNodeIntersections(
		childIDs,
		routes,
		nodeMap,
		groupPositions,
		localRelPos,
		allChildRelPos,
		padding,
	)
}

func childCompactRenderedSoftPenalty(
	childIDs []string,
	edges []model.Edge,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
	groupPos [2]float64,
) float64 {
	routes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, localRelPos, allChildRelPos)
	nodeHits := countRenderedCompactChildRouteNodeIntersections(
		childIDs,
		routes,
		nodeMap,
		groupPositions,
		localRelPos,
		allChildRelPos,
		childCompactRoutePad,
	)
	headerHits := countRenderedCompactChildHeaderIntersections(groupPos, childIDs, routes, localRelPos)
	return float64(nodeHits)*50_000_000 + float64(headerHits)*5_000_000
}

func countRenderedCompactChildRouteNodeIntersections(
	childIDs []string,
	routes []childPredictedRouteInfo,
	nodeMap map[string]*model.Node,
	groupPositions map[string][2]float64,
	localRelPos map[string][2]float64,
	allChildRelPos map[string][2]float64,
	padding float64,
) int {
	total := 0
	for _, route := range routes {
		for _, id := range childIDs {
			if id == route.srcID || id == route.tgtID {
				continue
			}
			pos, ok := absoluteNodePointForRenderedChildLayout(id, nodeMap, groupPositions, localRelPos, allChildRelPos)
			if !ok {
				continue
			}
			if polylineIntersectsRect(route.points, childCompactRectBoxAt([2]float64{pos.X, pos.Y}, padding)) {
				total++
			}
		}
	}
	return total
}

func countRenderedCompactChildHeaderIntersections(
	groupPos [2]float64,
	childIDs []string,
	routes []childPredictedRouteInfo,
	localRelPos map[string][2]float64,
) int {
	if len(childIDs) == 0 || len(routes) == 0 {
		return 0
	}
	_, headerRect := childCompactGroupRouteRects(childIDs, localRelPos)
	headerRect = translateRectBox(headerRect, groupPos[0], groupPos[1])
	hits := 0
	for _, route := range routes {
		if polylineIntersectsRect(route.points, headerRect) {
			hits++
		}
	}
	return hits
}

func childCompactAreaScore(groupW, groupH float64) float64 {
	return groupW*groupH + (groupW+groupH)*120
}

func cloneChildLayout(childIDs []string, relPos map[string][2]float64) map[string][2]float64 {
	cloned := make(map[string][2]float64, len(childIDs))
	for _, id := range childIDs {
		cloned[id] = relPos[id]
	}
	return cloned
}

func restoreChildLayout(relPos map[string][2]float64, snapshot map[string][2]float64) {
	for id, pos := range snapshot {
		relPos[id] = pos
	}
}

func recenterAndMeasureChildLayout(
	childIDs []string,
	relPos map[string][2]float64,
) (float64, float64) {
	return recenterAndMeasureChildLayoutWithSize(childIDs, relPos, childLayoutNodeW, childLayoutNodeH)
}

func recenterAndMeasureCompactChildLayout(
	childIDs []string,
	relPos map[string][2]float64,
) (float64, float64) {
	return recenterAndMeasureChildLayoutWithSize(childIDs, relPos, childCompactNodeW, childCompactNodeH)
}

func recenterAndMeasureChildLayoutWithSize(
	childIDs []string,
	relPos map[string][2]float64,
	nodeW,
	nodeH float64,
) (float64, float64) {
	if len(childIDs) == 0 {
		return minGroupW, minGroupH
	}

	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, id := range childIDs {
		pos := relPos[id]
		if pos[0] < minX {
			minX = pos[0]
		}
		if pos[0] > maxX {
			maxX = pos[0]
		}
		if pos[1] < minY {
			minY = pos[1]
		}
		if pos[1] > maxY {
			maxY = pos[1]
		}
	}

	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	if centerX != 0 || centerY != 0 {
		for _, id := range childIDs {
			pos := relPos[id]
			relPos[id] = [2]float64{
				pos[0] - centerX,
				pos[1] - centerY,
			}
		}
	}

	minX, minY = math.MaxFloat64, math.MaxFloat64
	maxX, maxY = -math.MaxFloat64, -math.MaxFloat64
	for _, id := range childIDs {
		pos := relPos[id]
		if pos[0] < minX {
			minX = pos[0]
		}
		if pos[0] > maxX {
			maxX = pos[0]
		}
		if pos[1] < minY {
			minY = pos[1]
		}
		if pos[1] > maxY {
			maxY = pos[1]
		}
	}

	groupW := math.Max((maxX-minX)+nodeW+groupPadX*2, minGroupW)
	groupH := math.Max((maxY-minY)+nodeH+groupPadTop+groupPadBot, minGroupH)
	return groupW, groupH
}

func expandChildLayoutUntilClear(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) (float64, float64) {
	if len(childIDs) == 0 {
		return minGroupW, minGroupH
	}

	groupW, groupH := recenterAndMeasureChildLayout(childIDs, relPos)
	for attempt := 0; attempt < childGrowPasses; attempt++ {
		violations := measureChildLayoutViolations(childIDs, childEdges, externalLinks, relPos)
		if childLayoutViolationsClear(violations) {
			break
		}

		growX := 1.0
		growY := 1.0
		if violations.overlapCount > 0 {
			if violations.overlapX >= violations.overlapY {
				growX += 0.22
				growY += 0.06
			} else {
				growY += 0.22
				growX += 0.06
			}
		}
		if violations.internalRoutes > 0 {
			if violations.routeHorizontal > violations.routeVertical {
				growY += 0.14
				growX += 0.04
			} else if violations.routeVertical > violations.routeHorizontal {
				growX += 0.14
				growY += 0.04
			} else if violations.spanW <= violations.spanH {
				growX += 0.08
			} else {
				growY += 0.08
			}
		}
		if violations.externalRoutes > 0 {
			if violations.routeHorizontal > violations.routeVertical {
				growY += 0.05
			} else if violations.routeVertical > violations.routeHorizontal {
				growX += 0.05
			} else if violations.spanW <= violations.spanH {
				growX += 0.03
			} else {
				growY += 0.03
			}
		}
		if violations.internalHeaders > 0 {
			growY += 0.14
		}
		if violations.externalHeaders > 0 {
			growY += 0.04
		}
		if violations.corridorHits > 0 {
			if violations.corridorHorizontal > violations.corridorVertical {
				growY += 0.08
				growX += 0.02
			} else if violations.corridorVertical > violations.corridorHorizontal {
				growX += 0.08
				growY += 0.02
			} else if violations.spanW <= violations.spanH {
				growX += 0.05
			} else {
				growY += 0.05
			}
		}
		if violations.headerCorridorHits > 0 {
			growY += 0.06
		}

		scaleChildLayoutAxes(childIDs, relPos, growX*childGrowFactor, growY*childGrowFactor)
		groupW, groupH = recenterAndMeasureChildLayout(childIDs, relPos)
	}

	// Once the layout is valid, pull it back in until just before it becomes
	// invalid again so groups don't stay overly large.
	for attempt := 0; attempt < childCompactPasses; attempt++ {
		violations := measureChildLayoutViolations(childIDs, childEdges, externalLinks, relPos)
		if !childLayoutViolationsClear(violations) {
			break
		}

		snapshot := cloneChildLayout(childIDs, relPos)
		shrinkX, shrinkY := childCompactFactor, childCompactFactor
		if violations.spanW > violations.spanH*1.15 {
			shrinkX = childCompactFactor * 0.96
		}
		if violations.spanH > violations.spanW*1.15 {
			shrinkY = childCompactFactor * 0.96
		}
		scaleChildLayoutAxes(childIDs, relPos, shrinkX, shrinkY)
		groupW, groupH = recenterAndMeasureChildLayout(childIDs, relPos)

		after := measureChildLayoutViolations(childIDs, childEdges, externalLinks, relPos)
		if !childLayoutViolationsClear(after) {
			restoreChildLayout(relPos, snapshot)
			groupW, groupH = recenterAndMeasureChildLayout(childIDs, relPos)
			break
		}
	}

	if childLayoutViolationsClear(measureChildLayoutViolations(childIDs, childEdges, externalLinks, relPos)) {
		for pass := 0; pass < 4; pass++ {
			improvedUniform := compactChildLayoutUniform(childIDs, childEdges, externalLinks, relPos)
			improvedX := compactChildLayoutAxis(childIDs, childEdges, externalLinks, relPos, 'x')
			improvedY := compactChildLayoutAxis(childIDs, childEdges, externalLinks, relPos, 'y')
			if !improvedUniform && !improvedX && !improvedY {
				break
			}
		}
		groupW, groupH = recenterAndMeasureChildLayout(childIDs, relPos)
	}
	return groupW, groupH
}

func countPredictedChildRouteNodeIntersections(
	childIDs []string,
	routes []childPredictedRouteInfo,
	relPos map[string][2]float64,
) int {
	return countPredictedChildRouteNodeIntersectionsDetailed(childIDs, routes, relPos).total
}

func countPredictedChildRouteNodeIntersectionsDetailed(
	childIDs []string,
	routes []childPredictedRouteInfo,
	relPos map[string][2]float64,
) childRouteIntersectionStats {
	return countPredictedChildRouteNodeIntersectionsDetailedWithRect(
		childIDs,
		routes,
		relPos,
		func(pos [2]float64) rectBox {
			return childRectBoxAt(pos, childRoutePad)
		},
	)
}

func countPredictedChildRouteNodeIntersectionsDetailedWithRect(
	childIDs []string,
	routes []childPredictedRouteInfo,
	relPos map[string][2]float64,
	nodeRect func([2]float64) rectBox,
) childRouteIntersectionStats {
	stats := childRouteIntersectionStats{}
	for _, route := range routes {
		hits := 0
		for _, id := range childIDs {
			if id == route.srcID || id == route.tgtID {
				continue
			}
			if polylineIntersectsRect(route.points, nodeRect(relPos[id])) {
				stats.total++
				hits++
			}
		}
		if hits == 0 || len(route.points) == 0 {
			continue
		}
		start := route.points[0]
		end := route.points[len(route.points)-1]
		if math.Abs(end.X-start.X) >= math.Abs(end.Y-start.Y) {
			stats.horizontal += hits
		} else {
			stats.vertical += hits
		}
	}
	return stats
}

func childRoutesShareEndpoint(a, b childPredictedRouteInfo) bool {
	return a.srcID == b.srcID ||
		a.srcID == b.tgtID ||
		a.tgtID == b.srcID ||
		a.tgtID == b.tgtID
}

func countPredictedChildRouteCrossings(routes []childPredictedRouteInfo) int {
	total := 0
	for i := 0; i < len(routes); i++ {
		a := routes[i]
		for j := i + 1; j < len(routes); j++ {
			b := routes[j]
			if childRoutesShareEndpoint(a, b) {
				continue
			}
			crossed := false
			for ai := 0; ai < len(a.points)-1 && !crossed; ai++ {
				for bi := 0; bi < len(b.points)-1; bi++ {
					a1, a2 := a.points[ai], a.points[ai+1]
					b1, b2 := b.points[bi], b.points[bi+1]
					if pointsApproxEqual(a1, b1) || pointsApproxEqual(a1, b2) ||
						pointsApproxEqual(a2, b1) || pointsApproxEqual(a2, b2) {
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
			}
		}
	}
	return total
}

func countPredictedChildHeaderIntersections(
	childIDs []string,
	routes []childPredictedRouteInfo,
	relPos map[string][2]float64,
) int {
	return countPredictedChildHeaderIntersectionsWithPadding(
		childIDs,
		routes,
		relPos,
		childLayoutNodeW,
		childLayoutNodeH,
		0,
	)
}

func countPredictedChildHeaderIntersectionsWithPadding(
	childIDs []string,
	routes []childPredictedRouteInfo,
	relPos map[string][2]float64,
	nodeW, nodeH, padding float64,
) int {
	if len(childIDs) == 0 || len(routes) == 0 {
		return 0
	}
	_, headerRect := childGroupRouteRectsWithSize(childIDs, relPos, nodeW, nodeH)
	if padding > 0 {
		headerRect = inflateRectBox(headerRect, padding, padding)
	}
	hits := 0
	for _, route := range routes {
		if polylineIntersectsRect(route.points, headerRect) {
			hits++
		}
	}
	return hits
}

func countPredictedCompactChildHeaderIntersections(
	childIDs []string,
	routes []childPredictedRouteInfo,
	relPos map[string][2]float64,
) int {
	return countPredictedChildHeaderIntersectionsWithPadding(
		childIDs,
		routes,
		relPos,
		childCompactNodeW,
		childCompactNodeH,
		0,
	)
}

func countChildRouteNodeIntersections(
	childIDs []string,
	childEdges []model.Edge,
	relPos map[string][2]float64,
) int {
	return countPredictedChildRouteNodeIntersections(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
	)
}

func countChildExternalRouteNodeIntersections(
	childIDs []string,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) int {
	return countPredictedChildRouteNodeIntersections(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
	)
}

func countChildRouteCrossings(childEdges []model.Edge, relPos map[string][2]float64) int {
	return countPredictedChildRouteCrossings(
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
	)
}

func countChildAllRouteCrossings(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) int {
	routes := buildChildPredictedRouteInfosFromRelPos(childEdges, relPos)
	routes = append(routes, buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos)...)
	return countPredictedChildRouteCrossings(routes)
}

func childExternalLinkAxisStats(externalLinks []childExternalLink) (horizontal, vertical int) {
	for _, link := range externalLinks {
		if math.Abs(link.dx) >= math.Abs(link.dy)*1.1 {
			horizontal++
		} else {
			vertical++
		}
	}
	return
}

func childLayoutAspectPenalty(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	groupW, groupH float64,
) float64 {
	shortSide := math.Min(groupW, groupH)
	if shortSide <= 0 {
		return 0
	}

	ratio := math.Max(groupW, groupH) / shortSide
	allowed := 2.0
	horizontal, vertical := childExternalLinkAxisStats(externalLinks)
	sparse := len(childEdges) <= max(1, len(childIDs)-2)

	if horizontal >= max(3, vertical*2) || vertical >= max(3, horizontal*2) {
		allowed = 2.35
	}
	if sparse && (horizontal >= max(4, vertical*3) || vertical >= max(4, horizontal*3)) {
		allowed = 2.75
	}

	if ratio <= allowed {
		return 0
	}

	excess := ratio - allowed
	penalty := excess * excess * 2200
	if ratio > allowed+0.8 {
		penalty += (ratio - allowed - 0.8) * 1800
	}
	return penalty
}

func countChildHeaderIntersections(
	childIDs []string,
	childEdges []model.Edge,
	relPos map[string][2]float64,
) int {
	return countPredictedChildHeaderIntersections(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
	)
}

func countChildExternalHeaderIntersections(
	childIDs []string,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) int {
	return countPredictedChildHeaderIntersections(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
	)
}

func countCompactChildHeaderIntersections(
	childIDs []string,
	childEdges []model.Edge,
	relPos map[string][2]float64,
) int {
	return countPredictedCompactChildHeaderIntersections(
		childIDs,
		buildChildPredictedRouteInfosFromRelPos(childEdges, relPos),
		relPos,
	)
}

func countCompactChildExternalHeaderIntersections(
	childIDs []string,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) int {
	return countPredictedCompactChildHeaderIntersections(
		childIDs,
		buildChildExternalRouteInfosFromRelPos(childIDs, externalLinks, relPos),
		relPos,
	)
}

func childLayoutCost(
	childIDs []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	relPos map[string][2]float64,
) float64 {
	crossings := countChildAllRouteCrossings(childIDs, childEdges, externalLinks, relPos)
	intersections := countChildRouteNodeIntersections(childIDs, childEdges, relPos) +
		countChildExternalRouteNodeIntersections(childIDs, externalLinks, relPos)
	headerHits := countChildHeaderIntersections(childIDs, childEdges, relPos) +
		countChildExternalHeaderIntersections(childIDs, externalLinks, relPos)
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, id := range childIDs {
		pos := relPos[id]
		if pos[0] < minX {
			minX = pos[0]
		}
		if pos[0] > maxX {
			maxX = pos[0]
		}
		if pos[1] < minY {
			minY = pos[1]
		}
		if pos[1] > maxY {
			maxY = pos[1]
		}
	}
	groupW := math.Max((maxX-minX)+childLayoutNodeW+groupPadX*2, minGroupW)
	groupH := math.Max((maxY-minY)+childLayoutNodeH+groupPadTop+groupPadBot, minGroupH)
	overlaps := countChildBoxOverlaps(childIDs, relPos, 4)
	aspectPenalty := childLayoutAspectPenalty(childIDs, childEdges, externalLinks, groupW, groupH)
	return float64(overlaps)*1_500_000 + float64(intersections)*1_000_000 + float64(headerHits)*50_000 + float64(crossings)*36_000 + (groupW+groupH)*16 + math.Max(groupW, groupH)*6 + aspectPenalty
}

func layeredChildLayoutCost(
	childIDs []string,
	childEdges []model.Edge,
	relPos map[string][2]float64,
) float64 {
	return childLayoutCost(childIDs, childEdges, nil, relPos)
}

func layeredColumnOffsetCandidates(layerIndex int, cellH float64) []float64 {
	offset := math.Max(childLayoutNodeH/2+8, cellH*0.35)
	if layerIndex%2 == 0 {
		return []float64{0, offset, -offset}
	}
	return []float64{offset, -offset, 0}
}

func layeredRowSlotSets(count int, cellH, centerY, columnOffset float64) [][]float64 {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return [][]float64{{centerY + columnOffset}}
	}

	pitch := math.Max(childLayoutNodeH+12, cellH*0.7)
	totalSlots := count + 1
	blankIdxs := []int{totalSlots / 2}
	if totalSlots%2 == 0 {
		blankIdxs = []int{totalSlots/2 - 1, totalSlots / 2}
	}

	slotSets := make([][]float64, 0, len(blankIdxs))
	for _, blankIdx := range blankIdxs {
		start := -float64(totalSlots-1)*pitch/2 + centerY + columnOffset
		slots := make([]float64, 0, count)
		for idx := 0; idx < totalSlots; idx++ {
			if idx == blankIdx {
				continue
			}
			slots = append(slots, start+float64(idx)*pitch)
		}
		slotSets = append(slotSets, slots)
	}
	return slotSets
}

func childNeighborAverageY(id string, childEdges []model.Edge, relPos map[string][2]float64) float64 {
	sum := 0.0
	count := 0.0
	for _, e := range childEdges {
		var other string
		switch {
		case e.SourceID == id:
			other = e.TargetID
		case e.TargetID == id:
			other = e.SourceID
		default:
			continue
		}
		pos, ok := relPos[other]
		if !ok {
			continue
		}
		sum += pos[1]
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func permuteStrings(ids []string) [][]string {
	if len(ids) <= 1 {
		out := make([]string, len(ids))
		copy(out, ids)
		return [][]string{out}
	}
	result := make([][]string, 0)
	for i := range ids {
		head := ids[i]
		rest := make([]string, 0, len(ids)-1)
		rest = append(rest, ids[:i]...)
		rest = append(rest, ids[i+1:]...)
		for _, tail := range permuteStrings(rest) {
			perm := make([]string, 0, len(ids))
			perm = append(perm, head)
			perm = append(perm, tail...)
			result = append(result, perm)
		}
	}
	return result
}

func columnOrderingCandidates(ids []string, childEdges []model.Edge, relPos map[string][2]float64) [][]string {
	if len(ids) <= 1 {
		out := make([]string, len(ids))
		copy(out, ids)
		return [][]string{out}
	}
	seen := make(map[string]bool)
	out := make([][]string, 0)
	addCandidate := func(candidate []string) {
		key := fmt.Sprint(candidate)
		if seen[key] {
			return
		}
		seen[key] = true
		dup := make([]string, len(candidate))
		copy(dup, candidate)
		out = append(out, dup)
	}

	current := make([]string, len(ids))
	copy(current, ids)
	addCandidate(current)

	asc := make([]string, len(ids))
	copy(asc, ids)
	sort.SliceStable(asc, func(i, j int) bool {
		left := childNeighborAverageY(asc[i], childEdges, relPos)
		right := childNeighborAverageY(asc[j], childEdges, relPos)
		if left != right {
			return left < right
		}
		return asc[i] < asc[j]
	})
	addCandidate(asc)

	desc := make([]string, len(asc))
	copy(desc, asc)
	for i, j := 0, len(desc)-1; i < j; i, j = i+1, j-1 {
		desc[i], desc[j] = desc[j], desc[i]
	}
	addCandidate(desc)

	if len(ids) <= 4 {
		for _, perm := range permuteStrings(ids) {
			addCandidate(perm)
		}
	}
	return out
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
	startX := -float64(numLayers-1) * cellW / 2
	centerY := (groupPadTop - groupPadBot) / 2

	for li, layer := range layerOrder {
		x := startX + float64(li)*cellW
		slots := layeredRowSlotSets(len(layer), cellH, centerY, 0)
		rowSlots := []float64{centerY}
		if len(slots) > 0 {
			rowSlots = slots[0]
		}
		for ni, id := range layer {
			if ni < len(rowSlots) {
				relPos[id] = [2]float64{x, rowSlots[ni]}
			}
		}
	}

	for sweep := 0; sweep < 2; sweep++ {
		for li, layer := range layerOrder {
			x := startX + float64(li)*cellW
			bestCost := layeredChildLayoutCost(childIDs, childEdges, relPos)
			bestOrder := append([]string(nil), layer...)
			bestSlots := make([]float64, len(layer))
			for i, id := range layer {
				bestSlots[i] = relPos[id][1]
			}

			for _, offset := range layeredColumnOffsetCandidates(li, cellH) {
				for _, slots := range layeredRowSlotSets(len(layer), cellH, centerY, offset) {
					for _, order := range columnOrderingCandidates(layer, childEdges, relPos) {
						for rowIdx, id := range order {
							relPos[id] = [2]float64{x, slots[rowIdx]}
						}
						cost := layeredChildLayoutCost(childIDs, childEdges, relPos)
						if cost < bestCost {
							bestCost = cost
							bestOrder = append(bestOrder[:0], order...)
							bestSlots = append(bestSlots[:0], slots...)
						}
					}
				}
			}

			copy(layerOrder[li], bestOrder)
			for rowIdx, id := range layerOrder[li] {
				relPos[id] = [2]float64{x, bestSlots[rowIdx]}
			}
		}
	}

	return expandChildLayoutUntilClear(childIDs, childEdges, nil, relPos)
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

	if expandedW, expandedH := expandChildLayoutUntilClear(childIDs, nil, nil, relPos); expandedW > groupW || expandedH > groupH {
		groupW, groupH = expandedW, expandedH
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
	buildCorners := func(sideX, sideY float64) []rectSlot {
		return []rectSlot{
			{X: -sideX, Y: -sideY + offsetY, NX: -1, NY: -1, Priority: 0},
			{X: sideX, Y: -sideY + offsetY, NX: 1, NY: -1, Priority: 0},
			{X: -sideX, Y: sideY + offsetY, NX: -1, NY: 1, Priority: 0},
			{X: sideX, Y: sideY + offsetY, NX: 1, NY: 1, Priority: 0},
		}
	}
	buildDenseTopMid := func(sideY float64) rectSlot {
		return rectSlot{X: 0, Y: -sideY + offsetY, NX: 0, NY: -1, Priority: 8}
	}
	buildDenseBotMid := func(sideY float64) rectSlot {
		return rectSlot{X: 0, Y: sideY + offsetY, NX: 0, NY: 1, Priority: 8}
	}
	buildDenseLeftMid := func(sideX float64) rectSlot {
		return rectSlot{X: -sideX, Y: offsetY, NX: -1, NY: 0, Priority: 8}
	}
	buildDenseRightMid := func(sideX float64) rectSlot {
		return rectSlot{X: sideX, Y: offsetY, NX: 1, NY: 0, Priority: 8}
	}
	if count <= 4 {
		return [][]rectSlot{
			{
				{X: -rectCompactCornerX, Y: -rectCompactCornerY + offsetY, NX: -1, NY: -1, Priority: 0},
				{X: rectCompactCornerX, Y: -rectCompactCornerY + offsetY, NX: 1, NY: -1, Priority: 0},
				{X: -rectCompactCornerX, Y: rectCompactCornerY + offsetY, NX: -1, NY: 1, Priority: 0},
				{X: rectCompactCornerX, Y: rectCompactCornerY + offsetY, NX: 1, NY: 1, Priority: 0},
			},
			{
				{X: -rectDenseCornerX, Y: -rectDenseCornerY + offsetY, NX: -1, NY: -1, Priority: 1},
				{X: rectDenseCornerX, Y: -rectDenseCornerY + offsetY, NX: 1, NY: -1, Priority: 1},
				{X: -rectDenseCornerX, Y: rectDenseCornerY + offsetY, NX: -1, NY: 1, Priority: 1},
				{X: rectDenseCornerX, Y: rectDenseCornerY + offsetY, NX: 1, NY: 1, Priority: 1},
			},
		}
	}

	corners := buildCorners(rectCompactSideX, rectCompactSideY)
	denseCorners := buildCorners(rectDenseSideX, rectDenseSideY)
	topMid := rectSlot{X: 0, Y: -rectCompactSideY + offsetY, NX: 0, NY: -1, Priority: 8}
	botMid := rectSlot{X: 0, Y: rectCompactSideY + offsetY, NX: 0, NY: 1, Priority: 8}
	leftMid := rectSlot{X: -rectCompactSideX, Y: offsetY, NX: -1, NY: 0, Priority: 8}
	rightMid := rectSlot{X: rectCompactSideX, Y: offsetY, NX: 1, NY: 0, Priority: 8}
	denseTopMid := buildDenseTopMid(rectDenseSideY)
	denseBotMid := buildDenseBotMid(rectDenseSideY)
	denseLeftMid := buildDenseLeftMid(rectDenseSideX)
	denseRightMid := buildDenseRightMid(rectDenseSideX)
	center := rectSlot{X: 0, Y: offsetY, NX: 0, NY: 0, Priority: 18}

	if count == 5 {
		staggerBottomLeft := []rectSlot{
			{X: -rectCompactSideX * 1.05, Y: -rectCompactSideY + offsetY, NX: -1, NY: -1, Priority: 0},
			{X: 0, Y: -rectCompactSideY + offsetY, NX: 0, NY: -1, Priority: 4},
			{X: rectCompactSideX * 1.05, Y: -rectCompactSideY + offsetY, NX: 1, NY: -1, Priority: 0},
			{X: -rectCompactSideX * 0.45, Y: rectCompactSideY + offsetY, NX: -0.45, NY: 1, Priority: 3},
			{X: rectCompactSideX * 0.55, Y: rectCompactSideY + offsetY, NX: 0.55, NY: 1, Priority: 3},
		}
		staggerBottomRight := []rectSlot{
			{X: -rectCompactSideX * 1.05, Y: -rectCompactSideY + offsetY, NX: -1, NY: -1, Priority: 0},
			{X: 0, Y: -rectCompactSideY + offsetY, NX: 0, NY: -1, Priority: 4},
			{X: rectCompactSideX * 1.05, Y: -rectCompactSideY + offsetY, NX: 1, NY: -1, Priority: 0},
			{X: -rectCompactSideX * 0.55, Y: rectCompactSideY + offsetY, NX: -0.55, NY: 1, Priority: 3},
			{X: rectCompactSideX * 0.45, Y: rectCompactSideY + offsetY, NX: 0.45, NY: 1, Priority: 3},
		}
		staggerTopLeft := []rectSlot{
			{X: -rectCompactSideX * 0.45, Y: -rectCompactSideY + offsetY, NX: -0.45, NY: -1, Priority: 3},
			{X: rectCompactSideX * 0.55, Y: -rectCompactSideY + offsetY, NX: 0.55, NY: -1, Priority: 3},
			{X: -rectCompactSideX * 1.05, Y: rectCompactSideY + offsetY, NX: -1, NY: 1, Priority: 0},
			{X: 0, Y: rectCompactSideY + offsetY, NX: 0, NY: 1, Priority: 4},
			{X: rectCompactSideX * 1.05, Y: rectCompactSideY + offsetY, NX: 1, NY: 1, Priority: 0},
		}
		staggerTopRight := []rectSlot{
			{X: -rectCompactSideX * 0.55, Y: -rectCompactSideY + offsetY, NX: -0.55, NY: -1, Priority: 3},
			{X: rectCompactSideX * 0.45, Y: -rectCompactSideY + offsetY, NX: 0.45, NY: -1, Priority: 3},
			{X: -rectCompactSideX * 1.05, Y: rectCompactSideY + offsetY, NX: -1, NY: 1, Priority: 0},
			{X: 0, Y: rectCompactSideY + offsetY, NX: 0, NY: 1, Priority: 4},
			{X: rectCompactSideX * 1.05, Y: rectCompactSideY + offsetY, NX: 1, NY: 1, Priority: 0},
		}
		// Keep one spare slot so the assignment can leave an intentional lane
		// open instead of filling every midpoint.
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
			append(append([]rectSlot(nil), corners...), topMid, botMid),
			append(append([]rectSlot(nil), corners...), leftMid, rightMid),
			append(append([]rectSlot(nil), denseCorners...), denseTopMid),
			append(append([]rectSlot(nil), denseCorners...), denseBotMid),
			append(append([]rectSlot(nil), denseCorners...), denseLeftMid),
			append(append([]rectSlot(nil), denseCorners...), denseRightMid),
			append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid),
			append(append([]rectSlot(nil), denseCorners...), denseLeftMid, denseRightMid),
			staggerBottomLeft,
			staggerBottomRight,
			staggerTopLeft,
			staggerTopRight,
		}
	}

	if count == 6 {
		// Prefer candidates that can leave a spare lane through the box.
		wide := append(append([]rectSlot(nil), corners...), topMid, botMid)
		tall := append(append([]rectSlot(nil), corners...), leftMid, rightMid)
		ring := append(append([]rectSlot(nil), corners...), topMid, botMid, leftMid, rightMid)
		denseWide := append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid)
		denseTall := append(append([]rectSlot(nil), denseCorners...), denseLeftMid, denseRightMid)
		denseRing := append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid, denseLeftMid, denseRightMid)
		return [][]rectSlot{wide, tall, ring, denseWide, denseTall, denseRing}
	}

	if count == 7 {
		// Prefer ring-style layouts that still leave at least one midpoint empty.
		h1 := append(append([]rectSlot(nil), corners...), topMid, botMid, leftMid)
		h2 := append(append([]rectSlot(nil), corners...), topMid, botMid, rightMid)
		v1 := append(append([]rectSlot(nil), corners...), leftMid, rightMid, topMid)
		v2 := append(append([]rectSlot(nil), corners...), leftMid, rightMid, botMid)
		ring := append(append([]rectSlot(nil), corners...), topMid, botMid, leftMid, rightMid)
		dh1 := append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid, denseLeftMid)
		dh2 := append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid, denseRightMid)
		dv1 := append(append([]rectSlot(nil), denseCorners...), denseLeftMid, denseRightMid, denseTopMid)
		dv2 := append(append([]rectSlot(nil), denseCorners...), denseLeftMid, denseRightMid, denseBotMid)
		denseRing := append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid, denseLeftMid, denseRightMid)
		return [][]rectSlot{h1, h2, v1, v2, ring, dh1, dh2, dv1, dv2, denseRing}
	}

	if count == 8 {
		rowXs := centeredAxisPositions(4, 92, 0)
		denseRowXs := centeredAxisPositions(4, 72, 0)
		row4x2 := []rectSlot{
			{X: rowXs[0], Y: -rectCompactSideY + offsetY, NX: -1, NY: -1, Priority: 0},
			{X: rowXs[1], Y: -rectCompactSideY + offsetY, NX: -0.35, NY: -1, Priority: 4},
			{X: rowXs[2], Y: -rectCompactSideY + offsetY, NX: 0.35, NY: -1, Priority: 4},
			{X: rowXs[3], Y: -rectCompactSideY + offsetY, NX: 1, NY: -1, Priority: 0},
			{X: rowXs[0], Y: rectCompactSideY + offsetY, NX: -1, NY: 1, Priority: 0},
			{X: rowXs[1], Y: rectCompactSideY + offsetY, NX: -0.35, NY: 1, Priority: 4},
			{X: rowXs[2], Y: rectCompactSideY + offsetY, NX: 0.35, NY: 1, Priority: 4},
			{X: rowXs[3], Y: rectCompactSideY + offsetY, NX: 1, NY: 1, Priority: 0},
		}
		denseRow4x2 := []rectSlot{
			{X: denseRowXs[0], Y: -rectDenseSideY + offsetY, NX: -1, NY: -1, Priority: 1},
			{X: denseRowXs[1], Y: -rectDenseSideY + offsetY, NX: -0.35, NY: -1, Priority: 5},
			{X: denseRowXs[2], Y: -rectDenseSideY + offsetY, NX: 0.35, NY: -1, Priority: 5},
			{X: denseRowXs[3], Y: -rectDenseSideY + offsetY, NX: 1, NY: -1, Priority: 1},
			{X: denseRowXs[0], Y: rectDenseSideY + offsetY, NX: -1, NY: 1, Priority: 1},
			{X: denseRowXs[1], Y: rectDenseSideY + offsetY, NX: -0.35, NY: 1, Priority: 5},
			{X: denseRowXs[2], Y: rectDenseSideY + offsetY, NX: 0.35, NY: 1, Priority: 5},
			{X: denseRowXs[3], Y: rectDenseSideY + offsetY, NX: 1, NY: 1, Priority: 1},
		}
		ring := append(append([]rectSlot(nil), corners...), topMid, botMid, leftMid, rightMid)
		denseRing := append(append([]rectSlot(nil), denseCorners...), denseTopMid, denseBotMid, denseLeftMid, denseRightMid)
		return [][]rectSlot{ring, row4x2, denseRing, denseRow4x2}
	}

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

func scoreBoundaryOrderEntries(
	childIDs []string,
	relPos map[string][2]float64,
	demands map[string]childBoundaryDemand,
	desiredOrder func(childBoundaryDemand) (float64, int, bool),
	actualOrder func([2]float64) float64,
) float64 {
	score := 0.0
	for i := 0; i < len(childIDs); i++ {
		leftDemand, ok := demands[childIDs[i]]
		if !ok {
			continue
		}
		leftDesired, leftWeight, leftOK := desiredOrder(leftDemand)
		if !leftOK {
			continue
		}
		leftActual := actualOrder(relPos[childIDs[i]])
		for j := i + 1; j < len(childIDs); j++ {
			rightDemand, ok := demands[childIDs[j]]
			if !ok {
				continue
			}
			rightDesired, rightWeight, rightOK := desiredOrder(rightDemand)
			if !rightOK {
				continue
			}
			deltaDesired := leftDesired - rightDesired
			if math.Abs(deltaDesired) < 24 {
				continue
			}

			rightActual := actualOrder(relPos[childIDs[j]])
			deltaActual := leftActual - rightActual
			if deltaDesired*deltaActual >= 0 {
				continue
			}

			weight := float64(leftWeight + rightWeight)
			score += (math.Abs(deltaDesired)/gridSize + 1) * weight * 220
		}
	}
	return score
}

func scoreChildBoundaryOrder(
	childIDs []string,
	relPos map[string][2]float64,
	demands map[string]childBoundaryDemand,
) float64 {
	if len(demands) == 0 {
		return 0
	}

	score := 0.0
	score += scoreBoundaryOrderEntries(
		childIDs,
		relPos,
		demands,
		func(d childBoundaryDemand) (float64, int, bool) {
			return d.leftOrder, d.leftCount, d.leftCount > 0
		},
		func(pos [2]float64) float64 { return pos[1] },
	)
	score += scoreBoundaryOrderEntries(
		childIDs,
		relPos,
		demands,
		func(d childBoundaryDemand) (float64, int, bool) {
			return d.rightOrder, d.rightCount, d.rightCount > 0
		},
		func(pos [2]float64) float64 { return pos[1] },
	)
	score += scoreBoundaryOrderEntries(
		childIDs,
		relPos,
		demands,
		func(d childBoundaryDemand) (float64, int, bool) {
			return d.topOrder, d.topCount, d.topCount > 0
		},
		func(pos [2]float64) float64 { return pos[0] },
	)
	score += scoreBoundaryOrderEntries(
		childIDs,
		relPos,
		demands,
		func(d childBoundaryDemand) (float64, int, bool) {
			return d.botOrder, d.botCount, d.botCount > 0
		},
		func(pos [2]float64) float64 { return pos[0] },
	)
	return score
}

func childBoundarySideCount(d childBoundaryDemand) int {
	count := 0
	if d.leftCount > 0 {
		count++
	}
	if d.rightCount > 0 {
		count++
	}
	if d.topCount > 0 {
		count++
	}
	if d.botCount > 0 {
		count++
	}
	return count
}

func childDemandHasOrthogonalTraffic(d childBoundaryDemand) bool {
	return (d.leftCount > 0 || d.rightCount > 0) && (d.topCount > 0 || d.botCount > 0)
}

func childIsHubNode(m childRectMetric, d childBoundaryDemand) bool {
	sideCount := childBoundarySideCount(d)
	totalDegree := m.externalDegree + m.internalDegree
	if sideCount >= 3 && m.externalDegree >= 2 {
		return true
	}
	if childDemandHasOrthogonalTraffic(d) && m.externalDegree >= 2 {
		return true
	}
	if m.externalDegree >= 3 && totalDegree >= 4 {
		return true
	}
	if m.externalDegree >= 2 && m.internalIn > 0 && m.internalOut > 0 && totalDegree >= 5 {
		return true
	}
	return false
}

func scoreChildPlacementAffinity(
	childIDs []string,
	relPos map[string][2]float64,
	metrics map[string]childRectMetric,
	demands map[string]childBoundaryDemand,
) float64 {
	score := 0.0
	for _, id := range childIDs {
		pos := relPos[id]
		m := metrics[id]
		d := demands[id]
		weight := math.Max(1, float64(m.externalDegree))

		if d.rightCount > d.leftCount && pos[0] < 0 {
			score += float64(d.rightCount-d.leftCount) * weight * 1200
		}
		if d.leftCount > d.rightCount && pos[0] > 0 {
			score += float64(d.leftCount-d.rightCount) * weight * 1200
		}
		if d.botCount > d.topCount && pos[1] < 0 {
			score += float64(d.botCount-d.topCount) * weight * 700
		}
		if d.topCount > d.botCount && pos[1] > 0 {
			score += float64(d.topCount-d.botCount) * weight * 700
		}

		if m.pull.count > 0 {
			if m.pull.avgX > 120 && pos[0] < 0 {
				score += weight * (1600 + math.Min(320, math.Abs(pos[0])*1.2))
			}
			if m.pull.avgX < -120 && pos[0] > 0 {
				score += weight * (1600 + math.Min(320, math.Abs(pos[0])*1.2))
			}
			if m.pull.avgY > 120 && pos[1] < 0 {
				score += weight * 900
			}
			if m.pull.avgY < -120 && pos[1] > 0 {
				score += weight * 900
			}
		}

		if m.externalOut > 0 && m.externalIn == 0 {
			if m.pull.avgX > 80 && pos[0] < childLayoutNodeW*0.2 {
				score += weight * 1400
			}
			if m.pull.avgX < -80 && pos[0] > -childLayoutNodeW*0.2 {
				score += weight * 1400
			}
		}
		if m.externalIn > 0 && m.externalOut == 0 {
			if m.pull.avgX > 80 && pos[0] > childLayoutNodeW*0.2 {
				score += weight * 900
			}
			if m.pull.avgX < -80 && pos[0] < -childLayoutNodeW*0.2 {
				score += weight * 900
			}
		}

		if childIsHubNode(m, d) {
			centerBandX := childLayoutNodeW * 0.7
			centerBandY := childLayoutNodeH * 0.7
			if math.Abs(pos[0]) < centerBandX {
				score += weight * (2800 + (centerBandX-math.Abs(pos[0]))*18)
			}
			if childDemandHasOrthogonalTraffic(d) && math.Abs(pos[1]) < centerBandY {
				score += weight * (2200 + (centerBandY-math.Abs(pos[1]))*14)
			}
			if math.Abs(pos[0]) < childLayoutNodeW*0.35 && math.Abs(pos[1]) < childLayoutNodeH*0.35 {
				score += weight * 5200
			}
		}
	}
	score += scoreChildBoundaryOrder(childIDs, relPos, demands)
	return score
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

func scoreRectSlotAssignmentWithBoundary(
	id string,
	slot rectSlot,
	metrics map[string]childRectMetric,
	demands map[string]childBoundaryDemand,
	count int,
) float64 {
	score := scoreRectSlotAssignment(id, slot, metrics, count)
	demand, ok := demands[id]
	if !ok {
		return score
	}

	m := metrics[id]
	weight := math.Max(1, float64(m.externalDegree))

	if demand.leftCount+demand.rightCount > 0 {
		desiredY := boundaryDesiredY(id, demands, m.pull, slot.Y)
		score += math.Abs(slot.Y-desiredY) * 0.12
	}
	if demand.topCount+demand.botCount > 0 {
		desiredX := boundaryDesiredX(id, demands, m.pull, slot.X)
		score += math.Abs(slot.X-desiredX) * 0.12
	}

	if demand.leftCount > demand.rightCount && slot.NX > -0.15 {
		score += float64(demand.leftCount-demand.rightCount) * 120 * weight
	}
	if demand.rightCount > demand.leftCount && slot.NX < 0.15 {
		score += float64(demand.rightCount-demand.leftCount) * 120 * weight
	}
	if demand.topCount > demand.botCount && slot.NY > -0.15 {
		score += float64(demand.topCount-demand.botCount) * 120 * weight
	}
	if demand.botCount > demand.topCount && slot.NY < 0.15 {
		score += float64(demand.botCount-demand.topCount) * 120 * weight
	}

	if childIsHubNode(m, demand) {
		if math.Abs(slot.NX) < 0.22 {
			score += 9_000 * weight
		}
		if childDemandHasOrthogonalTraffic(demand) && math.Abs(slot.NY) < 0.22 {
			score += 6_000 * weight
		}
		if math.Abs(slot.NX) < 0.35 && math.Abs(slot.NY) < 0.35 {
			score += 14_000 * weight
		}
		if !isCornerSlot(slot) && math.Max(math.Abs(slot.NX), math.Abs(slot.NY)) < 0.85 {
			score += 2_400 * weight
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

func childInternalEdges(childIDs []string, edges []model.Edge) []model.Edge {
	childSet := make(map[string]bool, len(childIDs))
	for _, id := range childIDs {
		childSet[id] = true
	}
	childEdges := make([]model.Edge, 0, len(edges))
	for _, e := range edges {
		if childSet[e.SourceID] && childSet[e.TargetID] && e.EdgeType != model.EdgeTypeRelated {
			childEdges = append(childEdges, e)
		}
	}
	return childEdges
}

func assignedRectPositions(kids []string, assigned map[string]rectSlot) map[string][2]float64 {
	rel := make(map[string][2]float64, len(kids))
	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			continue
		}
		rel[id] = [2]float64{slot.X, slot.Y}
	}
	return rel
}

func countAssignedChildRouteNodeIntersections(
	kids []string,
	childEdges []model.Edge,
	assigned map[string]rectSlot,
) int {
	rel := assignedRectPositions(kids, assigned)
	total := 0
	for _, e := range childEdges {
		srcPos, ok := rel[e.SourceID]
		if !ok {
			continue
		}
		tgtPos, ok := rel[e.TargetID]
		if !ok {
			continue
		}
		src := Point{X: srcPos[0], Y: srcPos[1]}
		tgt := Point{X: tgtPos[0], Y: tgtPos[1]}
		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		for _, id := range kids {
			if id == e.SourceID || id == e.TargetID {
				continue
			}
			if polylineIntersectsRect(points, childRectBoxAt(rel[id], childRoutePad)) {
				total++
			}
		}
	}
	return total
}

func countAssignedChildRouteCrossings(childEdges []model.Edge, assigned map[string]rectSlot) int {
	type route struct {
		src, tgt string
		points   []Point
	}
	rel := make(map[string][2]float64, len(assigned))
	for id, slot := range assigned {
		rel[id] = [2]float64{slot.X, slot.Y}
	}
	routes := make([]route, 0, len(childEdges))
	for _, e := range childEdges {
		srcPos, ok := rel[e.SourceID]
		if !ok {
			continue
		}
		tgtPos, ok := rel[e.TargetID]
		if !ok {
			continue
		}
		src := Point{X: srcPos[0], Y: srcPos[1]}
		tgt := Point{X: tgtPos[0], Y: tgtPos[1]}
		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		routes = append(routes, route{src: e.SourceID, tgt: e.TargetID, points: points})
	}
	total := 0
	for i := 0; i < len(routes); i++ {
		a := routes[i]
		for j := i + 1; j < len(routes); j++ {
			b := routes[j]
			if a.src == b.src || a.src == b.tgt || a.tgt == b.src || a.tgt == b.tgt {
				continue
			}
			crossed := false
			for ai := 0; ai < len(a.points)-1 && !crossed; ai++ {
				for bi := 0; bi < len(b.points)-1; bi++ {
					a1, a2 := a.points[ai], a.points[ai+1]
					b1, b2 := b.points[bi], b.points[bi+1]
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
			}
		}
	}
	return total
}

func routeHasDiagonalSegment(points []Point) bool {
	for i := 0; i < len(points)-1; i++ {
		dx := math.Abs(points[i+1].X - points[i].X)
		dy := math.Abs(points[i+1].Y - points[i].Y)
		if dx > 0.75 && dy > 0.75 {
			return true
		}
	}
	return false
}

func countAssignedRouteCorridorPenalty(
	kids []string,
	routes []childPredictedRouteInfo,
	rel map[string][2]float64,
	nodePad float64,
	headerPad float64,
) float64 {
	if len(kids) == 0 || len(routes) == 0 {
		return 0
	}

	penalty := 0.0
	for _, route := range routes {
		diagonal := routeHasDiagonalSegment(route.points)
		nodeHitWeight := 18_000.0
		headerHitWeight := 10_000.0
		if diagonal {
			nodeHitWeight = 34_000.0
			headerHitWeight = 18_000.0
		}

		for _, id := range kids {
			if id == route.srcID || id == route.tgtID {
				continue
			}
			if polylineIntersectsRect(route.points, childRectBoxAt(rel[id], nodePad)) {
				penalty += nodeHitWeight
			}
		}

		if countPredictedChildHeaderIntersectionsWithPadding(
			kids,
			[]childPredictedRouteInfo{route},
			rel,
			childLayoutNodeW,
			childLayoutNodeH,
			headerPad,
		) > 0 {
			penalty += headerHitWeight
		}
	}

	return penalty
}

func assignedRouteCorridorPenalty(
	kids []string,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
	assigned map[string]rectSlot,
) float64 {
	rel := assignedRectPositions(kids, assigned)
	internalRoutes := buildChildPredictedRouteInfosFromRelPos(childEdges, rel)
	externalRoutes := buildChildExternalRouteInfosFromRelPos(kids, externalLinks, rel)
	return countAssignedRouteCorridorPenalty(kids, internalRoutes, rel, childCorridorPad, childHeaderCorridorPad) +
		countAssignedRouteCorridorPenalty(kids, externalRoutes, rel, childCorridorPad, childHeaderCorridorPad)
}

func scoreRectangularAssignment(
	kids []string,
	assigned map[string]rectSlot,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
) float64 {
	if len(assigned) == 0 {
		return math.MaxFloat64
	}
	rel := assignedRectPositions(kids, assigned)
	centerCount := 0
	for _, id := range kids {
		slot := assigned[id]
		if math.Abs(slot.NX) < 0.25 && math.Abs(slot.NY) < 0.25 {
			centerCount++
		}
	}
	return childLayoutCost(kids, childEdges, externalLinks, rel) +
		float64(centerCount)*2_000 +
		assignedRouteCorridorPenalty(kids, childEdges, externalLinks, assigned)
}

func scoreRectangularBoundaryOrdering(
	kids []string,
	assigned map[string]rectSlot,
	metrics map[string]childRectMetric,
	demands map[string]childBoundaryDemand,
) float64 {
	rows := make(map[float64][]string)
	cols := make(map[float64][]string)
	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			continue
		}
		rows[slot.Y] = append(rows[slot.Y], id)
		cols[slot.X] = append(cols[slot.X], id)
	}

	score := 0.0
	for _, ids := range rows {
		if len(ids) < 2 {
			continue
		}
		sort.SliceStable(ids, func(i, j int) bool {
			return assigned[ids[i]].X < assigned[ids[j]].X
		})
		for i := 0; i < len(ids); i++ {
			leftID := ids[i]
			leftDesiredX := boundaryDesiredX(leftID, demands, metrics[leftID].pull, assigned[leftID].X)
			for j := i + 1; j < len(ids); j++ {
				rightID := ids[j]
				rightDesiredX := boundaryDesiredX(rightID, demands, metrics[rightID].pull, assigned[rightID].X)
				if leftDesiredX <= rightDesiredX {
					continue
				}
				score += 260 + (leftDesiredX-rightDesiredX)*0.35
			}
		}
	}

	for _, ids := range cols {
		if len(ids) < 2 {
			continue
		}
		sort.SliceStable(ids, func(i, j int) bool {
			return assigned[ids[i]].Y < assigned[ids[j]].Y
		})
		for i := 0; i < len(ids); i++ {
			topID := ids[i]
			topDesiredY := boundaryDesiredY(topID, demands, metrics[topID].pull, assigned[topID].Y)
			for j := i + 1; j < len(ids); j++ {
				bottomID := ids[j]
				bottomDesiredY := boundaryDesiredY(bottomID, demands, metrics[bottomID].pull, assigned[bottomID].Y)
				if topDesiredY <= bottomDesiredY {
					continue
				}
				score += 220 + (topDesiredY-bottomDesiredY)*0.3
			}
		}
	}

	return score
}

func shouldUsePassThroughChildLayout(
	kids []string,
	childEdges []model.Edge,
	metrics map[string]childRectMetric,
) bool {
	if len(kids) < 4 || len(kids) > 8 {
		return false
	}
	if len(childEdges) == 0 {
		return false
	}

	horizontalExternal := 0
	verticalExternal := 0
	inCount := 0
	outCount := 0
	bridgeCount := 0
	for _, id := range kids {
		m := metrics[id]
		if m.externalDegree == 0 {
			continue
		}
		if math.Abs(m.pull.avgX) >= math.Abs(m.pull.avgY)*1.1 {
			horizontalExternal += m.externalDegree
		} else {
			verticalExternal += m.externalDegree
		}
		if m.externalIn > 0 {
			inCount++
		}
		if m.externalOut > 0 {
			outCount++
		}
		if (m.externalIn > 0 && (m.externalOut > 0 || m.internalOut > 0)) ||
			(m.externalOut > 0 && (m.externalIn > 0 || m.internalIn > 0)) {
			bridgeCount++
		}
	}

	return horizontalExternal >= max(4, verticalExternal*2) && inCount > 0 && outCount > 0 && bridgeCount > 0
}

func childExternalAxisStats(kids []string, metrics map[string]childRectMetric) (horizontal, vertical, leftish, rightish, upish, downish int) {
	for _, id := range kids {
		m := metrics[id]
		if m.externalDegree == 0 {
			continue
		}
		if math.Abs(m.pull.avgX) >= math.Abs(m.pull.avgY)*1.1 {
			horizontal += m.externalDegree
		} else {
			vertical += m.externalDegree
		}
		if m.pull.avgX < -80 || m.externalIn > m.externalOut {
			leftish++
		}
		if m.pull.avgX > 80 || m.externalOut > m.externalIn {
			rightish++
		}
		if m.pull.avgY < -80 {
			upish++
		}
		if m.pull.avgY > 80 {
			downish++
		}
	}
	return
}

func childLineStructureIsSparse(kids []string, childEdges []model.Edge, metrics map[string]childRectMetric) bool {
	if len(childEdges) > max(1, len(kids)-1) {
		return false
	}

	branching := 0
	activeInternal := 0
	for _, id := range kids {
		m := metrics[id]
		if m.internalDegree > 0 {
			activeInternal++
		}
		if m.internalDegree >= 3 {
			branching++
		}
	}

	if branching > 1 {
		return false
	}
	if len(childEdges) >= len(kids)-1 && activeInternal >= len(kids)-1 && branching > 0 {
		return false
	}
	return true
}

func shouldUseVerticalLineChildLayout(
	kids []string,
	childEdges []model.Edge,
	metrics map[string]childRectMetric,
) bool {
	if len(kids) < 3 || len(kids) > 6 {
		return false
	}
	if !childLineStructureIsSparse(kids, childEdges, metrics) {
		return false
	}
	horizontal, vertical, leftish, rightish, _, _ := childExternalAxisStats(kids, metrics)
	if horizontal < max(3, vertical*2+1) {
		return false
	}
	return leftish+rightish >= max(2, len(kids)/2)
}

func shouldUseTwoColumnFlowChildLayout(
	kids []string,
	childEdges []model.Edge,
	metrics map[string]childRectMetric,
	demands map[string]childBoundaryDemand,
) bool {
	if len(kids) < 4 || len(kids) > 7 {
		return false
	}
	if len(childEdges) == 0 || len(childEdges) > len(kids)*2 {
		return false
	}

	horizontal, vertical, leftish, rightish, _, _ := childExternalAxisStats(kids, metrics)
	if horizontal < max(3, vertical) {
		return false
	}

	bridgeOrMain := 0
	rightSink := 0
	for _, id := range kids {
		m := metrics[id]
		d := demands[id]
		isBridge := m.internalIn > 0 && m.internalOut > 0
		isMainFlow := isBridge || m.internalOut > 0 || m.externalIn > m.externalOut || d.leftCount > d.rightCount
		isRightSink := (m.internalIn > 0 && m.internalOut == 0) || (d.rightCount > d.leftCount && m.internalOut == 0) || (m.externalOut > m.externalIn && m.internalOut == 0)
		if isMainFlow {
			bridgeOrMain++
		}
		if isRightSink {
			rightSink++
		}
	}

	return bridgeOrMain >= 2 && rightSink >= 1 && leftish >= 1 && rightish >= 1
}

func shouldUseHorizontalLineChildLayout(
	kids []string,
	childEdges []model.Edge,
	metrics map[string]childRectMetric,
) bool {
	if len(kids) < 3 || len(kids) > 6 {
		return false
	}
	if !childLineStructureIsSparse(kids, childEdges, metrics) {
		return false
	}
	horizontal, vertical, _, _, upish, downish := childExternalAxisStats(kids, metrics)
	if vertical < max(3, horizontal*2+1) {
		return false
	}
	return upish+downish >= max(2, len(kids)/2)
}

func centeredAxisPositions(count int, step, offset float64) []float64 {
	if count <= 0 {
		return nil
	}
	positions := make([]float64, count)
	base := float64(count-1) / 2
	for i := 0; i < count; i++ {
		positions[i] = offset + (float64(i)-base)*step
	}
	return positions
}

func verticalLineSlotPools(count int) [][]rectSlot {
	offsetY := (groupPadTop - groupPadBot) / 2
	makeMain := func(x float64, rows []float64, axisStep float64, priority float64) []rectSlot {
		slots := make([]rectSlot, 0, len(rows))
		for _, y := range rows {
			ny := 0.0
			if offsetY != y {
				ny = math.Max(-1, math.Min(1, (y-offsetY)/(axisStep*1.5)))
			}
			slots = append(slots, rectSlot{X: x, Y: y, NX: 0, NY: ny, Priority: priority})
		}
		return slots
	}
	makeSide := func(x float64, rows []float64, axisStep float64, priority float64) []rectSlot {
		slots := make([]rectSlot, 0, len(rows))
		nx := 1.0
		if x < 0 {
			nx = -1
		}
		for _, y := range rows {
			ny := 0.0
			if offsetY != y {
				ny = math.Max(-1, math.Min(1, (y-offsetY)/(axisStep*1.5)))
			}
			slots = append(slots, rectSlot{X: x, Y: y, NX: nx, NY: ny, Priority: priority})
		}
		return slots
	}

	pureRows := centeredAxisPositions(count, lineAxisStep, offsetY)
	pure := makeMain(0, pureRows, lineAxisStep, 2)

	mainCount := count - 1
	if mainCount < 3 {
		mainCount = count
	}
	mainRows := centeredAxisPositions(mainCount, lineAxisStep, offsetY)
	sideRows := centeredAxisPositions(min(2, count-1), lineAxisStep*1.2, offsetY)
	right := append(makeMain(-lineMainOffset, mainRows, lineAxisStep, 2), makeSide(lineSidecarOffset, sideRows, lineAxisStep, 8)...)
	left := append(makeMain(lineMainOffset, mainRows, lineAxisStep, 2), makeSide(-lineSidecarOffset, sideRows, lineAxisStep, 8)...)
	boundaryRight := append(makeMain(lineBoundaryMainX, mainRows, lineAxisStep, 2), makeSide(-lineBoundarySideX, sideRows, lineAxisStep, 8)...)
	boundaryLeft := append(makeMain(-lineBoundaryMainX, mainRows, lineAxisStep, 2), makeSide(lineBoundarySideX, sideRows, lineAxisStep, 8)...)

	compactPureRows := centeredAxisPositions(count, lineCompactAxisStep, offsetY)
	compactPure := makeMain(0, compactPureRows, lineCompactAxisStep, 3)
	compactMainRows := centeredAxisPositions(mainCount, lineCompactAxisStep, offsetY)
	compactSideRows := centeredAxisPositions(min(2, count-1), lineCompactAxisStep*1.05, offsetY)
	compactRight := append(makeMain(-lineCompactMainOffset, compactMainRows, lineCompactAxisStep, 3), makeSide(lineCompactSidecarOffset, compactSideRows, lineCompactAxisStep, 9)...)
	compactLeft := append(makeMain(lineCompactMainOffset, compactMainRows, lineCompactAxisStep, 3), makeSide(-lineCompactSidecarOffset, compactSideRows, lineCompactAxisStep, 9)...)
	compactBoundaryRight := append(makeMain(lineCompactBoundaryMainX, compactMainRows, lineCompactAxisStep, 3), makeSide(-lineCompactBoundarySideX, compactSideRows, lineCompactAxisStep, 9)...)
	compactBoundaryLeft := append(makeMain(-lineCompactBoundaryMainX, compactMainRows, lineCompactAxisStep, 3), makeSide(lineCompactBoundarySideX, compactSideRows, lineCompactAxisStep, 9)...)

	return [][]rectSlot{boundaryRight, boundaryLeft, right, left, pure, compactBoundaryRight, compactBoundaryLeft, compactRight, compactLeft, compactPure}
}

func twoColumnFlowSlotPools(count int) [][]rectSlot {
	offsetY := (groupPadTop - groupPadBot) / 2
	makeSlots := func(leftCount, rightCount int, staggerRight bool, leftX, rightX, stepY, basePriority float64) []rectSlot {
		leftRows := centeredAxisPositions(leftCount, stepY, offsetY)
		rightOffset := offsetY
		if staggerRight {
			rightOffset += stepY * 0.35
		}
		rightRows := centeredAxisPositions(rightCount, stepY, rightOffset)
		slots := make([]rectSlot, 0, leftCount+rightCount)
		for _, y := range leftRows {
			ny := 0.0
			if y != offsetY {
				ny = math.Max(-1, math.Min(1, (y-offsetY)/(stepY*1.5)))
			}
			slots = append(slots, rectSlot{X: leftX, Y: y, NX: -1, NY: ny, Priority: basePriority})
		}
		for _, y := range rightRows {
			ny := 0.0
			if y != offsetY {
				ny = math.Max(-1, math.Min(1, (y-offsetY)/(stepY*1.5)))
			}
			slots = append(slots, rectSlot{X: rightX, Y: y, NX: 1, NY: ny, Priority: basePriority + 2})
		}
		return slots
	}
	makeSlotsFromRows := func(leftRows, rightRows []float64, leftX, rightX, stepY, basePriority float64) []rectSlot {
		slots := make([]rectSlot, 0, len(leftRows)+len(rightRows))
		for _, y := range leftRows {
			ny := 0.0
			if y != offsetY {
				ny = math.Max(-1, math.Min(1, (y-offsetY)/(stepY*1.5)))
			}
			slots = append(slots, rectSlot{X: leftX, Y: y, NX: -1, NY: ny, Priority: basePriority})
		}
		for _, y := range rightRows {
			ny := 0.0
			if y != offsetY {
				ny = math.Max(-1, math.Min(1, (y-offsetY)/(stepY*1.5)))
			}
			slots = append(slots, rectSlot{X: rightX, Y: y, NX: 1, NY: ny, Priority: basePriority + 2})
		}
		return slots
	}

	switch count {
	case 4:
		return [][]rectSlot{
			makeSlots(3, 1, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(2, 2, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(3, 1, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
			makeSlots(2, 2, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
		}
	case 5:
		leftRows := centeredAxisPositions(3, twoColumnStepY, offsetY)
		compactLeftRows := centeredAxisPositions(3, twoColumnCompactStepY, offsetY)
		return [][]rectSlot{
			makeSlots(3, 2, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlotsFromRows(leftRows, []float64{leftRows[0], leftRows[2]}, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(4, 1, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(3, 2, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
			makeSlotsFromRows(compactLeftRows, []float64{compactLeftRows[0], compactLeftRows[2]}, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
			makeSlots(4, 1, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
		}
	case 6:
		return [][]rectSlot{
			makeSlots(4, 2, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(3, 3, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(4, 2, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
			makeSlots(3, 3, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
		}
	case 7:
		return [][]rectSlot{
			makeSlots(4, 3, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(5, 2, true, twoColumnLeftX, twoColumnRightX, twoColumnStepY, 2),
			makeSlots(4, 3, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
			makeSlots(5, 2, true, twoColumnCompactLeftX, twoColumnCompactRightX, twoColumnCompactStepY, 3),
		}
	default:
		return nil
	}
}

func horizontalLineSlotPools(count int) [][]rectSlot {
	offsetY := (groupPadTop - groupPadBot) / 2
	makeMain := func(y float64, cols []float64) []rectSlot {
		slots := make([]rectSlot, 0, len(cols))
		for _, x := range cols {
			nx := 0.0
			if x != 0 {
				nx = math.Max(-1, math.Min(1, x/(lineAxisStep*1.5)))
			}
			slots = append(slots, rectSlot{X: x, Y: y, NX: nx, NY: 0, Priority: 2})
		}
		return slots
	}
	makeSide := func(y float64, cols []float64) []rectSlot {
		slots := make([]rectSlot, 0, len(cols))
		ny := 1.0
		if y < offsetY {
			ny = -1
		}
		for _, x := range cols {
			nx := 0.0
			if x != 0 {
				nx = math.Max(-1, math.Min(1, x/(lineAxisStep*1.5)))
			}
			slots = append(slots, rectSlot{X: x, Y: y, NX: nx, NY: ny, Priority: 8})
		}
		return slots
	}

	pureCols := centeredAxisPositions(count, lineAxisStep, 0)
	pure := makeMain(offsetY, pureCols)

	mainCount := count - 1
	if mainCount < 3 {
		mainCount = count
	}
	mainCols := centeredAxisPositions(mainCount, lineAxisStep, 0)
	sideCols := centeredAxisPositions(min(2, count-1), lineAxisStep*1.2, 0)
	bottom := append(makeMain(offsetY-lineMainOffset, mainCols), makeSide(offsetY+lineSidecarOffset, sideCols)...)
	top := append(makeMain(offsetY+lineMainOffset, mainCols), makeSide(offsetY-lineSidecarOffset, sideCols)...)

	return [][]rectSlot{bottom, top, pure}
}

func scoreVerticalLineAssignment(id string, slot rectSlot, metrics map[string]childRectMetric, demands map[string]childBoundaryDemand) float64 {
	m := metrics[id]
	score := slot.Priority
	extWeight := math.Max(1, float64(m.externalDegree))
	isSidecar := math.Abs(slot.NX) > 0.6

	if isSidecar {
		if slot.NX > 0 && !(m.pull.avgX > 80 || m.externalOut > m.externalIn) {
			score += 150 * extWeight
		}
		if slot.NX < 0 && !(m.pull.avgX < -80 || m.externalIn > m.externalOut) {
			score += 150 * extWeight
		}
		if m.internalDegree > 1 {
			score += 80
		}
	} else {
		if m.pull.avgX > 120 && slot.X < 0 {
			score += 220 * extWeight
		}
		if m.pull.avgX < -120 && slot.X > 0 {
			score += 220 * extWeight
		}
		if math.Abs(m.pull.avgX) > 180 && m.externalDegree > 0 {
			score += 24 * extWeight
		}
		if m.internalDegree > 0 {
			score -= 8
		}
	}

	desiredY := boundaryDesiredY(id, demands, m.pull, slot.Y)
	score += math.Abs(slot.Y-desiredY) * 0.16
	desiredX := boundaryDesiredX(id, demands, m.pull, slot.X)
	score += math.Abs(slot.X-desiredX) * 0.22
	return score
}

func scoreHorizontalLineAssignment(id string, slot rectSlot, metrics map[string]childRectMetric, demands map[string]childBoundaryDemand) float64 {
	m := metrics[id]
	score := slot.Priority
	extWeight := math.Max(1, float64(m.externalDegree))
	isSidecar := math.Abs(slot.NY) > 0.6

	if isSidecar {
		if slot.NY > 0 && !(m.pull.avgY > 80 || m.externalOut > m.externalIn) {
			score += 150 * extWeight
		}
		if slot.NY < 0 && !(m.pull.avgY < -80 || m.externalIn > m.externalOut) {
			score += 150 * extWeight
		}
		if m.internalDegree > 1 {
			score += 80
		}
	} else {
		if math.Abs(m.pull.avgY) > 180 && m.externalDegree > 0 {
			score += 24 * extWeight
		}
		if m.internalDegree > 0 {
			score -= 8
		}
	}

	desiredX := boundaryDesiredX(id, demands, m.pull, slot.X)
	score += math.Abs(slot.X-desiredX) * 0.16
	return score
}

func scoreTwoColumnFlowAssignment(id string, slot rectSlot, metrics map[string]childRectMetric, demands map[string]childBoundaryDemand) float64 {
	m := metrics[id]
	d := demands[id]
	score := slot.Priority
	leftAffinity := float64(d.leftCount*3 + m.externalIn*2 + m.internalOut*2)
	rightAffinity := float64(d.rightCount*3 + m.externalOut*2)

	if m.pull.avgX < -80 {
		leftAffinity += 2
	}
	if m.pull.avgX > 80 {
		rightAffinity += 2
	}
	if m.internalIn > 0 && m.internalOut == 0 {
		rightAffinity += 3
	}
	if m.internalIn > 0 && m.internalOut > 0 {
		leftAffinity += 2
	}

	if slot.NX < 0 {
		if rightAffinity > leftAffinity {
			score += (rightAffinity - leftAffinity) * 80
		}
		if m.internalIn > 0 && m.internalOut == 0 && (d.rightCount > d.leftCount || m.externalOut > m.externalIn) {
			score += 120
		}
		desiredY := boundaryDesiredY(id, demands, m.pull, slot.Y)
		score += math.Abs(slot.Y-desiredY) * 0.14
		if m.internalDegree > 0 {
			score -= 8
		}
	} else {
		if leftAffinity > rightAffinity {
			score += (leftAffinity - rightAffinity) * 90
		}
		if m.externalIn > 0 && m.externalOut == 0 {
			score += 140
		}
		if m.internalOut > 0 {
			score += 70
		}
		desiredY := boundaryDesiredY(id, demands, m.pull, slot.Y)
		score += math.Abs(slot.Y-desiredY) * 0.11
		if m.internalIn > 0 && m.internalOut == 0 {
			score -= 12
		}
		if m.internalIn > 0 && m.internalOut > 0 {
			score += 18
		}
	}

	return score
}

func scoreLineAssignmentShape(
	kids []string,
	assigned map[string]rectSlot,
) float64 {
	sideLeft := 0
	sideRight := 0
	sideTop := 0
	sideBottom := 0
	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			continue
		}
		switch {
		case slot.NX > 0.6:
			sideRight++
		case slot.NX < -0.6:
			sideLeft++
		case slot.NY > 0.6:
			sideBottom++
		case slot.NY < -0.6:
			sideTop++
		}
	}
	score := 0.0
	if sideLeft > 0 && sideRight > 0 {
		score += 2_000
	}
	if sideTop > 0 && sideBottom > 0 {
		score += 2_000
	}
	if sideLeft > 2 {
		score += float64(sideLeft-2) * 600
	}
	if sideRight > 2 {
		score += float64(sideRight-2) * 600
	}
	if sideTop > 2 {
		score += float64(sideTop-2) * 600
	}
	if sideBottom > 2 {
		score += float64(sideBottom-2) * 600
	}
	return score
}

func scoreDominantHorizontalBoundaryShape(
	kids []string,
	assigned map[string]rectSlot,
	metrics map[string]childRectMetric,
) float64 {
	leftPull := 0.0
	rightPull := 0.0
	for _, id := range kids {
		m := metrics[id]
		if m.externalDegree == 0 {
			continue
		}
		weight := math.Max(1, float64(m.externalDegree))
		switch {
		case m.pull.avgX > 80:
			rightPull += weight
		case m.pull.avgX < -80:
			leftPull += weight
		}
	}

	if rightPull < leftPull*1.4 && leftPull < rightPull*1.4 {
		return 0
	}

	dominantRight := rightPull > leftPull
	score := 0.0
	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			continue
		}
		m := metrics[id]
		if m.externalDegree == 0 {
			continue
		}
		weight := math.Max(1, float64(m.externalDegree))
		if dominantRight && slot.X < 0 {
			score += 420 * weight
		}
		if !dominantRight && slot.X > 0 {
			score += 420 * weight
		}
	}
	return score
}

func scoreTwoColumnFlowShape(
	kids []string,
	assigned map[string]rectSlot,
	metrics map[string]childRectMetric,
) float64 {
	leftCount := 0
	rightCount := 0
	leftOutgoing := 0
	rightSinks := 0
	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			continue
		}
		m := metrics[id]
		if slot.NX < 0 {
			leftCount++
			if m.internalOut > 0 {
				leftOutgoing++
			}
		} else {
			rightCount++
			if m.internalIn > 0 && m.internalOut == 0 {
				rightSinks++
			}
		}
	}
	score := 0.0
	if leftCount == 0 || rightCount == 0 {
		score += 20_000
	}
	if leftCount < rightCount {
		score += float64(rightCount-leftCount) * 2_500
	}
	if leftOutgoing == 0 {
		score += 4_000
	}
	if rightSinks == 0 {
		score += 4_000
	}
	return score
}

func bestCustomSlotAssignmentWithCost(
	kids []string,
	slots []rectSlot,
	scoreFn func(string, rectSlot) float64,
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
			cost := scoreFn(kids[index], slots[slotIdx]) + solve(index+1, mask|(1<<slotIdx))
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

type childSlotExtraCostFn func(map[string]rectSlot, map[string][2]float64) float64

func cloneAssignedSlots(assigned map[string]rectSlot) map[string]rectSlot {
	cloned := make(map[string]rectSlot, len(assigned))
	for id, slot := range assigned {
		cloned[id] = slot
	}
	return cloned
}

func rectSlotSideBucket(slot rectSlot) string {
	switch {
	case slot.NX > 0.6:
		return "right"
	case slot.NX < -0.6:
		return "left"
	case slot.NY > 0.6:
		return "bottom"
	case slot.NY < -0.6:
		return "top"
	case math.Abs(slot.NX) >= math.Abs(slot.NY):
		if slot.NX >= 0 {
			return "center-right"
		}
		return "center-left"
	default:
		if slot.NY >= 0 {
			return "center-bottom"
		}
		return "center-top"
	}
}

func sameSideSlots(a, b rectSlot) bool {
	return rectSlotSideBucket(a) == rectSlotSideBucket(b)
}

func childSlotAssignmentCost(
	kids []string,
	assigned map[string]rectSlot,
	slotScoreFn func(string, rectSlot) float64,
	extraCostFn childSlotExtraCostFn,
) float64 {
	total := 0.0
	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			return math.MaxFloat64
		}
		total += slotScoreFn(id, slot)
	}
	if extraCostFn != nil {
		total += extraCostFn(assigned, assignedRectPositions(kids, assigned))
	}
	return total
}

func refineChildSlotAssignment(
	kids []string,
	pool []rectSlot,
	assigned map[string]rectSlot,
	slotScoreFn func(string, rectSlot) float64,
	extraCostFn childSlotExtraCostFn,
) (map[string]rectSlot, float64) {
	if len(kids) <= 1 || len(assigned) != len(kids) {
		return assigned, childSlotAssignmentCost(kids, assigned, slotScoreFn, extraCostFn)
	}

	current := cloneAssignedSlots(assigned)
	bestCost := childSlotAssignmentCost(kids, current, slotScoreFn, extraCostFn)

	for iter := 0; iter < 24; iter++ {
		improved := false

		phaseBest := bestCost
		var phaseAssigned map[string]rectSlot
		for i := 0; i < len(kids); i++ {
			for j := i + 1; j < len(kids); j++ {
				aID, bID := kids[i], kids[j]
				aSlot, bSlot := current[aID], current[bID]
				if !sameSideSlots(aSlot, bSlot) {
					continue
				}
				trial := cloneAssignedSlots(current)
				trial[aID], trial[bID] = bSlot, aSlot
				cost := childSlotAssignmentCost(kids, trial, slotScoreFn, extraCostFn)
				if cost+1e-6 < phaseBest {
					phaseBest = cost
					phaseAssigned = trial
				}
			}
		}
		if phaseAssigned != nil {
			current = phaseAssigned
			bestCost = phaseBest
			improved = true
			continue
		}

		if !improved {
			break
		}
	}

	return current, bestCost
}

func passThroughSlotPools(count int) [][]rectSlot {
	offsetY := (groupPadTop - groupPadBot) / 2
	upperY := offsetY - math.Max(passThroughLaneGap+childLayoutNodeH/2, rectCompactSideY+8)
	lowerY := offsetY + math.Max(passThroughLaneGap+childLayoutNodeH/2, rectCompactSideY+8)

	makeSlots := func(xsUpper, xsLower []float64) []rectSlot {
		maxAbs := 1.0
		for _, x := range xsUpper {
			if math.Abs(x) > maxAbs {
				maxAbs = math.Abs(x)
			}
		}
		for _, x := range xsLower {
			if math.Abs(x) > maxAbs {
				maxAbs = math.Abs(x)
			}
		}
		slots := make([]rectSlot, 0, len(xsUpper)+len(xsLower))
		for _, x := range xsUpper {
			slots = append(slots, rectSlot{X: x, Y: upperY, NX: x / maxAbs, NY: -1, Priority: 4})
		}
		for _, x := range xsLower {
			slots = append(slots, rectSlot{X: x, Y: lowerY, NX: x / maxAbs, NY: 1, Priority: 2})
		}
		return slots
	}

	switch count {
	case 4:
		return [][]rectSlot{
			makeSlots([]float64{-passThroughStep, passThroughStep}, []float64{-passThroughStep, passThroughStep}),
		}
	case 5:
		return [][]rectSlot{
			makeSlots([]float64{-passThroughStep, passThroughStep}, []float64{-1.6 * passThroughStep, 0, 1.6 * passThroughStep}),
			makeSlots([]float64{-1.6 * passThroughStep, 0, 1.6 * passThroughStep}, []float64{-passThroughStep, passThroughStep}),
		}
	case 6:
		return [][]rectSlot{
			makeSlots([]float64{-1.8 * passThroughStep, 0, 1.8 * passThroughStep}, []float64{-1.8 * passThroughStep, 0, 1.8 * passThroughStep}),
		}
	case 7:
		return [][]rectSlot{
			makeSlots([]float64{-2.3 * passThroughStep, -0.8 * passThroughStep, 0.8 * passThroughStep, 2.3 * passThroughStep}, []float64{-1.6 * passThroughStep, 0, 1.6 * passThroughStep}),
			makeSlots([]float64{-1.6 * passThroughStep, 0, 1.6 * passThroughStep}, []float64{-2.3 * passThroughStep, -0.8 * passThroughStep, 0.8 * passThroughStep, 2.3 * passThroughStep}),
		}
	case 8:
		return [][]rectSlot{
			makeSlots([]float64{-2.3 * passThroughStep, -0.8 * passThroughStep, 0.8 * passThroughStep, 2.3 * passThroughStep}, []float64{-2.3 * passThroughStep, -0.8 * passThroughStep, 0.8 * passThroughStep, 2.3 * passThroughStep}),
		}
	default:
		return nil
	}
}

func scorePassThroughAssignment(id string, slot rectSlot, metrics map[string]childRectMetric) float64 {
	m := metrics[id]
	score := slot.Priority
	extWeight := math.Max(1, float64(m.externalDegree))

	if math.Abs(m.pull.avgX) > 80 {
		if m.pull.avgX > 0 && slot.NX < 0 {
			score += 120 * extWeight
		}
		if m.pull.avgX < 0 && slot.NX > 0 {
			score += 120 * extWeight
		}
		score += (1 - math.Abs(slot.NX)) * 60 * extWeight
	}
	if math.Abs(m.pull.avgY) > 80 {
		if m.pull.avgY > 0 && slot.NY < 0 {
			score += 40 * extWeight
		}
		if m.pull.avgY < 0 && slot.NY > 0 {
			score += 40 * extWeight
		}
	}

	if (m.externalIn > 0 && m.externalOut > 0) || (m.externalDegree > 0 && m.internalIn > 0 && m.internalOut > 0) {
		score += math.Abs(slot.X) * 0.25
	}
	if m.externalDegree == 0 && m.internalDegree > 0 {
		score += math.Abs(slot.X) * 0.12
	}
	return score
}

func scorePassThroughAssignmentWithBoundary(id string, slot rectSlot, metrics map[string]childRectMetric, demands map[string]childBoundaryDemand) float64 {
	score := scorePassThroughAssignment(id, slot, metrics)
	m := metrics[id]
	demand, ok := demands[id]
	if !ok {
		return score
	}
	if demand.leftCount+demand.rightCount > 0 {
		desiredY := boundaryDesiredY(id, demands, m.pull, slot.Y)
		score += math.Abs(slot.Y-desiredY) * 0.12
	}
	if demand.topCount+demand.botCount > 0 {
		desiredX := boundaryDesiredX(id, demands, m.pull, slot.X)
		score += math.Abs(slot.X-desiredX) * 0.06
	}
	return score
}

func bestPassThroughSlotAssignmentWithCost(
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
			cost := scorePassThroughAssignment(kids[index], slots[slotIdx], metrics) + solve(index+1, mask|(1<<slotIdx))
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

func shouldUseExternalPullBoundaryLayout(
	kids []string,
	childEdges []model.Edge,
	pulls map[string]childExternalPull,
) bool {
	if len(kids) < 5 || len(kids) > 7 || len(childEdges) > len(kids) {
		return false
	}
	pulledKids := 0
	totalExternal := 0
	for _, id := range kids {
		pull, ok := pulls[id]
		if !ok || pull.count == 0 {
			continue
		}
		pulledKids++
		totalExternal += pull.count
	}
	if pulledKids < len(kids)/2 {
		return false
	}
	return totalExternal >= len(kids)
}

func layoutChildrenPassThroughCorridor(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
	demands map[string]childBoundaryDemand,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
) (float64, float64, bool) {
	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	if !shouldUsePassThroughChildLayout(childIDs, childEdges, metrics) {
		return 0, 0, false
	}

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
	for _, pool := range passThroughSlotPools(len(kids)) {
		if len(pool) < len(kids) {
			continue
		}
		slotScoreFn := func(id string, slot rectSlot) float64 {
			return scorePassThroughAssignmentWithBoundary(id, slot, metrics, demands)
		}
		cand, _ := bestCustomSlotAssignmentWithCost(kids, pool, slotScoreFn)
		if len(cand) != len(kids) {
			continue
		}
		cand, cost := refineChildSlotAssignment(kids, pool, cand, slotScoreFn, func(assigned map[string]rectSlot, rel map[string][2]float64) float64 {
			return scoreRectangularAssignment(kids, assigned, childEdges, externalLinks)
		})
		if cost < bestCost {
			bestCost = cost
			assigned = cand
		}
	}
	if assigned == nil {
		return 0, 0, false
	}

	for _, id := range kids {
		slot := assigned[id]
		relPos[id] = [2]float64{slot.X, slot.Y}
	}
	groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, relPos)
	return groupW, groupH, true
}

func layoutChildrenTwoColumnFlow(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
	demands map[string]childBoundaryDemand,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
) (float64, float64, bool) {
	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	if !shouldUseTwoColumnFlowChildLayout(childIDs, childEdges, metrics, demands) {
		return 0, 0, false
	}

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
	for _, pool := range twoColumnFlowSlotPools(len(kids)) {
		if len(pool) < len(kids) {
			continue
		}
		cand, cost := bestCustomSlotAssignmentWithCost(kids, pool, func(id string, slot rectSlot) float64 {
			return scoreTwoColumnFlowAssignment(id, slot, metrics, demands)
		})
		if len(cand) != len(kids) {
			continue
		}
		cost += scoreRectangularAssignment(kids, cand, childEdges, externalLinks)
		cost += scoreTwoColumnFlowShape(kids, cand, metrics)
		if cost < bestCost {
			bestCost = cost
			assigned = cand
		}
	}
	if assigned == nil {
		return 0, 0, false
	}

	for _, id := range kids {
		slot := assigned[id]
		relPos[id] = [2]float64{slot.X, slot.Y}
	}
	groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, relPos)
	return groupW, groupH, true
}

func layoutChildrenVerticalLine(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
	demands map[string]childBoundaryDemand,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
) (float64, float64, bool) {
	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	if !shouldUseVerticalLineChildLayout(childIDs, childEdges, metrics) {
		return 0, 0, false
	}

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
	for _, pool := range verticalLineSlotPools(len(kids)) {
		if len(pool) < len(kids) {
			continue
		}
		slotScoreFn := func(id string, slot rectSlot) float64 {
			return scoreVerticalLineAssignment(id, slot, metrics, demands)
		}
		cand, _ := bestCustomSlotAssignmentWithCost(kids, pool, slotScoreFn)
		if len(cand) != len(kids) {
			continue
		}
		cand, cost := refineChildSlotAssignment(kids, pool, cand, slotScoreFn, func(assigned map[string]rectSlot, rel map[string][2]float64) float64 {
			return scoreRectangularAssignment(kids, assigned, childEdges, externalLinks) +
				scoreLineAssignmentShape(kids, assigned) +
				scoreDominantHorizontalBoundaryShape(kids, assigned, metrics)
		})
		if cost < bestCost {
			bestCost = cost
			assigned = cand
		}
	}
	if assigned == nil {
		return 0, 0, false
	}

	for _, id := range kids {
		slot := assigned[id]
		relPos[id] = [2]float64{slot.X, slot.Y}
	}
	groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, relPos)
	return groupW, groupH, true
}

func layoutChildrenHorizontalLine(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
	demands map[string]childBoundaryDemand,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
) (float64, float64, bool) {
	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	if !shouldUseHorizontalLineChildLayout(childIDs, childEdges, metrics) {
		return 0, 0, false
	}

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
	for _, pool := range horizontalLineSlotPools(len(kids)) {
		if len(pool) < len(kids) {
			continue
		}
		slotScoreFn := func(id string, slot rectSlot) float64 {
			return scoreHorizontalLineAssignment(id, slot, metrics, demands)
		}
		cand, _ := bestCustomSlotAssignmentWithCost(kids, pool, slotScoreFn)
		if len(cand) != len(kids) {
			continue
		}
		cand, cost := refineChildSlotAssignment(kids, pool, cand, slotScoreFn, func(assigned map[string]rectSlot, rel map[string][2]float64) float64 {
			return scoreRectangularAssignment(kids, assigned, childEdges, externalLinks) +
				scoreLineAssignmentShape(kids, assigned)
		})
		if cost < bestCost {
			bestCost = cost
			assigned = cand
		}
	}
	if assigned == nil {
		return 0, 0, false
	}

	for _, id := range kids {
		slot := assigned[id]
		relPos[id] = [2]float64{slot.X, slot.Y}
	}
	groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, relPos)
	return groupW, groupH, true
}

func externalBoundarySlotPools() [][]rectSlot {
	offsetY := (groupPadTop - groupPadBot) / 2
	xOuter := rectCompactSideX + 84.0
	xInner := rectCompactCornerX + 44.0
	yTop := -rectCompactSideY + offsetY
	yUpper := offsetY - 18.0
	yLower := offsetY + 44.0
	yBottom := rectCompactSideY + offsetY + 18.0

	bottomHeavy := []rectSlot{
		{X: -xInner, Y: yTop, NX: -0.55, NY: -1, Priority: 8},
		{X: xInner, Y: yTop, NX: 0.55, NY: -1, Priority: 8},
		{X: -xOuter, Y: yUpper, NX: -1, NY: -0.2, Priority: 5},
		{X: xOuter, Y: yUpper, NX: 1, NY: -0.2, Priority: 5},
		{X: -xOuter, Y: yLower, NX: -1, NY: 0.45, Priority: 2},
		{X: xOuter, Y: yLower, NX: 1, NY: 0.45, Priority: 2},
		{X: -xOuter, Y: yBottom, NX: -1, NY: 1, Priority: 0},
		{X: -xInner, Y: yBottom, NX: -0.55, NY: 1, Priority: 0},
		{X: 0, Y: yBottom, NX: 0, NY: 1, Priority: 0},
		{X: xInner, Y: yBottom, NX: 0.55, NY: 1, Priority: 0},
		{X: xOuter, Y: yBottom, NX: 1, NY: 1, Priority: 0},
	}

	balanced := []rectSlot{
		{X: -xOuter, Y: yTop, NX: -1, NY: -1, Priority: 0},
		{X: -xInner, Y: yTop, NX: -0.55, NY: -1, Priority: 4},
		{X: xInner, Y: yTop, NX: 0.55, NY: -1, Priority: 4},
		{X: xOuter, Y: yTop, NX: 1, NY: -1, Priority: 0},
		{X: xOuter, Y: yUpper, NX: 1, NY: -0.2, Priority: 6},
		{X: xOuter, Y: yLower, NX: 1, NY: 0.45, Priority: 2},
		{X: xOuter, Y: yBottom, NX: 1, NY: 1, Priority: 0},
		{X: xInner, Y: yBottom, NX: 0.55, NY: 1, Priority: 0},
		{X: -xInner, Y: yBottom, NX: -0.55, NY: 1, Priority: 0},
		{X: -xOuter, Y: yBottom, NX: -1, NY: 1, Priority: 0},
		{X: -xOuter, Y: yLower, NX: -1, NY: 0.45, Priority: 2},
		{X: -xOuter, Y: yUpper, NX: -1, NY: -0.2, Priority: 6},
	}

	return [][]rectSlot{bottomHeavy, balanced}
}

func scoreExternalBoundaryAssignment(
	kids []string,
	assigned map[string]rectSlot,
	metrics map[string]childRectMetric,
	demands map[string]childBoundaryDemand,
) float64 {
	score := 0.0
	topUsed := 0
	totalWeightedY := 0.0
	totalWeight := 0.0

	for _, id := range kids {
		slot, ok := assigned[id]
		if !ok {
			continue
		}
		m := metrics[id]
		d := demands[id]
		weight := math.Max(1, float64(m.externalDegree))
		boundary := math.Max(math.Abs(slot.NX), math.Abs(slot.NY))
		if slot.NY < -0.2 {
			topUsed++
		}

		if m.externalDegree > 0 {
			score += (1 - boundary) * 220 * weight
		}

		if d.topCount+d.botCount > 0 {
			desiredX := boundaryDesiredX(id, demands, m.pull, slot.X)
			score += math.Abs(slot.X-desiredX) * 0.18
		}
		if d.leftCount+d.rightCount > 0 {
			desiredY := boundaryDesiredY(id, demands, m.pull, slot.Y)
			score += math.Abs(slot.Y-desiredY) * 0.18
		}

		if d.topCount > d.botCount && slot.NY > -0.45 {
			score += float64(d.topCount-d.botCount) * 220 * weight
		}
		if d.botCount > d.topCount && slot.NY < 0.35 {
			score += float64(d.botCount-d.topCount) * 180 * weight
		}
		if d.leftCount > d.rightCount && slot.NX > -0.35 {
			score += float64(d.leftCount-d.rightCount) * 180 * weight
		}
		if d.rightCount > d.leftCount && slot.NX < 0.35 {
			score += float64(d.rightCount-d.leftCount) * 180 * weight
		}
		if d.topCount+d.botCount > 0 && d.leftCount+d.rightCount > 0 && !isCornerSlot(slot) {
			score += 140 * weight
		}

		if m.pull.count == 0 {
			continue
		}

		totalWeightedY += m.pull.avgY * weight
		totalWeight += weight

		if m.pull.avgY > 120 && slot.NY < 0.25 {
			score += (0.25 - slot.NY) * 280 * weight
		}
		if m.pull.avgY > 220 && slot.NY < 0.6 {
			score += (0.6 - slot.NY) * 240 * weight
		}
		if m.pull.avgY < -120 && slot.NY > -0.25 {
			score += (slot.NY + 0.25) * 280 * weight
		}
		if m.pull.avgX > 120 && slot.NX < 0.1 {
			score += (0.1 - slot.NX) * 160 * weight
		}
		if m.pull.avgX < -120 && slot.NX > -0.1 {
			score += (slot.NX + 0.1) * 160 * weight
		}
		if m.externalOut > m.externalIn && slot.NY < 0 {
			score += 40 * weight
		}
		if m.externalIn > m.externalOut && slot.NY > 0.65 {
			score += 36 * weight
		}
	}

	if totalWeight > 0 {
		avgY := totalWeightedY / totalWeight
		topQuota := len(kids) / 4
		if topQuota < 1 {
			topQuota = 1
		}
		if avgY > 140 && topUsed > topQuota {
			score += float64(topUsed-topQuota) * 900
		}
		if avgY > 260 && topUsed > 1 {
			score += float64(topUsed-1) * 1500
		}
	}

	return score
}

func layoutChildrenExternalPullBoundary(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
	demands map[string]childBoundaryDemand,
	childEdges []model.Edge,
	externalLinks []childExternalLink,
) (float64, float64, bool) {
	if !shouldUseExternalPullBoundaryLayout(childIDs, childEdges, pulls) {
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
	for _, pool := range externalBoundarySlotPools() {
		if len(pool) < len(kids) {
			continue
		}
		slotScoreFn := func(id string, slot rectSlot) float64 {
			return scoreRectSlotAssignment(id, slot, metrics, len(kids))
		}
		cand, _ := bestRectangularSlotAssignmentWithCost(kids, pool, metrics)
		if len(cand) != len(kids) {
			continue
		}
		cand, cost := refineChildSlotAssignment(kids, pool, cand, slotScoreFn, func(assigned map[string]rectSlot, rel map[string][2]float64) float64 {
			return scoreRectangularAssignment(kids, assigned, childEdges, externalLinks) +
				scoreExternalBoundaryAssignment(kids, assigned, metrics, demands)
		})
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

	groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, relPos)
	return groupW, groupH, true
}

func layoutChildrenRectangular(
	childIDs []string,
	edges []model.Edge,
	relPos map[string][2]float64,
	pulls map[string]childExternalPull,
	demands map[string]childBoundaryDemand,
	externalLinks []childExternalLink,
) (float64, float64, bool) {
	if !shouldUseRectangularChildLayout(len(childIDs)) {
		return 0, 0, false
	}

	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	childEdges := childInternalEdges(childIDs, edges)
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
	pools := rectangularSlotPools(len(kids))
	if len(kids) == 8 && len(pools) >= 3 {
		horizontal, vertical, leftish, rightish, upish, downish := childExternalAxisStats(kids, metrics)
		switch {
		case vertical >= horizontal+2 || upish+downish > leftish+rightish+1:
			pools = pools[:2]
		case horizontal >= vertical+2 || leftish+rightish > upish+downish+1:
			pools = [][]rectSlot{pools[0], pools[2]}
		}
	}
	for _, pool := range pools {
		if len(pool) < len(kids) {
			continue
		}
		slotScoreFn := func(id string, slot rectSlot) float64 {
			return scoreRectSlotAssignmentWithBoundary(id, slot, metrics, demands, len(kids))
		}
		cand, cost := bestCustomSlotAssignmentWithCost(kids, pool, slotScoreFn)
		if len(cand) != len(kids) {
			continue
		}
		cand, cost = refineChildSlotAssignment(kids, pool, cand, slotScoreFn, func(assigned map[string]rectSlot, rel map[string][2]float64) float64 {
			return scoreRectangularAssignment(kids, assigned, childEdges, externalLinks) +
				scoreRectangularBoundaryOrdering(kids, assigned, metrics, demands)
		})
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

	groupW, groupH := expandChildLayoutUntilClear(kids, childEdges, externalLinks, relPos)
	return groupW, groupH, true
}

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
