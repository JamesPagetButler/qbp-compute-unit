package model

// Inventory represents the top-level container for a research programme (Definition 1).
type Inventory struct {
	Programme         string            `json:"programme"`
	Version           string            `json:"version"`
	Axioms            []Anchor          `json:"axioms"`
	DerivedPrinciples []Anchor          `json:"derived_principles"`
	Anchors           []Anchor          `json:"anchors"` // General pool
	Inputs            []Anchor          `json:"inputs"`
	Chains            []Chain           `json:"chains"`
	ConfluencePoints  []ConfluencePoint `json:"confluence_points"`
	Health            Health            `json:"health"`
}

// Health captures the programme's epistemic metrics.
type Health struct {
	NetCompressionRatio float64 `json:"net_compression_ratio"`
	GrossCompression    float64 `json:"gross_compression"`
	CompressionVelocity float64 `json:"compression_velocity"`
	InformationDeficit  float64 `json:"information_deficit"`
	CoherenceRatio      float64 `json:"coherence_ratio"`
	ConfluenceCoverage  float64 `json:"confluence_coverage"`
	ConfluenceDepth     int     `json:"confluence_depth"`
	MeanChainFidelity   float64 `json:"mean_chain_fidelity"`
}

// Validate performs a global check on the inventory's consistency.
func (inv *Inventory) Validate() error {
	for _, a := range inv.Axioms {
		if err := a.Validate(); err != nil {
			return err
		}
	}
	// Add more global validations as needed (e.g., source sets, arity)
	return nil
}
