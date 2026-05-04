package report

import (
	"fmt"
	"strings"

	"github.com/helpful-engineering/cth/model"
)

// Dashboard returns a compact text summary of the programme's epistemic health.
// Output is suitable for terminal display or embedding in a larger report.
func Dashboard(inv model.Inventory, a FullAnalysis) string {
	var b strings.Builder

	allAnchors := countAllAnchors(inv)
	coherent := countByStatus(inv, model.Coherent)

	fmt.Fprintf(&b, "┌─ %s (v%s) ─────────────────────────────────────────\n", inv.Programme, inv.Version)
	fmt.Fprintf(&b, "│  Anchors:        %d total  (%d coherent, %.0f%%)\n",
		allAnchors, coherent, pct(coherent, allAnchors))
	fmt.Fprintf(&b, "│  ρ_net:          %.3f", a.Rho)
	if a.SensRatio > 0 {
		robust := "fragile"
		if a.SensRatio > 0.5 {
			robust = "robust"
		}
		fmt.Fprintf(&b, "  [½H:%.3f  2H:%.3f  ratio:%.2f  %s]",
			a.Sensitivity[0], a.Sensitivity[2], a.SensRatio, robust)
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "│  Deficit:        %.2f bits (input)  Axiom H: %.2f bits\n",
		a.RhoDetail.InformationDeficit, a.RhoDetail.AxiomEntropyBits)

	if len(a.BridgeNodes) > 0 {
		top := a.BridgeNodes[0]
		fmt.Fprintf(&b, "│  Top bridge:     %s  (%d domains: %s)\n",
			top.ID, top.DomainCount, strings.Join(top.Domains, ", "))
	}

	if a.Sediment.SharpPartition {
		fmt.Fprintf(&b, "│  Sediment:       SHARP PARTITION  dirty=%v  clean=%v\n",
			a.Sediment.DirtyOnlyDomains, a.Sediment.CleanOnlyDomains)
	} else if len(a.Sediment.Heavy)+len(a.Sediment.Moderate) > 0 {
		fmt.Fprintf(&b, "│  Sediment:       %d heavy  %d moderate  %d low  %d laminar\n",
			len(a.Sediment.Heavy), len(a.Sediment.Moderate),
			len(a.Sediment.LowSediment), len(a.Sediment.Laminar))
	} else {
		fmt.Fprintf(&b, "│  Sediment:       clean (%d laminar  %d low)\n",
			len(a.Sediment.Laminar), len(a.Sediment.LowSediment))
	}

	if len(a.Eddies) > 0 {
		top := a.Eddies[0]
		fmt.Fprintf(&b, "│  Top eddy:       %s  proximity=%.3f  gap=%.1f\n",
			top.AnchorID, top.Proximity, top.WeightedGap)
	}

	fmt.Fprintf(&b, "└───────────────────────────────────────────────────────────\n")
	return b.String()
}

func countAllAnchors(inv model.Inventory) int {
	return len(inv.Axioms) + len(inv.DerivedPrinciples) + len(inv.Anchors) + len(inv.Inputs)
}

func countByStatus(inv model.Inventory, s model.Status) int {
	n := 0
	for _, a := range inv.Axioms {
		if a.Status == s {
			n++
		}
	}
	for _, a := range inv.DerivedPrinciples {
		if a.Status == s {
			n++
		}
	}
	for _, a := range inv.Anchors {
		if a.Status == s {
			n++
		}
	}
	for _, a := range inv.Inputs {
		if a.Status == s {
			n++
		}
	}
	return n
}

func pct(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
