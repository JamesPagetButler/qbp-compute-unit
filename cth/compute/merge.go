package compute

import (
	"fmt"

	"github.com/helpful-engineering/cth/model"
)

// MergeReport summarizes the results of a programme merge.
type MergeReport struct {
	SharedAnchorIDs    []string
	BridgeEdgesCreated int
	TheoreticalDeficit float64
	EngineeringDeficit float64
	ZeroTheoretical    bool
	Lossless           bool
}

// MergeProgrammes combines two CTH inventories according to §5 merge rules.
func MergeProgrammes(a, b model.Inventory) (model.Inventory, MergeReport) {
	merged := model.Inventory{
		Programme: fmt.Sprintf("%s + %s", a.Programme, b.Programme),
		Version:   "merged",
	}
	report := MergeReport{Lossless: true}

	// 1. Create maps for easier lookup and shared anchor detection
	anchorsA := make(map[string]model.Anchor)
	for _, group := range [][]model.Anchor{a.Axioms, a.DerivedPrinciples, a.Anchors, a.Inputs} {
		for _, anc := range group {
			anchorsA[anc.ID] = anc
		}
	}

	anchorsB := make(map[string]model.Anchor)
	for _, group := range [][]model.Anchor{b.Axioms, b.DerivedPrinciples, b.Anchors, b.Inputs} {
		for _, anc := range group {
			anchorsB[anc.ID] = anc
		}
	}

	// 2. Resolve all anchors
	allIDs := make(map[string]bool)
	for id := range anchorsA {
		allIDs[id] = true
	}
	for id := range anchorsB {
		allIDs[id] = true
	}

	resolvedAnchors := make(map[string]model.Anchor)
	for id := range allIDs {
		ancA, inA := anchorsA[id]
		ancB, inB := anchorsB[id]

		if inA && inB {
			report.SharedAnchorIDs = append(report.SharedAnchorIDs, id)
			// Merge Rules (§5.2):
			// Tier: min(A, B)
			tier := ancA.Tier
			if ancB.Tier < tier {
				tier = ancB.Tier
			}

			// Status: consensus
			status := model.Coherent
			if ancA.Status != ancB.Status {
				status = model.Incoherent // Conflict signals failed confluence
			} else {
				status = ancA.Status
			}

			// Residual Entropy: min(A, B)
			entropy := ancA.ResidualEntropyBits
			if ancB.ResidualEntropyBits < entropy {
				entropy = ancB.ResidualEntropyBits
			}

			mergedAnc := ancA
			mergedAnc.Tier = tier
			mergedAnc.Status = status
			mergedAnc.ResidualEntropyBits = entropy
			resolvedAnchors[id] = mergedAnc

			// Theorem 2: Lossless if both Tier 1
			if ancA.Tier != model.Proof || ancB.Tier != model.Proof {
				report.Lossless = false
			}
		} else if inA {
			resolvedAnchors[id] = ancA
		} else {
			resolvedAnchors[id] = ancB
		}
	}

	// 3. Populate merged inventory groups
	for _, anc := range resolvedAnchors {
		switch anc.Tier {
		case model.Axiom:
			merged.Axioms = append(merged.Axioms, anc)
		case model.Proof:
			merged.DerivedPrinciples = append(merged.DerivedPrinciples, anc)
		case model.Measurement:
			merged.Anchors = append(merged.Anchors, anc)
		case model.Prediction:
			if anc.Provenance == model.Input {
				merged.Inputs = append(merged.Inputs, anc)
				// Deficit Classification (Definition 19)
				if anc.Domain == "math" || anc.Domain == "theory" {
					report.TheoreticalDeficit += anc.ResidualEntropyBits
				} else {
					report.EngineeringDeficit += anc.ResidualEntropyBits
				}
			} else {
				merged.Anchors = append(merged.Anchors, anc)
			}
		}
	}

	// 4. Combine chains and confluence points
	merged.Chains = append(a.Chains, b.Chains...)
	merged.ConfluencePoints = append(a.ConfluencePoints, b.ConfluencePoints...)

	if report.TheoreticalDeficit == 0 {
		report.ZeroTheoretical = true
	}

	return merged, report
}
