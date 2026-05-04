package compute

import (
	"testing"

	"github.com/helpful-engineering/cth/model"
)

func TestLocaliseIncoherence(t *testing.T) {
	// Build a small chain: AX-1 → PR-1 → MEAS-bad (incoherent)
	// with a coherent confluence at PR-1.
	inv := model.Inventory{
		Programme: "LocaliseTest",
		Axioms: []model.Anchor{
			{ID: "AX-1", Tier: model.Axiom, Status: model.Coherent, Domain: "math"},
		},
		DerivedPrinciples: []model.Anchor{
			{ID: "PR-1", Tier: model.Proof, Status: model.Coherent, Domain: "lean"},
		},
		Anchors: []model.Anchor{
			{ID: "MEAS-bad", Tier: model.Measurement, Status: model.Incoherent, Domain: "lab"},
		},
		Chains: []model.Chain{
			{ID: "CH-1", SourceIDs: []string{"AX-1"}, TargetID: "PR-1", Fidelity: 1.0},
			{ID: "CH-2", SourceIDs: []string{"PR-1"}, TargetID: "MEAS-bad", Fidelity: 0.7},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{
				ID:       "CP-1",
				AnchorID: "PR-1",
				Status:   model.Coherent,
				Paths: []model.ChainRef{
					{ChainID: "CH-1", Provenance: model.Internal},
					{ChainID: "EXT", Provenance: model.External},
				},
			},
		},
	}

	result := LocaliseIncoherence("MEAS-bad", inv)

	if !result.Found {
		t.Fatal("expected incoherence to be found")
	}
	if result.ErrorEnd != "MEAS-bad" {
		t.Errorf("ErrorEnd = %s; want MEAS-bad", result.ErrorEnd)
	}
	// The walk should stop at PR-1 because it has a coherent confluence.
	if result.LastCoherentConfluence != "CP-1" {
		t.Errorf("LastCoherentConfluence = %s; want CP-1", result.LastCoherentConfluence)
	}
	// Weakest link is CH-2 (fidelity 0.7).
	if result.WeakestLinkID != "CH-2" {
		t.Errorf("WeakestLinkID = %s; want CH-2", result.WeakestLinkID)
	}
}

func TestLocaliseCoherentAnchor(t *testing.T) {
	inv := model.Inventory{
		Programme: "LocaliseTest",
		Anchors: []model.Anchor{
			{ID: "MEAS-ok", Tier: model.Measurement, Status: model.Coherent},
		},
	}
	result := LocaliseIncoherence("MEAS-ok", inv)
	if result.Found {
		t.Error("should not find incoherence in a coherent anchor")
	}
}
