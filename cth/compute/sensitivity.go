package compute

import "github.com/helpful-engineering/cth/model"

// SensitivityBracket computes ρ_net at three axiom-entropy scalings (Definition 15, §4.6).
//
// Returns (halfH, baseH, doubleH): ρ_net when every axiom's residual entropy is ½×, 1×,
// and 2× its stated value respectively.  The inputEntropy map is held constant across all
// three evaluations so the only variable is the axiom prior.
func SensitivityBracket(inv model.Inventory, inputEntropy map[string]float64) (halfH, baseH, doubleH float64) {
	baseH, _ = NetCompression(inv, inputEntropy)

	invHalf := inv
	invHalf.Axioms = scaleAxiomEntropy(inv.Axioms, 0.5)
	halfH, _ = NetCompression(invHalf, inputEntropy)

	invDouble := inv
	invDouble.Axioms = scaleAxiomEntropy(inv.Axioms, 2.0)
	doubleH, _ = NetCompression(invDouble, inputEntropy)

	return
}

// SensitivityRatio returns doubleH/halfH (Definition 15).
// Values > 0.5 indicate a robust programme whose ρ_net is not overly sensitive to the
// choice of axiom entropy.  Values ≤ 0.5 signal fragility.
func SensitivityRatio(halfH, doubleH float64) float64 {
	if halfH == 0 {
		return 0
	}
	return doubleH / halfH
}

// scaleAxiomEntropy returns a copy of the axiom slice with ResidualEntropyBits multiplied
// by factor.  The original slice is not modified.
func scaleAxiomEntropy(axioms []model.Anchor, factor float64) []model.Anchor {
	scaled := make([]model.Anchor, len(axioms))
	for i, a := range axioms {
		scaled[i] = a
		scaled[i].ResidualEntropyBits *= factor
	}
	return scaled
}
