package emulator

import "math"

// twoSum computes the exact sum of a and b.
func twoSum(a, b float64) (s, e float64) {
	s = a + b
	v := s - a
	e = (a - (s - v)) + (b - v)
	return
}

// fastTwoSum computes the exact sum of a and b, assuming |a| >= |b|.
func fastTwoSum(a, b float64) (s, e float64) {
	s = a + b
	e = b - (s - a)
	return
}

// twoProd computes the exact product of a and b using FMA.
func twoProd(a, b float64) (p, e float64) {
	p = a * b
	e = math.FMA(a, b, -p)
	return
}

// ddAdd adds two double-doubles.
func ddAdd(aHi, aLo, bHi, bLo float64) (sHi, sLo float64) {
	s1, e1 := twoSum(aHi, bHi)
	s2 := aLo + bLo
	s2 += e1
	sHi, sLo = fastTwoSum(s1, s2)
	return
}

// ddMul multiplies two double-doubles.
func ddMul(aHi, aLo, bHi, bLo float64) (pHi, pLo float64) {
	p1, e1 := twoProd(aHi, bHi)
	p2 := math.FMA(aHi, bLo, math.FMA(aLo, bHi, e1))
	pHi, pLo = fastTwoSum(p1, p2)
	return
}

func qmul128Scalar(dst, a, b *QW128) {
	// a: [w_hi, x_hi, y_hi, z_hi, w_lo, x_lo, y_lo, z_lo]

	// Term 1: a.W * b
	t1wHi, t1wLo := ddMul(a[0], a[4], b[0], b[4])
	t1xHi, t1xLo := ddMul(a[0], a[4], b[1], b[5])
	t1yHi, t1yLo := ddMul(a[0], a[4], b[2], b[6])
	t1zHi, t1zLo := ddMul(a[0], a[4], b[3], b[7])

	// Term 2: a.X * b (with signs: -x*x for w, x*w for x, -x*z for y, x*y for z)
	t2wHi, t2wLo := ddMul(-a[1], -a[5], b[1], b[5])
	t2xHi, t2xLo := ddMul(a[1], a[5], b[0], b[4])
	t2yHi, t2yLo := ddMul(-a[1], -a[5], b[3], b[7])
	t2zHi, t2zLo := ddMul(a[1], a[5], b[2], b[6])

	// Term 3: a.Y * b (signs: -y*y for w, y*z for x, y*w for y, -y*x for z)
	t3wHi, t3wLo := ddMul(-a[2], -a[6], b[2], b[6])
	t3xHi, t3xLo := ddMul(a[2], a[6], b[3], b[7])
	t3yHi, t3yLo := ddMul(a[2], a[6], b[0], b[4])
	t3zHi, t3zLo := ddMul(-a[2], -a[6], b[1], b[5])

	// Term 4: a.Z * b (signs: -z*z for w, -z*y for x, z*x for y, z*w for z)
	t4wHi, t4wLo := ddMul(-a[3], -a[7], b[3], b[7])
	t4xHi, t4xLo := ddMul(-a[3], -a[7], b[2], b[6])
	t4yHi, t4yLo := ddMul(a[3], a[7], b[1], b[5])
	t4zHi, t4zLo := ddMul(a[3], a[7], b[0], b[4])

	// Sum terms
	wHi, wLo := ddAdd(t1wHi, t1wLo, t2wHi, t2wLo)
	wHi, wLo = ddAdd(wHi, wLo, t3wHi, t3wLo)
	wHi, wLo = ddAdd(wHi, wLo, t4wHi, t4wLo)

	xHi, xLo := ddAdd(t1xHi, t1xLo, t2xHi, t2xLo)
	xHi, xLo = ddAdd(xHi, xLo, t3xHi, t3xLo)
	xHi, xLo = ddAdd(xHi, xLo, t4xHi, t4xLo)

	yHi, yLo := ddAdd(t1yHi, t1yLo, t2yHi, t2yLo)
	yHi, yLo = ddAdd(yHi, yLo, t3yHi, t3yLo)
	yHi, yLo = ddAdd(yHi, yLo, t4yHi, t4yLo)

	zHi, zLo := ddAdd(t1zHi, t1zLo, t2zHi, t2zLo)
	zHi, zLo = ddAdd(zHi, zLo, t3zHi, t3zLo)
	zHi, zLo = ddAdd(zHi, zLo, t4zHi, t4zLo)

	// Phase 5 Option B: Renormalize to canonical DD form
	// This restores strict associativity to within ε_DD for Capability.sandwich_mul
	wHi, wLo = twoSum(wHi, wLo)
	xHi, xLo = twoSum(xHi, xLo)
	yHi, yLo = twoSum(yHi, yLo)
	zHi, zLo = twoSum(zHi, zLo)

	dst[0], dst[4] = wHi, wLo
	dst[1], dst[5] = xHi, xLo
	dst[2], dst[6] = yHi, yLo
	dst[3], dst[7] = zHi, zLo
}

func qadd128Scalar(dst, a, b *QW128) {
	dst[0], dst[4] = ddAdd(a[0], a[4], b[0], b[4])
	dst[1], dst[5] = ddAdd(a[1], a[5], b[1], b[5])
	dst[2], dst[6] = ddAdd(a[2], a[6], b[2], b[6])
	dst[3], dst[7] = ddAdd(a[3], a[7], b[3], b[7])

	// Canonicalize
	dst[0], dst[4] = twoSum(dst[0], dst[4])
	dst[1], dst[5] = twoSum(dst[1], dst[5])
	dst[2], dst[6] = twoSum(dst[2], dst[6])
	dst[3], dst[7] = twoSum(dst[3], dst[7])
}

func qrot128Scalar(dst, q, v *QW128) {
	var tempRot QW128
	qmul128Scalar(&tempRot, q, v)
	var qConj QW128
	qconj128Scalar(&qConj, q)
	qmul128Scalar(dst, &tempRot, &qConj)
}

func qconj128Scalar(dst, a *QW128) {
	dst[0], dst[4] = a[0], a[4]
	dst[1], dst[5] = -a[1], -a[5]
	dst[2], dst[6] = -a[2], -a[6]
	dst[3], dst[7] = -a[3], -a[7]
}

func qnorm128Scalar(dst *QW128, a *QW128) {
	wHi, wLo := ddMul(a[0], a[4], a[0], a[4])
	xHi, xLo := ddMul(a[1], a[5], a[1], a[5])
	yHi, yLo := ddMul(a[2], a[6], a[2], a[6])
	zHi, zLo := ddMul(a[3], a[7], a[3], a[7])

	nHi, nLo := ddAdd(wHi, wLo, xHi, xLo)
	nHi, nLo = ddAdd(nHi, nLo, yHi, yLo)
	nHi, nLo = ddAdd(nHi, nLo, zHi, zLo)

	// Canonicalize
	nHi, nLo = twoSum(nHi, nLo)

	dst[0], dst[4] = nHi, nLo
	dst[1], dst[5] = 0, 0
	dst[2], dst[6] = 0, 0
	dst[3], dst[7] = 0, 0
}
