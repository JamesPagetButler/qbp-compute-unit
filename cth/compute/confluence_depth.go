package compute

import "github.com/helpful-engineering/cth/model"

// AnchorConfluenceDepth computes the arity-weighted confluence depth for every anchor (§4.7).
//
// For each confluence point, the shared target anchor earns (len(paths) − 1) depth units —
// so a 3-way confluence contributes 2 units, a 2-way contributes 1 unit.  An anchor
// downstream of multiple confluences accumulates depth from all of them.
func AnchorConfluenceDepth(inv model.Inventory) map[string]int {
	depths := make(map[string]int)
	for _, cp := range inv.ConfluencePoints {
		arity := len(cp.Paths) - 1
		if arity < 1 {
			arity = 1
		}
		depths[cp.AnchorID] += arity
	}
	return depths
}

// ChainConfluenceDepth computes the arity-weighted confluence depth for every chain (§4.7).
//
// A chain's depth is the sum of confluence depths accumulated by all anchors in its source
// set.  This measures how much independent corroboration feeds into each derivation step.
func ChainConfluenceDepth(inv model.Inventory) map[string]int {
	anchorDepths := AnchorConfluenceDepth(inv)

	chainDepths := make(map[string]int, len(inv.Chains))
	for _, c := range inv.Chains {
		total := 0
		for _, srcID := range c.SourceIDs {
			total += anchorDepths[srcID]
		}
		chainDepths[c.ID] = total
	}
	return chainDepths
}
