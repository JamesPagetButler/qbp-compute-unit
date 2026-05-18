package compute

import (
	"sort"

	"github.com/helpful-engineering/cth/model"
)

// SedimentReport partitions chains by fidelity regime and characterises any
// domain-aligned sediment (Method 3).
type SedimentReport struct {
	// Chain IDs by regime.
	Laminar     []string // μ ≥ 0.999
	LowSediment []string // μ ∈ [0.90, 0.999)
	Moderate    []string // μ ∈ [0.70, 0.90)
	Heavy       []string // μ < 0.70

	// Domains appearing only in the heavy/moderate partition.
	DirtyOnlyDomains []string
	// Domains appearing only in the laminar/low partition.
	CleanOnlyDomains []string

	// SharpPartition is true when both DirtyOnlyDomains and CleanOnlyDomains are
	// non-empty — meaning sediment is domain-correlated, not randomly distributed.
	SharpPartition bool
}

// DetectSedimentPartitions partitions the inventory's chains by fidelity regime and
// tests for domain correlation (Method 3).
func DetectSedimentPartitions(inv model.Inventory) SedimentReport {
	idx := anchorIndex(inv)

	cleanDomains := make(map[string]bool) // domains in laminar + low_sediment
	dirtyDomains := make(map[string]bool) // domains in moderate + heavy

	var report SedimentReport

	for _, c := range inv.Chains {
		regime := ClassifyFidelityRegime(c.Fidelity)

		// Collect domains touched by this chain.
		chainDomains := chainDomainSet(c, idx)

		switch regime {
		case "laminar":
			report.Laminar = append(report.Laminar, c.ID)
			for d := range chainDomains {
				cleanDomains[d] = true
			}
		case "low_sediment":
			report.LowSediment = append(report.LowSediment, c.ID)
			for d := range chainDomains {
				cleanDomains[d] = true
			}
		case "moderate":
			report.Moderate = append(report.Moderate, c.ID)
			for d := range chainDomains {
				dirtyDomains[d] = true
			}
		case "heavy":
			report.Heavy = append(report.Heavy, c.ID)
			for d := range chainDomains {
				dirtyDomains[d] = true
			}
		}
	}

	// Dirty-only: in dirty set but not clean set.
	for d := range dirtyDomains {
		if !cleanDomains[d] {
			report.DirtyOnlyDomains = append(report.DirtyOnlyDomains, d)
		}
	}
	// Clean-only: in clean set but not dirty set.
	for d := range cleanDomains {
		if !dirtyDomains[d] {
			report.CleanOnlyDomains = append(report.CleanOnlyDomains, d)
		}
	}

	sort.Strings(report.DirtyOnlyDomains)
	sort.Strings(report.CleanOnlyDomains)

	report.SharpPartition = len(report.DirtyOnlyDomains) > 0 && len(report.CleanOnlyDomains) > 0

	return report
}

// chainDomainSet returns the set of domains touched by a chain (source and target anchors).
func chainDomainSet(c model.Chain, idx map[string]model.Anchor) map[string]bool {
	ds := make(map[string]bool)
	if tgt, ok := idx[c.TargetID]; ok {
		ds[tgt.Domain] = true
	}
	for _, srcID := range c.SourceIDs {
		if src, ok := idx[srcID]; ok {
			ds[src.Domain] = true
		}
	}
	for _, db := range c.DomainBoundaries {
		ds[db.FromDomain] = true
		ds[db.ToDomain] = true
	}
	return ds
}
