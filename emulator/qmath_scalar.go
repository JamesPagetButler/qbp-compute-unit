package emulator

func qmul64Scalar(dst, a, b *QW64) {
	w := a[0]*b[0] - a[1]*b[1] - a[2]*b[2] - a[3]*b[3]
	x := a[0]*b[1] + a[1]*b[0] + a[2]*b[3] - a[3]*b[2]
	y := a[0]*b[2] - a[1]*b[3] + a[2]*b[0] + a[3]*b[1]
	z := a[0]*b[3] + a[1]*b[2] - a[2]*b[1] + a[3]*b[0]
	dst[0], dst[1], dst[2], dst[3] = w, x, y, z
}

func qadd64Scalar(dst, a, b *QW64) {
	dst[0] = a[0] + b[0]
	dst[1] = a[1] + b[1]
	dst[2] = a[2] + b[2]
	dst[3] = a[3] + b[3]
}

func qrot64Scalar(dst, q, v *QW64) {
	var tempRot QW64
	qmul64Scalar(&tempRot, q, v)
	var qConj QW64
	qconj64Scalar(&qConj, q)
	qmul64Scalar(dst, &tempRot, &qConj)
}

func qconj64Scalar(dst, a *QW64) {
	dst[0] = a[0]
	dst[1] = -a[1]
	dst[2] = -a[2]
	dst[3] = -a[3]
}

func qnorm64Scalar(dst *float64, a *QW64) {
	*dst = a[0]*a[0] + a[1]*a[1] + a[2]*a[2] + a[3]*a[3]
}
