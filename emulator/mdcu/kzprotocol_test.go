package mdcu

// Tests for the mdcu-kz ramp protocol, Stage 0 analysis and the §6 probe rule (written
// before the code). Seeds 12000+ only. Frozen text: §2a (hold, ramp, quench, read), §5
// (Stage 0a/0b), §6 (probe and the mechanical L/R rule).

import (
	"math"
	"math/rand/v2"
	"testing"
)

// mockArm records the temperature of every step and the number of detector calls.
type mockArm struct {
	temps    []float64
	etas     []float64
	detects  int
	detectAt int // len(temps) when Detect was called
}

func (m *mockArm) Step(eta, T float64, _ *rand.Rand) {
	m.temps = append(m.temps, T)
	m.etas = append(m.etas, eta)
}
func (m *mockArm) Order() float64 { return 0.5 }
func (m *mockArm) Detect(p Z2Params) Z2Result {
	m.detects++
	m.detectAt = len(m.temps)
	return Z2Result{L: 1, Cells: 1}
}

func TestKZRungProtocolSchedule(t *testing.T) {
	Tc, eta := 2.0, 0.02
	for _, tau := range KZLadder {
		m := &mockArm{}
		KZRunRung(m, eta, Tc, tau, KZNoiseRNG(12300, uint64(tau)), Amend1Z2Params())
		if want := 200 + tau + 50; len(m.temps) != want {
			t.Fatalf("tau %d: %d steps, want %d", tau, len(m.temps), want)
		}
		Thi, Tlo := 1.5*Tc, 0.5*Tc
		for k := 0; k < 200; k++ {
			if m.temps[k] != Thi {
				t.Fatalf("tau %d: hold step %d at T=%v, want %v", tau, k, m.temps[k], Thi)
			}
		}
		for k := 0; k < tau; k++ {
			want := Thi - (Thi-Tlo)*float64(k)/float64(tau)
			if math.Abs(m.temps[200+k]-want) > 1e-12 {
				t.Fatalf("tau %d: ramp step %d at T=%v, want %v", tau, k, m.temps[200+k], want)
			}
		}
		for k := 0; k < 50; k++ {
			if m.temps[200+tau+k] != 0 {
				t.Fatalf("tau %d: quench step %d at T=%v, want 0", tau, k, m.temps[200+tau+k])
			}
		}
		if m.detects != 1 || m.detectAt != len(m.temps) {
			t.Fatalf("tau %d: detector must be called once, after the last step (calls %d at %d of %d)", tau, m.detects, m.detectAt, len(m.temps))
		}
		for _, e := range m.etas {
			if e != eta {
				t.Fatalf("eta changed during the run")
			}
		}
	}
}

func TestKZLadderStepCount(t *testing.T) {
	sum := 0
	for _, tau := range KZLadder {
		sum += 200 + tau + 50
	}
	if sum != 4925 || KZStepsPerReplicate != 4925 {
		t.Fatalf("ladder steps per replicate = %d (const %d), frozen value is 4925", sum, KZStepsPerReplicate)
	}
	if len(KZLadder) != 7 || KZLadder[0] != 25 || KZLadder[6] != 1600 {
		t.Fatalf("ladder %v", KZLadder)
	}
}

func TestKZRungOnRealFieldsIsDeterministicAndPaired(t *testing.T) {
	L, eta, Tc, tau := 4, 0.02, 1.0, 25
	run := func(seed uint64) (Z2Result, float64) {
		a := &KZMainArm{F: NewKZPlaneFieldHaar(L, seed)}
		r := KZRunRung(a, eta, Tc, tau, KZNoiseRNG(seed, uint64(tau)), Amend1Z2Params())
		return r.Z2Result, r.Order
	}
	r1, o1 := run(12310)
	r2, o2 := run(12310)
	if r1.NegCells != r2.NegCells || r1.Ambiguous != r2.Ambiguous || o1 != o2 {
		t.Fatalf("not deterministic")
	}
	if _, o3 := run(12311); o3 == o1 {
		t.Fatalf("different seeds gave identical order parameter")
	}
	n := &KZNullArm{F: NewKZS3FieldHaar(L, 12310)}
	rn := KZRunRung(n, eta, Tc, tau, KZNoiseRNG(12310, uint64(tau)), Amend1Z2Params())
	if rn.NegCells != 0 || rn.Defects != 0 || !rn.ParityOK {
		t.Fatalf("null arm must read no -1 cells: %+v", rn.Z2Result)
	}
}

// ---- Stage 0a ----------------------------------------------------------------------

func TestKZGeomGrid(t *testing.T) {
	g := KZGeomGrid(0.05, 8, 40)
	if len(g) != 40 || math.Abs(g[0]-0.05) > 1e-12 || math.Abs(g[39]-8) > 1e-12 {
		t.Fatalf("grid ends %v %v len %d", g[0], g[39], len(g))
	}
	r := g[1] / g[0]
	for i := 2; i < len(g); i++ {
		if math.Abs(g[i]/g[i-1]-r) > 1e-9 {
			t.Fatalf("grid not geometric at %d", i)
		}
	}
}

func sigmoidCurve(T []float64, centre, width, top float64, seeds int, noise float64, seed uint64) [][]float64 {
	r := kzTestRand(seed)
	out := make([][]float64, seeds)
	for s := range out {
		out[s] = make([]float64, len(T))
		for i, x := range T {
			out[s][i] = top/(1+math.Exp((x-centre)/width)) + noise*r.NormFloat64()
		}
	}
	return out
}

func TestKZStage0aFindsSteepestDescent(t *testing.T) {
	T := KZGeomGrid(0.05, 8, 40)
	S := sigmoidCurve(T, 1.0, 0.08, 1.0, 3, 0.002, 12320)
	res := KZStage0aAnalyze(T, S)
	if !res.Ordered {
		t.Fatalf("ordered sigmoid reported as not ordered: %+v", res)
	}
	if res.Tc < 0.85 || res.Tc > 1.2 {
		t.Fatalf("Tc = %v, want near 1.0", res.Tc)
	}
	// Tc must lie between the two grid points of the steepest pair
	if res.PairLo < 0 || res.PairLo+1 >= len(T) || res.Tc < T[res.PairLo] || res.Tc > T[res.PairLo+1] {
		t.Fatalf("Tc %v not between grid points of the steepest pair %d", res.Tc, res.PairLo)
	}
}

func TestKZStage0aNoOrderingCases(t *testing.T) {
	T := KZGeomGrid(0.05, 8, 40)
	// flat, below 0.5 S_max at the lowest T
	flat := sigmoidCurve(T, 1e6, 1, 0.2, 3, 0.002, 12321)
	if r := KZStage0aAnalyze(T, flat); r.Ordered || r.Reason == "" {
		t.Fatalf("flat low curve must be 'no ordering': %+v", r)
	}
	// high at the lowest T but no step above 3 standard errors anywhere: a straight ramp
	// with large seed-to-seed noise
	noisy := make([][]float64, 3)
	rr := kzTestRand(12322)
	for s := range noisy {
		noisy[s] = make([]float64, len(T))
		for i := range T {
			noisy[s][i] = 0.9 + 0.3*rr.NormFloat64()
		}
	}
	if r := KZStage0aAnalyze(T, noisy); r.Ordered {
		t.Fatalf("noise-only curve must not be 'ordered': %+v", r)
	}
}

func TestKZStage0aRunShapeAndDeterminism(t *testing.T) {
	prm := KZStage0Params{NT: 5, Tmin: 0.2, Tmax: 4, Equil: 150, Meas: 50}
	mk := func(seed uint64) KZArm { return &KZMainArm{F: NewKZPlaneFieldHaar(4, seed)} }
	seeds := []uint64{12330, 12331}
	a := KZStage0aRun(mk, seeds, 0.02, prm)
	b := KZStage0aRun(mk, seeds, 0.02, prm)
	if len(a.S) != 2 || len(a.S[0]) != 5 || len(a.T) != 5 {
		t.Fatalf("shape %d x %d", len(a.S), len(a.S[0]))
	}
	for i := range a.S {
		for j := range a.S[i] {
			if a.S[i][j] != b.S[i][j] || math.IsNaN(a.S[i][j]) {
				t.Fatalf("not deterministic / NaN at %d,%d", i, j)
			}
		}
	}
	if a.T[0] >= a.T[4] {
		t.Fatalf("T must be reported ascending")
	}
	// cooling from high T: the hottest point is simulated first. With the same seeds the
	// order parameter at the coldest point must be larger than at the hottest.
	if a.S[0][0] <= a.S[0][4] {
		t.Fatalf("S at coldest T (%v) not above S at hottest T (%v)", a.S[0][0], a.S[0][4])
	}
}

// ---- Stage 0b ----------------------------------------------------------------------

func TestKZStage0bAnalyze(t *testing.T) {
	grid := KZLinGrid(0.5, 1.5, 41)
	n := len(grid)
	cool, heat := make([]float64, n), make([]float64, n)
	for i := range grid {
		cool[i] = 1 / (1 + math.Exp((grid[i]-1)/0.1))
		heat[i] = cool[i]
	}
	if r := KZStage0bAnalyze(cool, heat, 0.02, 0.02); !r.Pass {
		t.Fatalf("identical curves must pass: %+v", r)
	}
	heat2 := append([]float64(nil), heat...)
	heat2[20] += 0.049
	if r := KZStage0bAnalyze(cool, heat2, 0.02, 0.02); !r.Pass {
		t.Fatalf("a 4.9%% gap must pass: %+v", r)
	}
	heat2[20] = heat[20] + 0.0501
	if r := KZStage0bAnalyze(cool, heat2, 0.02, 0.02); r.Pass || r.Reason == "" {
		t.Fatalf("a 5.01%% gap must fail: %+v", r)
	}
	if r := KZStage0bAnalyze(cool, heat, 0.101, 0.02); r.Pass {
		t.Fatalf("a recorded-step jump of 10.1%% must fail")
	}
	if r := KZStage0bAnalyze(cool, heat, 0.02, 0.1001); r.Pass {
		t.Fatalf("a heating jump of 10.01%% must fail")
	}
	if r := KZStage0bAnalyze(cool, heat, 0.0999, 0.0999); !r.Pass {
		t.Fatalf("jumps just below 10%% must pass")
	}
}

func TestKZInterpAndJump(t *testing.T) {
	T := []float64{3, 2, 1} // a cooling record: descending T
	S := []float64{0, 0.5, 1}
	if got := KZInterp(T, S, 1.5); math.Abs(got-0.75) > 1e-12 {
		t.Fatalf("interp on descending T: %v", got)
	}
	if got := KZInterp(T, S, 0.5); got != 1 {
		t.Fatalf("clamp low end: %v", got)
	}
	if got := KZMaxJump([]float64{0, 0.3, 0.35, 0.1}); math.Abs(got-0.3) > 1e-12 {
		t.Fatalf("max jump %v", got)
	}
}

func TestKZStage0bRunShape(t *testing.T) {
	prm := KZStage0Params{TauQ: 40, RecEvery: 10, NGrid: 5}
	mk := func(seed uint64) KZArm { return &KZMainArm{F: NewKZPlaneFieldHaar(4, seed)} }
	r := KZStage0bRun(mk, []uint64{12340, 12341}, 0.02, 1.0, prm)
	if len(r.CoolOnGrid) != 5 || len(r.HeatOnGrid) != 5 {
		t.Fatalf("grid length %d %d", len(r.CoolOnGrid), len(r.HeatOnGrid))
	}
	if r.Records != 40/10+1 {
		t.Fatalf("records per branch = %d, want %d", r.Records, 40/10+1)
	}
	for _, v := range append(append([]float64{}, r.CoolOnGrid...), r.HeatOnGrid...) {
		if math.IsNaN(v) {
			t.Fatalf("NaN")
		}
	}
	// the heating branch starts from the cooled end state, so the first heating record
	// (T_lo) equals the last cooling record
	if math.Abs(r.CoolEndS-r.HeatStartS) > 1e-12 {
		t.Fatalf("heating must start from the cooled end state: %v vs %v", r.CoolEndS, r.HeatStartS)
	}
}

// ---- probe rule --------------------------------------------------------------------

func TestKZProbeWallFormula(t *testing.T) {
	w := KZProbeWall(0.01, 0.05, 100)
	want := 2*100*(4925*0.01+7*0.05) + 0.01*208000
	if math.Abs(w-want) > 1e-9 {
		t.Fatalf("W = %v, want %v", w, want)
	}
}

func TestKZProbeDecision(t *testing.T) {
	cases := []struct {
		name string
		tm   KZProbeTiming
		L, R int
		run  bool
	}{
		{"fast: L=24 R=200", KZProbeTiming{TS16: 0.001, TD16: 0.005, TS24: 0.002, TD24: 0.01}, 24, 200, true},
		{"L=24 R=50", KZProbeTiming{TS16: 0.001, TD16: 0.005, TS24: 0.005, TD24: 0.05}, 24, 50, true},
		{"R=25 at 24 only: fall back to 16", KZProbeTiming{TS16: 0.002, TD16: 0.01, TS24: 0.0078, TD24: 0.05}, 16, 200, true},
		{"slow: L=16 R=25", KZProbeTiming{TS16: 0.01, TD16: 0.05, TS24: 0.02, TD24: 0.1}, 16, 25, true},
		{"too slow: does not run", KZProbeTiming{TS16: 0.05, TD16: 0.2, TS24: 0.1, TD24: 0.4}, 0, 0, false},
	}
	for _, c := range cases {
		d := KZProbeDecide(c.tm)
		if d.Run != c.run || d.L != c.L || d.R != c.R {
			t.Errorf("%s: got %+v, want L=%d R=%d run=%v", c.name, d, c.L, c.R, c.run)
		}
		if d.Run && d.WallSeconds > 5400+1e-9 {
			t.Errorf("%s: chosen wall %v exceeds 1.5 h", c.name, d.WallSeconds)
		}
	}
	// R below 25 is never chosen and the decision is monotone in speed
	if d := KZProbeDecide(KZProbeTiming{TS16: 0.0305, TD16: 0.0, TS24: 1, TD24: 0}); d.Run && d.R < 25 {
		t.Fatalf("R < 25 chosen: %+v", d)
	}
}

func TestKZMedian(t *testing.T) {
	if m := KZMedian([]float64{5, 1, 3}); m != 3 {
		t.Fatalf("median %v", m)
	}
	if m := KZMedian([]float64{4, 1, 3, 2}); m != 2.5 {
		t.Fatalf("even median %v", m)
	}
}
