package compute

import (
	"testing"

	"github.com/helpful-engineering/cth/store"
)

func TestBridgeCentrality(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// minimal.json: AX-1(math), PR-1(lean), MEAS-1(lab), IN-1(const).
	// CH-1: AX-1 → PR-1 — cross math/lean.
	// CH-2: PR-1 → MEAS-1 — cross lean/lab.
	// Expected bridge: PR-1 spans lean+math+lab = 3 domains.
	nodes := BridgeCentrality(inv, false)

	if len(nodes) == 0 {
		t.Fatal("expected at least one bridge node")
	}

	top := nodes[0]
	if top.ID != "PR-1" {
		t.Errorf("expected top bridge PR-1, got %s", top.ID)
	}
	if top.DomainCount < 2 {
		t.Errorf("expected PR-1 to bridge ≥ 2 domains, got %d", top.DomainCount)
	}
}

func TestBridgeCentralityExcludeAxioms(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	all := BridgeCentrality(inv, false)
	noAxioms := BridgeCentrality(inv, true)

	// Excluding axioms should not return AX-1.
	for _, n := range noAxioms {
		if n.ID == "AX-1" {
			t.Errorf("AX-1 should be excluded when excludeAxioms=true")
		}
	}

	// There should be no more nodes with axioms excluded.
	if len(noAxioms) > len(all) {
		t.Errorf("excluding axioms produced more nodes: %d > %d", len(noAxioms), len(all))
	}
}
