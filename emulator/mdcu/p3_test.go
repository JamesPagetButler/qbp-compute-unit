package mdcu

import (
	"math"
	"strings"
	"testing"
)

func randAlbert(r interface{ NormFloat64() float64 }) (A Albert) {
	for i := 0; i < 3; i++ {
		A.D[i] = r.NormFloat64()
		for k := 1; k < 8; k++ { // imaginary + real parts
			A.X[i][k] = r.NormFloat64()
		}
		A.X[i][0] = r.NormFloat64()
	}
	return
}

func albertDiff(a, b Albert) (d float64) {
	for i := 0; i < 3; i++ {
		d = math.Max(d, math.Abs(a.D[i]-b.D[i]))
		for k := 0; k < 8; k++ {
			d = math.Max(d, math.Abs(a.X[i][k]-b.X[i][k]))
		}
	}
	return
}

// A4: fast Albert product vs brute-force 3×3 octonionic matrix product.
func TestA4_Albert(t *testing.T) {
	r := rng()
	worst := 0.0
	var ctr Counter
	for n := 0; n < 5000; n++ {
		X, Y := randAlbert(r), randAlbert(r)
		worst = math.Max(worst, albertDiff(AlbertFast(&X, &Y, &ctr), AlbertBrute(&X, &Y)))
	}
	t.Logf("A4 fast vs brute max err %.3g ; OMUL/product %d", worst, ctr.OMUL/5000)
	if worst > 1e-12 {
		t.Fatal("A4: fast Albert product disagrees with brute force")
	}
}

// L9: cycle table across tile counts (model column only).
func TestL9_CycleTable(t *testing.T) {
	var b strings.Builder
	b.WriteString("\nkernel          ops")
	for _, P := range Tiles {
		b.WriteString("   P=" + itoa(P))
	}
	for _, k := range Kernels {
		b.WriteString("\n" + pad(k.Name, 15) + pad(itoa(k.Ops()), 5))
		for _, P := range Tiles {
			b.WriteString(pad(itoa(k.Cycles(P)), 6))
		}
	}
	t.Log(b.String())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func pad(s string, w int) string {
	for len(s) < w {
		s = " " + s
	}
	return s
}

// ---------------------------------------------------------------- benches

func BenchmarkGradBasis(b *testing.B) {
	s := rf688(b)[0]
	for i := 0; i < b.N; i++ {
		_ = GradBasis(&s, nil)
	}
}

func BenchmarkGradCDMAC(b *testing.B) {
	s := rf688(b)[0]
	for i := 0; i < b.N; i++ {
		_ = GradCDMAC(&s, nil)
	}
}

func benchQuench(b *testing.B, e Engine) {
	seeds := rf688(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RunQuench(seeds, 0.02, 400, e, nil)
	}
}

func BenchmarkQuenchBasis(b *testing.B) { benchQuench(b, EngBasis) }
func BenchmarkQuenchCDMAC(b *testing.B) { benchQuench(b, EngCDMAC) }
func BenchmarkQuenchLeaf(b *testing.B)  { benchQuench(b, EngLeaf) }

// A3 break-even: leaf (split+N steps+join) vs CD-MAC N steps, single seed.
func benchSteps(b *testing.B, e Engine, n int) {
	seeds := rf688(b)[:1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RunQuench(seeds, 0.02, n, e, nil)
	}
}

func BenchmarkBreakEvenCDMAC1(b *testing.B) { benchSteps(b, EngCDMAC, 1) }
func BenchmarkBreakEvenLeaf1(b *testing.B)  { benchSteps(b, EngLeaf, 1) }
func BenchmarkBreakEvenCDMAC0(b *testing.B) { benchSteps(b, EngCDMAC, 0) }
func BenchmarkBreakEvenLeaf0(b *testing.B)  { benchSteps(b, EngLeaf, 0) }

func BenchmarkOMulCDMAC(b *testing.B) {
	r := rng()
	var x, y [8]float64
	copy(x[:], randVec(r, 8))
	copy(y[:], randVec(r, 8))
	for i := 0; i < b.N; i++ {
		x = OMulCDMAC(&x, &y)
	}
}

func BenchmarkOMulRec(b *testing.B) {
	r := rng()
	var x, y [8]float64
	copy(x[:], randVec(r, 8))
	copy(y[:], randVec(r, 8))
	for i := 0; i < b.N; i++ {
		x = OMulRec(&x, &y)
	}
}

func BenchmarkSMulCDMAC(b *testing.B) {
	r := rng()
	var x, y [16]float64
	copy(x[:], randVec(r, 16))
	copy(y[:], randVec(r, 16))
	for i := 0; i < b.N; i++ {
		x = SMulCDMAC(&x, &y)
	}
}

func BenchmarkSMulRec(b *testing.B) {
	r := rng()
	var x, y [16]float64
	copy(x[:], randVec(r, 16))
	copy(y[:], randVec(r, 16))
	for i := 0; i < b.N; i++ {
		x = SMulRec(&x, &y)
	}
}

func BenchmarkAlbertFast(b *testing.B) {
	r := rng()
	X, Y := randAlbert(r), randAlbert(r)
	for i := 0; i < b.N; i++ {
		_ = AlbertFast(&X, &Y, nil)
	}
}

func BenchmarkAlbertBrute(b *testing.B) {
	r := rng()
	X, Y := randAlbert(r), randAlbert(r)
	for i := 0; i < b.N; i++ {
		_ = AlbertBrute(&X, &Y)
	}
}

func BenchmarkLeafRoundTrip(b *testing.B) {
	s := rf688(b)[0]
	for i := 0; i < b.N; i++ {
		q, fr := QSplitLeaf(&s)
		s = QJoinLeaf(q, &fr)
	}
}
