//go:build !amd64

package emulator

func qmul64(dst, a, b *QW64) {
	qmul64Scalar(dst, a, b)
}

func qadd64(dst, a, b *QW64) {
	qadd64Scalar(dst, a, b)
}

func qrot64(dst, q, v *QW64) {
	qrot64Scalar(dst, q, v)
}

func qconj64(dst, a *QW64) {
	qconj64Scalar(dst, a)
}

func qnorm64(dst *float64, a *QW64) {
	qnorm64Scalar(dst, a)
}
