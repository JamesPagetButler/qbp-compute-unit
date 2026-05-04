package compute

import (
	"sort"

	"github.com/helpful-engineering/cth/model"
)

// BridgeNode represents an anchor that bridges two or more distinct domains (Method 5).
type BridgeNode struct {
	ID          string
	Domains     []string // sorted, deduplicated domain names
	DomainCount int
}

// BridgeCentrality identifies anchors that span the largest number of distinct domains.
//
// Domain membership is seeded from each anchor's own Domain field, then expanded via
// chains: a source anchor inherits the target's domain, and vice versa.  Explicit
// DomainBoundary entries on chains also expand the boundary anchor's domain set.
//
// If excludeAxioms is true, Tier 0 anchors are omitted from the result.
// Results are sorted descending by DomainCount; ties broken alphabetically by ID.
// Only anchors bridging ≥ 2 domains are returned.
func BridgeCentrality(inv model.Inventory, excludeAxioms bool) []BridgeNode {
	idx := anchorIndex(inv)

	// Seed each anchor's domain set with its own domain.
	domains := make(map[string]map[string]bool, len(idx))
	for id, a := range idx {
		if excludeAxioms && a.Tier == model.Axiom {
			continue
		}
		domains[id] = map[string]bool{a.Domain: true}
	}

	// Expand via chains.
	for _, c := range inv.Chains {
		tgt, hasTgt := idx[c.TargetID]
		if !hasTgt {
			continue
		}
		for _, srcID := range c.SourceIDs {
			src, hasSrc := idx[srcID]
			if !hasSrc {
				continue
			}
			// Source participates in target's domain.
			if ds, ok := domains[srcID]; ok {
				ds[tgt.Domain] = true
			}
			// Target participates in source's domain.
			if ds, ok := domains[c.TargetID]; ok {
				ds[src.Domain] = true
			}
		}
		// Explicit domain boundary crossings.
		for _, db := range c.DomainBoundaries {
			if ds, ok := domains[db.AtAnchorID]; ok {
				ds[db.FromDomain] = true
				ds[db.ToDomain] = true
			}
		}
	}

	var nodes []BridgeNode
	for id, ds := range domains {
		if len(ds) < 2 {
			continue
		}
		domList := make([]string, 0, len(ds))
		for d := range ds {
			domList = append(domList, d)
		}
		sort.Strings(domList)
		nodes = append(nodes, BridgeNode{
			ID:          id,
			Domains:     domList,
			DomainCount: len(ds),
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].DomainCount != nodes[j].DomainCount {
			return nodes[i].DomainCount > nodes[j].DomainCount
		}
		return nodes[i].ID < nodes[j].ID
	})

	return nodes
}
