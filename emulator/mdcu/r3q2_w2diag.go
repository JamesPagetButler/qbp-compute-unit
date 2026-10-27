package mdcu

// POST-HOC DIAGNOSTIC (arch seq 89): does the R3-Q2 W2 host-ns gap survive
// replacing the table-driven non-CD W2 with a straight-line kernel?
//
// Reading rule, fixed before any data: the gap is "explained" iff the frozen
// §3.3 yes-condition (CD median <= 0.5 x non-CD median, p < 0.05) no longer
// holds against the unrolled non-CD kernel; otherwise "survives". This changes
// no R3 verdict.

import (
	"fmt"
	"strings"
)

// BenchmarkPostHocW2NonCDUnrolled lives in r3q2_w2diag_bench_test.go.

// W2DiagReport formats the post-hoc comparison from parsed benchmark samples.
func W2DiagReport(m map[string][]float64) (string, error) {
	cd := m["BenchmarkR3Q2W2CD"]
	tab := m["BenchmarkR3Q2W2NonCD"]
	unr := m["BenchmarkPostHocW2NonCDUnrolled"]
	if len(cd) == 0 || len(tab) == 0 || len(unr) == 0 {
		return "", fmt.Errorf("need samples for BenchmarkR3Q2W2CD, BenchmarkR3Q2W2NonCD and BenchmarkPostHocW2NonCDUnrolled (got %d, %d, %d)", len(cd), len(tab), len(unr))
	}
	mc, mt, mu := Median(cd), Median(tab), Median(unr)
	pCDvsU := MannWhitneyP(cd, unr)
	pTvsU := MannWhitneyP(tab, unr)
	pCDvsT := MannWhitneyP(cd, tab)
	var sb strings.Builder
	sb.WriteString("# R3-Q2 W2 diagnostic: POST-HOC DIAGNOSTIC, no verdict change\n\n")
	sb.WriteString("| path | n | median ns |\n|---|---|---|\n")
	fmt.Fprintf(&sb, "| W2 CD (OMulCDMAC + dot7) | %d | %.0f |\n", len(cd), mc)
	fmt.Fprintf(&sb, "| W2 non-CD, table-driven (frozen) | %d | %.0f |\n", len(tab), mt)
	fmt.Fprintf(&sb, "| W2 non-CD, unrolled (post-hoc) | %d | %.0f |\n\n", len(unr), mu)
	sb.WriteString("| comparison | ratio of medians | p (Mann-Whitney) |\n|---|---|---|\n")
	fmt.Fprintf(&sb, "| CD / table-driven non-CD (R3-Q2 as frozen) | %.3f | %.2g |\n", mc/mt, pCDvsT)
	fmt.Fprintf(&sb, "| CD / unrolled non-CD | %.3f | %.2g |\n", mc/mu, pCDvsU)
	fmt.Fprintf(&sb, "| unrolled / table-driven non-CD | %.3f | %.2g |\n\n", mu/mt, pTvsU)
	if mc <= 0.5*mu && pCDvsU < 0.05 {
		sb.WriteString("Reading (rule fixed beforehand): the frozen yes-condition still holds against the unrolled kernel: **gap survives table removal**; the W2 disagreement is NOT explained by lookup overhead.\n")
	} else {
		sb.WriteString("Reading (rule fixed beforehand): the frozen yes-condition no longer holds against the unrolled kernel: **gap explained** as implementation shape (table lookups in the frozen non-CD W2), not architecture.\n")
	}
	sb.WriteString("\n> POST-HOC DIAGNOSTIC; no verdict change. Host ns measured on an FX-8350 without AVX2. The CD-MAC's wide-lane parallelism is serialised, so host ns is biased against CD.\n")
	return sb.String(), nil
}
