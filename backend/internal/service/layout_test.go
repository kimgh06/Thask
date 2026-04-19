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
	if w >= 360 || h >= 360 {
		t.Fatalf("expected compact rectangular layout, got w=%v h=%v", w, h)
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
