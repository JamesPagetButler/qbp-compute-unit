package compute

import (
	"sort"

	"github.com/helpful-engineering/cth/model"
)

// AbInitioResult reports the preferred ab initio derivation path for a multi-path anchor (R7).
type AbInitioResult struct {
	AnchorID    string
	BestChainID string
	Score       float64 // fidelity / (1 + input_count)
	Fidelity    float64
	InputCount  int
}

// AbInitioScore ranks anchors that have multiple incoming chains by their best
// ab initio preference score: score = μ(C) / (1 + |inputs(C)|).
//
// A chain with high fidelity and few irreducible inputs is preferred.  When
// fidelities are comparable, the path with fewer inputs wins (lower deficit).
// Results are sorted descending by score.
func AbInitioScore(inv model.Inventory) []AbInitioResult {
	// Build input-ID lookup set.
	inputIDs := make(map[string]bool, len(inv.Inputs))
	for _, a := range inv.Inputs {
		inputIDs[a.ID] = true
	}

	// Group incoming chains by target anchor.
	incoming := make(map[string][]model.Chain)
	for _, c := range inv.Chains {
		incoming[c.TargetID] = append(incoming[c.TargetID], c)
	}

	var results []AbInitioResult
	for anchorID, chains := range incoming {
		if len(chains) < 2 {
			continue // Only score anchors with multiple derivation paths.
		}

		var best AbInitioResult
		best.AnchorID = anchorID
		best.Score = -1

		for _, c := range chains {
			fidelity := ChainFidelity(c)
			inputCount := countChainInputs(c, inputIDs)
			score := fidelity / float64(1+inputCount)

			if score > best.Score {
				best.BestChainID = c.ID
				best.Score = score
				best.Fidelity = fidelity
				best.InputCount = inputCount
			}
		}

		if best.Score >= 0 {
			results = append(results, best)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// countChainInputs counts how many of a chain's direct source IDs are irreducible inputs.
func countChainInputs(c model.Chain, inputIDs map[string]bool) int {
	count := 0
	for _, srcID := range c.SourceIDs {
		if inputIDs[srcID] {
			count++
		}
	}
	return count
}
