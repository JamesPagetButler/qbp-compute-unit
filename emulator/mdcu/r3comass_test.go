package mdcu

// Tests for the POST-HOC comass diagnostic (r3comass.go). Written before the
// implementation. Fresh out-of-grid seeds only (9950-9959); none of these tests
// touches the R3 grid seeds 3000-3199.

import (
	"math"
	"testing"
)

// comassE123 is the elementary 3-form e^{123}: comass exactly 1, FrobSq 6.
func comassE123() *Form3 {
	var c [7][7][7]float64
	c[0][1][2], c[1][2][0], c[2][0][1] = 1, 1, 1
	c[1][0][2], c[0][2][1], c[2][1][0] = -1, -1, -1
	return NewForm3(c)
}

func comassScale(f *Form3, k float64) *Form3 {
	c := f.C
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for l := 0; l < 7; l++ {
				c[i][j][l] *= k
			}
		}
	}
	return NewForm3(c)
}

// comassRandOrtho is a deterministic random orthogonal 7x7 (Gram-Schmidt rows).
func comassRandOrtho(seed uint64) (R [7][7]float64) {
	rng := newRNG(seed)
	for i := 0; i < 7; i++ {
		for {
			var v [7]float64
			for k := range v {
				v[k] = rng.NormFloat64()
			}
			for j := 0; j < i; j++ {
				d := 0.0
				for k := 0; k < 7; k++ {
					d += v[k] * R[j][k]
				}
				for k := 0; k < 7; k++ {
					v[k] -= d * R[j][k]
				}
			}
			n := 0.0
			for k := range v {
				n += v[k] * v[k]
			}
			if n > 1e-6 {
				n = math.Sqrt(n)
				for k := range v {
					R[i][k] = v[k] / n
				}
				break
			}
		}
	}
	return
}

func comassRotate(f *Form3, R [7][7]float64) *Form3 {
	var t1, t2, t3 [7][7][7]float64
	for a := 0; a < 7; a++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				for i := 0; i < 7; i++ {
					t1[a][j][k] += R[a][i] * f.C[i][j][k]
				}
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for k := 0; k < 7; k++ {
				for j := 0; j < 7; j++ {
					t2[a][b][k] += R[b][j] * t1[a][j][k]
				}
			}
		}
	}
	for a := 0; a < 7; a++ {
		for b := 0; b < 7; b++ {
			for c := 0; c < 7; c++ {
				for k := 0; k < 7; k++ {
					t3[a][b][c] += R[c][k] * t2[a][b][k]
				}
			}
		}
	}
	return NewForm3(t3)
}

func TestComassElementaryFormIsOne(t *testing.T) {
	got := ComassEstimate(comassE123(), newRNG(9950), 20, 200)
	if got > 1+1e-9 || got < 1-1e-6 {
		t.Fatalf("comass(e123) = %.12f, want 1 (estimator is a lower bound, so never above 1)", got)
	}
}

func TestComassOfPhiIsOne(t *testing.T) {
	got := ComassEstimate(Phi(), newRNG(9951), 30, 300)
	if got > 1+1e-9 || got < 1-1e-5 {
		t.Fatalf("comass(phi) = %.12f, want 1 (Harvey-Lawson: phi has comass 1)", got)
	}
}

func TestComassHomogeneousDegreeOne(t *testing.T) {
	for _, f := range []*Form3{comassE123(), Phi()} {
		a := ComassEstimate(f, newRNG(9952), 30, 300)
		b := ComassEstimate(comassScale(f, 3.5), newRNG(9952), 30, 300)
		if math.Abs(b-3.5*a) > 1e-4*a {
			t.Fatalf("comass(3.5 f) = %.9f, want 3.5 * %.9f", b, a)
		}
	}
}

func TestComassRotationInvariant(t *testing.T) {
	R := comassRandOrtho(9953)
	for _, f := range []*Form3{comassE123(), Phi()} {
		a := ComassEstimate(f, newRNG(9954), 40, 300)
		b := ComassEstimate(comassRotate(f, R), newRNG(9954), 40, 300)
		if math.Abs(a-b) > 1e-4 {
			t.Fatalf("rotated comass %.9f vs %.9f", b, a)
		}
	}
}

func TestComassDeterministicAndStartsMonotone(t *testing.T) {
	f := gaussianForm(newRNG(9955))
	a := ComassEstimate(f, newRNG(9956), 10, 100)
	b := ComassEstimate(f, newRNG(9956), 10, 100)
	if a != b {
		t.Fatalf("same rng seed gave %v and %v", a, b)
	}
	// starts are consumed in order from the same stream, so more starts can
	// only keep or raise the best value found.
	c := ComassEstimate(f, newRNG(9956), 40, 100)
	if c < a-1e-15 {
		t.Fatalf("40 starts (%.12f) below 10 starts (%.12f)", c, a)
	}
}

// The ascent must beat (or match) brute-force random orthonormal triples; this
// guards against an estimator that silently returns its starting points.
func TestComassBeatsRandomTriples(t *testing.T) {
	for _, seed := range []uint64{9957, 9958, 9959} {
		f := gaussianForm(newRNG(seed))
		est := ComassEstimate(f, newRNG(seed+100), 30, 200)
		rng := newRNG(seed + 200)
		best := 0.0
		for n := 0; n < 20000; n++ {
			u, v, w := comassRandTriple(rng)
			if x := math.Abs(f.Eval(u[:], v[:], w[:])); x > best {
				best = x
			}
		}
		if est < best-1e-12 {
			t.Fatalf("seed %d: estimate %.9f below random-triple max %.9f", seed, est, best)
		}
	}
}

// The S2 spectrum recomputed from the seed's stream must match SampleS2 exactly
// on the quantities SampleS2 exposes. This is the guard that the replication of
// the RNG consumption order is faithful (the frozen sampler is not modified).
func TestS2SpectrumMatchesSampleS2(t *testing.T) {
	for _, seed := range []uint64{9950, 9952, 9954, 9956, 9958} {
		d := SampleS2(seed)
		s, ev := S2SpectrumFromSeed(seed)
		if s != d.S {
			t.Fatalf("seed %d: s %v != SampleS2 %v", seed, s, d.S)
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		tr := 0.0
		for _, e := range ev {
			lo, hi = math.Min(lo, e), math.Max(hi, e)
			tr += e
		}
		if math.Abs(tr-d.TraceS) > 1e-12 {
			t.Fatalf("seed %d: trace %v vs %v", seed, tr, d.TraceS)
		}
		cond := math.Exp(2 * s * (hi - lo))
		if math.Abs(cond-d.Cond) > 1e-12*d.Cond {
			t.Fatalf("seed %d: cond %v vs %v", seed, cond, d.Cond)
		}
	}
}

func TestGOpCubedFormula(t *testing.T) {
	for _, seed := range []uint64{9950, 9952} {
		s, ev := S2SpectrumFromSeed(seed)
		hi := math.Inf(-1)
		for _, e := range ev {
			hi = math.Max(hi, e)
		}
		want := math.Exp(3 * s * hi)
		if got := GOpCubed(seed); math.Abs(got-want) > 1e-12*want {
			t.Fatalf("seed %d: GOpCubed %v want %v", seed, got, want)
		}
		if GOpCubed(seed) < 1 {
			t.Fatalf("seed %d: traceless S has lambda_max >= 0, so ||g||_op >= 1", seed)
		}
	}
}

// PostHocComass bundles the per-seed diagnostics for the main-cell psi forms.
func TestPostHocComassFields(t *testing.T) {
	p := PostHocComass(9950)
	if p.Seed != 9950 || p.Stratum != "definite" {
		t.Fatalf("bad header %+v", p)
	}
	if !(p.PsiComass > 0 && p.PsiS1Comass > 0 && p.GOp3 >= 1) {
		t.Fatalf("even seed must fill S2, S1-definite and g: %+v", p)
	}
	q := PostHocComass(9951)
	if q.Stratum != "split" || q.GOp3 != 0 || q.PsiS1Comass != 0 || !(q.PsiComass > 0) {
		t.Fatalf("odd seed: only the split S1 form: %+v", q)
	}
	// Frobenius-matched forms have comass bounded below by the Frobenius norm
	// / sqrt(#ordered components) scale; just require a sane positive range.
	if p.PsiComass > 6 || p.PsiComass < 0.1 {
		t.Fatalf("implausible comass %v", p.PsiComass)
	}
	// deterministic
	if PostHocComass(9950) != p {
		t.Fatal("PostHocComass not deterministic")
	}
}
