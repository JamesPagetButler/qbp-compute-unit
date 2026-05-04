package model

import (
	"fmt"
	"time"
)

// Tier represents the epistemological character of an anchor (Definition 2).
type Tier int

const (
	Axiom       Tier = 0 // Foundational assumption
	Proof       Tier = 1 // Machine-verified formal proof
	Measurement Tier = 2 // Empirical result
	Prediction  Tier = 3 // Untested derived claim
)

// Status represents the epistemic health of an anchor.
type Status string

const (
	Coherent   Status = "coherent"
	Marginal   Status = "marginal"
	Incoherent Status = "incoherent"
	Untested   Status = "untested"
)

// Provenance classifies the source of the anchor.
type Provenance string

const (
	Theoretical Provenance = "T" // Theoretical derivation
	Experimental Provenance = "E" // Experimental measurement
	Input        Provenance = "I" // Irreducible input
)

// Anchor represents a claim in the research programme (Definition 1).
type Anchor struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Tier                 Tier       `json:"tier"`
	Derivable            bool       `json:"derivable"` // Must be false for Tier 0
	Status               Status     `json:"status"`
	Provenance           Provenance `json:"provenance"`
	Domain               string     `json:"domain"`
	ResidualEntropyBits  float64    `json:"residual_entropy_bits"`
	ConfirmatoryInfoBits float64    `json:"confirmatory_info_bits"`
	Description          string     `json:"description"`
	PredictionChain      []string   `json:"prediction_chain"`
	LastTestedAt         *time.Time `json:"last_tested_at,omitempty"`
}

// Validate checks the structural invariants of the anchor.
func (a *Anchor) Validate() error {
	// Definition 2a: Tier 0 Derivability Prohibition
	if a.Tier == Axiom && a.Derivable {
		return fmt.Errorf("anchor %s: Tier 0 (Axiom) cannot be derivable", a.ID)
	}
	return nil
}
