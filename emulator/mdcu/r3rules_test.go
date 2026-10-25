package mdcu

// Tests for the frozen §1.5 decision rules (r3rules.go), on synthetic data with
// known answers. Written before r3rules.go, and before any R3 grid output exists.

import (
	"math"
	"testing"
)

// synth builds a paired set of runs for one arm: lockN of n seeds 3-lock.
func synth(arm R3Arm, seeds []uint64, s3 func(i int) float64, lock func(i int) bool) []R3Run {
	out := make([]R3Run, len(seeds))
	for i, s := range seeds {
		cl := "diffuse"
		if lock(i) {
			cl = "3-lock"
		}
		var st R3Stats
		st.S3 = s3(i)
		st.Class = cl
		out[i] = R3Run{Cell: "main", Arm: arm, Seed: s, Stats: st}
	}
	return out
}

func seq(lo uint64, n int) []uint64 {
	s := make([]uint64, n)
	for i := range s {
		s[i] = lo + uint64(i)
	}
	return s
}

func constf(v float64) func(int) float64 { return func(int) float64 { return v } }
func firstK(k int) func(int) bool        { return func(i int) bool { return i < k } }

func TestRateAndPairing(t *testing.T) {
	a := synth(ArmFull, seq(0, 10), constf(1), firstK(7))
	b := synth(ArmGram, seq(5, 10), constf(1), firstK(10)) // overlaps seeds 5..9
	pa, pb := PairBySeed(a, b)
	if len(pa) != 5 || len(pb) != 5 || pa[0].Seed != 5 || pb[0].Seed != 5 {
		t.Fatalf("pairing wrong: %d/%d", len(pa), len(pb))
	}
	if r := LockRate(a); math.Abs(r-0.7) > 1e-12 {
		t.Fatalf("rate %v", r)
	}
}

func TestContrastIdenticalIsZero(t *testing.T) {
	a := synth(ArmFull, seq(0, 100), func(i int) float64 { return 0.9 + 0.001*float64(i%7) }, firstK(100))
	c := R3Contrast("same", a, a)
	if c.N != 100 || c.Median != 0 || c.Lo != 0 || c.Hi != 0 {
		t.Fatalf("%+v", c)
	}
}

func TestContrastDeterministic(t *testing.T) {
	a := synth(ArmFull, seq(0, 100), func(i int) float64 { return 0.9 + 0.001*float64(i%7) }, firstK(100))
	b := synth(ArmGram, seq(0, 100), func(i int) float64 { return 0.5 + 0.003*float64(i%11) }, firstK(40))
	c1, c2 := R3Contrast("x", a, b), R3Contrast("x", a, b)
	if c1 != c2 {
		t.Fatal("bootstrap not deterministic")
	}
	if c1.Lo > c1.Median || c1.Hi < c1.Median {
		t.Fatalf("CI does not bracket the median: %+v", c1)
	}
}

func TestQ1aYesNoInconclusive(t *testing.T) {
	full := synth(ArmFull, seq(0, 200), constf(0.99), firstK(200))
	// yes: gram reproduces full
	gYes := synth(ArmGram, seq(0, 200), constf(0.99), firstK(200))
	if v := VerdictQ1a(R3Contrast("a", full, gYes), LockRate(gYes)); v != "yes" {
		t.Fatalf("want yes, got %s", v)
	}
	// no: gram collapses (rate < 50%)
	gNo := synth(ArmGram, seq(0, 200), constf(1.0), firstK(20))
	if v := VerdictQ1a(R3Contrast("a", full, gNo), LockRate(gNo)); v != "no" {
		t.Fatalf("want no (rate<50%%), got %s", v)
	}
	// no: gram locks often (70%) but s3 gap >= 0.15 with CI excluding 0
	gNo2 := synth(ArmGram, seq(0, 200), constf(0.80), firstK(140))
	if v := VerdictQ1a(R3Contrast("a", full, gNo2), LockRate(gNo2)); v != "no" {
		t.Fatalf("want no (gap), got %s", v)
	}
	// inconclusive: gram locks 80% with tiny gap
	gInc := synth(ArmGram, seq(0, 200), constf(0.97), firstK(160))
	if v := VerdictQ1a(R3Contrast("a", full, gInc), LockRate(gInc)); v != "inconclusive" {
		t.Fatalf("want inconclusive, got %s", v)
	}
}

// Rate-gap boundary: exactly 10 points counts as within margin (≤ 10).
func TestQ1aTenPointBoundaryIsInclusive(t *testing.T) {
	full := synth(ArmFull, seq(0, 200), constf(0.99), firstK(200))
	g := synth(ArmGram, seq(0, 200), constf(0.99), firstK(180)) // 90%: rate>=90, gap = 10 points
	if v := VerdictQ1a(R3Contrast("b", full, g), LockRate(g)); v != "yes" {
		t.Fatalf("90%% vs 100%% with identical s3 must be yes (≤10pt inclusive), got %s", v)
	}
	g2 := synth(ArmGram, seq(0, 200), constf(0.99), firstK(179)) // 89.5% → < 90
	if v := VerdictQ1a(R3Contrast("b", full, g2), LockRate(g2)); v == "yes" {
		t.Fatal("89.5% gram rate must not be yes")
	}
}

func q1bSets(n int, phiS3, psiS3 float64, phiLock, psiLock int) (phi, psi []R3Run) {
	phi = synth(ArmPhi, seq(0, n), constf(phiS3), firstK(phiLock))
	psi = synth(ArmPsi, seq(0, n), constf(psiS3), firstK(psiLock))
	return
}

func TestQ1bRules(t *testing.T) {
	phi, psi := q1bSets(100, 0.97, 0.97, 90, 88)
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "no" {
		t.Fatalf("equivalent arms: want no, got %s", v)
	}
	phi, psi = q1bSets(100, 0.99, 0.70, 90, 30) // 60 point rate gap, large s3 gap
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "yes" {
		t.Fatalf("phi locks far more: want yes, got %s", v)
	}
	phi, psi = q1bSets(100, 0.90, 0.99, 80, 95)
	phi[0].Stats.S3 = 0.90 // keep constant
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "yes-reversed" && v != "inconclusive" {
		t.Fatalf("psi better: got %s", v)
	}
	// reversed needs Δ ≤ −0.15: here Δ = −0.09 → inconclusive
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "inconclusive" {
		t.Fatalf("Δ=-0.09 must be inconclusive, got %s", v)
	}
	phi, psi = q1bSets(100, 0.60, 0.99, 80, 95)
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "yes-reversed" {
		t.Fatalf("Δ=-0.39: want yes-reversed, got %s", v)
	}
}

func TestQ1bPreconditionAndPower(t *testing.T) {
	// K-phi does not lock (rate 20%): precondition fails.
	phi, psi := q1bSets(100, 0.7, 0.7, 20, 20)
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "inconclusive (φ-only does not lock)" {
		t.Fatalf("precondition: got %s", v)
	}
	// fewer than 50 paired seeds → underpowered (checked first)
	phi, psi = q1bSets(49, 0.99, 0.99, 49, 49)
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "inconclusive (underpowered)" {
		t.Fatalf("power: got %s", v)
	}
	phi, psi = q1bSets(50, 0.99, 0.99, 50, 50)
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "no" {
		t.Fatalf("n=50 is enough: got %s", v)
	}
}

func TestQ1bYesViaFortyPointRateGap(t *testing.T) {
	// s3 gap is small (Δ≈0.02) but the lock rates differ by 40 points.
	phi, psi := q1bSets(100, 0.98, 0.96, 90, 50)
	if v := VerdictQ1b(R3Contrast("q", phi, psi), phi, psi); v != "yes" {
		t.Fatalf("40-point rate gap: want yes, got %s", v)
	}
}

func TestReadingTable(t *testing.T) {
	cases := []struct {
		gate              bool
		q1a, bdef, bsplit string
		wantContains      string
	}{
		{true, "yes", "yes", "no", "Gram term alone"},
		{true, "no", "no", "no", "any 3-form does"},
		{true, "no", "no", "yes", "GL(7) type matters"},
		{true, "no", "yes", "yes", "G₂-specific"},
		{true, "no", "yes", "no", "Inconclusive, or a mixed"},
		{true, "no", "inconclusive", "no", "Inconclusive, or a mixed"},
		{true, "inconclusive", "no", "no", "Inconclusive, or a mixed"},
		{false, "no", "yes", "yes", "non-replication"},
	}
	for i, c := range cases {
		got := ReadingRow(c.gate, c.q1a, c.bdef, c.bsplit)
		if !contains(got, c.wantContains) {
			t.Errorf("case %d: reading %q does not contain %q", i, got, c.wantContains)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})())
}

func TestSpearman(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	if r := Spearman(x, []float64{10, 20, 30, 40, 50}); math.Abs(r-1) > 1e-12 {
		t.Fatalf("monotone up: %v", r)
	}
	if r := Spearman(x, []float64{9, 7, 5, 3, 1}); math.Abs(r+1) > 1e-12 {
		t.Fatalf("monotone down: %v", r)
	}
	// ties use average ranks
	if r := Spearman([]float64{1, 1, 2, 2}, []float64{1, 2, 3, 4}); r < 0.8 || r > 1 {
		t.Fatalf("ties: %v", r)
	}
	if !math.IsNaN(Spearman([]float64{1, 1, 1}, []float64{1, 2, 3})) {
		t.Fatal("constant input must give NaN")
	}
}

func TestMedianAndQuantile(t *testing.T) {
	if m := Median([]float64{3, 1, 2}); m != 2 {
		t.Fatal(m)
	}
	if m := Median([]float64{4, 1, 2, 3}); m != 2.5 {
		t.Fatal(m)
	}
}
