package service

import (
	"testing"

	"github.com/thask/backend/internal/model"
)

func makeTestEdges() []model.Edge {
	return []model.Edge{
		{SourceID: "main", TargetID: "handler", EdgeType: "depends_on"},
		{SourceID: "main", TargetID: "auth", EdgeType: "depends_on"},
		{SourceID: "handler", TargetID: "service", EdgeType: "depends_on"},
		{SourceID: "handler", TargetID: "auth", EdgeType: "depends_on"},
		{SourceID: "service", TargetID: "model", EdgeType: "depends_on"},
		{SourceID: "service", TargetID: "auth", EdgeType: "depends_on"},
		{SourceID: "auth", TargetID: "service", EdgeType: "depends_on"},
		{SourceID: "model", TargetID: "auth", EdgeType: "depends_on"},
	}
}

func TestDetectCycles(t *testing.T) {
	cycles := DetectCycles(makeTestEdges())
	if len(cycles) == 0 {
		t.Fatal("expected cycles, got none")
	}
	// Verify each cycle is a valid loop in the adjacency list
	adj := make(map[string]map[string]bool)
	for _, e := range makeTestEdges() {
		if e.EdgeType == "depends_on" || e.EdgeType == "blocks" {
			if adj[e.SourceID] == nil {
				adj[e.SourceID] = make(map[string]bool)
			}
			adj[e.SourceID][e.TargetID] = true
		}
	}
	for i, cycle := range cycles {
		if len(cycle) < 2 {
			t.Errorf("cycle %d too short: %v", i, cycle)
			continue
		}
		for j := 0; j < len(cycle); j++ {
			src := cycle[j]
			tgt := cycle[(j+1)%len(cycle)]
			if !adj[src][tgt] {
				t.Errorf("cycle %d: edge %s -> %s does not exist", i, src, tgt)
			}
		}
	}
	t.Logf("found %d valid cycles", len(cycles))
}

func TestDetectCyclesDAG(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "a", TargetID: "b", EdgeType: "depends_on"},
		{SourceID: "b", TargetID: "c", EdgeType: "depends_on"},
		{SourceID: "a", TargetID: "c", EdgeType: "depends_on"},
	}
	cycles := DetectCycles(edges)
	if len(cycles) != 0 {
		t.Errorf("expected no cycles in DAG, got %d", len(cycles))
	}
}

func TestDetectCyclesIgnoresRelated(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "a", TargetID: "b", EdgeType: "depends_on"},
		{SourceID: "b", TargetID: "a", EdgeType: "related"},
	}
	cycles := DetectCycles(edges)
	if len(cycles) != 0 {
		t.Errorf("expected no cycles (related edge ignored), got %d", len(cycles))
	}
}

func TestDetectCyclesSelfLoop(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "a", TargetID: "a", EdgeType: "depends_on"},
	}
	cycles := DetectCycles(edges)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 self-loop cycle, got %d", len(cycles))
	}
	if len(cycles[0]) != 1 || cycles[0][0] != "a" {
		t.Errorf("expected cycle [a], got %v", cycles[0])
	}
}

func TestCriticalPath(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "a", TargetID: "b", EdgeType: "depends_on"},
		{SourceID: "b", TargetID: "c", EdgeType: "depends_on"},
		{SourceID: "c", TargetID: "d", EdgeType: "depends_on"},
		{SourceID: "a", TargetID: "d", EdgeType: "depends_on"},
	}
	result := CriticalPath(edges)
	if len(result.Path) != 4 {
		t.Errorf("expected path length 4, got %d: %v", len(result.Path), result.Path)
	}
	if len(result.SkippedCycleNodes) != 0 {
		t.Errorf("expected no skipped nodes in DAG, got %v", result.SkippedCycleNodes)
	}
}

func TestCriticalPathEmpty(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "a", TargetID: "b", EdgeType: "related"},
	}
	result := CriticalPath(edges)
	if len(result.Path) != 0 {
		t.Errorf("expected empty path, got %v", result.Path)
	}
}

func TestCriticalPathWithCycles(t *testing.T) {
	result := CriticalPath(makeTestEdges())
	// Cycle-participating nodes should be reported
	if len(result.SkippedCycleNodes) == 0 {
		t.Error("expected skipped cycle nodes, got none")
	}
	// Verify skipped nodes are actually in the edge set
	nodeSet := make(map[string]bool)
	for _, e := range makeTestEdges() {
		nodeSet[e.SourceID] = true
		nodeSet[e.TargetID] = true
	}
	for _, n := range result.SkippedCycleNodes {
		if !nodeSet[n] {
			t.Errorf("skipped node %s not in edge set", n)
		}
	}
	t.Logf("critical path: %v, skipped: %v", result.Path, result.SkippedCycleNodes)
}
