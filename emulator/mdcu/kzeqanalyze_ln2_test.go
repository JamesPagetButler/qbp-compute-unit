package mdcu

// qbp-kz-g2 round 3 — the decision-level exact-equality boundary at ΔF*(16) = ln 2 (architect
// §I4 follow-up on #97, 2026-10-31: the `ΔF*(16) >= ln2` → `>` mutant survives because nothing
// runs the *decision* at exactly ln 2). TESTS ONLY; analysis code byte-identical to 45639ed.
//
// Construction: at (L = 16, t = 5) three seeds pool to a 15,000-value multiset whose shared-bin
// histogram (the L = 16 pooled range is pinned to [0, 60], width 1, by the anchors 0.0 and 60.0
// in the t = 5 data) is a flat floor of 227 counts with bumps to 454 = 2·227 at bins 19–21 and
// 39–41. KZEqSmooth3 gives a strict-max lower peak of 454 and a valley flat of 227, so
// ΔF = ln(454/227) = ln(2.0) BIT-EXACT (454.0/227.0 == 2.0). Every other point is a tight
// unimodal Gaussian well inside (0, 60), so ΔF* = 0 there; (a)'s other clauses hold trivially
// and (a) fires — unless the floor is `>` instead of `>=`, which this test KILLS.

import (
	"math"
	"math/rand/v2"
	"testing"
)

// kzLn2Multiset builds the deterministic 15,000-value e-multiset for (L=16, t=5), shuffled so
// each seed's slice is iid-like (Sokal τ ≈ ½, not under-sampled). Bin i has centre i+0.5;
// bin 0 carries one exact 0.0 and bin 59 one exact 60.0 to pin Lo = 0, Hi = 60 (width 1).
func kzLn2Multiset() []float64 {
	counts := make([]int, 60)
	for i := range counts {
		counts[i] = 227
	}
	for _, i := range []int{19, 20, 21, 39, 40, 41} {
		counts[i] = 454 // = 2·227 → smoothed peak 454, valley 227, ΔF = ln 2 exactly
	}
	for _, i := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 48, 49, 50, 51, 52, 53, 54, 55, 56} {
		counts[i]++ // 18 padding counts, all outside [18,42]: do not touch the barrier bins
	}
	vals := make([]float64, 0, 15000)
	for i := 0; i < 60; i++ {
		for k := 0; k < counts[i]; k++ {
			v := float64(i) + 0.5
			if i == 0 && k == 0 {
				v = 0.0
			}
			if i == 59 && k == 0 {
				v = 60.0
			}
			vals = append(vals, v)
		}
	}
	rr := rand.New(rand.NewPCG(424242, 7))
	rr.Shuffle(len(vals), func(a, b int) { vals[a], vals[b] = vals[b], vals[a] })
	return vals
}

func kzLn2Gen(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
	S := gauss(r, 5000, 0.3, 0.02)
	if L == 16 && math.Abs(T-1.15) < 1e-9 {
		all := kzLn2Multiset()
		return S, all[j*5000 : (j+1)*5000]
	}
	return S, gauss(r, 5000, 30, 2) // unimodal, well inside (0, 60)
}

// The decision at ΔF*(16) exactly ln 2 must fire (a). KILLS the `ΔF*(16) >= ln2` → `>` mutant.
func TestKZEqDecideDeltaFStarExactlyLn2(t *testing.T) {
	d := decide(t, synthSet(31, kzLn2Gen))
	l16 := d.Ls[2]
	star := l16.Points[l16.TStarPos]
	t.Logf("ΔF*(16) = %.17g  ln2 = %.17g  Δ = %g  resolved=%v  outcome=%s rule=%s A=%v",
		l16.DeltaFStar, math.Log(2), l16.DeltaFStar-math.Log(2), star.Barrier.Resolved, d.Outcome, d.Rule, d.A)
	if l16.DeltaFStar != math.Log(2) {
		t.Fatalf("ΔF*(16) = %.17g is not bit-exactly ln 2 = %.17g — the construction did not land on the boundary", l16.DeltaFStar, math.Log(2))
	}
	if !star.Barrier.Resolved {
		t.Fatalf("(16, T*) barrier not resolved; the ln2 case needs a resolved barrier at exactly ln2")
	}
	if d.Outcome != "H-FO" || d.Rule != "(a)" || !d.A {
		t.Fatalf("outcome %q rule %q A %v, want H-FO (a): the `>=` floor must fire at exactly ln2", d.Outcome, d.Rule, d.A)
	}
}

// Note on the D = 3 gate boundary (architect §I4 follow-up): a bit-exact D = 3 decision case is
// NOT reliably constructible — D = |Δmean| / √(SE_hot² + SE_cold²) is a quotient with a square
// root, and a cold series shifted by 3·se lands D at 2.9999999999999805 (float rounding of the
// mean-of-shifted-values and the division), never exactly 3.0. Since the `D > 3` → `>=` mutant
// differs only at exactly 3.0, it cannot be killed by data. Coverage for the gate is instead:
// the constant (KZEqDThreshold == 3, TestKZEqStartDepThresholdBoundary), the tight bracket
// (D = 3.074 flagged / 2.931 not), and the strict `>` operator (kzeqanalyze.go:626, read-verified
// by the architect). The run/analysis binary is 45639ed with that operator — no regression risk.
