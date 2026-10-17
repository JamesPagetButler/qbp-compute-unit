package mdcu

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func loadSeeds(t testing.TB, key string) [][]float64 {
	t.Helper()
	raw, err := os.ReadFile("testdata/numpy_seeds.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	var v [][]float64
	if err := json.Unmarshal(m[key], &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func rf688(t testing.TB) [][16]float64 {
	v := loadSeeds(t, "rf688_s0")
	out := make([][16]float64, len(v))
	for i := range v {
		copy(out[i][:], v[i])
	}
	return out
}

func eqBits(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			return false
		}
	}
	return true
}

// L1: tier ladder round-trip ℝ→ℂ→ℍ→𝕆→𝕊 and back, bit-exact.
func TestL1_TierRoundTrip(t *testing.T) {
	r := rng()
	for n := 0; n < 10000; n++ {
		x := randVec(r, 2)
		s := QLift(x, TierS)
		cur := s
		for len(cur) > 2 {
			b, f := QSplit(cur, 0)
			if norm2(f) != 0 {
				t.Fatal("non-zero fiber after lift")
			}
			cur = b
		}
		if !eqBits(cur, x) {
			t.Fatal("round-trip mismatch")
		}
	}
}

// g2Torus is the maximal-torus automorphism fixing e1: (e2,e3)·θ1,
// (e4,e5)·θ2, (e6,e7)·(θ1+θ2).
func g2Torus(x []float64, t1, t2 float64) []float64 {
	z := append([]float64(nil), x...)
	rot := func(i, j int, th float64) {
		c, s := math.Cos(th), math.Sin(th)
		z[i], z[j] = c*x[i]-s*x[j], s*x[i]+c*x[j]
	}
	rot(2, 3, t1)
	rot(4, 5, t2)
	rot(6, 7, t1+t2)
	return z
}

// L2: 7-frame split/join exact; frame-average bulk norm is G₂-invariant.
func TestL2_FanoFrames(t *testing.T) {
	r := rng()
	maxDef, maxAvg, maxSingle := 0.0, 0.0, 0.0
	for n := 0; n < 10000; n++ {
		x := randVec(r, 8)
		for f := 0; f < 8; f++ {
			b, fb := QSplit(x, f)
			if !eqBits(QJoin(b, fb, f), x) {
				t.Fatalf("frame %d join mismatch", f)
			}
		}
		// automorphism check
		y := randVec(r, 8)
		t1, t2 := r.Float64()*6, r.Float64()*6
		lhs := g2Torus(cdMul(x, y), t1, t2)
		rhs := cdMul(g2Torus(x, t1, t2), g2Torus(y, t1, t2))
		maxDef = math.Max(maxDef, norm(sub(lhs, rhs)))
		gx := g2Torus(x, t1, t2)
		var avg0, avg1 float64
		for f := 1; f < 8; f++ {
			b0, _ := QSplit(x, f)
			b1, _ := QSplit(gx, f)
			avg0 += norm2(b0) / 7
			avg1 += norm2(b1) / 7
			maxSingle = math.Max(maxSingle, math.Abs(norm2(b0)-norm2(b1)))
		}
		want := x[0]*x[0] + 3.0/7*(norm2(x)-x[0]*x[0])
		maxAvg = math.Max(maxAvg, math.Max(math.Abs(avg0-avg1), math.Abs(avg0-want)))
	}
	t.Logf("G2 torus defect %.3g; frame-avg invariance %.3g; single-frame Δ up to %.3g", maxDef, maxAvg, maxSingle)
	if maxDef > 1e-13 || maxAvg > 1e-14 || maxSingle < 1e-3 {
		t.Fatal("L2 gate failed")
	}
}

// L3: lossy QProj conserves ‖bulk‖² + ledger.
func TestL3_ProjLedger(t *testing.T) {
	r := rng()
	worst := 0.0
	for n := 0; n < 10000; n++ {
		x := randVec(r, 16)
		nx := norm(x)
		for i := range x {
			x[i] /= nx
		}
		var led Ledger
		cur := x
		for len(cur) > 1 {
			cur = QProj(cur, 0, &led)
		}
		worst = math.Max(worst, math.Abs(norm2(cur)+led.Total()-1))
	}
	t.Logf("max |‖bulk‖²+ledger−1| = %.3g", worst)
	if worst > 1e-15*4 {
		t.Fatal("L3 gate failed")
	}
}

// L4: leaf ≡ 16D trajectory on the #688 seeds.
func TestL4_LeafEquivalence(t *testing.T) {
	seeds := rf688(t)
	perStep, traj := 0.0, 0.0
	for _, s := range seeds {
		q, fr := QSplitLeaf(&s)
		s1 := RK4Step16(s, 0.02, GradCDMAC, nil)
		l1 := QJoinLeaf(RK4StepLeaf(q, 0.02, nil), &fr)
		for k := range s1 {
			perStep = math.Max(perStep, math.Abs(s1[k]-l1[k]))
		}
	}
	a := RunQuench(seeds, 0.02, 400, EngCDMAC, nil)
	b := RunQuench(seeds, 0.02, 400, EngLeaf, nil)
	for i := range a {
		for k := range a[i] {
			traj = math.Max(traj, math.Abs(a[i][k]-b[i][k]))
		}
	}
	t.Logf("per-step max err %.3g; t=8 trajectory max err %.3g (doc claim 1.72e-15)", perStep, traj)
	if perStep > 2e-15 {
		t.Fatal("L4 per-step gate failed")
	}
}

// L5: escape detection against the CSR-locked frame.
func TestL5_LeafEscape(t *testing.T) {
	seeds := rf688(t)
	const eps2 = 1e-20
	maxRes := 0.0
	for _, s := range seeds[:50] {
		_, fr := QSplitLeaf(&s)
		cur := s
		for st := 0; st < 400; st++ {
			cur = RK4Step16(cur, 0.02, GradCDMAC, nil)
			maxRes = math.Max(maxRes, LeafEscape(&cur, &fr))
		}
		// perturb off-plane: must trip
		p := cur
		var w [8]float64
		for k := 1; k < 8; k++ {
			w[k] = float64(k)
		}
		var d1, d2 float64
		for k := range w {
			d1 += w[k] * fr.U1[k]
			d2 += w[k] * fr.U2[k]
		}
		for k := 1; k < 8; k++ {
			p[k] += 1e-6 * (w[k] - d1*fr.U1[k] - d2*fr.U2[k])
		}
		if LeafEscape(&p, &fr) <= eps2 {
			t.Fatal("escape not detected")
		}
		// fresh extraction would NOT detect it (sanity of T-L5 rationale)
		_, fr2 := QSplitLeaf(&p)
		if LeafEscape(&p, &fr2) > eps2 {
			t.Fatal("fresh frame unexpectedly reports escape")
		}
	}
	t.Logf("unperturbed 16D max escape r² = %.3g (r = %.3g)", maxRes, math.Sqrt(maxRes))
	if maxRes > eps2 {
		t.Fatal("false escape on unperturbed flow")
	}
}

// L5b: degenerate frames are orthonormal and round-trip.
func TestL5b_Degenerate(t *testing.T) {
	cases := [][16]float64{{}, {8: 1}, {}, {}}
	cases[2][1], cases[2][9] = 0.6, 0.8 // c ∥ a
	cases[3][9], cases[3][10] = 0.6, 0.8 // a = 0
	for i, s := range cases {
		q, fr := QSplitLeaf(&s)
		var d, n1, n2 float64
		for k := range fr.U1 {
			d += fr.U1[k] * fr.U2[k]
			n1 += fr.U1[k] * fr.U1[k]
			n2 += fr.U2[k] * fr.U2[k]
		}
		back := QJoinLeaf(q, &fr)
		if !fr.Degenerate || math.Abs(d) > 1e-15 || math.Abs(n1-1) > 1e-15 || math.Abs(n2-1) > 1e-15 {
			t.Fatalf("case %d: bad frame", i)
		}
		if norm(sub(back[:], s[:])) > 1e-15 {
			t.Fatalf("case %d: round-trip err", i)
		}
	}
}

// L6: width ladder.
func TestL6_Width(t *testing.T) {
	r := rng()
	worstULP, seams := 0.0, 0
	for n := 0; n < 20000; n++ {
		x, y := randVec(r, 16), randVec(r, 16)
		w := QWiden(x)
		zb, _, _ := QNarrow(w, 0)
		if !eqBits(zb, x) {
			t.Fatal("widen/narrow not exact")
		}
		z, _, seam := QNarrow(MulDD(QWiden(x), QWiden(y)), 1e-30)
		if seam {
			seams++
		}
		worstULP = math.Max(worstULP, ulpErr(z, MulBig(ToBig(x), ToBig(y))))
	}
	t.Logf("QW128→QW64 product worst %.3f ulp; seam fired %d/20000 at budget 1e-30", worstULP, seams)
	if worstULP > 0.5+1e-9 || seams == 0 {
		t.Fatal("L6 gate failed")
	}
}

// L7: lift commutes with multiplication (diagonal), bit-exact.
func TestL7_LiftCommutes(t *testing.T) {
	r := rng()
	for n := 0; n < 20000; n++ {
		x, y := randVec(r, 8), randVec(r, 8)
		var xo, yo [8]float64
		copy(xo[:], x)
		copy(yo[:], y)
		o := OMulCDMAC(&xo, &yo)
		var xs, ys [16]float64
		copy(xs[:], QLift(x, TierS))
		copy(ys[:], QLift(y, TierS))
		s := SMulCDMAC(&xs, &ys)
		if !eqBits(QLift(o[:], TierS), s[:]) {
			t.Fatal("lift∘OMUL ≠ SMUL∘lift")
		}
	}
}

// L8: 10⁶-step leaf soak with periodic join/escape audits.
func TestL8_Soak(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	s := rf688(t)[0]
	q, fr := QSplitLeaf(&s)
	var ctr Counter
	maxNorm, maxEsc := 0.0, 0.0
	for st := 1; st <= 1_000_000; st++ {
		q = RK4StepLeaf(q, 0.02, &ctr)
		if st%1000 == 0 {
			maxNorm = math.Max(maxNorm, math.Abs(norm2(q[:])-1))
			j := QJoinLeaf(q, &fr)
			maxEsc = math.Max(maxEsc, LeafEscape(&j, &fr))
		}
	}
	t.Logf("10⁶ steps: max |‖q‖²−1| %.3g, max escape r² %.3g, doc MACs %d", maxNorm, maxEsc, ctr.DocMACs)
	if maxNorm > 1e-13 || maxEsc > 1e-28 {
		t.Fatal("L8 gate failed")
	}
}

// CPU program: split → join → leaf → assoc through Exec.
func TestCPU_Program(t *testing.T) {
	c := NewCPU()
	r := rng()
	var x, y [8]float64
	copy(x[:], randVec(r, 8))
	copy(y[:], randVec(r, 8))
	c.Reg.SetO(0, x)
	c.Reg.SetO(1, y)
	prog := []Instr{
		{Op: OpOMUL, Rd: 2, Rs1: 0, Rs2: 1},
		{Op: OpQSPLIT, Rd: 20, Rs1: 2, Frame: 5},
		{Op: OpQJOIN, Rd: 3, Rs1: 20, Rs2: 21, Frame: 5},
		{Op: OpQASSOC, Rd: 4, Rs1: 0, Rs2: 1, Rs3: 2},
	}
	for _, in := range prog {
		if err := c.Exec(in); err != nil {
			t.Fatal(err)
		}
	}
	if c.Reg.O(2) != c.Reg.O(3) {
		t.Fatal("split/join via Exec not exact")
	}
	s := rf688(t)[3]
	c.Reg.SetS(0, s)
	for _, in := range []Instr{{Op: OpQSPLITLF, Rs1: 0}, {Op: OpQJOINLF, Rd: 1}, {Op: OpLEAFCHK, Rs1: 1}} {
		if err := c.Exec(in); err != nil {
			t.Fatal(err)
		}
	}
	if c.CSR.Flags&FlagSeam != 0 {
		t.Fatal("spurious leaf escape")
	}
	if err := c.Exec(Instr{Op: OpQWIDEN}); err == nil {
		t.Fatal("expected non-dispatchable error")
	}
	t.Logf("cycles=%d issued=%v", c.Cycles, c.Issued)
}
