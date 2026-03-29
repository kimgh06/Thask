package service

import (
	"fmt"
	"testing"

	"github.com/thask/backend/internal/model"
)

func make1000NodeGraph() []model.Edge {
	edges := make([]model.Edge, 0, 2000)

	// Create a long chain: node0 -> node1 -> ... -> node999
	for i := 0; i < 999; i++ {
		edges = append(edges, model.Edge{
			ID:       fmt.Sprintf("edge-chain-%d", i),
			SourceID: fmt.Sprintf("node%04d", i),
			TargetID: fmt.Sprintf("node%04d", i+1),
			EdgeType: model.EdgeTypeDependsOn,
		})
	}

	// Add some cross edges (shortcuts)
	for i := 0; i < 500; i++ {
		edges = append(edges, model.Edge{
			ID:       fmt.Sprintf("edge-cross-%d", i),
			SourceID: fmt.Sprintf("node%04d", i),
			TargetID: fmt.Sprintf("node%04d", i+100),
			EdgeType: model.EdgeTypeDependsOn,
		})
	}

	return edges
}

func make1000NodeGraphWithCycles() []model.Edge {
	edges := make1000NodeGraph()

	// Add 10 cycles
	for i := 0; i < 10; i++ {
		base := i * 100
		edges = append(edges, model.Edge{
			ID:       fmt.Sprintf("edge-cycle-%d", i),
			SourceID: fmt.Sprintf("node%04d", base+5),
			TargetID: fmt.Sprintf("node%04d", base),
			EdgeType: model.EdgeTypeDependsOn,
		})
	}

	return edges
}

func BenchmarkDetectCycles1000Nodes(b *testing.B) {
	edges := make1000NodeGraphWithCycles()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectCycles(edges)
	}
}

func BenchmarkDetectCyclesDAG1000Nodes(b *testing.B) {
	edges := make1000NodeGraph() // no cycles
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectCycles(edges)
	}
}

func BenchmarkCriticalPath1000Nodes(b *testing.B) {
	edges := make1000NodeGraph()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CriticalPath(edges)
	}
}
