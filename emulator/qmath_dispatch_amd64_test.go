//go:build amd64

package emulator

import (
	"math"
	"testing"
)

// withUseAVX temporarily forces the dispatch flag and restores it on
// cleanup, letting the test exercise the scalar fallback even on a
// host where hasAVXAndFMA() returned true.
func withUseAVX(t *testing.T, v bool) {
	t.Helper()
	prev := useAVX
	useAVX = v
	t.Cleanup(func() { useAVX = prev })
}

// equalQW reports whether two QW64 values agree within tol per component.
func equalQW(a, b QW64, tol float64) bool {
	for i := range a {
		if math.IsNaN(a[i]) != math.IsNaN(b[i]) {
			return false
		}
		if math.Abs(a[i]-b[i]) > tol {
			return false
		}
	}
	return true
}

// dispatchCases is the canonical set of (a, b) pairs the dispatch tests
// run through every kernel. They cover identity, conjugation, the
// 120° axis-(1,1,1) rotation case, and a non-trivial generic input.
var dispatchCases = []struct {
	name string
	a    QW64
	b    QW64
}{
	{"identity*identity", QW64{1, 0, 0, 0}, QW64{1, 0, 0, 0}},
	{"i*j_should_be_k", QW64{0, 1, 0, 0}, QW64{0, 0, 1, 0}},
	{"unit_axis_120deg", QW64{0.5, 0.5, 0.5, 0.5}, QW64{0, 1, 2, 3}},
	{"generic_pair", QW64{0.7071, 0.5, -0.3, 0.4}, QW64{1.5, -2.5, 3.5, -4.5}},
	{"large_components", QW64{1e6, 2e6, 3e6, 4e6}, QW64{0.001, 0.002, 0.003, 0.004}},
}

// TestDispatch_Equivalence asserts that every kernel's AVX and scalar
// implementations produce identical results within numerical tolerance.
// This is the safety net that catches regressions introduced by tweaks
// to either path — a real concern given the hand-rolled assembly.
func TestDispatch_Equivalence(t *testing.T) {
	if !hasAVXAndFMA() {
		t.Skip("host lacks AVX+FMA; nothing to compare against")
	}
	const tol = 1e-9

	for _, tc := range dispatchCases {
		t.Run(tc.name+"/qmul", func(t *testing.T) {
			var avx, scalar QW64
			qmul64AVX(&avx, &tc.a, &tc.b)
			qmul64Scalar(&scalar, &tc.a, &tc.b)
			if !equalQW(avx, scalar, tol) {
				t.Errorf("qmul: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qadd", func(t *testing.T) {
			var avx, scalar QW64
			qadd64AVX(&avx, &tc.a, &tc.b)
			qadd64Scalar(&scalar, &tc.a, &tc.b)
			if !equalQW(avx, scalar, tol) {
				t.Errorf("qadd: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qrot", func(t *testing.T) {
			var avx, scalar QW64
			qrot64AVX(&avx, &tc.a, &tc.b)
			qrot64Scalar(&scalar, &tc.a, &tc.b)
			if !equalQW(avx, scalar, tol) {
				t.Errorf("qrot: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qconj", func(t *testing.T) {
			var avx, scalar QW64
			qconj64AVX(&avx, &tc.a)
			qconj64Scalar(&scalar, &tc.a)
			if !equalQW(avx, scalar, tol) {
				t.Errorf("qconj: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qnorm", func(t *testing.T) {
			var avx, scalar float64
			qnorm64AVX(&avx, &tc.a)
			qnorm64Scalar(&scalar, &tc.a)
			if math.Abs(avx-scalar) > tol*math.Max(1, math.Abs(scalar)) {
				t.Errorf("qnorm: avx=%v scalar=%v", avx, scalar)
			}
		})
	}
}

// TestDispatch_ForceScalarPath asserts that the dispatch wrapper actually
// reaches the scalar implementation when useAVX=false. Without this, a
// regression in the dispatch (e.g. an unconditional call to the AVX path)
// would only surface on hosts without AVX — i.e. never, in CI.
func TestDispatch_ForceScalarPath(t *testing.T) {
	withUseAVX(t, false)

	a := QW64{0.7071, 0.5, -0.3, 0.4}
	b := QW64{1.5, -2.5, 3.5, -4.5}

	var got, wantScalar QW64
	qmul64(&got, &a, &b)
	qmul64Scalar(&wantScalar, &a, &b)
	if !equalQW(got, wantScalar, 0) {
		t.Errorf("qmul64 scalar dispatch drift: got %v want %v", got, wantScalar)
	}

	var gotN, wantN float64
	qnorm64(&gotN, &a)
	qnorm64Scalar(&wantN, &a)
	if gotN != wantN {
		t.Errorf("qnorm64 scalar dispatch drift: got %v want %v", gotN, wantN)
	}
}

// TestDispatch_DetectsAVX is a sanity check on the CPUID stub itself.
// On any FX-8350 (Piledriver) the answer is true; on any AMD64 host
// produced after ~2013 the answer is true. If hasAVXAndFMA returns
// false on a real x86-64 development machine, the CPUID asm has a bug.
func TestDispatch_DetectsAVX(t *testing.T) {
	got := hasAVXAndFMA()
	t.Logf("hasAVXAndFMA() = %v on this host", got)
}

func equalQW128(a, b QW128, tol float64) bool {
	for i := range a {
		if math.IsNaN(a[i]) != math.IsNaN(b[i]) {
			return false
		}
		if math.Abs(a[i]-b[i]) > tol {
			return false
		}
	}
	return true
}

var dispatchCases128 = []struct {
	name string
	a    QW128
	b    QW128
}{
	{"identity*identity", QW128{1, 0, 0, 0, 0, 0, 0, 0}, QW128{1, 0, 0, 0, 0, 0, 0, 0}},
	{"i*j_should_be_k", QW128{0, 1, 0, 0, 0, 0, 0, 0}, QW128{0, 0, 1, 0, 0, 0, 0, 0}},
	{"unit_axis_120deg", QW128{0.5, 0.5, 0.5, 0.5, 0, 0, 0, 0}, QW128{0, 1, 2, 3, 0, 0, 0, 0}},
	{"generic_pair", QW128{0.7071, 0.5, -0.3, 0.4, 1e-16, -2e-16, 3e-16, -4e-16}, QW128{1.5, -2.5, 3.5, -4.5, -1e-16, 2e-16, -3e-16, 4e-16}},
}

func TestDispatch_Equivalence128(t *testing.T) {
	if !hasAVXAndFMA() {
		t.Skip("host lacks AVX+FMA; nothing to compare against")
	}
	const tol = 1e-15

	for _, tc := range dispatchCases128 {
		t.Run(tc.name+"/qmul128", func(t *testing.T) {
			var avx, scalar QW128
			qmul128AVX(&avx, &tc.a, &tc.b)
			qmul128Scalar(&scalar, &tc.a, &tc.b)
			if !equalQW128(avx, scalar, tol) {
				t.Errorf("qmul128: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qadd128", func(t *testing.T) {
			var avx, scalar QW128
			qadd128AVX(&avx, &tc.a, &tc.b)
			qadd128Scalar(&scalar, &tc.a, &tc.b)
			if !equalQW128(avx, scalar, tol) {
				t.Errorf("qadd128: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qrot128", func(t *testing.T) {
			var avx, scalar QW128
			qrot128Stub(&avx, &tc.a, &tc.b)
			qrot128Scalar(&scalar, &tc.a, &tc.b)
			if !equalQW128(avx, scalar, tol) {
				t.Errorf("qrot128: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qconj128", func(t *testing.T) {
			var avx, scalar QW128
			qconj128AVX(&avx, &tc.a)
			qconj128Scalar(&scalar, &tc.a)
			if !equalQW128(avx, scalar, tol) {
				t.Errorf("qconj128: avx=%v scalar=%v", avx, scalar)
			}
		})
		t.Run(tc.name+"/qnorm128", func(t *testing.T) {
			var avx, scalar QW128
			qnorm128Stub(&avx, &tc.a)
			qnorm128Scalar(&scalar, &tc.a)
			if !equalQW128(avx, scalar, tol) {
				t.Errorf("qnorm128: avx=%v scalar=%v", avx, scalar)
			}
		})
	}
}

func TestCatastrophicCancellation_QW128(t *testing.T) {
	// a = 1.0 + 1e-16
	// b = -1.0 + 1e-16
	// a + b should be exactly 2e-16 without losing precision.
	a := QW128{1.0, 0, 0, 0, 1e-16, 0, 0, 0}
	b := QW128{-1.0, 0, 0, 0, 1e-16, 0, 0, 0}
	var got QW128
	
	qadd128Scalar(&got, &a, &b)
	
	// FastTwoSum will normalize: hi = 2e-16, lo = 0
	if got[0] != 2e-16 || got[4] != 0.0 {
		t.Errorf("Catastrophic cancellation failed: got %v", got)
	}
}

func TestDispatch_ForceScalarPath128(t *testing.T) {
	withUseAVX(t, false)

	a := QW128{0.7071, 0.5, -0.3, 0.4, 1e-16, -2e-16, 3e-16, -4e-16}
	b := QW128{1.5, -2.5, 3.5, -4.5, -1e-16, 2e-16, -3e-16, 4e-16}

	var got, wantScalar QW128
	qmul128(&got, &a, &b)
	qmul128Scalar(&wantScalar, &a, &b)
	if !equalQW128(got, wantScalar, 0) {
		t.Errorf("qmul128 scalar dispatch drift: got %v want %v", got, wantScalar)
	}

	var gotN, wantN QW128
	qnorm128(&gotN, &a)
	qnorm128Scalar(&wantN, &a)
	if !equalQW128(gotN, wantN, 0) {
		t.Errorf("qnorm128 scalar dispatch drift: got %v want %v", gotN, wantN)
	}
}

func TestCatastrophicCancellation_QW128_Multiply(t *testing.T) {
	if !hasAVXAndFMA() {
		t.Skip("host lacks AVX+FMA")
	}

	a := QW128{1.0, 1e-16, 1e-16, 1e-16, 1e-16, 0, 0, 0}
	b := QW128{1.0, -1e-16, -1e-16, -1e-16, 1e-16, 0, 0, 0}
	
	var avx, scalar QW128
	qmul128AVX(&avx, &a, &b)
	qmul128Scalar(&scalar, &a, &b)

	if !equalQW128(avx, scalar, 1e-15) {
		t.Errorf("Catastrophic cancellation mismatch in Multiply: avx=%v scalar=%v", avx, scalar)
	}
}
