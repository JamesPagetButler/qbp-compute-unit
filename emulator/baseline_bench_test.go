// baseline_bench_test.go — benchmarks that complete per-instruction coverage
// for the QBP-CU performance baseline record (doc/benchmarks/).
//
// Gap-fill (the rest live in isa_bench_test.go + public_api_test.go):
//   - FANO — the octonion Fano-product ISA op had tests (fano_test.go) but no
//     benchmark; it's a core Funct7 opcode, so the baseline must time it.
//   - High-precision QMul (W256/W512/W1024) — the big.Float software fallback
//     (QMulHighPrec). This is the slowest path and the one most worth tracking
//     across changes, so the baseline must cover it. The CPU Step fast path only
//     computes <=W64 and W128 natively; W256+ is the public QMulHighPrec route.
package emulator

import "testing"

// BenchmarkCPU_FANO times the FANO ISA opcode (e_i * e_j octonion lookup) via
// the full CPU Step path — fills the FANO benchmark gap for the baseline.
func BenchmarkCPU_FANO(b *testing.B) {
	cpu := NewCPU()
	word := buildInst(Funct7FANO, 3, 2, 3, 1) // rs2=X[3]=j, rs1=X[2]=i, rd=X[1]
	cpu.X[2] = 4                              // i = e4
	cpu.X[3] = 5                              // j = e5

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cpu.Step(word)
	}
}

// High-precision QMul (big.Float software path). These allocate by design —
// allocs/op is a first-class metric to track here (the fast paths are 0-alloc).

func BenchmarkPublicAPI_QMulHighPrec_W256(b *testing.B) {
	g := NewGearbox()
	x := [4]float64{1, 2, 3, 4}
	y := [4]float64{5, 6, 7, 8}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = g.QMulHighPrec(W256, x, y)
	}
}

func BenchmarkPublicAPI_QMulHighPrec_W512(b *testing.B) {
	g := NewGearbox()
	x := [4]float64{1, 2, 3, 4}
	y := [4]float64{5, 6, 7, 8}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = g.QMulHighPrec(W512, x, y)
	}
}

func BenchmarkPublicAPI_QMulHighPrec_W1024(b *testing.B) {
	g := NewGearbox()
	x := [4]float64{1, 2, 3, 4}
	y := [4]float64{5, 6, 7, 8}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = g.QMulHighPrec(W1024, x, y)
	}
}

// ---- Native x86 reference baseline -----------------------------------------
// Plain Go implementations of the same ops, with NO Gearbox / emulator / ISA
// dispatch — i.e. what the host AMD x86 (FX-8350) does natively. The baseline
// record divides emulator ns/op by these to report the abstraction's overhead
// factor per op ("what the chip should be able to do" vs what the CU costs).
// These are the comparison floor, not production code.

//go:noinline
func nativeQuatMul(a, c [4]float64) [4]float64 {
	// Hamilton product (w,x,y,z).
	return [4]float64{
		a[0]*c[0] - a[1]*c[1] - a[2]*c[2] - a[3]*c[3],
		a[0]*c[1] + a[1]*c[0] + a[2]*c[3] - a[3]*c[2],
		a[0]*c[2] - a[1]*c[3] + a[2]*c[0] + a[3]*c[1],
		a[0]*c[3] + a[1]*c[2] - a[2]*c[1] + a[3]*c[0],
	}
}

func BenchmarkNativeX86_QuatMul64(b *testing.B) {
	a := [4]float64{1, 2, 3, 4}
	c := [4]float64{5, 6, 7, 8}
	var r [4]float64
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r = nativeQuatMul(a, c)
	}
	_ = r
}

func BenchmarkNativeX86_QuatAdd64(b *testing.B) {
	a := [4]float64{1, 2, 3, 4}
	c := [4]float64{5, 6, 7, 8}
	var r [4]float64
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r = [4]float64{a[0] + c[0], a[1] + c[1], a[2] + c[2], a[3] + c[3]}
	}
	_ = r
}

func BenchmarkNativeX86_QuatConj64(b *testing.B) {
	a := [4]float64{1, 2, 3, 4}
	var r [4]float64
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r = [4]float64{a[0], -a[1], -a[2], -a[3]}
	}
	_ = r
}

func BenchmarkNativeX86_QuatNormSq64(b *testing.B) {
	a := [4]float64{1, 2, 3, 4}
	var r float64
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r = a[0]*a[0] + a[1]*a[1] + a[2]*a[2] + a[3]*a[3]
	}
	_ = r
}

func BenchmarkNativeX86_ComplexMul64(b *testing.B) {
	x := complex(0.6, 0.8)
	y := complex(0.6, -0.8)
	var r complex128
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r = x * y
	}
	_ = r
}
