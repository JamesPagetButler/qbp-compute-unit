package mdcu

// qbp-kz-g2 round 1 analyzer (FROZEN-v1 §3, sha256 785e0ee2…f51e28): gap(τ) on the common
// grid at the four frozen ramp lengths, the decision rule with its precedence (b) → (a) →
// (c), the r5 harness-kill sub-clause (no partial-set evaluation), and the non-gating
// secondaries (OLS slope of log gap on log τ with its SE, the jump clause, the seed spread).
// It reads the Stage-0b-only raw files written by KZExecuteStage0bOnly. The round-1
// -analyze path (KZAnalyze / KZLoadRunRecords) is untouched and refuses these files.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

// Frozen §3 numbers. gap(τ) is in units of S_max = 1.
const (
	KZG2GapTol   = 0.05   // (b): gap(12800) < 0.05 → H-slow; τ* = smallest τ with gap < 0.05
	KZG2Ratio    = 0.6    // (a): gap(12800) ≥ 0.6 · gap(1600) → H-first
	KZG2JumpTol  = 0.10   // secondary: a seed-mean single-step jump > 0.10 is "jump observed"
	KZG2P4Centre = 0.5176 // §4 P4: gap(1600) reproduces round 1 within ±0.05
	KZG2P4Half   = 0.05
)

// KZG2Rung is one τ_Q's Stage-0b-only raw file, as loaded. Gap/GapIdx/GapT are recomputed
// from the stored curves on the common grid; RecMaxGap is what the file recorded.
type KZG2Rung struct {
	File      string
	Mode      string
	PreregSHA string
	Tau       int
	TauIndex  int
	Tc        float64
	Seeds     []uint64
	Complete  bool // meta + stage0b + end, and no abort record
	AbortCode int

	Grid, Cool, Heat       []float64
	CoolT, HeatT           []float64
	CoolSeries, HeatSeries [][]float64

	RecMaxGap    float64
	RecMaxGapIdx int
	Gap          float64
	GapIdx       int
	GapT         float64

	CoolJump, HeatJump       float64
	CoolJumpIdx, HeatJumpIdx int
	CoolJumpT, HeatJumpT     float64

	SeedGapN               int // per-seed max gap on the grid (seed spread), when series are present
	SeedGapMin, SeedGapMax float64
}

type kzG2Record struct {
	Kind       string      `json:"kind"`
	Mode       string      `json:"mode"`
	PreregSHA  string      `json:"prereg_sha"`
	Tau        int         `json:"tau"`
	TauIndex   int         `json:"tau_index"`
	Tc         float64     `json:"tc"`
	Seeds      []uint64    `json:"seeds"`
	Code       int         `json:"code"`
	Grid       []float64   `json:"grid"`
	Cool       []float64   `json:"cool"`
	Heat       []float64   `json:"heat"`
	CoolT      []float64   `json:"cool_t"`
	HeatT      []float64   `json:"heat_t"`
	CoolSeries [][]float64 `json:"cool_series"`
	HeatSeries [][]float64 `json:"heat_series"`
	MaxGap     float64     `json:"max_gap"`
	MaxGapIdx  int         `json:"max_gap_idx"`
	CoolJump   float64     `json:"cool_jump"`
	HeatJump   float64     `json:"heat_jump"`
	CoolJumpI  int         `json:"cool_jump_idx"`
	HeatJumpI  int         `json:"heat_jump_idx"`
	CoolJumpT  float64     `json:"cool_jump_T"`
	HeatJumpT  float64     `json:"heat_jump_T"`
	R          int         `json:"R"`
	Ladder     []int       `json:"ladder"`
}

// KZG2LoadRung reads one Stage-0b-only raw JSONL. It needs a meta record of mode "0b" or
// "smoke-0b" (a grid/probe/calibration file is an error) and one stage0b record; the rung
// is Complete only if an end record follows and no abort record is present.
func KZG2LoadRung(rd io.Reader) (KZG2Rung, error) {
	var r KZG2Rung
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	haveMeta, haveSB, haveEnd := false, false, false
	for sc.Scan() {
		var m kzG2Record
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			continue
		}
		switch m.Kind {
		case "meta":
			if haveMeta {
				return r, fmt.Errorf("two meta records")
			}
			if m.Mode != "0b" && m.Mode != "smoke-0b" {
				return r, fmt.Errorf("meta mode %q is not a Stage-0b-only file (R=%d ladder=%v)", m.Mode, m.R, m.Ladder)
			}
			haveMeta = true
			r.Mode, r.PreregSHA, r.Tau, r.TauIndex, r.Tc, r.Seeds = m.Mode, m.PreregSHA, m.Tau, m.TauIndex, m.Tc, m.Seeds
		case "stage0b":
			if haveSB {
				return r, fmt.Errorf("two stage0b records")
			}
			haveSB = true
			r.Grid, r.Cool, r.Heat, r.CoolT, r.HeatT = m.Grid, m.Cool, m.Heat, m.CoolT, m.HeatT
			r.CoolSeries, r.HeatSeries = m.CoolSeries, m.HeatSeries
			r.RecMaxGap, r.RecMaxGapIdx = m.MaxGap, m.MaxGapIdx
			r.CoolJump, r.HeatJump, r.CoolJumpIdx, r.HeatJumpIdx = m.CoolJump, m.HeatJump, m.CoolJumpI, m.HeatJumpI
			r.CoolJumpT, r.HeatJumpT = m.CoolJumpT, m.HeatJumpT
			if m.Tau != 0 && haveMeta && m.Tau != r.Tau {
				return r, fmt.Errorf("stage0b tau %d differs from meta tau %d", m.Tau, r.Tau)
			}
		case "end":
			haveEnd = true
		case "abort":
			r.AbortCode = m.Code
			if r.AbortCode == 0 {
				r.AbortCode = 1
			}
		}
	}
	if err := sc.Err(); err != nil {
		return r, err
	}
	if !haveMeta {
		return r, fmt.Errorf("no meta record")
	}
	if !haveSB {
		return r, fmt.Errorf("no stage0b record")
	}
	r.Complete = haveEnd && r.AbortCode == 0
	if len(r.Grid) > 0 && len(r.Cool) == len(r.Grid) && len(r.Heat) == len(r.Grid) {
		r.Gap, r.GapIdx, r.GapT = KZStage0bGap(r.Grid, r.Cool, r.Heat)
	}
	// seed spread: each seed's own cool/heat series interpolated onto the grid
	if n := len(r.CoolSeries); n > 0 && n == len(r.HeatSeries) && len(r.CoolT) > 1 && len(r.HeatT) > 1 {
		r.SeedGapMin = math.Inf(1)
		for s := 0; s < n; s++ {
			if len(r.CoolSeries[s]) != len(r.CoolT) || len(r.HeatSeries[s]) != len(r.HeatT) {
				continue
			}
			g := 0.0
			for _, T := range r.Grid {
				g = math.Max(g, math.Abs(KZInterp(r.CoolT, r.CoolSeries[s], T)-KZInterp(r.HeatT, r.HeatSeries[s], T)))
			}
			r.SeedGapN++
			r.SeedGapMin, r.SeedGapMax = math.Min(r.SeedGapMin, g), math.Max(r.SeedGapMax, g)
		}
		if r.SeedGapN == 0 {
			r.SeedGapMin = 0
		}
	}
	return r, nil
}

// KZG2Decision is the §3 reading of the four rungs.
type KZG2Decision struct {
	Rungs   []KZG2Rung // in KZKzG2R1TauSet order
	Gaps    []float64  // gap(τ) recomputed on the common grid, same order
	Outcome string     // "H-first", "H-slow", "inconclusive (rate)"
	Rule    string     // "(a)", "(b)", "(c)"
	TauStar int        // (b): the smallest τ in the set with gap < 0.05; 0 otherwise
	Overlap bool       // (a) and (b) both held; (b) governs (precedence clause)
	P4Hit   bool       // gap(1600) within ±0.05 of round 1's 0.5176

	// secondaries
	Slope, SlopeSE float64
	FitN           int
	FitExcluded    []int // τ with gap exactly 0, excluded from the fit
	SlopeOK        bool  // slope and SE both defined (n ≥ 3)
	Jumps          []int // τ at which a seed-mean single-step jump > 0.10 was recorded
	Warnings       []string
}

// KZG2Decide applies §3 to exactly the four complete rungs, one per τ_Q of the frozen set
// (any file order). Fewer, more, duplicated, incomplete, smoke or inconsistent rungs are
// refused: the rule is never applied to a partial set (r5 harness-kill sub-clause).
func KZG2Decide(in []KZG2Rung) (KZG2Decision, error) {
	var d KZG2Decision
	n := len(KZKzG2R1TauSet)
	if len(in) != n {
		return d, fmt.Errorf("refusing: %d rung(s) given; §3 is evaluated only on all %d complete rungs (tau_Q %v) — no partial-set reading (r5 harness-kill sub-clause)", len(in), n, KZKzG2R1TauSet)
	}
	byTau := map[int]KZG2Rung{}
	for _, r := range in {
		if _, ok := KZKzG2R1TauIndex(r.Tau); !ok {
			return d, fmt.Errorf("refusing: %s has tau_Q %d, not in the frozen set %v", r.File, r.Tau, KZKzG2R1TauSet)
		}
		if _, dup := byTau[r.Tau]; dup {
			return d, fmt.Errorf("refusing: two files for tau_Q %d (%s, %s)", r.Tau, byTau[r.Tau].File, r.File)
		}
		byTau[r.Tau] = r
	}
	d.Rungs = make([]KZG2Rung, n)
	d.Gaps = make([]float64, n)
	for i, tau := range KZKzG2R1TauSet {
		r, ok := byTau[tau]
		if !ok {
			return d, fmt.Errorf("refusing: no file for tau_Q %d; all four rungs are needed", tau)
		}
		if !r.Complete {
			return d, fmt.Errorf("refusing: %s (tau_Q %d) is incomplete (abort code %d / no end record): a harness kill voids the rung; re-run it on its reserved sub-block under a dated amendment", r.File, tau, r.AbortCode)
		}
		if r.Mode != "0b" {
			return d, fmt.Errorf("refusing: %s (tau_Q %d) is a %q file, not a run-mode file", r.File, tau, r.Mode)
		}
		if len(r.Grid) != 41 || len(r.Cool) != len(r.Grid) || len(r.Heat) != len(r.Grid) {
			return d, fmt.Errorf("refusing: %s (tau_Q %d) curves are %d/%d points on a %d-point grid; want 41", r.File, tau, len(r.Cool), len(r.Heat), len(r.Grid))
		}
		if i > 0 {
			p := d.Rungs[0]
			if r.Tc != p.Tc {
				return d, fmt.Errorf("refusing: tc differs between rungs (%v at tau_Q %d, %v at tau_Q %d)", p.Tc, p.Tau, r.Tc, tau)
			}
			for k := range r.Grid {
				if r.Grid[k] != p.Grid[k] {
					return d, fmt.Errorf("refusing: the common grid differs between tau_Q %d and %d at point %d", p.Tau, tau, k)
				}
			}
		}
		gap, idx, T := KZStage0bGap(r.Grid, r.Cool, r.Heat)
		r.Gap, r.GapIdx, r.GapT = gap, idx, T
		if math.Abs(gap-r.RecMaxGap) > 1e-9 {
			d.Warnings = append(d.Warnings, fmt.Sprintf("tau_Q %d: recorded max_gap %.6f differs from the gap recomputed on the common grid %.6f; the recomputed value is used", tau, r.RecMaxGap, gap))
		}
		d.Rungs[i], d.Gaps[i] = r, gap
	}

	// §3, precedence (b) → (a) → (c)
	g1600, g12800 := d.Gaps[0], d.Gaps[n-1]
	b := g12800 < KZG2GapTol
	a := g12800 >= KZG2Ratio*g1600
	switch {
	case b:
		d.Outcome, d.Rule = "H-slow", "(b)"
		for i, tau := range KZKzG2R1TauSet {
			if d.Gaps[i] < KZG2GapTol {
				d.TauStar = tau
				break
			}
		}
		d.Overlap = a
	case a:
		d.Outcome, d.Rule = "H-first", "(a)"
	default:
		d.Outcome, d.Rule = "inconclusive (rate)", "(c)"
	}
	d.P4Hit = math.Abs(g1600-KZG2P4Centre) <= KZG2P4Half

	// secondaries: OLS of ln gap on ln τ, zeros excluded and listed
	var xs, ys []float64
	for i, tau := range KZKzG2R1TauSet {
		if d.Gaps[i] == 0 {
			d.FitExcluded = append(d.FitExcluded, tau)
			continue
		}
		xs = append(xs, math.Log(float64(tau)))
		ys = append(ys, math.Log(d.Gaps[i]))
	}
	d.FitN = len(xs)
	d.Slope, d.SlopeSE, d.SlopeOK = kzG2OLS(xs, ys)
	for i, tau := range KZKzG2R1TauSet {
		if d.Rungs[i].CoolJump > KZG2JumpTol || d.Rungs[i].HeatJump > KZG2JumpTol {
			d.Jumps = append(d.Jumps, tau)
		}
	}
	sort.Ints(d.Jumps)
	return d, nil
}

// kzG2OLS is the least-squares slope of y on x with its standard error on n-2 degrees of
// freedom. ok is false when the SE is undefined (n < 3); the slope alone is still returned
// for n = 2, and NaN for n < 2.
func kzG2OLS(x, y []float64) (slope, se float64, ok bool) {
	n := float64(len(x))
	if len(x) < 2 {
		return math.NaN(), math.NaN(), false
	}
	mx, my := 0.0, 0.0
	for i := range x {
		mx += x[i]
		my += y[i]
	}
	mx /= n
	my /= n
	sxx, sxy := 0.0, 0.0
	for i := range x {
		sxx += (x[i] - mx) * (x[i] - mx)
		sxy += (x[i] - mx) * (y[i] - my)
	}
	slope = sxy / sxx
	if len(x) < 3 {
		return slope, math.NaN(), false
	}
	a := my - slope*mx
	ssr := 0.0
	for i := range x {
		r := y[i] - a - slope*x[i]
		ssr += r * r
	}
	return slope, math.Sqrt(ssr / (n - 2) / sxx), true
}

// KZG2FormatReport renders the decision: the per-rung table, the rule evaluation with its
// precedence, the outcome, and the secondaries. Tables first.
func KZG2FormatReport(d KZG2Decision) string {
	var b strings.Builder
	fmt.Fprintf(&b, "qbp-kz-g2 round 1 — Stage-0b-only analysis (FROZEN-v1 §3)\n")
	if len(d.Rungs) > 0 {
		fmt.Fprintf(&b, "tc = %.16g  seeds per rung = %d  grid = %d points\n\n", d.Rungs[0].Tc, len(d.Rungs[0].Seeds), len(d.Rungs[0].Grid))
	}
	fmt.Fprintf(&b, "| tau_Q | gap(tau) | at grid pt | T | seed gap min/max (n) | cool jump (rec, T) | heat jump (rec, T) | jump > 10%% | file |\n")
	fmt.Fprintf(&b, "|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range d.Rungs {
		jump := "no"
		for _, t := range d.Jumps {
			if t == r.Tau {
				jump = "YES"
			}
		}
		seedSpread := "n/a"
		if r.SeedGapN > 0 {
			seedSpread = fmt.Sprintf("%.4f / %.4f (%d)", r.SeedGapMin, r.SeedGapMax, r.SeedGapN)
		}
		fmt.Fprintf(&b, "| %d | %.4f | %d | %.4f | %s | %.4f (%d, %.4f) | %.4f (%d, %.4f) | %s | %s |\n",
			r.Tau, d.Gaps[i], r.GapIdx, r.GapT, seedSpread, r.CoolJump, r.CoolJumpIdx, r.CoolJumpT, r.HeatJump, r.HeatJumpIdx, r.HeatJumpT, jump, r.File)
	}
	n := len(d.Gaps)
	if n == 0 {
		return b.String()
	}
	g1600, g12800 := d.Gaps[0], d.Gaps[n-1]
	fmt.Fprintf(&b, "\nrule (precedence (b) -> (a) -> (c)):\n")
	fmt.Fprintf(&b, "  (b) gap(12800) = %.4f < %.2f ?  %v\n", g12800, KZG2GapTol, g12800 < KZG2GapTol)
	fmt.Fprintf(&b, "  (a) gap(12800) = %.4f >= %.1f * gap(1600) = %.4f ?  %v\n", g12800, KZG2Ratio, KZG2Ratio*g1600, g12800 >= KZG2Ratio*g1600)
	fmt.Fprintf(&b, "\nOUTCOME: %s by rule %s\n", d.Outcome, d.Rule)
	if d.Rule == "(b)" {
		fmt.Fprintf(&b, "  KZ premise restored at the smallest tau_Q with gap < %.2f: tau* = %d (a future ladder would start there)\n", KZG2GapTol, d.TauStar)
	}
	if d.Overlap {
		fmt.Fprintf(&b, "  (a) and (b) both held (gap(1600) = %.4f < %.4f): (b) governs; the P4 miss is reported beside it\n", g1600, KZG2GapTol/KZG2Ratio)
	}
	fmt.Fprintf(&b, "\nsecondaries (non-gating):\n")
	p4 := "MISS"
	if d.P4Hit {
		p4 = "hit"
	}
	fmt.Fprintf(&b, "  P4: gap(1600) = %.4f vs round 1's %.4f +/- %.2f -> %s\n", g1600, KZG2P4Centre, KZG2P4Half, p4)
	switch {
	case d.FitN < 2:
		fmt.Fprintf(&b, "  OLS slope of ln gap on ln tau_Q: not defined (%d non-zero point(s))\n", d.FitN)
	case !d.SlopeOK:
		fmt.Fprintf(&b, "  OLS slope of ln gap on ln tau_Q: %.4f (n = %d; SE undefined with %d dof)\n", d.Slope, d.FitN, d.FitN-2)
	default:
		fmt.Fprintf(&b, "  OLS slope of ln gap on ln tau_Q: %.4f +/- %.4f (SE, n = %d, %d dof)\n", d.Slope, d.SlopeSE, d.FitN, d.FitN-2)
	}
	if len(d.FitExcluded) > 0 {
		fmt.Fprintf(&b, "  gap exactly 0 at tau_Q %v: reported and excluded from the fit (fit on the remaining %d point(s))\n", d.FitExcluded, d.FitN)
	}
	if len(d.Jumps) == 0 {
		fmt.Fprintf(&b, "  jump clause: no seed-mean single-step jump > %.2f S_max at any tau_Q\n", KZG2JumpTol)
	}
	for _, t := range d.Jumps {
		fmt.Fprintf(&b, "  jump observed at tau_Q = %d (> %.2f S_max; supports H-first; does not override (a)-(c))\n", t, KZG2JumpTol)
	}
	for _, w := range d.Warnings {
		fmt.Fprintf(&b, "  WARNING: %s\n", w)
	}
	return b.String()
}
