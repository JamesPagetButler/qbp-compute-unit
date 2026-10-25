package mdcu

// Round-3 tests for the signed-energy ring runner (frozen §1.1, §5; written
// before r3ring.go). Out-of-grid seeds only.

import (
	"math"
	"testing"
)

func bitsEqual(a, b [][]float64) (bool, int, int) {
	if len(a) != len(b) {
		return false, -1, -1
	}
	for i := range a {
		for k := range a[i] {
			if math.Float64bits(a[i][k]) != math.Float64bits(b[i][k]) {
				return false, i, k
			}
		}
	}
	return true, 0, 0
}

// THE GATE: K-full through the new runner is bit-identical to round-2 RunRingGen.
func TestRingKFullBitIdenticalToRunRingGen(t *testing.T) {
	for _, n := range []int{12, 24, 48} {
		for _, seed := range []uint64{9601, 9602, 9603} {
			z0 := Q1Init(n, 0.1, newRNG(seed))
			want := RunRingGen(z0, LayOverlap, assoc8)
			got := RunRingEnergy(z0, R3EnergyFor(ArmFull, nil, nil), 250)
			if ok, i, k := bitsEqual(want, got); !ok {
				t.Fatalf("K-full differs from RunRingGen (N=%d seed=%d) at node %d component %d", n, seed, i, k)
			}
		}
	}
}

// Same gate for the carried reference arm (round-2 null 3 with the r3 stream constant).
func TestRingKRand3BitIdenticalToRunRingGen(t *testing.T) {
	for _, seed := range []uint64{9611, 9612} {
		z0 := Q1Init(24, 0.1, newRNG(seed))
		tt := NewRandomAlt(newRNG(seed ^ r3SeedRand3))
		want := RunRingGen(z0, LayOverlap, tt.Apply)
		got := RunRingEnergy(z0, R3EnergyFor(ArmRand3, nil, tt), 250)
		if ok, i, k := bitsEqual(want, got); !ok {
			t.Fatalf("K-rand3 differs from RunRingGen (seed=%d) at node %d component %d", seed, i, k)
		}
	}
}

// K-none (μ term only) equals RunRingGen with LayNone.
func TestRingKNoneBitIdentical(t *testing.T) {
	z0 := Q1Init(24, 0.1, newRNG(9621))
	want := RunRingGen(z0, LayNone, assoc8)
	got := RunRingEnergy(z0, nil, 250)
	if ok, i, k := bitsEqual(want, got); !ok {
		t.Fatalf("K-none differs from RunRingGen(LayNone) at node %d component %d", i, k)
	}
}

func TestRingStepsParameter(t *testing.T) {
	z0 := Q1Init(12, 0.1, newRNG(9631))
	a := RunRingEnergy(z0, R3EnergyFor(ArmFull, nil, nil), 250)
	b := RunRingEnergy(z0, R3EnergyFor(ArmFull, nil, nil), 1000)
	if ok, _, _ := bitsEqual(a, b); ok {
		t.Fatal("steps=1000 gave the same configuration as steps=250")
	}
	c := RunRingEnergy(z0, R3EnergyFor(ArmFull, nil, nil), 0)
	if ok, _, _ := bitsEqual(z0, c); !ok {
		t.Fatal("steps=0 must return the initial configuration")
	}
}

func TestRingArmsKeepUnitImaginary(t *testing.T) {
	psi := SampleS2(9641).Form
	z0 := Q1Init(24, 0.1, newRNG(9641))
	rand3 := NewRandomAlt(newRNG(9641 ^ r3SeedRand3))
	for _, arm := range R3AllArms {
		en := R3EnergyFor(arm, psi, rand3)
		Z := RunRingEnergy(z0, en, 250)
		for i, z := range Z {
			if z[0] != 0 {
				t.Fatalf("%s: node %d has a real part %v", arm, i, z[0])
			}
			if math.Abs(dot(z, z)-1) > 1e-12 {
				t.Fatalf("%s: node %d has norm² %v", arm, i, dot(z, z))
			}
		}
	}
}

func TestRingDeterministic(t *testing.T) {
	psi := SampleS2(9651).Form
	z0 := Q1Init(12, 0.1, newRNG(9651))
	en := func() R3Energy { return R3EnergyFor(ArmPsiGram, psi, nil) }
	a := RunRingEnergy(z0, en(), 250)
	b := RunRingEnergy(z0, en(), 250)
	if ok, _, _ := bitsEqual(a, b); !ok {
		t.Fatal("ring runner is not deterministic")
	}
}

func TestRingDoesNotMutateInput(t *testing.T) {
	z0 := Q1Init(12, 0.1, newRNG(9661))
	cp := make([][]float64, len(z0))
	for i := range z0 {
		cp[i] = append([]float64(nil), z0[i]...)
	}
	RunRingEnergy(z0, R3EnergyFor(ArmGram, nil, nil), 250)
	if ok, _, _ := bitsEqual(z0, cp); !ok {
		t.Fatal("RunRingEnergy mutated its input")
	}
}

// Every arm's energy is minimised, not maximised: after the run the arm's own
// energy per triple has not increased relative to the start (sanity check of
// gradient sign, K-full and K-gram on N=24).
func TestRingArmsDescend(t *testing.T) {
	z0 := Q1Init(24, 0.1, newRNG(9671))
	meanE := func(Z [][]float64, e func(a, b, c []float64) float64) float64 {
		n := len(Z)
		s := 0.0
		for i := 0; i < n; i++ {
			s += e(Z[(i-1+n)%n], Z[i], Z[(i+1)%n])
		}
		return s / float64(n)
	}
	for _, c := range []struct {
		arm R3Arm
		e   func(a, b, c []float64) float64
	}{
		{ArmFull, func(a, b, c []float64) float64 { return norm2(assoc8(a, b, c)) }},
		{ArmGram, func(a, b, c []float64) float64 { return 4 * Gram3(a, b, c) }},
	} {
		Z := RunRingEnergy(z0, R3EnergyFor(c.arm, nil, nil), 250)
		if before, after := meanE(z0, c.e), meanE(Z, c.e); !(after < before) {
			t.Fatalf("%s: mean triple energy did not decrease (%v -> %v)", c.arm, before, after)
		}
	}
}
