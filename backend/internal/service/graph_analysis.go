package service

import (
	"os"
	"strconv"

	"github.com/thask/backend/internal/model"
)

// MaxAnalysisEdges is the maximum number of edges allowed for graph analysis.
// Configurable via THASK_MAX_ANALYSIS_EDGES env var.
var MaxAnalysisEdges = func() int {
	if v := os.Getenv("THASK_MAX_ANALYSIS_EDGES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 10000
}()

// DetectCycles finds all cycles in the dependency graph using DFS coloring
// with stack-based path extraction (no parent map).
// Only traverses depends_on and blocks edges.
func DetectCycles(edges []model.Edge) [][]string {
	adj := make(map[string][]string)
	for _, e := range edges {
		if e.EdgeType == "depends_on" || e.EdgeType == "blocks" {
			adj[e.SourceID] = append(adj[e.SourceID], e.TargetID)
		}
	}

	nodeSet := make(map[string]bool)
	for _, e := range edges {
		nodeSet[e.SourceID] = true
		nodeSet[e.TargetID] = true
	}

	color := make(map[string]int) // 0=white, 1=gray, 2=black
	var cycles [][]string
	var path []string

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = 1
		path = append(path, u)
		for _, v := range adj[u] {
			if color[v] == 0 {
				dfs(v)
			} else if color[v] == 1 {
				// Back edge: extract cycle from path
				for i := len(path) - 1; i >= 0; i-- {
					if path[i] == v {
						cycle := make([]string, len(path)-i)
						copy(cycle, path[i:])
						cycles = append(cycles, cycle)
						break
					}
				}
			}
		}
		path = path[:len(path)-1]
		color[u] = 2
	}

	for node := range nodeSet {
		if color[node] == 0 {
			dfs(node)
		}
	}

	return cycles
}

// CriticalPathResult contains the longest path and any nodes excluded due to cycles.
type CriticalPathResult struct {
	Path             []string `json:"criticalPath"`
	SkippedCycleNodes []string `json:"skippedCycleNodes,omitempty"`
}

// CriticalPath finds the longest dependency chain by edge count.
// Uses Kahn's topological sort with longest-path DP.
// Nodes participating in cycles are excluded and reported in SkippedCycleNodes.
func CriticalPath(edges []model.Edge) CriticalPathResult {
	adj := make(map[string][]string)
	inDeg := make(map[string]int)
	nodeSet := make(map[string]bool)

	for _, e := range edges {
		if e.EdgeType != "depends_on" && e.EdgeType != "blocks" {
			continue
		}
		adj[e.SourceID] = append(adj[e.SourceID], e.TargetID)
		inDeg[e.TargetID]++
		nodeSet[e.SourceID] = true
		nodeSet[e.TargetID] = true
	}

	for n := range nodeSet {
		if _, ok := inDeg[n]; !ok {
			inDeg[n] = 0
		}
	}

	// Kahn's topological sort
	var queue []string
	for n := range nodeSet {
		if inDeg[n] == 0 {
			queue = append(queue, n)
		}
	}

	dist := make(map[string]int)
	prev := make(map[string]string)
	var topoOrder []string

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		topoOrder = append(topoOrder, u)

		for _, v := range adj[u] {
			if dist[u]+1 > dist[v] {
				dist[v] = dist[u] + 1
				prev[v] = u
			}
			inDeg[v]--
			if inDeg[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	// Detect skipped cycle nodes
	var skippedCycleNodes []string
	if len(topoOrder) < len(nodeSet) {
		topoSet := make(map[string]bool, len(topoOrder))
		for _, n := range topoOrder {
			topoSet[n] = true
		}
		for n := range nodeSet {
			if !topoSet[n] {
				skippedCycleNodes = append(skippedCycleNodes, n)
			}
		}
	}

	if len(topoOrder) == 0 {
		return CriticalPathResult{SkippedCycleNodes: skippedCycleNodes}
	}

	maxDist := 0
	endNode := ""
	for n, d := range dist {
		if d > maxDist {
			maxDist = d
			endNode = n
		}
	}

	if maxDist == 0 {
		return CriticalPathResult{SkippedCycleNodes: skippedCycleNodes}
	}

	path := []string{endNode}
	cur := endNode
	for prev[cur] != "" {
		cur = prev[cur]
		path = append([]string{cur}, path...)
	}

	return CriticalPathResult{Path: path, SkippedCycleNodes: skippedCycleNodes}
}
