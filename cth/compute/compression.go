package compute

import (
	"github.com/helpful-engineering/cth/model"
)

// VersionSnapshot captures the epistemic state at a point in time for velocity tracking (Definition 14).
type VersionSnapshot struct {
	Rho     float64 // Net compression ratio ρ_net
	NAnchor int     // Number of confirmed anchors
}

// NetCompressionDetail provides a per-component breakdown of the ρ_net calculation.
type NetCompressionDetail struct {
	GrossConfirmedBits float64
	InputCostBits      float64
	NetConfirmedBits   float64
	AxiomEntropyBits   float64
	InformationDeficit float64
	TotalDenominator   float64
}

// BuildInputEntropy builds the input-entropy map from an inventory's input anchors.
// The caller may scale values for sensitivity analysis before passing to NetCompression.
func BuildInputEntropy(inv model.Inventory) map[string]float64 {
	m := make(map[string]float64, len(inv.Inputs))
	for _, a := range inv.Inputs {
		m[a.ID] = ResidualEntropy(a)
	}
	return m
}

// GrossCompression computes ρ_gross (Definition 13).
func GrossCompression(inv model.Inventory) float64 {
	confirmed := 0.0
	for _, a := range inv.Anchors {
		confirmed += ConfirmatoryInfo(a)
	}
	for _, a := range inv.DerivedPrinciples {
		confirmed += ConfirmatoryInfo(a)
	}

	axioms := 0.0
	for _, a := range inv.Axioms {
		axioms += AxiomEntropy(a)
	}

	deficit := 0.0
	for _, a := range inv.Inputs {
		deficit += ResidualEntropy(a)
	}

	denom := axioms + deficit
	if denom == 0 {
		return 0
	}
	return confirmed / denom
}

// NetCompression computes ρ_net with input cost allocation (Definition 13).
// inputEntropy maps each input anchor ID to its entropy cost in bits; if an ID is
// absent the anchor's ResidualEntropyBits is used as a fallback.
// Pass BuildInputEntropy(inv) for the default case; scale the map for sensitivity analysis.
func NetCompression(inv model.Inventory, inputEntropy map[string]float64) (float64, NetCompressionDetail) {
	detail := NetCompressionDetail{}

	// Confirmed bits across all non-axiom, non-input groups.
	for _, a := range inv.Anchors {
		detail.GrossConfirmedBits += ConfirmatoryInfo(a)
	}
	for _, a := range inv.DerivedPrinciples {
		detail.GrossConfirmedBits += ConfirmatoryInfo(a)
	}

	// Axiom entropy.
	for _, a := range inv.Axioms {
		detail.AxiomEntropyBits += AxiomEntropy(a)
	}

	// Input deficit — use caller-supplied map so sensitivity analysis can scale values.
	for _, a := range inv.Inputs {
		cost, ok := inputEntropy[a.ID]
		if !ok {
			cost = ResidualEntropy(a)
		}
		detail.InformationDeficit += cost
	}

	detail.TotalDenominator = detail.AxiomEntropyBits + detail.InformationDeficit
	detail.InputCostBits = detail.InformationDeficit
	detail.NetConfirmedBits = detail.GrossConfirmedBits - detail.InputCostBits

	if detail.TotalDenominator == 0 {
		return 0, detail
	}
	return detail.NetConfirmedBits / detail.TotalDenominator, detail
}

// CompressionVelocity computes Δρ / Δn between two programme snapshots (Definition 14).
func CompressionVelocity(prev, curr VersionSnapshot) float64 {
	dn := curr.NAnchor - prev.NAnchor
	if dn <= 0 {
		return 0
	}
	return (curr.Rho - prev.Rho) / float64(dn)
}
