package mdcu

// R4 stage 2 tests (frozen R4 prereg §4 "Tests first"): the run layer — seed
// builders, tier snapshots, the S-D builder, the Tier C guard, the seed-range
// guards, and the bit-identity regression to the R3 raw.
//
// All dynamics tests use out-of-grid seeds 9970–9979 (reserved R4 ranges:
// 4000–4199, 4200–4399, 9960–9969). The regression test deliberately reuses R3
// seeds 3000–3009: a code check, not data (frozen §1.3).

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"reflect"
	"testing"
)

const r3RawPath = "../../doc/spikes/raw/mdcu-r3-2026-10-26.jsonl"
const r3RawSHA = "633670f9b4134bf5add1a82a7c04aaffbf9e43b8e73a23a7ee45b67add0f76bc"

// indexOfRecord returns the byte offset of the first line of b containing every
// substring in subs, or -1.
func indexOfRecord(b []byte, subs ...string) int {
	off := 0
	for _, line := range bytes.SplitAfter(b, []byte("\n")) {
		ok := true
		for _, s := range subs {
			if !bytes.Contains(line, []byte(s)) {
				ok = false
				break
			}
		}
		if ok {
			return off
		}
		off += len(line)
	}
	return -1
}

func TestR4SeedRangeGuards(t *testing.T) {
	cases := []struct {
		seed                   uint64
		grid, sd, probe, resvd bool
	}{
		{3999, false, false, false, false},
		{4000, true, false, false, true},
		{4199, true, false, false, true},
		{4200, false, true, false, true},
		{4399, false, true, false, true},
		{4400, false, false, false, false},
		{9959, false, false, false, false},
		{9960, false, false, true, true},
		{9969, false, false, true, true},
		{9970, false, false, false, false},
	}
	for _, c := range cases {
		if R4InGrid(c.seed) != c.grid || R4InSD(c.seed) != c.sd || R4InProbe(c.seed) != c.probe || R4Reserved(c.seed) != c.resvd {
			t.Fatalf("seed %d: guards (%v %v %v %v), want (%v %v %v %v)", c.seed,
				R4InGrid(c.seed), R4InSD(c.seed), R4InProbe(c.seed), R4Reserved(c.seed), c.grid, c.sd, c.probe, c.resvd)
		}
	}
}

func TestR4SDScaleFor(t *testing.T) {
	for seed, want := range map[uint64]float64{4200: 0.005, 4249: 0.005, 4250: 0.01, 4299: 0.01, 4300: 0.02, 4349: 0.02, 4350: 0.05, 4399: 0.05} {
		if got := R4SDScaleFor(seed); got != want {
			t.Fatalf("seed %d: s = %g, want %g", seed, got, want)
		}
	}
}

// Tier C guard, as ruled by qbp-architecture (bridge seq 115): the frozen sentence
// "fall back if the measured wall time extrapolates past the cap" is implemented as
//
//	predicted total wall = Σ over tiers (frozen run count × probe-measured per-run
//	cost at that tier's step count), with Tier C at its full 200 seeds;
//	fall back to 100 Tier C seeds iff predicted > 3600 s × (1 − 0.10).
//
// The frozen run counts are §2: Tier A 10 arms × 200 seeds at 250 steps, Tier B
// 4 × 200 at 1000, Tier C 2 × 200 at 4000, S-D 200 seeds × 2 arms × (250, 1000).
func TestR4PredictedWallFormula(t *testing.T) {
	// The frozen §2 per-run assumptions (0.1 s, 0.4 s, 1.6 s).
	c := R4StepCost{A250: 0.1, B1000: 0.4, C4000: 1.6}
	// 2000×0.1 + 800×0.4 + 400×1.6 + 400×0.1 + 400×0.4 = 200 + 320 + 640 + 40 + 160
	if got := R4PredictedWall(c); math.Abs(got-1360) > 1e-9 {
		t.Fatalf("predicted wall %g s, want 1360", got)
	}
	// Linear in the per-run costs, and each term is separately weighted.
	if got := R4PredictedWall(R4StepCost{A250: 1}); math.Abs(got-2400) > 1e-9 { // 2000 + 400
		t.Fatalf("A250 weight: %g, want 2400", got)
	}
	if got := R4PredictedWall(R4StepCost{B1000: 1}); math.Abs(got-1200) > 1e-9 { // 800 + 400
		t.Fatalf("B1000 weight: %g, want 1200", got)
	}
	if got := R4PredictedWall(R4StepCost{C4000: 1}); math.Abs(got-400) > 1e-9 {
		t.Fatalf("C4000 weight: %g, want 400", got)
	}
}

func TestR4TierCGuard(t *testing.T) {
	base := R4StepCost{A250: 0.1, B1000: 0.4, C4000: 1.6} // predicted 1360 s
	scale := func(k float64) R4StepCost {
		return R4StepCost{A250: base.A250 * k, B1000: base.B1000 * k, C4000: base.C4000 * k}
	}
	// 3240 s = 3600 × 0.9 is the threshold: 1360·k = 3240 at k ≈ 2.38235.
	for _, c := range []struct {
		cost R4StepCost
		want int
	}{
		{scale(1), 200},
		{scale(2.38), 200}, // 3236.8 s: still fits
		{scale(2.39), 100}, // 3250.4 s: over the margin
		{scale(10), 100},
		{R4StepCost{}, 200}, // zero cost never falls back
	} {
		n, trig := R4TierCSeeds(c.cost)
		if n != c.want || trig != (c.want == 100) {
			t.Fatalf("predicted %g s: (%d, %v), want %d", R4PredictedWall(c.cost), n, trig, c.want)
		}
	}
	// The old numeric trigger (4 × t1000 > 2.5 s, i.e. t1000 > 0.625 s) must NOT decide:
	// at t1000 = 0.7 s with small other costs the plan fits, so no fallback.
	if n, _ := R4TierCSeeds(R4StepCost{A250: 0.05, B1000: 0.7, C4000: 0.7}); n != 200 {
		t.Fatalf("t1000 above 0.625 s alone must not trigger a fallback; got %d", n)
	}
	// The 4000-step cost matters: a cheap t1000 with an expensive t4000 falls back.
	if n, _ := R4TierCSeeds(R4StepCost{A250: 0.1, B1000: 0.4, C4000: 7}); n != 100 { // 200+320+2800+40+160 = 3520
		t.Fatalf("expensive 4000-step runs must trigger the fallback; got %d", n)
	}
	if R4WallCapSec != 3600 || R4WallMargin != 0.10 || R4TierCFallback != 100 || R4TierCFull != 200 {
		t.Fatal("guard constants differ from the ruling (seq 115) / frozen §2")
	}
}

func TestR4ProbeTimesMean(t *testing.T) {
	p := R4ProbeTimes{
		Per250:  []float64{0.1, 0.3},
		Per1000: []float64{1, 3},
		Per4000: []float64{4, 8},
	}
	m := p.Mean()
	if math.Abs(m.A250-0.2) > 1e-12 || math.Abs(m.B1000-2) > 1e-12 || math.Abs(m.C4000-6) > 1e-12 {
		t.Fatalf("means %+v", m)
	}
	if (R4ProbeTimes{}).Mean() != (R4StepCost{}) {
		t.Fatal("empty probe must give zero cost, not NaN")
	}
}

func TestR4ProbeSeedsAreOutOfGrid(t *testing.T) {
	ps := R4ProbeSeeds()
	if len(ps) != 10 || ps[0] != 9960 || ps[9] != 9969 {
		t.Fatalf("probe seeds %v", ps)
	}
}

// A sampler miss (S1-split cap) leaves Primary nil and the seed is counted.
func TestR4BuildSeedSamplerMiss(t *testing.T) {
	miss := func(uint64, string) (S1Draw, bool) { return S1Draw{Draws: 20000}, false }
	sd := buildR4Seed(9973, miss) // odd = S1 split
	if !sd.Miss || sd.Primary != nil {
		t.Fatalf("miss not recorded: %+v", sd)
	}
	if len(R4ArmsFor(sd)) != 7 { // K-psi, K-psi@1, K-psi@κ absent
		t.Fatalf("arms on a missed seed: %v", R4ArmsFor(sd))
	}
}

func TestR4BuildSeedEvenOdd(t *testing.T) {
	e := BuildR4Seed(9972)
	if e.Stratum != "definite" || e.Sampler != "S2" || e.Primary == nil || !e.Converged || e.Psi1 == nil || e.PsiK == nil {
		t.Fatalf("even seed: %+v", e)
	}
	o := BuildR4Seed(9973)
	if o.Stratum != "split" || o.Sampler != "S1" || o.Primary == nil || !o.Converged {
		t.Fatalf("odd seed: %+v", o)
	}
	// The R4 builder reproduces R3's Primary ψ exactly (same samplers, same streams).
	if BuildR3Psi(9972).Primary.C != e.Primary.C || BuildR3Psi(9973).Primary.C != o.Primary.C {
		t.Fatal("R4 Primary ψ differs from R3's")
	}
	if e.Comass < 1.0 || e.Comass > 3.0 || e.RelDiff > R4ComassTol {
		t.Fatalf("native comass %g (relDiff %g)", e.Comass, e.RelDiff)
	}
	if d := e.Factor1 - 1/e.Comass; d > 1e-15 || d < -1e-15 {
		t.Fatalf("Factor1 %g, want %g", e.Factor1, 1/e.Comass)
	}
	if d := e.FactorK - R4Kappa/e.Comass; d > 1e-15 || d < -1e-15 {
		t.Fatalf("FactorK %g, want %g", e.FactorK, R4Kappa/e.Comass)
	}
}

// Arms offered per seed: all ten Tier A arms; unconverged comass drops exactly
// the comass-rescaled ψ arms (K-psi@1, K-psi@κ).
func TestR4ArmsFor(t *testing.T) {
	sd := BuildR4Seed(9972)
	if got := R4ArmsFor(sd); !reflect.DeepEqual(got, R4TierAArms) {
		t.Fatalf("arms: %v", got)
	}
	sd.Converged = false
	for _, a := range R4ArmsFor(sd) {
		if a == ArmPsi1 || a == ArmPsiK {
			t.Fatalf("unconverged seed still runs %s", a)
		}
	}
	if len(R4ArmsFor(sd)) != 8 {
		t.Fatalf("unconverged seed: %v", R4ArmsFor(sd))
	}
}

// Chained snapshots equal separate runs bit-for-bit, at 250, 1000 and 4000 steps.
func TestR4ChainedSnapshotsEqualSeparateRuns(t *testing.T) {
	sd := BuildR4Seed(9972)
	runs := RunR4Seed(sd, true)
	z0 := Q1Init(24, 0.1, newRNG(9972))
	type key struct {
		arm   R3Arm
		steps int
	}
	got := map[key]R3Run{}
	for _, r := range runs {
		got[key{r.Arm, r.Steps}] = r
	}
	check := func(arm R3Arm, steps int, calib *Form3) {
		t.Helper()
		en := R4EnergyFor(arm, sd.Primary, sd.Psi1, sd.PsiK)
		want := R3Measure(RunRingEnergy(z0, en, steps), calib)
		r, ok := got[key{arm, steps}]
		if !ok {
			t.Fatalf("missing %s @ %d", arm, steps)
		}
		if !reflect.DeepEqual(r.Stats, want) {
			t.Fatalf("%s @ %d: chained != separate", arm, steps)
		}
	}
	check(ArmPhi, 250, sd.Primary)
	check(ArmPhi, 1000, sd.Primary)
	check(ArmPhi, 4000, sd.Primary)
	check(ArmPsi1, 250, sd.Psi1)
	check(ArmPsi1, 1000, sd.Psi1)
	check(ArmPsi1, 4000, sd.Psi1)
	check(ArmPsiK, 1000, sd.PsiK)
	check(ArmFullK, 250, sd.Primary)
}

// Record shape: cells A250 (10 arms), B1000 (4), C4000 (2 with tier C, 0 without).
func TestR4RunSeedShape(t *testing.T) {
	sd := BuildR4Seed(9974)
	count := func(runs []R3Run) map[string]int {
		m := map[string]int{}
		for _, r := range runs {
			m[r.Cell]++
			if r.N != 24 || r.Sigma != 0.1 || r.Seed != 9974 || r.Stratum != "definite" {
				t.Fatalf("bad record %+v", r)
			}
		}
		return m
	}
	with := count(RunR4Seed(sd, true))
	if with["A250"] != 10 || with["B1000"] != 4 || with["C4000"] != 2 {
		t.Fatalf("tier C on: %v", with)
	}
	without := count(RunR4Seed(sd, false))
	if without["A250"] != 10 || without["B1000"] != 4 || without["C4000"] != 0 {
		t.Fatalf("tier C off: %v", without)
	}
}

func TestR4SDSeed(t *testing.T) {
	sd := buildR4SD(9976, 0.02)
	if sd.Stratum != "definite" || sd.Sampler != "S2" || sd.S2S == nil || *sd.S2S != 0.02 || !sd.Converged || sd.Psi1 == nil {
		t.Fatalf("S-D seed: %+v", sd)
	}
	if sd.Primary.C != SampleS2At(9976, 0.02).Form.C {
		t.Fatal("S-D ψ is not SampleS2At at the fixed scale")
	}
	cells := map[string]map[R3Arm]bool{}
	for _, r := range RunR4SD(sd) {
		if cells[r.Cell] == nil {
			cells[r.Cell] = map[R3Arm]bool{}
		}
		cells[r.Cell][r.Arm] = true
	}
	for _, c := range []string{"SD250", "SD1000"} {
		if len(cells[c]) != 2 || !cells[c][ArmPsi1] || !cells[c][ArmPhi] {
			t.Fatalf("S-D cell %s: %v", c, cells[c])
		}
	}
	sd.Converged = false
	if len(RunR4SD(sd)) != 2 { // K-phi only, at two rungs
		t.Fatalf("unconverged S-D seed should run K-phi only")
	}
}

// The R4 harness reproduces R3's statistics bit for bit on seeds 3000–3009
// (K-full, K-phi, K-psi native; 250 steps). Frozen §1.3 regression check.
func TestR4BitIdentityRegression(t *testing.T) {
	b, err := os.ReadFile(r3RawPath)
	if err != nil {
		t.Skipf("R3 raw not available: %v", err)
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != r3RawSHA {
		t.Fatalf("R3 raw sha256 %x, want %s", h, r3RawSHA)
	}
	n, err := R4Regression(r3RawPath)
	if err != nil {
		t.Fatal(err)
	}
	if n != 30 {
		t.Fatalf("compared %d records, want 30 (10 seeds × 3 arms)", n)
	}
}

// The regression must be able to fail: a mutated stored statistic is caught.
func TestR4RegressionDetectsDrift(t *testing.T) {
	b, err := os.ReadFile(r3RawPath)
	if err != nil {
		t.Skipf("R3 raw not available: %v", err)
	}
	mut := []byte(string(b))
	// corrupt the last digit of S3 in the seed-3000 K-phi "main" record
	i := indexOfRecord(mut, `"Cell":"main"`, `"Seed":3000,`, `"Arm":"K-phi"`)
	if i < 0 {
		t.Fatal("record not found")
	}
	k := bytes.Index(mut[i:], []byte(`"S3":`))
	if k < 0 {
		t.Fatal("S3 not found")
	}
	end := i + k + bytes.IndexByte(mut[i+k:], ',')
	if mut[end-1] == '1' {
		mut[end-1] = '2'
	} else {
		mut[end-1] = '1'
	}
	tmp := t.TempDir() + "/mut.jsonl"
	if err := os.WriteFile(tmp, mut, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := R4Regression(tmp); err == nil {
		t.Fatal("regression missed a mutated statistic")
	}
}
