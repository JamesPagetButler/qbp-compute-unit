package compute

import (
	"github.com/helpful-engineering/cth/model"
)

// StepFidelity returns the fidelity μ for a given step category (Definition 9, §4.4).
func StepFidelity(stepType string) float64 {
	switch stepType {
	case "lean", "formal_proof":
		return 1.000
	case "established_math", "hurwitz":
		return 1.000
	case "standard_physics":
		return 0.999
	case "verified_computation":
		return 0.999
	case "domain_boundary":
		return 0.950 // Default, can be overridden by specific edge
	case "semi_empirical":
		return 0.950
	case "conjecture":
		return 0.500
	default:
		return 0.900 // Default for unknown unproven steps
	}
}

// ChainFidelity computes the total fidelity μ(C) as a multiplicative product (Definition 9).
// If the chain has an explicit fidelity > 0, it uses that; otherwise it returns 1.0 (Crawl phase default).
func ChainFidelity(c model.Chain) float64 {
	if c.Fidelity > 0 {
		return c.Fidelity
	}
	return 1.0
}

// ClassifyFidelityRegime categorizes the fidelity into hydrodynamic regimes (§3.3, §4.4).
func ClassifyFidelityRegime(mu float64) string {
	if mu >= 0.999 {
		return "laminar"
	}
	if mu >= 0.900 {
		return "low_sediment"
	}
	if mu >= 0.700 {
		return "moderate"
	}
	return "heavy"
}
