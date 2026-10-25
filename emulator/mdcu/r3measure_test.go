package mdcu

// Round-3 measurement tests (frozen §1.3; written before r3measure.go).
// Synthetic configurations with known answers, plus a consistency check against
// round 2's Q1Measure. Out-of-grid seeds only.

import (
	"math"
	"testing"
)

func unitVec(idx ...int) []float64 {
	v := make([]float64, 8)
	for _, i := range idx {
		v[i] = 1
	}
	normalize(v)
	return v
}

// 24 random unit vectors confined to span(e1,e2,e3): a genuine 3-plane lock.
func planeConfig(n int, seed uint64, dims int) [][]float64 {
	rng := newRNG(seed)
	Z := make([][]float64, n)
	for i := range Z {
		z := make([]float64, 8)
		for k := 1; k <= dims; k++ {
			z[k] = rng.NormFloat64()
		}
		normalize(z)
		Z[i] = z
	}
	return Z
}

func TestMeasureThreePlaneLock(t *testing.T) {
	Z := planeConfig(24, 9701, 3)
	m := R3Measure(Z, Phi())
	if m.S3 < 0.999 || m.L3Share < 0.15 || m.Class != "3-lock" {
		t.Fatalf("3-plane config: S3=%v L3Share=%v class=%q", m.S3, m.L3Share, m.Class)
	}
}

// Alternating 2-plane ring: s3 is ~1 but the third eigenvalue is ~0, so it is
// a 2-collapse, NOT a 3-lock (round-2 limitation 1; the reason λ₃ share exists).
func TestMeasureTwoPlaneCollapseIsNotThreeLock(t *testing.T) {
	Z := make([][]float64, 24)
	for i := range Z {
		if i%2 == 0 {
			Z[i] = unitVec(1)
		} else {
			Z[i] = unitVec(2)
		}
	}
	m := R3Measure(Z, Phi())
	if m.S3 < 0.999 {
		t.Fatalf("S3 = %v, expected ~1 for a 2-plane config", m.S3)
	}
	if m.L3Share > 1e-9 {
		t.Fatalf("L3Share = %v, expected ~0", m.L3Share)
	}
	if m.Class != "2-collapse" {
		t.Fatalf("class = %q, want 2-collapse", m.Class)
	}
}

func TestMeasureRandomIsDiffuse(t *testing.T) {
	Z := planeConfig(24, 9703, 7)
	m := R3Measure(Z, Phi())
	if m.Class != "diffuse" {
		t.Fatalf("random 7-dim config classified %q (S2=%v S3=%v L3Share=%v)", m.Class, m.S2, m.S3, m.L3Share)
	}
}

func TestMeasureEigenvaluesSortedAndConserved(t *testing.T) {
	Z := planeConfig(24, 9704, 7)
	m := R3Measure(Z, Phi())
	sum := 0.0
	for i, e := range m.Eig {
		sum += e
		if i > 0 && e > m.Eig[i-1]+1e-12 {
			t.Fatalf("eigenvalues not descending: %v", m.Eig)
		}
		if e < -1e-12 {
			t.Fatalf("negative Gram eigenvalue %v", e)
		}
	}
	if math.Abs(sum-24) > 1e-9 { // trace of the Gram of 24 unit imaginary vectors
		t.Fatalf("eigenvalue sum %v, want 24", sum)
	}
}

// S3 here must be the same quantity round 2 reported as SVD3.
func TestMeasureS3MatchesQ1Measure(t *testing.T) {
	for _, seed := range []uint64{9711, 9712, 9713} {
		z0 := Q1Init(24, 0.1, newRNG(seed))
		Z := RunRingEnergy(z0, R3EnergyFor(ArmFull, nil, nil), 250)
		m := R3Measure(Z, Phi())
		q := Q1Measure(Z)
		if math.Abs(m.S3-q.SVD3) > 1e-12 {
			t.Fatalf("S3 %v != Q1Measure.SVD3 %v", m.S3, q.SVD3)
		}
		if math.Abs(m.Phi0Frac-q.Phi0Frac) > 1e-15 || math.Abs(m.PlanePhi0-q.PlanePhi0) > 1e-12 {
			t.Fatalf("legacy calibration fields differ from Q1Measure")
		}
	}
}

// A plane that is exactly an associative 3-plane has |φ₀| = 1 for φ (own-form
// calibration) and a ψ₀ that depends on ψ (the informative cross-calibration).
func TestMeasureCalibrationAssociativePlane(t *testing.T) {
	l := Frame(1) // a Fano line: span(e_I,e_J,e_K) is associative
	Z := make([][]float64, 24)
	rng := newRNG(9721)
	for i := range Z {
		z := make([]float64, 8)
		z[l.I], z[l.J], z[l.K] = rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()
		normalize(z)
		Z[i] = z
	}
	m := R3Measure(Z, Phi())
	if math.Abs(m.PlanePhi0-1) > 1e-9 {
		t.Fatalf("|φ₀| of an associative plane = %v, want 1", m.PlanePhi0)
	}
	psi := SampleS2(9721).Form
	mp := R3Measure(Z, psi)
	if !(mp.PlanePsi0 >= 0 && mp.PlanePsi0 < 10) {
		t.Fatalf("|ψ₀| = %v looks wrong", mp.PlanePsi0)
	}
	if mp.PlanePsi0 == m.PlanePsi0 && psi.C != Phi().C {
		t.Fatal("ψ₀ should depend on ψ")
	}
}

func TestMeasureThresholdBoundaries(t *testing.T) {
	if c := r3Classify(0.95, 0.95, 0.15); c != "3-lock" {
		t.Fatalf("boundary s3=0.95 l3=0.15 -> %q, want 3-lock", c)
	}
	if c := r3Classify(0.90, 0.9499999, 0.15); c != "diffuse" {
		t.Fatalf("s3<0.95 -> %q, want diffuse", c)
	}
	if c := r3Classify(0.90, 0.9999, 0.1499999); c != "diffuse" {
		t.Fatalf("l3<0.15 and s2<0.95 -> %q, want diffuse", c)
	}
	if c := r3Classify(0.95, 0.99, 0.01); c != "2-collapse" {
		t.Fatalf("s2=0.95 -> %q, want 2-collapse", c)
	}
}
