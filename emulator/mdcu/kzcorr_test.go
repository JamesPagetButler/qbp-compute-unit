package mdcu

// Tests for the secondary observable (frozen §3): the plane-overlap correlation length
// ξ(τ_Q). Non-gating; the definition is a HARNESS choice documented in kzcorr.go.

import (
	"math"
	"testing"
)

func blockField(L, block int, seed uint64) [][3][7]float64 {
	r := kzTestRand(seed)
	nb := (L + block - 1) / block
	planes := haarG2Set(r, nb*nb*nb)
	f := make([][3][7]float64, L*L*L)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				f[kzSite(L, i, j, k)] = planes[((i/block)*nb+j/block)*nb+k/block]
			}
		}
	}
	return f
}

func TestKZCorrelationLength(t *testing.T) {
	L := 12
	al := make([][3][7]float64, L*L*L)
	p := haarG2Plane(kzTestRand(12600))
	for i := range al {
		al[i] = p
	}
	xi, capped := KZPlaneCorrelationLength(L, al)
	if !capped || xi != float64(L/2) {
		t.Fatalf("aligned field: xi = %v capped = %v, want %d capped", xi, capped, L/2)
	}
	haar := haarG2Set(kzTestRand(12601), L*L*L)
	xh, ch := KZPlaneCorrelationLength(L, haar)
	if ch || xh >= 1 || xh <= 0 {
		t.Fatalf("i.i.d. Haar: xi = %v capped = %v, want in (0, 1)", xh, ch)
	}
	x2, _ := KZPlaneCorrelationLength(L, blockField(L, 2, 12602))
	x4, _ := KZPlaneCorrelationLength(L, blockField(L, 4, 12603))
	if !(xh < x2 && x2 < x4) {
		t.Fatalf("correlation length must grow with domain size: haar %v, 2-blocks %v, 4-blocks %v", xh, x2, x4)
	}
	if math.IsNaN(x2) || math.IsNaN(x4) {
		t.Fatalf("NaN")
	}
}

// Invariance: a global orthogonal rotation of R^7 leaves xi unchanged.
func TestKZCorrelationLengthRotationInvariant(t *testing.T) {
	L := 8
	f := blockField(L, 2, 12610)
	a, _ := KZPlaneCorrelationLength(L, f)
	// rotate by a fixed orthogonal map: a signed permutation of the axes
	perm := [7]int{3, 0, 6, 1, 5, 2, 4}
	sign := [7]float64{1, -1, 1, 1, -1, 1, -1}
	g := make([][3][7]float64, len(f))
	for s := range f {
		for r := 0; r < 3; r++ {
			for k := 0; k < 7; k++ {
				g[s][r][perm[k]] = sign[k] * f[s][r][k]
			}
		}
	}
	b, _ := KZPlaneCorrelationLength(L, g)
	if math.Abs(a-b) > 1e-9 {
		t.Fatalf("xi not invariant under a global rotation: %v vs %v", a, b)
	}
}
