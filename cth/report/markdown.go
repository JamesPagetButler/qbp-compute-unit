package report

import (
	"fmt"
	"strings"

	"github.com/helpful-engineering/cth/model"
)

// MarkdownReport returns a full analysis report in GitHub-flavoured Markdown.
func MarkdownReport(inv model.Inventory, a FullAnalysis) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# CTH Analysis: %s (v%s)\n\n", inv.Programme, inv.Version)

	// ── Compression ──────────────────────────────────────────────────────────
	fmt.Fprintln(&b, "## Compression")
	fmt.Fprintf(&b, "| Metric | Value |\n|--------|-------|\n")
	fmt.Fprintf(&b, "| ρ_net | %.4f |\n", a.Rho)
	fmt.Fprintf(&b, "| ρ_gross (confirmed bits) | %.4f |\n",
		safeDivide(a.RhoDetail.GrossConfirmedBits, a.RhoDetail.TotalDenominator))
	fmt.Fprintf(&b, "| Confirmed bits (gross) | %.2f |\n", a.RhoDetail.GrossConfirmedBits)
	fmt.Fprintf(&b, "| Axiom entropy | %.2f bits |\n", a.RhoDetail.AxiomEntropyBits)
	fmt.Fprintf(&b, "| Information deficit | %.2f bits |\n", a.RhoDetail.InformationDeficit)
	fmt.Fprintf(&b, "| Denominator | %.2f bits |\n\n", a.RhoDetail.TotalDenominator)

	// ── Sensitivity ──────────────────────────────────────────────────────────
	fmt.Fprintln(&b, "## Axiom Entropy Sensitivity")
	fmt.Fprintf(&b, "| Scaling | ρ_net |\n|---------|-------|\n")
	fmt.Fprintf(&b, "| ½× axiom H | %.4f |\n", a.Sensitivity[0])
	fmt.Fprintf(&b, "| 1× (base) | %.4f |\n", a.Sensitivity[1])
	fmt.Fprintf(&b, "| 2× axiom H | %.4f |\n\n", a.Sensitivity[2])

	robust := "fragile (≤ 0.5)"
	if a.SensRatio > 0.5 {
		robust = "robust (> 0.5)"
	}
	fmt.Fprintf(&b, "Sensitivity ratio (2H/½H): **%.3f** — %s\n\n", a.SensRatio, robust)

	// ── Bridge centrality ─────────────────────────────────────────────────────
	if len(a.BridgeNodes) > 0 {
		fmt.Fprintln(&b, "## Bridge Centrality (top 5)")
		fmt.Fprintf(&b, "| Anchor | Domain count | Domains |\n|--------|-------------|----------|\n")
		for i, n := range a.BridgeNodes {
			if i >= 5 {
				break
			}
			fmt.Fprintf(&b, "| %s | %d | %s |\n", n.ID, n.DomainCount, strings.Join(n.Domains, ", "))
		}
		fmt.Fprintln(&b)
	}

	// ── Sediment ──────────────────────────────────────────────────────────────
	fmt.Fprintln(&b, "## Fidelity Sediment")
	fmt.Fprintf(&b, "- Laminar (μ≥0.999): %d chains\n", len(a.Sediment.Laminar))
	fmt.Fprintf(&b, "- Low sediment (μ≥0.90): %d chains\n", len(a.Sediment.LowSediment))
	fmt.Fprintf(&b, "- Moderate (μ≥0.70): %d chains\n", len(a.Sediment.Moderate))
	fmt.Fprintf(&b, "- Heavy (μ<0.70): %d chains\n", len(a.Sediment.Heavy))
	if a.Sediment.SharpPartition {
		fmt.Fprintf(&b, "\n**Sharp domain partition detected.**\n- Dirty-only: %v\n- Clean-only: %v\n",
			a.Sediment.DirtyOnlyDomains, a.Sediment.CleanOnlyDomains)
	}
	fmt.Fprintln(&b)

	// ── Eddies ────────────────────────────────────────────────────────────────
	if len(a.Eddies) > 0 {
		fmt.Fprintln(&b, "## Eddy Ranking (top open problems)")
		fmt.Fprintf(&b, "| Anchor | Proximity | Weighted gap | Nearest proven |\n")
		fmt.Fprintf(&b, "|--------|-----------|-------------|----------------|\n")
		for i, e := range a.Eddies {
			if i >= 5 {
				break
			}
			fmt.Fprintf(&b, "| %s | %.3f | %.1f | %s |\n",
				e.AnchorID, e.Proximity, e.WeightedGap, e.NearestProven)
		}
		fmt.Fprintln(&b)
	}

	// ── Ab initio ─────────────────────────────────────────────────────────────
	if len(a.AbInitio) > 0 {
		fmt.Fprintln(&b, "## Ab Initio Preferences")
		fmt.Fprintf(&b, "| Anchor | Best chain | Score | Fidelity | Input count |\n")
		fmt.Fprintf(&b, "|--------|-----------|-------|----------|-------------|\n")
		for _, r := range a.AbInitio {
			fmt.Fprintf(&b, "| %s | %s | %.4f | %.3f | %d |\n",
				r.AnchorID, r.BestChainID, r.Score, r.Fidelity, r.InputCount)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}

func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
