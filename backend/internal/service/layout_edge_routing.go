package service

import (
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

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

type predictedRoute struct {
	SrcTop string
	TgtTop string
	Points []Point
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
