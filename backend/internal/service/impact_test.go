package service

import (
	"testing"

	"github.com/thask/backend/internal/model"
)

func containsID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// TestImpact_NoEdges: No edges returns empty result.
func TestImpact_NoEdges(t *testing.T) {
	result := ComputeImpact([]string{"A"}, nil, 3)
	if len(result) != 0 {
		t.Errorf("expected empty result with no edges, got %v", result)
	}
}

// TestImpact_SingleDepth: One level of connections.
func TestImpact_SingleDepth(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
		{SourceID: "A", TargetID: "C", EdgeType: model.EdgeTypeBlocks},
	}

	result := ComputeImpact([]string{"A"}, edges, 1)

	if !containsID(result, "B") {
		t.Errorf("expected B in result, got %v", result)
	}
	if !containsID(result, "C") {
		t.Errorf("expected C in result, got %v", result)
	}
	if containsID(result, "A") {
		t.Errorf("changed node A should not appear in result")
	}
}

// TestImpact_MultiDepth: Two levels deep.
func TestImpact_MultiDepth(t *testing.T) {
	// A -> B -> C
	edges := []model.Edge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
		{SourceID: "B", TargetID: "C", EdgeType: model.EdgeTypeBlocks},
	}

	result := ComputeImpact([]string{"A"}, edges, 2)

	if !containsID(result, "B") {
		t.Errorf("expected B in result, got %v", result)
	}
	if !containsID(result, "C") {
		t.Errorf("expected C in result, got %v", result)
	}
}

// TestImpact_Bidirectional: Both source→target and target→source traversal.
func TestImpact_Bidirectional(t *testing.T) {
	// Edge between B and A. A is changed. Both directions should be traversed.
	edges := []model.Edge{
		{SourceID: "B", TargetID: "A", EdgeType: model.EdgeTypeRelated},
	}

	result := ComputeImpact([]string{"A"}, edges, 1)

	if !containsID(result, "B") {
		t.Errorf("expected B in result via reverse traversal, got %v", result)
	}
}
