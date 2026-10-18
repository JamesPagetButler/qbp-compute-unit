package mdcu

import (
	"math"
	"testing"
)

// Red-team follow-ups (studybuddy mdcu-leaf-foliation-redteam, 2026-10-17).

// GradGram is ∇V from V = 4·GramDet(a,c): zero OMUL.
// ∇_a = 8(N(c)a − ⟨a,c⟩c), ∇_b(imag) = 8(N(a)c − ⟨a,c⟩a), real parts 0.
func GradGram(s *[16]float64) (g [16]float64) {
	var na, nc, ac float64
	for k := 1; k < 8; k++ {
		na += s[k] * s[k]
		nc += s[8+k] * s[8+k]
		ac += s[k] * s[8+k]
	}
	for k := 1; k < 8; k++ {
		g[k] = 8 * (nc*s[k] - ac*s[8+k])
		g[8+k] = 8 * (na*s[8+k] - ac*s[k])
	}
	return
}

func randSO7(r interface{ NormFloat64() float64 }) (R [8][8]float64) {
	// Gram-Schmidt a random 7×7 Gaussian matrix; fix e0.
	var M [7][7]float64
	for i := range M {
		for j := range M[i] {
			M[i][j] = r.NormFloat64()
		}
		for p := 0; p < i; p++ {
			d := 0.0
			for j := range M[i] {
				d += M[i][j] * M[p][j]
			}
			for j := range M[i] {
				M[i][j] -= d * M[p][j]
			}
		}
		n := math.Sqrt(norm2(M[i][:]))
		for j := range M[i] {
			M[i][j] /= n
		}
	}
	R[0][0] = 1
	for i := 0; i < 7; i++ {
		for j := 0; j < 7; j++ {
			R[i+1][j+1] = M[i][j]
		}
	}
	return
}

func rot8(R *[8][8]float64, x []float64) []float64 {
	z := make([]float64, 8)
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			z[i] += R[i][j] * x[j]
		}
	}
	return z
}

func rot16(R *[8][8]float64, s [16]float64) (t [16]float64) {
	copy(t[:8], rot8(R, s[:8]))
	copy(t[8:], rot8(R, s[8:]))
	return
}

// RT1: the Gram gradient equals the octonionic gradient on the flow domain.
func TestRT1_GramGradient(t *testing.T) {
	r := rng()
	worst := 0.0
	for n := 0; n < 20000; n++ {
		var s [16]float64
		copy(s[:], randVec(r, 16))
		s[0] = 0
		renorm16(&s, nil)
		a, b := GradCDMAC(&s, nil), GradGram(&s)
		worst = math.Max(worst, norm(sub(a[:], b[:])))
	}
	t.Logf("max ‖g_cdmac − g_gram‖ = %.3g (Gram path: 0 OMUL)", worst)
	if worst > 1e-14 {
		t.Fatal("Gram gradient mismatch")
	}
}

// RT2: the full 16D RK4 trajectory is equivariant under a generic SO(7)
// (non-G₂) rotation: flow(R s) = R flow(s). Stronger than V-invariance.
func TestRT2_TrajectoryO7Equivariance(t *testing.T) {
	r := rng()
	R := randSO7(r)
	// confirm R is NOT an automorphism
	x, y := randVec(r, 8), randVec(r, 8)
	autDef := norm(sub(rot8(&R, cdMul(x, y)), cdMul(rot8(&R, x), rot8(&R, y))))
	worst := 0.0
	for _, s := range rf688(t)[:50] {
		a, b := s, rot16(&R, s)
		for st := 0; st < 400; st++ {
			a = RK4Step16(a, 0.02, GradCDMAC, nil)
			b = RK4Step16(b, 0.02, GradCDMAC, nil)
		}
		ra := rot16(&R, a)
		worst = math.Max(worst, norm(sub(ra[:], b[:])))
	}
	t.Logf("generic SO(7): automorphism defect %.3g; trajectory equivariance err %.3g", autDef, worst)
	if autDef < 1e-3 || worst > 1e-12 {
		t.Fatal("RT2 failed")
	}
}

// RT3: contrast — the associator norm (used by Pillar 2 A/B) IS sensitive to
// octonion structure: invariant under the G₂ torus, not under generic SO(7).
func TestRT3_AssociatorIsG2Sensitive(t *testing.T) {
	r := rng()
	R := randSO7(r)
	var g2, so7, ref float64
	for n := 0; n < 2000; n++ {
		x, y, z := randVec(r, 8), randVec(r, 8), randVec(r, 8)
		x[0], y[0], z[0] = 0, 0, 0
		a0 := norm2(assoc8(x, y, z))
		t1, t2 := r.Float64()*6, r.Float64()*6
		ag := norm2(assoc8(g2Torus(x, t1, t2), g2Torus(y, t1, t2), g2Torus(z, t1, t2)))
		as := norm2(assoc8(rot8(&R, x), rot8(&R, y), rot8(&R, z)))
		g2 = math.Max(g2, math.Abs(ag-a0))
		so7 = math.Max(so7, math.Abs(as-a0))
		ref = math.Max(ref, a0)
	}
	t.Logf("‖[x,y,z]‖² change: G₂ torus %.3g, generic SO(7) %.3g (scale %.3g)", g2, so7, ref)
	if g2 > 1e-10 || so7 < 1e-3 {
		t.Fatal("RT3 failed")
	}
}

func BenchmarkGradGram(b *testing.B) {
	s := rf688(b)[0]
	for i := 0; i < b.N; i++ {
		_ = GradGram(&s)
	}
}

func BenchmarkQuenchGram(b *testing.B) {
	seeds := rf688(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range seeds {
			for t := 0; t < 400; t++ {
				s = RK4Step16(s, 0.02, func(x *[16]float64, _ *Counter) [16]float64 { return GradGram(x) }, nil)
			}
		}
	}
}
