package compute

import (
	"testing"

	"github.com/helpful-engineering/cth/model"
	"github.com/helpful-engineering/cth/store"
)

func TestAnchorConfluenceDepth(t *testing.T) {
	inv, err := store.LoadInventory("../testdata/minimal.json")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// minimal.json: CP-1 on MEAS-1 with 2 paths → arity weight = 1.
	depths := AnchorConfluenceDepth(inv)

	if depths["MEAS-1"] != 1 {
		t.Errorf("MEAS-1 depth = %d; want 1", depths["MEAS-1"])
	}
	// Anchors without confluences should have depth 0 (missing from map).
	if depths["AX-1"] != 0 {
		t.Errorf("AX-1 depth = %d; want 0", depths["AX-1"])
	}
}

func TestAnchorConfluenceDepthThreeWay(t *testing.T) {
	// A 3-way confluence should earn arity weight = 2, exceeding a 2-way (weight = 1).
	inv := model.Inventory{
		Programme: "DepthTest",
		Anchors: []model.Anchor{
			{ID: "MEAS-2way"},
			{ID: "MEAS-3way"},
		},
		ConfluencePoints: []model.ConfluencePoint{
			{
				ID:       "CP-2way",
				AnchorID: "MEAS-2way",
				Paths:    []model.ChainRef{{ChainID: "CH-A"}, {ChainID: "CH-B"}}, // 2 paths
			},
			{
				ID:       "CP-3way",
				AnchorID: "MEAS-3way",
				Paths:    []model.ChainRef{{ChainID: "CH-X"}, {ChainID: "CH-Y"}, {ChainID: "CH-Z"}}, // 3 paths
			},
		},
	}

	depths := AnchorConfluenceDepth(inv)

	if depths["MEAS-3way"] <= depths["MEAS-2way"] {
		t.Errorf("3-way anchor depth (%d) should exceed 2-way (%d)", depths["MEAS-3way"], depths["MEAS-2way"])
	}
}
