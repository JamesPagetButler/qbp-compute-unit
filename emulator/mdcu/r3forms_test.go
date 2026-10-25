package mdcu

// Round-3 tests for the 3-form layer (doc/spikes/prereg/mdcu-round3-prereg-FROZEN-v1.md,
// sections 1.1-1.2 and 5). Written before r3forms.go.
//
// Every test here uses out-of-grid seeds only (never 3000-3199, the round-3 main grid).

import (
	"math"
	"math/rand/v2"
	"testing"
)

// testImag draws a random imaginary 8-vector (not unit).
func testImag(rng *rand.Rand) []float64 {
	v := make([]float64, 8)
	for k := 1; k < 8; k++ {
		v[k] = rng.NormFloat64()
	}
	return v
}

func TestPhiFrobeniusAndAlternating(t *testing.T) {
	phi := Phi()
	if got := phi.FrobSq(); math.Abs(got-42) > 1e-12 {
		t.Fatalf("‖φ‖² = %v, want 42", got)
	}
	// 7 Fano lines x 6 orderings = 42 nonzero entries, each ±1.
	n := 0
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			for k := 0; k < 7; k++ {
				v := phi.C[i][j][k]
				if v != 0 {
					n++
					if math.Abs(math.Abs(v)-1) > 1e-15 {
						t.Fatalf("φ[%d][%d][%d] = %v, want ±1", i, j, k, v)
					}
				}
				if phi.C[i][j][k] != -phi.C[j][i][k] || phi.C[i][j][k] != -phi.C[i][k][j] {
					t.Fatalf("φ not alternating at %d,%d,%d", i, j, k)
				}
			}
		}
	}
	if n != 42 {
		t.Fatalf("φ has %d nonzero entries, want 42", n)
	}
}

// φ must be the package's own phi0: <xy, z> with the reference product.
func TestPhiMatchesPhi0(t *testing.T) {
	rng := newRNG(9201)
	phi := Phi()
	for n := 0; n < 200; n++ {
		a, b, c := testImag(rng), testImag(rng), testImag(rng)
		if d := math.Abs(phi.Eval(a, b, c) - phi0(a, b, c)); d > 1e-12 {
			t.Fatalf("φ.Eval vs phi0 differ by %v", d)
		}
	}
}

// The Harvey-Lawson identity, the algebra the whole round rests on:
// ‖[a,b,c]‖² = 4 (Gram₃ − φ²) for imaginary a,b,c (to 1e-12, relative).
func TestHarveyLawsonIdentity(t *testing.T) {
	rng := newRNG(9202)
	phi := Phi()
	worst := 0.0
	for n := 0; n < 2000; n++ {
		a, b, c := testImag(rng), testImag(rng), testImag(rng)
		lhs := norm2(assoc8(a, b, c))
		f := phi.Eval(a, b, c)
		rhs := 4 * (Gram3(a, b, c) - f*f)
		scale := 1 + math.Abs(lhs)
		worst = math.Max(worst, math.Abs(lhs-rhs)/scale)
	}
	if worst > 1e-12 {
		t.Fatalf("Harvey–Lawson identity violated: worst relative error %.3e", worst)
	}
}

func TestGram3MatchesWedge(t *testing.T) {
	rng := newRNG(9203)
	for n := 0; n < 200; n++ {
		a, b, c := testImag(rng), testImag(rng), testImag(rng)
		if d := math.Abs(Gram3(a, b, c) - Gram3Wedge(a, b, c)); d > 1e-9*(1+Gram3(a, b, c)) {
			t.Fatalf("Gram3 vs wedge (35-component) differ by %v", d)
		}
	}
}

// Gradients of every scalar arm energy are checked against central differences.
func TestScalarEnergyGradients(t *testing.T) {
	rng := newRNG(9204)
	psi := mustS2(t, 9204).Form
	cases := []struct {
		name string
		en   R3Energy
		eval func(x, y, z []float64) float64
	}{
		{"K-gram", NewScalarEnergy(true, nil), func(x, y, z []float64) float64 { return 4 * Gram3(x, y, z) }},
		{"K-phi", NewScalarEnergy(false, Phi()), func(x, y, z []float64) float64 { f := Phi().Eval(x, y, z); return -4 * f * f }},
		{"K-psi", NewScalarEnergy(false, psi), func(x, y, z []float64) float64 { f := psi.Eval(x, y, z); return -4 * f * f }},
		{"K-psigram", NewScalarEnergy(true, psi), func(x, y, z []float64) float64 {
			f := psi.Eval(x, y, z)
			return 4*Gram3(x, y, z) - 4*f*f
		}},
	}
	for _, c := range cases {
		for pos := 0; pos < 3; pos++ {
			for trial := 0; trial < 20; trial++ {
				args := [3][]float64{testImag(rng), testImag(rng), testImag(rng)}
				g := c.en.Grad(args[0], args[1], args[2], pos)
				if g[0] != 0 {
					t.Fatalf("%s: gradient has a real component %v", c.name, g[0])
				}
				for k := 1; k < 8; k++ {
					const h = 1e-6
					p, m := [3][]float64{args[0], args[1], args[2]}, [3][]float64{args[0], args[1], args[2]}
					p[pos] = append([]float64(nil), args[pos]...)
					m[pos] = append([]float64(nil), args[pos]...)
					p[pos][k] += h
					m[pos][k] -= h
					fd := (c.eval(p[0], p[1], p[2]) - c.eval(m[0], m[1], m[2])) / (2 * h)
					if d := math.Abs(fd - g[k]); d > 1e-6*(1+math.Abs(fd)) {
						t.Fatalf("%s pos=%d comp=%d: analytic %v vs finite-difference %v", c.name, pos, k, g[k], fd)
					}
				}
			}
		}
	}
}

// Alternation: every arm's energy is symmetric under any permutation of its
// three arguments (its square is alternating), checked through the gradient.
func TestScalarEnergySymmetric(t *testing.T) {
	rng := newRNG(9205)
	psi := mustS2(t, 9205).Form
	for _, en := range []R3Energy{NewScalarEnergy(true, nil), NewScalarEnergy(false, Phi()), NewScalarEnergy(true, psi)} {
		for trial := 0; trial < 20; trial++ {
			a, b, c := testImag(rng), testImag(rng), testImag(rng)
			g1 := en.Grad(a, b, c, 2)
			g2 := en.Grad(b, a, c, 2) // swap the two fixed arguments
			g3 := en.Grad(a, c, b, 1) // same gradient w.r.t. b-position arg moved
			_ = g3
			for k := 1; k < 8; k++ {
				if math.Abs(g1[k]-g2[k]) > 1e-9*(1+math.Abs(g1[k])) {
					t.Fatalf("energy not symmetric in its two fixed arguments")
				}
			}
			// gradient wrt c with (a,b) fixed equals gradient wrt the middle slot of (a,c,b).
			h := en.Grad(a, c, b, 1)
			for k := 1; k < 8; k++ {
				if math.Abs(g1[k]-h[k]) > 1e-9*(1+math.Abs(g1[k])) {
					t.Fatalf("energy not symmetric under slot permutation")
				}
			}
		}
	}
}

func TestBPhiDefinite(t *testing.T) {
	kind, ev := ClassifyPsi(Phi())
	if kind != "definite" {
		t.Fatalf("B_φ classified %q, want definite (eig %v)", kind, ev)
	}
	for _, e := range ev {
		if math.Abs(math.Abs(e)-144) > 1e-6 {
			t.Fatalf("B_φ eigenvalue magnitude %v, want 144 (B_φ = ±144·I)", e)
		}
		if math.Signbit(e) != math.Signbit(ev[0]) {
			t.Fatalf("B_φ eigenvalues of mixed sign: %v", ev)
		}
	}
}

// φ + 0.05·noise (Frobenius-matched) is still in the φ orbit.
func TestBPhiPerturbedDefinite(t *testing.T) {
	rng := newRNG(9206)
	for n := 0; n < 100; n++ {
		noise := gaussianForm(rng)
		var c [7][7][7]float64
		phi := Phi()
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				for k := 0; k < 7; k++ {
					c[i][j][k] = phi.C[i][j][k] + 0.05*noise.C[i][j][k]
				}
			}
		}
		f := NewForm3(c)
		f.Rescale(42)
		if kind, ev := ClassifyPsi(f); kind != "definite" {
			t.Fatalf("φ+0.05·noise classified %q (eig %v)", kind, ev)
		}
	}
}

// S1 split draws are only signature (3,4) or (4,3); Frobenius-matched; the draw count is recorded.
func TestS1SplitSignatures(t *testing.T) {
	for seed := uint64(9301); seed < 9341; seed += 2 { // odd seeds -> split
		d, ok := SampleS1(seed, "split")
		if !ok {
			t.Fatalf("S1 split sampler missed on seed %d", seed)
		}
		if d.Draws < 1 {
			t.Fatalf("S1 draw count %d", d.Draws)
		}
		if got := d.Form.FrobSq(); math.Abs(got-42) > 1e-10 {
			t.Fatalf("S1 ψ Frobenius² = %v", got)
		}
		pos, neg := Signature(d.Form)
		if !((pos == 3 && neg == 4) || (pos == 4 && neg == 3)) {
			t.Fatalf("S1 split draw has signature (%d,%d)", pos, neg)
		}
	}
}

func TestS1DefiniteIsDefinite(t *testing.T) {
	d, ok := SampleS1(9302, "definite")
	if !ok {
		t.Skip("S1 definite sampler hit the cap on this seed (probability ~e^-40); investigate")
	}
	if kind, _ := ClassifyPsi(d.Form); kind != "definite" {
		t.Fatalf("S1 'definite' draw classified %q", kind)
	}
	if d.Draws < 2 {
		t.Logf("S1 definite draw count %d (rare but allowed)", d.Draws)
	}
}

func TestS1Deterministic(t *testing.T) {
	a, _ := SampleS1(9303, "split")
	b, _ := SampleS1(9303, "split")
	if a.Draws != b.Draws || a.Form.C != b.Form.C {
		t.Fatal("S1 is not deterministic for a fixed seed")
	}
}

// S2: always definite, S traceless, s in [0.05,0.25], Frobenius-matched,
// recorded distance and condition number consistent with the draw.
func TestS2Properties(t *testing.T) {
	for seed := uint64(9400); seed < 9440; seed += 2 {
		d := mustS2(t, seed)
		if d.S < 0.05 || d.S >= 0.25 {
			t.Fatalf("s = %v outside [0.05, 0.25)", d.S)
		}
		if math.Abs(d.TraceS) > 1e-12 {
			t.Fatalf("S not traceless: tr = %v", d.TraceS)
		}
		if got := d.Form.FrobSq(); math.Abs(got-42) > 1e-10 {
			t.Fatalf("S2 ψ Frobenius² = %v", got)
		}
		if kind, ev := ClassifyPsi(d.Form); kind != "definite" {
			t.Fatalf("S2 draw classified %q on seed %d (eig %v)", kind, seed, ev)
		}
		if d.Dist <= 0 || d.Dist > 1.2 {
			t.Fatalf("S2 distance %v out of the disclosed range", d.Dist)
		}
		// recompute the distance independently
		var s2 float64
		phi := Phi()
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				for k := 0; k < 7; k++ {
					x := d.Form.C[i][j][k] - phi.C[i][j][k]
					s2 += x * x
				}
			}
		}
		if math.Abs(math.Sqrt(s2/42)-d.Dist) > 1e-12 {
			t.Fatalf("recorded distance %v != recomputed %v", d.Dist, math.Sqrt(s2/42))
		}
		if d.Cond < 1 {
			t.Fatalf("condition number %v < 1", d.Cond)
		}
	}
}

func TestS2Deterministic(t *testing.T) {
	a, b := mustS2(t, 9450), mustS2(t, 9450)
	if a.Form.C != b.Form.C || a.S != b.S {
		t.Fatal("S2 is not deterministic for a fixed seed")
	}
}

// S2 must not use the S1 or rand3 stream constants.
func TestStreamConstantsDistinct(t *testing.T) {
	cs := map[uint64]string{0xa5a5a5a5: "round-2 rand3", 0x5a5a5a5a: "round-2 shuffle"}
	for name, c := range map[string]uint64{"r3 rand3": r3SeedRand3, "r3 S1": r3SeedPsiS1, "r3 S2": r3SeedPsiS2} {
		if prev, dup := cs[c]; dup {
			t.Fatalf("stream constant for %s collides with %s", name, prev)
		}
		cs[c] = name
	}
}

func TestStratumByParity(t *testing.T) {
	if R3Stratum(3000) != "definite" || R3Stratum(3001) != "split" || R3Stratum(3199) != "split" {
		t.Fatal("parity -> stratum assignment is wrong")
	}
	nd, ns := 0, 0
	for s := uint64(3000); s <= 3199; s++ {
		if R3Stratum(s) == "definite" {
			nd++
		} else {
			ns++
		}
	}
	if nd != 100 || ns != 100 {
		t.Fatalf("strata over 3000–3199 are %d definite / %d split, want 100/100", nd, ns)
	}
}

func mustS2(t *testing.T, seed uint64) S2Draw {
	t.Helper()
	return SampleS2(seed)
}
