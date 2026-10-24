package mdcu

import (
	"math"
	"testing"
)

// Agreement gate (frozen plan section 4, arch #4): every variant agrees
// normwise to <= 1e-12 on the same inputs, and matches the reference
// determinant built from the recursive CD product (cdMul, an independent path).
func TestQ2_AgreementGate(t *testing.T) {
	ins := NewQ2Insts(20000, 77)
	w := Q2Agree(ins)
	t.Logf("worst normwise disagreement over %d instances: %.3g", len(ins), w)
	if w > 1e-12 {
		t.Fatalf("agreement gate failed: %.3g", w)
	}
}

func TestQ2_MatchesReferenceDet(t *testing.T) {
	ins := NewQ2Insts(2000, 78)
	worst := 0.0
	for k := range ins {
		in := &ins[k]
		ref := albertDet(in.D, in.X[:], in.Y[:], in.Z[:])
		scale := 1 + math.Abs(ref)
		for _, v := range Q2Variants {
			worst = math.Max(worst, math.Abs(v.Fn(in, nil)-ref)/scale)
		}
	}
	t.Logf("worst |variant - reference|/(1+|ref|) = %.3g", worst)
	if worst > 1e-12 {
		t.Fatal("variant disagrees with the independent reference determinant")
	}
}

// The tape's multiply count is the MAC column. It must equal the analytic
// count of each variant, so a silent change to either side is caught.
func TestQ2_TapeCountsMatchAnalytic(t *testing.T) {
	in := NewQ2Insts(1, 79)[0]
	// common part: 26 + 3 = 29 multiplies.
	want := map[string]int{
		"B1lit":  29 + 64 + 64,
		"B1lean": 29 + 64 + 8,
		"B0a":    29 + 64 + 64,
		"B0b":    29 + 64 + 25,
	}
	for _, v := range Q2Variants {
		var tp Tape
		v.Fn(&in, &tp)
		if tp.TotalMults() != want[v.Name] {
			t.Errorf("%s: tape %d multiplies, analytic %d", v.Name, tp.TotalMults(), want[v.Name])
		}
	}
}

// Recording must not change the numerical result, and nil tape must work.
func TestQ2_TapeDoesNotChangeValue(t *testing.T) {
	in := NewQ2Insts(1, 80)[0]
	for _, v := range Q2Variants {
		var tp Tape
		if v.Fn(&in, &tp) != v.Fn(&in, nil) {
			t.Errorf("%s: value depends on tape", v.Name)
		}
	}
}

func TestQ2_SchedulerBounds(t *testing.T) {
	// Hand-checked DAG: chain of 3 batches plus 2 independent batches.
	var tp Tape
	a := tp.add(64)
	b := tp.add(64, a)
	tp.add(64, b)
	tp.add(64)
	tp.add(64)
	if got := tp.CritPath(); got != 3 {
		t.Errorf("critical path %d, want 3", got)
	}
	if got := tp.DepCycles(1); got != 5 {
		t.Errorf("P=1 dependency cycles %d, want 5", got)
	}
	if got := tp.DepCycles(2); got != 3 {
		t.Errorf("P=2 dependency cycles %d, want 3", got)
	}
	if got := tp.DepCycles(8); got != 3 {
		t.Errorf("P=8 dependency cycles %d, want 3 (critical path)", got)
	}
	if got := tp.IdealCycles(2); got != 3 { // 320 mults / 128 lanes
		t.Errorf("P=2 ideal cycles %d, want 3", got)
	}
	if got := tp.IdealCycles(8); got != 1 {
		t.Errorf("P=8 ideal cycles %d, want 1", got)
	}
	// Bound (ii) can never beat bound (i) or the critical path.
	for _, v := range Q2Variants {
		var big Tape
		for _, in := range NewQ2Insts(64, 81) {
			in := in
			v.Fn(&in, &big)
		}
		for _, P := range Tiles {
			d, i := big.DepCycles(P), big.IdealCycles(P)
			if d < i || d < big.CritPath() {
				t.Errorf("%s P=%d: dependency %d below ideal %d or critical path %d", v.Name, P, d, i, big.CritPath())
			}
		}
	}
}

// Admission rule (frozen plan section 4): the workload must (a) fail RT2 --
// change under a generic non-automorphism SO(7) -- and (b) pass RT3 -- be
// invariant under the G2 torus (automorphisms) but not under generic SO(7).
// RT4 (the triality triple) is the same gate; it lives in albert_triality_test.go.
func TestQ2_Admission(t *testing.T) {
	r := rng()
	R := randSO7(r)
	var g2, so7, scale float64
	for n := 0; n < 2000; n++ {
		d := [3]float64{r.NormFloat64(), r.NormFloat64(), r.NormFloat64()}
		x, y, z := randVec(r, 8), randVec(r, 8), randVec(r, 8)
		d0 := albertDet(d, x, y, z)
		t1, t2 := r.Float64()*6, r.Float64()*6
		dg := albertDet(d, g2Torus(x, t1, t2), g2Torus(y, t1, t2), g2Torus(z, t1, t2))
		ds := albertDet(d, rot8(&R, x), rot8(&R, y), rot8(&R, z))
		g2 = math.Max(g2, math.Abs(dg-d0))
		so7 = math.Max(so7, math.Abs(ds-d0))
		scale = math.Max(scale, math.Abs(d0))
	}
	t.Logf("det change: G2 torus %.3g, generic SO(7) %.3g (scale %.3g)", g2, so7, scale)
	if g2 > 1e-10 {
		t.Fatal("determinant not invariant under G2 torus (admission b)")
	}
	if so7 < 1e-3 {
		t.Fatal("determinant invariant under generic SO(7): octonion structure not load-bearing (admission a/b)")
	}
}

func BenchmarkQ2(b *testing.B) {
	ins := NewQ2Insts(4096, 82)
	for _, v := range Q2Variants {
		v := v
		b.Run(v.Name, func(b *testing.B) {
			var s float64
			for i := 0; i < b.N; i++ {
				s += v.Fn(&ins[i&4095], nil)
			}
			_ = s
		})
	}
}
