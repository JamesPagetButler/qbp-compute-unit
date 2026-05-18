//go:build amd64

package quat

// mulAVX is implemented in quat_amd64.s using AVX instructions.
//
//go:noescape
func mulAVX(q, r, res *Quat)

// mulAccumAVX is implemented in quat_amd64.s using AVX instructions.
//
//go:noescape
func mulAccumAVX(dest, q, r *Quat)

// Mul computes the Hamilton product q * r.
//
// This maps to the hardware-accelerated AVX kernel on amd64,
// reducing 16 scalar multiplications and 12 additions to a few
// parallel SIMD instructions.
//
//	(q.W + q.X*i + q.Y*j + q.Z*k) * (r.W + r.X*i + r.Y*j + r.Z*k)
func Mul(q, r Quat) Quat {
	var res Quat
	mulAVX(&q, &r, &res)
	return res
}

// MulAccum computes dest += q * r (quaternion multiply-accumulate).
// This is the QMAC instruction — the quaternionic analogue of OMAC/TMAC.
func MulAccum(dest, q, r Quat) Quat {
	mulAccumAVX(&dest, &q, &r)
	return dest
}
