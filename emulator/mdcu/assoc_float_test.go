package mdcu

import (
	"math"
	"testing"
)

// T3 follow-up (studybuddy via qbp-cu-implementor, seq 3080): QAssoc returns
// an exact zero for tier ≤ ℍ BY CONSTRUCTION. This test measures what the
// float arithmetic actually gives on quaternions, so the zero is not mistaken
// for evidence: the computed associator is ~eps, not 0, and 𝕆 is O(1).
func TestRT5_AssocFloatAtH(t *testing.T) {
	r := rng()
	var maxH, maxO float64
	nonzero := 0
	for n := 0; n < 20000; n++ {
		x, y, z := randVec(r, 4), randVec(r, 4), randVec(r, 4)
		l := cdMul(cdMul(x, y), z)
		rr := cdMul(x, cdMul(y, z))
		d := norm(sub(l, rr))
		if d != 0 {
			nonzero++
		}
		maxH = math.Max(maxH, d/(norm(x)*norm(y)*norm(z)))
		if q := QAssoc(TierH, x, y, z); norm(q) != 0 {
			t.Fatal("QAssoc(ℍ) must return the architectural exact zero")
		}
		xo, yo, zo := randVec(r, 8), randVec(r, 8), randVec(r, 8)
		maxO = math.Max(maxO, norm(QAssoc(TierO, xo, yo, zo))/(norm(xo)*norm(yo)*norm(zo)))
	}
	t.Logf("ℍ float associator, normwise: max %.3g (%.1f eps); nonzero in %d/20000 samples; 𝕆 max %.3g", maxH, maxH/0x1p-52, nonzero, maxO)
	if maxH > 16*0x1p-52 {
		t.Fatal("quaternion float associator exceeds a few eps")
	}
	if nonzero == 0 {
		t.Fatal("computed ℍ associator was identically 0: test is vacuous")
	}
	if maxO < 0.1 {
		t.Fatal("𝕆 associator should be O(1)")
	}
}
