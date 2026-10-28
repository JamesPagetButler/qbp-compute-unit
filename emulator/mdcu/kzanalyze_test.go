package mdcu

// Tests for the mdcu-kz analyzer (frozen §3 observable, §5 Q5a/Q5b/Q5c and kills),
// written before the code. Synthetic data only; seeds 12000+.

import (
	"math"
	"math/rand/v2"
	"testing"
)

func kzPoissonT(r *rand.Rand, lam float64) int {
	if lam > 30 {
		v := int(math.Round(lam + math.Sqrt(lam)*r.NormFloat64()))
		if v < 0 {
			v = 0
		}
		return v
	}
	l, k, p := math.Exp(-lam), 0, 1.0
	for {
		p *= r.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

// synthetic rung data: n(tau) = f(tau) defects per cell, R replicates, Poisson counts.
func kzSynthRung(r *rand.Rand, tau, R, cells int, n float64, ambigFrac float64) KZRungData {
	d := KZRungData{Tau: tau, Cells: cells, Defects: make([]int, R), Ambiguous: make([]int, R), Neg: make([]int, R)}
	for j := 0; j < R; j++ {
		d.Defects[j] = kzPoissonT(r, n*float64(cells))
		d.Ambiguous[j] = int(math.Round(ambigFrac * float64(cells)))
		d.Neg[j] = d.Defects[j] + d.Ambiguous[j]/2
	}
	return d
}

func kzSynthArm(seed uint64, R, cells int, f func(tau int) float64, ambig func(tau int) float64) []KZRungData {
	r := kzTestRand(seed)
	out := make([]KZRungData, len(KZLadder))
	for i, tau := range KZLadder {
		a := 0.0
		if ambig != nil {
			a = ambig(tau)
		}
		out[i] = kzSynthRung(r, tau, R, cells, f(tau), a)
	}
	return out
}

func kzPowerLaw(a0, alpha float64) func(int) float64 {
	return func(tau int) float64 { return a0 * math.Pow(float64(tau), -alpha) }
}

func kzZeroArm(R, cells int) []KZRungData {
	out := make([]KZRungData, len(KZLadder))
	for i, tau := range KZLadder {
		out[i] = KZRungData{Tau: tau, Cells: cells, Defects: make([]int, R), Ambiguous: make([]int, R), Neg: make([]int, R)}
	}
	return out
}

func TestKZFitRecoversPowerLawAndCurvature(t *testing.T) {
	x := make([]float64, 7)
	y := make([]float64, 7)
	for i, tau := range KZLadder {
		x[i] = math.Log10(float64(tau))
		y[i] = math.Log10(0.2 * math.Pow(float64(tau), -0.75))
	}
	f := KZFitLogLog(x, y)
	if math.Abs(f.Slope+0.75) > 1e-9 || math.Abs(f.QuadC) > 1e-9 {
		t.Fatalf("pure power law: slope %v quad %v", f.Slope, f.QuadC)
	}
	if f.DeltaAIC < -2 {
		t.Fatalf("pure power law must not prefer the quadratic: dAIC = %v", f.DeltaAIC)
	}
	for i := range y {
		y[i] += 0.15 * (x[i] - 2.0) * (x[i] - 2.0)
	}
	f = KZFitLogLog(x, y)
	if math.Abs(f.QuadC-0.15) > 1e-9 {
		t.Fatalf("quad coefficient %v, want 0.15", f.QuadC)
	}
	if f.DeltaAIC > -2 {
		t.Fatalf("strong curvature must give dAIC < -2, got %v", f.DeltaAIC)
	}
}

func TestKZRungStatistics(t *testing.T) {
	d := KZRungData{Tau: 100, Cells: 1000, Defects: []int{10, 20, 30, 40}, Ambiguous: []int{5, 5, 5, 5}, Neg: []int{12, 22, 32, 42}}
	s := KZRungStats(d, 12345)
	if math.Abs(s.N-0.025) > 1e-12 {
		t.Fatalf("n = %v, want 0.025", s.N)
	}
	if s.CILo > s.N || s.CIHi < s.N || s.CILo <= 0.010 || s.CIHi >= 0.040 {
		t.Fatalf("CI [%v, %v] around %v is implausible", s.CILo, s.CIHi, s.N)
	}
	if s.PooledDefects != 100 {
		t.Fatalf("pooled defects %d", s.PooledDefects)
	}
	if math.Abs(s.AmbigFrac-0.005) > 1e-12 {
		t.Fatalf("ambiguous fraction %v", s.AmbigFrac)
	}
	s2 := KZRungStats(d, 12345)
	if s != s2 {
		t.Fatalf("bootstrap must be deterministic")
	}
	// identical replicates: a degenerate CI
	c := KZRungData{Tau: 50, Cells: 100, Defects: []int{5, 5, 5}, Ambiguous: []int{0, 0, 0}, Neg: []int{5, 5, 5}}
	if cs := KZRungStats(c, 1); cs.CILo != cs.N || cs.CIHi != cs.N {
		t.Fatalf("degenerate CI expected: %+v", cs)
	}
}

func TestKZAnalyzePowerLawYes(t *testing.T) {
	const R, cells = 50, 4096
	main := kzSynthArm(12400, R, cells, kzPowerLaw(0.2, 0.75), nil)
	null := kzZeroArm(R, cells)
	d := KZAnalyze(main, null)
	if d.Q5a != "yes" || d.Q5b != "yes" || d.Q5c != "yes" || d.NullFires {
		t.Fatalf("a clean alpha = 0.75 power law must read yes/yes/yes: %+v", d)
	}
	if d.AlphaLo > 0.75 || d.AlphaHi < 0.75 {
		t.Fatalf("alpha CI [%v, %v] must contain the true 0.75 (alpha = %v)", d.AlphaLo, d.AlphaHi, d.Alpha)
	}
	if len(d.Used) != 7 {
		t.Fatalf("all 7 rungs usable, got %d", len(d.Used))
	}
}

func TestKZAnalyzeBandAndNoSlope(t *testing.T) {
	const R, cells = 50, 4096
	shallow := KZAnalyze(kzSynthArm(12410, R, cells, kzPowerLaw(0.05, 0.3), nil), kzZeroArm(R, cells))
	if shallow.Q5a != "yes" || shallow.Q5b != "no" {
		t.Fatalf("alpha 0.3 is a power law outside the band: %+v", shallow)
	}
	flat := KZAnalyze(kzSynthArm(12411, R, cells, func(int) float64 { return 0.01 }, nil), kzZeroArm(R, cells))
	if flat.Q5a != "no" {
		t.Fatalf("a flat n(tau) must not be Q5a yes (CI on alpha includes 0): %+v", flat)
	}
	steep := KZAnalyze(kzSynthArm(12412, R, cells, kzPowerLaw(2.0, 1.3), nil), kzZeroArm(R, cells))
	if steep.Q5b != "no" {
		t.Fatalf("alpha 1.3 is outside [0.6, 1.0]: %+v", steep)
	}
}

func TestKZAnalyzeCurvature(t *testing.T) {
	const R, cells = 100, 8192
	curved := func(tau int) float64 {
		x := math.Log10(float64(tau))
		return math.Pow(10, -1.5-0.5*x+0.35*(x-2.2)*(x-2.2))
	}
	d := KZAnalyze(kzSynthArm(12420, R, cells, curved, nil), kzZeroArm(R, cells))
	if d.Q5a != "no" {
		t.Fatalf("a strongly curved log-log relation must be Q5a no: %+v", d)
	}
}

func TestKZAnalyzeDropsAndInconclusives(t *testing.T) {
	const R, cells = 50, 4096
	// fastest rung 10% ambiguous: dropped (resolution), span 50..1600 = 1.505 decades: still yes
	amb := func(tau int) float64 {
		if tau == 25 {
			return 0.10
		}
		return 0.0
	}
	d := KZAnalyze(kzSynthArm(12430, R, cells, kzPowerLaw(0.2, 0.75), amb), kzZeroArm(R, cells))
	if len(d.Used) != 6 || d.SpanDecades < 1.5 || math.Abs(d.Alpha-0.75) > 0.1 {
		t.Fatalf("dropping the fastest rung must leave 6 usable rungs spanning >= 1.5 decades with alpha near 0.75: %+v", d)
	}
	if d.Q5a == "inconclusive (span)" || d.Q5a == "inconclusive (resolution)" {
		t.Fatalf("unexpected inconclusive: %q", d.Q5a)
	}
	if d.RungStatus[0] != "inconclusive (resolution)" {
		t.Fatalf("rung 25 status %q", d.RungStatus[0])
	}
	// both ends ambiguous: span 100..800 < 1.5 decades
	amb2 := func(tau int) float64 {
		if tau == 25 || tau == 50 || tau == 1600 {
			return 0.10
		}
		return 0
	}
	d = KZAnalyze(kzSynthArm(12431, R, cells, kzPowerLaw(0.2, 0.75), amb2), kzZeroArm(R, cells))
	if d.Q5a != "inconclusive (span)" || d.Q5b != "inconclusive (span)" {
		t.Fatalf("expected inconclusive (span): %+v", d)
	}
	// slow rungs under the 20-defect pooled floor: finite size
	d = KZAnalyze(kzSynthArm(12432, 25, 512, kzPowerLaw(0.02, 0.75), nil), kzZeroArm(25, 512))
	for i, st := range d.RungStatus {
		if KZLadder[i] >= 100 && st != "inconclusive (finite size)" {
			t.Fatalf("rung %d status %q, want finite size", KZLadder[i], st)
		}
	}
	if d.Q5a != "inconclusive (span)" {
		t.Fatalf("after losing the slow rungs the span is short: %q", d.Q5a)
	}
	// ambiguity exactly at the 5% boundary: under 5% is fine, 5% and over is not
	atFive := func(tau int) float64 { return 0.05 }
	d = KZAnalyze(kzSynthArm(12433, R, cells, kzPowerLaw(0.2, 0.75), atFive), kzZeroArm(R, cells))
	if len(d.Used) != 0 {
		t.Fatalf("ambiguity of exactly 5%% is not 'fewer than 5%%': %d rungs used", len(d.Used))
	}
}

func TestKZAnalyzeGuardNeedsFourRungs(t *testing.T) {
	const R, cells = 50, 4096
	// only rungs 50, 400 and 1600 usable would span 1.5 decades with 3 points: the quadratic
	// is exact there, so the harness refuses (HARNESS guard) instead of reading curvature.
	amb := func(tau int) float64 {
		if tau == 25 || tau == 100 || tau == 200 || tau == 800 {
			return 0.10
		}
		return 0
	}
	d := KZAnalyze(kzSynthArm(12440, R, cells, kzPowerLaw(0.2, 0.75), amb), kzZeroArm(R, cells))
	if d.Q5a != "inconclusive (span)" {
		t.Fatalf("3 usable rungs must be inconclusive (span): %+v", d)
	}
}

func TestKZNullKills(t *testing.T) {
	const R, cells = 50, 4096
	main := kzSynthArm(12450, R, cells, kzPowerLaw(0.2, 0.75), nil)
	// (1) a single non-ambiguous -1 on any null cell invalidates the round
	null := kzZeroArm(R, cells)
	null[3].Defects[7] = 1
	d := KZAnalyze(main, null)
	if !d.NullFires || d.Q5a != "inconclusive (detector)" || d.Q5b != "inconclusive (detector)" {
		t.Fatalf("a null -1 must invalidate: %+v", d)
	}
	// (2) null n above 10% of main n with non-overlapping CIs at a used rung
	nullHigh := kzSynthArm(12451, R, cells, kzPowerLaw(0.1, 0.75), nil) // 50% of main
	d = KZAnalyze(main, nullHigh)
	if !d.NullFires || d.Q5c == "yes" {
		t.Fatalf("a null at 50%% of main must fire and not be Q5c yes: %+v", d)
	}
	// (3) clean zero null never fires
	if d = KZAnalyze(main, kzZeroArm(R, cells)); d.NullFires {
		t.Fatalf("zero null fired")
	}
	// (4) a null at a few percent of main does not fire the ratio clause when its CI sits
	// under 10% of main's CI... but any null -1 still fires clause 2 (per cell), tested in (1)
}

func TestKZAnalyzeQ5cNeedsSeparation(t *testing.T) {
	const R, cells = 50, 4096
	main := kzSynthArm(12460, R, cells, kzPowerLaw(0.2, 0.75), nil)
	// null defects present only as ambiguous cells (no -1 on unambiguous cells): zero defects
	null := kzZeroArm(R, cells)
	for i := range null {
		for j := range null[i].Ambiguous {
			null[i].Ambiguous[j] = cells / 2
			null[i].Neg[j] = 7 // raw -1s inside ambiguous cells do not count as defects
		}
	}
	d := KZAnalyze(main, null)
	if d.NullFires {
		t.Fatalf("raw -1 inside ambiguous cells must not fire the null kill: %+v", d)
	}
	if d.Q5c != "yes" {
		t.Fatalf("Q5c should be yes against a zero-defect null: %+v", d)
	}
}
