package service

import (
	"fmt"
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

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

	bestCost := layoutCost()
	improved := false
	for pass := 0; pass < 3; pass++ {
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
		for pass := 0; pass < 2; pass++ {
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
