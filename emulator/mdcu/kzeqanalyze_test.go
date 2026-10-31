package mdcu

// qbp-kz-g2 round 3 analyzer tests (FROZEN-v1 §3, sha256 132f6a42…1a623), written before
// the code. Synthetic series only: tiny L, iid or AR(1) draws; no simulation is run here.

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

// ---- histogram, smoothing, the barrier and its 3σ floor ------------------------------------

func TestKZEqBinningAndSmoothing(t *testing.T) {
	b := KZEqBinningOf([]float64{0, 0.5, 1}, []float64{0.25}, nil)
	if b.N != 60 || b.Lo != 0 || b.Hi != 1 {
		t.Fatalf("binning %+v, want 60 bins over [0, 1]", b)
	}
	if b.Index(0) != 0 || b.Index(1) != 59 || b.Index(0.5) != 30 || b.Index(0.999) != 59 {
		t.Errorf("indices %d %d %d %d", b.Index(0), b.Index(1), b.Index(0.5), b.Index(0.999))
	}
	if c := b.Centre(0); math.Abs(c-1.0/120) > 1e-12 {
		t.Errorf("centre of bin 0 = %g", c)
	}
	h := KZEqHistogram(b, []float64{0, 0.5, 1}, []float64{0.25})
	if h[0] != 1 || h[30] != 1 || h[59] != 1 || h[15] != 1 {
		t.Errorf("histogram %v", h)
	}
	// 3-bin moving average, truncated at the edges (the two edge bins average 2 values)
	s := KZEqSmooth3([]float64{3, 0, 0, 6, 0})
	want := []float64{1.5, 1, 2, 2, 3}
	for i := range want {
		if math.Abs(s[i]-want[i]) > 1e-12 {
			t.Errorf("smooth3 %v, want %v", s, want)
			break
		}
	}
	// a degenerate pooled sample (all equal) still gives a usable binning
	d := KZEqBinningOf([]float64{1, 1, 1})
	if d.Index(1) < 0 || d.Index(1) >= 60 || math.IsNaN(d.Hi-d.Lo) {
		t.Errorf("degenerate binning %+v", d)
	}
}

func TestKZEqBarrierResolvedAndFloor(t *testing.T) {
	// two clean peaks: lower 800, valley 100 -> resolved, dF = ln 8
	h := make([]float64, 60)
	h[20], h[21], h[22] = 100, 800, 100
	h[30] = 100
	h[40], h[41], h[42] = 100, 1000, 100
	b := KZEqBarrierOf(h)
	if !b.Resolved || !b.Candidate {
		t.Fatalf("clean two-peak not resolved: %+v", b)
	}
	if math.Abs(b.DeltaF-math.Log(8)) > 1e-12 || b.LowerPeak != 21 || b.UpperPeak != 41 || b.HLower != 800 || b.HValley != 100 {
		t.Errorf("barrier %+v, want dF = ln 8 between bins 21 and 41", b)
	}
	// the valley index is the minimum between the peaks (first on a tie)
	if h[b.Valley] != 0 {
		t.Errorf("valley bin %d has count %g, want a 0 bin", b.Valley, h[b.Valley])
	}
	// empty valley -> 0.5 before the log
	g := make([]float64, 60)
	g[10], g[20] = 100, 100
	b = KZEqBarrierOf(g)
	if !b.Resolved || math.Abs(b.DeltaF-math.Log(200)) > 1e-12 || b.HValley != 0.5 {
		t.Errorf("empty valley: %+v, want dF = ln 200", b)
	}
	// adjacent maxima only (no bin between) -> not a candidate
	a := make([]float64, 60)
	a[10], a[11] = 100, 100 // a plateau is not two strict maxima either
	a[12] = 50
	a[13] = 60
	if b = KZEqBarrierOf(a); b.Candidate {
		t.Errorf("adjacent/plateau read as a candidate: %+v", b)
	}
	// the 3σ floor: lower 630 vs valley 620 -> 10 < 3·sqrt(1250) -> unresolved, dF = 0
	f := make([]float64, 60)
	f[25], f[26], f[27] = 630, 620, 700
	b = KZEqBarrierOf(f)
	if b.Resolved || b.DeltaF != 0 {
		t.Errorf("sub-floor valley resolved: %+v", b)
	}
	if !b.Candidate {
		t.Errorf("sub-floor valley must still be a candidate (unresolved): %+v", b)
	}
	// exactly at the floor: n_lower - n_valley == 3 sqrt(sum) is NOT resolved (strict >)
	e := make([]float64, 60)
	e[10], e[20] = 100, 64 // 36 vs 3*sqrt(164) = 38.4 -> unresolved
	if b = KZEqBarrierOf(e); b.Resolved {
		t.Errorf("36 > 38.4? resolved wrongly: %+v", b)
	}
	e[20] = 40 // 60 vs 3*sqrt(140) = 35.5 -> resolved
	if b = KZEqBarrierOf(e); !b.Resolved {
		t.Errorf("60 > 35.5 not resolved: %+v", b)
	}
	// unimodal -> no candidate, dF = 0
	u := make([]float64, 60)
	for i := range u {
		u[i] = 1000 * math.Exp(-float64((i-30)*(i-30))/50)
	}
	if b = KZEqBarrierOf(u); b.Candidate || b.DeltaF != 0 {
		t.Errorf("unimodal: %+v", b)
	}
}

// §3: "a one-bin ripple in a single peak never reads as a barrier". Raw counts of a
// Gaussian bell with a ±30 ripple on the flank, smoothed, then tested: unimodal, dF = 0.
func TestKZEqSinglePeakRippleReadsUnimodal(t *testing.T) {
	raw := make([]float64, 60)
	for i := range raw {
		raw[i] = math.Round(3000 * math.Exp(-float64((i-30)*(i-30))/60))
	}
	raw[22] += 60
	raw[23] -= 60
	sm := KZEqSmooth3(raw)
	b := KZEqBarrierOf(sm)
	if b.Resolved || b.DeltaF != 0 {
		t.Errorf("ripple read as a barrier: %+v (smoothed %v)", b, sm[20:26])
	}
	// even unsmoothed the 3σ floor holds for this ripple
	if b = KZEqBarrierOf(raw); b.Resolved || b.DeltaF != 0 {
		t.Errorf("raw ripple read as a barrier: %+v", b)
	}
}

// Three maxima: the pair with the largest resolved barrier is chosen (≥ 1 bin apart).
func TestKZEqBarrierPairChoice(t *testing.T) {
	h := make([]float64, 60)
	h[10] = 1000
	h[15] = 50 // valley A-B
	h[20] = 300
	h[30] = 200 // valley B-C
	h[40] = 900
	for i := range h {
		if h[i] == 0 && i > 10 && i < 40 {
			h[i] = 250 // floor between the peaks except the two valleys
		}
	}
	h[15], h[30] = 50, 200
	b := KZEqBarrierOf(h)
	// pairs: (A,B) ln(300/50) = 1.79; (B,C) ln(300/200) = 0.41; (A,C) valley = 50, ln(900/50) = 2.89
	if !b.Resolved || b.LowerPeak != 10 || b.UpperPeak != 40 || math.Abs(b.DeltaF-math.Log(18)) > 1e-12 {
		t.Errorf("pair choice %+v, want (10, 40) with dF = ln 18", b)
	}
	if len(b.Peaks) != 3 {
		t.Errorf("peaks %v, want three", b.Peaks)
	}
}

// ---- jackknife, ε, blocked SE, start dependence, Binder -----------------------------------

func TestKZEqJackknifeAndEpsilon(t *testing.T) {
	if se := KZEqJackknifeSE([]float64{1, 2, 3}); math.Abs(se-math.Sqrt(4.0/3)) > 1e-12 {
		t.Errorf("jackknife SE %g, want sqrt(4/3)", se)
	}
	if se := KZEqJackknifeSE([]float64{2, 2, 2}); se != 0 {
		t.Errorf("jackknife SE of equal values %g, want 0", se)
	}
	if se := KZEqJackknifeSE([]float64{2}); se != 0 {
		t.Errorf("jackknife SE of one value %g, want 0", se)
	}
	if e := KZEqEpsilon(0.05, 0.05); e != 0.2 {
		t.Errorf("eps(0.05, 0.05) = %g, want the 0.2 floor", e)
	}
	if e := KZEqEpsilon(0.2, 0.3); math.Abs(e-2*math.Sqrt(0.13)) > 1e-12 {
		t.Errorf("eps(0.2, 0.3) = %g, want 2 sqrt(0.13)", e)
	}
	if KZEqDeltaF0 != math.Log(2) {
		t.Errorf("dF0 = %g, want ln 2", KZEqDeltaF0)
	}
}

func TestKZEqBlockedSE(t *testing.T) {
	// 20 blocks of 250 alternating 0 / 1 -> mean 0.5, sd(n-1) = sqrt(20·0.25/19), SE = sd/sqrt(20)
	x := make([]float64, 5000)
	for i := range x {
		if (i/250)%2 == 1 {
			x[i] = 1
		}
	}
	m, se, nb := KZEqBlockedSE([][]float64{x}, 250)
	if nb != 20 || math.Abs(m-0.5) > 1e-12 || math.Abs(se-math.Sqrt(20*0.25/19)/math.Sqrt(20)) > 1e-12 {
		t.Errorf("blocked SE: mean %g se %g nb %d", m, se, nb)
	}
	// pooled hot pair: blocks never straddle the two seeds (two series of 500 -> 2 + 2 blocks)
	a := make([]float64, 500)
	b := make([]float64, 500)
	for i := range b {
		b[i] = 1
	}
	m, se, nb = KZEqBlockedSE([][]float64{a, b}, 250)
	if nb != 4 || m != 0.5 || math.Abs(se-math.Sqrt(4*0.25/3)/2) > 1e-12 {
		t.Errorf("pooled blocked SE: mean %g se %g nb %d", m, se, nb)
	}
	// a straddling block would mix 0 and 1: with 600-long series and block 250 the tail 100 is dropped
	c := make([]float64, 600)
	for i := range c {
		c[i] = float64(i % 2)
	}
	if _, _, nb = KZEqBlockedSE([][]float64{c}, 250); nb != 2 {
		t.Errorf("600/250 -> %d blocks, want 2 (tail dropped)", nb)
	}
	// one block: SE undefined -> 0 with nb = 1, no NaN
	if _, se, nb = KZEqBlockedSE([][]float64{a[:250]}, 250); nb != 1 || se != 0 || math.IsNaN(se) {
		t.Errorf("single block: se %g nb %d", se, nb)
	}
}

func TestKZEqStartDependence(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 7))
	draw := func(n int, mu, sd float64) []float64 {
		x := make([]float64, n)
		for i := range x {
			x[i] = mu + sd*r.NormFloat64()
		}
		return x
	}
	hot := [][]float64{draw(5000, 1, 0.03), draw(5000, 1, 0.03)}
	cold := draw(5000, 1.02, 0.03)
	sd := KZEqStartDependence(hot, [][]float64{cold}, 250)
	if sd.D < 10 || sd.NHot != 40 || sd.NCold != 20 {
		t.Errorf("shifted cold start: D = %.2f (nhot %d ncold %d), want >> 3 with 40/20 blocks", sd.D, sd.NHot, sd.NCold)
	}
	same := KZEqStartDependence(hot, [][]float64{draw(5000, 1, 0.03)}, 250)
	if same.D > 3 {
		t.Errorf("same distribution: D = %.2f, want < 3", same.D)
	}
	// identical constant series: SEs are 0, the difference 0 -> D = 0, not NaN
	k := make([]float64, 5000)
	z := KZEqStartDependence([][]float64{k, k}, [][]float64{k}, 250)
	if z.D != 0 || math.IsNaN(z.D) {
		t.Errorf("constant series D = %v, want 0", z.D)
	}
}

func TestKZEqBinder(t *testing.T) {
	// a constant distribution has <e^4> = <e^2>^2 -> V = 2/3
	k := make([]float64, 100)
	for i := range k {
		k[i] = 1.3
	}
	if v := KZEqBinder(k); math.Abs(v-2.0/3) > 1e-12 {
		t.Errorf("Binder of a constant = %g, want 2/3", v)
	}
	// two-state equal weight at e1 = 1, e2 = 3: <e^2> = 5, <e^4> = 41 -> V = 1 - 41/75
	two := []float64{1, 3, 1, 3}
	if v := KZEqBinder(two); math.Abs(v-(1-41.0/75)) > 1e-12 {
		t.Errorf("two-state Binder = %g, want %g", v, 1-41.0/75)
	}
}

func TestKZEqMaxStepAndSokalOnSeries(t *testing.T) {
	x := []float64{0, 0.1, 0.05, 0.9, 0.85}
	if j, i := KZEqMaxStep(x); math.Abs(j-0.85) > 1e-12 || i != 2 {
		t.Errorf("max step %g at %d, want 0.85 at 2", j, i)
	}
	if j, i := KZEqMaxStep([]float64{1}); j != 0 || i != -1 {
		t.Errorf("single record max step %g/%d", j, i)
	}
}

// ---- synthetic 27-point sets and the rule ---------------------------------------------------

// synthGen draws the (S, e) series of replicate j at (L, T).
type synthGen func(L int, T float64, j int, r *rand.Rand) (S, E []float64)

func gauss(r *rand.Rand, n int, mu, sd float64) []float64 {
	x := make([]float64, n)
	for i := range x {
		x[i] = mu + sd*r.NormFloat64()
	}
	return x
}

// mixture draws n iid samples from N(mu1, sd) w.p. p and N(mu2, sd) otherwise.
func mixture(r *rand.Rand, n int, p, mu1, mu2, sd float64) []float64 {
	x := make([]float64, n)
	for i := range x {
		mu := mu2
		if r.Float64() < p {
			mu = mu1
		}
		x[i] = mu + sd*r.NormFloat64()
	}
	return x
}

// slowAR1 is a very slowly wandering series (phi = 0.999): tau_int_rec >> 125.
func slowAR1(r *rand.Rand, n int, mu, sd float64) []float64 {
	x := make([]float64, n)
	phi := 0.999
	v := 0.0
	for i := range x {
		v = phi*v + math.Sqrt(1-phi*phi)*r.NormFloat64()
		x[i] = mu + sd*v
	}
	return x
}

func synthPoint(L, l, tt int, r *rand.Rand, gen synthGen) KZEqPoint {
	prm := DefaultKZEqParams()
	p := KZEqPoint{Mode: "eq", PreregSHA: "p", L: L, LIndex: l, TIndex: tt, T: KZKzG2R3TTable[tt], Eta: 0.1,
		Seeds: KZKzG2R3Seeds(l, tt, false), BurnIn: prm.BurnIn, Sampling: prm.Sampling, RecEvery: prm.RecEvery,
		Complete: true, HaveEnd: true}
	starts := KZKzG2R3Starts()
	for j := 0; j < 3; j++ {
		S, E := gen(L, p.T, j, r)
		tau, _ := KZSokalTauInt(E, 6)
		ne, under := KZEqNEff(len(E), tau)
		p.Runs = append(p.Runs, KZEqSeedResult{Seed: p.Seeds[j], Start: starts[j], S: S, E: E, TauIntRec: tau, NEff: ne, Undersampled: under})
	}
	p.File = "synth-L" + itoa(L) + "-t" + itoa(tt)
	return p
}

func itoa(i int) string { return string(rune('0' + i/10)) + string(rune('0' + i%10)) }

func synthSet(seed uint64, gen synthGen) []KZEqPoint {
	r := rand.New(rand.NewPCG(seed, 99))
	var out []KZEqPoint
	for l, L := range KZKzG2R3LSet {
		for tt := range KZKzG2R3TTable {
			if KZKzG2R3Seeds(l, tt, false) == nil {
				continue
			}
			out = append(out, synthPoint(L, l, tt, r, gen))
		}
	}
	return out
}

// continuous: one Gaussian peak at every (L, T), narrowing with L; hot and cold alike
func contGen(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
	n := 5000
	return gauss(r, n, 0.3, 0.02), gauss(r, n, 1.0+0.3*(T-1.08), 0.03*8/float64(L))
}

// first-order: at L = 12, 16 the point t = 5 (T = 1.15) is an equal-weight two-state mixture
// (barrier not shrinking with L); every other point and all of L = 8 unimodal
func foGen(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
	n := 5000
	S := gauss(r, n, 0.3, 0.02)
	if L >= 12 && T == 1.15 {
		return S, mixture(r, n, 0.5, 0.9, 1.3, 0.02)
	}
	return S, gauss(r, n, 1.0+0.3*(T-1.08), 0.03*8/float64(L))
}

func decide(t *testing.T, pts []KZEqPoint) KZEqDecision {
	d, err := KZEqDecide(pts)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	return d
}

func TestKZEqDecideHCont(t *testing.T) {
	d := decide(t, synthSet(1, contGen))
	if d.Outcome != "H-CONT" || d.Rule != "(b)" || d.SubLabel != "" {
		t.Fatalf("outcome %q rule %q sub %q, want H-CONT (b)\n%s", d.Outcome, d.Rule, d.SubLabel, KZEqFormatReport(d))
	}
	for _, lr := range d.Ls {
		if lr.DeltaFStar != 0 {
			t.Errorf("L = %d: dF* = %g, want 0", lr.L, lr.DeltaFStar)
		}
		if lr.TStarIdx != 0 || lr.TStar != KZKzG2R3TTable[0] {
			t.Errorf("L = %d: T* tiebreak -> %g (idx %d), want the lowest grid T", lr.L, lr.TStar, lr.TStarIdx)
		}
		if len(lr.Points) != 7 && lr.L == 8 || len(lr.Points) != 10 && lr.L != 8 {
			t.Errorf("L = %d: %d points", lr.L, len(lr.Points))
		}
		if lr.VMin >= 2.0/3 || lr.VMin < 0.6 {
			t.Errorf("L = %d: V_min = %g, want just below 2/3", lr.L, lr.VMin)
		}
	}
	if d.Secondary16Reported {
		t.Error("secondaries at (16, T*(16)) must not be reported when dF*(16) = 0")
	}
	if d.CoexistReadable {
		t.Error("no coexistence T is readable with no bimodal point")
	}
	if d.Eps1612 < 0.2 || d.Eps128 < 0.2 {
		t.Errorf("eps below the floor: %g %g", d.Eps1612, d.Eps128)
	}
	rep := KZEqFormatReport(d)
	for _, want := range []string{"OUTCOME: H-CONT", "(b)", "dF*(16)", "tau_int", "V_min"} {
		if !strings.Contains(rep, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	if _, err := KZEqPlotJSON(d); err != nil {
		t.Errorf("plot json: %v", err)
	}
}

func TestKZEqDecideHFO(t *testing.T) {
	d := decide(t, synthSet(2, foGen))
	if d.Outcome != "H-FO" || d.Rule != "(a)" {
		t.Fatalf("outcome %q rule %q sub %q, want H-FO (a)\n%s", d.Outcome, d.Rule, d.SubLabel, KZEqFormatReport(d))
	}
	l16, l12, l8 := d.Ls[2], d.Ls[1], d.Ls[0]
	if l16.TStar != 1.15 || l12.TStar != 1.15 || l16.TStarIdx != 5 {
		t.Errorf("T*(16) = %g, T*(12) = %g, want 1.15", l16.TStar, l12.TStar)
	}
	if l16.DeltaFStar < math.Log(2) || l12.DeltaFStar < math.Log(2) || l8.DeltaFStar != 0 {
		t.Errorf("dF* = %g / %g / %g", l8.DeltaFStar, l12.DeltaFStar, l16.DeltaFStar)
	}
	if !d.Secondary16Reported || d.PhaseLockCount != 0 || d.DeltaE16 < 0.3 || d.DeltaE16 > 0.5 {
		t.Errorf("secondaries at (16, T*): reported %v phase-lock %d de %g", d.Secondary16Reported, d.PhaseLockCount, d.DeltaE16)
	}
	// only one bimodal grid point -> no coexistence T
	if d.CoexistReadable || d.CoexistN != 1 {
		t.Errorf("coexistence readable %v with %d bimodal points, want not readable / 1", d.CoexistReadable, d.CoexistN)
	}
	// the jackknife SE is finite and small for three alike seeds
	p := l16.Points[5]
	if math.IsNaN(p.SigmaDeltaF) || p.SigmaDeltaF > 0.5 || len(p.JackDeltaF) != 3 {
		t.Errorf("sigma_dF at (16, 1.15) = %g (%v)", p.SigmaDeltaF, p.JackDeltaF)
	}
	if !strings.Contains(KZEqFormatReport(d), "OUTCOME: H-FO") {
		t.Error("report lacks the outcome line")
	}
}

// (c) plain: a resolved barrier at L = 12 only.
func TestKZEqDecideInconclusivePlain(t *testing.T) {
	gen := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L == 12 {
			return foGen(L, T, j, r)
		}
		return contGen(L, T, j, r)
	}
	d := decide(t, synthSet(3, gen))
	if d.Rule != "(c)" || d.Outcome != "inconclusive" || d.SubLabel != "" {
		t.Fatalf("outcome %q rule %q sub %q, want plain (c)\n%s", d.Outcome, d.Rule, d.SubLabel, KZEqFormatReport(d))
	}
}

// (a)-side under-sampling gate: H-FO data with one under-sampled run at (16, T*(16)).
func TestKZEqDecideSamplingGateA(t *testing.T) {
	gen := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L == 16 && T == 1.15 && j == 0 {
			return gauss(r, 5000, 0.3, 0.02), slowAR1(r, 5000, 0.9, 0.002)
		}
		return foGen(L, T, j, r)
	}
	d := decide(t, synthSet(4, gen))
	if d.Rule != "(c)" || d.SubLabel != "sampling" || !d.A || !d.UnderA {
		t.Fatalf("outcome %q rule %q sub %q A %v underA %v, want (c) sampling\n%s", d.Outcome, d.Rule, d.SubLabel, d.A, d.UnderA, KZEqFormatReport(d))
	}
	if !d.Ls[2].Points[5].Runs[0].Undersampled {
		t.Error("the slow run at (16, 1.15, j = 0) is not flagged")
	}
}

// MUTATION CHECK: (b) must not fire with an under-sampled run at any (L ∈ {12,16}, T); here
// L = 12, t = 2, the cold seed. A build in which (b) still fires here is KILLED by this test.
func TestKZEqMutationBGateUndersampledL12(t *testing.T) {
	gen := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L == 12 && T == 1.12 && j == 2 {
			return gauss(r, 5000, 0.3, 0.02), slowAR1(r, 5000, 1.0+0.3*(T-1.08), 0.001)
		}
		return contGen(L, T, j, r)
	}
	d := decide(t, synthSet(5, gen))
	if d.Outcome == "H-CONT" {
		t.Fatalf("(b) fired with an under-sampled L = 12 run\n%s", KZEqFormatReport(d))
	}
	if d.Rule != "(c)" || d.SubLabel != "sampling" || !d.B || !d.UnderB {
		t.Fatalf("outcome %q rule %q sub %q B %v underB %v, want (c) sampling\n%s", d.Outcome, d.Rule, d.SubLabel, d.B, d.UnderB, KZEqFormatReport(d))
	}
	// flags at L = 8 are reported and do not gate
	gen8 := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L == 8 && T == 1.12 && j == 1 {
			return gauss(r, 5000, 0.3, 0.02), slowAR1(r, 5000, 1.0+0.3*(T-1.08), 0.001)
		}
		return contGen(L, T, j, r)
	}
	d = decide(t, synthSet(6, gen8))
	if d.Outcome != "H-CONT" || d.UnderB {
		t.Fatalf("an L = 8 flag gated (b): outcome %q underB %v\n%s", d.Outcome, d.UnderB, KZEqFormatReport(d))
	}
	if !d.Ls[0].Points[2].Runs[1].Undersampled {
		t.Error("the L = 8 flag is not reported")
	}
}

// Start dependence without a resolved pair: (c) metastability; with an under-sampled run
// as well: (c) sampling (precedence). A start-dependent point WITH a resolved pair feeds dF.
func TestKZEqDecideStartDependenceAndSubLabels(t *testing.T) {
	meta := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L == 16 && T == 1.13 && j == 2 {
			return gauss(r, 5000, 0.3, 0.02), gauss(r, 5000, 1.0+0.3*(T-1.08)+0.02, 0.015)
		}
		return contGen(L, T, j, r)
	}
	d := decide(t, synthSet(7, meta))
	if d.Rule != "(c)" || d.SubLabel != "metastability" || !d.B || !d.StartDep || !d.MetaFlag {
		t.Fatalf("outcome %q rule %q sub %q B %v startdep %v, want (c) metastability\n%s", d.Outcome, d.Rule, d.SubLabel, d.B, d.StartDep, KZEqFormatReport(d))
	}
	p := d.Ls[2].Points[3]
	if !p.StartDependent || p.Start.D <= 3 || p.Barrier.Resolved {
		t.Errorf("(16, 1.13): startdep %v D %.1f resolved %v", p.StartDependent, p.Start.D, p.Barrier.Resolved)
	}
	// sampling takes precedence over metastability
	both := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L == 12 && T == 1.18 && j == 0 {
			return gauss(r, 5000, 0.3, 0.02), slowAR1(r, 5000, 1.0+0.3*(T-1.08), 0.001)
		}
		return meta(L, T, j, r)
	}
	d = decide(t, synthSet(8, both))
	if d.Rule != "(c)" || d.SubLabel != "sampling" || !d.StartDep || !d.UnderB {
		t.Fatalf("outcome %q rule %q sub %q, want (c) sampling over metastability\n%s", d.Outcome, d.Rule, d.SubLabel, KZEqFormatReport(d))
	}
	// a start-dependent point with a resolved two-peak structure contributes to dF as any other
	locked := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		if L >= 12 && T == 1.15 {
			if j == 2 {
				return gauss(r, 5000, 0.3, 0.02), gauss(r, 5000, 0.9, 0.02)
			}
			return gauss(r, 5000, 0.3, 0.02), gauss(r, 5000, 1.3, 0.02)
		}
		return contGen(L, T, j, r)
	}
	d = decide(t, synthSet(9, locked))
	p = d.Ls[2].Points[5]
	if !p.StartDependent || !p.Barrier.Resolved || p.DeltaF < math.Log(2) {
		t.Errorf("(16, 1.15) locked starts: startdep %v resolved %v dF %g", p.StartDependent, p.Barrier.Resolved, p.DeltaF)
	}
	if d.Outcome != "H-FO" {
		t.Errorf("locked two-state data read %q (%s), want H-FO (the barrier between the starts' phases is two-state evidence)", d.Outcome, d.SubLabel)
	}
	if d.PhaseLockCount != 3 {
		t.Errorf("phase-locking count at (16, T*) = %d, want 3 (every seed stays on one side of the valley)", d.PhaseLockCount)
	}
}

// Coexistence T: equal-weight crossing interpolated between two bimodal grid points.
func TestKZEqCoexistence(t *testing.T) {
	gen := func(L int, T float64, j int, r *rand.Rand) ([]float64, []float64) {
		S := gauss(r, 5000, 0.3, 0.02)
		if L == 16 && (T == 1.14 || T == 1.15 || T == 1.16) {
			p := map[float64]float64{1.14: 0.7, 1.15: 0.55, 1.16: 0.3}[T] // weight of the low-e phase
			return S, mixture(r, 5000, p, 0.9, 1.3, 0.02)
		}
		if L == 12 && T == 1.15 {
			return S, mixture(r, 5000, 0.5, 0.9, 1.3, 0.02)
		}
		return S, gauss(r, 5000, 1.0+0.3*(T-1.08), 0.03*8/float64(L))
	}
	d := decide(t, synthSet(10, gen))
	if !d.CoexistReadable || d.CoexistN != 3 {
		t.Fatalf("coexistence readable %v n %d\n%s", d.CoexistReadable, d.CoexistN, KZEqFormatReport(d))
	}
	// the crossing of (w_low - w_high) from +0.1 at 1.15 to -0.4 at 1.16: ~1.152
	if d.CoexistT < 1.15 || d.CoexistT > 1.16 {
		t.Errorf("coexistence T = %g, want in (1.15, 1.16)", d.CoexistT)
	}
}

// Complete set: exactly 27 files, one per (L, t), each Complete, same prereg sha, mode eq.
func TestKZEqDecideRefusesPartialOrBadSets(t *testing.T) {
	full := synthSet(11, contGen)
	try := func(name string, pts []KZEqPoint, want string) {
		_, err := KZEqDecide(pts)
		if err == nil {
			t.Errorf("%s: accepted", name)
			return
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %q lacks %q", name, err, want)
		}
	}
	try("26 files", full[:26], "partial")
	try("28 files", append(append([]KZEqPoint(nil), full...), full[0]), "partial")
	try("no files", nil, "partial")
	dup := append([]KZEqPoint(nil), full...)
	dup[3] = dup[2]
	try("duplicate point", dup, "two files")
	sm := append([]KZEqPoint(nil), full...)
	sm[5].Mode = "smoke-eq"
	try("a smoke file", sm, "smoke")
	sha := append([]KZEqPoint(nil), full...)
	sha[7].PreregSHA = "q"
	try("mismatched prereg sha", sha, "prereg_sha")
	ab := append([]KZEqPoint(nil), full...)
	ab[9].Complete, ab[9].AbortCode = false, 2
	try("an aborted file", ab, "incomplete")
	ne := append([]KZEqPoint(nil), full...)
	ne[9].Complete, ne[9].HaveEnd = false, false
	try("no end record", ne, "incomplete")
	short := append([]KZEqPoint(nil), full...)
	short[1].Runs = append([]KZEqSeedResult(nil), short[1].Runs...)
	short[1].Runs[0].E = short[1].Runs[0].E[:4999]
	try("a short series", short, "records")
	two := append([]KZEqPoint(nil), full...)
	two[1].Runs = two[1].Runs[:2]
	try("two seeds", two, "seeds")
	bad := append([]KZEqPoint(nil), full...)
	bad[1].Seeds = []uint64{13000, 13001, 13002}
	try("wrong seeds", bad, "seed")
	st := append([]KZEqPoint(nil), full...)
	st[1].Runs = append([]KZEqSeedResult(nil), st[1].Runs...)
	st[1].Runs[2].Start = "haar"
	try("wrong start kinds", st, "start")
	tt := append([]KZEqPoint(nil), full...)
	tt[1].T = 1.11
	try("T off the table", tt, "T")
	bi := append([]KZEqPoint(nil), full...)
	bi[1].BurnIn = 500
	try("short burn-in", bi, "burn")
	// positive control
	if _, err := KZEqDecide(full); err != nil {
		t.Errorf("the full set refused: %v", err)
	}
}

// Loader: the records written by KZExecuteEqOnly round-trip through JSON; a 0b file, a
// smoke file and an aborted file are recognised as such.
func TestKZEqLoad(t *testing.T) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	prm := KZEqParams{BurnIn: 2, Sampling: 20, RecEvery: 10}
	cfg := KZEqOnlyConfig{L: 4, LIndex: -1, TIndex: 1, T: 1.10, Eta: 0.1, Smoke: true, Seeds: []uint64{13030, 13031},
		Starts: []string{"haar", "aligned"}, Prm: prm, Meta: map[string]any{"prereg_sha": "abc"},
		Emit: func(r map[string]any) {
			if r["kind"] == "end" {
				r["exit_code"] = 0
			}
			if err := enc.Encode(r); err != nil {
				t.Fatal(err)
			}
		}}
	if _, err := KZExecuteEqOnly(cfg); err != nil {
		t.Fatal(err)
	}
	p, err := KZEqLoad(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.Mode != "smoke-eq" || p.PreregSHA != "abc" || p.L != 4 || p.TIndex != 1 || p.T != 1.10 || p.Eta != 0.1 ||
		p.BurnIn != 2 || p.Sampling != 20 || p.RecEvery != 10 || !p.Complete || !p.HaveEnd || p.AbortCode != 0 {
		t.Errorf("loaded %+v", p)
	}
	if len(p.Runs) != 2 || p.Runs[0].Start != "haar" || p.Runs[1].Start != "aligned" || len(p.Runs[0].E) != 2 || len(p.Runs[1].S) != 2 || p.Runs[1].Seed != 13031 {
		t.Errorf("runs %+v", p.Runs)
	}
	// an aborted file
	var ab bytes.Buffer
	ab.WriteString(`{"kind":"meta","mode":"eq","prereg_sha":"abc","L":16}` + "\n" + `{"kind":"abort","code":3,"reason":"pause"}` + "\n")
	q, err := KZEqLoad(&ab)
	if err != nil || q.Complete || q.AbortCode != 3 {
		t.Errorf("aborted file: %+v err %v", q, err)
	}
	// a Stage-0b file is not an eq file
	var ob bytes.Buffer
	ob.WriteString(`{"kind":"meta","mode":"0b","tau":1600}` + "\n")
	if _, err := KZEqLoad(&ob); err == nil {
		t.Error("a 0b file loaded as eq")
	}
	if _, err := KZEqLoad(strings.NewReader("")); err == nil {
		t.Error("an empty file loaded")
	}
}
