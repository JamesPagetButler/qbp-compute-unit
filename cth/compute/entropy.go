package compute

import (
	"math"

	"github.com/helpful-engineering/cth/model"
)

// ResidualEntropy computes η(v) for an anchor (Definition 7).
func ResidualEntropy(a model.Anchor) float64 {
	switch a.Tier {
	case model.Axiom:
		// Assigned entropy for Tier 0
		return a.ResidualEntropyBits
	case model.Proof:
		// Proof is a lossless channel (conditional entropy = 0)
		return 0
	case model.Measurement:
		// Measurement entropy depends on discrepancy (delta)
		// Note: The inventory stores the already-computed bits in Crawl phase.
		// In Walk phase, this would be computed from raw delta.
		return a.ResidualEntropyBits
	case model.Prediction:
		// Prediction entropy is inherited from the weakest upstream link
		return a.ResidualEntropyBits
	default:
		return a.ResidualEntropyBits
	}
}

// ConfirmatoryInfo computes ι(v) for an anchor (Definition 7a).
func ConfirmatoryInfo(a model.Anchor) float64 {
	switch a.Tier {
	case model.Axiom, model.Proof:
		return 0
	case model.Measurement:
		// ι(v) = log2(1/|delta|) if delta > 0, else 1.0 (structural)
		return a.ConfirmatoryInfoBits
	case model.Prediction:
		return 0
	default:
		return 0
	}
}

// InputEntropy computes the bit cost of a dimensionless constant known 
// to n significant figures (Definition 7).
func InputEntropy(significantFigures int) float64 {
	if significantFigures <= 0 {
		return 0
	}
	return 3.32 * float64(significantFigures)
}

// AxiomEntropy is a semantic alias for clarity in merge/compression workflows.
func AxiomEntropy(a model.Anchor) float64 {
	if a.Tier != model.Axiom {
		return 0
	}
	return a.ResidualEntropyBits
}

// PredictionEntropyInheritance calculates the inherited entropy for a Tier 3 
// anchor given its weakest upstream link and edge fidelity (Definition 7).
func PredictionEntropyInheritance(weakestLinkEntropy float64, edgeFidelity float64) float64 {
	if edgeFidelity <= 0 {
		return math.Inf(1)
	}
	if edgeFidelity >= 1.0 {
		return weakestLinkEntropy
	}
	
	// entropy = η(weakest) + (1-μ) * log2(1/μ)
	leakage := (1.0 - edgeFidelity) * math.Log2(1.0/edgeFidelity)
	return weakestLinkEntropy + leakage
}
