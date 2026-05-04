package model

// Chain represents a derivation from premise set to conclusion (Definition 1).
type Chain struct {
	ID               string           `json:"id"`
	SourceIDs        []string         `json:"source_ids"`      // Premise anchors
	TargetID         string           `json:"target_id"`       // Conclusion anchor
	Steps            int              `json:"steps"`           // Chain length
	WeakestLinkID    string           `json:"weakest_link_id"` // ID of lowest-fidelity edge
	Fidelity         float64          `json:"fidelity"`        // μ(C) = product of edge fidelities
	Status           Status           `json:"status"`
	DomainBoundaries []DomainBoundary `json:"domain_boundaries"`
}

// DomainBoundary represents a point where a claim crosses verification methods (Definition 6a).
type DomainBoundary struct {
	FromDomain string  `json:"from_domain"`
	ToDomain   string  `json:"to_domain"`
	AtAnchorID string  `json:"at_anchor_id"`
	Fidelity   float64 `json:"fidelity"`
	Hypothesis string  `json:"hypothesis"`
}
