package service

import (
	"testing"
	"time"

	"github.com/thask/backend/internal/model"
)

func strPtr(s string) *string { return &s }

func findChange(changes []StatusChange, nodeID string) *StatusChange {
	for i := range changes {
		if changes[i].NodeID == nodeID {
			return &changes[i]
		}
	}
	return nil
}

// TestWaterfall_NoChangeForNonPassFail: IN_PROGRESS and BLOCKED don't propagate.
func TestWaterfall_NoChangeForNonPassFail(t *testing.T) {
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusInProgress},
		{ID: "B", Status: model.NodeStatusInProgress},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
	}

	for _, status := range []model.NodeStatus{model.NodeStatusInProgress, model.NodeStatusBlocked} {
		changes := ComputeWaterfall("A", status, nodes, edges)
		if len(changes) != 0 {
			t.Errorf("expected no changes for status %q, got %d", status, len(changes))
		}
	}
}

// TestWaterfall_BlocksPropagation: A fails → B (which A blocks) becomes BLOCKED.
func TestWaterfall_BlocksPropagation(t *testing.T) {
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusInProgress},
		{ID: "B", Status: model.NodeStatusInProgress},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
	}

	changes := ComputeWaterfall("A", model.NodeStatusFail, nodes, edges)

	c := findChange(changes, "B")
	if c == nil {
		t.Fatal("expected a status change for node B")
	}
	if c.NewStatus != model.NodeStatusBlocked {
		t.Errorf("expected B to become BLOCKED, got %q", c.NewStatus)
	}
	if c.OldStatus != model.NodeStatusInProgress {
		t.Errorf("expected B old status IN_PROGRESS, got %q", c.OldStatus)
	}
}

// TestWaterfall_BlocksResolution: A passes → B (which A blocks, was BLOCKED) becomes IN_PROGRESS.
func TestWaterfall_BlocksResolution(t *testing.T) {
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusInProgress},
		{ID: "B", Status: model.NodeStatusBlocked},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
	}

	changes := ComputeWaterfall("A", model.NodeStatusPass, nodes, edges)

	c := findChange(changes, "B")
	if c == nil {
		t.Fatal("expected a status change for node B")
	}
	if c.NewStatus != model.NodeStatusInProgress {
		t.Errorf("expected B to become IN_PROGRESS, got %q", c.NewStatus)
	}
}

// TestWaterfall_DependsOnPropagation: B fails → A (depends_on B) becomes BLOCKED.
func TestWaterfall_DependsOnPropagation(t *testing.T) {
	// Edge: A --depends_on--> B (A depends on B)
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusInProgress},
		{ID: "B", Status: model.NodeStatusInProgress},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeDependsOn},
	}

	changes := ComputeWaterfall("B", model.NodeStatusFail, nodes, edges)

	c := findChange(changes, "A")
	if c == nil {
		t.Fatal("expected a status change for node A")
	}
	if c.NewStatus != model.NodeStatusBlocked {
		t.Errorf("expected A to become BLOCKED, got %q", c.NewStatus)
	}
}

// TestWaterfall_DependsOnAllMet: All dependencies pass → dependent unblocks.
func TestWaterfall_DependsOnAllMet(t *testing.T) {
	// A depends_on B and C. C is already PASS. B transitions to PASS → A unblocks.
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusBlocked},
		{ID: "B", Status: model.NodeStatusInProgress},
		{ID: "C", Status: model.NodeStatusPass},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeDependsOn},
		{SourceID: "A", TargetID: "C", EdgeType: model.EdgeTypeDependsOn},
	}

	changes := ComputeWaterfall("B", model.NodeStatusPass, nodes, edges)

	c := findChange(changes, "A")
	if c == nil {
		t.Fatal("expected a status change for node A")
	}
	if c.NewStatus != model.NodeStatusInProgress {
		t.Errorf("expected A to become IN_PROGRESS, got %q", c.NewStatus)
	}
}

// TestWaterfall_TriggersPropagation: A passes → B (triggered by A, was BLOCKED) becomes IN_PROGRESS.
func TestWaterfall_TriggersPropagation(t *testing.T) {
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusInProgress},
		{ID: "B", Status: model.NodeStatusBlocked},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeTriggers},
	}

	changes := ComputeWaterfall("A", model.NodeStatusPass, nodes, edges)

	c := findChange(changes, "B")
	if c == nil {
		t.Fatal("expected a status change for node B")
	}
	if c.NewStatus != model.NodeStatusInProgress {
		t.Errorf("expected B to become IN_PROGRESS, got %q", c.NewStatus)
	}
}

// TestWaterfall_ParentAggregation: All children PASS → parent becomes PASS.
func TestWaterfall_ParentAggregation(t *testing.T) {
	parent := "P"
	nodes := []WaterfallNode{
		{ID: "P", Status: model.NodeStatusInProgress},
		{ID: "C1", Status: model.NodeStatusInProgress, ParentID: strPtr(parent)},
		{ID: "C2", Status: model.NodeStatusPass, ParentID: strPtr(parent)},
	}
	edges := []WaterfallEdge{}

	// C1 transitions to PASS → all children are now PASS → parent should become PASS.
	changes := ComputeWaterfall("C1", model.NodeStatusPass, nodes, edges)

	c := findChange(changes, "P")
	if c == nil {
		t.Fatal("expected a status change for parent P")
	}
	if c.NewStatus != model.NodeStatusPass {
		t.Errorf("expected P to become PASS, got %q", c.NewStatus)
	}
}

// TestWaterfall_ParentFailOnAnyChildFail: Any child FAIL → parent becomes FAIL.
func TestWaterfall_ParentFailOnAnyChildFail(t *testing.T) {
	parent := "P"
	nodes := []WaterfallNode{
		{ID: "P", Status: model.NodeStatusInProgress},
		{ID: "C1", Status: model.NodeStatusInProgress, ParentID: strPtr(parent)},
		{ID: "C2", Status: model.NodeStatusPass, ParentID: strPtr(parent)},
	}
	edges := []WaterfallEdge{}

	changes := ComputeWaterfall("C1", model.NodeStatusFail, nodes, edges)

	c := findChange(changes, "P")
	if c == nil {
		t.Fatal("expected a status change for parent P")
	}
	if c.NewStatus != model.NodeStatusFail {
		t.Errorf("expected P to become FAIL, got %q", c.NewStatus)
	}
}

// TestWaterfall_MaxDepthLimit: Chain longer than 10 stops propagating.
func TestWaterfall_MaxDepthLimit(t *testing.T) {
	// Build a chain: N0 --blocks--> N1 --blocks--> ... --blocks--> N15
	// All nodes start IN_PROGRESS. N0 fails. Propagation should stop at depth 10.
	const chainLen = 15
	ids := make([]string, chainLen+1)
	for i := 0; i <= chainLen; i++ {
		ids[i] = string(rune('A' + i))
	}

	nodes := make([]WaterfallNode, chainLen+1)
	for i := 0; i <= chainLen; i++ {
		nodes[i] = WaterfallNode{ID: ids[i], Status: model.NodeStatusInProgress}
	}
	edges := make([]WaterfallEdge, chainLen)
	for i := 0; i < chainLen; i++ {
		edges[i] = WaterfallEdge{
			SourceID: ids[i],
			TargetID: ids[i+1],
			EdgeType: model.EdgeTypeBlocks,
		}
	}

	changes := ComputeWaterfall(ids[0], model.NodeStatusFail, nodes, edges)

	for _, c := range changes {
		// Find the index of this node id in our ids slice.
		idx := -1
		for i, id := range ids {
			if id == c.NodeID {
				idx = i
				break
			}
		}
		// The changed node is at depth idx from the root (ids[0]).
		// Propagation stops when depth >= maxWaterfallDepth (10), so index > 10 should not appear.
		if idx > maxWaterfallDepth {
			t.Errorf("node %s (chain index %d) was changed beyond max depth %d", c.NodeID, idx, maxWaterfallDepth)
		}
	}
}

// TestWaterfall_NoCycle: Circular edges don't cause infinite loop.
func TestWaterfall_NoCycle(t *testing.T) {
	// A --blocks--> B --blocks--> A (cycle)
	nodes := []WaterfallNode{
		{ID: "A", Status: model.NodeStatusInProgress},
		{ID: "B", Status: model.NodeStatusInProgress},
	}
	edges := []WaterfallEdge{
		{SourceID: "A", TargetID: "B", EdgeType: model.EdgeTypeBlocks},
		{SourceID: "B", TargetID: "A", EdgeType: model.EdgeTypeBlocks},
	}

	done := make(chan []StatusChange, 1)
	go func() {
		done <- ComputeWaterfall("A", model.NodeStatusFail, nodes, edges)
	}()

	select {
	case changes := <-done:
		// A should not appear as a cascaded change — the visited guard prevents cycles.
		aChange := findChange(changes, "A")
		if aChange != nil {
			t.Errorf("A should not appear as a cascaded change (cycle guard)")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ComputeWaterfall timed out — likely an infinite loop")
	}
}
