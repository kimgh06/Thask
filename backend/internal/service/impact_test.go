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

// TestImpact_Bidirectional: Both source→target and target→source traversal for related edges.
func TestImpact_Bidirectional(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "B", TargetID: "A", EdgeType: model.EdgeTypeRelated},
	}

	result := ComputeImpact([]string{"A"}, edges, 1)

	if !containsID(result, "B") {
		t.Errorf("expected B in result via reverse traversal, got %v", result)
	}
}

// TestImpact_BlocksForwardOnly: blocks propagates source→target only.
func TestImpact_BlocksForwardOnly(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
	}

	// A changed → B affected (forward)
	result := ComputeImpact([]string{"A"}, edges, 1)
	if !containsID(result, "B") {
		t.Errorf("expected B in result when A changed via blocks, got %v", result)
	}

	// B changed → nothing (no backward propagation)
	result2 := ComputeImpact([]string{"B"}, edges, 1)
	if len(result2) != 0 {
		t.Errorf("expected empty result when B changed via blocks, got %v", result2)
	}
}

// TestImpact_DependsOnBackward: depends_on propagates target→source only.
// A --depends_on--> B means A depends on B. If B changes, A is affected.
func TestImpact_DependsOnBackward(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeDependsOn},
	}

	// B changed → A affected (backward)
	result := ComputeImpact([]string{"B"}, edges, 1)
	if !containsID(result, "A") {
		t.Errorf("expected A in result when B changed via depends_on, got %v", result)
	}

	// A changed → nothing (no forward propagation)
	result2 := ComputeImpact([]string{"A"}, edges, 1)
	if len(result2) != 0 {
		t.Errorf("expected empty result when A changed via depends_on, got %v", result2)
	}
}

// TestImpact_TriggersForwardOnly: triggers propagates source→target only.
func TestImpact_TriggersForwardOnly(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeTriggers},
	}

	// A changed → B affected (forward)
	result := ComputeImpact([]string{"A"}, edges, 1)
	if !containsID(result, "B") {
		t.Errorf("expected B in result when A changed via triggers, got %v", result)
	}

	// B changed → nothing
	result2 := ComputeImpact([]string{"B"}, edges, 1)
	if len(result2) != 0 {
		t.Errorf("expected empty result when B changed via triggers, got %v", result2)
	}
}

// TestImpact_ParentChildSkipped: parent_child edges are not traversed.
func TestImpact_ParentChildSkipped(t *testing.T) {
	edges := []model.Edge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeParentChild},
	}

	result := ComputeImpact([]string{"A"}, edges, 1)
	if len(result) != 0 {
		t.Errorf("expected empty result for parent_child from A, got %v", result)
	}

	result2 := ComputeImpact([]string{"B"}, edges, 1)
	if len(result2) != 0 {
		t.Errorf("expected empty result for parent_child from B, got %v", result2)
	}
}
