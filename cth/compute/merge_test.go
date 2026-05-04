package compute

import (
	"testing"

	"github.com/helpful-engineering/cth/store"
)

func TestMergeProgrammes(t *testing.T) {
	// 1. Load fixtures
	invA, err := store.LoadInventory("../testdata/qbp_v3_2.json")
	if err != nil {
		t.Fatalf("Load A failed: %v", err)
	}

	invB, err := store.LoadInventory("../testdata/qbp_quantum_v0_1.json")
	if err != nil {
		t.Fatalf("Load B failed: %v", err)
	}

	// 2. Perform merge
	merged, report := MergeProgrammes(invA, invB)

	// 3. Verify report
	// QBP and QBP-Quantum in my testdata don't share IDs yet, 
	// so shared should be 0 unless I update them.
	if len(report.SharedAnchorIDs) != 0 {
		t.Errorf("expected 0 shared anchors, got %d", len(report.SharedAnchorIDs))
	}

	// Check combined program name
	if merged.Programme != "QBP + QBP-Quantum" {
		t.Errorf("expected combined programme name, got '%s'", merged.Programme)
	}

	// 4. Test shared anchor resolution (inject one)
	invA.Axioms[0].ID = "SHARED-1"
	invA.Axioms[0].Tier = 1
	invB.Axioms[0].ID = "SHARED-1"
	invB.Axioms[0].Tier = 0 // Tier 0 is "more trustworthy" in enum? 
	// Wait, Tier 0 (Axiom) < Tier 1 (Proof) numerically. Min(0, 1) = 0.
	
	merged2, report2 := MergeProgrammes(invA, invB)
	if len(report2.SharedAnchorIDs) != 1 {
		t.Errorf("expected 1 shared anchor after injection, got %d", len(report2.SharedAnchorIDs))
	}
	
	// Find the shared anchor in merged2
	found := false
	for _, a := range merged2.Axioms {
		if a.ID == "SHARED-1" {
			found = true
			if a.Tier != 0 {
				t.Errorf("expected tier 0 for SHARED-1, got %d", a.Tier)
			}
		}
	}
	if !found {
		t.Errorf("SHARED-1 not found in merged axioms")
	}
}
