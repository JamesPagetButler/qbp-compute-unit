package mdcu

import (
	"strings"
	"testing"
)

func kzDecisionFixture() KZDecision {
	st := func(tau int, n float64) KZRungStat {
		return KZRungStat{Tau: tau, N: n, CILo: n * 0.9, CIHi: n * 1.1, PooledDefects: 100, AmbigFrac: 0.01}
	}
	return KZDecision{Q5a: "yes", Q5b: "yes", Q5c: "yes", Q5cLiteral: true, Alpha: 0.75, AlphaLo: 0.7, AlphaHi: 0.8,
		QuadC: 0.01, QuadCLo: -0.02, QuadCHi: 0.04, DeltaAIC: 1.2, SpanDecades: 1.8, Used: []int{0, 1},
		RungStatus: []string{"used", "used"}, MainStats: []KZRungStat{st(25, 0.05), st(50, 0.03)},
		NullStats: []KZRungStat{st(25, 0), st(50, 0)}, Resamples: 10000}
}

func TestKZFormatReport(t *testing.T) {
	s := KZFormatReport(kzDecisionFixture(), 24, 200, "complete")
	for _, want := range []string{"Q5a: yes", "Q5b: yes", "Q5c: yes", "L=24", "R=200", "spike/napkin grade",
		"no physics claim", "no CD speed claim", "10000 resamples"} {
		if !strings.Contains(s, want) {
			t.Errorf("report lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "LOW POWER") {
		t.Error("R=200 is not low power")
	}
	for _, R := range []int{25, 50} {
		if !strings.Contains(KZFormatReport(kzDecisionFixture(), 16, R, "complete"), "LOW POWER") {
			t.Errorf("R=%d must carry the low-power label", R)
		}
	}
}

func TestKZFormatReportNullFires(t *testing.T) {
	d := kzDecisionFixture()
	d.NullFires, d.NullReason, d.Q5c = true, "null replicate 3 at tau 25 has 1 non-ambiguous -1 cell(s)", "inconclusive (detector)"
	s := KZFormatReport(d, 16, 100, "complete")
	if !strings.Contains(s, "NULL FIRES") || !strings.Contains(s, d.NullReason) {
		t.Fatalf("%s", s)
	}
}
