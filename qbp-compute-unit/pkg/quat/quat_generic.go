//go:build !amd64

package quat

// Mul computes the Hamilton product q * r.
//
// This is the fundamental QBP operation. On Run-phase RISC-V hardware,
// this maps to a single QMUL instruction. On Crawl hardware without AVX, it is
// 16 multiplications and 12 additions.
//
//	(q.W + q.X*i + q.Y*j + q.Z*k) * (r.W + r.X*i + r.Y*j + r.Z*k)
func Mul(q, r Quat) Quat {
	return mulGeneric(q, r)
}

// MulAccum computes dest += q * r (quaternion multiply-accumulate).
// This is the QMAC instruction — the quaternionic analogue of OMAC/TMAC.
func MulAccum(dest, q, r Quat) Quat {
	return mulAccumGeneric(dest, q, r)
}
