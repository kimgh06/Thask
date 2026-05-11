package service

import (
	"sort"

	"github.com/thask/backend/internal/model"
)

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
