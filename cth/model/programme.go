package model

// ProgrammeMeta holds metadata about a research programme beyond the inventory fields.
type ProgrammeMeta struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// MergeInput describes a shared input anchor used across programme boundaries.
// Produced by MergeProgrammes when two programmes share an irreducible input.
type MergeInput struct {
	AnchorID       string  `json:"anchor_id"`
	Programme      string  `json:"programme"`
	SharedWith     string  `json:"shared_with"`
	BridgeFidelity float64 `json:"bridge_fidelity"`
}
