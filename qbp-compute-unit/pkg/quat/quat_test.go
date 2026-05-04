package quat

import (
	"math"
	"testing"
)

func TestMulAVX(t *testing.T) {
	q := Quat{W: 1, X: 2, Y: 3, Z: 4}
	r := Quat{W: 0.5, X: 1.5, Y: 2.5, Z: 3.5}

	// Calculate using the scalar generic fallback for ground truth
	expected := mulGeneric(q, r)

	// Calculate using the public API (which maps to AVX on amd64)
	actual := Mul(q, r)

	const epsilon = 1e-12
	if math.Abs(expected.W-actual.W) > epsilon ||
		math.Abs(expected.X-actual.X) > epsilon ||
		math.Abs(expected.Y-actual.Y) > epsilon ||
		math.Abs(expected.Z-actual.Z) > epsilon {
		t.Errorf("AVX Mul failed. Expected %v, got %v", expected, actual)
	}
}

func BenchmarkMulGeneric(b *testing.B) {
	q := Quat{W: 1.1, X: 2.2, Y: 3.3, Z: 4.4}
	r := Quat{W: 0.5, X: 1.5, Y: 2.5, Z: 3.5}
	var res Quat
	for i := 0; i < b.N; i++ {
		res = mulGeneric(q, r)
	}
	_ = res
}

func BenchmarkMulAVX(b *testing.B) {
	q := Quat{W: 1.1, X: 2.2, Y: 3.3, Z: 4.4}
	r := Quat{W: 0.5, X: 1.5, Y: 2.5, Z: 3.5}
	var res Quat
	for i := 0; i < b.N; i++ {
		res = Mul(q, r)
	}
	_ = res
}
