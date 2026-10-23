package mdcu

import (
	"math"
	"testing"
	"time"
)

// The generalised runner must reproduce canonical Test B bit-for-bit.
func TestQ1_GenMatchesCanonical(t *testing.T) {
	zb := loadSeeds(t, "testB_Z")
	for _, c := range []struct {
		lay Layout
		ov  bool
	}{{LayLocal, false}, {LayOverlap, true}} {
		a, b := RunRing(zb, c.ov), RunRingGen(zb, c.lay, assoc8)
		for i := range a {
			for k := range a[i] {
				if a[i][k] != b[i][k] {
					t.Fatalf("%v: site %d comp %d differs: %v vs %v", c.lay, i, k, a[i][k], b[i][k])
				}
			}
		}
	}
}

// Q1Measure's primary statistic must equal the canonical ringMetrics SVD3
// (canonical numbers 0.6097 local / 0.998 locked, one seed).
func TestQ1_MeasureMatchesCanonical(t *testing.T) {
	zb := loadSeeds(t, "testB_Z")
	for _, ov := range []bool{false, true} {
		Z := RunRing(zb, ov)
		want := ringMetrics(Z, [3][]int{{0}, {1}, {2}}).SVD3
		got := Q1Measure(Z).SVD3
		if math.Abs(want-got) > 1e-12 {
			t.Fatalf("overlap=%v SVD3 %v vs canonical %v", ov, got, want)
		}
		t.Logf("overlap=%v SVD3=%.4f %+v", ov, got, Q1Measure(Z))
	}
}

// The random null must be alternating in (a,b,c) and match the associator norm.
func TestQ1_RandomNullAlternatingAndNorm(t *testing.T) {
	r := newRNG(7)
	T := NewRandomAlt(r)
	x, y, z := make([]float64, 8), make([]float64, 8), make([]float64, 8)
	for k := 1; k < 8; k++ {
		x[k], y[k], z[k] = r.NormFloat64(), r.NormFloat64(), r.NormFloat64()
	}
	a, b := T.Apply(x, y, z), T.Apply(y, x, z)
	c := T.Apply(x, z, y)
	for d := range a {
		if math.Abs(a[d]+b[d]) > 1e-12 || math.Abs(a[d]+c[d]) > 1e-12 {
			t.Fatalf("not alternating at %d: %v %v %v", d, a[d], b[d], c[d])
		}
	}
	var ss float64
	for a := 1; a < 8; a++ {
		for b := 1; b < 8; b++ {
			for c := 1; c < 8; c++ {
				for d := 1; d < 8; d++ {
					ss += T.R[a][b][c][d] * T.R[a][b][c][d]
				}
			}
		}
	}
	if w := AssocTensorNormSq(); math.Abs(ss-w) > 1e-9*w {
		t.Fatalf("norm² %v vs associator %v", ss, w)
	}
}

// The true associator must be exactly alternating too (sanity for the null's premise).
func TestQ1_AssocAlternating(t *testing.T) {
	r := newRNG(3)
	x, y, z := Q1Init(3, 0, r)[0], Q1Init(3, 0, r)[1], Q1Init(3, 0, r)[2]
	a, b := assoc8(x, y, z), assoc8(y, x, z)
	for d := range a {
		if math.Abs(a[d]+b[d]) > 1e-12 {
			t.Fatalf("assoc not antisymmetric: %v vs %v", a[d], b[d])
		}
	}
}

// Tiny smoke only: one seed (9000, deliberately OUTSIDE the pre-registered grid
// seeds 1000-1199), N=12, every kind. Proves the harness runs; the grid is NOT run
// here (box-safety gate).
func TestQ1_Smoke(t *testing.T) {
	for _, k := range []string{"local", "overlap", "disjoint", "none", "rand3", "shuffle"} {
		t0 := time.Now()
		r := RunQ1Kind(12, 0.1, 9000, k)
		t.Logf("%-8s init SVD3=%.3f → final SVD3=%.3f phi0frac=%.2f planePhi0=%.3f frame=%d (%.2fs)",
			k, r.Init.SVD3, r.Final.SVD3, r.Final.Phi0Frac, r.Final.PlanePhi0, r.Final.Frame, time.Since(t0).Seconds())
		if math.IsNaN(r.Final.SVD3) {
			t.Fatalf("NaN for %s", k)
		}
	}
}
