package mdcu

import (
	"math"
	"testing"
)

func kzFreeSeeds(n int, base uint64) []uint64 {
	s := make([]uint64, n)
	for i := range s {
		s[i] = base + uint64(i)
	}
	return s
}

func TestKZSmoothFieldBasics(t *testing.T) {
	L := 8
	q0 := KZSmoothField(L, 0, 13400)
	for _, q := range q0 {
		if q != [4]float64{1, 0, 0, 0} {
			t.Fatalf("A=0 must be the identity, got %v", q)
		}
	}
	a := KZSmoothField(L, 6, 13400)
	b := KZSmoothField(L, 6, 13400)
	c := KZSmoothField(L, 6, 13401)
	same, diff := true, false
	for i := range a {
		if a[i] != b[i] {
			same = false
		}
		if a[i] != c[i] {
			diff = true
		}
		n := a[i][0]*a[i][0] + a[i][1]*a[i][1] + a[i][2]*a[i][2] + a[i][3]*a[i][3]
		if math.Abs(n-1) > 1e-12 {
			t.Fatalf("site %d not unit: %v", i, n)
		}
	}
	if !same || !diff {
		t.Fatalf("determinism=%v seed-sensitivity=%v", same, diff)
	}
}

func TestKZAmbigMonotoneEnds(t *testing.T) {
	p := Amend1Z2Params()
	seeds := kzFreeSeeds(3, 13400)
	f0, cells, _ := KZAmbigFracPooled(12, 0, seeds, p)
	if f0 != 0 || cells == 0 {
		t.Fatalf("A=0 ambiguity must be 0, got %v over %d cells", f0, cells)
	}
	fHi, _, _ := KZAmbigFracPooled(12, KZAmpBracketHi, seeds, p)
	if fHi <= KZAmpTargetFrac {
		t.Fatalf("A=32 should be rougher than the target, got %v", fHi)
	}
}

func TestKZAmplitudeHalf(t *testing.T) {
	p := Amend1Z2Params()
	seeds := kzFreeSeeds(4, 13410)
	r := KZAmplitudeHalf(12, seeds, p)
	t.Logf("L=12 A_half=%.4f frac=%.4f it=%d status=%s", r.A, r.Frac, r.Iterations, r.Status)
	if r.Status != "converged" {
		t.Fatalf("status %s", r.Status)
	}
	if math.Abs(r.Frac-KZAmpTargetFrac) > KZAmpTolerance {
		t.Fatalf("fraction %v outside tolerance", r.Frac)
	}
	// re-evaluating at the returned A reproduces the fraction
	f, _, _ := KZAmbigFracPooled(12, r.A, seeds, p)
	if f != r.Frac {
		t.Fatalf("not reproducible: %v vs %v", f, r.Frac)
	}
	if r2 := KZAmplitudeHalf(12, seeds, p); r2 != r {
		t.Fatalf("bisection not deterministic: %v vs %v", r2, r)
	}
}

func TestKZCalibrateSmooth(t *testing.T) {
	p := Amend1Z2Params()
	fs, ts := kzFreeSeeds(4, 13410), kzFreeSeeds(4, 13420)
	c := KZCalibrateSmooth(12, fs, ts, p)
	if len(c.Rows) != len(KZSigmaGrid) {
		t.Fatalf("rows %d", len(c.Rows))
	}
	// sigma* is consistent with the rows by the rule
	s, fb := KZSigmaStar(c.Rows)
	if s != c.Sigma || fb != c.Fallback {
		t.Fatalf("sigma* mismatch")
	}
	// the untilted fraction at A_half is the one the bisection reported
	f, _, _ := KZAmbigFracPooled(12, c.Amp.A, fs, p)
	if f != c.Amp.Frac {
		t.Fatalf("frac %v vs %v", f, c.Amp.Frac)
	}
	for _, r := range c.Rows {
		t.Logf("sigma=%.2f pooled=%.4f neg=%d defects=%d", r.Sigma, r.PooledAmbigFrac, r.NegCells, r.Defects)
	}
	t.Logf("A_half=%.4f sigma*=%g fallback=%v", c.Amp.A, c.Sigma, c.Fallback)
}
