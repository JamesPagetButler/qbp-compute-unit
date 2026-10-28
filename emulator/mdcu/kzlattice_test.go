package mdcu

// mdcu-kz lattice, dynamics, order-parameter and seed-guard tests (frozen mdcu-kz
// prereg §2a, §4, §5, §7). Written BEFORE the code (tests first). Seeds 12000+ only
// (outside every frozen or used kz/R3/R4 range).

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestKZOctonionTableMatchesMulRef(t *testing.T) {
	r := kzTestRand(12100)
	for n := 0; n < 50; n++ {
		var u, v [7]float64
		for i := 0; i < 7; i++ {
			u[i], v[i] = r.NormFloat64(), r.NormFloat64()
		}
		x8, y8 := make([]float64, 8), make([]float64, 8)
		copy(x8[1:], u[:])
		copy(y8[1:], v[:])
		want := MulRef(x8, y8)
		got := kzOctIm(u, v)
		for k := 0; k < 7; k++ {
			if math.Abs(got[k]-want[k+1]) > 1e-12 {
				t.Fatalf("component %d: %v vs MulRef %v", k, got[k], want[k+1])
			}
		}
	}
}

func TestKZHaarStartIsAssociativeFrame(t *testing.T) {
	f := NewKZPlaneFieldHaar(6, 12110)
	fr := f.Frames()
	for s := range fr {
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				d := 0.0
				for k := 0; k < 7; k++ {
					d += fr[s][i][k] * fr[s][j][k]
				}
				want := 0.0
				if i == j {
					want = 1
				}
				if math.Abs(d-want) > 1e-12 {
					t.Fatalf("site %d frame not orthonormal: <%d,%d> = %v", s, i, j, d)
				}
			}
		}
	}
	// same seed -> same field; a different seed -> a different field
	g := NewKZPlaneFieldHaar(6, 12110)
	h := NewKZPlaneFieldHaar(6, 12111)
	if f.U[5] != g.U[5] || f.U[5] == h.U[5] {
		t.Fatalf("seed does not fix the Haar start state as required")
	}
}

func TestKZGradientMatchesFiniteDifference(t *testing.T) {
	f := NewKZPlaneFieldHaar(4, 12120)
	gu, gv := f.Gradient()
	const h = 1e-6
	for _, s := range []int{0, 7, 33, 63} {
		for k := 0; k < 7; k++ {
			f.U[s][k] += h
			ep := f.Energy()
			f.U[s][k] -= 2 * h
			em := f.Energy()
			f.U[s][k] += h
			if fd := (ep - em) / (2 * h); math.Abs(fd-gu[s][k]) > 1e-6*(1+math.Abs(fd)) {
				t.Fatalf("dE/du[%d][%d]: analytic %v vs finite difference %v", s, k, gu[s][k], fd)
			}
			f.V[s][k] += h
			ep = f.Energy()
			f.V[s][k] -= 2 * h
			em = f.Energy()
			f.V[s][k] += h
			if fd := (ep - em) / (2 * h); math.Abs(fd-gv[s][k]) > 1e-6*(1+math.Abs(fd)) {
				t.Fatalf("dE/dv[%d][%d]: analytic %v vs finite difference %v", s, k, gv[s][k], fd)
			}
		}
	}
}

func TestKZEnergyBounds(t *testing.T) {
	f := NewKZPlaneFieldHaar(5, 12130)
	if e := f.Energy(); e < 0 || e > 3*3*125 {
		t.Fatalf("energy %v outside [0, 3·bonds]", e)
	}
	al := NewKZPlaneFieldAligned(5, 12130)
	if e := al.Energy(); math.Abs(e) > 1e-9 {
		t.Fatalf("aligned field must have zero energy, got %v", e)
	}
}

func TestKZStepKeepsStiefelPair(t *testing.T) {
	f := NewKZPlaneFieldHaar(5, 12140)
	rng := KZNoiseRNG(12140, 25)
	for n := 0; n < 40; n++ {
		f.Step(0.02, 0.8, rng)
	}
	for s := range f.U {
		uu, vv, uv := 0.0, 0.0, 0.0
		for k := 0; k < 7; k++ {
			uu += f.U[s][k] * f.U[s][k]
			vv += f.V[s][k] * f.V[s][k]
			uv += f.U[s][k] * f.V[s][k]
		}
		if math.Abs(uu-1) > 1e-12 || math.Abs(vv-1) > 1e-12 || math.Abs(uv) > 1e-12 {
			t.Fatalf("site %d left the Stiefel manifold: %v %v %v", s, uu, vv, uv)
		}
	}
}

func TestKZT0DescentNonIncreasing(t *testing.T) {
	f := NewKZPlaneFieldHaar(6, 12150)
	rng := KZNoiseRNG(12150, 0)
	prev := f.Energy()
	for n := 0; n < 200; n++ {
		f.Step(0.01, 0, rng)
		e := f.Energy()
		if e > prev+1e-9*math.Abs(prev) {
			t.Fatalf("step %d: energy rose %v -> %v at eta = 0.01", n, prev, e)
		}
		prev = e
	}
	if prev >= NewKZPlaneFieldHaar(6, 12150).Energy() {
		t.Fatalf("T = 0 relaxation did not lower the energy")
	}
}

func TestKZS3Dynamics(t *testing.T) {
	q := NewKZS3FieldHaar(6, 12160)
	rng := KZNoiseRNG(12160, 0)
	prev := q.Energy()
	for n := 0; n < 200; n++ {
		q.Step(0.02, 0, rng)
		e := q.Energy()
		if e > prev+1e-9*math.Abs(prev) {
			t.Fatalf("S3 step %d: energy rose %v -> %v", n, prev, e)
		}
		prev = e
	}
	for s := range q.Q {
		n := 0.0
		for c := 0; c < 4; c++ {
			n += q.Q[s][c] * q.Q[s][c]
		}
		if math.Abs(n-1) > 1e-12 {
			t.Fatalf("site %d not on S3: |q|² = %v", s, n)
		}
	}
	al := NewKZS3FieldAligned(5)
	if math.Abs(al.OrderM()-1) > 1e-12 || math.Abs(al.Energy()) > 1e-9 {
		t.Fatalf("aligned S3: m = %v, E = %v", al.OrderM(), al.Energy())
	}
}

func TestKZOrderParameters(t *testing.T) {
	al := NewKZPlaneFieldAligned(5, 12170)
	if s := al.OrderS(); math.Abs(s-1) > 1e-12 {
		t.Fatalf("aligned S = %v, want 1 (normalised by sqrt(12/7))", s)
	}
	h := NewKZPlaneFieldHaar(8, 12171)
	if s := h.OrderS(); s > 0.2 {
		t.Fatalf("Haar S = %v, want ~0", s)
	}
	hq := NewKZS3FieldHaar(8, 12172)
	if m := hq.OrderM(); m > 0.2 {
		t.Fatalf("Haar m = %v, want ~0", m)
	}
}

func TestKZOrderingGrowsAtLowT(t *testing.T) {
	f := NewKZPlaneFieldHaar(6, 12180)
	s0 := f.OrderS()
	rng := KZNoiseRNG(12180, 0)
	for n := 0; n < 600; n++ {
		f.Step(0.02, 0.05, rng)
	}
	if s1 := f.OrderS(); s1 <= s0+0.2 {
		t.Fatalf("no ordering at T = 0.05: S %v -> %v", s0, s1)
	}
	q := NewKZS3FieldHaar(6, 12181)
	m0 := q.OrderM()
	for n := 0; n < 600; n++ {
		q.Step(0.05, 0.05, rng)
	}
	if m1 := q.OrderM(); m1 <= m0+0.2 {
		t.Fatalf("S3 does not order at low T: m %v -> %v", m0, m1)
	}
}

func TestKZNoiseStreamsAreDeterministicAndDistinct(t *testing.T) {
	a, b := KZNoiseRNG(12190, 25), KZNoiseRNG(12190, 25)
	c := KZNoiseRNG(12190, 50)
	if a.NormFloat64() != b.NormFloat64() {
		t.Fatalf("same (seed, stream) must give the same noise")
	}
	if KZNoiseRNG(12190, 25).NormFloat64() == c.NormFloat64() {
		t.Fatalf("different streams must give different noise")
	}
}

func TestKZEtaRule(t *testing.T) {
	eta, ok := KZChooseEta(6, []uint64{12200, 12201, 12202})
	if !ok {
		t.Fatalf("no eta in the ladder passes the descent test")
	}
	found := false
	for _, e := range KZEtaLadder {
		if e == eta {
			found = true
		}
	}
	if !found {
		t.Fatalf("eta %v is not in the ladder", eta)
	}
	// the chosen eta is the largest passing one: every larger eta must fail on some seed
	for _, e := range KZEtaLadder {
		if e > eta {
			pass := true
			for _, sd := range []uint64{12200, 12201, 12202} {
				if !kzDescentOK(6, sd, e, true) || !kzDescentOK(6, sd, e, false) {
					pass = false
				}
			}
			if pass {
				t.Fatalf("eta %v is larger than the chosen %v and also passes", e, eta)
			}
		}
	}
}

func TestKZSeedGuard(t *testing.T) {
	cases := []struct {
		seed uint64
		want KZSeedClass
	}{
		{5000, KZSeedMain}, {5199, KZSeedMain},
		{5200, KZSeedNull}, {5399, KZSeedNull},
		{5400, KZSeedReserved}, {5599, KZSeedReserved},
		{5600, KZSeedStage0}, {5629, KZSeedStage0},
		{9930, KZSeedProbe}, {9949, KZSeedProbe},
		{3000, KZSeedUsed}, {3199, KZSeedUsed}, {4000, KZSeedUsed}, {4399, KZSeedUsed},
		{9960, KZSeedUsed}, {9979, KZSeedUsed}, {9100, KZSeedUsed}, {9929, KZSeedUsed}, {9950, KZSeedUsed},
		{5630, KZSeedSaturation}, {5649, KZSeedSaturation}, // amend-1 §4 (arch seq 212)
		{12000, KZSeedFree}, {1, KZSeedFree}, {5650, KZSeedFree}, {9980, KZSeedFree},
	}
	for _, c := range cases {
		if got := KZClassifySeed(c.seed); got != c.want {
			t.Errorf("seed %d: class %v, want %v", c.seed, got, c.want)
		}
	}
	if err := KZCheckSmokeSeeds([]uint64{12000, 12001}); err != nil {
		t.Errorf("free seeds refused: %v", err)
	}
	for _, bad := range [][]uint64{{5000}, {5200}, {5400}, {5600}, {5630}, {5649}, {9930}, {3000}, {4000}, {9960}, {12000, 5100}} {
		if err := KZCheckSmokeSeeds(bad); err == nil {
			t.Errorf("smoke seeds %v were accepted", bad)
		}
	}
}

// keep the rand import used even if tests change
var _ = rand.NewPCG
