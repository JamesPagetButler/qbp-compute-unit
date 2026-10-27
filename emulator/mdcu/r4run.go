package mdcu

// Round-4 run layer (frozen R4 prereg §1.1–§1.3, §2, §3).
//
//   - BuildR4Seed / buildR4SD draw ψ with R3's samplers (S2 on even seeds, S1-split
//     on odd), compute the native comass with two independent restart sets, and
//     build the comass-1 and comass-κ rescales. Unlike R3 they do NOT draw the
//     S1-definite sensitivity form (K-psi-S1 is dropped in R4), which is the slow
//     part of BuildR3Psi on even seeds. The Primary ψ is bit-identical to R3's.
//   - RunR4Seed runs every applicable arm with chained snapshots at 250 / 1000 /
//     4000 steps. RunRingEnergy is stateless and normalises each step, so a
//     chained run equals a separate run bit for bit (TestR4ChainedSnapshotsEqual…).
//   - R4Regression is the frozen bit-identity check against the R3 raw.
//
// Process guards: R4Reserved reports whether a seed belongs to a pre-registered
// R4 range; the driver refuses those seeds without -run-grid / -run-probe.
// All tests and smoke runs use out-of-range seeds (9970–9979).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

// Tier C guard (frozen §2): probe K-phi at 1000 steps on seeds 9960–9969. If
// R4TierCFactor × (probe mean per-run seconds) > R4TierCCap, Tier C runs on the
// first R4TierCFallback seeds instead of all 200.
const (
	R4TierCCap      = 2.5
	R4TierCFactor   = 4.0
	R4TierCFallback = 100
	R4TierCFull     = 200
)

func R4InGrid(seed uint64) bool  { return seed >= R4GridLo && seed <= R4GridHi }
func R4InSD(seed uint64) bool    { return seed >= R4SDLo && seed <= R4SDHi }
func R4InProbe(seed uint64) bool { return seed >= R4ProbeLo && seed <= R4ProbeHi }

// R4Reserved reports whether seed lies in any pre-registered R4 range.
func R4Reserved(seed uint64) bool { return R4InGrid(seed) || R4InSD(seed) || R4InProbe(seed) }

// R4SDScaleFor is the S-D ladder scale of a seed: 50 consecutive seeds per rung.
func R4SDScaleFor(seed uint64) float64 {
	if !R4InSD(seed) {
		panic(fmt.Sprintf("mdcu: seed %d is not an S-D seed", seed))
	}
	return R4SDScales[(seed-R4SDLo)/R4SDPerScale]
}

// R4ProbeSeeds are the K = 10 out-of-grid timing-probe seeds.
func R4ProbeSeeds() []uint64 {
	var out []uint64
	for s := uint64(R4ProbeLo); s <= R4ProbeHi; s++ {
		out = append(out, s)
	}
	return out
}

// R4TierCSeeds applies the frozen guard to a probe mean (seconds per K-phi run at
// 1000 steps): the number of seeds Tier C runs on, and whether the guard fired.
func R4TierCSeeds(probeMeanSec float64) (n int, fallback bool) {
	if R4TierCFactor*probeMeanSec > R4TierCCap {
		return R4TierCFallback, true
	}
	return R4TierCFull, false
}

// R4TimingProbe times K-phi at R4StepsB steps (ring run plus measurement, calibrated
// under φ) on each seed and returns the mean seconds per run. It looks at no outcome.
func R4TimingProbe(seeds []uint64) (mean float64, per []float64) {
	var tot float64
	for _, s := range seeds {
		z0 := Q1Init(24, 0.1, newRNG(s))
		t0 := time.Now()
		Z := RunRingEnergy(z0, R4EnergyFor(ArmPhi, nil, nil, nil), R4StepsB)
		_ = R3Measure(Z, Phi())
		d := time.Since(t0).Seconds()
		per = append(per, d)
		tot += d
	}
	return tot / float64(len(seeds)), per
}

// R4Seed is everything the arms need for one seed, plus its per-seed record
// (frozen §1.2: native comass, rescale factors, two-set convergence difference,
// and on even seeds s and d).
type R4Seed struct {
	Seed         uint64   `json:"seed"`
	Kind         string   `json:"kind"` // "grid" | "sd"
	Stratum      string   `json:"stratum"`
	Sampler      string   `json:"sampler"` // "S2" | "S1"
	Miss         bool     `json:"miss"`    // sampler miss (odd seeds: S1-split cap)
	S1Draws      int      `json:"s1_draws"`
	S1Degenerate int      `json:"s1_degenerate"`
	S2S          *float64 `json:"s2_s,omitempty"`
	S2Dist       *float64 `json:"s2_dist,omitempty"`
	Comass       float64  `json:"comass"`   // native comass (larger of the two restart sets)
	ComassB      float64  `json:"comass_b"` // the other estimate
	RelDiff      float64  `json:"comass_reldiff"`
	Converged    bool     `json:"comass_converged"`
	Factor1      float64  `json:"factor_1"` // 1/comass
	FactorK      float64  `json:"factor_kappa"`

	Primary *Form3 `json:"-"` // native ψ (Frobenius 42); nil on a sampler miss
	Psi1    *Form3 `json:"-"` // comass 1; nil if unconverged
	PsiK    *Form3 `json:"-"` // comass κ; nil if unconverged
}

func (sd *R4Seed) applyComass(wantK bool) {
	if sd.Primary == nil {
		return
	}
	cc := ComassCheck(sd.Primary, sd.Seed)
	sd.Comass, sd.ComassB, sd.RelDiff, sd.Converged = cc.Comass, cc.ComassB, cc.RelDiff, cc.Converged
	if !cc.Converged || cc.Comass <= 0 {
		sd.Converged = false
		return
	}
	sd.Factor1 = 1 / cc.Comass
	sd.Psi1 = ScaleForm3(sd.Primary, 1/cc.Comass)
	if wantK {
		sd.FactorK = R4Kappa / cc.Comass
		sd.PsiK = ScaleForm3(sd.Primary, R4Kappa/cc.Comass)
	}
}

// buildR4Seed is BuildR4Seed with an injectable S1 sampler (for miss-rule tests).
func buildR4Seed(seed uint64, s1 func(uint64, string) (S1Draw, bool)) *R4Seed {
	sd := &R4Seed{Seed: seed, Kind: "grid", Stratum: R3Stratum(seed)}
	if sd.Stratum == "definite" {
		s2 := SampleS2(seed)
		sd.Sampler, sd.Primary = "S2", s2.Form
		sd.S2S, sd.S2Dist = &s2.S, &s2.Dist
	} else {
		sd.Sampler = "S1"
		d, ok := s1(seed, "split")
		sd.S1Draws, sd.S1Degenerate, sd.Miss = d.Draws, d.Degenerate, !ok
		if ok {
			sd.Primary = d.Form
		}
	}
	sd.applyComass(true)
	return sd
}

// BuildR4Seed draws the seed's ψ with the frozen R3 samplers and builds its levels.
func BuildR4Seed(seed uint64) *R4Seed { return buildR4Seed(seed, SampleS1) }

// buildR4SD builds an S-D seed: S2 at the fixed scale s (always definite), ψ rescaled
// to comass 1. Level κ is not used by S-D.
func buildR4SD(seed uint64, s float64) *R4Seed {
	d := SampleS2At(seed, s)
	sd := &R4Seed{Seed: seed, Kind: "sd", Stratum: "definite", Sampler: "S2", Primary: d.Form}
	sd.S2S, sd.S2Dist = &d.S, &d.Dist
	sd.applyComass(false)
	return sd
}

// BuildR4SD builds the S-D seed at its ladder scale.
func BuildR4SD(seed uint64) *R4Seed { return buildR4SD(seed, R4SDScaleFor(seed)) }

// BuildR4SDAt builds an S-D-style seed at an explicit scale. The driver's smoke mode
// uses it on out-of-range seeds; the grid uses BuildR4SD.
func BuildR4SDAt(seed uint64, s float64) *R4Seed { return buildR4SD(seed, s) }

// R4ArmsFor lists the Tier A arms that run on this seed, in reporting order. A
// sampler miss drops the ψ arms; an unconverged comass drops exactly the
// comass-rescaled ψ arms (K-psi@1, K-psi@κ), which are the arms whose form
// depends on the comass estimate. K-phi@κ, K-full@κ, K-gram@κ do not use it.
func R4ArmsFor(sd *R4Seed) []R3Arm {
	var out []R3Arm
	for _, a := range R4TierAArms {
		switch a {
		case ArmPsi:
			if sd.Primary == nil {
				continue
			}
		case ArmPsi1:
			if !sd.Converged || sd.Psi1 == nil {
				continue
			}
		case ArmPsiK:
			if !sd.Converged || sd.PsiK == nil {
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

func inArms(list []R3Arm, a R3Arm) bool {
	for _, x := range list {
		if x == a {
			return true
		}
	}
	return false
}

// r4Calib is the calibration form for R3Measure: the arm's own ψ where it has one,
// else the seed's Primary ψ, else φ (R3's convention). PlanePsi0 and Psi0Frac are
// diagnostics only and never enter a decision.
func r4Calib(arm R3Arm, sd *R4Seed) *Form3 {
	switch arm {
	case ArmPsi1:
		return sd.Psi1
	case ArmPsiK:
		return sd.PsiK
	}
	if sd.Primary != nil {
		return sd.Primary
	}
	return Phi()
}

func r4CellName(steps int) string {
	switch steps {
	case R4StepsA:
		return fmt.Sprintf("A%d", steps)
	case R4StepsB:
		return fmt.Sprintf("B%d", steps)
	case R4StepsC:
		return fmt.Sprintf("C%d", steps)
	}
	panic("mdcu: not an R4 step rung")
}

// runR4 runs the grid arms of one seed with chained snapshots. only (nil = all)
// restricts the arms; maxSteps caps the ladder (regression uses R4StepsA).
func runR4(sd *R4Seed, tierC bool, only map[R3Arm]bool, maxSteps int) []R3Run {
	z0 := Q1Init(24, 0.1, newRNG(sd.Seed))
	var out []R3Run
	for _, arm := range R4ArmsFor(sd) {
		if only != nil && !only[arm] {
			continue
		}
		top := R4StepsA
		if inArms(R4TierBArms, arm) {
			top = R4StepsB
		}
		if tierC && inArms(R4TierCArms, arm) {
			top = R4StepsC
		}
		if top > maxSteps {
			top = maxSteps
		}
		en := R4EnergyFor(arm, sd.Primary, sd.Psi1, sd.PsiK)
		calib := r4Calib(arm, sd)
		Z, done := z0, 0
		for _, rung := range []int{R4StepsA, R4StepsB, R4StepsC} {
			if rung > top {
				break
			}
			Z = RunRingEnergy(Z, en, rung-done)
			done = rung
			out = append(out, R3Run{
				Cell: r4CellName(rung), N: 24, Steps: rung, Sigma: 0.1,
				Seed: sd.Seed, Stratum: sd.Stratum, Arm: arm, Stats: R3Measure(Z, calib),
			})
		}
	}
	return out
}

// RunR4Seed runs every applicable Tier A arm, Tier B arms to 1000 steps, and (if
// tierC) Tier C arms to 4000 steps on one grid seed. Every arm starts from the same
// Q1Init draw (paired design).
func RunR4Seed(sd *R4Seed, tierC bool) []R3Run { return runR4(sd, tierC, nil, R4StepsC) }

// RunR4SD runs the S-D arms (K-psi@1 and K-phi) at 250 and 1000 steps. An
// unconverged seed runs K-phi only.
func RunR4SD(sd *R4Seed) []R3Run {
	z0 := Q1Init(24, 0.1, newRNG(sd.Seed))
	var out []R3Run
	arms := []R3Arm{ArmPsi1, ArmPhi}
	for _, arm := range arms {
		if arm == ArmPsi1 && (!sd.Converged || sd.Psi1 == nil) {
			continue
		}
		en := R4EnergyFor(arm, sd.Primary, sd.Psi1, nil)
		calib := r4Calib(arm, sd)
		Z, done := z0, 0
		for _, rung := range []int{R4StepsA, R4StepsB} {
			Z = RunRingEnergy(Z, en, rung-done)
			done = rung
			out = append(out, R3Run{
				Cell: fmt.Sprintf("SD%d", rung), N: 24, Steps: rung, Sigma: 0.1,
				Seed: sd.Seed, Stratum: sd.Stratum, Arm: arm, Stats: R3Measure(Z, calib),
			})
		}
	}
	return out
}

// R4RunRec is the JSONL shape of a run record (same flattening as R3's driver).
type R4RunRec struct {
	Type string `json:"type"`
	R3Run
}

// R4Regression is the frozen bit-identity check (§1.3): rerun seeds 3000–3009 for
// K-full, K-phi and K-psi (native) at 250 steps through the R4 run layer and require
// every statistic to equal the R3 raw file. It returns the number of records
// compared. This is a code check, not data.
func R4Regression(r3RawPath string) (int, error) {
	f, err := os.Open(r3RawPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	type key struct {
		seed uint64
		arm  R3Arm
	}
	want := map[key]R3Run{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var r R4RunRec
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return 0, fmt.Errorf("decoding R3 raw: %w", err)
		}
		if r.Type != "run" || r.Cell != "main" || r.Seed < 3000 || r.Seed > 3009 {
			continue
		}
		if r.Arm == ArmFull || r.Arm == ArmPhi || r.Arm == ArmPsi {
			want[key{r.Seed, r.Arm}] = r.R3Run
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	only := map[R3Arm]bool{ArmFull: true, ArmPhi: true, ArmPsi: true}
	n := 0
	for seed := uint64(3000); seed <= 3009; seed++ {
		for _, got := range runR4(BuildR4Seed(seed), false, only, R4StepsA) {
			if got.Cell != "A250" {
				continue
			}
			w, ok := want[key{seed, got.Arm}]
			if !ok {
				return n, fmt.Errorf("regression: no R3 record for seed %d arm %s", seed, got.Arm)
			}
			w.Cell = got.Cell
			if !reflect.DeepEqual(w, got) {
				return n, fmt.Errorf("regression: seed %d arm %s differs from the R3 raw (S3 R3 %v, R4 %v)", seed, got.Arm, w.Stats.S3, got.Stats.S3)
			}
			n++
		}
	}
	if n != len(want) {
		return n, fmt.Errorf("regression: compared %d records but the R3 raw has %d", n, len(want))
	}
	return n, nil
}
