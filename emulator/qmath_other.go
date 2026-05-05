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

func qmul128(dst, a, b *QW128) {
	qmul128Scalar(dst, a, b)
}

func qadd128(dst, a, b *QW128) {
	qadd128Scalar(dst, a, b)
}

func qrot128(dst, q, v *QW128) {
	qrot128Scalar(dst, q, v)
}

func qconj128(dst, a *QW128) {
	qconj128Scalar(dst, a)
}

func qnorm128(dst *QW128, a *QW128) {
	qnorm128Scalar(dst, a)
}
