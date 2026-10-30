package mdcu

// Tests for the qbp-kz-g2 round 1 analyzer (FROZEN-v1 §3, sha256 785e0ee2…f51e28), written
// before the code. Synthetic rungs only; no seed is touched.

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// kzG2SynthRung builds a complete rung whose recomputed gap on the common grid is exactly
// gap (cool[20] = 0, heat[20] = gap, so |cool - heat| carries no rounding) and whose jumps
// are the given magnitudes.
func kzG2SynthRung(tau int, gap, coolJump, heatJump float64) KZG2Rung {
	tc := 1.0666262219986817
	grid := KZLinGrid(0.5*tc, 1.5*tc, 41)
	cool, heat := make([]float64, 41), make([]float64, 41)
	for i, T := range grid {
		cool[i] = 1 / (1 + math.Exp((T-tc)/0.1))
		heat[i] = cool[i]
	}
	cool[20], heat[20] = 0, gap
	idx, _ := KZKzG2R1TauIndex(tau)
	return KZG2Rung{File: "synthetic", Mode: "0b", Tau: tau, TauIndex: idx, Tc: tc, Seeds: KZKzG2R1Seeds(idx, false),
		Grid: grid, Cool: cool, Heat: heat, RecMaxGap: gap, RecMaxGapIdx: 20,
		CoolJump: coolJump, HeatJump: heatJump, CoolJumpIdx: 3, HeatJumpIdx: 5, Complete: true}
}

func kzG2Set(gaps ...float64) []KZG2Rung {
	out := make([]KZG2Rung, len(gaps))
	for i, g := range gaps {
		out[i] = kzG2SynthRung(KZKzG2R1TauSet[i], g, 0.02, 0.02)
	}
	return out
}

func TestKZG2DecideHFirst(t *testing.T) {
	// gap(12800) = 0.36 >= 0.6 * 0.52 = 0.312 -> (a); P4 hit (0.52 in [0.4676, 0.5676])
	d, err := KZG2Decide(kzG2Set(0.52, 0.45, 0.40, 0.36))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "H-first" || d.Rule != "(a)" || d.Overlap || !d.P4Hit || d.TauStar != 0 {
		t.Fatalf("%+v", d)
	}
	for i, g := range []float64{0.52, 0.45, 0.40, 0.36} {
		if math.Abs(d.Gaps[i]-g) > 1e-12 {
			t.Fatalf("gap %d = %v, want %v (recomputed on the common grid)", i, d.Gaps[i], g)
		}
	}
	if d.FitN != 4 || len(d.FitExcluded) != 0 || !d.SlopeOK || d.Slope >= 0 || d.SlopeSE <= 0 {
		t.Fatalf("fit %+v", d)
	}
	// the slope is the OLS slope of ln gap on ln tau, with its SE on n-2 dof
	x := []float64{math.Log(1600), math.Log(3200), math.Log(6400), math.Log(12800)}
	y := []float64{math.Log(0.52), math.Log(0.45), math.Log(0.40), math.Log(0.36)}
	slope, se := kzOLS(x, y)
	if math.Abs(d.Slope-slope) > 1e-12 || math.Abs(d.SlopeSE-se) > 1e-12 {
		t.Fatalf("slope %v ± %v, want %v ± %v", d.Slope, d.SlopeSE, slope, se)
	}
	rep := KZG2FormatReport(d)
	for _, w := range []string{"H-first", "(a)", "1600", "12800", "slope", "P4"} {
		if !strings.Contains(rep, w) {
			t.Fatalf("report lacks %q:\n%s", w, rep)
		}
	}
}

// kzOLS is the reference slope/SE used only by the tests.
func kzOLS(x, y []float64) (float64, float64) {
	n := float64(len(x))
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
	b := sxy / sxx
	a := my - b*mx
	ssr := 0.0
	for i := range x {
		r := y[i] - a - b*x[i]
		ssr += r * r
	}
	return b, math.Sqrt(ssr / (n - 2) / sxx)
}

func TestKZG2DecideHSlow(t *testing.T) {
	// (b): gap(12800) = 0.01 < 0.05; tau* = smallest tau with gap < 0.05 = 6400
	d, err := KZG2Decide(kzG2Set(0.52, 0.20, 0.04, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "H-slow" || d.Rule != "(b)" || d.TauStar != 6400 || d.Overlap || !d.P4Hit {
		t.Fatalf("%+v", d)
	}
	if d.Slope >= 0 {
		t.Fatalf("H-slow slope must be negative: %v", d.Slope)
	}
	// tau* is the smallest tau with gap < 0.05 even if a smaller tau also dipped below
	d, err = KZG2Decide(kzG2Set(0.52, 0.04, 0.06, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "H-slow" || d.TauStar != 3200 {
		t.Fatalf("%+v", d)
	}
	// only the last rung below 5 %: tau* = 12800
	d, err = KZG2Decide(kzG2Set(0.52, 0.30, 0.10, 0.04))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "H-slow" || d.TauStar != 12800 {
		t.Fatalf("%+v", d)
	}
	if !strings.Contains(KZG2FormatReport(d), "12800") {
		t.Fatal("report names tau*")
	}
}

func TestKZG2DecideInconclusiveRate(t *testing.T) {
	// 0.10 < 0.6 * 0.52 = 0.312 and 0.10 >= 0.05 -> (c)
	d, err := KZG2Decide(kzG2Set(0.52, 0.30, 0.20, 0.10))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "inconclusive (rate)" || d.Rule != "(c)" || d.TauStar != 0 {
		t.Fatalf("%+v", d)
	}
	// exactly at the (a) threshold: 0.312 >= 0.312 is (a)
	d, _ = KZG2Decide(kzG2Set(0.52, 0.30, 0.20, 0.6*0.52))
	if d.Outcome != "H-first" {
		t.Fatalf("boundary of (a) belongs to (a): %+v", d)
	}
	// exactly 0.05 is not below 0.05: not (b)
	d, _ = KZG2Decide(kzG2Set(0.52, 0.30, 0.20, 0.05))
	if d.Outcome != "inconclusive (rate)" {
		t.Fatalf("gap(12800) = 0.05 is not < 0.05: %+v", d)
	}
}

func TestKZG2DecidePrecedenceOverlap(t *testing.T) {
	// (a) and (b) both hold only when gap(1600) < 0.0833: 0.045 < 0.05 and 0.045 >= 0.6*0.07 = 0.042.
	// (b) governs; the P4 miss is reported beside it.
	d, err := KZG2Decide(kzG2Set(0.07, 0.06, 0.05, 0.045))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "H-slow" || d.Rule != "(b)" || !d.Overlap || d.P4Hit || d.TauStar != 12800 {
		t.Fatalf("%+v", d)
	}
	rep := KZG2FormatReport(d)
	if !strings.Contains(rep, "P4") || !strings.Contains(rep, "(b) governs") {
		t.Fatalf("report must name the P4 miss and the precedence:\n%s", rep)
	}
}

func TestKZG2DecideZeroGapExcludedFromFit(t *testing.T) {
	d, err := KZG2Decide(kzG2Set(0.52, 0.30, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != "H-slow" || d.TauStar != 6400 {
		t.Fatalf("%+v", d)
	}
	if d.FitN != 2 || len(d.FitExcluded) != 2 || d.FitExcluded[0] != 6400 || d.FitExcluded[1] != 12800 {
		t.Fatalf("zero gaps must be excluded from the fit and listed: %+v", d)
	}
	if d.SlopeOK {
		t.Fatal("two points give a slope but no standard error; SlopeOK must be false")
	}
	rep := KZG2FormatReport(d)
	if !strings.Contains(rep, "excluded") {
		t.Fatalf("report must say the zeros were excluded:\n%s", rep)
	}
}

func TestKZG2DecideJumpReport(t *testing.T) {
	rungs := kzG2Set(0.52, 0.45, 0.40, 0.36)
	rungs[2].HeatJump = 0.15 // at tau = 6400
	rungs[3].CoolJump = 0.1001
	d, err := KZG2Decide(rungs)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Jumps) != 2 || d.Jumps[0] != 6400 || d.Jumps[1] != 12800 {
		t.Fatalf("jumps %v", d.Jumps)
	}
	if d.Outcome != "H-first" {
		t.Fatal("the jump clause does not override (a)-(c)")
	}
	if !strings.Contains(KZG2FormatReport(d), "jump observed at tau_Q = 6400") {
		t.Fatalf("report:\n%s", KZG2FormatReport(d))
	}
	rungs = kzG2Set(0.52, 0.30, 0.20, 0.10)
	rungs[0].CoolJump = 0.5
	if d, _ := KZG2Decide(rungs); d.Outcome != "inconclusive (rate)" || len(d.Jumps) != 1 {
		t.Fatalf("jump does not change (c): %+v", d)
	}
}

func TestKZG2DecideRefusesPartialOrBadSets(t *testing.T) {
	full := kzG2Set(0.52, 0.45, 0.40, 0.36)
	refuse := func(name string, rungs []KZG2Rung) {
		t.Helper()
		if _, err := KZG2Decide(rungs); err == nil {
			t.Errorf("%s: accepted; the rule is never applied to a partial or malformed set", name)
		}
	}
	refuse("three rungs", full[:3])
	refuse("endpoints only", []KZG2Rung{full[0], full[3]})
	refuse("none", nil)
	// the r5 sub-clause is named in the refusal, so a reader of the message knows why
	for _, partial := range [][]KZG2Rung{full[:3], {full[0], full[3]}, nil} {
		_, err := KZG2Decide(partial)
		if err == nil || !strings.Contains(err.Error(), "no partial-set reading") {
			t.Errorf("%d rung(s): refusal must cite the r5 harness-kill sub-clause, got %v", len(partial), err)
		}
	}
	refuse("five rungs", append(append([]KZG2Rung{}, full...), full[1]))
	dup := append([]KZG2Rung{}, full...)
	dup[2] = kzG2SynthRung(3200, 0.3, 0.02, 0.02)
	refuse("duplicate tau, one missing", dup)
	off := append([]KZG2Rung{}, full...)
	off[1].Tau = 3300
	refuse("tau off the set", off)
	inc := append([]KZG2Rung{}, full...)
	inc[3].Complete = false
	refuse("an incomplete rung (no end record)", inc)
	smoke := append([]KZG2Rung{}, full...)
	smoke[0].Mode = "smoke-0b"
	refuse("a smoke-0b file", smoke)
	tc := append([]KZG2Rung{}, full...)
	tc[2].Tc = 1.0666
	refuse("rungs with different tc", tc)
	grid := append([]KZG2Rung{}, full...)
	grid[1].Grid = KZLinGrid(0.5, 1.5, 41)
	refuse("rungs on different grids", grid)
	short := append([]KZG2Rung{}, full...)
	short[1].Cool = short[1].Cool[:40]
	refuse("a curve shorter than the grid", short)
	// order of the files does not matter
	if _, err := KZG2Decide([]KZG2Rung{full[3], full[1], full[0], full[2]}); err != nil {
		t.Fatalf("any file order is accepted: %v", err)
	}
}

func TestKZG2LoadRung(t *testing.T) {
	tc := 1.0666262219986817
	grid := KZLinGrid(0.5*tc, 1.5*tc, 41)
	cool, heat := make([]float64, 41), make([]float64, 41)
	coolT, heatT := []float64{1.5 * tc, tc, 0.5 * tc}, []float64{0.5 * tc, tc, 1.5 * tc}
	for i := range grid {
		cool[i] = 0.8
		heat[i] = 0.3
	}
	seeds := KZKzG2R1Seeds(1, false)
	enc := func(recs ...map[string]any) []byte {
		var b bytes.Buffer
		e := json.NewEncoder(&b)
		for _, r := range recs {
			if err := e.Encode(r); err != nil {
				t.Fatal(err)
			}
		}
		return b.Bytes()
	}
	meta := map[string]any{"kind": "meta", "mode": "0b", "tau": 3200, "tau_index": 1, "tc": tc, "seeds": seeds,
		"prereg_sha": "785e0ee2c81740b25fbeb9dfa5bb26e22159d8b7b221de2bdb75c2f7a6f51e28"}
	sb := map[string]any{"kind": "stage0b", "tau": 3200, "tc": tc, "seeds": seeds, "grid": grid, "cool": cool, "heat": heat,
		"cool_t": coolT, "heat_t": heatT,
		"cool_series": [][]float64{{0.9, 0.8, 0.8}, {0.7, 0.8, 0.8}}, "heat_series": [][]float64{{0.8, 0.3, 0.3}, {0.8, 0.3, 0.3}},
		"max_gap": 0.5, "max_gap_idx": 0, "max_gap_T": grid[0],
		"cool_jump": 0.1, "cool_jump_idx": 0, "cool_jump_T": coolT[0], "heat_jump": 0.5, "heat_jump_idx": 0, "heat_jump_T": heatT[0]}
	end := map[string]any{"kind": "end", "verdict": "0b complete", "tau": 3200}

	r, err := KZG2LoadRung(bytes.NewReader(enc(meta, sb, end)))
	if err != nil {
		t.Fatal(err)
	}
	if !r.Complete || r.Mode != "0b" || r.Tau != 3200 || r.TauIndex != 1 || r.Tc != tc || len(r.Seeds) != 10 || r.Seeds[0] != 5910 {
		t.Fatalf("%+v", r)
	}
	if len(r.Grid) != 41 || len(r.Cool) != 41 || len(r.Heat) != 41 || len(r.CoolSeries) != 2 || len(r.HeatT) != 3 {
		t.Fatalf("arrays %+v", r)
	}
	if r.RecMaxGap != 0.5 || r.HeatJump != 0.5 || r.HeatJumpIdx != 0 || r.PreregSHA == "" {
		t.Fatalf("scalars %+v", r)
	}
	// per-seed gap spread on the grid: seed 0 cool [0.9 0.8 0.8] (T_hi->T_lo) vs heat
	// [0.8 0.3 0.3] (T_lo->T_hi) -> 0.9 - 0.3 = 0.6 at T_hi; seed 1 cool [0.7 0.8 0.8] -> 0.5 at tc
	if r.SeedGapN != 2 || math.Abs(r.SeedGapMax-0.6) > 1e-12 || math.Abs(r.SeedGapMin-0.5) > 1e-12 {
		t.Fatalf("seed gaps %v %v n=%d", r.SeedGapMin, r.SeedGapMax, r.SeedGapN)
	}
	// without an end record the rung is incomplete (a kill): loaded, but Complete = false
	r2, err := KZG2LoadRung(bytes.NewReader(enc(meta, sb)))
	if err != nil || r2.Complete {
		t.Fatalf("no end: err=%v complete=%v", err, r2.Complete)
	}
	// an abort record marks it incomplete even with the stage0b record present
	ab := map[string]any{"kind": "abort", "code": 3, "reason": "pause file"}
	r3, _ := KZG2LoadRung(bytes.NewReader(enc(meta, sb, ab)))
	if r3.Complete || r3.AbortCode != 3 {
		t.Fatalf("abort: %+v", r3)
	}
	// no meta / no stage0b: error
	if _, err := KZG2LoadRung(bytes.NewReader(enc(sb, end))); err == nil {
		t.Fatal("no meta must be an error")
	}
	if _, err := KZG2LoadRung(bytes.NewReader(enc(meta, end))); err == nil {
		t.Fatal("no stage0b must be an error")
	}
	// a grid-mode (round 1) file is not a 0b-only file
	grid1 := map[string]any{"kind": "meta", "mode": "grid", "L": 16, "R": 25, "ladder": []int{25, 50}}
	if _, err := KZG2LoadRung(bytes.NewReader(enc(grid1, sb, end))); err == nil {
		t.Fatal("a grid-mode file must be refused")
	}
}
