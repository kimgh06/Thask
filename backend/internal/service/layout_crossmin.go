package service

import (
	"math"
	"sort"
)

func siftingCrossMin(layerNodes map[int][]string, maxLayer int, adj map[string]map[string]bool) {
	layerOf := make(map[string]int)
	for l, ids := range layerNodes {
		for _, id := range ids {
			layerOf[id] = l
		}
	}

	// Multiple sweeps for convergence (forward + backward)
	for sweep := 0; sweep < 4; sweep++ {
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
