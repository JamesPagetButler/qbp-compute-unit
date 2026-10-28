package mdcu

// mdcu-kz ramp protocol, Stage 0 and the §6 probe rule (frozen mdcu-kz prereg §2a, §5,
// §6). Harness choices that the frozen text leaves open are marked HARNESS and are
// listed in the check-in so the signers can see them.

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
)

// KZLadder is the frozen τ_Q ladder.
var KZLadder = []int{25, 50, 100, 200, 400, 800, 1600}

const (
	KZHoldSteps         = 200  // steps at T_hi before the ramp
	KZQuenchSteps       = 50   // T = 0 steps after the ramp (Q)
	KZStepsPerReplicate = 4925 // 7*200 + sum(ladder) + 7*50, per arm
)

// Noise streams for Stage 0 (HARNESS). The ladder uses stream = τ_Q (25..1600), so these
// are kept clear of it.
const (
	kzStreamStage0a = 100000
	kzStreamStage0b = 100001
)

// KZArm is what the protocol needs from a field: one step, its order parameter and a
// detector pass.
type KZArm interface {
	Step(eta, T float64, rng *rand.Rand)
	Order() float64
	Detect(p Z2Params) Z2Result
}

// KZMainArm wraps the plane field.
type KZMainArm struct{ F *KZPlaneField }

func (a *KZMainArm) Step(eta, T float64, rng *rand.Rand) { a.F.Step(eta, T, rng) }
func (a *KZMainArm) Order() float64                      { return a.F.OrderS() }
func (a *KZMainArm) Detect(p Z2Params) Z2Result {
	return Z2DetectPlanes(a.F.L, a.F.Frames(), p)
}

// KZNullArm wraps the S³ field; the detector gets only the SO(3) frames (amend-1 item 3).
type KZNullArm struct{ F *KZS3Field }

func (a *KZNullArm) Step(eta, T float64, rng *rand.Rand) { a.F.Step(eta, T, rng) }
func (a *KZNullArm) Order() float64                      { return a.F.OrderM() }
func (a *KZNullArm) Detect(p Z2Params) Z2Result          { return Z2DetectS3(a.F.L, a.F.Q, p) }

// KZRungResult is the single read at the end of a run.
type KZRungResult struct {
	Z2Result
	Order float64
}

// KZRunRung is one run of the frozen protocol: hold 200 steps at T_hi = 1.5 Tc, ramp
// T(k) = T_hi - (T_hi - T_lo) k / τ_Q for k = 0..τ_Q-1 (exactly τ_Q steps) to T_lo =
// 0.5 Tc, Q = 50 steps at T = 0, then a single read. Nothing is read before it.
func KZRunRung(arm KZArm, eta, Tc float64, tauQ int, rng *rand.Rand, p Z2Params) KZRungResult {
	Thi, Tlo := 1.5*Tc, 0.5*Tc
	for k := 0; k < KZHoldSteps; k++ {
		arm.Step(eta, Thi, rng)
	}
	for k := 0; k < tauQ; k++ {
		arm.Step(eta, Thi-(Thi-Tlo)*float64(k)/float64(tauQ), rng)
	}
	for k := 0; k < KZQuenchSteps; k++ {
		arm.Step(eta, 0, rng)
	}
	return KZRungResult{Z2Result: arm.Detect(p), Order: arm.Order()}
}

// ---- grids and small numerics ---------------------------------------------------------

func KZGeomGrid(lo, hi float64, n int) []float64 {
	g := make([]float64, n)
	r := math.Pow(hi/lo, 1/float64(n-1))
	for i := range g {
		g[i] = lo * math.Pow(r, float64(i))
	}
	g[n-1] = hi
	return g
}

func KZLinGrid(lo, hi float64, n int) []float64 {
	g := make([]float64, n)
	for i := range g {
		g[i] = lo + (hi-lo)*float64(i)/float64(n-1)
	}
	return g
}

func KZMedian(x []float64) float64 {
	y := append([]float64(nil), x...)
	sort.Float64s(y)
	n := len(y)
	if n%2 == 1 {
		return y[n/2]
	}
	return (y[n/2-1] + y[n/2]) / 2
}

// KZInterp linearly interpolates S(T) at x for monotone T (either direction); x outside
// the record is clamped to the nearest end.
func KZInterp(T, S []float64, x float64) float64 {
	n := len(T)
	asc := T[n-1] >= T[0]
	lo, hi := T[0], T[n-1]
	if !asc {
		lo, hi = hi, lo
	}
	if x <= lo {
		if asc {
			return S[0]
		}
		return S[n-1]
	}
	if x >= hi {
		if asc {
			return S[n-1]
		}
		return S[0]
	}
	for i := 0; i+1 < n; i++ {
		a, b := T[i], T[i+1]
		if (a <= x && x <= b) || (b <= x && x <= a) {
			if a == b {
				return S[i]
			}
			return S[i] + (S[i+1]-S[i])*(x-a)/(b-a)
		}
	}
	return S[n-1]
}

// KZMaxJump is the largest |S[i+1] - S[i]|.
func KZMaxJump(S []float64) float64 {
	m := 0.0
	for i := 0; i+1 < len(S); i++ {
		m = math.Max(m, math.Abs(S[i+1]-S[i]))
	}
	return m
}

// ---- Stage 0 --------------------------------------------------------------------------

// KZStage0Params holds the Stage 0 numbers; DefaultKZStage0Params are the frozen ones.
type KZStage0Params struct {
	NT         int     // 0a: grid points
	Tmin, Tmax float64 // 0a: T window (units of J)
	Equil      int     // 0a: equilibration steps per point
	Meas       int     // 0a: measured steps per point
	TauQ       int     // 0b: ramp length
	RecEvery   int     // 0b: record spacing in steps
	NGrid      int     // 0b: common T grid size
}

func DefaultKZStage0Params() KZStage0Params {
	return KZStage0Params{NT: 40, Tmin: 0.05, Tmax: 8, Equil: 300, Meas: 300, TauQ: 1600, RecEvery: 10, NGrid: 41}
}

// KZStage0aRaw is the raw 0a output: T ascending, S[seed][T index].
type KZStage0aRaw struct {
	T []float64
	S [][]float64
}

// KZStage0aRun is the coarse quasi-static scan: cooling from the highest T, each point
// starting from the previous point's end state; per point Equil steps, then Meas steps
// with the order parameter averaged over the measured steps (read after each step).
func KZStage0aRun(mk func(seed uint64) KZArm, seeds []uint64, eta float64, prm KZStage0Params) KZStage0aRaw {
	grid := KZGeomGrid(prm.Tmin, prm.Tmax, prm.NT)
	out := KZStage0aRaw{T: grid, S: make([][]float64, len(seeds))}
	for si, seed := range seeds {
		arm := mk(seed)
		rng := KZNoiseRNG(seed, kzStreamStage0a)
		out.S[si] = make([]float64, prm.NT)
		for idx := prm.NT - 1; idx >= 0; idx-- {
			for k := 0; k < prm.Equil; k++ {
				arm.Step(eta, grid[idx], rng)
			}
			sum := 0.0
			for k := 0; k < prm.Meas; k++ {
				arm.Step(eta, grid[idx], rng)
				sum += arm.Order()
			}
			out.S[si][idx] = sum / float64(prm.Meas)
		}
	}
	return out
}

// KZStage0aResult is the analysis of a 0a scan.
type KZStage0aResult struct {
	T        []float64
	Mean, SE []float64
	Tc       float64 // midpoint of the steepest pair (HARNESS: arithmetic midpoint)
	PairLo   int     // index of the lower grid point of the steepest pair
	Ordered  bool
	Reason   string
}

// KZStage0aAnalyze finds T_c = the T of steepest descent of the seed-mean S(T): the
// neighbouring pair with the largest -ΔS/ΔT. Not ordered if mean S at the lowest T is
// below 0.5 S_max (S_max = 1 for both arms), or if the steepest pair's drop does not
// exceed 3 times the seed standard error of the pair (HARNESS: SE of a difference of two
// seed means, sqrt(se_i² + se_{i+1}²)).
func KZStage0aAnalyze(T []float64, S [][]float64) KZStage0aResult {
	n := len(T)
	res := KZStage0aResult{T: T, Mean: make([]float64, n), SE: make([]float64, n)}
	ns := float64(len(S))
	for i := 0; i < n; i++ {
		m := 0.0
		for s := range S {
			m += S[s][i]
		}
		m /= ns
		v := 0.0
		for s := range S {
			v += (S[s][i] - m) * (S[s][i] - m)
		}
		res.Mean[i] = m
		if len(S) > 1 {
			res.SE[i] = math.Sqrt(v/(ns-1)) / math.Sqrt(ns)
		}
	}
	best, bi := math.Inf(-1), -1
	for i := 0; i+1 < n; i++ {
		slope := (res.Mean[i] - res.Mean[i+1]) / (T[i+1] - T[i])
		if slope > best {
			best, bi = slope, i
		}
	}
	res.PairLo = bi
	if bi >= 0 {
		res.Tc = (T[bi] + T[bi+1]) / 2
	}
	switch {
	case res.Mean[0] < 0.5:
		res.Reason = fmt.Sprintf("mean S at the lowest T is %.3f, below 0.5 S_max", res.Mean[0])
	default:
		gain := res.Mean[bi] - res.Mean[bi+1]
		se := math.Sqrt(res.SE[bi]*res.SE[bi] + res.SE[bi+1]*res.SE[bi+1])
		if gain <= 3*se {
			res.Reason = fmt.Sprintf("steepest drop %.4f does not exceed 3 SE (%.4f)", gain, 3*se)
		} else {
			res.Ordered = true
		}
	}
	return res
}

// KZStage0bRaw is the 0b output on the common grid (seed-mean S).
type KZStage0bRaw struct {
	Grid                   []float64
	CoolOnGrid, HeatOnGrid []float64
	CoolJump, HeatJump     float64 // largest recorded-step jump of the seed-mean S
	Records                int     // records per branch
	CoolEndS, HeatStartS   float64 // seed-mean S at the end of cooling / start of heating
}

// KZStage0bRun is the hysteresis check: per seed, hold 200 steps at T_hi, cool over
// τ_Q steps, then heat over τ_Q steps from the cooled end state at the same rate; S is
// recorded after every RecEvery steps (and at step 0) with the nominal T(k) of the ramp
// as its label. HARNESS: the 41-point common grid is linear in T over [0.5 Tc, 1.5 Tc]
// and the seed-mean curves are interpolated linearly onto it.
func KZStage0bRun(mk func(seed uint64) KZArm, seeds []uint64, eta, Tc float64, prm KZStage0Params) KZStage0bRaw {
	Thi, Tlo := 1.5*Tc, 0.5*Tc
	nrec := prm.TauQ/prm.RecEvery + 1
	coolS, heatS := make([]float64, nrec), make([]float64, nrec)
	coolT, heatT := make([]float64, nrec), make([]float64, nrec)
	for r := 0; r < nrec; r++ {
		k := float64(r * prm.RecEvery)
		coolT[r] = Thi - (Thi-Tlo)*k/float64(prm.TauQ)
		heatT[r] = Tlo + (Thi-Tlo)*k/float64(prm.TauQ)
	}
	for _, seed := range seeds {
		arm := mk(seed)
		rng := KZNoiseRNG(seed, kzStreamStage0b)
		for k := 0; k < KZHoldSteps; k++ {
			arm.Step(eta, Thi, rng)
		}
		coolS[0] += arm.Order()
		for k := 0; k < prm.TauQ; k++ {
			arm.Step(eta, Thi-(Thi-Tlo)*float64(k)/float64(prm.TauQ), rng)
			if (k+1)%prm.RecEvery == 0 {
				coolS[(k+1)/prm.RecEvery] += arm.Order()
			}
		}
		heatS[0] += arm.Order()
		for k := 0; k < prm.TauQ; k++ {
			arm.Step(eta, Tlo+(Thi-Tlo)*float64(k)/float64(prm.TauQ), rng)
			if (k+1)%prm.RecEvery == 0 {
				heatS[(k+1)/prm.RecEvery] += arm.Order()
			}
		}
	}
	ns := float64(len(seeds))
	for r := range coolS {
		coolS[r] /= ns
		heatS[r] /= ns
	}
	grid := KZLinGrid(Tlo, Thi, prm.NGrid)
	out := KZStage0bRaw{Grid: grid, CoolOnGrid: make([]float64, prm.NGrid), HeatOnGrid: make([]float64, prm.NGrid),
		CoolJump: KZMaxJump(coolS), HeatJump: KZMaxJump(heatS), Records: nrec,
		CoolEndS: coolS[nrec-1], HeatStartS: heatS[0]}
	for i, g := range grid {
		out.CoolOnGrid[i] = KZInterp(coolT, coolS, g)
		out.HeatOnGrid[i] = KZInterp(heatT, heatS, g)
	}
	return out
}

// KZStage0bResult is the pass/fail of the hysteresis check.
type KZStage0bResult struct {
	Pass   bool
	Reason string
	MaxGap float64
}

// KZStage0bAnalyze: pass iff |cool - heat| <= 5% of S_max (= 1) at every grid T and no
// recorded-step jump of either branch exceeds 10% of S_max.
func KZStage0bAnalyze(cool, heat []float64, coolJump, heatJump float64) KZStage0bResult {
	const gapTol, jumpTol, slack = 0.05, 0.10, 1e-12
	res := KZStage0bResult{Pass: true}
	for i := range cool {
		g := math.Abs(cool[i] - heat[i])
		res.MaxGap = math.Max(res.MaxGap, g)
		if g > gapTol+slack {
			res.Pass = false
			res.Reason = fmt.Sprintf("hysteresis gap %.4f > 5%% of S_max at grid point %d", g, i)
		}
	}
	if coolJump > jumpTol+slack || heatJump > jumpTol+slack {
		res.Pass = false
		res.Reason += fmt.Sprintf(" recorded-step jump %.4f/%.4f > 10%% of S_max", coolJump, heatJump)
	}
	if !res.Pass {
		res.Reason = "first-order or glassy, premise fails:" + res.Reason
	}
	return res
}

// ---- §6 probe rule ----------------------------------------------------------------------

// KZProbeBudgetSeconds is 1.5 h; the hard cap of 2 h is a runtime kill.
const KZProbeBudgetSeconds = 5400.0

// KZProbeTiming holds the medians measured by the probe (both arms averaged for t_s).
type KZProbeTiming struct{ TS16, TD16, TS24, TD24 float64 }

// KZProbeWall is W(L, R) = 2·R·(4925·t_s + 7·t_d) + 208,000·t_s.
func KZProbeWall(ts, td float64, R int) float64 {
	return 2*float64(R)*(float64(KZStepsPerReplicate)*ts+7*td) + 208000*ts
}

// KZProbeDecision is the mechanical L/R outcome.
type KZProbeDecision struct {
	Run         bool
	L, R        int
	WallSeconds float64
	LowPower    bool // R of 25 or 50
}

// KZProbeDecide applies the frozen rule: R ∈ {200,100,50,25}; for a given L the largest
// R with W <= 1.5 h; L = 24 iff some R >= 50 fits at L = 24, else L = 16 with the largest
// fitting R; if even R = 25 does not fit at L = 16, the round does not run. It reads
// timings only, never counts.
func KZProbeDecide(tm KZProbeTiming) KZProbeDecision {
	rs := []int{200, 100, 50, 25}
	pick := func(ts, td float64) (int, float64, bool) {
		for _, R := range rs {
			if w := KZProbeWall(ts, td, R); w <= KZProbeBudgetSeconds {
				return R, w, true
			}
		}
		return 0, 0, false
	}
	if R, w, ok := pick(tm.TS24, tm.TD24); ok && R >= 50 {
		return KZProbeDecision{Run: true, L: 24, R: R, WallSeconds: w, LowPower: R <= 50}
	}
	if R, w, ok := pick(tm.TS16, tm.TD16); ok {
		return KZProbeDecision{Run: true, L: 16, R: R, WallSeconds: w, LowPower: R <= 50}
	}
	return KZProbeDecision{}
}
