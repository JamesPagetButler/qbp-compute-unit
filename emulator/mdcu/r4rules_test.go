package mdcu

// R4 stage 3 tests (frozen R4 prereg §1.3–§1.5, §3): the verdict functions on
// synthetic data with known answers, covering every row of the reading table.
//
// The data are fabricated R3Run records (Class and S3 only); nothing here runs
// the dynamics and no reserved R4 seed is simulated.

import (
	"math"
	"strings"
	"testing"
)

func seedRange(lo, hi uint64) []uint64 {
	var s []uint64
	for x := lo; x <= hi; x++ {
		s = append(s, x)
	}
	return s
}

// lockPct locks the first pct% (by seed index mod 100) of seeds: nested patterns.
func lockPct(pct int) func(uint64) bool {
	return func(seed uint64) bool { return int(seed-R4GridLo)%100 < pct }
}

func synthArm(seeds []uint64, arm R3Arm, cell string, lock func(uint64) bool) []R3Run {
	var out []R3Run
	for _, s := range seeds {
		r := R3Run{Cell: cell, N: 24, Seed: s, Stratum: R3Stratum(s), Arm: arm}
		if lock(s) {
			r.Stats.Class, r.Stats.S3 = "3-lock", 0.98
		} else {
			r.Stats.Class, r.Stats.S3 = "diffuse", 0.60
		}
		out = append(out, r)
	}
	return out
}

func newSynth() *R4Data {
	return &R4Data{Seeds: map[uint64]R4Seed{}, Runs: map[string]map[R3Arm][]R3Run{}}
}

func (d *R4Data) put(runs []R3Run) {
	for _, r := range runs {
		d.Add(r)
	}
}

var grid200 = seedRange(R4GridLo, R4GridHi)

func TestR4Gate(t *testing.T) {
	for _, c := range []struct {
		pct  int
		pass bool
	}{{100, true}, {90, true}, {89, false}, {0, false}} {
		d := newSynth()
		d.put(synthArm(grid200, ArmFull, "A250", lockPct(c.pct)))
		g, rate := R4Gate(d)
		if g != c.pass || math.Abs(rate-float64(c.pct)/100) > 1e-9 {
			t.Fatalf("K-full %d%%: gate %v rate %g", c.pct, g, rate)
		}
	}
}

func TestR4Q4a(t *testing.T) {
	mk := func(p250, p1000, p4000 int, n4000 int) *R4Data {
		d := newSynth()
		d.put(synthArm(grid200, ArmPhi, "A250", lockPct(p250)))
		d.put(synthArm(grid200, ArmPhi, "B1000", lockPct(p1000)))
		d.put(synthArm(grid200[:n4000], ArmPhi, "C4000", lockPct(p4000)))
		return d
	}
	cases := []struct {
		name    string
		d       *R4Data
		verdict string
		rstar   int
	}{
		{"locks slowly, R*=1000", mk(5, 60, 90, 200), "locks slowly", 1000},
		{"locks at 250", mk(55, 80, 90, 200), "locks slowly", 250},
		{"locks only at 4000", mk(0, 20, 50, 200), "locks slowly", 4000},
		{"does not lock", mk(0, 2, 9, 200), "does not lock by 4000", 0},
		{"partial (10% is not below 10%)", mk(0, 5, 10, 200), "partial", 0},
		{"partial (49%)", mk(0, 20, 49, 200), "partial", 0},
		{"fallback 100 seeds is enough", mk(0, 20, 60, 100), "locks slowly", 4000},
		{"below 100 seeds is underpowered", mk(0, 20, 60, 99), "inconclusive (power)", 0},
		{"no tier C at all", mk(0, 20, 60, 0), "inconclusive (power)", 0},
	}
	for _, c := range cases {
		r := R4Q4a(c.d)
		if r.Verdict != c.verdict || r.RStar != c.rstar {
			t.Fatalf("%s: got %q R*=%d, want %q R*=%d (%+v)", c.name, r.Verdict, r.RStar, c.verdict, c.rstar, r.Rungs)
		}
	}
}

func q4bData(phi, phiK, psi1, psiK int) *R4Data {
	d := newSynth()
	d.put(synthArm(grid200, ArmPhi, "A250", lockPct(phi)))
	d.put(synthArm(grid200, ArmPhiK, "A250", lockPct(phiK)))
	d.put(synthArm(grid200, ArmPsi1, "A250", lockPct(psi1)))
	d.put(synthArm(grid200, ArmPsiK, "A250", lockPct(psiK)))
	return d
}

func TestR4Q4b(t *testing.T) {
	cases := []struct {
		name                  string
		phi, phiK, psi1, psiK int
		verdict               string
		floor                 int // expected number of floor-equivalent pairs
	}{
		{"COUPLING: gap closes at matched coupling", 20, 90, 20, 90, "COUPLING", 0},
		{"COUPLING with a floor pair (reported, not a rule)", 0, 90, 0, 90, "COUPLING", 1},
		{"FORM: psi locks where phi does not at both levels", 0, 0, 100, 100, "FORM", 0},
		{"MIXED", 0, 50, 50, 70, "MIXED", 0},
		{"MIXED: coupling closes the gap but phiK does not rise", 20, 20, 20, 20, "MIXED", 0},
	}
	for _, c := range cases {
		r := R4Q4b(q4bData(c.phi, c.phiK, c.psi1, c.psiK))
		if r.Verdict != c.verdict {
			t.Fatalf("%s: got %s (%+v)", c.name, r.Verdict, r.Flags)
		}
		if len(r.FloorPairs) != c.floor {
			t.Fatalf("%s: floor pairs %v, want %d", c.name, r.FloorPairs, c.floor)
		}
		if r.N != 200 || r.Rates[ArmPsi1] != float64(c.psi1)/100 {
			t.Fatalf("%s: N=%d rates=%v", c.name, r.N, r.Rates)
		}
		if len(r.RatesByStratum["definite"]) != 4 || len(r.RatesByStratum["split"]) != 4 {
			t.Fatalf("%s: strata not reported: %v", c.name, r.RatesByStratum)
		}
	}
	// Power: fewer than 100 paired seeds.
	d := newSynth()
	for _, a := range []R3Arm{ArmPhi, ArmPhiK, ArmPsi1, ArmPsiK} {
		d.put(synthArm(grid200[:99], a, "A250", lockPct(50)))
	}
	if v := R4Q4b(d).Verdict; v != "inconclusive (power)" {
		t.Fatalf("99 pairs: %s", v)
	}
}

// Dropped seeds (unconverged comass) shrink only the psi-arm pairs.
func TestR4Q4bUsesPairedSeeds(t *testing.T) {
	d := q4bData(20, 90, 20, 90)
	keep := func(rs []R3Run) []R3Run { return rs[:len(rs)-10] }
	d.Runs["A250"][ArmPsi1] = keep(d.Runs["A250"][ArmPsi1])
	d.Runs["A250"][ArmPsiK] = keep(d.Runs["A250"][ArmPsiK])
	r := R4Q4b(d)
	if r.Verdict != "COUPLING" {
		t.Fatalf("verdict %s", r.Verdict)
	}
	if r.Contrasts["Δ(K-psi@1, K-phi)"].N != 190 || r.Contrasts["Δ(K-phi@κ, K-phi)"].N != 200 {
		t.Fatalf("paired counts: %v", r.Contrasts)
	}
}

func q4cData(cell string, phiArm, psiArm R3Arm, phiPct, psiPct int, n int) []R3Run {
	return append(synthArm(grid200[:n], phiArm, cell, lockPct(phiPct)), synthArm(grid200[:n], psiArm, cell, lockPct(psiPct))...)
}

func TestR4Q4c(t *testing.T) {
	// first qualifying cell and the R3 Q1b rule on it
	d := newSynth()
	d.put(q4cData("A250", ArmPhi, ArmPsi1, 100, 100, 200)) // (1,250): phi locks, psi equal  -> no
	for _, st := range []string{"definite", "split"} {
		r := R4Q4c(d, st)
		if r.Cell != "(1,250)" || r.Verdict != "no" {
			t.Fatalf("%s: %+v", st, r)
		}
	}
	// (1,250) does not lock -> (κ,250) is evaluated, with K-phi@κ vs K-psi@κ
	d = newSynth()
	d.put(q4cData("A250", ArmPhi, ArmPsi1, 5, 5, 200))
	d.put(q4cData("A250", ArmPhiK, ArmPsiK, 100, 60, 200)) // φ above ψ by 40 points -> yes
	if r := R4Q4c(d, "definite"); r.Cell != "(κ,250)" || r.Verdict != "yes" {
		t.Fatalf("κ,250: %+v", r)
	}
	// reversed: φ only just qualifies (50%); ψ locks everywhere with higher S3, so the
	// paired median of S3(φ) − S3(ψ) is ≤ −0.15 with a CI below 0
	d = newSynth()
	phiRuns := synthArm(grid200, ArmPhi, "A250", lockPct(50))
	for i := range phiRuns {
		if phiRuns[i].Stats.Class == "3-lock" {
			phiRuns[i].Stats.S3 = 0.96
		}
	}
	d.put(phiRuns)
	d.put(synthArm(grid200, ArmPsi1, "A250", lockPct(100)))
	if r := R4Q4c(d, "split"); r.Verdict != "yes-reversed" {
		t.Fatalf("reversed: %+v", r)
	}
	// later cells in the fixed order
	d = newSynth()
	d.put(q4cData("A250", ArmPhi, ArmPsi1, 0, 0, 200))
	d.put(q4cData("A250", ArmPhiK, ArmPsiK, 0, 0, 200))
	d.put(q4cData("B1000", ArmPhi, ArmPsi1, 55, 55, 200))
	d.put(q4cData("B1000", ArmPhiK, ArmPsiK, 90, 90, 200))
	if r := R4Q4c(d, "definite"); r.Cell != "(1,1000)" {
		t.Fatalf("order: %+v", r)
	}
	d = newSynth()
	d.put(q4cData("B1000", ArmPhiK, ArmPsiK, 90, 90, 200)) // (1,1000) absent/non-locking, (κ,1000) qualifies
	d.put(q4cData("B1000", ArmPhi, ArmPsi1, 10, 10, 200))
	if r := R4Q4c(d, "definite"); r.Cell != "(κ,1000)" {
		t.Fatalf("(κ,1000): %+v", r)
	}
	d = newSynth()
	d.put(q4cData("C4000", ArmPhi, ArmPsi1, 80, 80, 200))
	if r := R4Q4c(d, "definite"); r.Cell != "(1,4000)" {
		t.Fatalf("(1,4000): %+v", r)
	}
	// never locks
	d = newSynth()
	d.put(q4cData("A250", ArmPhi, ArmPsi1, 3, 3, 200))
	if r := R4Q4c(d, "definite"); r.Verdict != "inconclusive (φ never locks)" {
		t.Fatalf("never: %+v", r)
	}
	// underpowered stratum: 49 paired seeds in the stratum
	d = newSynth()
	d.put(q4cData("A250", ArmPhi, ArmPsi1, 100, 100, 98))
	if r := R4Q4c(d, "definite"); r.Verdict != "inconclusive (underpowered)" {
		t.Fatalf("power: %+v", r)
	}
}

func TestR4ReadingTable(t *testing.T) {
	cases := []struct {
		gate          bool
		q4b, def, spl string
		wantSub       string
	}{
		{true, "COUPLING", "no", "no", "not φ-specific"},
		{true, "COUPLING", "yes", "yes", "φ locks more than ψ"},
		{true, "COUPLING", "yes", "no", "Recorded as-is. No claim."},
		{true, "COUPLING", "inconclusive", "no", "Recorded as-is. No claim."},
		{true, "COUPLING", "yes-reversed", "yes-reversed", "Recorded as-is. No claim."},
		{true, "FORM", "no", "no", "not coupling scale"},
		{true, "FORM", "yes", "yes", "not coupling scale"},
		{true, "MIXED", "no", "no", "Recorded as-is. No claim."},
		{false, "COUPLING", "no", "no", "Inconclusive (non-replication)"},
		{true, "inconclusive (power)", "no", "no", "Inconclusive (power)"},
		{true, "inconclusive (comass)", "no", "no", "Inconclusive (comass)"},
		{true, "inconclusive (sampler)", "no", "no", "Inconclusive (sampler)"},
	}
	for _, c := range cases {
		got := R4Reading(c.gate, c.q4b, c.def, c.spl)
		if !strings.Contains(got, c.wantSub) {
			t.Fatalf("gate=%v %s/%s/%s: %q does not contain %q", c.gate, c.q4b, c.def, c.spl, got, c.wantSub)
		}
	}
	// The two COUPLING rows must differ.
	if R4Reading(true, "COUPLING", "no", "no") == R4Reading(true, "COUPLING", "yes", "yes") {
		t.Fatal("reading rows collapsed")
	}
}

func TestR4MechanismKills(t *testing.T) {
	eq := ContrastStat{N: 200, RateA: 0.80, RateB: 0.80, Median: 0, Lo: -0.01, Hi: 0.01}
	neq := ContrastStat{N: 200, RateA: 0.30, RateB: 0.80, Median: -0.2, Lo: -0.3, Hi: -0.1}
	hi := ContrastStat{N: 200, RateA: 0.80, RateB: 0.30, Median: 0.2, Lo: 0.1, Hi: 0.3}
	nonHaar := SponResult{N: 100, P: 0.001}
	haar := SponResult{N: 100, P: 0.30}
	few := SponResult{N: 10, P: 0.0}

	// Kill FORM: phi@κ ≡ psi@κ AND phi@κ non-Haar (p<0.01, ≥20 locked)
	if m := R4Mechanism(eq, hi, nonHaar, nonHaar); !m.KillForm {
		t.Fatal("kill FORM should fire")
	}
	if m := R4Mechanism(eq, hi, few, nonHaar); m.KillForm {
		t.Fatal("kill FORM needs ≥ 20 locked runs")
	}
	if m := R4Mechanism(eq, hi, haar, nonHaar); m.KillForm {
		t.Fatal("kill FORM needs p < 0.01")
	}
	if m := R4Mechanism(neq, hi, nonHaar, nonHaar); m.KillForm {
		t.Fatal("kill FORM needs equivalence")
	}
	// Kill COUPLING-ONLY (a): phi@κ lags psi@1 (not equivalent and lower)
	if m := R4Mechanism(eq, neq, few, few); !m.KillCouplingOnly {
		t.Fatal("kill COUPLING-ONLY (lag) should fire")
	}
	if m := R4Mechanism(eq, hi, few, few); m.KillCouplingOnly {
		t.Fatal("phi@κ above psi@1 is not a lag")
	}
	// (b): phi@κ Haar-uniform while psi@1 clustered
	if m := R4Mechanism(eq, eq, haar, nonHaar); !m.KillCouplingOnly {
		t.Fatal("kill COUPLING-ONLY (Haar vs clustered) should fire")
	}
	if m := R4Mechanism(eq, eq, haar, haar); m.KillCouplingOnly {
		t.Fatal("psi@1 must be clustered (p<0.01)")
	}
	if m := R4Mechanism(eq, eq, SponResult{N: 19, P: 0.9}, nonHaar); m.KillCouplingOnly {
		t.Fatal("(b) needs ≥ 20 locked phi@κ runs")
	}
	// neither fires -> undetermined
	if m := R4Mechanism(eq, eq, few, few); m.KillForm || m.KillCouplingOnly || !strings.Contains(m.Reading, "undetermined") {
		t.Fatalf("neither: %+v", m)
	}
}

func TestR4SpontaneityHypotheses(t *testing.T) {
	ok := func(p float64) SponResult { return SponResult{N: 50, P: p, Testable: true} }
	un := SponResult{N: 5, P: 0.5}
	// H-A1
	if h := R4SA(ok(0.545), nil, nil); h.A1 != "holds" {
		t.Fatalf("A1 %s", h.A1)
	}
	if h := R4SA(ok(0.005), nil, nil); h.A1 != "killed" {
		t.Fatalf("A1 %s", h.A1)
	}
	if h := R4SA(ok(0.03), nil, nil); h.A1 != "undetermined" {
		t.Fatalf("A1 %s", h.A1)
	}
	if h := R4SA(un, nil, nil); h.A1 != "untestable" {
		t.Fatalf("A1 %s", h.A1)
	}
	// H-A2: every ψ arm with ≥ 20 locked differs from Haar (p<0.01); kill: any p ≥ 0.05
	if h := R4SA(un, []SponResult{ok(0.001), ok(0.0), un}, nil); h.A2 != "holds" {
		t.Fatalf("A2 %s", h.A2)
	}
	if h := R4SA(un, []SponResult{ok(0.001), ok(0.06)}, nil); h.A2 != "killed" {
		t.Fatalf("A2 %s", h.A2)
	}
	if h := R4SA(un, []SponResult{ok(0.001), ok(0.03)}, nil); h.A2 != "undetermined" {
		t.Fatalf("A2 %s", h.A2)
	}
	if h := R4SA(un, []SponResult{un}, nil); h.A2 != "untestable" {
		t.Fatalf("A2 %s", h.A2)
	}
	// H-A3: any φ arm locking in ≥ 20 runs is Haar-consistent (p ≥ 0.05); kill p < 0.01
	if h := R4SA(un, nil, []SponResult{ok(0.4), un}); h.A3 != "holds" {
		t.Fatalf("A3 %s", h.A3)
	}
	if h := R4SA(un, nil, []SponResult{ok(0.4), ok(0.005)}); h.A3 != "killed" {
		t.Fatalf("A3 %s", h.A3)
	}
	if h := R4SA(un, nil, []SponResult{ok(0.03)}); h.A3 != "undetermined" {
		t.Fatalf("A3 %s", h.A3)
	}
	if h := R4SA(un, nil, nil); h.A3 != "untestable" {
		t.Fatalf("A3 %s", h.A3)
	}
}

func TestR4SB(t *testing.T) {
	// verdicts from (rho, CI)
	cases := []struct {
		rho, lo, hi float64
		want        string
	}{
		{-0.258, -0.40, -0.10, "holds"},    // |ρ| ≥ 0.20, CI excludes 0, same sign as R3 (negative)
		{0.30, 0.10, 0.45, "undetermined"}, // |ρ| ≥ 0.20, CI excludes 0 but opposite sign to R3
		{-0.05, -0.20, 0.10, "killed"},     // CI includes 0 and |ρ| < 0.10
		{-0.15, -0.30, 0.05, "undetermined"},
		{-0.19, -0.30, -0.05, "undetermined"}, // CI excludes 0 but |ρ| < 0.20
	}
	for _, c := range cases {
		if got := R4SBVerdict(c.rho, c.lo, c.hi); got != c.want {
			t.Fatalf("ρ=%g CI[%g,%g]: %s, want %s", c.rho, c.lo, c.hi, got, c.want)
		}
	}
	// bootstrap: a perfectly monotone relation has ρ = −1 and a CI above −1.0001
	x := make([]float64, 100)
	y := make([]float64, 100)
	for i := range x {
		x[i], y[i] = float64(100-i), float64(i)
	}
	rho, lo, hi, n := R4SBStat("t", x, y)
	if n != 100 || math.Abs(rho+1) > 1e-12 || lo > rho+1e-12 || hi < rho-1e-12 {
		t.Fatalf("ρ=%g [%g,%g] n=%d", rho, lo, hi, n)
	}
}

func TestR4SC(t *testing.T) {
	mk := func(full, gram, phiK int) *R4Data {
		d := newSynth()
		d.put(synthArm(grid200, ArmFullK, "A250", lockPct(full)))
		d.put(synthArm(grid200, ArmGramK, "A250", lockPct(gram)))
		d.put(synthArm(grid200, ArmPhiK, "A250", lockPct(phiK)))
		return d
	}
	if s := R4SC(mk(95, 10, 20)); s != "holds" {
		t.Fatalf("holds: %s", s)
	}
	for _, d := range []*R4Data{mk(95, 60, 20), mk(95, 10, 60), mk(80, 10, 20), mk(95, 50, 20)} {
		if s := R4SC(d); !strings.Contains(s, "(a) fails at matched coupling: a single term locks") {
			t.Fatalf("fails: %s", s)
		}
	}
}

func TestR4SDKill(t *testing.T) {
	mk := func(psiPct, phiPct int) *R4Data {
		d := newSynth()
		sds := seedRange(R4SDLo, R4SDLo+49) // the s = 0.005 rung
		for _, s := range sds {
			d.Seeds[s] = R4Seed{Seed: s, Kind: "sd", S2S: f64(0.005)}
		}
		lock := func(pct int) func(uint64) bool { return func(s uint64) bool { return int(s-R4SDLo) < pct/2 } }
		d.put(synthArm(sds, ArmPsi1, "SD250", lock(psiPct)))
		d.put(synthArm(sds, ArmPhi, "SD250", lock(phiPct)))
		d.put(synthArm(sds, ArmPsi1, "SD1000", lock(psiPct)))
		d.put(synthArm(sds, ArmPhi, "SD1000", lock(phiPct)))
		return d
	}
	if r := R4SD(mk(100, 20)); !r.Killed {
		t.Fatalf("jump of 80 points at s=0.005 must kill H-D: %+v", r)
	}
	if r := R4SD(mk(100, 50)); !r.Killed {
		t.Fatalf("exactly 50 points kills: %+v", r)
	}
	if r := R4SD(mk(60, 20)); r.Killed {
		t.Fatalf("40 points must not kill: %+v", r)
	}
	r := R4SD(mk(60, 20))
	if len(r.Rungs) == 0 || r.Rungs[0].S != 0.005 || r.Rungs[0].N != 50 {
		t.Fatalf("rungs: %+v", r.Rungs)
	}
}

func f64(x float64) *float64 { return &x }

func TestR4KillConditions(t *testing.T) {
	seeds := func(missOdd, unconv, n int) map[uint64]R4Seed {
		m := map[uint64]R4Seed{}
		odd, k := 0, 0
		for s := uint64(R4GridLo); s < uint64(R4GridLo+n); s++ {
			sd := R4Seed{Seed: s, Kind: "grid", Stratum: R3Stratum(s), Converged: true}
			if s%2 == 1 {
				if odd < missOdd {
					sd.Miss, sd.Converged = true, false
				}
				odd++
			}
			if !sd.Miss {
				if k < unconv {
					sd.Converged = false
				}
				k++
			}
			m[s] = sd
		}
		return m
	}
	// 200 seeds: 100 odd. Sampler: > 5% of S1 calls. Comass: > 5% of seeds with a ψ.
	if k := R4Kills(seeds(5, 0, 200)); k.Sampler {
		t.Fatalf("5/100 misses is not above 5%%: %+v", k)
	}
	if k := R4Kills(seeds(6, 0, 200)); !k.Sampler {
		t.Fatalf("6/100 misses is above 5%%: %+v", k)
	}
	if k := R4Kills(seeds(0, 10, 200)); k.Comass {
		t.Fatalf("10/200 unconverged is exactly 5%%: %+v", k)
	}
	if k := R4Kills(seeds(0, 11, 200)); !k.Comass {
		t.Fatalf("11/200 unconverged is above 5%%: %+v", k)
	}
}
