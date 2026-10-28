package mdcu

// mdcu-kz saturation sanity check (amend-1 item 4): the raw -1 cell density over n
// i.i.d. Haar-G2 plane fields at the run's L must be 0.50 +- 0.05, else the grid is
// blocked. It is a code sanity check, not a physics reference. The raw density is the
// fraction of cells whose sign product is -1, ambiguous or not (NegCells / cells).

import (
	"math"
	"math/rand/v2"
)

// kzHaarStream is the PCG stream constant for Haar start states (same seed -> same field).
const kzHaarStream = 0x6b7a4861

func kzHaarRand(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, kzHaarStream)) }

const (
	KZSaturationCentre = 0.50
	KZSaturationTol    = 0.05
)

// KZSaturationRawDensity is the mean raw -1 density over n fields; field i is drawn
// from seed0+i with the same Haar generator as the main arm's start state.
func KZSaturationRawDensity(L, n int, seed0 uint64) float64 {
	N := L * L * L
	s := 0.0
	for i := 0; i < n; i++ {
		f := haarG2Set(kzHaarRand(seed0+uint64(i)), N)
		r := Z2DetectPlanes(L, f, Amend1Z2Params())
		s += float64(r.NegCells) / float64(N)
	}
	return s / float64(n)
}

// KZSaturationWithinTol reports whether d is inside 0.50 +- 0.05 (inclusive, with a
// 1e-12 float slack).
func KZSaturationWithinTol(d float64) bool {
	return math.Abs(d-KZSaturationCentre) <= KZSaturationTol+1e-12
}
