package mdcu

import (
	"errors"
	"math"
	"math/rand/v2"
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
