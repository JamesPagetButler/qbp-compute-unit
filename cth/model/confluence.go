package model

// ChainProvenance classifies a chain's relationship to the current CTH.
type ChainProvenance string

const (
	Internal       ChainProvenance = "internal"
	External       ChainProvenance = "external"
	CrossProgramme ChainProvenance = "cross_programme"
)

// ChainRef represents a reference to a derivation (Definition 4a).
type ChainRef struct {
	ChainID    string          `json:"chain_id"` // null for external chains
	Programme  string          `json:"programme"`
	Provenance ChainProvenance `json:"provenance"`
	Fidelity   float64         `json:"fidelity"`
	Summary    string          `json:"summary"`
}

// ConfluencePoint represents an N-ary error-detecting code (Definition 5).
type ConfluencePoint struct {
	ID             string     `json:"id"`
	AnchorID       string     `json:"anchor_id"` // Shared target
	Paths          []ChainRef `json:"paths"`     // N >= 2 chain references
	MutualInfoBits float64    `json:"mutual_info_bits"`
	Status         Status     `json:"status"`
}
