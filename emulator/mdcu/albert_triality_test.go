package mdcu

import (
	"math"
	"testing"
)

// Prototype of studybuddy's Albert triality gate (2026-10-17).
// det(X) = abc − a|x|² − b|y|² − c|z|² + 2·Re(xyz).
// Triality triple from Moufang u(xy)u = (ux)(yu), for unit u:
//   x ↦ ux, y ↦ yu, z ↦ ū z ū  preserves |x|,|y|,|z| and Re(xyz).
// Applying the SAME map (x,y,z ↦ ux,uy,uz) to all three must change det.
//
// Note (qbp-cu-implementor, seq 3080): Re((xy)z) is independent of bracketing
// ONLY because Re([x,y,z]) = 0 for octonions (the associator is purely
// imaginary). That is what makes this determinant well-defined; the code
// below uses the left-bracketed form (xy)z.

func albertDet(d [3]float64, x, y, z []float64) float64 {
	return d[0]*d[1]*d[2] - d[0]*norm2(x) - d[1]*norm2(y) - d[2]*norm2(z) +
		2*cdMul(cdMul(x, y), z)[0]
}

func conj8(u []float64) []float64 {
	c := make([]float64, 8)
	c[0] = u[0]
	for i := 1; i < 8; i++ {
		c[i] = -u[i]
	}
	return c
}

func TestRT4_AlbertTriality(t *testing.T) {
	r := rng()
	var tri, same, scale float64
	for n := 0; n < 5000; n++ {
		d := [3]float64{r.NormFloat64(), r.NormFloat64(), r.NormFloat64()}
		x, y, z := randVec(r, 8), randVec(r, 8), randVec(r, 8)
		u := randVec(r, 8)
		normalize(u)
		ub := conj8(u)
		d0 := albertDet(d, x, y, z)
		d1 := albertDet(d, cdMul(u, x), cdMul(y, u), cdMul(cdMul(ub, z), ub))
		d2 := albertDet(d, cdMul(u, x), cdMul(u, y), cdMul(u, z))
		tri = math.Max(tri, math.Abs(d1-d0))
		same = math.Max(same, math.Abs(d2-d0))
		scale = math.Max(scale, math.Abs(d0))
	}
	t.Logf("det change: triality triple %.3g, same rotation ×3 %.3g (scale %.3g)", tri, same, scale)
	if tri > 1e-11 || same < 1e-2 {
		t.Fatal("RT4 triality gate failed")
	}
}
