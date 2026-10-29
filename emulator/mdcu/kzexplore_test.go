package mdcu

// Exploration only (skipped unless KZ_EXPLORE=1; not a gate): how the B null behaves on fields of increasing
// roughness, on FREE seeds, to inform the calibration-ensemble question. Run with
// KZ_EXPLORE=1 go test ./mdcu -run TestKZExplore -v

import (
	"fmt"
	"math"
	"os"
	"testing"
)

func kzSmoothQ(L int, A float64, seed uint64) [][4]float64 {
	rng := kzTestRand(seed)
	type mode struct {
		k   [3]float64
		phi [3]float64
	}
	var modes []mode
	for _, k := range [][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 0}, {0, 1, 1}, {1, 0, 1}} {
		modes = append(modes, mode{k, [3]float64{rng.Float64() * 2 * math.Pi, rng.Float64() * 2 * math.Pi, rng.Float64() * 2 * math.Pi}})
	}
	q := make([][4]float64, L*L*L)
	for i := 0; i < L; i++ {
		for j := 0; j < L; j++ {
			for k := 0; k < L; k++ {
				var rv [3]float64
				for _, m := range modes {
					ph := 2 * math.Pi * (m.k[0]*float64(i) + m.k[1]*float64(j) + m.k[2]*float64(k)) / float64(L)
					for c := 0; c < 3; c++ {
						rv[c] += A * math.Cos(ph+m.phi[c]) / float64(len(modes))
					}
				}
				n := math.Sqrt(rv[0]*rv[0] + rv[1]*rv[1] + rv[2]*rv[2])
				if n < 1e-15 {
					q[kzSiteIdx(L, i, j, k)] = [4]float64{1, 0, 0, 0}
					continue
				}
				h := n / 2
				q[kzSiteIdx(L, i, j, k)] = [4]float64{math.Cos(h), math.Sin(h) * rv[0] / n, math.Sin(h) * rv[1] / n, math.Sin(h) * rv[2] / n}
			}
		}
	}
	return q
}

func TestKZExplore(t *testing.T) {
	if os.Getenv("KZ_EXPLORE") == "" {
		t.Skip()
	}
	p := Amend1Z2Params()
	for _, L := range []int{16} {
		fmt.Printf("L=%d  pooled over 4 free fields; columns: sigma, ambiguous frac, raw -1 cells, defects (non-ambiguous -1)\n", L)
		for _, A := range []float64{0, 4, 8} {
			for _, sg := range []float64{0.4, 0.8, 1.2, 1.6, 2.4, 3.2} {
				cells, amb, neg, def := 0, 0, 0, 0
				for s := uint64(0); s < 4; s++ {
					q := kzSmoothQ(L, A, 13300+s)
					r := KZDetectS3Tilted(L, q, sg, 13310+s, p)
					cells += r.Cells
					amb += r.Ambiguous
					neg += r.NegCells
					def += r.Defects
				}
				fmt.Printf("  A=%-4v sigma=%-4v  amb=%.4f  raw-1=%-5d defects=%d\n", A, sg, float64(amb)/float64(cells), neg, def)
			}
		}
	}
}
