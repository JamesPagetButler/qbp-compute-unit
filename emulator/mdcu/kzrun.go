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
