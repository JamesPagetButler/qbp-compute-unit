package emulator

import (
	"errors"
	"math"
	"testing"
)

// approxEqual reports whether |a - b| <= tol. NaNs compare unequal.
func approxEqual(a, b, tol float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}
	return math.Abs(a-b) <= tol
}

func approxEqualSlice(a, b []float64, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !approxEqual(a[i], b[i], tol) {
			return false
		}
	}
	return true
}

// --- Quaternion W64 ---------------------------------------------------

func TestPublicAPI_QMul64_Identity(t *testing.T) {
	g := NewGearbox()
	a := [4]float64{1, 0, 0, 0}
	b := [4]float64{0.5, -0.25, 0.125, -0.0625}
	got := g.QMul64(a, b)
	if !approxEqualSlice(got[:], b[:], 0) {
		t.Fatalf("QMul64(identity, b) = %v, want %v", got, b)
	}
}

func TestPublicAPI_QMul64_IJK(t *testing.T) {
	g := NewGearbox()
	// i * j = k (Hamilton convention): a=[0,1,0,0], b=[0,0,1,0] -> [0,0,0,1]
	got := g.QMul64([4]float64{0, 1, 0, 0}, [4]float64{0, 0, 1, 0})
	want := [4]float64{0, 0, 0, 1}
	if !approxEqualSlice(got[:], want[:], 0) {
		t.Fatalf("QMul64(i, j) = %v, want %v", got, want)
	}
}

func TestPublicAPI_QAdd64(t *testing.T) {
	g := NewGearbox()
	got := g.QAdd64([4]float64{1, 2, 3, 4}, [4]float64{0.5, -1, 1, -2})
	want := [4]float64{1.5, 1, 4, 2}
	if !approxEqualSlice(got[:], want[:], 0) {
		t.Fatalf("QAdd64 = %v, want %v", got, want)
	}
}

func TestPublicAPI_QConj64(t *testing.T) {
	g := NewGearbox()
	got := g.QConj64([4]float64{1, 2, 3, 4})
	want := [4]float64{1, -2, -3, -4}
	if got != want {
		t.Fatalf("QConj64 = %v, want %v", got, want)
	}
}

func TestPublicAPI_QNorm64(t *testing.T) {
	g := NewGearbox()
	got := g.QNorm64([4]float64{1, 2, 3, 4})
	want := 1.0 + 4 + 9 + 16
	if got != want {
		t.Fatalf("QNorm64 = %v, want %v", got, want)
	}
}

func TestPublicAPI_QRot64_Identity(t *testing.T) {
	g := NewGearbox()
	// Unit quaternion {1,0,0,0} rotates v to itself.
	v := [4]float64{0, 1, 2, 3}
	got := g.QRot64([4]float64{1, 0, 0, 0}, v)
	if !approxEqualSlice(got[:], v[:], 1e-12) {
		t.Fatalf("QRot64(1, v) = %v, want %v", got, v)
	}
}

// --- Quaternion W128 --------------------------------------------------

func TestPublicAPI_QMul128_Identity(t *testing.T) {
	g := NewGearbox()
	// Identity in DD layout: hi=[1,0,0,0], lo=[0,0,0,0]
	a := [8]float64{1, 0, 0, 0, 0, 0, 0, 0}
	b := [8]float64{0.5, -0.25, 0.125, -0.0625, 0, 0, 0, 0}
	got := g.QMul128(a, b)
	want := b
	if !approxEqualSlice(got[:], want[:], 1e-15) {
		t.Fatalf("QMul128(identity, b) = %v, want %v", got, want)
	}
}

func TestPublicAPI_QAdd128(t *testing.T) {
	g := NewGearbox()
	a := [8]float64{1, 2, 3, 4, 0, 0, 0, 0}
	b := [8]float64{0.5, -1, 1, -2, 0, 0, 0, 0}
	got := g.QAdd128(a, b)
	wantHi := [4]float64{1.5, 1, 4, 2}
	for i := 0; i < 4; i++ {
		if !approxEqual(got[i], wantHi[i], 1e-15) {
			t.Fatalf("QAdd128[%d] = %v, want %v", i, got[i], wantHi[i])
		}
	}
}

func TestPublicAPI_QConj128(t *testing.T) {
	g := NewGearbox()
	a := [8]float64{1, 2, 3, 4, 1e-16, 2e-16, 3e-16, 4e-16}
	got := g.QConj128(a)
	want := [8]float64{1, -2, -3, -4, 1e-16, -2e-16, -3e-16, -4e-16}
	if got != want {
		t.Fatalf("QConj128 = %v, want %v", got, want)
	}
}

func TestPublicAPI_QNorm128(t *testing.T) {
	g := NewGearbox()
	a := [8]float64{1, 2, 3, 4, 0, 0, 0, 0}
	got := g.QNorm128(a)
	// Norm² = 30; result lives in hi-W (index 0); other components must be zero.
	if !approxEqual(got[0], 30, 1e-12) {
		t.Fatalf("QNorm128.W_hi = %v, want 30", got[0])
	}
	for i := 1; i < 4; i++ {
		if got[i] != 0 {
			t.Fatalf("QNorm128[%d] = %v, want 0", i, got[i])
		}
	}
}

func TestPublicAPI_QRot128_Identity(t *testing.T) {
	g := NewGearbox()
	v := [8]float64{0, 1, 2, 3, 0, 0, 0, 0}
	got := g.QRot128([8]float64{1, 0, 0, 0, 0, 0, 0, 0}, v)
	for i := 0; i < 4; i++ {
		if !approxEqual(got[i], v[i], 1e-12) {
			t.Fatalf("QRot128(1, v)[%d] = %v, want %v", i, got[i], v[i])
		}
	}
}

// --- High-precision software fallback ---------------------------------

func TestPublicAPI_QMulHighPrec_W256_Identity(t *testing.T) {
	g := NewGearbox()
	a := [4]float64{1, 0, 0, 0}
	b := [4]float64{0.5, -0.25, 0.125, -0.0625}
	got, err := g.QMulHighPrec(W256, a, b)
	if err != nil {
		t.Fatalf("QMulHighPrec(W256) returned error: %v", err)
	}
	if !approxEqualSlice(got[:], b[:], 1e-15) {
		t.Fatalf("QMulHighPrec(W256, identity, b) = %v, want %v", got, b)
	}
}

func TestPublicAPI_QMulHighPrec_W512_IJK(t *testing.T) {
	g := NewGearbox()
	got, err := g.QMulHighPrec(W512, [4]float64{0, 1, 0, 0}, [4]float64{0, 0, 1, 0})
	if err != nil {
		t.Fatalf("QMulHighPrec(W512) returned error: %v", err)
	}
	want := [4]float64{0, 0, 0, 1}
	if !approxEqualSlice(got[:], want[:], 0) {
		t.Fatalf("QMulHighPrec(W512, i, j) = %v, want %v", got, want)
	}
}

func TestPublicAPI_QMulHighPrec_W1024(t *testing.T) {
	g := NewGearbox()
	a := [4]float64{1, 2, 3, 4}
	b := [4]float64{0.5, -0.25, 0.125, -0.0625}
	got, err := g.QMulHighPrec(W1024, a, b)
	if err != nil {
		t.Fatalf("QMulHighPrec(W1024) returned error: %v", err)
	}
	// Compare against QMul64 reference at fp64 precision (slack).
	want := g.QMul64(a, b)
	if !approxEqualSlice(got[:], want[:], 1e-12) {
		t.Fatalf("QMulHighPrec(W1024) = %v, want ~ %v", got, want)
	}
}

func TestPublicAPI_QMulHighPrec_RejectsFastPathWidths(t *testing.T) {
	g := NewGearbox()
	for _, w := range []Width{W8, W16, W32, W64, W128} {
		_, err := g.QMulHighPrec(w, [4]float64{1, 0, 0, 0}, [4]float64{1, 0, 0, 0})
		if err == nil {
			t.Errorf("QMulHighPrec(%v) = nil error, want non-nil (fast-path widths must be rejected)", w)
		}
	}
}

func TestPublicAPI_QMulHighPrec_PreservesActiveWidth(t *testing.T) {
	g := NewGearbox()
	g.SetWidth(W64)
	prev := g.ActiveWidth
	_, err := g.QMulHighPrec(W256, [4]float64{1, 0, 0, 0}, [4]float64{1, 0, 0, 0})
	if err != nil {
		t.Fatalf("QMulHighPrec returned error: %v", err)
	}
	if g.ActiveWidth != prev {
		t.Fatalf("QMulHighPrec mutated ActiveWidth: was %v, now %v", prev, g.ActiveWidth)
	}
}

// --- Complex ----------------------------------------------------------

func TestPublicAPI_CMul64(t *testing.T) {
	g := NewGearbox()
	// (1 + 2i)(3 + 4i) = (3 - 8) + (4 + 6)i = -5 + 10i
	got := g.CMul64([2]float64{1, 2}, [2]float64{3, 4})
	want := [2]float64{-5, 10}
	if got != want {
		t.Fatalf("CMul64 = %v, want %v", got, want)
	}
}

func TestPublicAPI_CAdd64(t *testing.T) {
	g := NewGearbox()
	got := g.CAdd64([2]float64{1, 2}, [2]float64{0.5, -1.5})
	want := [2]float64{1.5, 0.5}
	if got != want {
		t.Fatalf("CAdd64 = %v, want %v", got, want)
	}
}

func TestPublicAPI_CMul128(t *testing.T) {
	g := NewGearbox()
	// At hi-precision the high components carry the value; lo components ~0.
	a := [4]float64{1, 2, 0, 0}
	b := [4]float64{3, 4, 0, 0}
	got := g.CMul128(a, b)
	if !approxEqual(got[0], -5, 1e-15) {
		t.Fatalf("CMul128.real = %v, want -5", got[0])
	}
	if !approxEqual(got[1], 10, 1e-15) {
		t.Fatalf("CMul128.imag = %v, want 10", got[1])
	}
}

// --- Octonion / Sedenion (Crawl returns ErrTierUnsupported) -----------

func TestPublicAPI_OMul64_ReturnsTierUnsupported(t *testing.T) {
	g := NewGearbox()
	_, err := g.OMul64([8]float64{1, 0, 0, 0, 0, 0, 0, 0}, [8]float64{1, 0, 0, 0, 0, 0, 0, 0})
	if !errors.Is(err, ErrTierUnsupported) {
		t.Fatalf("OMul64 err = %v, want errors.Is(ErrTierUnsupported)", err)
	}
}

func TestPublicAPI_OAdd64_ReturnsTierUnsupported(t *testing.T) {
	g := NewGearbox()
	_, err := g.OAdd64([8]float64{1, 0, 0, 0, 0, 0, 0, 0}, [8]float64{1, 0, 0, 0, 0, 0, 0, 0})
	if !errors.Is(err, ErrTierUnsupported) {
		t.Fatalf("OAdd64 err = %v, want errors.Is(ErrTierUnsupported)", err)
	}
}

func TestPublicAPI_SMul64_ReturnsTierUnsupported(t *testing.T) {
	g := NewGearbox()
	res, err := g.SMul64([16]float64{}, [16]float64{})
	if !errors.Is(err, ErrTierUnsupported) {
		t.Fatalf("SMul64 err = %v, want errors.Is(ErrTierUnsupported)", err)
	}
	if res != (SedenionResult{}) {
		t.Fatalf("SMul64 res = %+v, want zero SedenionResult", res)
	}
}

func TestPublicAPI_SAdd64_ReturnsTierUnsupported(t *testing.T) {
	g := NewGearbox()
	_, err := g.SAdd64([16]float64{}, [16]float64{})
	if !errors.Is(err, ErrTierUnsupported) {
		t.Fatalf("SAdd64 err = %v, want errors.Is(ErrTierUnsupported)", err)
	}
}

// --- SedenionResult struct shape --------------------------------------

func TestPublicAPI_SedenionResult_Fields(t *testing.T) {
	r := SedenionResult{
		Value:     [16]float64{1, 2, 3},
		ZDClass:   1,
		ZDIndices: [4]uint8{4, 5, 6, 7},
	}
	if r.Value[0] != 1 || r.ZDClass != 1 || r.ZDIndices[3] != 7 {
		t.Fatalf("SedenionResult field round-trip failed: %+v", r)
	}
}

// --- Error sentinel ---------------------------------------------------

func TestPublicAPI_ErrTierUnsupported_IsExported(t *testing.T) {
	if ErrTierUnsupported == nil {
		t.Fatal("ErrTierUnsupported is nil; expected exported sentinel")
	}
	if ErrTierUnsupported.Error() == "" {
		t.Fatal("ErrTierUnsupported has empty message")
	}
}

// --- Hot-path zero-allocation benchmarks ------------------------------

var (
	benchQ4Sink  [4]float64
	benchQ8Sink  [8]float64
	benchC2Sink  [2]float64
	benchC4Sink  [4]float64
	benchF64Sink float64
)

func BenchmarkPublicAPI_QMul64(b *testing.B) {
	g := NewGearbox()
	a := [4]float64{0.7071, 0.5, -0.3, 0.4}
	c := [4]float64{1.5, -2.5, 3.5, -4.5}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchQ4Sink = g.QMul64(a, c)
	}
}

func BenchmarkPublicAPI_QAdd64(b *testing.B) {
	g := NewGearbox()
	a := [4]float64{0.7071, 0.5, -0.3, 0.4}
	c := [4]float64{1.5, -2.5, 3.5, -4.5}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchQ4Sink = g.QAdd64(a, c)
	}
}

func BenchmarkPublicAPI_QConj64(b *testing.B) {
	g := NewGearbox()
	a := [4]float64{0.7071, 0.5, -0.3, 0.4}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchQ4Sink = g.QConj64(a)
	}
}

func BenchmarkPublicAPI_QNorm64(b *testing.B) {
	g := NewGearbox()
	a := [4]float64{0.7071, 0.5, -0.3, 0.4}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchF64Sink = g.QNorm64(a)
	}
}

func BenchmarkPublicAPI_QRot64(b *testing.B) {
	g := NewGearbox()
	q := [4]float64{0.7071, 0.5, -0.3, 0.4}
	v := [4]float64{0, 1, 2, 3}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchQ4Sink = g.QRot64(q, v)
	}
}

func BenchmarkPublicAPI_QMul128(b *testing.B) {
	g := NewGearbox()
	a := [8]float64{0.7071, 0.5, -0.3, 0.4, 1e-16, -2e-16, 3e-16, -4e-16}
	c := [8]float64{1.5, -2.5, 3.5, -4.5, -1e-16, 2e-16, -3e-16, 4e-16}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchQ8Sink = g.QMul128(a, c)
	}
}

func BenchmarkPublicAPI_CMul64(b *testing.B) {
	g := NewGearbox()
	a := [2]float64{1, 2}
	c := [2]float64{3, 4}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchC2Sink = g.CMul64(a, c)
	}
}

func BenchmarkPublicAPI_CAdd64(b *testing.B) {
	g := NewGearbox()
	a := [2]float64{1, 2}
	c := [2]float64{3, 4}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchC2Sink = g.CAdd64(a, c)
	}
}

func BenchmarkPublicAPI_CMul128(b *testing.B) {
	g := NewGearbox()
	a := [4]float64{1, 2, 0, 0}
	c := [4]float64{3, 4, 0, 0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchC4Sink = g.CMul128(a, c)
	}
}
