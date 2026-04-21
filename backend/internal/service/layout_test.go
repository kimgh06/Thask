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

	w, h, ok := layoutChildrenRectangular(childIDs, edges, relPos, pulls)
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
	if !(api[0] > 0 && api[1] > centerY) {
		t.Fatalf("expected api in lower-right quadrant, got %+v", api)
	}
	if !(math.Abs(mcp[0]) < math.Abs(api[0]) && mcp[1] > centerY) {
		t.Fatalf("expected mcp in lower middle area, got %+v", mcp)
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

	w, h := expandChildLayoutUntilClear(childIDs, childEdges, relPos)
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

func TestLayoutChildrenExternalPullBoundaryLeavesTopMostlyOpen(t *testing.T) {
	childIDs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "a", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "b", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "c", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e4", SourceID: "d", TargetID: "repo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e5", SourceID: "e", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e6", SourceID: "f", TargetID: "svc", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e7", SourceID: "g", TargetID: "mw", EdgeType: model.EdgeTypeTriggers},
		{ID: "e8", SourceID: "h", TargetID: "mw", EdgeType: model.EdgeTypeTriggers},
	}
	pulls := map[string]childExternalPull{
		"a": {avgX: 200, avgY: 360, count: 1},
		"b": {avgX: 120, avgY: 360, count: 1},
		"c": {avgX: 0, avgY: 420, count: 1},
		"d": {avgX: -40, avgY: 420, count: 1},
		"e": {avgX: 240, avgY: 360, count: 1},
		"f": {avgX: 160, avgY: 360, count: 1},
		"g": {avgX: -200, avgY: 420, count: 1},
		"h": {avgX: -240, avgY: 420, count: 1},
	}
	relPos := map[string][2]float64{}

	w, h, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, nil)
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
	if bottomCount < 5 {
		t.Fatalf("expected most nodes to sit in lower half, got bottomCount=%d relPos=%v", bottomCount, relPos)
	}
}

func TestLayoutChildrenExternalPullBoundaryBackendLikeCase(t *testing.T) {
	childIDs := []string{"project", "impact", "echo", "auth", "edge", "team", "event", "node"}
	edges := []model.Edge{
		{ID: "e1", SourceID: "api_client", TargetID: "echo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e2", SourceID: "cli_client", TargetID: "echo", EdgeType: model.EdgeTypeDependsOn},
		{ID: "e3", SourceID: "echo", TargetID: "authmw", EdgeType: model.EdgeTypeTriggers},
		{ID: "e4", SourceID: "auth", TargetID: "authsvc", EdgeType: model.EdgeTypeDependsOn},
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
		"auth":    {avgX: 360, avgY: 400, count: 1},
		"edge":    {avgX: -40, avgY: 453.3, count: 3},
		"team":    {avgX: -240, avgY: 480, count: 2},
		"event":   {avgX: -60, avgY: 440, count: 2},
		"node":    {avgX: 20, avgY: 440, count: 6},
	}
	relPos := map[string][2]float64{}

	_, _, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, nil)
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

	w, _, ok := layoutChildrenExternalPullBoundary(childIDs, edges, relPos, pulls, nil)
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

	w, h, ok := layoutChildrenPassThroughCorridor(childIDs, edges, relPos, pulls, nil, childEdges)
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

	w, h, ok := layoutChildrenVerticalLine(childIDs, edges, relPos, pulls, nil, childEdges)
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

	_, _, ok := layoutChildrenVerticalLine(childIDs, edges, relPos, pulls, demands, childEdges)
	if !ok {
		t.Fatal("expected vertical line layout to apply")
	}

	if !(relPos["layout"][1] < relPos["eventhub"][1] &&
		relPos["eventhub"][1] < relPos["auth"][1] &&
		relPos["auth"][1] < relPos["waterfall"][1]) {
		t.Fatalf("expected left-boundary order to be preserved down the spine, got relPos=%v", relPos)
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

	w, h, ok := layoutChildrenHorizontalLine(childIDs, edges, relPos, pulls, nil, childEdges)
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
