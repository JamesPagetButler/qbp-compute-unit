package mdcu

// Round-3 R3-Q2 tests (frozen §3.2 correctness gate + tape counts). Written
// before r3q2.go. All inputs are out-of-grid fixed seeds; none of this touches
// R3-Q1 data.

import (
	"math"
	"testing"
)

const r3q2GateSeed = 9801

func maxAbs8(a, b [8]float64) (m float64) {
	for i := range a {
		m = math.Max(m, math.Abs(a[i]-b[i]))
	}
	return
}

func TestR3Q2W2PathsAgree(t *testing.T) {
	for k, in := range NewV3Ins(1000, r3q2GateSeed) {
		in := in
		ref := phi0(in.A[:], in.B[:], in.C[:])
		cd, nc := W2CD(&in, nil), W2NonCD(&in, nil)
		if math.Abs(cd-ref) > 1e-12 || math.Abs(nc-ref) > 1e-12 {
			t.Fatalf("input %d: ref %v cd %v noncd %v", k, ref, cd, nc)
		}
	}
}

func TestR3Q2W3PathsAgree(t *testing.T) {
	for k, in := range NewV3Ins(1000, r3q2GateSeed+1) {
		in := in
		ref := assoc8(in.A[:], in.B[:], in.C[:])
		var r8 [8]float64
		copy(r8[:], ref)
		cd, nc := W3CD(&in, nil), W3NonCD(&in, nil)
		if maxAbs8(cd, r8) > 1e-12 || maxAbs8(nc, r8) > 1e-12 {
			t.Fatalf("input %d: ref %v cd %v noncd %v", k, r8, cd, nc)
		}
	}
}

func TestR3Q2W1PathsAgree(t *testing.T) {
	ins := NewQ2Insts(1000, r3q2GateSeed+2)
	for k := range ins {
		in := &ins[k]
		ref := DetB0a(in, nil) // round-2 reference (full 3-form contraction)
		scale := 1 + math.Abs(ref)
		for _, v := range R3W1Paths {
			if got := v.Fn(in, nil); math.Abs(got-ref)/scale > 1e-12 {
				t.Fatalf("input %d path %s: ref %v got %v", k, v.Name, ref, got)
			}
		}
	}
}

func tapeMults(f func(*Tape)) int {
	t := &Tape{}
	f(t)
	return t.TotalMults()
}

// Counting rule: data×data multiplies only; the frozen §3.2 numbers.
func TestR3Q2TapeCounts(t *testing.T) {
	in3 := NewV3Ins(1, r3q2GateSeed+3)[0]
	in1 := NewQ2Insts(1, r3q2GateSeed+4)[0]
	want := map[string]int{
		"W3 CD":      256,
		"W3 nonCD":   126,
		"W2 nonCD":   49,
		"W2 CD":      71,
		"W1 CD":      101, // B1lean, round-2 measured
		"W1 B0b":     118, // round-2 measured
		"W1 B0cross": 104,
	}
	got := map[string]int{
		"W3 CD":      tapeMults(func(tp *Tape) { W3CD(&in3, tp) }),
		"W3 nonCD":   tapeMults(func(tp *Tape) { W3NonCD(&in3, tp) }),
		"W2 nonCD":   tapeMults(func(tp *Tape) { W2NonCD(&in3, tp) }),
		"W2 CD":      tapeMults(func(tp *Tape) { W2CD(&in3, tp) }),
		"W1 CD":      tapeMults(func(tp *Tape) { DetB1Lean(&in1, tp) }),
		"W1 B0b":     tapeMults(func(tp *Tape) { DetB0b(&in1, tp) }),
		"W1 B0cross": tapeMults(func(tp *Tape) { DetB0Cross(&in1, tp) }),
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: tape counts %d multiplies, want %d", k, got[k], w)
		}
	}
}

// The tape node width guard (≤ TileLanes) must hold for every path.
func TestR3Q2NoNodeWiderThanTile(t *testing.T) {
	in3 := NewV3Ins(1, r3q2GateSeed+5)[0]
	in1 := NewQ2Insts(1, r3q2GateSeed+6)[0]
	tp := &Tape{}
	W2CD(&in3, tp)
	W2NonCD(&in3, tp)
	W3CD(&in3, tp)
	W3NonCD(&in3, tp)
	DetB0Cross(&in1, tp)
	for _, n := range tp.Nodes {
		if n.Mults > TileLanes {
			t.Fatalf("node of %d multiplies", n.Mults)
		}
	}
}

// Closure verdict machinery: advantage = best non-CD mults / best CD mults.
func TestR3Q2ClosureRatios(t *testing.T) {
	rows := R3Q2Closure()
	if len(rows) != 3 {
		t.Fatalf("want 3 workloads, got %d", len(rows))
	}
	for _, r := range rows {
		if r.CDMults <= 0 || r.NonCDMults <= 0 {
			t.Fatalf("%s: empty counts", r.Workload)
		}
		if got := float64(r.NonCDMults) / float64(r.CDMults); math.Abs(got-r.CDAdvantage) > 1e-12 {
			t.Fatalf("%s: advantage %v vs %v", r.Workload, r.CDAdvantage, got)
		}
		if r.CDYes != (r.CDAdvantage >= 2) {
			t.Fatalf("%s: CDYes inconsistent", r.Workload)
		}
	}
}
