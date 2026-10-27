package mdcu

// R4 stage 1 tests (frozen R4 prereg §4 "Tests first"): level-scaled energies,
// comass rescale, phi_L comass, S2 at a fixed scale, arm tables.
//
// Seeds used here are out-of-grid and avoid the reserved R4 ranges
// (4000–4199 grid, 4200–4399 S-D, 9960–9969 timing probe): 9970–9979.

import (
	"math"
	"math/rand/v2"
	"testing"
)

func r4RandTriple(rng *rand.Rand) (x, y, z []float64) {
	mk := func() []float64 {
		v := make([]float64, 8)
		for k := 1; k < 8; k++ {
			v[k] = rng.NormFloat64()
		}
		return v
	}
	return mk(), mk(), mk()
}

func maxAbsDiff(a, b []float64) float64 {
	m := 0.0
	for i := range a {
		if d := math.Abs(a[i] - b[i]); d > m {
			m = d
		}
	}
	return m
}

func TestR4ScaleForm3(t *testing.T) {
	phi := Phi()
	before := phi.C
	s := ScaleForm3(phi, 1.88)
	if phi.C != before {
		t.Fatal("ScaleForm3 mutated its input")
	}
	rng := newRNG(9970)
	x, y, z := r4RandTriple(rng)
	if d := math.Abs(s.Eval(x, y, z) - 1.88*phi.Eval(x, y, z)); d > 1e-12 {
		t.Fatalf("scaled Eval off by %g", d)
	}
	if d := math.Abs(s.FrobSq() - 1.88*1.88*42); d > 1e-9 {
		t.Fatalf("scaled FrobSq off by %g", d)
	}
}

// phi_L = L·phi has comass exactly L (comass(phi) = 1).
func TestR4PhiLComass(t *testing.T) {
	for _, L := range []float64{1, R4Kappa, 0.5} {
		got := ComassEstimate(ScaleForm3(Phi(), L), newRNG(9971), 60, 300)
		if math.Abs(got-L)/L > 1e-6 {
			t.Fatalf("comass(phi_%g) = %.9f", L, got)
		}
	}
}

// A rescaled psi has comass L to the check tolerance; the two independent
// restart sets agree; rescaling is deterministic and does not mutate its input.
func TestR4RescaleToComass(t *testing.T) {
	for _, seed := range []uint64{9972, 9973} { // even = S2 (definite), odd = S1 (split)
		p := BuildR3Psi(seed)
		if p.Primary == nil {
			t.Fatalf("seed %d: sampler missed", seed)
		}
		in := p.Primary.C
		for _, L := range []float64{1, R4Kappa} {
			r := RescaleToComass(p.Primary, seed, L)
			if p.Primary.C != in {
				t.Fatalf("seed %d: RescaleToComass mutated its input", seed)
			}
			if !r.Converged || r.RelDiff > R4ComassTol {
				t.Fatalf("seed %d L=%g: two-set check failed: converged=%v relDiff=%g", seed, L, r.Converged, r.RelDiff)
			}
			// independent third estimate of the rescaled form's comass
			chk := ComassEstimate(r.Form, newRNG(seed^0x5eed), 100, 400)
			if math.Abs(chk-L)/L > 1e-5 {
				t.Fatalf("seed %d: rescaled comass %.9f, want %g", seed, chk, L)
			}
			r2 := RescaleToComass(p.Primary, seed, L)
			if r2.Form.C != r.Form.C || r2.Comass != r.Comass {
				t.Fatalf("seed %d L=%g: rescale is not deterministic", seed, L)
			}
		}
	}
}

// K-full = K-gram + K-phi at every level (Harvey–Lawson): the gradients agree
// to 1e-12 at level 1 and, with the WHOLE triple energy scaled by kappa^2, at level kappa.
func TestR4HarveyLawsonPartition(t *testing.T) {
	rng := newRNG(9974)
	k2 := R4Kappa * R4Kappa
	one := map[string]R3Energy{
		"full": R4EnergyFor(ArmFull, nil, nil, nil),
		"gram": R4EnergyFor(ArmGram, nil, nil, nil),
		"phi":  R4EnergyFor(ArmPhi, nil, nil, nil),
	}
	kap := map[string]R3Energy{
		"full": R4EnergyFor(ArmFullK, nil, nil, nil),
		"gram": R4EnergyFor(ArmGramK, nil, nil, nil),
		"phi":  R4EnergyFor(ArmPhiK, nil, nil, nil),
	}
	_ = k2
	for trial := 0; trial < 200; trial++ {
		x, y, z := r4RandTriple(rng)
		for pos := 0; pos < 3; pos++ {
			for name, set := range map[string]map[string]R3Energy{"level1": one, "levelK": kap} {
				gf := set["full"].Grad(x, y, z, pos)
				gs := set["gram"].Grad(x, y, z, pos)
				gp := set["phi"].Grad(x, y, z, pos)
				sum := make([]float64, 8)
				for k := range sum {
					sum[k] = gs[k] + gp[k]
				}
				scale := 1.0
				for _, v := range gf {
					scale = math.Max(scale, math.Abs(v))
				}
				if d := maxAbsDiff(gf, sum); d > 1e-12*scale {
					t.Fatalf("%s pos %d trial %d: K-full != K-gram + K-phi (diff %g, scale %g)", name, pos, trial, d, scale)
				}
			}
		}
	}
}

// Native arms at level 1 use exactly R3's energies (bit-identical gradients),
// so the bit-identity regression to the R3 raw can hold.
func TestR4NativeArmsAreR3Arms(t *testing.T) {
	p := BuildR3Psi(9972)
	rng := newRNG(9975)
	x, y, z := r4RandTriple(rng)
	pairs := []struct {
		r4, r3 R3Arm
	}{{ArmFull, ArmFull}, {ArmGram, ArmGram}, {ArmPhi, ArmPhi}, {ArmPsi, ArmPsi}}
	for _, pr := range pairs {
		e4 := R4EnergyFor(pr.r4, p.Primary, nil, nil)
		e3 := R3EnergyFor(pr.r3, p.Primary, nil)
		for pos := 0; pos < 3; pos++ {
			a, b := e4.Grad(x, y, z, pos), e3.Grad(x, y, z, pos)
			for k := range a {
				if a[k] != b[k] {
					t.Fatalf("%s pos %d component %d: R4 %v != R3 %v", pr.r4, pos, k, a[k], b[k])
				}
			}
		}
	}
	if R4EnergyFor(ArmNone, nil, nil, nil) != nil {
		t.Fatal("K-none must have a nil energy (mu term only)")
	}
}

// Every arm's energy is an alternating-form energy: invariant under argument
// permutations, so d/d(slot) commutes with swaps.
func TestR4AlternationOfEveryArm(t *testing.T) {
	p := BuildR3Psi(9972)
	psi1 := RescaleToComass(p.Primary, 9972, 1).Form
	psiK := RescaleToComass(p.Primary, 9972, R4Kappa).Form
	rng := newRNG(9976)
	for _, arm := range R4TierAArms {
		en := R4EnergyFor(arm, p.Primary, psi1, psiK)
		if en == nil {
			continue
		}
		for trial := 0; trial < 20; trial++ {
			x, y, z := r4RandTriple(rng)
			// E(x,y,z) = E(y,x,z):  dE/d(slot 1)(x,y,z) = dE/d(slot 0)(y,x,z)
			if d := maxAbsDiff(en.Grad(x, y, z, 1), en.Grad(y, x, z, 0)); d > 1e-10 {
				t.Fatalf("%s: swap(0,1) breaks symmetry by %g", arm, d)
			}
			// E(x,y,z) = E(x,z,y):  dE/d(slot 2)(x,y,z) = dE/d(slot 1)(x,z,y)
			if d := maxAbsDiff(en.Grad(x, y, z, 2), en.Grad(x, z, y, 1)); d > 1e-10 {
				t.Fatalf("%s: swap(1,2) breaks symmetry by %g", arm, d)
			}
		}
	}
}

// SampleS2At(seed, s) replays SampleS2's RNG order (it still consumes the scale
// draw, so the deformation matrix is the seed's own) with s fixed.
func TestR4SampleS2At(t *testing.T) {
	for _, seed := range []uint64{9972, 9974, 9976} {
		ref := SampleS2(seed)
		got := SampleS2At(seed, ref.S)
		if got.Form.C != ref.Form.C || got.S != ref.S || got.Dist != ref.Dist || got.Cond != ref.Cond || got.TraceS != ref.TraceS {
			t.Fatalf("seed %d: SampleS2At(seed, SampleS2.S) is not bit-identical to SampleS2(seed)", seed)
		}
	}
	// s -> 0 approaches phi; the ladder is monotone in distance from phi.
	d0 := SampleS2At(9972, 1e-9).Dist
	if d0 > 1e-6 {
		t.Fatalf("s=1e-9 should be ~phi, Dist=%g", d0)
	}
	prev := 0.0
	for _, s := range R4SDScales {
		d := SampleS2At(9972, s).Dist
		if d <= prev {
			t.Fatalf("S-D ladder not monotone in distance: s=%g Dist=%g after %g", s, d, prev)
		}
		prev = d
	}
	if SampleS2At(9972, 0.02).Dist == SampleS2At(9974, 0.02).Dist {
		t.Fatal("different seeds must give different deformations at the same s")
	}
}

func TestR4TierTables(t *testing.T) {
	if len(R4TierAArms) != 10 || len(R4TierBArms) != 4 || len(R4TierCArms) != 2 {
		t.Fatalf("tier sizes %d/%d/%d, want 10/4/2", len(R4TierAArms), len(R4TierBArms), len(R4TierCArms))
	}
	if R4StepsA != 250 || R4StepsB != 1000 || R4StepsC != 4000 {
		t.Fatal("step ladder must be 250/1000/4000")
	}
	if R4Kappa != 1.88 {
		t.Fatal("kappa must be the constant 1.88")
	}
	if R4GridLo != 4000 || R4GridHi != 4199 || R4SDLo != 4200 || R4SDHi != 4399 || R4ProbeLo != 9960 || R4ProbeHi != 9969 {
		t.Fatal("seed ranges differ from the frozen text")
	}
	if len(R4SDScales) != 4 || R4SDScales[0] != 0.005 || R4SDScales[3] != 0.05 || R4SDPerScale != 50 {
		t.Fatal("S-D ladder differs from the frozen text")
	}
	want := map[R3Arm]bool{ArmFull: true, ArmGram: true, ArmPhi: true, ArmPsi: true, ArmPsi1: true, ArmPhiK: true, ArmPsiK: true, ArmFullK: true, ArmGramK: true, ArmNone: true}
	for _, a := range R4TierAArms {
		if !want[a] {
			t.Fatalf("unexpected Tier A arm %s", a)
		}
		delete(want, a)
	}
	if len(want) != 0 {
		t.Fatalf("missing Tier A arms %v", want)
	}
}
