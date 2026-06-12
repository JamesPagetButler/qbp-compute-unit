package emulator

import "testing"

func TestQW8_PackUnpack(t *testing.T) {
	cases := [][4]float64{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{127, -127, 64, -64},
		{200, -200, 0, 0}, // clamps to ±127
	}
	for _, in := range cases {
		packed := PackQW8(in)
		got := UnpackQW8(packed)
		for i := range 4 {
			want := in[i]
			if want > 127 {
				want = 127
			}
			if want < -127 {
				want = -127
			}
			if got[i] != want {
				t.Errorf("PackQW8(%v) component %d: got %v, want %v", in, i, got[i], want)
			}
		}
	}
}

func TestQW8_QMul8_Identity(t *testing.T) {
	g := NewGearbox()
	one := QW8{1, 0, 0, 0}
	i := QW8{0, 1, 0, 0}
	// 1 · i = i
	got := g.QMul8(one, i)
	if got != i {
		t.Fatalf("QMul8(1, i) = %v, want %v", got, i)
	}
	// i · i = -1
	got = g.QMul8(i, i)
	want := QW8{-1, 0, 0, 0}
	if got != want {
		t.Fatalf("QMul8(i, i) = %v, want %v", got, want)
	}
}

func TestQW8_QMul8_BasisProducts(t *testing.T) {
	g := NewGearbox()
	// Quaternion basis: i·j=k, j·k=i, k·i=j; anti-commutative
	qi := QW8{0, 1, 0, 0}
	qj := QW8{0, 0, 1, 0}
	qk := QW8{0, 0, 0, 1}

	if got := g.QMul8(qi, qj); got != qk {
		t.Errorf("i·j: got %v, want %v", got, qk)
	}
	if got := g.QMul8(qj, qi); got != (QW8{0, 0, 0, -1}) {
		t.Errorf("j·i: got %v, want k-negated", got)
	}
	if got := g.QMul8(qj, qk); got != qi {
		t.Errorf("j·k: got %v, want %v", got, qi)
	}
	if got := g.QMul8(qk, qi); got != qj {
		t.Errorf("k·i: got %v, want %v", got, qj)
	}
}

func TestQW8_QAdd8(t *testing.T) {
	g := NewGearbox()
	a := QW8{10, -10, 5, -5}
	b := QW8{20, 20, -10, 0}
	got := g.QAdd8(a, b)
	want := QW8{30, 10, -5, -5}
	if got != want {
		t.Fatalf("QAdd8: got %v, want %v", got, want)
	}
	// Saturation
	big := QW8{100, 100, 100, 100}
	sat := g.QAdd8(big, big)
	for _, c := range sat {
		if c != 127 {
			t.Fatalf("QAdd8 saturation: got %v, want 127", c)
		}
	}
}

func TestQW8_QConj8(t *testing.T) {
	g := NewGearbox()
	q := QW8{3, 1, -2, 4}
	got := g.QConj8(q)
	want := QW8{3, -1, 2, -4}
	if got != want {
		t.Fatalf("QConj8: got %v, want %v", got, want)
	}
}

func TestQW8_QNorm8(t *testing.T) {
	g := NewGearbox()
	// ||(1,0,0,0)||² = 1
	if n := g.QNorm8(QW8{1, 0, 0, 0}); n != 1 {
		t.Fatalf("QNorm8(1,0,0,0) = %d, want 1", n)
	}
	// ||(3,4,0,0)||² = 9+16 = 25
	if n := g.QNorm8(QW8{3, 4, 0, 0}); n != 25 {
		t.Fatalf("QNorm8(3,4,0,0) = %d, want 25", n)
	}
}

func TestQW8_DetectSeam8(t *testing.T) {
	g := NewGearbox()
	// Identity rotation: residue should be 0
	one := QW8{1, 0, 0, 0}
	v := QW8{0, 3, 4, 0}
	residue, seam := g.DetectSeam8(one, v, 5)
	if seam {
		t.Errorf("identity rotation: unexpected seam, residue=%d", residue)
	}
	// Large threshold: never fires
	_, seam = g.DetectSeam8(one, v, 1000)
	if seam {
		t.Error("threshold 1000: unexpected seam")
	}
}

func TestQW8_QRot8(t *testing.T) {
	g := NewGearbox()
	one := QW8{1, 0, 0, 0}
	qi := QW8{0, 1, 0, 0}
	qk := QW8{0, 0, 0, 1}
	vi := QW8{0, 1, 0, 0}
	vj := QW8{0, 0, 1, 0}
	vk := QW8{0, 0, 0, 1}
	cases := []struct {
		name       string
		q, v, want QW8
	}{
		{"identity·i", one, vi, vi},
		{"identity·j", one, vj, vj},
		{"rotX180·i", qi, vi, QW8{0, 1, 0, 0}},  // axis fixed
		{"rotX180·j", qi, vj, QW8{0, 0, -1, 0}}, // j -> -j
		{"rotX180·k", qi, vk, QW8{0, 0, 0, -1}}, // k -> -k
		{"rotZ180·i", qk, vi, QW8{0, -1, 0, 0}}, // i -> -i
		{"rotZ180·k", qk, vk, QW8{0, 0, 0, 1}},  // axis fixed
	}
	for _, c := range cases {
		if got := g.QRot8(c.q, c.v); got != c.want {
			t.Errorf("QRot8 %s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestQW8_DetectSeam8_PositiveSeam(t *testing.T) {
	g := NewGearbox()
	qi := QW8{0, 1, 0, 0}
	vj := QW8{0, 0, 1, 0}
	residue, seam := g.DetectSeam8(qi, vj, 1) // |(-1)-(1)| = 2 > 1
	if !seam || residue != 2 {
		t.Errorf("expected seam=true residue=2, got seam=%v residue=%d", seam, residue)
	}
}

func BenchmarkQMul8(b *testing.B) {
	g := NewGearbox()
	a := QW8{1, 0, 0, 0}
	v := QW8{0, 1, 0, 0}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = g.QMul8(a, v)
	}
}
