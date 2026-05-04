package compute

import "github.com/helpful-engineering/cth/model"

// allAnchors returns every anchor from all groups in the inventory.
func allAnchors(inv model.Inventory) []model.Anchor {
	var all []model.Anchor
	all = append(all, inv.Axioms...)
	all = append(all, inv.DerivedPrinciples...)
	all = append(all, inv.Anchors...)
	all = append(all, inv.Inputs...)
	return all
}

// findAnchor looks up an anchor by ID across all inventory groups.
// Returns nil if not found.
func findAnchor(inv model.Inventory, id string) *model.Anchor {
	for i := range inv.Axioms {
		if inv.Axioms[i].ID == id {
			return &inv.Axioms[i]
		}
	}
	for i := range inv.DerivedPrinciples {
		if inv.DerivedPrinciples[i].ID == id {
			return &inv.DerivedPrinciples[i]
		}
	}
	for i := range inv.Anchors {
		if inv.Anchors[i].ID == id {
			return &inv.Anchors[i]
		}
	}
	for i := range inv.Inputs {
		if inv.Inputs[i].ID == id {
			return &inv.Inputs[i]
		}
	}
	return nil
}

// anchorIndex builds a map[ID]Anchor from all inventory groups for O(1) lookup.
func anchorIndex(inv model.Inventory) map[string]model.Anchor {
	all := allAnchors(inv)
	m := make(map[string]model.Anchor, len(all))
	for _, a := range all {
		m[a.ID] = a
	}
	return m
}
