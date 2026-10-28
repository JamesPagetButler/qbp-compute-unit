package mdcu

// mdcu-kz secondary observable: the two-point plane-overlap correlation length ξ(τ_Q)
// (frozen §3 names it; it is NOT part of any decision rule). HARNESS definition:
//
//	G(r) = < tr(Π_x Π_{x+r e_a}) > - 9/7,   averaged over sites x and the three axes a,
//	C(r) = G(r) / G(0),   G(0) = 3 - 9/7 = 12/7,
//
// where tr(Π_x Π_y) = ||F_x F_y^T||_F² and 9/7 is its value for independent planes. ξ is
// the distance r in [0, L/2] at which C first falls below 1/e, interpolated linearly
// between lattice distances. If C never falls below 1/e within L/2, ξ = L/2 and the
// second return value reports that ξ is capped.

import "math"

func KZPlaneCorrelationLength(L int, f [][3][7]float64) (xi float64, capped bool) {
	half := L / 2
	C := make([]float64, half+1)
	C[0] = 1
	for r := 1; r <= half; r++ {
		sum, n := 0.0, 0
		for i := 0; i < L; i++ {
			for j := 0; j < L; j++ {
				for k := 0; k < L; k++ {
					a := kzSite(L, i, j, k)
					for _, b := range [3]int{kzSite(L, (i+r)%L, j, k), kzSite(L, i, (j+r)%L, k), kzSite(L, i, j, (k+r)%L)} {
						m := kzOverlap(&f[a], &f[b])
						t := 0.0
						for x := 0; x < 3; x++ {
							for y := 0; y < 3; y++ {
								t += m[x][y] * m[x][y]
							}
						}
						sum += t
						n++
					}
				}
			}
		}
		C[r] = (sum/float64(n) - 9.0/7.0) / (12.0 / 7.0)
	}
	thr := 1 / math.E
	for r := 1; r <= half; r++ {
		if C[r] < thr {
			return float64(r-1) + (C[r-1]-thr)/(C[r-1]-C[r]), false
		}
	}
	return float64(half), true
}
