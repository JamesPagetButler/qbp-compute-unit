package mdcu

// Exploration only (skipped unless KZ_EXPLORE=1; not a gate): how the B null behaves on fields of increasing
// roughness, on FREE seeds, to inform the calibration-ensemble question. Run with
// KZ_EXPLORE=1 go test ./mdcu -run TestKZExplore -v

import (
	"fmt"
	"os"
	"testing"
)

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
					q := KZSmoothField(L, A, 13300+s)
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
