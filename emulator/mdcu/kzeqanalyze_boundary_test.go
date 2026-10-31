package mdcu

// qbp-kz-g2 round 3 — threshold / boundary coverage for rule (a) and the gates (architect
// §I4 on qbp-compute-unit#97, 2026-10-31: five mutations on (a)'s thresholds survived the
// first suite, incl. dropping the L-monotonicity clause). TESTS ONLY — no analysis code
// changes; the run commit's analyzer stays byte-identical. Synthetic series; no simulation.

import (
	"math"
	"math/rand/v2"
	"testing"
)

// kzBarSpec places a resolved two-state barrier of the given separation/width at one T of one
// L; everywhere else the point is a single narrowing Gaussian (contGen-like).
type kzBarSpec struct{ T, sep, sd float64 }

func kzBarrierGen(spec map[int]kzBarSpec) synthGen {
	return func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		S := gauss(r, 5000, 0.3, 0.02)
		if s, ok := spec[L]; ok && math.Abs(T-s.T) < 1e-9 {
			return S, mixture(r, 5000, 0.5, 1.1-s.sep/2, 1.1+s.sep/2, s.sd)
		}
		return S, gauss(r, 5000, 1.0+0.3*(T-1.08), 0.03*8/float64(L))
	}
}

// (a) clause ΔF*(16) ≥ ΔF*(12) − ε: a resolved barrier at L = 16 that is ≥ ln 2 but has SHRUNK
// from L = 12 by more than ε must NOT fire (a) — it reads (c). A build that drops the 16-vs-12
// L-monotonicity clause would wrongly read H-FO and is KILLED here.
func TestKZEqDecideBarrierShrinks1612(t *testing.T) {
	d := decide(t, synthSet(21, kzBarrierGen(map[int]kzBarSpec{
		12: {1.15, 0.40, 0.030}, // deep at L = 12
		16: {1.15, 0.26, 0.045}, // ≥ ln 2 but shallower at L = 16
	})))
	l16, l12 := d.Ls[2], d.Ls[1]
	t.Logf("dF*: L8=%.4f L12=%.4f L16=%.4f  eps1612=%.4f  outcome=%s rule=%s sub=%q A=%v",
		d.Ls[0].DeltaFStar, l12.DeltaFStar, l16.DeltaFStar, d.Eps1612, d.Outcome, d.Rule, d.SubLabel, d.A)
	if d.Rule != "(c)" || d.Outcome != "inconclusive" || d.SubLabel != "" {
		t.Fatalf("outcome %q rule %q sub %q, want plain (c)", d.Outcome, d.Rule, d.SubLabel)
	}
	if d.A {
		t.Fatalf("(a) fired although the barrier shrank 12->16")
	}
	if l16.DeltaFStar < KZEqDeltaF0 {
		t.Fatalf("ΔF*(16) = %.4f < ln2: the ΔF0 clause, not the monotonicity clause, is failing", l16.DeltaFStar)
	}
	if !(l16.DeltaFStar < l12.DeltaFStar-d.Eps1612) {
		t.Fatalf("ΔF*(16) %.4f not below ΔF*(12) %.4f − ε %.4f: the 16-vs-12 clause is not the one failing", l16.DeltaFStar, l12.DeltaFStar, d.Eps1612)
	}
}

// (a) clause ΔF*(12) ≥ ΔF*(8) − ε: isolate the 12-vs-8 leg. L = 8 carries the deepest barrier
// (at a coarse point it runs, T = 1.14), L = 12 shallower by more than ε, L = 16 ≥ ln 2 and
// ≥ ΔF*(12) − ε (so the other two clauses hold). Dropping the 12-vs-8 clause reads H-FO: KILLED.
func TestKZEqDecideBarrierShrinks128(t *testing.T) {
	d := decide(t, synthSet(22, kzBarrierGen(map[int]kzBarSpec{
		8:  {1.14, 0.40, 0.030}, // deepest at L = 8 (coarse point)
		12: {1.15, 0.26, 0.045}, // shallower at L = 12
		16: {1.15, 0.30, 0.040}, // ≥ ln 2 and ≥ ΔF*(12) − ε
	})))
	l16, l12, l8 := d.Ls[2], d.Ls[1], d.Ls[0]
	t.Logf("dF*: L8=%.4f L12=%.4f L16=%.4f  eps1612=%.4f eps128=%.4f  outcome=%s rule=%s A=%v",
		l8.DeltaFStar, l12.DeltaFStar, l16.DeltaFStar, d.Eps1612, d.Eps128, d.Outcome, d.Rule, d.A)
	if d.Rule != "(c)" || d.Outcome != "inconclusive" {
		t.Fatalf("outcome %q rule %q, want (c)", d.Outcome, d.Rule)
	}
	if d.A {
		t.Fatalf("(a) fired although the barrier shrank 8->12")
	}
	if l16.DeltaFStar < KZEqDeltaF0 || !(l16.DeltaFStar >= l12.DeltaFStar-d.Eps1612) {
		t.Fatalf("the 16-vs-12 leg or ΔF0 is the failing one, not 12-vs-8 (L16=%.4f L12=%.4f eps=%.4f)", l16.DeltaFStar, l12.DeltaFStar, d.Eps1612)
	}
	if !(l12.DeltaFStar < l8.DeltaFStar-d.Eps128) {
		t.Fatalf("ΔF*(12) %.4f not below ΔF*(8) %.4f − ε %.4f: the 12-vs-8 clause is not the one failing", l12.DeltaFStar, l8.DeltaFStar, d.Eps128)
	}
}

// (a) clause ΔF*(16) ≥ ΔF₀ = ln 2: resolved barriers that do NOT shrink with L but sit BELOW
// ln 2 must read (c), not H-FO. A build that lowers or drops the ln 2 floor is KILLED.
func TestKZEqDecideBarrierBelowLn2(t *testing.T) {
	d := decide(t, synthSet(23, kzBarrierGen(map[int]kzBarSpec{
		12: {1.15, 0.16, 0.050}, // resolved but ΔF < ln 2 (shallow dip)
		16: {1.15, 0.16, 0.050},
	})))
	l16 := d.Ls[2]
	star := l16.Points[l16.TStarPos]
	t.Logf("dF*(16)=%.4f (ln2=%.4f) resolved=%v  outcome=%s rule=%s A=%v",
		l16.DeltaFStar, KZEqDeltaF0, star.Barrier.Resolved, d.Outcome, d.Rule, d.A)
	if d.Rule != "(c)" || d.Outcome != "inconclusive" {
		t.Fatalf("outcome %q rule %q, want (c)", d.Outcome, d.Rule)
	}
	if d.A {
		t.Fatalf("(a) fired on a sub-ln2 barrier")
	}
	if !star.Barrier.Resolved {
		t.Fatalf("(16, T*) barrier is not resolved — the test needs a resolved-but-sub-ln2 barrier, got ΔF*=%.4f", l16.DeltaFStar)
	}
	if !(l16.DeltaFStar > 0 && l16.DeltaFStar < KZEqDeltaF0) {
		t.Fatalf("ΔF*(16) = %.4f is not in (0, ln2); this does not exercise the ΔF0 floor", l16.DeltaFStar)
	}
}

// Exact boundaries of the 3σ resolved-floor and the ΔF = ln 2 value in KZEqBarrierOf.
func TestKZEqBarrierExactFloorAndLn2(t *testing.T) {
	// 3σ floor exactly: lower peak 65, valley 35 -> hl−hv = 30 == 3·√(hl+hv) = 3·√100 = 30;
	// strict `>` => NOT resolved.
	eq := make([]float64, 60)
	for i := 21; i <= 39; i++ {
		eq[i] = 50
	}
	eq[20], eq[40], eq[30] = 65, 70, 35
	if b := KZEqBarrierOf(eq); b.Resolved || b.DeltaF != 0 {
		t.Fatalf("hl−hv = 30 == 3√100 must be UNresolved (strict >): %+v", b)
	}
	// one count deeper: valley 34 -> hl−hv = 31 > 3·√(99) = 29.85 -> resolved.
	eq[30] = 34
	if b := KZEqBarrierOf(eq); !b.Resolved || b.HLower != 65 || b.HValley != 34 {
		t.Fatalf("hl−hv = 31 > 3√99 must be resolved: %+v", b)
	}
	// ΔF == ln 2 exactly: hl = 200, hv = 100.
	ln2 := make([]float64, 60)
	for i := 21; i <= 39; i++ {
		ln2[i] = 150
	}
	ln2[20], ln2[40], ln2[30] = 200, 220, 100
	b := KZEqBarrierOf(ln2)
	if !b.Resolved || math.Abs(b.DeltaF-math.Log(2)) > 1e-12 {
		t.Fatalf("hl/hv = 2 must give ΔF = ln 2 exactly: %+v", b)
	}
	if b.DeltaF < KZEqDeltaF0 {
		t.Fatalf("ΔF = ln2 must satisfy the (a) threshold ΔF ≥ ΔF0 (>=): ΔF=%.17g ΔF0=%.17g", b.DeltaF, KZEqDeltaF0)
	}
}

// The start-dependence gate is D > 3 (strict). Bracket the threshold with a cold-start shift
// tuned just above and just below D = 3; the flag must track the strict comparison.
func TestKZEqStartDepThresholdBoundary(t *testing.T) {
	if KZEqDThreshold != 3.0 {
		t.Fatalf("D threshold = %g, want 3", KZEqDThreshold)
	}
	// deterministic 20-block alternating series → exact, stable block SE (no RNG flake); cold =
	// hot shifted by δ. With pooled hot SE ≈ 0.0801 and cold SE ≈ 0.1147, D = δ / 0.1399, so
	// δ = 0.43 → D ≈ 3.07 (just above) and δ = 0.41 → D ≈ 2.93 (just below) bracket D = 3.
	blk := func(shift float64) []float64 {
		x := make([]float64, 5000)
		for i := range x {
			x[i] = float64((i/250)%2) + shift
		}
		return x
	}
	hot := [][]float64{blk(0), blk(0)}
	above := KZEqStartDependence(hot, [][]float64{blk(0.43)}, KZEqBlock)
	below := KZEqStartDependence(hot, [][]float64{blk(0.41)}, KZEqBlock)
	t.Logf("D above=%.4f below=%.4f (threshold 3)", above.D, below.D)
	if !(above.D > KZEqDThreshold) {
		t.Fatalf("above-shift D = %.4f, want > 3", above.D)
	}
	if !(below.D <= KZEqDThreshold) {
		t.Fatalf("below-shift D = %.4f, want <= 3", below.D)
	}
}
