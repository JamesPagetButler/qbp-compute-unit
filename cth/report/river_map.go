package report

import (
	"fmt"
	"strings"

	"github.com/helpful-engineering/cth/model"
)

// RiverMap returns a narrative description of the programme's epistemic flow,
// using the hydrological river metaphor from CTH theory §3.
//
// The description highlights the main channel (highest-fidelity chains),
// sediment zones, eddy basins (open problems), and confluence checkpoints.
func RiverMap(inv model.Inventory) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## River Map: %s\n\n", inv.Programme)

	// Source (axioms).
	if len(inv.Axioms) > 0 {
		names := make([]string, 0, len(inv.Axioms))
		for _, a := range inv.Axioms {
			names = append(names, fmt.Sprintf("%s (%.1f bits)", a.Name, a.ResidualEntropyBits))
		}
		fmt.Fprintf(&b, "**Source springs:** %s\n\n", strings.Join(names, "; "))
	}

	// Main channel (laminar chains).
	laminar := make([]string, 0)
	for _, c := range inv.Chains {
		if c.Fidelity >= 0.999 {
			laminar = append(laminar, c.ID)
		}
	}
	if len(laminar) > 0 {
		fmt.Fprintf(&b, "**Main channel** (%d laminar chains, μ ≥ 0.999): %s\n\n",
			len(laminar), strings.Join(laminar, ", "))
	}

	// Domain crossings (turbulence points).
	var crossings []string
	for _, c := range inv.Chains {
		for _, db := range c.DomainBoundaries {
			crossings = append(crossings, fmt.Sprintf("%s→%s at %s (μ=%.3f, hypothesis: %s)",
				db.FromDomain, db.ToDomain, db.AtAnchorID, db.Fidelity, db.Hypothesis))
		}
	}
	if len(crossings) > 0 {
		fmt.Fprintln(&b, "**Turbulence points** (domain crossings):")
		for _, x := range crossings {
			fmt.Fprintf(&b, "- %s\n", x)
		}
		fmt.Fprintln(&b)
	}

	// Confluence checkpoints.
	if len(inv.ConfluencePoints) > 0 {
		fmt.Fprintf(&b, "**Confluence checkpoints** (%d):\n", len(inv.ConfluencePoints))
		for _, cp := range inv.ConfluencePoints {
			fmt.Fprintf(&b, "- %s → %s  [%d paths, status: %s]\n",
				cp.ID, cp.AnchorID, len(cp.Paths), cp.Status)
		}
		fmt.Fprintln(&b)
	}

	// Eddy basins (inputs that are irreducible open problems).
	if len(inv.Inputs) > 0 {
		fmt.Fprintf(&b, "**Eddy basins** (%d irreducible inputs):\n", len(inv.Inputs))
		for _, a := range inv.Inputs {
			fmt.Fprintf(&b, "- %s: %s  (%.2f bits)\n", a.ID, a.Description, a.ResidualEntropyBits)
		}
		fmt.Fprintln(&b)
	}

	// Delta (predictions / untested anchors).
	var preds []model.Anchor
	for _, a := range inv.Anchors {
		if a.Tier == model.Prediction {
			preds = append(preds, a)
		}
	}
	if len(preds) > 0 {
		fmt.Fprintf(&b, "**Delta** (%d untested predictions):\n", len(preds))
		for _, a := range preds {
			fmt.Fprintf(&b, "- %s: %s\n", a.ID, a.Description)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}
