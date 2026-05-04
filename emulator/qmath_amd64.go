package emulator

var useAVX = hasAVXAndFMA()

// qmul64AVX computes the Hamilton product of a and b using AVX/FMA3, storing the result in dst.
//go:noescape
func qmul64AVX(dst *QW64, a *QW64, b *QW64)

// qadd64AVX computes the component-wise addition of a and b using AVX, storing the result in dst.
//go:noescape
func qadd64AVX(dst *QW64, a *QW64, b *QW64)

// qrot64AVX applies unit quaternion q to vector v as qvq* using AVX/FMA3, storing the result in dst.
//go:noescape
func qrot64AVX(dst *QW64, q *QW64, v *QW64)

// qconj64AVX computes the conjugate of a using AVX, storing the result in dst.
//go:noescape
func qconj64AVX(dst *QW64, a *QW64)

// qnorm64AVX computes the norm squared (dot product) of a using AVX, storing the scalar result.
//go:noescape
func qnorm64AVX(dst *float64, a *QW64)

func qmul64(dst, a, b *QW64) {
	if useAVX {
		qmul64AVX(dst, a, b)
		return
	}
	qmul64Scalar(dst, a, b)
}

func qadd64(dst, a, b *QW64) {
	if useAVX {
		qadd64AVX(dst, a, b)
		return
	}
	qadd64Scalar(dst, a, b)
}

func qrot64(dst, q, v *QW64) {
	if useAVX {
		qrot64AVX(dst, q, v)
		return
	}
	qrot64Scalar(dst, q, v)
}

func qconj64(dst, a *QW64) {
	if useAVX {
		qconj64AVX(dst, a)
		return
	}
	qconj64Scalar(dst, a)
}

func qnorm64(dst *float64, a *QW64) {
	if useAVX {
		qnorm64AVX(dst, a)
		return
	}
	qnorm64Scalar(dst, a)
}
