package service

import (
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

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
