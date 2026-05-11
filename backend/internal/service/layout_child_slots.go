package service

import (
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

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
