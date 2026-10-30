package mdcu

// Tests for the mdcu-kz ramp protocol, Stage 0 analysis and the §6 probe rule (written
// before the code). Seeds 12000+ only. Frozen text: §2a (hold, ramp, quench, read), §5
// (Stage 0a/0b), §6 (probe and the mechanical L/R rule).

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
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
	// qbp-kz-g2 r1 (FROZEN-v1 §2 addition 4): the maximum gap, its grid index and its T are
	// what the record carries, not the last failing point. Two failing points: 10 (0.30) and
	// 30 (0.06); the maximum is at 10 even though 30 is the last one.
	heat3 := append([]float64(nil), heat...)
	heat3[10] += 0.30
	heat3[30] -= 0.06
	gap, idx, T := KZStage0bGap(grid, cool, heat3)
	if math.Abs(gap-0.30) > 1e-12 || idx != 10 || T != grid[10] {
		t.Fatalf("max gap %v at %d (T %v), want 0.30 at 10 (T %v)", gap, idx, T, grid[10])
	}
	if r := KZStage0bAnalyze(cool, heat3, 0.02, 0.02); math.Abs(r.MaxGap-gap) > 1e-12 {
		t.Fatalf("KZStage0bGap %v disagrees with KZStage0bAnalyze.MaxGap %v", gap, r.MaxGap)
	}
	// the sign of the gap does not matter
	if g2, i2, _ := KZStage0bGap(grid, heat3, cool); g2 != gap || i2 != idx {
		t.Fatalf("gap must be symmetric: %v at %d", g2, i2)
	}
	// identical curves: gap 0 at index 0
	if g0, i0, T0 := KZStage0bGap(grid, cool, heat); g0 != 0 || i0 != 0 || T0 != grid[0] {
		t.Fatalf("zero gap: %v %d %v", g0, i0, T0)
	}
	raw := KZStage0bRaw{Grid: grid, CoolOnGrid: cool, HeatOnGrid: heat3, MaxGap: gap, MaxGapIdx: idx, MaxGapT: T,
		CoolJump: 0.02, HeatJump: 0.15, CoolJumpIdx: 3, HeatJumpIdx: 7, CoolJumpT: 1.4, HeatJumpT: 0.7}
	reason := KZStage0bReason(raw)
	for _, want := range []string{"0.3000", "grid point 10", fmt.Sprintf("%.4f", grid[10]), "record 7", "0.1500"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("reason %q does not name %q", reason, want)
		}
	}
	if strings.Contains(reason, "grid point 30") {
		t.Fatalf("reason %q names the last failing point, not the maximum", reason)
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
	// qbp-kz-g2 r1 (FROZEN-v1 §2 addition 5): the jump's lower record index i of (i, i+1)
	if j, i := KZMaxJumpAt([]float64{0, 0.3, 0.35, 0.1}); math.Abs(j-0.3) > 1e-12 || i != 0 {
		t.Fatalf("max jump at: %v %d, want 0.3 at 0", j, i)
	}
	if j, i := KZMaxJumpAt([]float64{0, 0.1, 0.5, 0.45}); math.Abs(j-0.4) > 1e-12 || i != 1 {
		t.Fatalf("max jump at: %v %d, want 0.4 at 1", j, i)
	}
	if j, i := KZMaxJumpAt([]float64{0, 0.2, 0.0, 0.2}); math.Abs(j-0.2) > 1e-12 || i != 0 {
		t.Fatalf("ties resolve to the first pair: %v %d", j, i)
	}
	if j, i := KZMaxJumpAt([]float64{0.5}); j != 0 || i != -1 {
		t.Fatalf("a single record has no jump: %v %d", j, i)
	}
	for _, S := range [][]float64{{0, 0.3, 0.35, 0.1}, {1, 0.2, 0.9, 0.9, 0.3}} {
		if j, _ := KZMaxJumpAt(S); j != KZMaxJump(S) {
			t.Fatalf("KZMaxJumpAt magnitude %v != KZMaxJump %v", j, KZMaxJump(S))
		}
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
	// KZStage0bRun itself does not carry the qbp-kz-g2 r1 fields
	if r.CoolSeries != nil || r.HeatSeries != nil || r.MaxGapT != 0 {
		t.Fatalf("KZStage0bRun must stay as frozen; the series live on KZStage0bRunWithSeries")
	}

	// qbp-kz-g2 r1 (FROZEN-v1 §2 additions 3-5): the same run through the series-carrying
	// wrapper reproduces the seed-mean grid curves bit for bit and adds the per-seed series,
	// the record T labels, the max gap (index, T) and the jump positions.
	seeds := []uint64{12340, 12341}
	rs := KZStage0bRunWithSeries(mk, seeds, 0.02, 1.0, prm)
	for i := range r.Grid {
		if rs.Grid[i] != r.Grid[i] || rs.CoolOnGrid[i] != r.CoolOnGrid[i] || rs.HeatOnGrid[i] != r.HeatOnGrid[i] {
			t.Fatalf("grid curves differ at %d", i)
		}
	}
	if rs.CoolJump != r.CoolJump || rs.HeatJump != r.HeatJump || rs.Records != r.Records || rs.CoolEndS != r.CoolEndS {
		t.Fatalf("scalar fields differ: %+v vs %+v", rs, r)
	}
	nrec := prm.TauQ/prm.RecEvery + 1
	if len(rs.CoolSeries) != len(seeds) || len(rs.HeatSeries) != len(seeds) {
		t.Fatalf("series per seed: %d %d", len(rs.CoolSeries), len(rs.HeatSeries))
	}
	if len(rs.CoolT) != nrec || len(rs.HeatT) != nrec {
		t.Fatalf("record labels %d %d, want %d", len(rs.CoolT), len(rs.HeatT), nrec)
	}
	Thi, Tlo := 1.5, 0.5
	if rs.CoolT[0] != Thi || math.Abs(rs.CoolT[nrec-1]-Tlo) > 1e-12 || rs.HeatT[0] != Tlo || math.Abs(rs.HeatT[nrec-1]-Thi) > 1e-12 {
		t.Fatalf("record labels run T_hi->T_lo (cool) and T_lo->T_hi (heat): %v %v", rs.CoolT, rs.HeatT)
	}
	meanCool, meanHeat := make([]float64, nrec), make([]float64, nrec)
	for s := range seeds {
		if len(rs.CoolSeries[s]) != nrec || len(rs.HeatSeries[s]) != nrec {
			t.Fatalf("seed %d: series length %d %d, want %d", s, len(rs.CoolSeries[s]), len(rs.HeatSeries[s]), nrec)
		}
		// per seed too, heating starts from the cooled end state
		if rs.CoolSeries[s][nrec-1] != rs.HeatSeries[s][0] {
			t.Fatalf("seed %d: heat[0] %v != cool[end] %v", s, rs.HeatSeries[s][0], rs.CoolSeries[s][nrec-1])
		}
		for k := 0; k < nrec; k++ {
			meanCool[k] += rs.CoolSeries[s][k]
			meanHeat[k] += rs.HeatSeries[s][k]
		}
	}
	for k := range meanCool {
		meanCool[k] /= float64(len(seeds))
		meanHeat[k] /= float64(len(seeds))
	}
	// the seed-mean of the stored series, interpolated onto the grid, is the stored grid curve
	for i, g := range rs.Grid {
		if c := KZInterp(rs.CoolT, meanCool, g); math.Abs(c-rs.CoolOnGrid[i]) > 1e-12 {
			t.Fatalf("grid %d: cool from series %v vs stored %v", i, c, rs.CoolOnGrid[i])
		}
		if h := KZInterp(rs.HeatT, meanHeat, g); math.Abs(h-rs.HeatOnGrid[i]) > 1e-12 {
			t.Fatalf("grid %d: heat from series %v vs stored %v", i, h, rs.HeatOnGrid[i])
		}
	}
	if j, i := KZMaxJumpAt(meanCool); j != rs.CoolJump || i != rs.CoolJumpIdx || rs.CoolJumpT != rs.CoolT[i] {
		t.Fatalf("cool jump %v at %d (T %v) vs series %v at %d", rs.CoolJump, rs.CoolJumpIdx, rs.CoolJumpT, j, i)
	}
	if j, i := KZMaxJumpAt(meanHeat); j != rs.HeatJump || i != rs.HeatJumpIdx || rs.HeatJumpT != rs.HeatT[i] {
		t.Fatalf("heat jump %v at %d (T %v) vs series %v at %d", rs.HeatJump, rs.HeatJumpIdx, rs.HeatJumpT, j, i)
	}
	if g, i, T := KZStage0bGap(rs.Grid, rs.CoolOnGrid, rs.HeatOnGrid); g != rs.MaxGap || i != rs.MaxGapIdx || T != rs.MaxGapT {
		t.Fatalf("max gap %v at %d (T %v) vs recomputed %v at %d (T %v)", rs.MaxGap, rs.MaxGapIdx, rs.MaxGapT, g, i, T)
	}
	if rs.MaxGapT != rs.Grid[rs.MaxGapIdx] {
		t.Fatalf("max gap T must be the grid value at its index")
	}
	// deterministic
	rs2 := KZStage0bRunWithSeries(mk, seeds, 0.02, 1.0, prm)
	for s := range seeds {
		for k := 0; k < nrec; k++ {
			if rs2.CoolSeries[s][k] != rs.CoolSeries[s][k] {
				t.Fatalf("not deterministic")
			}
		}
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
