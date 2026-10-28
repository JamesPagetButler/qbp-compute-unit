package mdcu

import (
	"fmt"
	"strings"
)

// KZFormatReport renders a KZDecision as plain text. It states the grade of the spike and
// carries the low-power label for R of 25 or 50.
func KZFormatReport(d KZDecision, L, R int, verdict string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "mdcu-kz analysis  L=%d R=%d  run verdict: %s\n", L, R, verdict)
	b.WriteString("Branch B spike/napkin grade: no physics claim and no CD speed claim at any reading.\n")
	if R <= 50 {
		fmt.Fprintf(&b, "LOW POWER: R=%d. A \"no\" here is weak evidence; the frozen Q5a rule can miss a true power law.\n", R)
	}
	fmt.Fprintf(&b, "\nQ5a: %s\nQ5b: %s\nQ5c: %s   (literal comparison: %v)\n", d.Q5a, d.Q5b, d.Q5c, d.Q5cLiteral)
	if d.NullFires {
		fmt.Fprintf(&b, "NULL FIRES: %s\n", d.NullReason)
	}
	fmt.Fprintf(&b, "\nalpha = %.4f  95%% CI [%.4f, %.4f]\n", d.Alpha, d.AlphaLo, d.AlphaHi)
	fmt.Fprintf(&b, "quadratic coefficient = %.4f  95%% CI [%.4f, %.4f]\n", d.QuadC, d.QuadCLo, d.QuadCHi)
	fmt.Fprintf(&b, "delta AIC (line - quadratic) = %.3f   span = %.3f decades   bootstrap: %d resamples\n",
		d.DeltaAIC, d.SpanDecades, d.Resamples)
	b.WriteString("\ntau_Q   main n [95% CI]                      null n [95% CI]                      defects  ambig    status\n")
	for i := range d.MainStats {
		m, n := d.MainStats[i], d.NullStats[i]
		st := ""
		if i < len(d.RungStatus) {
			st = d.RungStatus[i]
		}
		fmt.Fprintf(&b, "%5d   %.6f [%.6f, %.6f]   %.6f [%.6f, %.6f]   %7d  %.4f   %s\n",
			m.Tau, m.N, m.CILo, m.CIHi, n.N, n.CILo, n.CIHi, m.PooledDefects, m.AmbigFrac, st)
	}
	return b.String()
}
