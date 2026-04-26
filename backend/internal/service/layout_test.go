package service

import (
	"math"
	"testing"

	"github.com/thask/backend/internal/model"
)

func TestCalculateLayoutOrdersGroupChildrenByExternalFlow(t *testing.T) {
	groupID := "infra"

	nodes := []model.Node{
		{ID: "infra", Title: "Infrastructure", Type: model.NodeTypeGroup},
		{ID: "docker", Title: "Docker + docker-compose", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "npm", Title: "npm Packages", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "github", Title: "GitHub Actions CI/CD", Type: model.NodeTypeTask},
		{ID: "runtime", Title: "Container Runtime", Type: model.NodeTypeTask},
	}

	edges := []model.Edge{
		{ID: "e1", SourceID: "github", TargetID: "npm", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "docker", TargetID: "runtime", EdgeType: model.EdgeTypeDependsOn},
	}

	result := CalculateLayout(nodes, edges, "dagre")
	pos := make(map[string]LayoutPosition, len(result.Positions))
	for _, p := range result.Positions {
		pos[p.ID] = p
	}

	if pos["github"].X >= pos["infra"].X {
		t.Fatalf("expected github to be left of group, got github=%v group=%v", pos["github"].X, pos["infra"].X)
	}
	if pos["runtime"].X <= pos["infra"].X {
		t.Fatalf("expected runtime to be right of group, got runtime=%v group=%v", pos["runtime"].X, pos["infra"].X)
	}
	if pos["npm"].X >= pos["docker"].X {
		t.Fatalf("expected npm child to be left of docker child, got npm=%v docker=%v", pos["npm"].X, pos["docker"].X)
	}
}

func TestCalculateLayoutMovesGroupAwayFromLongEdgeCorridor(t *testing.T) {
	groupID := "mid_group"

	nodes := []model.Node{
		{ID: "src", Title: "Source", Type: model.NodeTypeTask},
		{ID: "helper", Title: "Helper", Type: model.NodeTypeTask},
		{ID: "target", Title: "Target", Type: model.NodeTypeTask},
		{ID: "mid_group", Title: "Middle Group", Type: model.NodeTypeGroup},
		{ID: "g1", Title: "Child 1", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "g2", Title: "Child 2", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "g3", Title: "Child 3", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "g4", Title: "Child 4", Type: model.NodeTypeTask, ParentID: &groupID},
	}

	edges := []model.Edge{
		{ID: "e1", SourceID: "src", TargetID: "helper", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "helper", TargetID: "target", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "src", TargetID: "target", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "src", TargetID: "mid_group", EdgeType: model.EdgeTypeDependsOn},
	}

	result := CalculateLayout(nodes, edges, "dagre")
	pos := make(map[string]LayoutPosition, len(result.Positions))
	for _, p := range result.Positions {
		pos[p.ID] = p
	}

	group := pos["mid_group"]
	if group.Width == nil || group.Height == nil {
		t.Fatal("expected group dimensions to be set")
	}
	if !(pos["src"].X < group.X && group.X < pos["target"].X) {
		t.Fatalf("expected group x=%v to sit between src=%v and target=%v", group.X, pos["src"].X, pos["target"].X)
	}

	left := group.X - *group.Width/2
	right := group.X + *group.Width/2
	lineTop := interpolateLineY(pos["src"], pos["target"], left)
	lineBottom := interpolateLineY(pos["src"], pos["target"], right)
	corridorTop := math.Min(lineTop, lineBottom) - layerGapY/2
	corridorBottom := math.Max(lineTop, lineBottom) + layerGapY/2

	groupTop := group.Y - *group.Height/2
	groupBottom := group.Y + *group.Height/2
	if intervalsOverlap(groupTop, groupBottom, corridorTop, corridorBottom) {
		t.Fatalf(
			"expected long edge corridor [%v,%v] to avoid group vertical span [%v,%v]",
			corridorTop, corridorBottom, groupTop, groupBottom,
		)
	}
}

func TestCalculateLayoutKeepsTopLevelGroupsCompactByFlowRole(t *testing.T) {
	infraID := "infra"
	frontendID := "frontend"
	backendID := "backend"
	cliID := "cli"

	nodes := []model.Node{
		{ID: infraID, Title: "Infrastructure", Type: model.NodeTypeGroup},
		{ID: "docker", Title: "Docker + docker-compose", Type: model.NodeTypeTask, ParentID: &infraID},
		{ID: frontendID, Title: "Frontend", Type: model.NodeTypeGroup},
		{ID: "frontend_client", Title: "Frontend API Client", Type: model.NodeTypeTask, ParentID: &frontendID},
		{ID: backendID, Title: "Backend", Type: model.NodeTypeGroup},
		{ID: "entry", Title: "Entry Point", Type: model.NodeTypeTask, ParentID: &backendID},
		{ID: "repo", Title: "Repository Layer", Type: model.NodeTypeTask, ParentID: &backendID},
		{ID: cliID, Title: "CLI", Type: model.NodeTypeGroup},
		{ID: "cli_client", Title: "CLI API Client", Type: model.NodeTypeTask, ParentID: &cliID},
		{ID: "db", Title: "PostgreSQL", Type: model.NodeTypeTask},
	}

	edges := []model.Edge{
		{ID: "e1", SourceID: "docker", TargetID: "entry", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "docker", TargetID: "frontend_client", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "frontend_client", TargetID: "entry", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "cli_client", TargetID: "entry", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "entry", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "repo", TargetID: "db", EdgeType: model.EdgeTypeDependsOn},
	}

	result := CalculateLayout(nodes, edges, "dagre")
	pos := make(map[string]LayoutPosition, len(result.Positions))
	for _, p := range result.Positions {
		pos[p.ID] = p
	}

	if math.Abs(pos[infraID].Y-pos[frontendID].Y) > 200 {
		t.Fatalf("expected infrastructure to stay near frontend band, got infra=%v frontend=%v", pos[infraID].Y, pos[frontendID].Y)
	}
	if math.Abs(pos[cliID].Y-pos[frontendID].Y) > 420 {
		t.Fatalf("expected cli to stay within a compact distance of frontend, got frontend=%v cli=%v", pos[frontendID].Y, pos[cliID].Y)
	}
	if math.Abs(pos["db"].Y-pos[backendID].Y) > 280 {
		t.Fatalf("expected database sink to remain near backend band, got db=%v backend=%v", pos["db"].Y, pos[backendID].Y)
	}
	if !(pos[infraID].X < pos[backendID].X && pos[frontendID].X < pos[backendID].X) {
		t.Fatalf("expected infra/frontend to be left of backend, got infra=%v frontend=%v backend=%v", pos[infraID].X, pos[frontendID].X, pos[backendID].X)
	}
}

func TestLayoutChildrenRectangularUsesRoleAwareTemplate(t *testing.T) {
	childIDs := []string{"commands", "scanner", "entry", "mcp", "api"}
	relPos := make(map[string][2]float64, len(childIDs))
	pulls := map[string]childExternalPull{
		"commands": {avgX: -420, avgY: -220, count: 1},
		"scanner":  {avgX: 360, avgY: -160, count: 1},
		"entry":    {avgX: -300, avgY: 300, count: 1},
		"api":      {avgX: 360, avgY: 320, count: 1},
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "ext_left", TargetID: "entry", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "commands", TargetID: "ext_upper_left", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "scanner", TargetID: "ext_upper_right", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "api", TargetID: "ext_lower_right", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "entry", TargetID: "mcp", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "mcp", TargetID: "api", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "commands", TargetID: "api", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e8", SourceID: "scanner", TargetID: "api", EdgeType: model.EdgeTypeDependsOn},
	}

	w, h, ok := layoutChildrenRectangular(childIDs, edges, relPos, pulls, nil, nil)
	if !ok {
		t.Fatal("expected rectangular child layout to apply")
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected group to have sensible size, got w=%v h=%v", w, h)
	}
	if w >= 520 || h >= 520 {
		t.Fatalf("expected rectangular layout to stay reasonably bounded, got w=%v h=%v", w, h)
	}

	centerY := (groupPadTop - groupPadBot) / 2
	commands := relPos["commands"]
	scanner := relPos["scanner"]
	api := relPos["api"]
	entry := relPos["entry"]
	mcp := relPos["mcp"]
	if !(commands[0] < 0 && commands[1] < centerY) {
		t.Fatalf("expected commands in upper-left quadrant, got %+v", commands)
	}
	if !(scanner[0] > 0 && scanner[1] < centerY) {
		t.Fatalf("expected scanner in upper-right quadrant, got %+v", scanner)
	}
	if !(entry[0] < 0 && entry[1] > centerY) {
		t.Fatalf("expected entry in lower-left quadrant, got %+v", entry)
	}
	if !(api[0] > 0 && api[1] >= centerY-16) {
		t.Fatalf("expected api on right side and not pulled into upper half, got %+v", api)
	}
	if !(mcp[1] > centerY) {
		t.Fatalf("expected mcp to stay in lower half, got %+v", mcp)
	}

	intersections := 0
	for _, e := range edges {
		srcPos, srcOK := relPos[e.SourceID]
		tgtPos, tgtOK := relPos[e.TargetID]
		if !srcOK || !tgtOK {
			continue
		}
		src := Point{X: srcPos[0], Y: srcPos[1]}
		tgt := Point{X: tgtPos[0], Y: tgtPos[1]}
		points := []Point{src}
		points = append(points, compute8DirWaypoints(src, tgt)...)
		points = append(points, tgt)
		for _, id := range childIDs {
			if id == e.SourceID || id == e.TargetID {
				continue
			}
			if polylineIntersectsRect(points, childRectBoxAt(relPos[id], 6)) {
				intersections++
			}
		}
	}
	if intersections != 0 {
		t.Fatalf("expected rectangular layout to avoid internal node-over-node routes, got %d intersections relPos=%v", intersections, relPos)
	}
}

func TestChildLayoutCostPenalizesExternalRouteNodeIntersections(t *testing.T) {
	childIDs := []string{"auth", "eventhub"}
	externalLinks := []childExternalLink{
		{routeID: "e1", childID: "eventhub", dx: -420, dy: 0, inbound: true},
	}

	badRel := map[string][2]float64{
		"auth":     {0, 0},
		"eventhub": {176, 0},
	}
	goodRel := map[string][2]float64{
		"auth":     {0, 132},
		"eventhub": {176, 0},
	}

	badHits := countChildExternalRouteNodeIntersections(childIDs, externalLinks, badRel)
	if badHits == 0 {
		t.Fatalf("expected left-entry route to hit auth in bad layout, got badRel=%v", badRel)
	}

	goodHits := countChildExternalRouteNodeIntersections(childIDs, externalLinks, goodRel)
	if goodHits >= badHits {
		t.Fatalf("expected improved layout to reduce external route hits, got bad=%d good=%d", badHits, goodHits)
	}

	badCost := childLayoutCost(childIDs, nil, externalLinks, badRel)
	goodCost := childLayoutCost(childIDs, nil, externalLinks, goodRel)
	if goodCost >= badCost {
		t.Fatalf("expected external route intersections to increase layout cost, got bad=%v good=%v", badCost, goodCost)
	}
}

func TestChildLayoutCostPenalizesOverlyElongatedShape(t *testing.T) {
	childIDs := []string{"a", "b", "c", "d"}
	wideRel := map[string][2]float64{
		"a": {-300, 0},
		"b": {-100, 0},
		"c": {100, 0},
		"d": {300, 0},
	}
	compactRel := map[string][2]float64{
		"a": {-90, -70},
		"b": {90, -70},
		"c": {-90, 70},
		"d": {90, 70},
	}

	wideCost := childLayoutCost(childIDs, nil, nil, wideRel)
	compactCost := childLayoutCost(childIDs, nil, nil, compactRel)
	if compactCost >= wideCost {
		t.Fatalf("expected elongated layout to cost more than compact layout, wide=%v compact=%v", wideCost, compactCost)
	}
}

func TestCountPredictedChildRouteCrossingsIgnoresSharedEndpointsButCountsRealCrossings(t *testing.T) {
	routes := []childPredictedRouteInfo{
		{
			srcID:  "a",
			tgtID:  "b",
			points: []Point{{X: -100, Y: -100}, {X: 100, Y: 100}},
		},
		{
			srcID:  "c",
			tgtID:  "d",
			points: []Point{{X: -100, Y: 100}, {X: 100, Y: -100}},
		},
		{
			srcID:  "b",
			tgtID:  "e",
			points: []Point{{X: 100, Y: 100}, {X: 220, Y: 220}},
		},
	}

	crossings := countPredictedChildRouteCrossings(routes)
	if crossings != 1 {
		t.Fatalf("expected exactly one real crossing, got %d routes=%v", crossings, routes)
	}
}

func TestChildLayoutCostPenalizesMixedInternalExternalCrossings(t *testing.T) {
	childIDs := []string{"left", "right", "sink"}
	childEdges := []model.Edge{
		{ID: "e1", SourceID: "left", TargetID: "right", EdgeType: model.EdgeTypeDependsOn},
	}
	externalLinks := []childExternalLink{
		{routeID: "in", childID: "sink", dx: 0, dy: -420, inbound: true},
	}

	badRel := map[string][2]float64{
		"left":  {-150, 0},
		"right": {150, 0},
		"sink":  {0, 80},
	}
	goodRel := map[string][2]float64{
		"left":  {-150, 180},
		"right": {150, 180},
		"sink":  {0, 80},
	}

	badCrossings := countChildAllRouteCrossings(childIDs, childEdges, externalLinks, badRel)
	goodCrossings := countChildAllRouteCrossings(childIDs, childEdges, externalLinks, goodRel)
	if badCrossings == 0 {
		t.Fatalf("expected mixed internal/external crossing in bad layout, got badRel=%v", badRel)
	}
	if goodCrossings >= badCrossings {
		t.Fatalf("expected improved layout to reduce mixed crossings, got bad=%d good=%d", badCrossings, goodCrossings)
	}

	badCost := childLayoutCost(childIDs, childEdges, externalLinks, badRel)
	goodCost := childLayoutCost(childIDs, childEdges, externalLinks, goodRel)
	if goodCost >= badCost {
		t.Fatalf("expected mixed crossings to raise layout cost, got bad=%v good=%v", badCost, goodCost)
	}
}

func TestScoreChildPlacementAffinityPenalizesBoundaryOrderInversions(t *testing.T) {
	childIDs := []string{"top", "bottom"}
	metrics := map[string]childRectMetric{
		"top": {
			externalDegree: 1,
			pull:           childExternalPull{avgX: -320, avgY: -220, count: 1},
		},
		"bottom": {
			externalDegree: 1,
			pull:           childExternalPull{avgX: -320, avgY: 220, count: 1},
		},
	}
	demands := map[string]childBoundaryDemand{
		"top": {
			leftOrder: -240,
			leftCount: 1,
		},
		"bottom": {
			leftOrder: 240,
			leftCount: 1,
		},
	}

	goodRel := map[string][2]float64{
		"top":    {-140, -120},
		"bottom": {-140, 120},
	}
	badRel := map[string][2]float64{
		"top":    {-140, 120},
		"bottom": {-140, -120},
	}

	goodScore := scoreChildPlacementAffinity(childIDs, goodRel, metrics, demands)
	badScore := scoreChildPlacementAffinity(childIDs, badRel, metrics, demands)
	if badScore <= goodScore {
		t.Fatalf("expected boundary order inversion to be penalized, good=%v bad=%v", goodScore, badScore)
	}
}

func TestRefineChildSlotAssignmentSwapsSameSideOrder(t *testing.T) {
	childIDs := []string{"top", "bottom"}
	pool := []rectSlot{
		{X: -120, Y: -120, NX: -1, NY: -1},
		{X: -120, Y: 120, NX: -1, NY: 1},
	}
	assigned := map[string]rectSlot{
		"top":    pool[1],
		"bottom": pool[0],
	}
	metrics := map[string]childRectMetric{
		"top": {
			externalDegree: 1,
			pull:           childExternalPull{avgX: -320, avgY: -220, count: 1},
		},
		"bottom": {
			externalDegree: 1,
			pull:           childExternalPull{avgX: -320, avgY: 220, count: 1},
		},
	}
	demands := map[string]childBoundaryDemand{
		"top": {
			leftOrder: -240,
			leftCount: 1,
		},
		"bottom": {
			leftOrder: 240,
			leftCount: 1,
		},
	}

	slotScoreFn := func(string, rectSlot) float64 { return 0 }
	extraCostFn := func(_ map[string]rectSlot, rel map[string][2]float64) float64 {
		return scoreChildPlacementAffinity(childIDs, rel, metrics, demands)
	}

	beforeCost := childSlotAssignmentCost(childIDs, assigned, slotScoreFn, extraCostFn)
	refined, afterCost := refineChildSlotAssignment(childIDs, pool, assigned, slotScoreFn, extraCostFn)

	if !(refined["top"].Y < refined["bottom"].Y) {
		t.Fatalf("expected refinement to restore top-before-bottom ordering, got refined=%v", refined)
	}
	if afterCost >= beforeCost {
		t.Fatalf("expected refinement to reduce slot assignment cost, before=%v after=%v refined=%v", beforeCost, afterCost, refined)
	}
}

func TestExpandChildLayoutUntilClearGrowsPerpendicularToHorizontalExternalRoute(t *testing.T) {
	childIDs := []string{"auth", "shared", "require"}
	externalLinks := []childExternalLink{
		{routeID: "in", childID: "auth", dx: 420, dy: 0, inbound: true},
	}
	relPos := map[string][2]float64{
		"auth":    {-140, 0},
		"shared":  {140, 16},
		"require": {140, -160},
	}

	beforeHits := countChildExternalRouteNodeIntersections(childIDs, externalLinks, relPos)
	if beforeHits == 0 {
		t.Fatalf("expected seed layout to have a horizontal external-route collision, got relPos=%v", relPos)
	}

	beforeGap := math.Abs(relPos["shared"][1] - relPos["auth"][1])
	_, _ = expandChildLayoutUntilClear(childIDs, nil, externalLinks, relPos)
	afterHits := countChildExternalRouteNodeIntersections(childIDs, externalLinks, relPos)
	afterGap := math.Abs(relPos["shared"][1] - relPos["auth"][1])

	if afterHits != 0 {
		t.Fatalf("expected expansion to clear horizontal external-route collision, got hits=%d relPos=%v", afterHits, relPos)
	}
	if afterGap <= beforeGap {
		t.Fatalf("expected horizontal-route cleanup to increase vertical separation, beforeGap=%v afterGap=%v relPos=%v", beforeGap, afterGap, relPos)
	}
}

func TestScoreChildPlacementAffinityKeepsRightPulledNodeOnRight(t *testing.T) {
	childIDs := []string{"shared"}
	metrics := map[string]childRectMetric{
		"shared": {
			externalDegree: 1,
			externalOut:    1,
			pull:           childExternalPull{avgX: 480, avgY: 40, count: 1},
		},
	}
	demands := map[string]childBoundaryDemand{
		"shared": {rightCount: 1, rightOrder: 40},
	}

	leftScore := scoreChildPlacementAffinity(childIDs, map[string][2]float64{"shared": {-140, 0}}, metrics, demands)
	rightScore := scoreChildPlacementAffinity(childIDs, map[string][2]float64{"shared": {140, 0}}, metrics, demands)

	if rightScore >= leftScore {
		t.Fatalf("expected right-bound placement to be preferred, got left=%v right=%v", leftScore, rightScore)
	}
}

func TestLayoutChildrenByLayersLeavesTrafficLaneThroughMiddleColumn(t *testing.T) {
	childIDs := []string{"src", "mid_a", "mid_b", "mid_c", "tgt"}
	childEdges := []model.Edge{
		{ID: "e1", SourceID: "src", TargetID: "mid_a", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "src", TargetID: "mid_b", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "src", TargetID: "mid_c", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "src", TargetID: "tgt", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "mid_a", TargetID: "tgt", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "mid_b", TargetID: "tgt", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "mid_c", TargetID: "tgt", EdgeType: model.EdgeTypeDependsOn},
	}
	relPos := make(map[string][2]float64, len(childIDs))

	w, h := layoutChildrenByLayers(childIDs, childEdges, childLayoutNodeW+childLayoutCellPad, childLayoutNodeH+childLayoutCellPad, relPos)
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible group size, got w=%v h=%v", w, h)
	}

	intersections := countChildRouteNodeIntersections(childIDs, childEdges, relPos)
	if intersections != 0 {
		t.Fatalf("expected layered child layout to keep a middle traffic lane, got %d intersections relPos=%v", intersections, relPos)
	}

	midY := []float64{relPos["mid_a"][1], relPos["mid_b"][1], relPos["mid_c"][1]}
	centerHits := 0
	for _, y := range midY {
		if math.Abs(y) < 24 {
			centerHits++
		}
	}
	if centerHits != 0 {
		t.Fatalf("expected middle column to avoid the central traffic lane, got relPos=%v", relPos)
	}
}

func TestExpandChildLayoutUntilClearGrowsDenseChildCluster(t *testing.T) {
	childIDs := []string{"a", "b", "c", "d", "e"}
	childEdges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "d", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "b", TargetID: "e", EdgeType: model.EdgeTypeDependsOn},
	}
	relPos := map[string][2]float64{
		"a": {-40, -20},
		"b": {0, -20},
		"c": {40, -20},
		"d": {-20, 20},
		"e": {20, 20},
	}

	beforeOverlaps := countChildBoxOverlaps(childIDs, relPos, 4)
	if beforeOverlaps == 0 {
		t.Fatal("expected dense seed positions to overlap before expansion")
	}

	w, h := expandChildLayoutUntilClear(childIDs, childEdges, nil, relPos)
	if countChildBoxOverlaps(childIDs, relPos, 4) != 0 {
		t.Fatalf("expected expansion to remove child overlap, got relPos=%v", relPos)
	}
	if countChildRouteNodeIntersections(childIDs, childEdges, relPos) != 0 {
		t.Fatalf("expected expansion to remove route-vs-node intersections, got relPos=%v", relPos)
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible grown group size, got w=%v h=%v", w, h)
	}
}

func TestExpandChildLayoutUntilClearCompactsOverWideSafeLayout(t *testing.T) {
	childIDs := []string{"a", "b", "c"}
	relPos := map[string][2]float64{
		"a": {-520, -180},
		"b": {0, 0},
		"c": {520, 180},
	}

	beforeW, beforeH := recenterAndMeasureChildLayout(childIDs, relPos)
	w, h := expandChildLayoutUntilClear(childIDs, nil, nil, relPos)
	violations := measureChildLayoutViolations(childIDs, nil, nil, relPos)

	if !childLayoutViolationsClear(violations) {
		t.Fatalf("expected compacted layout to remain valid, got violations=%+v relPos=%v", violations, relPos)
	}
	if w >= beforeW*0.7 {
		t.Fatalf("expected overly wide safe layout to compact substantially, beforeW=%v afterW=%v relPos=%v", beforeW, w, relPos)
	}
	if h >= beforeH*0.8 {
		t.Fatalf("expected overly tall safe layout to compact substantially, beforeH=%v afterH=%v relPos=%v", beforeH, h, relPos)
	}
}

func TestComputeLayerXPositionsUsesActualGroupWidth(t *testing.T) {
	groupSizes := map[string][2]float64{
		"wide": {760, 320},
	}
	layerNodes := map[int][]string{
		0: {"wide"},
		1: {"next"},
	}

	layerX := computeLayerXPositions(layerNodes, 1, groupSizes)
	expected := snapToGrid(760) +
		math.Ceil(layerGapX/gridSize)*gridSize +
		snapToGrid(nodeW/2)

	if layerX[1] != expected {
		t.Fatalf("expected second layer x=%v using actual group width, got %v", expected, layerX[1])
	}
}

func TestCleanupRouteBoxIntersectionsMovesBlockingNode(t *testing.T) {
	nodes := []model.Node{
		{ID: "src", Title: "Source", Type: model.NodeTypeTask},
		{ID: "blocker", Title: "Blocker", Type: model.NodeTypeTask},
		{ID: "tgt", Title: "Target", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	edges := []model.Edge{
		{ID: "e1", SourceID: "src", TargetID: "tgt", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "src", TargetID: "blocker", EdgeType: model.EdgeTypeDependsOn},
	}
	positions := map[string][2]float64{
		"src":     {0, 0},
		"blocker": {200, 0},
		"tgt":     {400, 0},
	}

	before, byNode := countPredictedRouteBoxIntersections(edges, positions, nodeMap, map[string][2]float64{}, map[string][2]float64{})
	if before == 0 || byNode["blocker"] == 0 {
		t.Fatalf("expected blocker to intersect the predicted src->tgt route before cleanup, got total=%d blocker=%d", before, byNode["blocker"])
	}

	cleanupRouteBoxIntersections(positions, edges, nodeMap, map[string][2]float64{}, map[string][2]float64{})

	after, byNode := countPredictedRouteBoxIntersections(edges, positions, nodeMap, map[string][2]float64{}, map[string][2]float64{})
	if after != 0 || byNode["blocker"] != 0 {
		t.Fatalf("expected cleanup to remove blocker intersection, got total=%d blocker=%d positions=%v", after, byNode["blocker"], positions)
	}
	if positions["blocker"][1] == 0 {
		t.Fatalf("expected blocker to move off the route, got positions=%v", positions)
	}
}

func TestCountPredictedRouteCrossingsDetectsCrossingRoutes(t *testing.T) {
	nodes := []model.Node{
		{ID: "a", Title: "A", Type: model.NodeTypeTask},
		{ID: "b", Title: "B", Type: model.NodeTypeTask},
		{ID: "c", Title: "C", Type: model.NodeTypeTask},
		{ID: "d", Title: "D", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	edges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "d", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "c", TargetID: "b", EdgeType: model.EdgeTypeDependsOn},
	}
	positions := map[string][2]float64{
		"a": {0, 0},
		"b": {400, 0},
		"c": {0, 400},
		"d": {400, 400},
	}

	crossings, byNode := countPredictedRouteCrossings(edges, positions, nodeMap, map[string][2]float64{})
	if crossings == 0 {
		t.Fatalf("expected predicted routes to cross, got 0")
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if byNode[id] == 0 {
			t.Fatalf("expected crossing attribution for %s, got %+v", id, byNode)
		}
	}
}

func TestCountPredictedRouteHotspotsDetectsSharedRouteRegion(t *testing.T) {
	nodes := []model.Node{
		{ID: "a", Title: "A", Type: model.NodeTypeTask},
		{ID: "b", Title: "B", Type: model.NodeTypeTask},
		{ID: "c", Title: "C", Type: model.NodeTypeTask},
		{ID: "d", Title: "D", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	edges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "d", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "c", TargetID: "b", EdgeType: model.EdgeTypeDependsOn},
	}
	positions := map[string][2]float64{
		"a": {0, 0},
		"b": {400, 0},
		"c": {0, 400},
		"d": {400, 400},
	}

	hotspots, byNode := countPredictedRouteHotspots(edges, positions, nodeMap, map[string][2]float64{})
	if hotspots == 0 {
		t.Fatalf("expected crossing routes to share at least one hotspot cell, got 0")
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if byNode[id] == 0 {
			t.Fatalf("expected hotspot attribution for %s, got %+v", id, byNode)
		}
	}
}

func TestRefineLayerOrderByPredictedRoutesReducesTopLevelCrossings(t *testing.T) {
	nodes := []model.Node{
		{ID: "a", Title: "A", Type: model.NodeTypeTask},
		{ID: "b", Title: "B", Type: model.NodeTypeTask},
		{ID: "c", Title: "C", Type: model.NodeTypeTask},
		{ID: "d", Title: "D", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	layerNodes := map[int][]string{
		0: {"a", "c"},
		1: {"b", "d"},
	}
	layerX := map[int]float64{
		0: 0,
		1: 400,
	}
	yCoords := map[string]float64{
		"a": 0,
		"c": 160,
		"b": 0,
		"d": 160,
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "d", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "c", TargetID: "b", EdgeType: model.EdgeTypeDependsOn},
	}

	beforePos := buildTopLevelPositions(layerNodes, 1, layerX, yCoords)
	beforeCrossings, _ := countPredictedRouteCrossings(edges, beforePos, nodeMap, map[string][2]float64{})
	beforeCost := topLevelPredictedRouteCost(layerNodes, 1, layerX, yCoords, map[string][2]float64{}, edges, nodeMap, map[string][2]float64{})
	if beforeCrossings == 0 {
		t.Fatalf("expected starting order to produce a crossing, got positions=%v", beforePos)
	}

	refineLayerOrderByPredictedRoutes(layerNodes, 1, layerX, yCoords, map[string][2]float64{}, edges, nodeMap, map[string][2]float64{})

	afterPos := buildTopLevelPositions(layerNodes, 1, layerX, yCoords)
	afterCrossings, _ := countPredictedRouteCrossings(edges, afterPos, nodeMap, map[string][2]float64{})
	afterCost := topLevelPredictedRouteCost(layerNodes, 1, layerX, yCoords, map[string][2]float64{}, edges, nodeMap, map[string][2]float64{})

	if afterCrossings >= beforeCrossings {
		t.Fatalf("expected local route-aware reorder to reduce crossings, before=%d after=%d beforePos=%v afterPos=%v", beforeCrossings, afterCrossings, beforePos, afterPos)
	}
	if afterCost >= beforeCost {
		t.Fatalf("expected route-aware reorder to lower cost, before=%v after=%v", beforeCost, afterCost)
	}
}

func TestRefineLayerCentersByPredictedRoutesMovesNodeOffRouteCorridor(t *testing.T) {
	nodes := []model.Node{
		{ID: "src", Title: "Source", Type: model.NodeTypeTask},
		{ID: "mid", Title: "Middle", Type: model.NodeTypeTask},
		{ID: "tgt", Title: "Target", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	layerNodes := map[int][]string{
		0: {"src"},
		1: {"mid"},
		2: {"tgt"},
	}
	layerX := map[int]float64{
		0: 0,
		1: 240,
		2: 480,
	}
	yCoords := map[string]float64{
		"src": 0,
		"mid": 0,
		"tgt": 0,
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "src", TargetID: "tgt", EdgeType: model.EdgeTypeDependsOn},
	}

	beforePos := buildTopLevelPositions(layerNodes, 2, layerX, yCoords)
	beforeHits, _ := countPredictedRouteBoxIntersections(edges, beforePos, nodeMap, map[string][2]float64{}, map[string][2]float64{})
	if beforeHits == 0 {
		t.Fatalf("expected src->tgt corridor to hit middle node before refinement, got positions=%v", beforePos)
	}

	refineLayerCentersByPredictedRoutes(layerNodes, 2, layerX, yCoords, map[string][2]float64{}, edges, nodeMap, map[string][2]float64{}, nil)

	afterPos := buildTopLevelPositions(layerNodes, 2, layerX, yCoords)
	afterHits, _ := countPredictedRouteBoxIntersections(edges, afterPos, nodeMap, map[string][2]float64{}, map[string][2]float64{})
	if afterHits >= beforeHits {
		t.Fatalf("expected route-aware center refinement to reduce corridor hits, before=%d after=%d before=%v after=%v", beforeHits, afterHits, beforePos, afterPos)
	}
	if afterPos["mid"][1] == beforePos["mid"][1] {
		t.Fatalf("expected middle node to move away from the route corridor, got before=%v after=%v", beforePos, afterPos)
	}
}

func TestRefineTopLevelLayersByPredictedRoutesMovesNodeToAdjacentLayer(t *testing.T) {
	nodes := []model.Node{
		{ID: "src1", Title: "Source 1", Type: model.NodeTypeTask},
		{ID: "src2", Title: "Source 2", Type: model.NodeTypeTask},
		{ID: "mid", Title: "Middle", Type: model.NodeTypeTask},
		{ID: "sink", Title: "Sink", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	topLevel := []string{"src1", "src2", "mid", "sink"}
	layers := map[string]int{
		"src1": 0,
		"src2": 0,
		"mid":  2,
		"sink": 3,
	}
	outEdges := map[string][]string{
		"src1": {"mid"},
		"src2": {"mid"},
		"mid":  {"sink"},
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "src1", TargetID: "mid", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "src2", TargetID: "mid", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "mid", TargetID: "sink", EdgeType: model.EdgeTypeDependsOn},
	}

	beforeCost := layerAssignmentOrderAndCost(topLevel, layers, outEdges, edges, nodeMap, map[string][2]float64{}, map[string][2]float64{})
	refineTopLevelLayersByPredictedRoutes(topLevel, layers, outEdges, edges, nodeMap, map[string][2]float64{}, map[string][2]float64{})
	afterCost := layerAssignmentOrderAndCost(topLevel, layers, outEdges, edges, nodeMap, map[string][2]float64{}, map[string][2]float64{})

	if layers["mid"] != 1 {
		t.Fatalf("expected middle node to shift to adjacent layer 1, got layers=%v", layers)
	}
	if afterCost >= beforeCost {
		t.Fatalf("expected layer refinement to reduce total placement cost, before=%v after=%v", beforeCost, afterCost)
	}
}

func TestLayoutChildrenExternalPullBoundaryLeavesTopMostlyOpen(t *testing.T) {
	childIDs := []string{"a", "b", "c", "d", "e", "f", "g"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "b", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "c", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "d", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "e", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "f", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "g", TargetID: "mw", EdgeType: model.EdgeTypeTriggers},
	}
	pulls := map[string]childExternalPull{
		"a": {avgX: 200, avgY: 360, count: 1},
		"b": {avgX: 120, avgY: 360, count: 1},
		"c": {avgX: 0, avgY: 420, count: 1},
		"d": {avgX: -40, avgY: 420, count: 1},
		"e": {avgX: 240, avgY: 360, count: 1},
		"f": {avgX: 160, avgY: 360, count: 1},
		"g": {avgX: -200, avgY: 420, count: 1},
	}
	relPos := map[string][2]float64{}

	w, h, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, nil, nil, nil)
	if !ok {
		t.Fatal("expected external pull boundary layout to apply")
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible group size, got w=%v h=%v", w, h)
	}

	topCount := 0
	bottomCount := 0
	for _, id := range childIDs {
		if relPos[id][1] < 0 {
			topCount++
		}
		if relPos[id][1] > 0 {
			bottomCount++
		}
	}
	if topCount > 2 {
		t.Fatalf("expected bottom-biased boundary layout, got topCount=%d relPos=%v", topCount, relPos)
	}
	if bottomCount < 4 {
		t.Fatalf("expected most nodes to sit in lower half, got bottomCount=%d relPos=%v", bottomCount, relPos)
	}
}

func TestLayoutChildrenExternalPullBoundaryBackendLikeCase(t *testing.T) {
	childIDs := []string{"project", "impact", "echo", "edge", "team", "event", "node"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "api_client", TargetID: "echo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "cli_client", TargetID: "echo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "echo", TargetID: "authmw", EdgeType: model.EdgeTypeTriggers},
		{ID: "e5", SourceID: "edge", TargetID: "edgerepo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "edge", TargetID: "eventhub", EdgeType: model.EdgeTypeTriggers},
		{ID: "e7", SourceID: "event", TargetID: "eventhub", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e8", SourceID: "impact", TargetID: "impactsvc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e9", SourceID: "node", TargetID: "eventhub", EdgeType: model.EdgeTypeTriggers},
		{ID: "e10", SourceID: "node", TargetID: "layoutsvc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e11", SourceID: "node", TargetID: "noderepo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e12", SourceID: "node", TargetID: "waterfall", EdgeType: model.EdgeTypeTriggers},
		{ID: "e13", SourceID: "project", TargetID: "projectrepo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e14", SourceID: "mw", TargetID: "edge", EdgeType: model.EdgeTypeTriggers},
		{ID: "e15", SourceID: "mw", TargetID: "event", EdgeType: model.EdgeTypeTriggers},
		{ID: "e16", SourceID: "mw", TargetID: "impact", EdgeType: model.EdgeTypeTriggers},
		{ID: "e17", SourceID: "mw", TargetID: "node", EdgeType: model.EdgeTypeTriggers},
		{ID: "e18", SourceID: "mw", TargetID: "project", EdgeType: model.EdgeTypeTriggers},
		{ID: "e19", SourceID: "sharedmw", TargetID: "node", EdgeType: model.EdgeTypeTriggers},
		{ID: "e20", SourceID: "team", TargetID: "userrepo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e21", SourceID: "teammw", TargetID: "team", EdgeType: model.EdgeTypeTriggers},
	}
	pulls := map[string]childExternalPull{
		"project": {avgX: -240, avgY: 480, count: 2},
		"impact":  {avgX: -60, avgY: 440, count: 2},
		"echo":    {avgX: -186.7, avgY: 160, count: 3},
		"edge":    {avgX: -40, avgY: 453.3, count: 3},
		"team":    {avgX: -240, avgY: 480, count: 2},
		"event":   {avgX: -60, avgY: 440, count: 2},
		"node":    {avgX: 20, avgY: 440, count: 6},
	}
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, nil, nil, nil)
	if !ok {
		t.Fatal("expected external pull boundary layout to apply")
	}

	topTitles := make([]string, 0)
	for _, id := range childIDs {
		if relPos[id][1] < 0 {
			topTitles = append(topTitles, id)
		}
	}
	if len(topTitles) > 2 {
		t.Fatalf("expected backend-like case to keep top row sparse, got top=%v relPos=%v", topTitles, relPos)
	}
	if relPos["node"][1] < 0 || relPos["impact"][1] < 0 || relPos["edge"][1] < 0 {
		t.Fatalf("expected down-pulled handlers to avoid upper half, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenExternalPullBoundaryRecentersAsymmetricGroup(t *testing.T) {
	childIDs := []string{"eventhub", "layout", "auth", "waterfall", "impact"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "eventhub", TargetID: "backend", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "layout", TargetID: "backend", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "auth", TargetID: "backend", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "waterfall", TargetID: "backend", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "impact", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
	}
	pulls := map[string]childExternalPull{
		"eventhub":  {avgX: -300, avgY: -40, count: 1},
		"layout":    {avgX: -260, avgY: 40, count: 1},
		"auth":      {avgX: -260, avgY: 120, count: 1},
		"waterfall": {avgX: -240, avgY: 220, count: 1},
		"impact":    {avgX: -120, avgY: 220, count: 1},
	}
	relPos := map[string][2]float64{}

	w, _, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, nil, nil, nil)
	if !ok {
		t.Fatal("expected external pull boundary layout to apply")
	}

	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	for _, id := range childIDs {
		if relPos[id][0] < minX {
			minX = relPos[id][0]
		}
		if relPos[id][0] > maxX {
			maxX = relPos[id][0]
		}
	}
	if !(minX < 0 && maxX > 0) {
		t.Fatalf("expected asymmetric boundary layout to be recentered, got minX=%v maxX=%v relPos=%v", minX, maxX, relPos)
	}
	if w < minGroupW {
		t.Fatalf("expected sensible width after recenter, got %v", w)
	}
}

func TestLayoutChildrenExternalPullBoundaryRespectsBoundaryDemands(t *testing.T) {
	childIDs := []string{"user", "node", "project", "edge", "postgres"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "team_handler", TargetID: "user", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "node_handler", TargetID: "node", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "project_handler", TargetID: "project", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "edge_handler", TargetID: "edge", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "user", TargetID: "postgres", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "node", TargetID: "postgres", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "project", TargetID: "postgres", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e8", SourceID: "edge", TargetID: "postgres", EdgeType: model.EdgeTypeDependsOn},
	}
	pulls := map[string]childExternalPull{
		"user":     {avgX: -360, avgY: 260, count: 1},
		"node":     {avgX: -120, avgY: 260, count: 1},
		"project":  {avgX: 80, avgY: 260, count: 1},
		"edge":     {avgX: 220, avgY: 260, count: 1},
		"postgres": {avgX: 260, avgY: -180, count: 4},
	}
	demands := map[string]childBoundaryDemand{
		"user": {
			leftCount: 1,
			leftOrder: 220,
			botCount:  1,
			botOrder:  -260,
		},
		"node": {
			botCount: 1,
			botOrder: -60,
		},
		"project": {
			botCount: 1,
			botOrder: 80,
		},
		"edge": {
			rightCount: 1,
			rightOrder: 180,
			botCount:   1,
			botOrder:   220,
		},
		"postgres": {
			topCount:   4,
			topOrder:   0,
			rightCount: 1,
			rightOrder: 0,
		},
	}
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, demands, childEdges, nil)
	if !ok {
		t.Fatal("expected external pull boundary layout to apply")
	}

	if relPos["postgres"][1] > 0 {
		t.Fatalf("expected postgres to stay in upper half for top-boundary demand, got relPos=%v", relPos)
	}
	if relPos["user"][1] < 0 || relPos["edge"][1] < 0 {
		t.Fatalf("expected bottom-demand repos to stay in lower half, got relPos=%v", relPos)
	}
	if relPos["user"][0] > 0 || relPos["edge"][0] < 0 {
		t.Fatalf("expected left/right corner demands to shape repo positions, got relPos=%v", relPos)
	}
}

func TestShouldUseExternalPullBoundaryLayoutRejectsEightNodeGroups(t *testing.T) {
	kids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	pulls := make(map[string]childExternalPull, len(kids))
	for _, id := range kids {
		pulls[id] = childExternalPull{avgX: 280, avgY: 0, count: 1}
	}

	if shouldUseExternalPullBoundaryLayout(kids, nil, pulls) {
		t.Fatal("expected external-pull boundary layout to reject 8-node groups")
	}
}

func TestLayoutChildrenPassThroughCorridorKeepsCenterLaneOpen(t *testing.T) {
	childIDs := []string{"layout", "eventhub", "auth", "waterfall", "impact"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "gateway", TargetID: "auth", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "gateway", TargetID: "eventhub", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "upstream", TargetID: "waterfall", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "auth", TargetID: "impact", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "eventhub", TargetID: "impact", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "waterfall", TargetID: "impact", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "impact", TargetID: "layout", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e8", SourceID: "impact", TargetID: "store", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e9", SourceID: "layout", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
	}
	pulls := map[string]childExternalPull{
		"layout":    {avgX: 280, avgY: -20, count: 1},
		"eventhub":  {avgX: -260, avgY: -20, count: 1},
		"auth":      {avgX: -220, avgY: 0, count: 1},
		"waterfall": {avgX: -240, avgY: 40, count: 1},
		"impact":    {avgX: 260, avgY: 40, count: 1},
	}
	relPos := map[string][2]float64{}
	childEdges := childInternalEdges(childIDs, edges)

	w, h, ok := layoutChildrenPassThroughCorridor(childIDs, edges, relPos, pulls, nil, childEdges, nil)
	if !ok {
		t.Fatal("expected pass-through corridor layout to apply")
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible group size, got w=%v h=%v", w, h)
	}

	for _, id := range childIDs {
		box := childRectBoxAt(relPos[id], 0)
		if box.Top < childRoutePad && box.Bottom > -childRoutePad {
			t.Fatalf("expected center lane to stay open, but %s occupies it: box=%+v relPos=%v", id, box, relPos[id])
		}
	}

	if intersections := countChildRouteNodeIntersections(childIDs, childEdges, relPos); intersections != 0 {
		t.Fatalf("expected pass-through layout to avoid internal node intersections, got %d relPos=%v", intersections, relPos)
	}
}

func TestLayoutChildrenVerticalLineBuildsMainSpineWithSidecar(t *testing.T) {
	childIDs := []string{"layout", "eventhub", "auth", "waterfall", "impact"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "gateway", TargetID: "layout", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "gateway", TargetID: "eventhub", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "gateway", TargetID: "auth", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "gateway", TargetID: "waterfall", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "auth", TargetID: "impact", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "impact", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
	}
	pulls := map[string]childExternalPull{
		"layout":    {avgX: -260, avgY: -160, count: 1},
		"eventhub":  {avgX: -280, avgY: -40, count: 1},
		"auth":      {avgX: -260, avgY: 40, count: 1},
		"waterfall": {avgX: -240, avgY: 180, count: 1},
		"impact":    {avgX: 260, avgY: 140, count: 1},
	}
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	w, h, ok := layoutChildrenVerticalLine(childIDs, edges, relPos, pulls, nil, childEdges, nil)
	if !ok {
		t.Fatal("expected vertical line layout to apply")
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible group size, got w=%v h=%v", w, h)
	}

	colCounts := make(map[float64]int)
	for _, id := range childIDs {
		colCounts[relPos[id][0]]++
	}
	maxCol := 0
	for _, count := range colCounts {
		if count > maxCol {
			maxCol = count
		}
	}
	if maxCol < 3 {
		t.Fatalf("expected vertical line layout to keep a dominant shared column, got relPos=%v", relPos)
	}
	if len(colCounts) < 2 {
		t.Fatalf("expected vertical line layout to use at least two columns, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenVerticalLineRespectsBoundaryPortOrder(t *testing.T) {
	groupID := "services"
	childIDs := []string{"layout", "eventhub", "auth", "waterfall", "impact"}
	nodes := []model.Node{
		{ID: groupID, Title: "Services", Type: model.NodeTypeGroup},
		{ID: "layout", Title: "Layout", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "eventhub", Title: "EventHub", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "auth", Title: "Auth", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "waterfall", Title: "Waterfall", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "impact", Title: "Impact", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "left_top", Title: "Left Top", Type: model.NodeTypeTask},
		{ID: "left_mid", Title: "Left Mid", Type: model.NodeTypeTask},
		{ID: "left_low", Title: "Left Low", Type: model.NodeTypeTask},
		{ID: "left_bottom", Title: "Left Bottom", Type: model.NodeTypeTask},
		{ID: "right_mid", Title: "Right Mid", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "left_top", TargetID: "layout", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "left_mid", TargetID: "eventhub", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "left_low", TargetID: "auth", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "left_bottom", TargetID: "waterfall", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "auth", TargetID: "impact", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "impact", TargetID: "right_mid", EdgeType: model.EdgeTypeDependsOn},
	}
	positions := map[string][2]float64{
		groupID:       {0, 0},
		"left_top":    {-400, -240},
		"left_mid":    {-400, -80},
		"left_low":    {-400, 40},
		"left_bottom": {-400, 220},
		"right_mid":   {400, 60},
	}
	pulls := childExternalPulls(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	demands := buildChildBoundaryDemands(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenVerticalLine(childIDs, edges, relPos, pulls, demands, childEdges, nil)
	if !ok {
		t.Fatal("expected vertical line layout to apply")
	}

	if !(relPos["layout"][1] < relPos["eventhub"][1] &&
		relPos["eventhub"][1] < relPos["auth"][1] &&
		relPos["auth"][1] < relPos["waterfall"][1]) {
		t.Fatalf("expected left-boundary order to be preserved down the spine, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenVerticalLineCanKeepStrongRightPulledNodesOnRightBoundary(t *testing.T) {
	childIDs := []string{"auth", "project", "team", "require", "shared"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "echo", TargetID: "auth", EdgeType: model.EdgeTypeTriggers},
		{ID: "e2", SourceID: "auth", TargetID: "project", EdgeType: model.EdgeTypeTriggers},
		{ID: "e3", SourceID: "auth", TargetID: "team", EdgeType: model.EdgeTypeTriggers},
		{ID: "e4", SourceID: "project", TargetID: "require", EdgeType: model.EdgeTypeTriggers},
		{ID: "e5", SourceID: "shared", TargetID: "node", EdgeType: model.EdgeTypeTriggers},
		{ID: "e6", SourceID: "require", TargetID: "node", EdgeType: model.EdgeTypeTriggers},
		{ID: "e7", SourceID: "require", TargetID: "team_handler", EdgeType: model.EdgeTypeTriggers},
	}
	pulls := map[string]childExternalPull{
		"auth":    {avgX: 320, avgY: -120, count: 1},
		"project": {avgX: 260, avgY: 120, count: 1},
		"team":    {avgX: 320, avgY: 220, count: 1},
		"require": {avgX: 380, avgY: 260, count: 3},
		"shared":  {avgX: 360, avgY: 40, count: 1},
	}
	demands := map[string]childBoundaryDemand{
		"auth": {
			rightOrder: -120,
			rightCount: 1,
		},
		"project": {
			rightOrder: 120,
			rightCount: 1,
		},
		"team": {
			rightOrder: 220,
			rightCount: 1,
		},
		"require": {
			rightOrder: 260,
			rightCount: 3,
		},
		"shared": {
			rightOrder: 40,
			rightCount: 1,
		},
	}
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenVerticalLine(childIDs, edges, relPos, pulls, demands, childEdges, nil)
	if !ok {
		t.Fatal("expected vertical line layout to apply")
	}

	colCounts := make(map[float64]int)
	for _, id := range childIDs {
		colCounts[relPos[id][0]]++
	}
	if len(colCounts) < 2 {
		t.Fatalf("expected right-boundary vertical line layout to use at least two columns, got relPos=%v", relPos)
	}
	if !(relPos["require"][0] > 0 && relPos["shared"][0] > 0) {
		t.Fatalf("expected strongest right-pulled nodes to stay on the right half, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenVerticalLineRejectsDenseBranchyGroup(t *testing.T) {
	childIDs := []string{"a", "b", "c", "d", "e"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "b", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "a", TargetID: "c", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "a", TargetID: "d", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "e", TargetID: "b", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "e", TargetID: "c", EdgeType: model.EdgeTypeDependsOn},
	}
	pulls := map[string]childExternalPull{
		"a": {avgX: -260, avgY: -120, count: 1},
		"b": {avgX: -280, avgY: -20, count: 1},
		"c": {avgX: -240, avgY: 60, count: 1},
		"d": {avgX: 260, avgY: 120, count: 1},
		"e": {avgX: 220, avgY: 180, count: 1},
	}
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenVerticalLine(childIDs, edges, relPos, pulls, nil, childEdges, nil)
	if ok {
		t.Fatalf("expected dense branchy group to skip vertical-line layout, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenTwoColumnFlowBuildsMainAndSinkColumns(t *testing.T) {
	groupID := "services"
	childIDs := []string{"layout", "eventhub", "auth", "waterfall", "impact"}
	nodes := []model.Node{
		{ID: groupID, Title: "Services", Type: model.NodeTypeGroup},
		{ID: "layout", Title: "Layout", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "eventhub", Title: "EventHub", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "auth", Title: "Auth", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "waterfall", Title: "Waterfall", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "impact", Title: "Impact", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "left_top", Title: "Left Top", Type: model.NodeTypeTask},
		{ID: "left_mid", Title: "Left Mid", Type: model.NodeTypeTask},
		{ID: "left_low", Title: "Left Low", Type: model.NodeTypeTask},
		{ID: "store", Title: "Store", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "left_top", TargetID: "layout", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "left_mid", TargetID: "waterfall", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "left_low", TargetID: "auth", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "auth", TargetID: "impact", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "waterfall", TargetID: "eventhub", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "impact", TargetID: "eventhub", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "impact", TargetID: "store", EdgeType: model.EdgeTypeDependsOn},
	}
	positions := map[string][2]float64{
		groupID:    {0, 0},
		"left_top": {-420, -220},
		"left_mid": {-420, 20},
		"left_low": {-420, 220},
		"store":    {-520, -160},
	}
	pulls := childExternalPulls(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	demands := buildChildBoundaryDemands(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	w, h, ok := layoutChildrenTwoColumnFlow(childIDs, edges, relPos, pulls, demands, childEdges, nil)
	if !ok {
		t.Fatal("expected two-column flow layout to apply")
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible group size, got w=%v h=%v", w, h)
	}

	if !(relPos["layout"][0] < relPos["eventhub"][0] &&
		relPos["waterfall"][0] < relPos["eventhub"][0] &&
		relPos["auth"][0] < relPos["impact"][0]) {
		t.Fatalf("expected main-column nodes left of sink/bridge column, got relPos=%v", relPos)
	}
	if !(relPos["layout"][1] < relPos["waterfall"][1] && relPos["waterfall"][1] < relPos["auth"][1]) {
		t.Fatalf("expected left-boundary order to shape main column, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenTwoColumnFlowCanUseExtremeRightSlotsToKeepCenterCorridorOpen(t *testing.T) {
	groupID := "middleware"
	childIDs := []string{"auth", "project", "team", "require", "shared"}
	nodes := []model.Node{
		{ID: groupID, Title: "Middleware", Type: model.NodeTypeGroup},
		{ID: "auth", Title: "Auth Middleware", Type: model.NodeTypeBranch, ParentID: &groupID},
		{ID: "project", Title: "ProjectAccess MW", Type: model.NodeTypeBranch, ParentID: &groupID},
		{ID: "team", Title: "TeamAccess MW", Type: model.NodeTypeBranch, ParentID: &groupID},
		{ID: "require", Title: "RequireRole MW", Type: model.NodeTypeBranch, ParentID: &groupID},
		{ID: "shared", Title: "SharedAccess MW", Type: model.NodeTypeBranch, ParentID: &groupID},
		{ID: "echo", Title: "Echo Router", Type: model.NodeTypeTask},
		{ID: "node", Title: "Node Handler", Type: model.NodeTypeTask},
		{ID: "team_handler", Title: "Team Handler", Type: model.NodeTypeTask},
		{ID: "impact", Title: "Impact Handler", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "echo", TargetID: "auth", EdgeType: model.EdgeTypeTriggers},
		{ID: "e2", SourceID: "auth", TargetID: "project", EdgeType: model.EdgeTypeTriggers},
		{ID: "e3", SourceID: "auth", TargetID: "team", EdgeType: model.EdgeTypeTriggers},
		{ID: "e4", SourceID: "project", TargetID: "require", EdgeType: model.EdgeTypeTriggers},
		{ID: "e5", SourceID: "shared", TargetID: "node", EdgeType: model.EdgeTypeTriggers},
		{ID: "e6", SourceID: "require", TargetID: "node", EdgeType: model.EdgeTypeTriggers},
		{ID: "e7", SourceID: "require", TargetID: "team_handler", EdgeType: model.EdgeTypeTriggers},
		{ID: "e8", SourceID: "require", TargetID: "impact", EdgeType: model.EdgeTypeTriggers},
	}
	positions := map[string][2]float64{
		groupID:        {0, 0},
		"echo":         {520, 60},
		"node":         {560, 80},
		"team_handler": {560, 320},
		"impact":       {560, 480},
	}
	pulls := childExternalPulls(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	demands := buildChildBoundaryDemands(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	childEdges := childInternalEdges(childIDs, edges)
	externalLinks := buildChildExternalLinks(groupID, positions[groupID], childIDs, edges, nodeMap, positions)
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenTwoColumnFlow(childIDs, edges, relPos, pulls, demands, childEdges, externalLinks)
	if !ok {
		t.Fatal("expected two-column flow layout to apply")
	}

	if !(relPos["require"][0] > 0 && relPos["shared"][0] > 0) {
		t.Fatalf("expected right-pulled middleware nodes to stay in right column, got relPos=%v", relPos)
	}
	if math.Abs(relPos["require"][1]-relPos["shared"][1]) < 200 {
		t.Fatalf("expected right-column nodes to use spread slots and keep corridor open, got relPos=%v", relPos)
	}
}

func TestLayoutChildrenHorizontalLineBuildsMainRowWithSidecar(t *testing.T) {
	childIDs := []string{"collect", "normalize", "publish", "sink"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "collect", TargetID: "normalize", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "normalize", TargetID: "publish", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "publish", TargetID: "sink", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "top", TargetID: "collect", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "top", TargetID: "normalize", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "sink", TargetID: "bottom", EdgeType: model.EdgeTypeDependsOn},
	}
	pulls := map[string]childExternalPull{
		"collect":   {avgX: -200, avgY: -260, count: 1},
		"normalize": {avgX: -40, avgY: -240, count: 1},
		"publish":   {avgX: 160, avgY: -220, count: 1},
		"sink":      {avgX: 80, avgY: 260, count: 1},
	}
	childEdges := childInternalEdges(childIDs, edges)
	relPos := map[string][2]float64{}

	w, h, ok := layoutChildrenHorizontalLine(childIDs, edges, relPos, pulls, nil, childEdges, nil)
	if !ok {
		t.Fatal("expected horizontal line layout to apply")
	}
	if w < minGroupW || h < minGroupH {
		t.Fatalf("expected sensible group size, got w=%v h=%v", w, h)
	}

	rowCounts := make(map[float64]int)
	for _, id := range childIDs {
		rowCounts[relPos[id][1]]++
	}
	maxRow := 0
	for _, count := range rowCounts {
		if count > maxRow {
			maxRow = count
		}
	}
	if maxRow < 3 {
		t.Fatalf("expected horizontal line layout to keep a dominant shared row, got relPos=%v", relPos)
	}
}

func TestRenderedChildLayoutCostCountsActualExternalNodeHits(t *testing.T) {
	groupID := "services"
	childIDs := []string{"left", "right"}
	nodes := []model.Node{
		{ID: groupID, Title: "Services", Type: model.NodeTypeGroup},
		{ID: "left", Title: "Left", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "right", Title: "Right", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "external", Title: "External", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "external", TargetID: "right", EdgeType: model.EdgeTypeDependsOn},
	}
	groupPositions := map[string][2]float64{
		groupID:    {0, 0},
		"external": {-240, 0},
	}
	childRelPos := map[string][2]float64{
		"left":  {0, 0},
		"right": {120, 0},
	}

	routes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, childRelPos, childRelPos)
	hits := countRenderedChildRouteNodeIntersections(childIDs, routes, nodeMap, groupPositions, childRelPos, childRelPos)
	if hits != 1 {
		t.Fatalf("expected actual rendered route to cross one sibling node, got %d routes=%+v", hits, routes)
	}

	cost := renderedChildLayoutCost(groupPositions[groupID], childIDs, edges, nodeMap, groupPositions, childRelPos, childRelPos)
	if cost <= 0 {
		t.Fatalf("expected rendered child layout cost to penalize actual external crossing, got %v", cost)
	}
}

func TestOptimizeChildAssignmentsForRenderedRoutesSwapsSlotsToReduceHits(t *testing.T) {
	groupID := "services"
	childIDs := []string{"left", "right"}
	nodes := []model.Node{
		{ID: groupID, Title: "Services", Type: model.NodeTypeGroup},
		{ID: "left", Title: "Left", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "right", Title: "Right", Type: model.NodeTypeTask, ParentID: &groupID},
		{ID: "external", Title: "External", Type: model.NodeTypeTask},
	}
	nodeMap := make(map[string]*model.Node, len(nodes))
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}
	edges := []model.Edge{
		{ID: "e1", SourceID: "external", TargetID: "right", EdgeType: model.EdgeTypeDependsOn},
	}
	groupPositions := map[string][2]float64{
		groupID:    {0, 0},
		"external": {-240, 0},
	}
	childRelPos := map[string][2]float64{
		"left":  {0, 0},
		"right": {120, 0},
	}

	pulls := childExternalPulls(groupID, groupPositions[groupID], childIDs, edges, nodeMap, groupPositions)
	demands := buildChildBoundaryDemands(groupID, groupPositions[groupID], childIDs, edges, nodeMap, groupPositions)
	metrics := computeChildRectMetrics(childIDs, edges, pulls)
	childEdges := childInternalEdges(childIDs, edges)
	externalLinks := buildChildExternalLinks(groupID, groupPositions[groupID], childIDs, edges, nodeMap, groupPositions)

	beforeRoutes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, childRelPos, childRelPos)
	beforeHits := countRenderedChildRouteNodeIntersections(childIDs, beforeRoutes, nodeMap, groupPositions, childRelPos, childRelPos)
	if beforeHits != 1 {
		t.Fatalf("expected initial rendered crossing count to be 1, got %d", beforeHits)
	}

	improved := optimizeChildAssignmentsForRenderedRoutes(
		groupPositions[groupID],
		childIDs,
		edges,
		nodeMap,
		groupPositions,
		childRelPos,
		childRelPos,
		childEdges,
		externalLinks,
		metrics,
		demands,
	)
	if !improved {
		t.Fatal("expected rendered-route optimizer to improve the assignment")
	}

	afterRoutes := buildRenderedChildRouteInfos(childIDs, edges, nodeMap, groupPositions, childRelPos, childRelPos)
	afterHits := countRenderedChildRouteNodeIntersections(childIDs, afterRoutes, nodeMap, groupPositions, childRelPos, childRelPos)
	if afterHits >= beforeHits {
		t.Fatalf("expected fewer rendered node hits after swap, before=%d after=%d relPos=%v", beforeHits, afterHits, childRelPos)
	}
	if !(childRelPos["right"][0] < childRelPos["left"][0]) {
		t.Fatalf("expected externally connected node to move to the left slot, got relPos=%v", childRelPos)
	}
}

func interpolateLineY(src, tgt LayoutPosition, x float64) float64 {
	if tgt.X == src.X {
		return (src.Y + tgt.Y) / 2
	}
	t := (x - src.X) / (tgt.X - src.X)
	return src.Y + (tgt.Y-src.Y)*t
}

func intervalsOverlap(aTop, aBottom, bTop, bBottom float64) bool {
	return math.Min(aBottom, bBottom) > math.Max(aTop, bTop)
}
