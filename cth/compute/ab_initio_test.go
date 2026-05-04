package compute

import (
	"math"
	"testing"

	"github.com/helpful-engineering/cth/model"
)

func TestAbInitioScore(t *testing.T) {
	// Two paths to MEAS-1:
	//   CH-A: fidelity=0.99, source=PR-1 (not input)  → score = 0.99/1 = 0.99
	//   CH-B: fidelity=0.95, source=INST-1 (input)    → score = 0.95/2 = 0.475
	// CH-A should win.
	inv := model.Inventory{
		Programme: "AbInitioTest",
		Inputs: []model.Anchor{
			{ID: "INST-1", Tier: model.Prediction, Provenance: model.Input},
		},
		DerivedPrinciples: []model.Anchor{
			{ID: "PR-1", Tier: model.Proof},
		},
		Anchors: []model.Anchor{
			{ID: "MEAS-1", Tier: model.Measurement},
		},
		Chains: []model.Chain{
			{ID: "CH-A", SourceIDs: []string{"PR-1"}, TargetID: "MEAS-1", Fidelity: 0.99},
			{ID: "CH-B", SourceIDs: []string{"INST-1"}, TargetID: "MEAS-1", Fidelity: 0.95},
		},
	}

	results := AbInitioScore(inv)

	if len(results) != 1 {
		t.Fatalf("expected 1 result (MEAS-1), got %d", len(results))
	}
	r := results[0]
	if r.AnchorID != "MEAS-1" {
		t.Errorf("AnchorID = %s; want MEAS-1", r.AnchorID)
	}
	if r.BestChainID != "CH-A" {
		t.Errorf("BestChainID = %s; want CH-A", r.BestChainID)
	}
	if math.Abs(r.Score-0.99) > 0.001 {
		t.Errorf("Score = %v; want 0.99", r.Score)
	}
	if r.InputCount != 0 {
		t.Errorf("InputCount = %d; want 0", r.InputCount)
	}
}

func TestAbInitioScoreSinglePath(t *testing.T) {
	// Single-path anchors should not appear in results.
	inv := model.Inventory{
		Programme: "SinglePath",
		Anchors:   []model.Anchor{{ID: "MEAS-1"}},
		Chains: []model.Chain{
			{ID: "CH-A", SourceIDs: []string{"AX-1"}, TargetID: "MEAS-1", Fidelity: 0.99},
		},
	}

	results := AbInitioScore(inv)
	if len(results) != 0 {
		t.Errorf("expected 0 results for single-path anchor, got %d", len(results))
	}
}
