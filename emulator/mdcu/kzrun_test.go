package mdcu

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

// kzMockArm is a noise-free sigmoid order parameter with a known T_c. It stands in for a
// field so that the orchestrator's flow, stop rules and aborts can be tested cheaply.
type kzMockArm struct {
	seed     uint64
	tc, w    float64
	lo       float64 // floor of the order parameter; 0.2 with ordered=false
	ordered  bool
	hyst     bool // sticky on heating: S does not follow T upwards
	badPar   bool // odd raw -1 count
	nan      bool
	s, lastT float64
}

func (a *kzMockArm) Step(eta, T float64, rng *rand.Rand) {
	target := 1 / (1 + math.Exp((T-a.tc)/a.w))
	if !a.ordered {
		target = 0.2
	}
	if a.hyst && T > a.lastT {
		target = 1
	}
	a.lastT = T
	a.s += 0.5 * (target - a.s)
}

func (a *kzMockArm) Order() float64 {
	if a.nan {
		return math.NaN()
	}
	return a.s
}

func (a *kzMockArm) Detect(p Z2Params) Z2Result {
	neg := 24 + int(a.seed%5)*2
	if a.badPar {
		neg++
	}
	return Z2Result{L: 8, Cells: 512, Defects: neg - 2, Ambiguous: 2, NegCells: neg, ParityOK: neg%2 == 0}
}

type kzMockOpts struct{ mainOrdered, nullOrdered, mainHyst, badPar, nan bool }

func kzMockCfg(o kzMockOpts, recs *[]map[string]any) KZRunConfig {
	prm := KZStage0Params{NT: 40, Tmin: 0.05, Tmax: 8, Equil: 20, Meas: 20, TauQ: 100, RecEvery: 5, NGrid: 41}
	mk := func(ordered, hyst bool) func(uint64) KZArm {
		return func(seed uint64) KZArm {
			return &kzMockArm{seed: seed, tc: 1, w: 0.3, s: 1, lastT: math.Inf(1), ordered: ordered, hyst: hyst,
				badPar: o.badPar, nan: o.nan}
		}
	}
	return KZRunConfig{
		L: 8, R: 3, Eta: 0.1, Ladder: []int{25, 50},
		Stage0:    prm,
		MainSeed0: 13000, NullSeed0: 13200,
		MainStage0a: []uint64{13400, 13401, 13402}, MainStage0b: []uint64{13410, 13411, 13412, 13413},
		NullStage0a: []uint64{13420, 13421, 13422}, NullStage0b: []uint64{13430, 13431, 13432, 13433},
		SatSeed0: 12200, SatN: 20,
		MkMain: mk(o.mainOrdered, o.mainHyst), MkNull: mk(o.nullOrdered, false),
		Emit: func(r map[string]any) { *recs = append(*recs, r) },
	}
}

func kzKinds(recs []map[string]any) []string {
	k := make([]string, len(recs))
	for i, r := range recs {
		k[i] = r["kind"].(string)
	}
	return k
}

func kzCount(recs []map[string]any, kind string) int {
	n := 0
	for _, r := range recs {
		if r["kind"] == kind {
			n++
		}
	}
	return n
}

func TestKZRunFlow(t *testing.T) {
	var recs []map[string]any
	cfg := kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true}, &recs)
	out, err := KZExecute(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != "complete" {
		t.Fatalf("verdict %q", out.Verdict)
	}
	kinds := kzKinds(recs)
	want := []string{"saturation", "stage0a", "stage0b", "stage0a", "stage0b"}
	for i, w := range want {
		if kinds[i] != w {
			t.Fatalf("record %d is %q, want %q (%v)", i, kinds[i], w, kinds)
		}
	}
	if recs[1]["arm"] != "main" || recs[3]["arm"] != "null" {
		t.Fatal("stage 0 order is main then null")
	}
	if n := kzCount(recs, "run"); n != 2*cfg.R*len(cfg.Ladder) {
		t.Fatalf("run records %d", n)
	}
	if kinds[len(kinds)-1] != "end" || kzCount(recs, "end") != 1 {
		t.Fatalf("last record must be the single end record: %v", kinds[len(kinds)-3:])
	}
	// replicate-major order: for rep j, main over the ladder then null over the ladder
	runs := []map[string]any{}
	for _, r := range recs {
		if r["kind"] == "run" {
			runs = append(runs, r)
		}
	}
	i := 0
	for j := 0; j < cfg.R; j++ {
		for _, arm := range []string{"main", "null"} {
			seed0 := cfg.MainSeed0
			if arm == "null" {
				seed0 = cfg.NullSeed0
			}
			for _, tau := range cfg.Ladder {
				r := runs[i]
				i++
				if r["arm"] != arm || r["tau"] != tau || r["rep"] != j || r["seed"] != seed0+uint64(j) {
					t.Fatalf("run %d: got arm=%v tau=%v rep=%v seed=%v", i, r["arm"], r["tau"], r["rep"], r["seed"])
				}
				if r["parity"] != true {
					t.Fatal("parity")
				}
			}
		}
	}
	// the mock's T_c is 1; the steepest-pair midpoint on the geometric grid is near it
	for _, r := range recs {
		if r["kind"] == "stage0a" {
			tc := r["tc"].(float64)
			if tc < 0.7 || tc > 1.4 {
				t.Fatalf("stage 0a T_c %v", tc)
			}
		}
	}
}

func TestKZRunMainStage0FailureStops(t *testing.T) {
	var recs []map[string]any
	out, err := KZExecute(kzMockCfg(kzMockOpts{mainOrdered: false, nullOrdered: true}, &recs))
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != "premise failed" {
		t.Fatalf("verdict %q", out.Verdict)
	}
	if kzCount(recs, "run") != 0 {
		t.Fatal("no sweep may run after a failed Stage 0")
	}
	for _, r := range recs {
		if r["arm"] == "null" {
			t.Fatal("null Stage 0 must not run after a main failure")
		}
	}
	if kzCount(recs, "end") != 1 {
		t.Fatal("end record")
	}
}

func TestKZRunMainHysteresisFailureStops(t *testing.T) {
	var recs []map[string]any
	out, err := KZExecute(kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true, mainHyst: true}, &recs))
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != "premise failed" || kzCount(recs, "run") != 0 {
		t.Fatalf("verdict %q runs %d", out.Verdict, kzCount(recs, "run"))
	}
}

func TestKZRunNullStage0FailureStops(t *testing.T) {
	var recs []map[string]any
	out, err := KZExecute(kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: false}, &recs))
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != "inconclusive (null)" {
		t.Fatalf("verdict %q", out.Verdict)
	}
	if kzCount(recs, "run") != 0 {
		t.Fatal("no sweep may run after a failed null Stage 0")
	}
	if kzCount(recs, "stage0a") != 2 {
		t.Fatal("main Stage 0 must have run before the null failed")
	}
}

func kzAbortCode(t *testing.T, err error) int {
	t.Helper()
	var ae *KZAbortError
	if !errors.As(err, &ae) {
		t.Fatalf("want a KZAbortError, got %v", err)
	}
	return ae.Code
}

func TestKZRunParityViolationAborts7(t *testing.T) {
	var recs []map[string]any
	_, err := KZExecute(kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true, badPar: true}, &recs))
	if c := kzAbortCode(t, err); c != 7 {
		t.Fatalf("code %d", c)
	}
	if kzCount(recs, "abort") != 1 {
		t.Fatal("abort record")
	}
	if kzCount(recs, "run") != 0 {
		t.Fatal("the violating run must not be written as a good record")
	}
}

func TestKZRunNaNAborts6(t *testing.T) {
	var recs []map[string]any
	_, err := KZExecute(kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true, nan: true}, &recs))
	if c := kzAbortCode(t, err); c != 6 {
		t.Fatalf("code %d", c)
	}
}

func TestKZRunCheckHookAborts(t *testing.T) {
	var recs []map[string]any
	cfg := kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true}, &recs)
	calls := 0
	cfg.Check = func() error {
		calls++
		if calls > 3000 {
			return &KZAbortError{Code: 3, Reason: "pause file"}
		}
		return nil
	}
	_, err := KZExecute(cfg)
	if c := kzAbortCode(t, err); c != 3 {
		t.Fatalf("code %d", c)
	}
	if calls != 3001 {
		t.Fatalf("Check must stop the run at the step that raised it, calls=%d", calls)
	}
	if kzCount(recs, "abort") != 1 || kzCount(recs, "end") != 0 {
		t.Fatal("an aborted run writes an abort record and no end record")
	}
}

func TestKZRunSaturationGate(t *testing.T) {
	var recs []map[string]any
	cfg := kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true}, &recs)
	cfg.SatFn = func(L, n int, seed0 uint64) float64 { return 0.9 }
	_, err := KZExecute(cfg)
	if c := kzAbortCode(t, err); c != 9 {
		t.Fatalf("code %d", c)
	}
	if kzCount(recs, "stage0a") != 0 {
		t.Fatal("nothing may run after a failed saturation gate")
	}
	recs = nil
	cfg = kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true}, &recs)
	var gotL, gotN int
	var gotSeed uint64
	cfg.SatFn = func(L, n int, seed0 uint64) float64 { gotL, gotN, gotSeed = L, n, seed0; return 0.466 }
	if _, err := KZExecute(cfg); err != nil {
		t.Fatal(err)
	}
	if gotL != 8 || gotN != 20 || gotSeed != 12200 {
		t.Fatalf("saturation called with L=%d n=%d seed0=%d", gotL, gotN, gotSeed)
	}
}

func TestKZRunRealSaturationAtL8(t *testing.T) {
	var recs []map[string]any
	cfg := kzMockCfg(kzMockOpts{mainOrdered: true, nullOrdered: true}, &recs)
	cfg.SatFn = nil
	cfg.R = 1
	if _, err := KZExecute(cfg); err != nil {
		t.Fatal(err)
	}
	d := recs[0]["density"].(float64)
	if !KZSaturationWithinTol(d) {
		t.Fatalf("density %v", d)
	}
}

func TestKZMainArmXi(t *testing.T) {
	arm := &KZMainArm{F: NewKZPlaneFieldAligned(8, 13999)}
	xi, capped := arm.Xi()
	if !capped || xi != 4 {
		t.Fatalf("an aligned field has no decay: xi=%v capped=%v", xi, capped)
	}
}

// ---- qbp-kz-g2 round 1: Stage-0b-only entry point (FROZEN-v1 §2 addition 1) --------------

// kzCountArm wraps a mock arm and counts its steps.
type kzCountArm struct {
	KZArm
	steps *int
}

func (a *kzCountArm) Step(eta, T float64, rng *rand.Rand) { *a.steps++; a.KZArm.Step(eta, T, rng) }

func kzStage0bOnlyCfg(seeds []uint64, tau int, smoke bool, recs *[]map[string]any, made *[]uint64, steps map[uint64]*int, nan bool) KZStage0bOnlyConfig {
	idx, _ := KZKzG2R1TauIndex(tau)
	return KZStage0bOnlyConfig{
		L: 8, Eta: 0.1, Tc: 1.0, Tau: tau, TauIndex: idx, Seeds: seeds, Smoke: smoke,
		Stage0: KZStage0Params{RecEvery: 5, NGrid: 41},
		Mk: func(seed uint64) KZArm {
			*made = append(*made, seed)
			n := new(int)
			steps[seed] = n
			return &kzCountArm{KZArm: &kzMockArm{seed: seed, tc: 1, w: 0.3, s: 1, lastT: math.Inf(1), ordered: true, hyst: true, nan: nan}, steps: n}
		},
		Emit: func(r map[string]any) { *recs = append(*recs, r) },
		Meta: map[string]any{"prereg_sha": "785e0ee2c81740b25fbeb9dfa5bb26e22159d8b7b221de2bdb75c2f7a6f51e28"},
	}
}

func kzFreeBlock(first uint64) []uint64 {
	s := make([]uint64, 10)
	for j := range s {
		s[j] = first + uint64(j)
	}
	return s
}

func TestKZStage0bOnlyFlow(t *testing.T) {
	var recs []map[string]any
	var made []uint64
	steps := map[uint64]*int{}
	seeds := kzFreeBlock(13500)
	tau := 40
	cfg := kzStage0bOnlyCfg(seeds, tau, true, &recs, &made, steps, false)
	raw, err := KZExecuteStage0bOnly(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if kinds := kzKinds(recs); len(kinds) != 3 || kinds[0] != "meta" || kinds[1] != "stage0b" || kinds[2] != "end" {
		t.Fatalf("records %v, want [meta stage0b end]", kinds)
	}
	for _, k := range []string{"saturation", "stage0a", "run", "abort"} {
		if kzCount(recs, k) != 0 {
			t.Fatalf("a Stage-0b-only run must not emit %q", k)
		}
	}
	// one arm per seed, in the given order, each stepped hold + cool + heat
	if len(made) != len(seeds) {
		t.Fatalf("arms made %v", made)
	}
	for j, s := range seeds {
		if made[j] != s {
			t.Fatalf("seed order %v, want %v", made, seeds)
		}
		if n := *steps[s]; n != KZHoldSteps+2*tau {
			t.Fatalf("seed %d: %d steps, want %d", s, n, KZHoldSteps+2*tau)
		}
	}
	meta := recs[0]
	if meta["prereg_sha"] != cfg.Meta["prereg_sha"] || meta["tau"] != tau || meta["tc"] != 1.0 || meta["mode"] != "smoke-0b" {
		t.Fatalf("meta %v", meta)
	}
	if got, ok := meta["seeds"].([]uint64); !ok || len(got) != 10 || got[0] != 13500 {
		t.Fatalf("meta seeds %v", meta["seeds"])
	}
	if _, has := meta["R"]; has {
		t.Fatal("a 0b-only meta must not look like a grid meta (R, ladder); -analyze refuses it by design")
	}
	sb := recs[1]
	if sb["tau"] != tau || sb["tc"] != 1.0 || sb["arm"] != "main" {
		t.Fatalf("stage0b record %v", sb)
	}
	nrec := tau/5 + 1
	if g := sb["grid"].([]float64); len(g) != 41 || g[0] != 0.5 || math.Abs(g[40]-1.5) > 1e-12 {
		t.Fatalf("grid %v", g)
	}
	if len(sb["cool"].([]float64)) != 41 || len(sb["heat"].([]float64)) != 41 {
		t.Fatal("cool/heat seed-mean arrays must be the 41-point grid curves")
	}
	cs, hs := sb["cool_series"].([][]float64), sb["heat_series"].([][]float64)
	if len(cs) != 10 || len(hs) != 10 || len(cs[0]) != nrec || len(hs[9]) != nrec {
		t.Fatalf("per-seed series %d x %d, %d x %d; want 10 x %d", len(cs), len(cs[0]), len(hs), len(hs[9]), nrec)
	}
	if len(sb["cool_t"].([]float64)) != nrec || len(sb["heat_t"].([]float64)) != nrec {
		t.Fatal("record T labels")
	}
	gap, idx, T := KZStage0bGap(raw.Grid, raw.CoolOnGrid, raw.HeatOnGrid)
	if sb["max_gap"] != gap || sb["max_gap_idx"] != idx || sb["max_gap_T"] != T || gap <= 0 {
		t.Fatalf("max gap fields %v %v %v vs %v %v %v", sb["max_gap"], sb["max_gap_idx"], sb["max_gap_T"], gap, idx, T)
	}
	if sb["cool_jump"] != raw.CoolJump || sb["heat_jump"] != raw.HeatJump || sb["cool_jump_idx"] != raw.CoolJumpIdx ||
		sb["heat_jump_idx"] != raw.HeatJumpIdx || sb["cool_jump_T"] != raw.CoolJumpT || sb["heat_jump_T"] != raw.HeatJumpT {
		t.Fatalf("jump fields %v", sb)
	}
	reason := sb["reason"].(string)
	if !strings.Contains(reason, fmt.Sprintf("grid point %d (T = %.4f)", idx, T)) {
		t.Fatalf("reason %q does not name the max-gap index and T", reason)
	}
	if recs[2]["verdict"] != "0b complete" || recs[2]["tau"] != tau {
		t.Fatalf("end %v", recs[2])
	}
	if _, ok := recs[2]["wall_s"].(float64); !ok {
		t.Fatal("end record carries wall seconds")
	}
}

func TestKZStage0bOnlyGuardsRefuseBeforeAnyStep(t *testing.T) {
	type c struct {
		name  string
		seeds []uint64
		tau   int
		smoke bool
	}
	cases := []c{
		{"smoke on a frozen kz seed", kzFreeBlock(5610), 40, true},
		{"smoke on the kz-g2 r1 block", kzFreeBlock(5900), 40, true},
		{"smoke on the re-run block", kzFreeBlock(5940), 40, true},
		{"run mode on free seeds", kzFreeBlock(13500), 1600, false},
		{"run mode with tau off the set", kzFreeBlock(13500), 1700, false},
		{"run mode with a tau index that does not match the seeds", nil, 1600, false},
	}
	for _, k := range cases {
		var recs []map[string]any
		var made []uint64
		steps := map[uint64]*int{}
		cfg := kzStage0bOnlyCfg(k.seeds, k.tau, k.smoke, &recs, &made, steps, false)
		_, err := KZExecuteStage0bOnly(cfg)
		if code := kzAbortCode(t, err); code != 4 {
			t.Errorf("%s: code %d, want 4", k.name, code)
		}
		if len(made) != 0 || len(recs) != 0 {
			t.Errorf("%s: %d arms made, %d records written; the guard runs before anything", k.name, len(made), len(recs))
		}
	}
	// the run-mode guard accepts exactly the derived sub-block (the arm is never built: the
	// mock refuses via a Check hook that fires on the first step, so no frozen seed is run)
	for i := range KZKzG2R1TauSet {
		var recs []map[string]any
		var made []uint64
		steps := map[uint64]*int{}
		cfg := kzStage0bOnlyCfg(KZKzG2R1Seeds(i, false), KZKzG2R1TauSet[i], false, &recs, &made, steps, false)
		cfg.Mk = func(seed uint64) KZArm { panic("the test must not build an arm on a frozen seed") }
		cfg.Check = func() error { return &KZAbortError{Code: 3, Reason: "pause file (test)"} }
		_, err := KZExecuteStage0bOnly(cfg)
		if code := kzAbortCode(t, err); code != 3 {
			t.Errorf("tau index %d: code %d, want 3 (guard accepted, start check fired)", i, code)
		}
		if kinds := kzKinds(recs); len(kinds) != 2 || kinds[0] != "meta" || kinds[1] != "abort" {
			t.Errorf("tau index %d: records %v, want [meta abort]", i, kinds)
		}
	}
}

func TestKZStage0bOnlyAborts(t *testing.T) {
	// pause via the Check hook mid-run: abort record, no stage0b, no end
	var recs []map[string]any
	var made []uint64
	steps := map[uint64]*int{}
	cfg := kzStage0bOnlyCfg(kzFreeBlock(13500), 40, true, &recs, &made, steps, false)
	calls := 0
	cfg.Check = func() error {
		calls++
		if calls > 500 {
			return &KZAbortError{Code: 3, Reason: "pause file"}
		}
		return nil
	}
	_, err := KZExecuteStage0bOnly(cfg)
	if c := kzAbortCode(t, err); c != 3 {
		t.Fatalf("code %d", c)
	}
	if calls != 501 {
		t.Fatalf("Check must stop the run at the step that raised it, calls=%d", calls)
	}
	if kinds := kzKinds(recs); len(kinds) != 2 || kinds[0] != "meta" || kinds[1] != "abort" {
		t.Fatalf("records %v, want [meta abort]", kinds)
	}
	if recs[1]["code"] != 3 {
		t.Fatalf("abort record %v", recs[1])
	}
	// wall cap and pause are the driver's Check; NaN is screened here: exit 6
	recs, made = nil, nil
	cfg = kzStage0bOnlyCfg(kzFreeBlock(13500), 40, true, &recs, &made, steps, true)
	_, err = KZExecuteStage0bOnly(cfg)
	if c := kzAbortCode(t, err); c != 6 {
		t.Fatalf("code %d", c)
	}
	if kzCount(recs, "stage0b") != 0 || kzCount(recs, "end") != 0 || kzCount(recs, "abort") != 1 {
		t.Fatal("a NaN run writes an abort record and nothing else")
	}
}
