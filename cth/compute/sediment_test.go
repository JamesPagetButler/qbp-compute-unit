package compute

import (
	"testing"

	"github.com/helpful-engineering/cth/model"
	"github.com/helpful-engineering/cth/store"
)

func TestSedimentPartitions(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// minimal.json: CH-1 fidelity=1.0 (laminar), CH-2 fidelity=0.99 (low_sediment).
	report := DetectSedimentPartitions(inv)

	if len(report.Laminar) != 1 || report.Laminar[0] != "CH-1" {
		t.Errorf("expected CH-1 in Laminar, got %v", report.Laminar)
	}
	if len(report.LowSediment) != 1 || report.LowSediment[0] != "CH-2" {
		t.Errorf("expected CH-2 in LowSediment, got %v", report.LowSediment)
	}
	if len(report.Moderate) != 0 || len(report.Heavy) != 0 {
		t.Errorf("unexpected Moderate/Heavy entries: %v / %v", report.Moderate, report.Heavy)
	}
}

func TestSedimentSharpPartition(t *testing.T) {
	// Build an inventory with one clean chain (lean domain) and one heavy chain (rebco domain).
	inv := model.Inventory{
		Programme: "TestSharp",
		Axioms: []model.Anchor{
			{ID: "AX-1", Domain: "math", Tier: model.Axiom},
		},
		DerivedPrinciples: []model.Anchor{
			{ID: "PR-1", Domain: "lean", Tier: model.Proof},
			{ID: "PR-2", Domain: "rebco", Tier: model.Proof},
		},
		Chains: []model.Chain{
			{ID: "CH-clean", SourceIDs: []string{"AX-1"}, TargetID: "PR-1", Fidelity: 1.0},
			{ID: "CH-heavy", SourceIDs: []string{"AX-1"}, TargetID: "PR-2", Fidelity: 0.5},
		},
	}

	report := DetectSedimentPartitions(inv)

	if !report.SharpPartition {
		t.Errorf("expected sharp partition: dirty=%v clean=%v", report.DirtyOnlyDomains, report.CleanOnlyDomains)
	}

	foundRebco := false
	for _, d := range report.DirtyOnlyDomains {
		if d == "rebco" {
			foundRebco = true
		}
	}
	if !foundRebco {
		t.Errorf("expected rebco in DirtyOnlyDomains, got %v", report.DirtyOnlyDomains)
	}
}
