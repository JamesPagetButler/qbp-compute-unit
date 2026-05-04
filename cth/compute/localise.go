package compute

import "github.com/helpful-engineering/cth/model"

// LocalisationResult describes the segment of a derivation chain where an incoherence
// was isolated (Method 6).
type LocalisationResult struct {
	// ErrorStart is the ID of the last coherent anchor before the incoherence.
	ErrorStart string
	// ErrorEnd is the ID of the anchor where the incoherence was detected.
	ErrorEnd string
	// LastCoherentConfluence is the ID of the most recent coherent ConfluencePoint
	// encountered while walking backwards from ErrorEnd.  Empty if none found.
	LastCoherentConfluence string
	// WeakestLinkID is the ID of the lowest-fidelity chain in the error segment.
	WeakestLinkID string
	// Found reports whether a non-coherent anchor was actually located.
	Found bool
}

// LocaliseIncoherence walks backwards through the derivation graph from anchorID,
// using ConfluencePoints as checkpoints, and returns the tightest error segment
// that can be isolated (Method 6).
//
// The walk terminates when it reaches an axiom, a coherent confluence, or a chain
// whose source anchors are all coherent.  The weakest chain in the isolated segment
// is identified from chain fidelity scores.
func LocaliseIncoherence(anchorID string, inv model.Inventory) LocalisationResult {
	result := LocalisationResult{ErrorEnd: anchorID}

	start := findAnchor(inv, anchorID)
	if start == nil {
		return result
	}
	if start.Status == model.Coherent {
		// Nothing to localise.
		return result
	}
	result.Found = true

	// Build lookup structures.
	confluenceByAnchor := make(map[string]model.ConfluencePoint)
	for _, cp := range inv.ConfluencePoints {
		confluenceByAnchor[cp.AnchorID] = cp
	}

	// Map target → incoming chains.
	incoming := make(map[string][]model.Chain)
	for _, c := range inv.Chains {
		incoming[c.TargetID] = append(incoming[c.TargetID], c)
	}

	// Walk backwards until we find a coherent confluence or run out of chain.
	currentID := anchorID
	var weakestChainID string
	weakestFidelity := 1.0

	visited := make(map[string]bool)
	for {
		if visited[currentID] {
			break
		}
		visited[currentID] = true

		// Check for a coherent confluence at this anchor (checkpoint).
		if cp, ok := confluenceByAnchor[currentID]; ok && cp.Status == model.Coherent {
			result.LastCoherentConfluence = cp.ID
			result.ErrorStart = currentID
			break
		}

		chains := incoming[currentID]
		if len(chains) == 0 {
			// Reached an axiom or root — set start here.
			result.ErrorStart = currentID
			break
		}

		// Follow the lowest-fidelity incoming chain (most suspicious step).
		worst := chains[0]
		for _, c := range chains[1:] {
			if ChainFidelity(c) < ChainFidelity(worst) {
				worst = c
			}
		}

		f := ChainFidelity(worst)
		if f < weakestFidelity {
			weakestFidelity = f
			weakestChainID = worst.ID
		}

		// Step back to source(s).  If multiple sources, follow the incoherent one.
		nextID := ""
		for _, srcID := range worst.SourceIDs {
			a := findAnchor(inv, srcID)
			if a != nil && a.Status != model.Coherent {
				nextID = srcID
				break
			}
		}
		if nextID == "" && len(worst.SourceIDs) > 0 {
			nextID = worst.SourceIDs[0]
		}
		if nextID == "" {
			result.ErrorStart = currentID
			break
		}
		currentID = nextID
	}

	result.WeakestLinkID = weakestChainID
	if result.ErrorStart == "" {
		result.ErrorStart = anchorID
	}
	return result
}
