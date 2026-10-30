package mdcu

// mdcu-kz orchestrator: saturation gate, Stage 0 per arm, then the replicate-major
// ladder grid (frozen mdcu-kz prereg §5, amend-1 §4). It owns the stop rules and the
// aborts; it writes records through Emit and never touches the filesystem itself, so the
// driver (cmd/mdcu-kz) decides what is written and where.

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// KZAbortError is a run-ending condition. Codes follow the driver's exit codes:
// 3 pause file, 2 wall, 5 memory/output, 6 NaN, 7 parity violation, 9 saturation gate.
type KZAbortError struct {
	Code   int
	Reason string
}

func (e *KZAbortError) Error() string { return fmt.Sprintf("abort %d: %s", e.Code, e.Reason) }

// Xi is the secondary (non-gating) correlation length of the plane field.
func (a *KZMainArm) Xi() (float64, bool) {
	return KZPlaneCorrelationLength(a.F.L, a.F.Frames())
}

// KZRunConfig is everything one run needs. Seeds are passed in, never chosen here.
type KZRunConfig struct {
	L, R   int
	Eta    float64
	Ladder []int
	Stage0 KZStage0Params

	MainSeed0, NullSeed0     uint64
	MainStage0a, MainStage0b []uint64
	NullStage0a, NullStage0b []uint64
	SatSeed0                 uint64
	SatN                     int
	SatFn                    func(L, n int, seed0 uint64) float64 // nil: KZSaturationRawDensity
	MkMain, MkNull           func(seed uint64) KZArm
	Check                    func() error // called at every step; a non-nil error aborts
	Emit                     func(rec map[string]any)
	Z2                       Z2Params
}

// KZRunOutcome is the verdict of the orchestration (not of the analysis).
type KZRunOutcome struct {
	Verdict          string // "complete", "premise failed", "inconclusive (null)"
	Density          float64
	MainTc, NullTc   float64
	MainStage0Reason string
	NullStage0Reason string
	Runs             int
}

// kzTick wraps an arm so that every step polls Check and every order read is screened
// for NaN. Aborts travel as panics of *KZAbortError and are recovered in KZExecute.
type kzTick struct {
	KZArm
	check func() error
}

func (t *kzTick) Step(eta, T float64, rng *rand.Rand) {
	if t.check != nil {
		if err := t.check(); err != nil {
			if ae, ok := err.(*KZAbortError); ok {
				panic(ae)
			}
			panic(&KZAbortError{Code: 1, Reason: err.Error()})
		}
	}
	t.KZArm.Step(eta, T, rng)
}

func (t *kzTick) Order() float64 {
	o := t.KZArm.Order()
	if math.IsNaN(o) || math.IsInf(o, 0) {
		panic(&KZAbortError{Code: 6, Reason: "non-finite order parameter"})
	}
	return o
}

// KZExecute runs the saturation gate, Stage 0 and the grid. On an abort it emits one
// "abort" record and returns the *KZAbortError; it emits "end" only on a normal finish.
func KZExecute(cfg KZRunConfig) (out KZRunOutcome, err error) {
	emit := cfg.Emit
	if emit == nil {
		emit = func(map[string]any) {}
	}
	p := cfg.Z2
	if p == (Z2Params{}) {
		p = Amend1Z2Params()
	}
	defer func() {
		if r := recover(); r != nil {
			ae, ok := r.(*KZAbortError)
			if !ok {
				panic(r)
			}
			emit(map[string]any{"kind": "abort", "code": ae.Code, "reason": ae.Reason})
			err = ae
		}
	}()
	abort := func(code int, format string, a ...any) {
		panic(&KZAbortError{Code: code, Reason: fmt.Sprintf(format, a...)})
	}
	wrap := func(mk func(uint64) KZArm) func(uint64) KZArm {
		return func(seed uint64) KZArm { return &kzTick{KZArm: mk(seed), check: cfg.Check} }
	}
	finish := func(verdict string) (KZRunOutcome, error) {
		out.Verdict = verdict
		emit(map[string]any{"kind": "end", "verdict": verdict, "runs": out.Runs})
		return out, nil
	}

	// 1. saturation gate (amend-1 §4)
	satFn := cfg.SatFn
	if satFn == nil {
		satFn = KZSaturationRawDensity
	}
	out.Density = satFn(cfg.L, cfg.SatN, cfg.SatSeed0)
	emit(map[string]any{"kind": "saturation", "L": cfg.L, "n": cfg.SatN, "seed0": cfg.SatSeed0,
		"density": out.Density, "within": KZSaturationWithinTol(out.Density)})
	if !KZSaturationWithinTol(out.Density) {
		abort(9, "saturation gate: raw -1 density %.4f outside 0.50 +/- 0.05", out.Density)
	}

	// 2. Stage 0, main then null; a failure stops before any sweep
	stage0 := func(name string, mk func(uint64) KZArm, a, b []uint64) (float64, bool, string) {
		raw := KZStage0aRun(mk, a, cfg.Eta, cfg.Stage0)
		ra := KZStage0aAnalyze(raw.T, raw.S)
		emit(map[string]any{"kind": "stage0a", "arm": name, "seeds": a, "ordered": ra.Ordered,
			"reason": ra.Reason, "tc": ra.Tc, "T": ra.T, "mean": ra.Mean, "se": ra.SE})
		if !ra.Ordered {
			return ra.Tc, false, "stage 0a: " + ra.Reason
		}
		rb := KZStage0bRun(mk, b, cfg.Eta, ra.Tc, cfg.Stage0)
		res := KZStage0bAnalyze(rb.CoolOnGrid, rb.HeatOnGrid, rb.CoolJump, rb.HeatJump)
		emit(map[string]any{"kind": "stage0b", "arm": name, "seeds": b, "pass": res.Pass,
			"reason": res.Reason, "max_gap": res.MaxGap, "cool_jump": rb.CoolJump, "heat_jump": rb.HeatJump,
			"tc": ra.Tc})
		if !res.Pass {
			return ra.Tc, false, "stage 0b: " + res.Reason
		}
		return ra.Tc, true, ""
	}
	mkMain, mkNull := wrap(cfg.MkMain), wrap(cfg.MkNull)
	var ok bool
	out.MainTc, ok, out.MainStage0Reason = stage0("main", mkMain, cfg.MainStage0a, cfg.MainStage0b)
	if !ok {
		return finish("premise failed")
	}
	out.NullTc, ok, out.NullStage0Reason = stage0("null", mkNull, cfg.NullStage0a, cfg.NullStage0b)
	if !ok {
		return finish("inconclusive (null)")
	}

	// 3. replicate-major grid: for rep j, main over the ladder, then null over the ladder
	type armSpec struct {
		name string
		mk   func(uint64) KZArm
		seed uint64
		tc   float64
	}
	arms := []armSpec{{"main", mkMain, cfg.MainSeed0, out.MainTc}, {"null", mkNull, cfg.NullSeed0, out.NullTc}}
	for j := 0; j < cfg.R; j++ {
		for _, a := range arms {
			seed := a.seed + uint64(j)
			for _, tau := range cfg.Ladder {
				t0 := time.Now()
				arm := a.mk(seed)
				res := KZRunRung(arm, cfg.Eta, a.tc, tau, KZNoiseRNG(seed, uint64(tau)), p)
				if math.IsNaN(res.Order) {
					abort(6, "NaN order parameter at arm=%s tau=%d rep=%d", a.name, tau, j)
				}
				if !res.ParityOK {
					abort(7, "raw -1 count %d is odd at arm=%s tau=%d rep=%d seed=%d",
						res.NegCells, a.name, tau, j, seed)
				}
				rec := map[string]any{"kind": "run", "arm": a.name, "tau": tau, "rep": j, "seed": seed, "L": cfg.L,
					"cells": res.Cells, "defects": res.Defects, "ambiguous": res.Ambiguous, "neg": res.NegCells,
					"parity": res.ParityOK, "order": res.Order}
				if x, isMain := arm.(*kzTick).KZArm.(interface{ Xi() (float64, bool) }); isMain {
					xi, capped := x.Xi()
					rec["xi"], rec["xi_capped"] = xi, capped
				}
				rec["secs"] = time.Since(t0).Seconds()
				emit(rec)
				out.Runs++
			}
		}
	}
	return finish("complete")
}

// ---- qbp-kz-g2 round 1: Stage-0b-only orchestration (FROZEN-v1 §2 harness addition 1) ----
//
// One τ_Q per process. No saturation gate, no Stage 0a, no null arm, no ladder. The
// round-1 KZExecute pipeline above is not modified; this entry point calls the same
// frozen Stage 0b mechanics (KZStage0bRun via KZStage0bRunWithSeries) with the given T_c.

// KZStage0bOnlyConfig is everything one Stage-0b-only process needs. Seeds are passed in
// and checked against the τ index here as well as in the driver (FROZEN-v1 §5 seed guard).
type KZStage0bOnlyConfig struct {
	L        int
	Eta, Tc  float64
	Tau      int            // this process's τ_Q; one of KZKzG2R1TauSet unless Smoke
	TauIndex int            // its τ index; the seeds must be KZKzG2R1Seeds(TauIndex, Rerun)
	Rerun    bool           // the reserved re-run sub-block (dated amendment only)
	Seeds    []uint64       // exactly the derived sub-block (or, in a smoke, free seeds)
	Smoke    bool           // a free-seed smoke of the mode: any τ_Q, KZSeedFree seeds only
	Stage0   KZStage0Params // RecEvery and NGrid (frozen 10 and 41); TauQ is Tau
	Mk       func(seed uint64) KZArm
	Check    func() error // called at the start and at every step; a non-nil error aborts
	Emit     func(rec map[string]any)
	Meta     map[string]any // driver-supplied meta fields (prereg_sha, binary/code sha, load …)
}

// KZExecuteStage0bOnly runs one Stage-0b-only process: meta, then the stage0b record
// (grid, seed-mean curves, per-seed series, max gap with index and T, jumps with their
// record index and T, the reason string), then end. Any abort emits one "abort" record
// and returns the *KZAbortError (codes as KZExecute: 3 pause, 2 wall, 6 NaN, 4 guard).
// The guards run before any arm is built and before any record is written.
func KZExecuteStage0bOnly(cfg KZStage0bOnlyConfig) (raw KZStage0bRaw, err error) {
	// 1. guards (exit 4 before any step; the driver has already refused, this is the backstop)
	mode := "0b"
	switch {
	case cfg.Mk == nil:
		return raw, &KZAbortError{Code: 4, Reason: "no arm constructor"}
	case cfg.Stage0.RecEvery <= 0 || cfg.Stage0.NGrid < 2 || cfg.Tau < cfg.Stage0.RecEvery:
		return raw, &KZAbortError{Code: 4, Reason: fmt.Sprintf("tau %d is below the record spacing %d", cfg.Tau, cfg.Stage0.RecEvery)}
	case math.IsNaN(cfg.Tc) || math.IsInf(cfg.Tc, 0) || cfg.Tc <= 0 || cfg.Eta <= 0:
		return raw, &KZAbortError{Code: 4, Reason: fmt.Sprintf("tc %v and eta %v must be finite and positive", cfg.Tc, cfg.Eta)}
	case cfg.Smoke:
		mode = "smoke-0b"
		if len(cfg.Seeds) == 0 {
			return raw, &KZAbortError{Code: 4, Reason: "no seeds"}
		}
		if e := KZCheckSmokeSeeds(cfg.Seeds); e != nil {
			return raw, &KZAbortError{Code: 4, Reason: "smoke-0b: " + e.Error()}
		}
	default:
		idx, ok := KZKzG2R1TauIndex(cfg.Tau)
		if !ok || idx != cfg.TauIndex {
			return raw, &KZAbortError{Code: 4, Reason: fmt.Sprintf("tau %d (index %d) is not in the frozen set %v", cfg.Tau, cfg.TauIndex, KZKzG2R1TauSet)}
		}
		if e := KZCheckKzG2R1Seeds(cfg.Seeds, cfg.TauIndex, cfg.Rerun); e != nil {
			return raw, &KZAbortError{Code: 4, Reason: "seed guard: " + e.Error()}
		}
	}

	emit := cfg.Emit
	if emit == nil {
		emit = func(map[string]any) {}
	}
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			ae, ok := r.(*KZAbortError)
			if !ok {
				panic(r)
			}
			emit(map[string]any{"kind": "abort", "code": ae.Code, "reason": ae.Reason, "tau": cfg.Tau,
				"wall_s": time.Since(start).Seconds()})
			err = ae
		}
	}()

	// 2. meta
	prm := cfg.Stage0
	prm.TauQ = cfg.Tau
	meta := map[string]any{"kind": "meta", "mode": mode, "L": cfg.L, "eta": cfg.Eta, "tc": cfg.Tc,
		"tau": cfg.Tau, "tau_index": cfg.TauIndex, "rerun": cfg.Rerun, "seeds": cfg.Seeds,
		"hold_steps": KZHoldSteps, "rec_every": prm.RecEvery, "ngrid": prm.NGrid, "noise_stream": kzStreamStage0b,
		"t_hi": 1.5 * cfg.Tc, "t_lo": 0.5 * cfg.Tc, "started": start.Format(time.RFC3339)}
	for k, v := range cfg.Meta {
		meta[k] = v
	}
	emit(meta)

	// 3. start check (pause file / wall), then the frozen Stage 0b mechanics per seed
	if cfg.Check != nil {
		if e := cfg.Check(); e != nil {
			if ae, ok := e.(*KZAbortError); ok {
				panic(ae)
			}
			panic(&KZAbortError{Code: 1, Reason: e.Error()})
		}
	}
	mk := func(seed uint64) KZArm { return &kzTick{KZArm: cfg.Mk(seed), check: cfg.Check} }
	raw = KZStage0bRunWithSeries(mk, cfg.Seeds, cfg.Eta, cfg.Tc, prm)

	// 4. the record. The round-1 5 % / 10 % clauses are reported as measurement definitions
	// (KZStage0bAnalyze, unchanged), not as a gate: §3 reads the four gaps together.
	r1 := KZStage0bAnalyze(raw.CoolOnGrid, raw.HeatOnGrid, raw.CoolJump, raw.HeatJump)
	emit(map[string]any{"kind": "stage0b", "arm": "main", "tau": cfg.Tau, "tau_index": cfg.TauIndex, "tc": cfg.Tc,
		"seeds": cfg.Seeds, "records": raw.Records,
		"grid": raw.Grid, "cool": raw.CoolOnGrid, "heat": raw.HeatOnGrid,
		"cool_t": raw.CoolT, "heat_t": raw.HeatT, "cool_series": raw.CoolSeries, "heat_series": raw.HeatSeries,
		"max_gap": raw.MaxGap, "max_gap_idx": raw.MaxGapIdx, "max_gap_T": raw.MaxGapT,
		"cool_jump": raw.CoolJump, "cool_jump_idx": raw.CoolJumpIdx, "cool_jump_T": raw.CoolJumpT,
		"heat_jump": raw.HeatJump, "heat_jump_idx": raw.HeatJumpIdx, "heat_jump_T": raw.HeatJumpT,
		"cool_end_s": raw.CoolEndS, "heat_start_s": raw.HeatStartS,
		"reason":              KZStage0bReason(raw),
		"round1_clauses_pass": r1.Pass, "round1_clauses_reason": r1.Reason})
	emit(map[string]any{"kind": "end", "verdict": "0b complete", "tau": cfg.Tau, "seeds_run": len(cfg.Seeds),
		"wall_s": time.Since(start).Seconds()})
	return raw, nil
}
