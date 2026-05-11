package service

import (
	"math"
	"sort"

	"github.com/thask/backend/internal/model"
)

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
