package mdcu

// Round-4 decision rules (frozen R4 prereg §1.3–§1.5, §3), kept in the library so
// they are unit-tested on synthetic data with known answers (r4rules_test.go). The
// analyzer command is a thin reader/printer around these functions.
//
// Definitions come from R3's machinery (r3rules.go), unchanged:
//
//	rate        = 3-lock rate;   Δ(X,Y) = median paired S3(X) − S3(Y), 95% bootstrap CI
//	equivalent  = |rate(X) − rate(Y)| ≤ 10 points ∧ CI of Δ inside ±0.05
//	clearly-higher(X,Y) = rate(X) − rate(Y) ≥ 40 points ∨ (Δ ≥ 0.15 ∧ CI excludes 0)
//
// INTERPRETATION CHOICES the frozen text does not spell out (fixed here, before any
// R4 data exists, and disclosed with every verdict):
//
//  1. Q4b pools both strata (the text says "all 200 seeds"); strata rates are
//     reported alongside. Contrasts pair by seed, so seeds whose ψ arms were dropped
//     (unconverged comass) simply leave the pair.
//  2. "Rescaled arms" dropped on an unconverged comass are exactly K-psi@1 and K-psi@κ
//     (the arms whose form depends on the comass estimate; φ_L = L·φ does not).
//  3. The sampler-miss rate is misses / S1-split calls (odd seeds); S2 cannot miss.
//     The comass rate is unconverged / grid seeds that have a ψ.
//  4. "inconclusive (sampler)" and "inconclusive (comass)" apply to the verdicts that
//     use a ψ arm (Q4b, Q4c and the ψ secondaries). Q4a (K-phi only) and G4 are not
//     affected by either.
//  5. Q4c picks the first cell whose K-phi rate on the stratum's seeds is ≥ 50%, then
//     applies R3's Q1b rule (VerdictQ1b) to that cell's paired seeds.
//  6. S-D's kill is evaluated at each of the 250 and 1000-step cells; it is reported
//     as fired if it fires at either. S-D carries no decision weight.
//  7. Floor-equivalence (both rates < 10%) is reported, never changes a verdict
//     (oppenheimer seq 101, arch seq 102: disclosure, not rule).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
)

const (
	R4G4Rate       = 0.90 // gate G4: K-full 3-lock rate at 250 steps
	R4PowerFloor   = 100  // paired seeds below which a verdict reads "inconclusive (power)"
	R4MissLimit    = 0.05 // sampler-miss / unconverged-comass kill thresholds
	R4LockedMin    = 20   // locked runs needed for a spontaneity test
	r4HaarKill     = 0.01
	r4HaarConsist  = 0.05
	r4SBBig        = 0.20
	r4SBSmall      = 0.10
	r4SDKillPoints = 0.50
)

// R4Data holds grid and S-D records as the driver wrote them.
type R4Data struct {
	Seeds map[uint64]R4Seed
	Runs  map[string]map[R3Arm][]R3Run // cell → arm → runs
}

// NewR4Data returns an empty container.
func NewR4Data() *R4Data {
	return &R4Data{Seeds: map[uint64]R4Seed{}, Runs: map[string]map[R3Arm][]R3Run{}}
}

// Add files one run record under its cell and arm.
func (d *R4Data) Add(r R3Run) {
	if d.Runs[r.Cell] == nil {
		d.Runs[r.Cell] = map[R3Arm][]R3Run{}
	}
	d.Runs[r.Cell][r.Arm] = append(d.Runs[r.Cell][r.Arm], r)
}

// Get returns the runs of (cell, arm), restricted to a stratum if stratum != "".
func (d *R4Data) Get(cell string, arm R3Arm, stratum string) []R3Run {
	var out []R3Run
	for _, r := range d.Runs[cell][arm] {
		if stratum == "" || r.Stratum == stratum {
			out = append(out, r)
		}
	}
	return out
}

// R4Meta is what the loader keeps of the driver's meta/end lines.
type R4Meta struct {
	Meta  map[string]any
	Ended string // "end: …" or "abort: …" ("" if the run has no clean end line)
	Probe map[string]any
	Plan  map[string]any
}

// LoadR4 reads a JSONL file written by the R4 driver.
func LoadR4(path string) (*R4Data, *R4Meta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	d, m := NewR4Data(), &R4Meta{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<22), 1<<24)
	for sc.Scan() {
		var l struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(sc.Bytes(), &l) != nil {
			continue
		}
		switch l.Type {
		case "meta":
			_ = json.Unmarshal(sc.Bytes(), &m.Meta)
		case "probe":
			_ = json.Unmarshal(sc.Bytes(), &m.Probe)
		case "plan":
			_ = json.Unmarshal(sc.Bytes(), &m.Plan)
		case "seed":
			var s R4Seed
			if json.Unmarshal(sc.Bytes(), &s) == nil {
				d.Seeds[s.Seed] = s
			}
		case "run":
			var r R4RunRec
			if json.Unmarshal(sc.Bytes(), &r) == nil {
				d.Add(r.R3Run)
			}
		case "end", "abort":
			var x map[string]any
			_ = json.Unmarshal(sc.Bytes(), &x)
			m.Ended = fmt.Sprintf("%s: %v", l.Type, x["reason"])
		}
	}
	return d, m, sc.Err()
}

// ---- equivalence and dominance (frozen §1.4) -------------------------------------

func r4Equivalent(c ContrastStat) bool { return within10(c.RateA, c.RateB) && c.ciInsideEq() }

func r4ClearlyHigher(c ContrastStat) bool {
	return c.bigPositive() || c.RateA-c.RateB >= r3Rate40-r3Eps
}

func r4Floor(c ContrastStat) bool { return c.RateA < 0.10 && c.RateB < 0.10 }

// ---- G4 --------------------------------------------------------------------------

// R4Gate is G4: K-full at 250 steps 3-locks in at least 90% of seeds.
func R4Gate(d *R4Data) (pass bool, rate float64) {
	rate = LockRate(d.Get("A250", ArmFull, ""))
	return !math.IsNaN(rate) && rate >= R4G4Rate-r3Eps, rate
}

// ---- kill conditions (frozen §3) -------------------------------------------------

// R4KillStatus reports the two data-dependent kill conditions on the grid seeds.
type R4KillStatus struct {
	Sampler              bool
	Comass               bool
	Misses, S1Calls      int
	Unconverged, WithPsi int
	MissRate, UnconvRate float64
}

// R4Kills evaluates the sampler-miss and unconverged-comass kill conditions.
func R4Kills(seeds map[uint64]R4Seed) R4KillStatus {
	var k R4KillStatus
	for _, s := range seeds {
		if s.Kind != "grid" {
			continue
		}
		if s.Sampler == "S1" || (s.Sampler == "" && s.Seed%2 == 1) {
			k.S1Calls++
			if s.Miss {
				k.Misses++
			}
		}
		if !s.Miss {
			k.WithPsi++
			if !s.Converged {
				k.Unconverged++
			}
		}
	}
	if k.S1Calls > 0 {
		k.MissRate = float64(k.Misses) / float64(k.S1Calls)
	}
	if k.WithPsi > 0 {
		k.UnconvRate = float64(k.Unconverged) / float64(k.WithPsi)
	}
	k.Sampler = k.MissRate > R4MissLimit+r3Eps
	k.Comass = k.UnconvRate > R4MissLimit+r3Eps
	return k
}

// ---- Q4a -------------------------------------------------------------------------

type Q4aRung struct {
	Steps int
	Cell  string
	N     int
	Rate  float64
}

type Q4aResult struct {
	Rungs   []Q4aRung
	Verdict string // locks slowly | partial | does not lock by 4000 | inconclusive (power)
	RStar   int    // first rung with rate ≥ 50% (0 if none)
}

// R4Q4a: does φ lock if given more steps? K-phi at 250 / 1000 / 4000.
func R4Q4a(d *R4Data) Q4aResult {
	var r Q4aResult
	for _, p := range []struct {
		cell  string
		steps int
	}{{"A250", R4StepsA}, {"B1000", R4StepsB}, {"C4000", R4StepsC}} {
		rs := d.Get(p.cell, ArmPhi, "")
		rung := Q4aRung{Steps: p.steps, Cell: p.cell, N: len(rs), Rate: LockRate(rs)}
		r.Rungs = append(r.Rungs, rung)
		if r.RStar == 0 && len(rs) > 0 && rung.Rate >= 0.50-r3Eps {
			r.RStar = p.steps
		}
	}
	last := r.Rungs[2]
	switch {
	case last.N < R4PowerFloor:
		r.Verdict, r.RStar = "inconclusive (power)", 0
	case last.Rate >= 0.50-r3Eps:
		r.Verdict = "locks slowly"
	case last.Rate < 0.10-r3Eps:
		r.Verdict = "does not lock by 4000"
	default:
		r.Verdict = "partial"
	}
	if r.Verdict != "locks slowly" && r.Verdict != "inconclusive (power)" {
		r.RStar = 0
	}
	return r
}

// ---- Q4b -------------------------------------------------------------------------

type Q4bFlags struct {
	EqPsi1Phi  bool // equivalent(K-psi@1, K-phi)
	EqPsiKPhiK bool // equivalent(K-psi@κ, K-phi@κ)
	PhiKHigher bool // clearly-higher(K-phi@κ, K-phi)
	FormAt1    bool // clearly-higher(K-psi@1, K-phi)
	FormAtK    bool // clearly-higher(K-psi@κ, K-phi@κ)
}

type Q4bResult struct {
	Verdict        string // COUPLING | FORM | MIXED | inconclusive (power)
	N              int    // smallest paired count over the three contrasts
	Rates          map[R3Arm]float64
	RatesByStratum map[string]map[R3Arm]float64
	Contrasts      map[string]ContrastStat
	Flags          Q4bFlags
	FloorPairs     []string // equivalent pairs with both rates < 10%: a floor, not a finding
}

var q4bArms = []R3Arm{ArmPhi, ArmPhiK, ArmPsi1, ArmPsiK}

// R4Q4b: coupling scale or form? Tier A, four arms.
func R4Q4b(d *R4Data) Q4bResult {
	g := func(a R3Arm) []R3Run { return d.Get("A250", a, "") }
	phi, phiK, psi1, psiK := g(ArmPhi), g(ArmPhiK), g(ArmPsi1), g(ArmPsiK)
	r := Q4bResult{Rates: map[R3Arm]float64{}, RatesByStratum: map[string]map[R3Arm]float64{}, Contrasts: map[string]ContrastStat{}}
	for _, a := range q4bArms {
		r.Rates[a] = LockRate(g(a))
	}
	for _, st := range []string{"definite", "split"} {
		r.RatesByStratum[st] = map[R3Arm]float64{}
		for _, a := range q4bArms {
			r.RatesByStratum[st][a] = LockRate(d.Get("A250", a, st))
		}
	}
	c1 := R3Contrast("Δ(K-psi@1, K-phi)", psi1, phi)
	cK := R3Contrast("Δ(K-psi@κ, K-phi@κ)", psiK, phiK)
	cP := R3Contrast("Δ(K-phi@κ, K-phi)", phiK, phi)
	r.Contrasts[c1.Label], r.Contrasts[cK.Label], r.Contrasts[cP.Label] = c1, cK, cP
	r.N = minInt(c1.N, minInt(cK.N, cP.N))
	r.Flags = Q4bFlags{
		EqPsi1Phi: r4Equivalent(c1), EqPsiKPhiK: r4Equivalent(cK), PhiKHigher: r4ClearlyHigher(cP),
		FormAt1: r4ClearlyHigher(c1), FormAtK: r4ClearlyHigher(cK),
	}
	if r.Flags.EqPsi1Phi && r4Floor(c1) {
		r.FloorPairs = append(r.FloorPairs, "K-psi@1 ~ K-phi")
	}
	if r.Flags.EqPsiKPhiK && r4Floor(cK) {
		r.FloorPairs = append(r.FloorPairs, "K-psi@κ ~ K-phi@κ")
	}
	switch {
	case r.N < R4PowerFloor:
		r.Verdict = "inconclusive (power)"
	case r.Flags.EqPsi1Phi && r.Flags.EqPsiKPhiK && r.Flags.PhiKHigher:
		r.Verdict = "COUPLING"
	case r.Flags.FormAt1 && r.Flags.FormAtK:
		r.Verdict = "FORM"
	default:
		r.Verdict = "MIXED"
	}
	return r
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---- Q4c -------------------------------------------------------------------------

type Q4cResult struct {
	Stratum string
	Cell    string // "(1,250)" … ; "" if no cell qualified
	Verdict string // no | yes | yes-reversed | inconclusive … (R3 Q1b vocabulary)
	Stat    ContrastStat
}

type q4cCell struct {
	Name     string
	Cell     string
	Phi, Psi R3Arm
}

// The fixed evaluation order of the frozen text.
var q4cCells = []q4cCell{
	{"(1,250)", "A250", ArmPhi, ArmPsi1},
	{"(κ,250)", "A250", ArmPhiK, ArmPsiK},
	{"(1,1000)", "B1000", ArmPhi, ArmPsi1},
	{"(κ,1000)", "B1000", ArmPhiK, ArmPsiK},
	{"(1,4000)", "C4000", ArmPhi, ArmPsi1},
}

// R4Q4c: does φ specifically matter at matched coupling? R3's Q1b rule at the first
// cell (in the fixed order) whose K-phi rate on this stratum's seeds is ≥ 50%.
func R4Q4c(d *R4Data, stratum string) Q4cResult {
	for _, c := range q4cCells {
		phi := d.Get(c.Cell, c.Phi, stratum)
		if len(phi) == 0 || LockRate(phi) < r3PhiPre-r3Eps {
			continue
		}
		psi := d.Get(c.Cell, c.Psi, stratum)
		pa, pb := PairBySeed(phi, psi)
		st := R3Contrast("Δ("+string(c.Phi)+", "+string(c.Psi)+") "+c.Name+" "+stratum, phi, psi)
		return Q4cResult{Stratum: stratum, Cell: c.Name, Verdict: VerdictQ1b(st, pa, pb), Stat: st}
	}
	return Q4cResult{Stratum: stratum, Verdict: "inconclusive (φ never locks)"}
}

// ---- reading table (frozen §1.4) -------------------------------------------------

// R4Reading maps the gate, Q4b and the two Q4c strata to the pre-registered reading.
func R4Reading(gate bool, q4b, q4cDef, q4cSplit string) string {
	if !gate {
		return "Inconclusive (non-replication): K-full did not 3-lock in ≥ 90% of seeds, so no R4 reading is made."
	}
	if strings.HasPrefix(q4b, "inconclusive") {
		reason := strings.TrimSuffix(strings.TrimPrefix(q4b, "inconclusive ("), ")")
		return "Inconclusive (" + reason + "): no R4-Q4b reading is made."
	}
	both := func(v string) bool { return q4cDef == v && q4cSplit == v }
	switch q4b {
	case "COUPLING":
		switch {
		case both("no"):
			return "The R3 gap was coupling scale. At matched coupling φ and ψ lock alike: a 3-form term drives locking; not φ-specific. (COUPLING, no/no)"
		case both("yes"):
			return "Coupling scale explains why φ looked inert, but at matched coupling φ locks more than ψ. The only cell supporting a φ-specific reading (still napkin; needs a fresh-seed replication before any anchor). (COUPLING, yes/yes)"
		}
		return "Recorded as-is. No claim. (COUPLING, Q4c mixed or inconclusive)"
	case "FORM":
		return "The gap is not coupling scale. ψ locks where φ does not at matched comass. Reported; ψ-locks-more is not read as G₂ evidence. Candidate mechanisms are for oppenheimer to name. (FORM)"
	}
	return "Recorded as-is. No claim. (MIXED)"
}

// ---- spontaneity result and hypotheses -------------------------------------------

// SponResult is a Haar-G₂ KS spontaneity test of one arm's locked principal planes.
type SponResult struct {
	Arm      R3Arm
	Cell     string
	N        int // locked runs (planes) used
	D, P     float64
	Testable bool // N ≥ R4LockedMin
}

func (s SponResult) ok() bool { return s.N >= R4LockedMin }

// MechResult is the pre-registered mechanism reading (no claim either way).
type MechResult struct {
	KillForm, KillCouplingOnly bool
	Reading                    string
}

// R4Mechanism evaluates the two kills. cKappa = Δ(K-psi@κ, K-phi@κ); cLag =
// Δ(K-phi@κ, K-psi@1); phiK and psi1 are the spontaneity results of K-phi@κ and
// K-psi@1 (≥ 20 locked runs needed).
func R4Mechanism(cKappa, cLag ContrastStat, phiK, psi1 SponResult) MechResult {
	var m MechResult
	m.KillForm = r4Equivalent(cKappa) && phiK.ok() && phiK.P < r4HaarKill
	lag := !r4Equivalent(cLag) && cLag.RateA < cLag.RateB
	haarVsCluster := phiK.ok() && phiK.P >= r4HaarConsist && psi1.ok() && psi1.P < r4HaarKill
	m.KillCouplingOnly = lag || haarVsCluster
	switch {
	case m.KillForm && m.KillCouplingOnly:
		m.Reading = "both kills fire (reported as-is, no claim)"
	case m.KillForm:
		m.Reading = "FORM killed: K-phi@κ ≡ K-psi@κ and K-phi@κ locks non-Haar; the gap was coupling"
	case m.KillCouplingOnly:
		m.Reading = "COUPLING-ONLY killed: K-phi@κ lags K-psi@1, or locks Haar-uniform while K-psi@1 clusters"
	default:
		m.Reading = "mechanism reading undetermined (neither kill fires)"
	}
	return m
}

// SAResult holds the S-A hypothesis statuses.
type SAResult struct{ A1, A2, A3 string }

// R4SA evaluates H-A1 (K-full Haar-consistent), H-A2 (every ψ arm non-Haar) and
// H-A3 (every locking φ arm Haar-consistent).
func R4SA(full SponResult, psiArms, phiArms []SponResult) SAResult {
	var h SAResult
	switch {
	case !full.ok():
		h.A1 = "untestable"
	case full.P < r4HaarKill:
		h.A1 = "killed"
	case full.P >= r4HaarConsist:
		h.A1 = "holds"
	default:
		h.A1 = "undetermined"
	}
	var n int
	allNon, anyConsist := true, false
	for _, s := range psiArms {
		if !s.ok() {
			continue
		}
		n++
		if s.P >= r4HaarConsist {
			anyConsist = true
		}
		if s.P >= r4HaarKill {
			allNon = false
		}
	}
	switch {
	case n == 0:
		h.A2 = "untestable"
	case anyConsist:
		h.A2 = "killed"
	case allNon:
		h.A2 = "holds"
	default:
		h.A2 = "undetermined"
	}
	n = 0
	allHaar, anyNon := true, false
	for _, s := range phiArms {
		if !s.ok() {
			continue
		}
		n++
		if s.P < r4HaarKill {
			anyNon = true
		}
		if s.P < r4HaarConsist {
			allHaar = false
		}
	}
	switch {
	case n == 0:
		h.A3 = "untestable"
	case anyNon:
		h.A3 = "killed"
	case allHaar:
		h.A3 = "holds"
	default:
		h.A3 = "undetermined"
	}
	return h
}

// ---- S-B -------------------------------------------------------------------------

// R4SBStat is the Spearman ρ of x against y with a 95% percentile bootstrap CI over
// 10,000 resamples of the pairs (RNG seeded from the label). NaN resamples are skipped.
func R4SBStat(label string, x, y []float64) (rho, lo, hi float64, n int) {
	n = len(x)
	rho = Spearman(x, y)
	if n < 3 {
		return rho, math.NaN(), math.NaN(), n
	}
	rng := rand.New(rand.NewPCG(bootSeed(label), 0x13198a2e03707344))
	bx, by := make([]float64, n), make([]float64, n)
	rs := make([]float64, 0, r3Boot)
	for b := 0; b < r3Boot; b++ {
		for i := range bx {
			j := rng.IntN(n)
			bx[i], by[i] = x[j], y[j]
		}
		if v := Spearman(bx, by); !math.IsNaN(v) {
			rs = append(rs, v)
		}
	}
	sort.Float64s(rs)
	if len(rs) == 0 {
		return rho, math.NaN(), math.NaN(), n
	}
	return rho, quantileF(rs, .025), quantileF(rs, .975), n
}

// R4SBVerdict: H-B holds iff |ρ| ≥ 0.20, the CI excludes 0 and ρ has R3's sign
// (negative). Killed iff the CI includes 0 and |ρ| < 0.10. Otherwise undetermined.
func R4SBVerdict(rho, lo, hi float64) string {
	if math.IsNaN(rho) || math.IsNaN(lo) {
		return "undetermined"
	}
	excl0 := lo > 0 || hi < 0
	switch {
	case math.Abs(rho) >= r4SBBig-r3Eps && excl0 && rho < 0:
		return "holds"
	case !excl0 && math.Abs(rho) < r4SBSmall-r3Eps:
		return "killed"
	}
	return "undetermined"
}

// ---- S-C -------------------------------------------------------------------------

// R4SC: finding (a) at matched coupling. Holds iff K-full@κ ≥ 90% ∧ K-gram@κ < 50% ∧
// K-phi@κ < 50%; otherwise the frozen sentence, with the three rates.
func R4SC(d *R4Data) string {
	f, g, p := LockRate(d.Get("A250", ArmFullK, "")), LockRate(d.Get("A250", ArmGramK, "")), LockRate(d.Get("A250", ArmPhiK, ""))
	if f >= 0.90-r3Eps && g < 0.50-r3Eps && p < 0.50-r3Eps {
		return "holds"
	}
	return fmt.Sprintf("(a) fails at matched coupling: a single term locks (K-full@κ %.1f%%, K-gram@κ %.1f%%, K-phi@κ %.1f%%)", 100*f, 100*g, 100*p)
}

// ---- S-D -------------------------------------------------------------------------

type SDRung struct {
	S                 float64
	Cell              string
	N                 int
	RatePsi1, RatePhi float64
}

type SDResult struct {
	Rungs  []SDRung
	Killed bool // H-D killed: at s = 0.005, rate(K-psi@1) − rate(K-phi) ≥ 50 points (either cell)
}

// R4SD groups the S-D runs by the seed's ladder scale and reports both arms' rates.
// "Falls continuously as d → 0" has no numeric rule in the frozen text, so only the
// kill is evaluated; the per-rung rates are the descriptive record.
func R4SD(d *R4Data) SDResult {
	scale := map[uint64]float64{}
	for s, sd := range d.Seeds {
		if sd.Kind == "sd" && sd.S2S != nil {
			scale[s] = *sd.S2S
		}
	}
	var res SDResult
	for _, cell := range []string{"SD250", "SD1000"} {
		for _, s := range R4SDScales {
			pick := func(arm R3Arm) []R3Run {
				var out []R3Run
				for _, r := range d.Runs[cell][arm] {
					if v, ok := scale[r.Seed]; ok && v == s {
						out = append(out, r)
					}
				}
				return out
			}
			psi, phi := pick(ArmPsi1), pick(ArmPhi)
			if len(phi) == 0 && len(psi) == 0 {
				continue
			}
			pa, pb := PairBySeed(psi, phi)
			rung := SDRung{S: s, Cell: cell, N: len(phi), RatePsi1: LockRate(pa), RatePhi: LockRate(pb)}
			if len(pa) == 0 {
				rung.RatePsi1, rung.RatePhi = math.NaN(), LockRate(phi)
			}
			res.Rungs = append(res.Rungs, rung)
			if s == R4SDScales[0] && !math.IsNaN(rung.RatePsi1) && rung.RatePsi1-rung.RatePhi >= r4SDKillPoints-r3Eps {
				res.Killed = true
			}
		}
	}
	return res
}
